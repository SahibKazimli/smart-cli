package generator

import (
	"context"
	"fmt"
	"smart-cli/go-backend/chunk_retriever"
	"sort"
	"strings"
	"time"
)

type Generator struct {
	// llm client (e.g., Vertex AI) goes here
	projectID string
	location  string
	modelName string
}

func NewAgent(ctx context.Context, projectID, location, modelName string) (*Generator, error) {
	// init LLM client
	return &Generator{
		projectID: projectID,
		location:  location,
		modelName: modelName,
	}, nil
}

func (g *Generator) Close() error {
	// close client if needed
	return nil
}

func (g *Generator) Answer(ctx context.Context, query string, chunks []chunk_retriever.Chunk) (string, error) {
	// 1) select/rerank
	// 2) budget
	// 3) build context (ctxText, citeMap)
	// 4) assemble prompt
	// 5) call LLM (non-streaming)
	// 6) post-process + map citations
	// 7) return
	return "...answer...", nil
}

func (g *Generator) StreamAnswer(ctx context.Context, query string, chunks []chunk_retriever.Chunk, onToken func(string)) (string, error) {
	// same as Answer but streaming
	// call onToken as tokens arrive, build final buffer
	return "...final...", nil
}

// Helper: optional rerank hook (no-op initial)
func maybeRerank(ctx context.Context, query string, chunks []chunk_retriever.Chunk, use bool) []chunk_retriever.Chunk {
	// if use { call external reranker } else { return chunks }
	return chunks
}

// Helper: budget fit
func fitToBudget(chunks []chunk_retriever.Chunk, max int) []chunk_retriever.Chunk {
	// estimate per-chunk size and include until limit
	return chunks
}

// Helper: build context and citation map
func buildContext(chunks []chunk_retriever.Chunk) string {
	const charBudget = 10000
	if len(chunks) == 0 {
		return ""
	}
	// Sort by ascending distance (lower score = better)
	sort.SliceStable(chunks, func(i, j int) bool {
		return chunks[i].Score < chunks[j].Score
	})
	// Start building context
	var builder strings.Builder
	for _, ch := range chunks {
		txt := strings.TrimSpace(ch.Text)
		if txt == "" {
			continue
		}
		// Add a blank-line separator between chunks (no headers)
		if builder.Len() > 0 {
			if builder.Len()+2 > charBudget {
				break
			}
			builder.WriteString("\n\n")
		}

		remaining := charBudget - builder.Len()
		if remaining <= 0 {
			break
		}
		// If txt is longer than remaining, slice to remaining and append.
		if len(txt) > remaining {
			txt = txt[:remaining]
		}
		builder.WriteString(txt)
		if builder.Len() > charBudget {
			break
		}
	}
	return builder.String()
}

// Helper: prompt assembly
func assemblePrompt(system, query, ctxText string, requireCites bool) string {
	// combine system + user + context + rules
	return ""
}
