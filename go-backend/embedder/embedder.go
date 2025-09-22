package embedder

import (
	"context"
	"encoding/binary"
	"fmt"
	"google.golang.org/api/option"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"smart-cli/go-backend/chunk_retriever"
	"sync"
	"time"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/types/known/structpb"
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

/* // EmbedderClient creates a new Embedder instance with the Vertex AI Prediction client, Redis client,
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
	model := "text-embedding-005"

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
}*/

func EmbedderClient(ctx context.Context, credsFile string, rdb *redis.Client, model string) (*Embedder, error) {
	// Validate env for resource building
	projectID := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION")
	if projectID == "" || location == "" {
		return nil, fmt.Errorf("missing GCP_PROJECT_ID or GCP_LOCATION")
	}
	// Respect CLI-provided model; default if empty
	if model == "" {
		model = "text-embedding-005"
	}

	// IMPORTANT: use the regional host for Prediction API
	client, err := aiplatform.NewPredictionClient(
		ctx,
		option.WithCredentialsFile(credsFile),
		option.WithEndpoint(fmt.Sprintf("%s-aiplatform.googleapis.com:443", location)),
	)
	if err != nil {
		return nil, err
	}

	// Build the publisher model resource (not an endpoint host)
	modelResource := fmt.Sprintf(
		"projects/%s/locations/%s/publishers/google/models/%s",
		projectID, location, model,
	)

	return &Embedder{
		Client:        client,
		RDB:           rdb,
		Ctx:           ctx,
		ModelEndpoint: modelResource, // consider renaming to ModelResource to avoid confusion
	}, nil
}

// ===== Repo scanning helpers =====

func ShouldSkipDir(name string) bool {
	// A helper to decide whether a dir should be skipped for processing
	skipDirs := map[string]struct{}{
		"venv": {}, "__pycache__": {}, "node_modules": {}, ".git": {},
	}
	_, skip := skipDirs[name]
	return skip
}

func defaultExtensions() []string {
	return []string{".go", ".py", ".js", ".ts", ".tsx", ".jsx", ".json", ".md", ".txt", ".yaml", ".yml"}
}

func isAllowedExtension(path string, allowed []string) bool {
	use := allowed
	if len(use) == 0 {
		use = defaultExtensions()
	}
	ext := filepath.Ext(path)
	for _, a := range use {
		if ext == a {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// findProjectRoot walks up from start until it finds a .git or go.mod directory/file.
// If neither is found, returns the original start directory.
func findProjectRoot(start string) string {
	curr := start
	for {
		if fileExists(filepath.Join(curr, ".git")) || fileExists(filepath.Join(curr, "go.mod")) {
			return curr
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			return start
		}
		curr = parent
	}
}

// detectBase decides the base directory to embed.
// If dir is "", ".", or "./", it resolves to the project root based on the current working directory.
// Otherwise, it returns the absolute path of dir.
func detectBase(dir string) (string, error) {
	var start string
	if dir == "" || dir == "." || dir == "./" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		start = cwd
	} else {
		if !filepath.IsAbs(dir) {
			cwd, err := os.Getwd()
			if err != nil {
				return "", err
			}
			dir = filepath.Clean(filepath.Join(cwd, dir))
		}
		start = dir
	}
	root := findProjectRoot(start)
	return root, nil
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
		// Handle dirs
		if d.IsDir() && ShouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		// Handle files
		if !d.IsDir() && isAllowedExtension(path, extensions) {
			// process file
			ctn, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			files = append(files, FileData{
				Path:    path,
				Content: string(ctn),
			})
		}
		return nil
	})

	if newErr != nil {
		return nil, newErr
	}
	return
}

// ===== Vertex AI helpers =====

func parsePrediction(pred *structpb.Value) ([]float32, error) {
	// A helper to parse the prediction produced by the embedding model
	// Try to parse the prediction as a list, checking for embeddings.values
	structVal := pred.GetStructValue()
	if structVal == nil {
		return nil, fmt.Errorf("warning prediction is not a struct")
	}
	embeddingsVal, ok := structVal.Fields["embeddings"]
	if !ok {
		return nil, fmt.Errorf("embeddings field missing")
	}
	embeddingsStruct := embeddingsVal.GetStructValue()
	if embeddingsStruct == nil {
		return nil, fmt.Errorf("warning: embeddings not a struct")
	}
	valuesField, ok := embeddingsStruct.Fields["values"]
	if !ok {
		return nil, fmt.Errorf("values field missing")
	}
	listValue := valuesField.GetListValue()
	if listValue == nil {
		return nil, fmt.Errorf("values field is not a list")
	}
	embedding := make([]float32, len(listValue.Values))
	for idx, val := range listValue.Values {
		embedding[idx] = float32(val.GetNumberValue())
	}
	return embedding, nil
}

func (e *Embedder) EmbedContent(content string) ([]float32, error) {
	instance, err := structpb.NewStruct(map[string]interface{}{
		"content": content,
	})
	// Per-call timeout
	ctx, cancel := context.WithTimeout(e.Ctx, 20*time.Second)
	defer cancel()

	if err != nil {
		return nil, err
	}
	request := &aiplatformpb.PredictRequest{
		Endpoint:  e.ModelEndpoint,
		Instances: []*structpb.Value{structpb.NewStructValue(instance)},
	}
	resp, err := e.Client.Predict(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("prediction failed: %w", err)
	}
	if len(resp.Predictions) == 0 {
		return nil, fmt.Errorf("no predictions returned")
	}
	return parsePrediction(resp.Predictions[0])
}

func (e *Embedder) EmbedQuery(userInput string) ([]float32, error) {
	// A function that will embed the query from the user
	instance, err := structpb.NewStruct(map[string]interface{}{
		"content": userInput,
	})
	// Per-call timeout
	ctx, cancel := context.WithTimeout(e.Ctx, 20*time.Second)
	defer cancel()

	if err != nil {
		return nil, err
	}
	queryRequest := &aiplatformpb.PredictRequest{
		Endpoint:  e.ModelEndpoint,
		Instances: []*structpb.Value{structpb.NewStructValue(instance)},
	}
	// Call the Vertex AI API, get response from model
	resp, err := e.Client.Predict(ctx, queryRequest)
	if err != nil {
		return nil, fmt.Errorf("warning could not create struct")
	}

	// Check if resp is empty
	if len(resp.Predictions) == 0 {
		return nil, fmt.Errorf("warning: empty query does not generate response")
	}
	queryEmbedding, err := parsePrediction(resp.Predictions[0])

	if err != nil {
		return nil, fmt.Errorf("warning")
	}
	return queryEmbedding, nil
}

// ===== Redis Helpers =====

// float32ToLEBytes converts a float32 slice to little-endian byte slice for RediSearch VECTOR.
func float32ToLEBytes(vec []float32) []byte {
	buf := make([]byte, 4*len(vec))
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

// storeFileEmbeddingInRedis stores a diagnostic copy under key "embedding:<file.Path>".
// embedding should be a little-endian []byte vector.
// Unneeded
func (e *Embedder) storeFileEmbeddingInRedis(file FileData, embedding []byte) error {
	if e.RDB == nil {
		return nil
	}

	key := fmt.Sprintf("embedding:%s", file.Path)
	return e.RDB.HSet(e.Ctx, key, map[string]interface{}{
		"path":      file.Path,
		"content":   file.Content,
		"embedding": embedding,
	}).Err()
}

// storeEmbeddingsInRedis writes indexable entries under "<prefix><i>".
// Each hash contains fields: text, file, chunk, embedding (LE bytes).
func (e *Embedder) storeEmbeddingsInRedis(prefix string, embeddings []FileEmbedding) (int, error) {
	if e.RDB == nil {
		return 0, fmt.Errorf("redis client is nil")
	}
	for i, ebd := range embeddings {
		key := fmt.Sprintf("%s%d", prefix, i)
		vec := float32ToLEBytes(ebd.Embedding)

		if err := e.RDB.HSet(e.Ctx, key, map[string]interface{}{
			"text":      ebd.Content,
			"embedding": vec,
			"file":      ebd.Path,
			"chunk":     i,
		}).Err(); err != nil {
			return i, fmt.Errorf("failed to store embedding for %s: %w", ebd.Path, err)
		}
	}
	return len(embeddings), nil
}

// ===== Public API =====

func (e *Embedder) EmbedText(content string) ([]float32, error) {
	return e.EmbedContent(content)
}

/*
EmbedDirectory calls ReadDirectory and reads from all files in the working directory.
For each file, calls Vertex AI to generate embeddings.
Parses the predictions response to get the vector.
*/
func (e *Embedder) EmbedDirectory(dir string, extensions []string) ([]FileEmbedding, error) {

	var wg sync.WaitGroup
	fileCh := make(chan FileData)
	embCh := make(chan FileEmbedding)
	errCh := make(chan error)

	base, err := detectBase(dir)
	if err != nil {
		return nil, err
	}
	// Calling read directory worker
	wg.Add(1)
	go ReadDirWorker(base, extensions, fileCh, &wg, errCh)

	// Spawning embedding workers for each file
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileCh {
				EmbedFileWorker(e, file, embCh, nil, errCh)
			}
		}()
	}
	// Close the workers after waiting for them to finish
	go func() {
		wg.Wait()
		close(embCh)
		close(errCh)
	}()
	// Collect embeddings
	var embeddings []FileEmbedding
	for emb := range embCh {
		embeddings = append(embeddings, emb)
	}
	// Error logs
	for err := range errCh {
		fmt.Println("Warning:", err)
	}
	return embeddings, nil
}

// EmbedAndIndex embeds and writes the results to a RediSearch index.
// - If dir is "", ".", or "./", it embeds from the project root.
func (e *Embedder) EmbedAndIndex(dir, indexName string, extensions []string) (string, int, error) {
	base, err := detectBase(dir)
	if err != nil {
		return "", 0, err
	}

	embeddings, err := e.EmbedDirectory(base, extensions)
	if err != nil {
		return "", 0, err
	}
	if len(embeddings) == 0 {
		return "", 0, fmt.Errorf("no embeddings generated")
	}

	if indexName == "" {
		indexName = filepath.Base(base) + "_index"
	}
	prefix := indexName + ":"

	// Ensure FT index exists with correct dimension
	dim := len(embeddings[0].Embedding)
	if err := chunk_retriever.EnsureIndex(e.RDB, indexName, prefix, dim); err != nil {
		return indexName, 0, err
	}

	// Store to index prefix
	n, err := e.storeEmbeddingsInRedis(prefix, embeddings)
	return indexName, n, err
}

// ===== Goroutine workers =====

func ReadDirWorker(dir string, extensions []string, ch chan<- FileData, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()
	defer close(ch)
	files, err := ReadDirectory(dir, extensions)
	if err != nil {
		errCh <- fmt.Errorf("failed reading %s: %w", dir, err)
		return
	}
	for _, f := range files {
		ch <- f
	}
}

// EmbedFileWorker is a worker for embedding single files, and will be called in EmbedDirectory
// which will act as a manager
func EmbedFileWorker(e *Embedder, file FileData, ch chan<- FileEmbedding, _ *sync.WaitGroup, errCh chan<- error) {
	// Skip empty files
	if len(file.Content) == 0 {
		return
	}
	// embed content
	emb, err := e.EmbedContent(file.Content)
	if err != nil {
		errCh <- fmt.Errorf("warning: embedding failed for file: %s: %w", file.Path, err)
		return
	}
	ch <- FileEmbedding{
		Path:      file.Path,
		Content:   file.Content,
		Embedding: emb,
	}
}
