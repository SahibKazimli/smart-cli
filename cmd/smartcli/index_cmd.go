package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"path/filepath"
	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/embedder"
	"smart-cli/go-backend/re_indexer"
)

func createIndexCmd() *cobra.Command {
	var dir string
	var indexName string
	var force bool
	var model string
	var chunkSize int
	var overlap int

	indexCmd := &cobra.Command{
		Use:   "index",
		Short: "Index your codebase for AI-powered search",
		Long:  `Scan and index your codebase to enable AI-powered code review and error explanation`,
		Example: `  smartcli index                    # Index current directory
  smartcli index --dir ./my-project  # Index specific directory
  smartcli index --force             # Re-index even if index exists`,
		Run: func(cmd *cobra.Command, args []string) {
			indexCodebase(dir, indexName, force, model, chunkSize, overlap)
		},
	}

	indexCmd.Flags().StringVarP(&dir, "dir", "d", "", "Directory to index (defaults to current directory)")
	indexCmd.Flags().StringVarP(&indexName, "name", "n", "", "Index name (auto-generated if not provided)")
	indexCmd.Flags().BoolVarP(&force, "force", "f", false, "Force re-indexing even if index already exists")
	indexCmd.Flags().StringVarP(&model, "model", "m", "text-embedding-005", "Embedding model to use")
	indexCmd.Flags().IntVar(&chunkSize, "chunk-size", 800, "Size of text chunks")
	indexCmd.Flags().IntVar(&overlap, "overlap", 50, "Overlap between chunks")

	return indexCmd
}

// ===== Helpers =====

func indexCodebase(dir, indexName string, force bool, model string, chunkSize, overlap int) {
	if dir == "" {
		dir = "."
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Printf("Error resolving directory: %v\n", err)
		return
	}

	fmt.Println("Connecting to Redis...")
	rdb := chunk_retriever.Connect()
	if rdb == nil {
		fmt.Println("Cannot connect to Redis.")
		fmt.Println("Make sure Redis is running on localhost:6379 or set REDIS_ADDR environment variable.")
		return
	}
	defer func() { _ = rdb.Close() }()
	fmt.Println("Redis connection OK")

	// Ensure GCP credentials exist; mustGCP() will exit if missing.
	_, _, creds := mustGCP()

	ctx := context.Background()
	emb, err := embedder.EmbedderClient(ctx, creds, rdb, model)
	if err != nil {
		fmt.Printf("Error creating embedder: %v\n", err)
		return
	}

	// Build indexer (auto-derives index name from dir if not provided)
	indexer := re_indexer.NewIndexer(rdb, emb, absDir, indexName)

	// Detect an existing index with the same derived/default name and bail unless --force
	if !force {
		if existing, err := chunk_retriever.GetIndexName(rdb); err == nil && existing == indexer.IndexName {
			fmt.Printf("Index %q already exists. Use --force to re-index.\n", existing)
			return
		}
	}

	fmt.Println("-------------------------------------------------")
	fmt.Printf("Indexing directory: %s\n", absDir)
	fmt.Printf("Index name:        %s\n", indexer.IndexName)
	if model != "" {
		fmt.Printf("Embedding model:   %s\n", model)
	} else {
		fmt.Printf("Embedding model:   (default)\n")
	}
	fmt.Printf("Chunk size:        %d\n", chunkSize)
	fmt.Printf("Overlap:           %d\n", overlap)
	if force {
		fmt.Printf("Force re-index:    %v\n", force)
	}
	fmt.Println("-------------------------------------------------")

	if err := indexer.ReIndexDirectory(ctx, absDir, chunkSize, overlap); err != nil {
		fmt.Printf("Indexing failed: %v\n", err)
		return
	}

	fmt.Println("Indexing completed")
	fmt.Printf("You can now run:\n  smartcli review -f <file> -q \"what does this do?\"\n")
}
