package re_indexer

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"path/filepath"
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
		for vector, err := i.Embedder.EmbedText(chunk.Text)
		if err != nil {
			fmt.Printf("Warning: failed embedding chunk")
			continue
		}

		}
	}

}
