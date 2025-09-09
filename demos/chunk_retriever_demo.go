package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/embedder"
)

type QueryResult struct {
	Query     string
	Chunks    []chunk_retriever.Chunk
	Err       error
	Elapsed   time.Duration
	IndexName string
}

func main() {
	_ = godotenv.Load(".env")

	projectID := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION")
	creds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	fmt.Println("GCP_PROJECT_ID:", projectID)
	fmt.Println("GCP_LOCATION:", location)
	fmt.Println("GOOGLE_APPLICATION_CREDENTIALS:", creds)
	if projectID == "" || location == "" || creds == "" {
		log.Fatal("Missing one of GCP_PROJECT_ID, GCP_LOCATION, GOOGLE_APPLICATION_CREDENTIALS")
	}

	// Connect Redis
	rdb := chunk_retriever.Connect()

	indexName, err := chunk_retriever.GetIndexName(rdb)
	if err != nil {
		log.Fatalf("Failed to discover index name: %v", err)
	}
	log.Printf("Using index name: %s", indexName)

	ctx := context.Background()

	// Embedder client
	embeddingModel := "text-embedding-005"
	emb, err := embedder.EmbedderClient(ctx, creds, rdb, embeddingModel)
	if err != nil {
		log.Fatalf("Failed to init embedder: %v", err)
	}

	queries := []string{
		"What does embedder.go do?",
		"How are files chunked?",
		"Explain the Redis storage schema.",
		"How are query embeddings generated?",
	}

	topK := 5
	concurrency := 4

	overallStart := time.Now() // start timing for all queries
	results := retrieveConcurrent(ctx, emb, rdb, indexName, queries, topK, concurrency)
	overallElapsed := time.Since(overallStart)

	fmt.Printf("\n=== All queries completed in %s ===\n", overallElapsed)

	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("\n--- Query: %q (ERROR) ---\n%v\n", r.Query, r.Err)
			continue
		}
		fmt.Printf("\n--- Query: %q | %d chunks | %s | index=%s ---\n",
			r.Query, len(r.Chunks), r.Elapsed, r.IndexName)
		if len(r.Chunks) == 0 {
			fmt.Println("No chunks retrieved.")
			continue
		}
		for i, ch := range r.Chunks {
			file := ch.Metadata["file"]
			fmt.Printf("[#%d score=%.4f file=%s]\n%s\n---\n", i, ch.Score, file, ch.Text)
		}
	}
}

// retrieveConcurrent runs multiple semantic retrievals concurrently.
func retrieveConcurrent(
	ctx context.Context,
	emb *embedder.Embedder,
	rdb *redis.Client, // using any to avoid tight coupling to redis.Client type signature here
	indexName string,
	queries []string,
	topK int,
	maxParallel int,
) []QueryResult {

	type task struct {
		Query string
	}
	tasks := make(chan task)
	resultsCh := make(chan QueryResult)

	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for t := range tasks {
			start := time.Now()
			// Embed query
			queryVec, err := emb.EmbedQuery(t.Query)
			if err != nil {
				resultsCh <- QueryResult{Query: t.Query, Err: fmt.Errorf("embed error: %w", err)}
				continue
			}
			// Prepare + retrieve
			cq := chunk_retriever.PrepareQuery(t.Query, topK, indexName)
			chunks, err := chunk_retriever.RetrieveChunks(rdb, cq, queryVec)
			if err != nil {
				resultsCh <- QueryResult{Query: t.Query, Err: fmt.Errorf("retrieve error: %w", err)}
				continue
			}
			// Sort by Score ascending (lower = more similar per your struct)
			sort.Slice(chunks, func(i, j int) bool {
				return chunks[i].Score < chunks[j].Score
			})
			elapsed := time.Since(start)
			resultsCh <- QueryResult{
				Query:     t.Query,
				Chunks:    chunks,
				Elapsed:   elapsed,
				IndexName: indexName,
			}
		}
	}

	// Start workers
	if maxParallel <= 0 {
		maxParallel = 1
	}
	if maxParallel > len(queries) {
		maxParallel = len(queries)
	}
	wg.Add(maxParallel)
	for i := 0; i < maxParallel; i++ {
		go worker()
	}

	// Feed tasks
	go func() {
		for _, q := range queries {
			tasks <- task{Query: q}
		}
		close(tasks)
	}()

	// Close results channel when workers done
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var collected []QueryResult
	for r := range resultsCh {
		collected = append(collected, r)
	}
	return collected
}
