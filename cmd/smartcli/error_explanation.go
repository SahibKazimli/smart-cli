package main

import (
	"github.com/spf13/cobra"
	"strings"
)

func createErrorCommand() *cobra.Command {
	var errorText string
	var filePath string
	var autoDetect bool
	var buildErrors bool

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

		}
	}
}

// ===== Helpers =====

// For very common errors, get instant feedback
func getErrorExplanation(message string) string {
	explanations := map[string]string{
		"undefined":     "The identifier is not declared or not accessible in this scope",
		"cannot use":    "Type mismatch - the value type doesn't match what's expected",
		"too many":      "You're providing more arguments than the function accepts",
		"not enough":    "You're providing fewer arguments than the function requires",
		"imported and not used": "You've imported a package but haven't used it in your code",
		"declared and not used": "You've declared a variable but haven't used it",
	}