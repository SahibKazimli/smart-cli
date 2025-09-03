package re_indexer

import (
	"bytes"
	"os/exec"
	"strings"
)

// ChangedFiles returns files changed between baseRef and HEAD (paths relative to repo root).
// Example baseRef: "origin/main" or a commit hash.
func ChangedFiles(baseReference string) []string {
	if strings.TrimSpace(baseReference) == "" {
		baseReference = "origin/main"
	}
	// We will run this command
	// We will check if re-indexing is needed by running git diff commands
	cmd := exec.Command("git", "diff", "--name-only", baseReference+"..HEAD")
	var commandOut bytes.Buffer
	cmd.Stdout = &commandOut
	_ = cmd.Run()

	changedLines := strings.Split(commandOut.String(), "\n")
	var changedFiles []string
	for _, line := range changedLines {
		trimLine := strings.TrimSpace(line)
		if trimLine != "" {
			changedFiles = append(changedFiles, trimLine)
		}
	}
	return changedFiles
}
