package re_indexer

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"strings"
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

func (i *Indexer) IndexFile(ctx context.Context, path string, chunkSize int, overlap int) (int, error) {
	chunks, err := chunker.SplitFile(path, 600, 200)
	if err != nil {
		return 0, err
	}
	if len(chunks) == 0 {
		return 0, fmt.Errorf("Warning: no chunks produced from file %s", path)
	}
	for _, chunk := range chunks {
			vector, err := i.Embedder.EmbedText(chunk.Text)
			if err != nil {
				fmt.Printf("Warning: failed embedding chunk")
				continue
			}

		}
	}
}

// ===== Helpers =====

// float32ToBytes converts a float32 slice to little-endian bytes
func float32ToBytes(vec []float32) []byte {
	b := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(v))
	}
	return b
}

