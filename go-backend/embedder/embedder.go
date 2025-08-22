package embedder

import (
	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
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

// FileData represents a file read from disk
type FileData struct {
	Path    string
	Content string
}

// EmbedderClient creates a new Embedder instance with the Vertex AI Prediction client, Redis client,
// and context, then returns a pointer to it along with nil error.
func EmbedderClient(ctx context.Context, credsFile string, rdb *redis.Client, modelEndpoint string) (*Embedder, error) {
	// Initialize Vertex AI Prediction client using a service account JSON key
	client, err := aiplatform.NewPredictionClient(ctx, option.WithCredentialsFile(credsFile))
	if err != nil {
		return nil, err
	}

	// Load in necessary ID's for embedding model initialization
	projectID := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION")
	model := "textembedding-gecko@003"

	endpoint := fmt.Sprintf(
		"projects/%s/locations/%s/publishers/google/models/%s",
		projectID, location, model,
	)

	return &Embedder{
		Client:        client,
		RDB:           rdb,
		Ctx:           ctx,
		ModelEndpoint: endpoint,
	}, nil
}

// ReadDirectory Will walk through the current directory to read in content
func ReadDirectory(dir string, extensions []string) (files []FileData, err error) {
	// Recursively walks the directory tree starting at dir
	// Filters by given extensions
	// Returns a slice of FileData structs containing file paths and content
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
			files = append(files, FileData{
				Path:    path,
				Content: text,
			})
		}
		return nil
	})
	if newErr != nil {
		return nil, newErr
	}
	return
}

func (e *Embedder) EmbedDirectory(dir string, extensions []string) ([]FileEmbedding, error) {
	/*
		Calls ReadDirectory and reads from all files in the working directory.
		For each file, calls Vertex AI to generate embeddings.
		Parses the predictions response to get the vector.
	*/

	files, err := ReadDirectory(dir, extensions)
	if err != nil {
		return nil, err
	}
	var embeddings []FileEmbedding
	for _, file := range files {
		fmt.Println("Processing file:", file.Path)

		// Wrap the file content for Vertex AI gRPC request
		instance, err := structpb.NewStruct(map[string]interface{}{
			"content": file.Content,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create struct for %s: %w", file.Path, err)
		}
		// Create PredictRequest to call the embedding model
		request := &aiplatformpb.PredictRequest{
			Endpoint:  e.ModelEndpoint,
			Instances: []*structpb.Value{structpb.NewStructValue(instance)},
		}

		resp, err := e.Client.Predict(e.Ctx, request)
		if err != nil {
			return nil, fmt.Errorf("prediction failed for %s: %w", file.Path, err)
		}
		// Check that the response has predictions
		if len(resp.Predictions) == 0 {
			return nil, fmt.Errorf("no predictions returned for %s", file.Path)
		}

		predStruct := resp.Predictions[0].GetStructValue()
		if predStruct == nil {
			return nil, fmt.Errorf("prediction is not a struct for %s", file.Path)
		}
		// Extract the "embedding" field
		embeddingField, ok := predStruct.Fields["embedding"]
		if !ok {
			return nil, fmt.Errorf("embedding field missing in prediction for %s", file.Path)
		}

		listValue := embeddingField.GetListValue()
		if listValue == nil {
			return nil, fmt.Errorf("embedding field is not a list for %s", file.Path)
		}
		// Convert embedding to slice of type float32
		embedding := make([]float32, len(listValue.Values))
		for i, v := range listValue.Values {
			number := v.GetNumberValue()
			embedding[i] = float32(number)
		}
		// Append the embedding to the result slice
		embeddings = append(embeddings, FileEmbedding{
			Path:      file.Path,
			Content:   file.Content,
			Embedding: embedding,
		})

	}
	return embeddings, nil
}
