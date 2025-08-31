package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/embedder"
	"smart-cli/go-backend/generator"
)

func main() {
	// Load environment variables
	err := godotenv.Load("/Users/sahibkazimli/go-projects/smart-cli/.env")
	if err != nil {
		log.Println("Could not load .env file, relying on system environment")
	}

	// For the current generator.go, you need GEMINI_API_KEY
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set GOOGLE_API_KEY environment variable")
	}

	// These are still needed for the embedder
	projectID := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION")
	creds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	embeddingModel := "text-embedding-005"
	generationModel := "gemini-1.5-pro" // Use a stable model

	log.Printf("Using Generation Model: %s", generationModel)

	// Connect to Redis
	log.Println("Connecting to Redis...")
	rdb := chunk_retriever.Connect()
	if rdb == nil {
		log.Fatal("Failed to connect to Redis")
	}
	log.Println("✓ Connected to Redis")

	// Initialize LLM agent
	ctx := context.Background()
	log.Printf("Initializing LLM agent with model: %s", generationModel)
	agent, err := generator.NewAgent(ctx, generationModel)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	// Note: No Close() method available in current implementation
	log.Println("✓ Agent Initialized")

	// Initialize embedder (only if you have GCP credentials)
	if projectID != "" && location != "" && creds != "" {
		embedClient, err := embedder.EmbedderClient(ctx, creds, rdb, embeddingModel)
		if err != nil {
			log.Fatalf("Failed to create embedder client: %v", err)
		}
		log.Println("✓ Embedder initialized")

		// Prepare query
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

		// Print retrieved context
		fmt.Println("\n=== Retrieved Context ===")
		for i, ch := range chunks {
			file := ch.Metadata["file"]
			fmt.Printf("\n--- Chunk %d (Score: %.4f) ---\n", i+1, ch.Score)
			fmt.Printf("File: %s\n", file)
			fmt.Printf("Content:\n%s\n", strings.TrimSpace(ch.Text))
		}

		// Generate answer using the retrieved chunks
		fmt.Println("\n=== Generating LLM Response ===")
		response, err := agent.Answer(ctx, query, chunks)
		if err != nil {
			log.Fatalf("Generation failed: %v", err)
		}

		// Print response
		fmt.Println("\n=== LLM Response ===")
		fmt.Println(response)
	} else {
		// Test with empty chunks if no embedder available
		log.Println("No GCP credentials found, testing with empty chunks...")

		query := "Explain what a command-line interface is."
		var emptyChunks []chunk_retriever.Chunk

		fmt.Printf("\nQuery: %s\n", query)
		response, err := agent.Answer(ctx, query, emptyChunks)
		if err != nil {
			log.Fatalf("Generation failed: %v", err)
		}

		fmt.Println("\n=== LLM Response ===")
		fmt.Println(response)
	}
}
