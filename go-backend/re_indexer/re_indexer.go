package re_indexer

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/redis/go-redis/v9"
	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/chunker"
	"smart-cli/go-backend/embedder"
)

type Indexer struct {
	Redis     *redis.Client
	Embedder  *embedder.Embedder
	Root      string
	IndexName string

	ensureOnce sync.Once
}

func NewIndexer(redisClient *redis.Client, emb *embedder.Embedder, root string, indexName string) *Indexer {
	if root == "" {
		root = "."
	}
	if indexName == "" {
		indexName = filepath.Base(root) + "_index"
	}
	return &Indexer{
		Redis:     redisClient,
		Embedder:  emb,
		Root:      root,
		IndexName: indexName,
	}
}

func (i *Indexer) ensureIndex(dim int) error {
	var err error
	i.ensureOnce.Do(func() {
		prefix := i.IndexName + ":"
		err = chunk_retriever.EnsureIndex(i.Redis, i.IndexName, prefix, dim)
	})
	return err
}

func (i *Indexer) IndexFile(ctx context.Context, path string, chunkSize int, overlap int) error {
	chunks, err := chunker.SplitFile(path, chunkSize, overlap)
	if err != nil {
		return err
	}
	if len(chunks) == 0 {
		return fmt.Errorf("warning: no chunks produced from file %s", path)
	}
	for _, chunk := range chunks {
		vector, err := i.Embedder.EmbedText(chunk.Text)
		if err != nil {
			fmt.Printf("Warning: failed embedding chunk %d: %v\n", chunk.Index, err)
			continue
		}
		// Ensure the vector index exists once, using the first vector's dimension
		if err := i.ensureIndex(len(vector)); err != nil {
			return fmt.Errorf("failed to ensure index %q: %w", i.IndexName, err)
		}
		if err := i.storeChunk(ctx, path, chunk.Index, chunk.Text, vector); err != nil {
			fmt.Printf("Warning: failed storing chunk %d: %v\n", chunk.Index, err)
			continue
		}
	}
	return nil
}

func (i *Indexer) ReIndexDirectory(ctx context.Context, dir string, chunkSize, overlap int) error {
	files, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return err
	}
	for _, file := range files {
		// Skip directories
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			continue
		}
		fmt.Printf("Indexing file: %s\n", file)
		if err := i.IndexFile(ctx, file, chunkSize, overlap); err != nil {
			fmt.Printf("Warning: failed indexing file %s: %v\n", file, err)
		}
	}
	return nil
}

// ===== Helpers =====

// storeChunk saves a single chunk in Redis under a simple key
func (ix *Indexer) storeChunk(ctx context.Context, filePath string, chunkNo int, text string, vec []float32) error {
	key := fmt.Sprintf("%s:%s:%d", ix.IndexName, filepath.Base(filePath), chunkNo)
	return ix.Redis.HSet(ctx, key, map[string]any{
		"text":      text,
		"file":      filePath,
		"chunk":     chunkNo,
		"embedding": float32ToBytes(vec),
	}).Err()
}

// float32ToBytes converts a float32 slice to little-endian bytes
func float32ToBytes(vec []float32) []byte {
	b := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(v))
	}
	return b
}
