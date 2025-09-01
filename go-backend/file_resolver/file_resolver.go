package file_resolver

import (
	"io/fs"
	"path/filepath"
	"smart-cli/go-backend/embedder"
	"strings"
)

// Resolver is a struct to manage file paths and allow lookup by filename
// Maps lowercase filenames to all (list of full paths)
type Resolver struct {
	Root   string
	byBase map[string][]string
	all    []string
}

// NewRoot constructs a new Resolver and scans all files under the root
func NewRoot(root string) (*Resolver, error) {
	r := &Resolver{
		Root:   root,
		byBase: make(map[string][]string),
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable
		}
		if d.IsDir() {
			name := d.Name()
			// if embedder flags the file, skip it
			if embedder.ShouldSkipDir(name) {
				return filepath.SkipDir
			}
			return nil // otherwise, continue walking
		}
		// Lowercasing the name for lookup
		base := strings.ToLower(filepath.Base(path))
		r.byBase[base] = append(r.byBase[base], path)
		r.all = append(r.all, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

// Helper to skip dirs like caches and dependencies etc
func shouldSkipDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", "node_modules", "vendor", "dist", "build", "target", ".venv", "__pycache__":
		return true
	}
	return false
}

// Helper to see if it's a code file
func isCodeFile(name string) bool {
	ln := strings.ToLower(name)
	switch {
	case strings.HasSuffix(ln, ".go"),
		strings.HasSuffix(ln, ".py"),
		strings.HasSuffix(ln, ".js"),
		strings.HasSuffix(ln, ".ts"),
		strings.HasSuffix(ln, ".tsx"),
		strings.HasSuffix(ln, ".jsx"),
		strings.HasSuffix(ln, ".java"),
		strings.HasSuffix(ln, ".rb"),
		strings.HasSuffix(ln, ".rs"),
		strings.HasSuffix(ln, ".cpp"),
		strings.HasSuffix(ln, ".c"),
		strings.HasSuffix(ln, ".cs"),
		strings.HasSuffix(ln, ".md"):
		return true
	default:
		return false
	}
}
