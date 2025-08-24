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
	model := "gemini-embedding-001"

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

func shouldSkipDir(name string) bool {
	// A helper to decide whether a dir should be skipped for processing
	skipDirs := map[string]struct{}{
		"venv": {}, "__pycache__": {}, "node_modules": {}, ".git": {},
	}
	_, skip := skipDirs[name]
	return skip
}

func isAllowedExtension(path string) bool {
	// A helper to decide whether an extension is allowed,
	// thus allowing processing
	allowed := []string{".go", ".py", ".js", ".cpp", ".txt"}
	ext := filepath.Ext(path)
	for _, i := range allowed {
		if ext == i {
			return true
		}
	}
	return false
}

func parsePrediction(pred *structpb.Value) ([]float32, error) {
	// A helper to parse the prediction produced by the embedding model
	// Try to parse the prediction as a list, checking for embeddings.values
	structVal := pred.GetStructValue()
	if structVal == nil {
		return nil, fmt.Errorf("Warning: Prediction is not a struct\n")
	}
	embeddingsVal, ok := structVal.Fields["embeddings"]
	if !ok {
		return nil, fmt.Errorf("Warning: Embeddings field missing\n")
	}
	embeddingsStruct := embeddingsVal.GetStructValue()
	if embeddingsStruct == nil {
		return nil, fmt.Errorf("Warning: Embeddings not a struct\n")
	}
	valuesField, ok := embeddingsStruct.Fields["values"]
	if !ok {
		return nil, fmt.Errorf("Warning: Values field missing\n")
	}
	listValue := valuesField.GetListValue()
	if listValue == nil {
		return nil, fmt.Errorf("Warning: Values field is not a list\n")
	}
	embedding := make([]float32, len(listValue.Values))
	for idx, val := range listValue.Values {
		embedding[idx] = float32(val.GetNumberValue())
	}
	return embedding, nil
}

func (e *Embedder) EmbedQuery(userInput string) ([]float32, error) {
	// A function that will embed the query from the user
	instance, err := structpb.NewStruct(map[string]interface{}{
		"content": userInput,
	})
	if err != nil {
		return nil, err
	}
	queryRequest := &aiplatformpb.PredictRequest{
		Endpoint:  e.ModelEndpoint,
		Instances: []*structpb.Value{structpb.NewStructValue(instance)},
	}
	// Call the Vertex AI API, get response from model
	resp, err := e.Client.Predict(e.Ctx, queryRequest)
	if err != nil {
		return nil, fmt.Errorf("Warning: Could not create struct\n")
	}
	// Check if resp is empty
	if len(resp.Predictions) == 0 {
		return nil, fmt.Errorf("Warning: Empty query does not generate response\n")
	}
	queryEmbedding, err := parsePrediction(resp.Predictions[0])
	if err != nil {
		return nil, fmt.Errorf("Warning")
	}
	return queryEmbedding, nil
}

// ReadDirectory Will walk through the current directory to read in content
func ReadDirectory(dir string) (files []FileData, err error) {
	// Recursively walks the directory tree starting at dir
	// Filters by given extensions
	// Returns a slice of FileData structs containing file paths and content
	newErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Handle dirs
		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		// Handle files
		if !d.IsDir() && isAllowedExtension(path) {
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

	files, err := ReadDirectory(dir)
	if err != nil {
		return nil, err
	}
	var embeddings []FileEmbedding
	for _, file := range files {
		if len(file.Content) == 0 {
			fmt.Printf("Skipping empty file: %s\n", file.Path)
			continue
		}
		fmt.Println("Processing file:", file.Path)
		instance, err := structpb.NewStruct(map[string]interface{}{
			"content": file.Content,
		})
		if err != nil {
			fmt.Printf("Warning: failed to create struct for %s: %v\n", file.Path, err)
			continue
		}

		request := &aiplatformpb.PredictRequest{
			Endpoint:  e.ModelEndpoint,
			Instances: []*structpb.Value{structpb.NewStructValue(instance)},
		}

		resp, err := e.Client.Predict(e.Ctx, request)
		if err != nil {
			fmt.Printf("Warning: prediction failed for %s: %v\n", file.Path, err)
			continue
		}
		// Check if the embedding model generated a response
		if len(resp.Predictions) == 0 {
			fmt.Printf("Warning: no predictions returned for %s\n", file.Path)
			continue
		}
		embedding, err := parsePrediction(resp.Predictions[0])
		if err != nil {
			fmt.Printf("Warning: %v for %s\n", err, file.Path)
			continue
		}

		embeddings = append(embeddings, FileEmbedding{
			Path:      file.Path,
			Content:   file.Content,
			Embedding: embedding,
		})
	}
	return embeddings, nil
}
