package main

import (
	"bufio"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {

	_ = godotenv.Load()
	if exe, err := os.Executable(); err == nil {
		_ = godotenv.Load(filepath.Join(filepath.Dir(exe), ".env"))
	}

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
	fmt.Println("Type 'help' for available commands or 'exit' to quit")
	fmt.Println()

	// Reading user input
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("smartcli> ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle exit commands
		if input == "exit" || input == "quit" || input == "q" {
			fmt.Println("Goodbye!")
			break
		}

		// Handle help command
		if input == "help" || input == "h" {
			showHelp()
			continue
		}

		// Parse and execute commands
		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}

		executeInteractiveCommand(args)
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
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

func showHelp() {
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("   init                    - Set up SmartCLI (check environment variables)")
	fmt.Println("   index                   - Index your codebase for AI search")
	fmt.Println("   review -f <file> -q <query> - Ask questions about specific code files")
	fmt.Println("   explain <error_message> - Get AI explanations for error messages")
	fmt.Println("   help                    - Show this help message")
	fmt.Println("   exit                    - Exit interactive mode")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("   review -f main.go -q \"what does this do?\"")
	fmt.Println("   explain \"undefined variable error\"")
	fmt.Println("   index")
	fmt.Println()
}

func executeInteractiveCommand(args []string) {
	command := args[0]

	switch command {
	case "init":
		// Execute init command logic
		initCmd := createInitCmd()
		initCmd.Run(initCmd, args[1:])

	case "index":
		// Execute index command logic
		indexCmd := createIndexCmd()
		indexCmd.Run(indexCmd, args[1:])

	case "review":
		// Execute review command logic
		reviewCmd := createCodeReviewCmd()
		// Set the args for cobra to parse
		reviewCmd.SetArgs(args[1:])
		if err := reviewCmd.Execute(); err != nil {
			fmt.Printf("Error executing review: %v\n", err)
		}

	case "explain":
		// Execute explain command logic
		explainCmd := createErrorCommand()
		explainCmd.SetArgs(args[1:])
		if err := explainCmd.Execute(); err != nil {
			fmt.Printf("Error executing explain: %v\n", err)
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Type 'help' for available commands")
	}
}
