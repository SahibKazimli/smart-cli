package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"google.golang.org/api/content/v2"
	"log"
	"os"
	"path/filepath"
	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/chunker"
	"smart-cli/go-backend/embedder"
	"smart-cli/go-backend/file_resolver"
	"smart-cli/go-backend/generator"
	"strings"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "smartcli",
		Short: "AI-enhanced command line interface",
		Long:  "Modular AI-enhanced command line interface for coding assistance",
	}
	// Add the code review command
	rootCmd.AddCommand(createCodeReviewCmd())

	// And more commands will be added later
	rootCmd.AddCommand()
	rootCmd.AddCommand()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("smartcli commands:")
	fmt.Println("  index   Re-index the project into Redis")
	fmt.Println("  ask     Ask a question using the indexed project context")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  smartcli index [--dir .] [--chunk-size 800] [--overlap 50] [--index-name NAME] [--embedding-model text-embedding-005]")
	fmt.Println("  smartcli ask   --q \"your question\" [--topk 5] [--generation-model gemini-1.5-pro] [--embedding-model text-embedding-005] [--show-context]")
}

func mustGCP() (projectID, location, creds string) {
	projectID = os.Getenv("GCP_PROJECT_ID")
	location = os.Getenv("GCP_LOCATION")
	creds = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if projectID == "" || location == "" || creds == "" {
		log.Fatal("Please set GCP_PROJECT_ID, GCP_LOCATION, and GOOGLE_APPLICATION_CREDENTIALS")
	}
	return
}

func createCodeReviewCmd() *cobra.Command {
	var filePath string
	var detailLevel string

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

			// Call the function that will handle the code review

		},
	}

	// Add flags specific to code review
	codeReviewCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to file for code review")
	codeReviewCmd.Flags().StringVarP(&detailLevel, "detail", "d", "medium", "Level of review detail (low, medium, high)")

	return codeReviewCmd
}

func performCodeReview(filePath string, detailLevel string) {
	fmt.Printf("Performing %s level code review for: %s\n", detailLevel, filePath)
	// TODO: Implement the code review logic and call functions in generator.go
	// 1. Run the vector similarity search
	// 2. Call the AI
	// 3. Format and display results
	retrievedChunks := chunk_retriever.ConcurrentChunkRetrieval(rdb*redis.Client{}, quer)

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

func reviewCodeFile(
	ctx context.Context,
	gen *generator.Generator,
	filePath string,
	rdb *redis.Client,
	detailLevel string,
	indexName string,
	chunkSize int,
	overlap int,
) {

	fmt.Printf("\n--- Reviewing %s ---\n", filePath)
	// Content
	fileContent, err := getFileContent(filePath)
	if err != nil {
		fmt.Printf("warning: cannot read file: %v\n", err)
		return
	}

	// Embeddings
	_, _, creds := mustGCP()
	embedderClient, err := embedder.EmbedderClient(ctx, creds, rdb, "")
	if err != nil {
		fmt.Printf("warning: cannot create embedder: %v\n", err)
		return
	}
	reviewEmbedding, reviewQuery := createEmbeddings(filePath, ctx, rdb, strings.ToLower(detailLevel))
	fileEmbedding, err := embedderClient.EmbedQuery(fileContent)
	if err != nil {
		fmt.Printf("warning: cannot embed file content: %v\n", err)
		return
	}

	// Queries
	reviewChunkQuery := chunk_retriever.PrepareQuery(reviewQuery, 5, indexName)
	fileContentQuery := chunk_retriever.PrepareQuery(fileContent, 10, indexName)
	queries := []chunk_retriever.ChunkQuery{reviewChunkQuery, fileContentQuery}
	embeddings := [][]float32{reviewEmbedding, fileEmbedding}

	// Retrieve
	chunks, err := chunk_retriever.ConcurrentChunkRetrieval(rdb, queries, embeddings, 10)
	if err != nil {
		fmt.Printf("warning: retrieval error: %v\n", err)
	}
	// Creating a generator
	if gen == nil {
		gen, err = generator.NewAgent(ctx, "gemini-1.5-pro")
		if err != nil {
			fmt.Printf("warning: failed to create agent: %v\n", err)
			return
		}
	}
	out, err := gen.Answer(ctx, reviewQuery, chunks)
	if err != nil {
		fmt.Printf("warning: failed to generate review: %v\n", err)
		return
	}
	fmt.Println(out)
}

// ===== Helpers =====

func searchQuery(filePath string) (chunk_retriever.ChunkQuery, error) {
	// Connect to Redis for retrieving relevant chunks
	rdb := chunk_retriever.Connect()

	// Get the index name
	indexName, err := chunk_retriever.GetIndexName(rdb)
	if err != nil {
		return chunk_retriever.ChunkQuery{}, fmt.Errorf("failed to get index name: %v", err)
	}

	// Prepare query based on file content
	filename := filepath.Base(filePath)

	// Create a review query based on the file content
	query := fmt.Sprintf("Review this %s code for bugs, improvements, and best practices: %s", filename)

	// Prepare chunk query
	chunkQuery := chunk_retriever.PrepareQuery(query, 10, indexName)
	return chunkQuery, nil
}

func createEmbeddings(filePath string, ctx context.Context, rdb *redis.Client, detailLevel string) ([]float32, string) {
	// Create an embedder
	_, _, creds := mustGCP()
	embedderClient, err := embedder.EmbedderClient(ctx, creds, rdb, "")
	if err != nil {
		fmt.Printf("Error creating embedder: %v\n", err)
	}

	// Generate query based on file content and detail level
	fileExt := filepath.Ext(filePath)
	language := strings.TrimPrefix(fileExt, ".")

	// Adjust query based on detail level
	var reviewDetail string
	switch detailLevel {
	case "low":
		reviewDetail = "Provide a basic review focusing only on major issues"
	case "high":
		reviewDetail = "Provide a detailed, thorough review covering all aspects"
	default:
		reviewDetail = "Provide a balanced review of important issues"
	}

	reviewQuery := fmt.Sprintf("Review this %s code file %s. %s:",
		language, filepath.Base(filePath), reviewDetail)

	// Generate embedding for the query
	queryEmbedding, err := embedderClient.EmbedQuery(reviewQuery)
	if err != nil {
		fmt.Printf("Error generating query embedding: %v\n", err)
	}
	return queryEmbedding, reviewQuery
}

func getFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
