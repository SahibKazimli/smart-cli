package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"smart-cli/go-backend/embedder"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	var (
		envFile = flag.String("env", "", "path to .env file (optional). If empty, will try .env in the current working directory")
		dir     = flag.String("dir", ".", "directory to embed (project root auto-detected from here)")
		index   = flag.String("index", "", "index name (defaults to <project-root>_index)")
		// If you pass these flags, they override env; otherwise .env/env vars are used
		creds    = flag.String("creds", "", "path to service account JSON (falls back to $GOOGLE_APPLICATION_CREDENTIALS)")
		project  = flag.String("project", "", "GCP project ID (falls back to $GCP_PROJECT_ID)")
		location = flag.String("location", "", "GCP location/region, e.g. us-central1 (falls back to $GCP_LOCATION)")
		redisAdr = flag.String("redis", "", "Redis address host:port (falls back to $REDIS_ADDR or $REDIS_HOST:$REDIS_PORT or 127.0.0.1:6379)")
		redisPwd = flag.String("redis-pass", "", "Redis password (falls back to $REDIS_PASSWORD)")
	)
	flag.Parse()

	// Load env from .env (no export needed)
	if *envFile != "" {
		if err := godotenv.Load(*envFile); err != nil {
			fmt.Printf("WARN: could not load env file %s: %v\n", *envFile, err)
		}
	} else {
		if err := godotenv.Load(); err != nil {
			_ = godotenv.Load("../.env")
		}
	}

	// Resolve configuration from flags or env
	if *creds == "" {
		*creds = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}
	if *project == "" {
		*project = os.Getenv("GCP_PROJECT_ID")
	}
	if *location == "" {
		*location = os.Getenv("GCP_LOCATION")
	}
	// Build Redis address from env if flag not provided
	if *redisAdr == "" {
		host := getenv("REDIS_HOST", "")
		port := getenv("REDIS_PORT", "")
		addr := getenv("REDIS_ADDR", "")
		switch {
		case addr != "":
			*redisAdr = addr
		case host != "" && port != "":
			*redisAdr = host + ":" + port
		default:
			*redisAdr = "127.0.0.1:6379"
		}
	}
	if *redisPwd == "" {
		*redisPwd = os.Getenv("REDIS_PASSWORD")
	}

	// Validate inputs early
	fail := false
	if *creds == "" {
		fmt.Println("ERROR: credentials path not provided. Use -creds or set GOOGLE_APPLICATION_CREDENTIALS (via .env or shell)")
		fail = true
	} else if _, err := os.Stat(*creds); err != nil {
		fmt.Printf("ERROR: credentials file not found: %s (%v)\n", *creds, err)
		fail = true
	}
	if *project == "" {
		fmt.Println("ERROR: GCP project not provided. Use -project or set GCP_PROJECT_ID (via .env or shell)")
		fail = true
	}
	if *location == "" {
		fmt.Println("ERROR: GCP location not provided. Use -location or set GCP_LOCATION (via .env or shell)")
		fail = true
	}
	if fail {
		os.Exit(1)
	}

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:        *redisAdr,
		Password:    *redisPwd, // empty is fine
		ReadTimeout: 30 * time.Second,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		fmt.Printf("ERROR: redis ping failed (%s): %v\n", *redisAdr, err)
		os.Exit(1)
	}

	// EmbedderClient reads GCP_PROJECT_ID and GCP_LOCATION from env.
	// If flags were used, ensure they are present in env for EmbedderClient.
	_ = os.Setenv("GCP_PROJECT_ID", *project)
	_ = os.Setenv("GCP_LOCATION", *location)

	ctx := context.Background()
	emb, err := embedder.EmbedderClient(ctx, *creds, rdb, "text-embedding-005")
	if err != nil {
		fmt.Printf("ERROR: creating embedder: %v\n", err)
		os.Exit(1)
	}

	absDir := *dir
	if !filepath.IsAbs(absDir) {
		if absDir, err = filepath.Abs(*dir); err != nil {
			fmt.Printf("ERROR: resolving dir: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Printf("Embedding (concurrent) from base: %s\n", absDir)

	idx, n, err := emb.EmbedAndIndex(*dir, *index, nil)
	if err != nil {
		fmt.Printf("ERROR: EmbedAndIndex failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Done. Index: %s, documents stored: %d\n", idx, n)
}
