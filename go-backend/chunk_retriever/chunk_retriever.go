package chunk_retriever

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

type Chunk struct {
	Text     string
	Metadata map[string]string
	// Score is the vector distance (COSINE). Lower = more similar.
	Score float64
}

type ChunkQuery struct {
	Query     string
	IndexName string
	TopK      int
}

func Connect() *redis.Client {
	ctx := context.Background()
	addr := "localhost:6379"
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		addr = v
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		// If you want to force RESP3 consistently, uncomment:
		// Protocol: 3,
	})
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		panic(fmt.Errorf("failed to connect to redis at %s: %w", addr, err))
	}
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
	raw, ok := res.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected FT._LIST response type %T", res)
	}
	indexes := make([]string, 0, len(raw))
	for _, v := range raw {
		switch s := v.(type) {
		case string:
			indexes = append(indexes, s)
		case []byte:
			indexes = append(indexes, string(s))
		default:
			indexes = append(indexes, fmt.Sprintf("%v", v))
		}
	}
	return indexes, nil
}

func GetIndexName(rdb *redis.Client) (string, error) {
	indexes, err := getIndexes(rdb)
	if err != nil {
		return "", err
	}
	if len(indexes) == 0 {
		return "", fmt.Errorf("no indexes found in Redis")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	folderName := filepath.Base(cwd)
	want := folderName + "_index"
	for _, idx := range indexes {
		if idx == want {
			return idx, nil
		}
	}
	return indexes[0], nil
}

// EnsureIndex creates a RediSearch vector index if it does not already exist.
// prefix should be something like "<indexName>:".
func EnsureIndex(rdb *redis.Client, indexName, prefix string, dim int) error {
	// If it exists, do nothing
	indexes, err := getIndexes(rdb)
	if err != nil {
		return err
	}
	for _, idx := range indexes {
		if idx == indexName {
			return nil
		}
	}
	// Create the index
	ctx := context.Background()
	args := []interface{}{
		"FT.CREATE", indexName,
		"ON", "HASH",
		"PREFIX", 1, prefix,
		"SCHEMA",
		"text", "TEXT",
		"file", "TAG",
		"chunk", "NUMERIC",
		"embedding", "VECTOR", "HNSW", 6,
		"TYPE", "FLOAT32",
		"DIM", dim,
		"DISTANCE_METRIC", "COSINE",
	}
	if _, err := rdb.Do(ctx, args...).Result(); err != nil {
		// If someone created it concurrently, ignore "Index already exists"
		if !strings.Contains(strings.ToLower(err.Error()), "exists") {
			return fmt.Errorf("failed to create index %s: %w", indexName, err)
		}
	}
	return nil
}

func float32SliceToLEBytes(vec []float32) []byte {
	buf := make([]byte, 4*len(vec))
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

func RetrieveChunks(rdb *redis.Client, query ChunkQuery, queryEmbedding []float32) ([]Chunk, error) {
	ctx := context.Background()
	vec := float32SliceToLEBytes(queryEmbedding)

	res, err := rdb.Do(
		ctx,
		"FT.SEARCH",
		query.IndexName,
		fmt.Sprintf("*=>[KNN %d @embedding $vec AS vector_score]", query.TopK),
		"PARAMS", 2, "vec", vec,
		"SORTBY", "vector_score",
		"RETURN", 3, "text", "file", "vector_score",
		"LIMIT", 0, query.TopK,
		"DIALECT", 2,
	).Result()
	if err != nil {
		return nil, err
	}
	return parseSearchResults(res)
}

func concurrentChunkRetrieval(rdb *redis.Client,
	queries []ChunkQuery,
	embeddings [][]float32,
	numWorkers int,
) ([]Chunk, error) {

	// Create channels
	queryCh := make(chan struct {
		Query     ChunkQuery
		Embedding []float32
	})
	resultCh := make(chan []Chunk)
	errCh := make(chan error)

	// Spawn workers
	var wg sync.WaitGroup
	numWorkers = 7
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go RetrieveWorker(rdb, queryCh, resultCh, &wg, errCh)
	}
	go func() {
		wg.Wait()
		close(resultCh)
		close(errCh)
	}()
	var allChunks []Chunk
	for res := range resultCh {
		allChunks = append(allChunks, res...)
	}
	var resultErr error
	for err := range errCh {
		fmt.Println("Warning:", err)
		resultErr = err
	}
	return allChunks, resultErr
}

func parseSearchResults(res any) ([]Chunk, error) {
	out := []Chunk{}

	// RESP3: Redis Stack may return a map with "results".
	if m, ok := res.(map[interface{}]interface{}); ok {
		resultsAny := getMapVal(m, "results")
		resultsArr, ok := resultsAny.([]interface{})
		if !ok {
			return out, nil
		}
		for _, item := range resultsArr {
			itMap, ok := item.(map[interface{}]interface{})
			if !ok {
				continue
			}
			extraAny := getMapVal(itMap, "extra_attributes")
			extra, ok := extraAny.(map[interface{}]interface{})
			if !ok {
				continue
			}
			ch := Chunk{Metadata: map[string]string{}}
			for k, v := range extra {
				ks := toString(k)
				vs := toString(v)
				switch ks {
				case "text":
					ch.Text = vs
				case "vector_score":
					if f, err := strconv.ParseFloat(vs, 64); err == nil {
						ch.Score = f
					}
				default:
					ch.Metadata[ks] = vs
				}
			}
			out = append(out, ch)
		}
		return out, nil
	}
	return out, fmt.Errorf("unexpected FT.SEARCH response type: %T", res)
}

func getMapVal(m map[interface{}]interface{}, key string) any {
	for k, v := range m {
		if toString(k) == key {
			return v
		}
	}
	return nil
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// ===== Chunk retriever workers =====

// RetrieveWorker listens for queries on queryCh, runs retrieval, and sends results.
func RetrieveWorker(
	rdb *redis.Client,
	queryCh <-chan struct {
		Query     ChunkQuery
		Embedding []float32
	},
	resultCh chan<- []Chunk,
	wg *sync.WaitGroup,
	errCh chan<- error,
) {
	defer wg.Done()
	for q := range queryCh {
		results, err := RetrieveChunks(rdb, q.Query, q.Embedding)
		if err != nil {
			errCh <- fmt.Errorf("retrieval failed for %q: %w", q.Query.Query, err)
			continue
		}
		resultCh <- results
	}
}
