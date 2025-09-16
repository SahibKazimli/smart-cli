package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:     "smartcli",
		Short:   "AI-enhanced command line interface",
		Long:    "Modular AI-enhanced command line interface for coding assistance",
		Example: `smartcli review -f embedder.go -q "what does this file do?"`,
	}
	// Add the code review command
	rootCmd.AddCommand(createCodeReviewCmd())

	// And more commands will be added later
	// rootCmd.AddCommand()
	// rootCmd.AddCommand()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func startInteractiveMode() {
	fmt.Println("🚀 SmartCLI Interactive Mode")
	fmt.Println("Ask questions about your codebase!")
	fmt.Println("Commands:")
	fmt.Println("  review <file> - Ask questions about a specific file")
	fmt.Println("  explain <error> - Get explanations for errors")
	fmt.Println("  help - Show available commands")
	fmt.Println("  exit - Quit")
	fmt.Println()

	fmt.Println("For now, use: smartcli review -f <file> -q \"your question\"")
}

func createStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Interactive mode - ask questions about your codebase",
		Long:  `Start an interactive session to ask questions about your codebase`,
		Run: func(cmd *cobra.Command, args []string) {
			startInteractiveMode()
		},
	}
}

// ===== Helpers =====
func mustGCP() (projectID, location, creds string) {
	projectID = os.Getenv("GCP_PROJECT_ID")
	location = os.Getenv("GCP_LOCATION")
	creds = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if projectID == "" || location == "" || creds == "" {
		log.Fatal("Please set GCP_PROJECT_ID, GCP_LOCATION, and GOOGLE_APPLICATION_CREDENTIALS")
	}
	return
}
