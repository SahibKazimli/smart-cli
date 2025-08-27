package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"

	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/embedder"
	"smart-cli/go-backend/generator"
)

func main() {
	// 1. Load environment variables
	err := godotenv.Load("/Users/sahibkazimli/go-projects/smart-cli/.env")
	if err != nil {
		log.Println("⚠️  Could not load .env file, relying on system environment")
	}
	projectID := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION")
	modelName := "gemini-embedding-001" // or your LLM model
	creds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	if projectID == "" || location == "" || creds == "" {
		log.Fatal("Please set GCP_PROJECT_ID, GCP_LOCATION, and GOOGLE_APPLICATION_CREDENTIALS")
	}

	// 2. Connect to Redis
	rdb := chunk_retriever.Connect()

	// 3. Initialize the LLM agent
	ctx := context.Background()
	agent, err := generator.NewAgent(ctx, projectID, location, modelName)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// 4. Initialize embedder client
	embedClient, err := embedder.EmbedderClient(ctx, creds, rdb, modelName)
	if err != nil {
		log.Fatalf("Failed to create embedder client: %v", err)
	}

	// 5. Prepare your query
	query := "What does build_index.py do?"

	// 6. Retrieve chunks from Redis
	indexName, err := chunk_retriever.GetIndexName(rdb)
	if err != nil {
		log.Fatalf("Failed to get Redis index: %v", err)
	}

	chunkQuery := chunk_retriever.PrepareQuery(query, 5, indexName)
	queryEmbedding, err := embedClient.EmbedQuery(query)
	if err != nil {
		log.Fatalf("Query embedding failed: %v", err)
	}

	chunks, err := chunk_retriever.RetrieveChunks(rdb, chunkQuery, queryEmbedding)
	if err != nil {
		log.Fatalf("Chunk retrieval failed: %v", err)
	}

	// 6. Ask the LLM to answer based on chunks
	answer, err := agent.Answer(ctx, query, chunks)
	if err != nil {
		log.Fatalf("LLM failed to answer: %v", err)
	}

	// 7. Print the result
	fmt.Println("=== LLM Response ===")
	fmt.Println(answer)
}
