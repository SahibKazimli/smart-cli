package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/embedder"
	"smart-cli/go-backend/generator"
	"smart-cli/go-backend/re_indexer"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

}

func usage() {
	fmt.Println("smartcli commands:")
	fmt.Println("  index   Re-index the project into Redis")
	fmt.Println("  ask     Ask a question using the indexed project context")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  smartcli index [--dir .] [--chunk-size 800] [--overlap 50] [--index-name NAME] [--embedding-model text-embedding-005]")
	fmt.Println("  smartcli ask   --q \"your question\" [--topk 5] [--generation-model gemini-1.5-pro] [--embedding-model text-embedding-005] [--show-context]")
}

func mustGCP() (projectID, location, creds string) {
	projectID = os.Getenv("GCP_PROJECT_ID")
	location = os.Getenv("GCP_LOCATION")
	creds = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if projectID == "" || location == "" || creds == "" {
		log.Fatal("Please set GCP_PROJECT_ID, GCP_LOCATION, and GOOGLE_APPLICATION_CREDENTIALS")
	}
	return
}
