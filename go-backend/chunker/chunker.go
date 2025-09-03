package chunker

import (
	"os"
	"os/exec"
)

type Chunks struct {
	num  int
	Text string
}

// SplitText does a simple character-count chunk with overlap (UTF-8 safe).
func SplitText(s string, size, overlap int) []string {
	// default chunk size if unspecified
	if size <= 0 {
		size = 800
	}

	if overlap < 0 {
		overlap = 0
	}
	// create slice of runes for UTF-8 handling
	var chunks []string
	runes := []rune(s)

	// Loop over the runes and create chunks
	for start := 0; start < len(runes); {
		end := start + size
		if end > len(runes) { // in case the last chunk is too big, shorten it
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
		if end == len(runes) {
			break
		}
		// step forward but keep overlap
		start = end - overlap
		if start < 0 {
			start = 0
		}
	}
	return chunks
}
