package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/embedder"
	"smart-cli/go-backend/file_resolver"
	"smart-cli/go-backend/generator"
)

func createCodeReviewCmd() *cobra.Command {
	var filePath string
	var detailLevel string
	var autoExplain bool
	var userQuery string

	codeReviewCmd := &cobra.Command{
		Use:   "review",
		Short: "Review code for improvements",
		Long:  `Analyze code for potential bugs, style improvements, and optimization opportunities.`,
		Run: func(cmd *cobra.Command, args []string) {
			// If no file path is provided but there are arguments, use the first argument
			if filePath == "" && len(args) > 0 {
				filePath = args[0]
			}

			if filePath == "" {
				fmt.Println("Error: Please provide a file to review")
				return
			}
			// Require a user query
			if userQuery == "" {
				fmt.Println("Error: Please provide a question with -q or --query")
				fmt.Println("Example: smartcli review -f embedder.go -q \"what does this file do?\"")
				return
			}

			// Call the function that will handle the code review
			performCodeReview(filePath, detailLevel, userQuery)

		},
	}

	// Add flags specific to code review
	codeReviewCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to file for analysis (required)")
	codeReviewCmd.Flags().StringVarP(&detailLevel, "detail", "d", "medium", "Level of detail (low, medium, high)")
	codeReviewCmd.Flags().StringVarP(&userQuery, "query", "q", "", "Your question about the code")
	codeReviewCmd.Flags().BoolVar(&autoExplain, "explain", false, "Automatically explain errors/issues in the file")

	return codeReviewCmd
}

func performCodeReview(filePath string, detailLevel string, userQuery string) {
	fmt.Printf("Performing %s level code review for: %s\n", detailLevel, filePath)
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
	queryEmbedding := createEmbedding(userQuery, embedderClient)
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

	// Create a prompt that asks the LLM to answer the user's specific question
	generationPrompt := fmt.Sprintf(`Based on the provided code context, 
						please answer this question about the file '%s': %s\n\n
						Provide a %s level of detail in your answer.`, filePath, userQuery, detailLevel)

	// Generate answer/review
	gen, err := generator.NewAgent(ctx, "gemini-1.5-pro")
	if err != nil {
		fmt.Printf("warning: failed to create agent: %v\n", err)
		return
	}
	answer, err := gen.Answer(ctx, generationPrompt, retrievedChunks)
	if err != nil {
		fmt.Printf("warning: failed to generate review: %v\n", err)
		return
	}
	fmt.Println("\n===== Review =====")
	fmt.Println(answer)
}

// ===== Helpers =====

func createEmbedding(userQuery string, embedderClient *embedder.Embedder) []float32 {
	queryEmbedding, err := embedderClient.EmbedQuery(userQuery)
	if err != nil {
		fmt.Printf("Error generating query embedding: %v\n", err)
	}
	return queryEmbedding
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
