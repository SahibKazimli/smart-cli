package re_indexer

import (
	""
	"bytes"
	"context"
	"os/exec"
	"strings"
)

/* The goal of this module is to keep the semantic search fresh
whenever a user changes the codebase. But we don't want to re-embed
every time a character is changed, so I'll try figuring out a way to
control re-embeddings.
*/

// ChangedFiles returns files changed between baseRef and HEAD.
// Example baseRef: "origin/main" or a commit hash.
// Falls back to empty slice on error.
