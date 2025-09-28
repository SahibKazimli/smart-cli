package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/embedder"
	"smart-cli/go-backend/generator"
	"smart-cli/go-backend/re_indexer"
)

func main() {
	// Load environment variables from .env if present
	if err := godotenv.Load("/Users/sahibkazimli/go-projects/smart-cli/.env"); err != nil {
		log.Println("Could not load .env file, relying on system environment")
	}

	// CLI flags
	dirFlag := flag.String("dir", ".", "Directory to index/query (project root auto-detected)")
	reindexFlag := flag.Bool("reindex", false, "Whether to manually re-index the project before querying")
	flag.Parse()

	// Required environment variables
	projectID := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION")
	creds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if projectID == "" || location == "" || creds == "" {
		log.Fatal("Please set GCP_PROJECT_ID, GCP_LOCATION, and GOOGLE_APPLICATION_CREDENTIALS")
	}

	// Connect to Redis
	log.Println("Connecting to Redis...")
	rdb := chunk_retriever.Connect()
	if rdb == nil {
		log.Fatal("Failed to connect to Redis")
	}
	log.Println("✓ Connected to Redis")

	ctx := context.Background()

	// Initialize Embedder
	embeddingModel := "text-embedding-005"
	embedClient, err := embedder.EmbedderClient(ctx, creds, rdb, embeddingModel)
	if err != nil {
		log.Fatalf("Failed to create embedder client: %v", err)
	}
	log.Println("✓ Embedder initialized")

	// Initialize LLM agent
	log.Println("Initializing LLM agent...")
	agent, err := generator.NewAgent(ctx, "")
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	log.Printf("✓ Agent Initialized with model: %s", agent.ModelName())

	// Manual re-index if requested
	if *reindexFlag {
		log.Println("Manual re-index requested...")
		indexer := re_indexer.NewIndexer(rdb, embedClient, *dirFlag, "")
		if err := indexer.ReIndexDirectory(ctx, *dirFlag, 800, 50); err != nil {
			log.Fatalf("Re-indexing failed: %v", err)
		}
		log.Println("✓ Re-indexing completed")
	}

	// Prepare a sample query
	query := "What does embedder.go do?"

	// Retrieve index name
	indexName, err := chunk_retriever.GetIndexName(rdb)
	if err != nil {
		log.Fatalf("Failed to get Redis index: %v", err)
	}

	// Embed query
	queryEmbedding, err := embedClient.EmbedQuery(query)
	if err != nil {
		log.Fatalf("Query embedding failed: %v", err)
	}

	// Retrieve relevant chunks
	chunkQuery := chunk_retriever.PrepareQuery(query, 5, indexName)
	chunks, err := chunk_retriever.RetrieveChunks(rdb, chunkQuery, queryEmbedding)
	if err != nil {
		log.Fatalf("Chunk retrieval failed: %v", err)
	}

	// Build context from chunks
	var builder strings.Builder
	for _, ch := range chunks {
		file := ch.Metadata["file"]
		builder.WriteString(fmt.Sprintf("File: %s\n%s\n\n", file, ch.Text))
	}
	contextText := builder.String()

	fmt.Println("=== Retrieved Context ===")
	fmt.Println(contextText)

	// Generate answer using the retrieved chunks
	response, err := agent.Answer(ctx, query, chunks)
	if err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	fmt.Println("=== LLM Response ===")
	fmt.Println(response)
}
