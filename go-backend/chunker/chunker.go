package chunker

import (
	"os"
	"unicode/utf8"
)

type Chunk struct {
	Index int
	Text  string
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

// SplitFile reads a file from disk and splits its content into chunks of text
// The return type is a slice of Chunk structs
func SplitFile(filePath string, chunkSize, overlap int) ([]Chunk, error) {
	// read entire file into memory
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	// Safety measures for UTF-8 encoding
	fileContent := string(fileBytes)
	if !utf8.ValidString(fileContent) {
		// skip binary or invalid UTF-8 files
		return nil, nil
	}
	// Break the string down into chunks
	chunkStrings := SplitText(fileContent, chunkSize, overlap)

	chunks := make([]Chunk, len(chunkStrings))

	// Creating slice of Chunk structs containing different metadata
	for i, text := range chunkStrings {
		chunks[i] = Chunk{
			num:  i,
			Text: text,
		}
	}

	return chunks, nil
}
