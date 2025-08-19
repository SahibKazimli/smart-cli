package chunk_retriever

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"os"
	"path/filepath"
)

type Chunk struct {
	Text     string
	Metadata map[string]string
	Score    float64
}

type ChunkQuery struct {
	Query     string
	IndexName string
	TopK      int
}

func Connect() *redis.Client {
	ctx := context.Background()
	// Connecting to the redis database
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println("Redis ping:", pong)
	return rdb
}

func PrepareQuery(queryText string, topK int) ChunkQuery {
	// Read in files dynamically based on the directory the user is in
	cwd, _ := os.Getwd()
	folderName := filepath.Base(cwd)
	indexName := folderName + "_index"

	return ChunkQuery{
		Query:     queryText,
		IndexName: indexName,
		TopK:      topK,
	}
}

func RetrieveChunks(rdb *redis.Client, query ChunkQuery) ([]Chunk, error) {
	ctx := context.Background()

	// Starting with a simple search method with just text
	// Will sort by vector similarity
	// FT.SEARCH
	args := []interface{}{
		"FT.SEARCH",
		query.IndexName,
		"*", // match everything
		"RETURN", "2", "text", "metadata",
		"LIMIT", "0", query.TopK,
	}

	res, err := rdb.Do(ctx, args...).Result()
	if err != nil {
		return nil, err
	}
	results := []Chunk{}

	// Parsing reply
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 2 {
		return results, nil
	}

	// Iterate over the search results returned by Redis
	// Skip the first element since it's the total count of matches
	for i := 1; i < len(arr); i += 2 {
		fields, _ := arr[i+1].([]interface{})
		ch := Chunk{Metadata: map[string]string{}}

		// Iterate over the key-value pairs for this result
		for j := 0; j < len(fields); j += 2 {
			var key string
			switch k := fields[j].(type) {
			case []byte:
				key = string(k)
			case string:
				key = k
			default:
				key = fmt.Sprintf("%v", k)
			}

			val := fields[j+1]
			var valStr string
			switch v := val.(type) {
			case []byte:
				valStr = string(v)
			case string:
				valStr = v
			default:
				valStr = fmt.Sprintf("%v", v)
			}

			// Assign text to the Text field, everything else goes into Metadata
			switch key {
			case "text":
				ch.Text = valStr
			default:
				ch.Metadata[key] = valStr
			}
		}
		results = append(results, ch)
	}

	return results, nil
}
