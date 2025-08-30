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
	// Load env
	_ = godotenv.Load("/Users/sahibkazimli/go-projects/smart-cli/.env")

	ctx := context.Background()

	// Connect Redis with safe fallback
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	addr := "127.0.0.1:6379"
	if host != "" && port != "" {
		addr = host + ":" + port
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("Redis ping failed at %s: %v", addr, err)
	}

	creds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	project := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION")
	if creds == "" || project == "" || location == "" {
		log.Fatal("Missing required envs: GOOGLE_APPLICATION_CREDENTIALS, GCP_PROJECT_ID, GCP_LOCATION")
	}

	// Build embedder
	emb, err := embedder.EmbedderClient(ctx, creds, rdb, "text-embedding-005")
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	// Embed from project root and index under a stable name
	indexName := "smart-cli_index"
	idx, count, err := emb.EmbedAndIndex("", indexName, []string{".go", ".py"})
	if err != nil {
		log.Fatalf("EmbedAndIndex failed: %v", err)
	}
	fmt.Printf("Indexed %d files into %s\n", count, idx)
}
