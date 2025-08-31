package file_resolver

import (
	"io/fs"
	"os"
	"path/filepath"
	"smart-cli/go-backend/embedder"
	"sort"
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
