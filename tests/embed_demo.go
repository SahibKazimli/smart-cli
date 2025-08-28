package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"smart-cli/go-backend/embedder"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: could not load .env file, falling back to system env")
	}

	ctx := context.Background()

	// Setup Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	if pong, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	} else {
		fmt.Println("Redis ping:", pong)
	}

	// Create embedder client
	creds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	project := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION")

	if creds == "" || project == "" || location == "" {
		log.Fatal("Missing required GCP environment variables")
	}

	embedClient, err := embedder.EmbedderClient(ctx, creds, rdb, "")
	if err != nil {
		log.Fatalf("Failed to create embedder client: %v", err)
	}

	// Run embedding for project directory
	embeddings, err := embedClient.EmbedDirectory("./", []string{".go", ".py"})
	if err != nil {
		log.Fatalf("Failed to embed directory: %v", err)
	}

	// Store in Redis
	for i, e := range embeddings {
		key := fmt.Sprintf("smart-cli_index:%d", i)
		err := rdb.HSet(ctx, key, map[string]interface{}{
			"text":      e.Content,
			"embedding": e.Embedding,
			"file":      e.Path,
			"chunk":     i,
		}).Err()
		if err != nil {
			log.Printf("Failed to store embedding for %s: %v", e.Path, err)
		} else {
			fmt.Printf("Stored embedding for %s\n", e.Path)
		}
	}

	fmt.Println("Embedding completed and stored in Redis")
}
