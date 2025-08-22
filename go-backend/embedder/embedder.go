package embedder

import (
	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"context"
	"github.com/redis/go-redis/v9"
	"google.golang.org/api/option"
	"io/fs"
	"os"
	"path/filepath"
)

type FileEmbedding struct {
	Path      string
	Content   string
	Embedding []float32
}

// Embedder manages embedding files with Vertex AI and storing in Redis
type Embedder struct {
	Client        *aiplatform.PredictionClient
	RDB           *redis.Client
	Ctx           context.Context
	ModelEndpoint string
}

// EmbedderClient creates a new Embedder instance with the Vertex AI Prediction client, Redis client,
// and context, then returns a pointer to it along with nil error.
func EmbedderClient(ctx context.Context, credsFile string, rdb *redis.Client, modelEndpoint string) (*Embedder, error) {
	client, err := aiplatform.NewPredictionClient(ctx, option.WithCredentialsFile(credsFile))
	if err != nil {
		return nil, err
	}

	return &Embedder{
		Client:        client,
		RDB:           rdb,
		Ctx:           ctx,
		ModelEndpoint: modelEndpoint,
	}, nil
}

// ReadDirectory Will walk through the current directory to read in content
func ReadDirectory(dir string, extensions []string) (fileContents []string, err error) {
	newErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// filter by extension
		ext := filepath.Ext(path)
		if ext == ".go" || ext == ".py" || ext == ".js" || ext == ".cpp" || ext == ".txt" {
			// process file
			ctn, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(ctn)
			fileContents = append(fileContents, text)
		}
		return nil
	})
	if newErr != nil {
		return nil, newErr
	}
	return
}
