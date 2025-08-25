package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/embedder"

	"github.com/joho/godotenv"
)

func loadEnv() error {
	// Try to load .env file from current directory
	if err := godotenv.Load("/Users/sahibkazimli/go-projects/smart-cli/.env"); err != nil {
		// Try to load from parent directory (go-backend)
		if err := godotenv.Load("../.env"); err != nil {
			// Try to load from project root
			if err := godotenv.Load("../../.env"); err != nil {
				fmt.Println("⚠️  No .env file found, using system environment variables")
			} else {
				fmt.Println("✅ Loaded .env from project root")
			}
		} else {
			fmt.Println("✅ Loaded .env from go-backend directory")
		}
	} else {
		fmt.Println("✅ Loaded .env from current directory")
	}
	return nil
}

func main() {
	fmt.Println("🧪 Testing chunk_retriever and embedder functionality...")
	fmt.Println("=====================================================")

	// Load environment variables
	fmt.Println("\n1. Loading environment variables...")
	if err := loadEnv(); err != nil {
		log.Printf("Warning: Failed to load .env file: %v", err)
	}

	// Check environment variables
	fmt.Println("\n2. Checking environment variables...")
	projectID := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION")
	credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	if projectID == "" {
		log.Fatal("❌ GCP_PROJECT_ID environment variable not set")
	}
	if location == "" {
		log.Fatal("❌ GCP_LOCATION environment variable not set")
	}
	if credsFile == "" {
		log.Fatal("❌ GOOGLE_APPLICATION_CREDENTIALS environment variable not set")
	}

	fmt.Printf("✅ GCP_PROJECT_ID: %s\n", projectID)
	fmt.Printf("✅ GCP_LOCATION: %s\n", location)
	fmt.Printf("✅ GOOGLE_APPLICATION_CREDENTIALS: %s\n", credsFile)

	// Check if credentials file exists
	if _, err := os.Stat(credsFile); os.IsNotExist(err) {
		log.Fatalf("❌ Credentials file not found: %s", credsFile)
	}
	fmt.Printf("✅ Credentials file exists: %s\n", credsFile)

	// Test 3: Connect to Redis
	fmt.Println("\n3. Testing Redis connection...")
	rdb := chunk_retriever.Connect()
	if rdb == nil {
		log.Fatal("❌ Failed to connect to Redis")
	}
	fmt.Println("✅ Redis connection successful")

	// Test 4: Initialize Embedder
	fmt.Println("\n4. Testing embedder initialization...")
	ctx := context.Background()
	model := "gemini-embedding-001"
	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", projectID, location, model)

	fmt.Printf("   Model endpoint: %s\n", endpoint)

	embedClient, err := embedder.EmbedderClient(ctx, credsFile, rdb, endpoint)
	if err != nil {
		log.Fatalf("❌ Failed to create embedder client: %v", err)
	}
	fmt.Println("✅ Embedder client initialized successfully")

	// Test 5: Test query embedding
	fmt.Println("\n5. Testing query embedding...")
	query := "What does build_index.py do?"
	fmt.Printf("Query: '%s'\n", query)

	queryEmbedding, err := embedClient.EmbedQuery(query)
	if err != nil {
		log.Fatalf("❌ Query embedding failed: %v", err)
	}

	fmt.Printf("✅ Query embedded successfully! Vector length: %d\n", len(queryEmbedding))
	if len(queryEmbedding) >= 5 {
		fmt.Printf("   First 5 values: [%f, %f, %f, %f, %f]\n",
			queryEmbedding[0], queryEmbedding[1], queryEmbedding[2], queryEmbedding[3], queryEmbedding[4])
	} else {
		fmt.Printf("   All values: %v\n", queryEmbedding)
	}

	// Test 6: Get Redis index
	fmt.Println("\n6. Testing Redis index retrieval...")
	indexName, err := chunk_retriever.GetIndexName(rdb)
	if err != nil {
		log.Fatalf("❌ Failed to get index name: %v", err)
	}
	fmt.Printf("✅ Using index: %s\n", indexName)

	// Test 7: Test vector similarity search
	fmt.Println("\n7. Testing vector similarity search...")
	chunkQuery := chunk_retriever.PrepareQuery(query, 10, indexName)

	fmt.Printf("   TopK: %d\n", chunkQuery.TopK)
	fmt.Printf("   Index: %s\n", chunkQuery.IndexName)

	// Run retrieval (debug prints removed)
	chunks, err := chunk_retriever.RetrieveChunks(rdb, chunkQuery, queryEmbedding)
	if err != nil {
		log.Fatalf("❌ Vector similarity search failed: %v", err)
	}

	// Test 8: Analyze results
	fmt.Println("\n8. Analyzing search results...")
	fmt.Printf("✅ Found %d chunks for query: '%s'\n", len(chunks), query)

	if len(chunks) == 0 {
		fmt.Println("⚠️  No chunks found - this might be expected if:")
		fmt.Println("   - No data exists in Redis")
		fmt.Println("   - The index is empty")
		fmt.Println("   - The query doesn't match any existing data")
		fmt.Println("\n💡 To populate Redis with data, you can:")
		fmt.Println("   - Use the embedder to process files in your directory")
		fmt.Println("   - Create a Redis index with some test data")
	} else {
		fmt.Println("\n📋 Retrieved chunks:")
		for i, chunk := range chunks {
			fmt.Printf("\n--- Chunk %d ---\n", i+1)
			fmt.Printf("Text: %s\n", chunk.Text)
			fmt.Printf("Score: %f\n", chunk.Score)
			fmt.Printf("Metadata: %+v\n", chunk.Metadata)

			// Validate chunk structure
			if chunk.Text == "" {
				fmt.Printf("⚠️  Warning: Chunk %d has empty text\n", i+1)
			}
			if chunk.Metadata == nil {
				fmt.Printf("⚠️  Warning: Chunk %d has nil metadata\n", i+1)
			}
			if chunk.Score < 0 || chunk.Score > 1 {
				fmt.Printf("⚠️  Warning: Chunk %d has unusual score: %f (expected 0-1)\n", i+1, chunk.Score)
			}
		}
	}

	fmt.Println("\n🎉 Testing completed successfully!")
	fmt.Println("\n📝 Summary:")
	fmt.Printf("   - Environment variables: ✅ Loaded\n")
	fmt.Printf("   - Embedder: ✅ Working\n")
	fmt.Printf("   - Redis connection: ✅ Working\n")
	fmt.Printf("   - Vector similarity search: ✅ Working\n")
	fmt.Printf("   - Chunks found: %d\n", len(chunks))

	if len(chunks) == 0 {
		fmt.Println("\n🔧 Next steps:")
		fmt.Println("   1. Ensure Redis has data in the index")
		fmt.Println("   2. Check that the index contains relevant documents")
		fmt.Println("   3. Verify the embedding model is working correctly")
	}
}
