package main

import (
	"context"
	"fmt"
	"smart-cli/go-backend/chunk_retriever"
	"smart-cli/go-backend/generator"
	"strings"

	"github.com/spf13/cobra"
)

func createErrorCommand() *cobra.Command {
	var errorText string

	errExplainCmd := &cobra.Command{
		Use:   "explain",
		Short: "Explain errors in your code",
		Long:  `Explain compilation errors, runtime errors, or general code issues using AI`,
		Example: `  smartcli explain -e "undefined: fmt.Println"
  smartcli explain -f main.go --auto     # Auto-detect issues in file
  smartcli explain --build               # Explain current build errors
  smartcli explain -e "cannot use string as int"`,
		Run: func(cmd *cobra.Command, args []string) {
			// If error text is provided as argument
			if errorText == "" && len(args) > 0 {
				errorText = strings.Join(args, " ")
			}
			if errorText == "" {
				fmt.Println("Error: Please provide an error message to explain")
				fmt.Println("Examples:")
				fmt.Println("smartcli explain -e ")
			}
			explainError(errorText)
		},
	}
	errExplainCmd.Flags().StringVarP(&errorText, "error", "e", "", "Error message to explain")
	return errExplainCmd
}

// ===== Helpers =====

func explainError(errorText string) {
	fmt.Printf("Explaining error: %s\n", errorText)

	ctx := context.Background()

	// Generate explanation using AI
	answer := generateAIExplanation(ctx, errorText, nil)
	if answer != "" {
		fmt.Println("\n===== AI Explanation =====")
		fmt.Println(answer)
	}
}

func generateAIExplanation(ctx context.Context, errorText string, retrievedChunks []chunk_retriever.Chunk) string {
	prompt := fmt.Sprintf(`Please explain this programming error and provide a solution:

Error: %s

Please explain:
1. What this error means in simple terms
2. What typically causes this error
3. How to fix it with clear steps and code examples
4. How to prevent it in the future

Keep the explanation clear, practical, and focused on Go programming.`, errorText)

	gen, err := generator.NewAgent(ctx, "gemini-2.5-flash")
	if err != nil {
		fmt.Printf("Error creating AI agent: %v\n", err)
		return ""
	}

	answer, err := gen.Answer(ctx, prompt, retrievedChunks)
	if err != nil {
		fmt.Printf("Error getting AI explanation: %v\n", err)
		return ""
	}

	return answer
}
