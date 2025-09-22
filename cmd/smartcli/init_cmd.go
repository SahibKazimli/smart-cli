package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"smart-cli/go-backend/chunk_retriever"
)

func createInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Set up SmartCLI for your project",
		Long:  `Guide you through setting up SmartCLI with the required environment variables and initial indexing`,
		Run: func(cmd *cobra.Command, args []string) {
			setupSmartCLI()
		},
	}
}

// ===== Helpers =====

func checkCreds() {
	// Check environment variables
	fmt.Println("Checking environment variables...")

	vars := map[string]string{
		"GCP_PROJECT_ID":                 os.Getenv("GCP_PROJECT_ID"),
		"GCP_LOCATION":                   os.Getenv("GCP_LOCATION"),
		"GOOGLE_APPLICATION_CREDENTIALS": os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
	}

	allSet := true
	for key, val := range vars {
		if val == "" {
			fmt.Printf("key not set: %s", key)
			allSet = false
		} else {
			fmt.Printf("%s: %s\n", key, val)
		}
	}
	if !allSet {
		fmt.Println("\nMissing required environment variables!")
		fmt.Println("Please set the following in your shell or .env file:\n")

		if vars["GCP_PROJECT_ID"] == "" {
			fmt.Println(`export GCP_PROJECT_ID="your-gcp-project-id"`)
		}
		if vars["GCP_LOCATION"] == "" {
			fmt.Println(`export GCP_LOCATION="us-central1"`)
		}
		if vars["GOOGLE_APPLICATION_CREDENTIALS"] == "" {
			fmt.Println(`export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"`)
		}

		fmt.Println("\nAfter setting these variables, run: smartcli init")
	}
}

func setupSmartCLI() {
	checkCreds()

	// Check Redis connection
	fmt.Println("\n🔧 Checking Redis connection...")
	rdb := chunk_retriever.Connect()
	if rdb == nil {
		fmt.Println("Cannot connect to Redis")
		fmt.Println("Make sure Redis is running on localhost:6379")
		fmt.Println("Or set REDIS_ADDR environment variable")
		return
	}
	defer func() { _ = rdb.Close() }()
	fmt.Println("Redis connection successful")

	// Check if index exists
	fmt.Println("\nChecking for existing index...")
	indexName, err := chunk_retriever.GetIndexName(rdb)
	if err != nil || indexName == "" {
		fmt.Println("No index found - you'll need to create one")
		fmt.Println("\n Next steps:")
		fmt.Println("1. Run: smartcli index")
		fmt.Println("2. Then try: smartcli review -f <file> -q \"what does this do?\"")
	} else {
		fmt.Printf("Found existing index: %s\n", indexName)
		fmt.Println("\nYou're all set! Try these commands:")
		fmt.Println("  smartcli review -f <file> -q \"explain this function\"")
		fmt.Println("  smartcli explain \"your error message\"")
	}

	fmt.Println("\nFor more help: smartcli --help")
}
