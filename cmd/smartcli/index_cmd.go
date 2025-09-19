package main

import "github.com/spf13/cobra"

func createIndexCmd() *cobra.Command {
	var dir string
	var indexName string
	var force bool

	indexCmd := &cobra.Command{
		Use:   "index",
		Short: "Index your codebase for AI-powered search",
		Long:  `Scan and index your codebase to enable AI-powered code review and error explanation`,
		Example: `  smartcli index                    # Index current directory
  smartcli index --dir ./my-project  # Index specific directory
  smartcli index --force             # Re-index even if index exists`,
		Run: func(cmd *cobra.Command, args []string) {
			if dir == "" {
				dir = "."
			}

			// Future helper: indexCodebase(dir, indexName, force)
		},
	}

	indexCmd.Flags().StringVarP(&dir, "dir", "d", "", "Directory to index (defaults to current directory)")
	indexCmd.Flags().StringVarP(&indexName, "name", "n", "", "Index name (auto-generated if not provided)")
	indexCmd.Flags().BoolVarP(&force, "force", "f", false, "Force re-indexing even if index already exists")

	return indexCmd
}
