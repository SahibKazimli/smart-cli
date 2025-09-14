package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "smartcli",
		Short: "AI-enhanced command line interface",
		Long:  "Modular AI-enhanced command line interface for coding assistance",
	}
	// Add the code review command
	rootCmd.AddCommand(createCodeReviewCmd())

	// And more commands will be added later
	rootCmd.AddCommand()
	rootCmd.AddCommand()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
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

func createCodeReviewCmd() *cobra.Command {
	var filePath string
	var detailLevel string

	codeReviewCmd := &cobra.Command{
		Use:   "review",
		Short: "Review code for improvements",
		Long:  `Analyze code for potential bugs, style improvements, and optimization opportunities.`,
		Run: func(cmd *cobra.Command, args []string) {
			// If no file path is provided but there are arguments, use the first argument
			if filePath == "" && len(args) > 0 {
				filePath = args[0]
			}

			if filePath == "" {
				fmt.Println("Error: Please provide a file to review")
				return
			}

			// Call the function that will handle the code review

		},
	}

	// Add flags specific to code review
	codeReviewCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to file for code review")
	codeReviewCmd.Flags().StringVarP(&detailLevel, "detail", "d", "medium", "Level of review detail (low, medium, high)")

	return codeReviewCmd
}
