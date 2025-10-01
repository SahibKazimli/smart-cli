package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/embedder"
	"smart-cli/go-backend/generator"
	"strings"

	"github.com/spf13/cobra"
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

			// If still no query, use a sensible default
			if strings.TrimSpace(userQuery) == "" {
				userQuery = "Summarize this file, list key functions/methods and explain what they do. Highlight any potential issues."
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

	// Resolve file path (supports bare filenames)
	resolvedPath, err := resolveFilePath(filePath)
	if err != nil {
		fmt.Printf("Error resolving file: %v\n", err)
		return
	}

	// Read file content and always include it as a context chunk
	fileContent, err := getFileContent(resolvedPath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	fileChunk := chunk_retriever.Chunk{
		Text:     fileContent,
		Metadata: map[string]string{"file": resolvedPath, "source": "file"},
	}

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

	// Always include the target file chunk first so the LLM can answer even if RAG is empty
	retrievedChunks = append([]chunk_retriever.Chunk{fileChunk}, retrievedChunks...)
	fmt.Printf("Using %d context chunk(s) (file + %d retrieved)\n", len(retrievedChunks), len(retrievedChunks)-1)

	// Sanitize user query BEFORE embedding
	rawQuery := userQuery
	userQuery = strings.TrimSpace(userQuery)
	if userQuery == "" {
		userQuery = "Summarize this file."
	}
	if !strings.HasSuffix(userQuery, "?") && !strings.HasSuffix(userQuery, ".") {
		userQuery += "?"
	}

	// Create a prompt that asks the LLM to answer the user's specific question
	instructions := fmt.Sprintf(
		"You are analyzing file %s. Provide a %s level of detail. Focus ONLY on function RetrieveChunks if the question is about it.",
		filePath, detailLevel,
	)

	// Generate answer/review
	gen, err := generator.NewAgent(ctx, "gemini-2.5-flash")
	if err != nil {
		fmt.Printf("warning: failed to create agent: %v\n", err)
		return
	}
	answer, err := gen.Answer(ctx, userQuery+"\n\n"+instructions, retrievedChunks)
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

func getFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// resolveFilePath supports:
// - Absolute or relative paths
// - Bare filenames searched from current directory recursively
func resolveFilePath(input string) (string, error) {
	// If the path exists as given, use it
	if _, err := os.Stat(input); err == nil {
		abs, _ := filepath.Abs(input)
		return abs, nil
	}
	// Try relative to CWD
	candidate := filepath.Join(".", input)
	if _, err := os.Stat(candidate); err == nil {
		abs, _ := filepath.Abs(candidate)
		return abs, nil
	}
	// Search recursively by basename
	target := filepath.Base(input)
	var found string
	_ = filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		// Skip common large/vendor dirs
		base := d.Name()
		if base == ".git" || strings.Contains(path, "/.git/") ||
			strings.Contains(path, "/node_modules/") ||
			strings.Contains(path, "/venv/") ||
			strings.Contains(path, "/.venv/") {
			return nil
		}
		if filepath.Base(path) == target {
			found = path
			return filepath.SkipDir // stop early on first match
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("file not found: %s", input)
	}
	abs, _ := filepath.Abs(found)
	return abs, nil
}
