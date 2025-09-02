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
	cmd := exec.Command("git", "diff", "--name-only", baseReference+"..HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run()
}
