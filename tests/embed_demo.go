package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"smart-cli/go-backend/chunk_retriever"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"smart-cli/go-backend/embedder"
)

func float32ToLEBytes(vec []float32) []byte {
	buf := make([]byte, 4*len(vec))
	for i, v := range vec {
		u := math.Float32bits(v)
		buf[i*4+0] = byte(u)
		buf[i*4+1] = byte(u >> 8)
		buf[i*4+2] = byte(u >> 16)
		buf[i*4+3] = byte(u >> 24)
	}
	return buf
}

func main() {
	// Load environment variables
	err := godotenv.Load("/Users/sahibkazimli/go-projects/smart-cli/.env")
	if err != nil {
		log.Println("Warning: could not load .env file, falling back to system env")
	}

	ctx := context.Background()

	// Setup Redis with safe fallback
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
	if pong, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("Failed to connect to Redis at %s: %v", addr, err)
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

	// Derive index name with cwd
	cwd, _ := os.Getwd()
	indexName := filepath.Base(cwd) + "_index"
	prefix := indexName + ":"

	// Ensure FT index exists with correct dimension
	dim := len(embeddings[0].Embedding)
	if err := chunk_retriever.EnsureIndex(rdb, indexName, prefix, dim); err != nil {
		log.Fatalf("Failed to ensure index: %v", err)
	}
	fmt.Printf("Index ensured: %s (DIM=%d)\n", indexName, dim)

	// Store in Redis
	for i, e := range embeddings {
		key := fmt.Sprintf("%s%d", prefix, i)
		vec := float32ToLEBytes(e.Embedding)

		err := rdb.HSet(ctx, key, map[string]interface{}{
			"text":      e.Content,
			"embedding": vec,
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
