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
	// Add commands
	rootCmd.AddCommand(createCodeReviewCmd())
	rootCmd.AddCommand(createErrorCommand())
	rootCmd.AddCommand(createIndexCmd())
	rootCmd.AddCommand(createInitCmd())
	rootCmd.AddCommand(createStartCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func startInteractiveMode() {
	fmt.Println("SmartCLI - AI-Powered Code Analysis")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("   init    - Set up SmartCLI (check environment variables)")
	fmt.Println("   index   - Index your codebase for AI search")
	fmt.Println("   review  - Ask questions about specific code files")
	fmt.Println("   explain - Get AI explanations for error messages")
	fmt.Println("   start   - Show this help (you are here!)")
	fmt.Println()
	fmt.Println("Quick Start:")
	fmt.Println("1. smartcli init     # Check your setup")
	fmt.Println("2. smartcli index    # Index your codebase")
	fmt.Println("3. smartcli review -f main.go -q \"what does this do?\"")
	fmt.Println()
	fmt.Println("For detailed help: smartcli <command> --help")
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
