package chunk_retriever

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/redis/go-redis/v9"
	"math"
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

func PrepareQuery(queryText string, topK int, indexName string) ChunkQuery {
	return ChunkQuery{
		Query:     queryText,
		IndexName: indexName,
		TopK:      topK,
	}
}

func getIndexes(rdb *redis.Client) ([]string, error) {
	ctx := context.Background()
	res, err := rdb.Do(ctx, "FT._LIST").Result()
	if err != nil {
		return nil, err
	}
	var indexes []string
	for _, v := range res.([]interface{}) {
		indexes = append(indexes, fmt.Sprintf("%s", v))
	}
	return indexes, nil
}

func GetIndexName(rdb *redis.Client) (string, error) {
	// Get all Redis indexes
	indexes, err := getIndexes(rdb)
	if err != nil {
		return "", err
	}

	if len(indexes) == 0 {
		return "", fmt.Errorf("no indexes found in Redis")
	}

	// Get the current folder name
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	folderName := filepath.Base(cwd)

	// Try to find an index that matches the folder
	for _, idx := range indexes {
		if idx == folderName+"_index" {
			return idx, nil
		}
	}

	// Fallback: return the first available index
	return indexes[0], nil
}

// RetrieveChunks searches Redis for the top K most similar chunks to the given query.
// It uses vector similarity search (KNN) and returns a slice of Chunk structs.
func RetrieveChunks(rdb *redis.Client, query ChunkQuery, queryEmbedding []float32) ([]Chunk, error) {
	ctx := context.Background()
	// We need to convert the embedded query to a format redis understands
	embeddedQueryByte := make([]byte, 4*len(queryEmbedding))
	for idx, val := range queryEmbedding {
		binary.LittleEndian.PutUint32(embeddedQueryByte[idx*4:], math.Float32bits(val))
	}
	// Construct Redis search arguments for KNN
	args := []interface{}{
		"FT.SEARCH",
		query.IndexName,
		fmt.Sprintf("*=>[KNN %d @embedding $vec AS score]", query.TopK),
		"PARAMS", "1", "vec", embeddedQueryByte,
		"RETURN", "3", "text", "metadata", "score",
		"SORTBY", "score",
		"LIMIT", "0", query.TopK,
	}
	// Execute the search
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

		// Convert each key-value pair to string and populate Chunk struct
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
