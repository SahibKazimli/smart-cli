package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/embedder"
	"smart-cli/go-backend/file_resolver"
	"smart-cli/go-backend/generator"
)

func performCodeReview(filePath string, detailLevel string, userQuery string) {
	fmt.Printf("Performing %s level code review for: %s\n", detailLevel, filePath)
	// 1. Run the vector similarity search
	// 2. Call the AI
	// 3. Format and display results

	ctx := context.Background()

	// Connect to Redis and resolve index name
	rdb := chunk_retriever.Connect()
	defer func() { _ = rdb.Close() }()

	indexName, err := chunk_retriever.GetIndexName(rdb)
	if err != nil {
		fmt.Printf("Error: failed to get index name: %v\n", err)
		return
	}

	// Use the user query for embedding and retrieval
	_, _, creds := mustGCP()
	embedderClient, err := embedder.EmbedderClient(ctx, creds, rdb, "")
	if err != nil {
		fmt.Printf("Error creating embedder: %v\n", err)
		return
	}
	queryEmbedding, retrievalQuery := createEmbeddings(userQuery, embedderClient)
	chunkQuery := chunk_retriever.PrepareQuery(userQuery, 10, indexName)

	// Concurrent chunk retrieval
	// Retrieve relevant chunks
	retrievedChunks, err := chunk_retriever.ConcurrentChunkRetrieval(rdb,
		[]chunk_retriever.ChunkQuery{chunkQuery},
		[][]float32{queryEmbedding},
		10)

	if err != nil {
		fmt.Printf("Warning: retrieval error: %v\n", err)
	}

	fmt.Printf("Retrieved %d context chunks\n", len(retrievedChunks))

	// Generate answer/review
	gen, err := generator.NewAgent(ctx, "gemini-1.5-pro")
	if err != nil {
		fmt.Printf("warning: failed to create agent: %v\n", err)
		return
	}
	answer, err := gen.Answer(ctx, retrievalQuery, retrievedChunks)
	if err != nil {
		fmt.Printf("warning: failed to generate review: %v\n", err)
		return
	}
	fmt.Println("\n===== Review =====")
	fmt.Println(answer)
}

// ===== Helpers =====

func createEmbeddings(userQuery string, embedderClient *embedder.Embedder) ([]float32, string) {
	queryEmbedding, err := embedderClient.EmbedQuery(userQuery)
	if err != nil {
		fmt.Printf("Error generating query embedding: %v\n", err)
	}
	return queryEmbedding, userQuery
}

func getCodeFilesFromDir(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && file_resolver.IsCodeFile(path) {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func getFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
