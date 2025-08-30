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
	projectID := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION")
	embeddingModel := "text-embedding-005"
	generationModel := "gemini-2.5-pro"
	creds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	if projectID == "" || location == "" || creds == "" {
		log.Fatal("Please set GCP_PROJECT_ID, GCP_LOCATION, and GOOGLE_APPLICATION_CREDENTIALS")
	}

	log.Printf("Using Project ID: %s, Location: %s", projectID, location)

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
	defer func() {
		if err := agent.Close(); err != nil {
			log.Printf("Error: Closing agent: %v", err)
		}
	}()
	log.Println("Agent Initialized")

	// Initialize embedder
	embedClient, err := embedder.EmbedderClient(ctx, creds, rdb, embeddingModel)
	if err != nil {
		log.Fatalf("Failed to create embedder client: %v", err)
	}

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

	// Build context from chunks
	var builder strings.Builder
	for _, ch := range chunks {
		file := ch.Metadata["file"]
		builder.WriteString(fmt.Sprintf("File: %s\n%s\n\n", file, ch.Text))
	}
	contextText := builder.String()

	// Print retrieved context
	fmt.Println("=== Retrieved Context ===")
	fmt.Println(contextText)

	// Generate answer using the retrieved chunks
	response, err := agent.Answer(ctx, query, chunks)
	if err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	// Print response
	fmt.Println("=== LLM Response ===")
	fmt.Println(response)
}
