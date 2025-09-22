package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/embedder"
	"smart-cli/go-backend/generator"
	"strings"
)

func createErrorCommand() *cobra.Command {
	var errorText string

	errExplainCmd := &cobra.Command{
		Use:   "explain",
		Short: "Explain errors in your code",
		Long:  `Explain compilation errors, runtime errors, or general code issues using AI`,
		Example: `  smartcli explain -e "undefined: fmt.Println"
  smartcli explain -f main.go --auto     # Auto-detect issues in file
  smartcli explain --build               # Explain current build errors
  smartcli explain -e "cannot use string as int"`,
		Run: func(cmd *cobra.Command, args []string) {
			// If error text is provided as argument
			if errorText == "" && len(args) > 0 {
				errorText = strings.Join(args, " ")
			}
			if errorText == "" {
				fmt.Println("Error: Please provide an error message to explain")
				fmt.Println("Examples:")
				fmt.Println("smartcli explain -e ")
			}
			explainError(errorText)
		},
	}
	errExplainCmd.Flags().StringVarP(&errorText, "error", "e", "", "Error message to explain")
	return errExplainCmd
}

// ===== Helpers =====

func explainError(errorText string) {
	fmt.Printf("Explaining error: %s\n", errorText)

	ctx := context.Background()

	// Connect to services
	rdb := chunk_retriever.Connect()
	defer func() { _ = rdb.Close() }()

	indexName, err := chunk_retriever.GetIndexName(rdb)
	if err != nil {
		fmt.Printf("Warning: Could not get index name: %v\n", err)
	}

	_, _, creds := mustGCP()
	embedderClient, err := embedder.EmbedderClient(ctx, creds, rdb, "")
	if err != nil {
		fmt.Printf("Error creating embedder: %v\n", err)
		return
	}

	// Prepare retrieval
	retrievedChunks := retrieveRelevantChunks(ctx, errorText, embedderClient, rdb, indexName)

	// Generate explanation using AI
	answer := generateAIExplanation(ctx, errorText, retrievedChunks)
	if answer != "" {
		fmt.Println("\n===== AI Explanation =====")
		fmt.Println(answer)
	}
}

func retrieveRelevantChunks(ctx context.Context, errorText string, embedderClient *embedder.Embedder, rdb *redis.Client, indexName string) []chunk_retriever.Chunk {
	searchQuery := fmt.Sprintf("error %s golang programming fix solution", errorText)
	queryEmbedding := createEmbedding(searchQuery, embedderClient)

	if indexName == "" {
		return nil
	}

	chunkQuery := chunk_retriever.PrepareQuery(searchQuery, 5, indexName)
	retrievedChunks, _ := chunk_retriever.ConcurrentChunkRetrieval(rdb,
		[]chunk_retriever.ChunkQuery{chunkQuery},
		[][]float32{queryEmbedding},
		5)
	fmt.Printf("Retrieved %d context chunks\n", len(retrievedChunks))
	return retrievedChunks
}

func generateAIExplanation(ctx context.Context, errorText string, retrievedChunks []chunk_retriever.Chunk) string {
	prompt := fmt.Sprintf(`Please explain this programming error and provide a solution:

Error: %s

Please explain:
1. What this error means in simple terms
2. What typically causes this error
3. How to fix it with clear steps and code examples
4. How to prevent it in the future

Keep the explanation clear, practical, and focused on Go programming.`, errorText)

	gen, err := generator.NewAgent(ctx, "gemini-1.5-pro")
	if err != nil {
		fmt.Printf("Error creating AI agent: %v\n", err)
		return ""
	}

	answer, err := gen.Answer(ctx, prompt, retrievedChunks)
	if err != nil {
		fmt.Printf("Error getting AI explanation: %v\n", err)
		return ""
	}

	return answer
}
