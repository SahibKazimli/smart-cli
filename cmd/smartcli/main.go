package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path/filepath"
	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/chunker"
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

func reviewCodeFile(ctx context.Context, gen *generator.Generator, filePath string, chunkSize int, overlap int) {
	fmt.Printf("\n--- Reviewing %s ---\n", filePath)

	// Read and chunk the file
	fileChunks, err := chunker.SplitFile(filePath, chunkSize, overlap)
	if err != nil {
		fmt.Printf("Error processing file %s: %v\n", filePath, err)
		return
	}

	if len(fileChunks) == 0 {
		fmt.Printf("No content to review in %s\n", filePath)
		return
	}

	// Prepare content for the AI
	var fullContent string
	for _, chunk := range fileChunks {
		fullContent += chunk.Text
	}

}
