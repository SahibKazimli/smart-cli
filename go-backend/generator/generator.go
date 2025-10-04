package generator

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"smart-cli/go-backend/chunk_retriever"

	"google.golang.org/genai"
)

type Generator struct {
	modelName string
	client    *genai.Client
}

func NewAgent(ctx context.Context, modelName string) (*Generator, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{})

	if err != nil {
		return nil, err
	}

	return &Generator{
		modelName: modelName,
		client:    client,
	}, nil
}

func (g *Generator) Answer(ctx context.Context, query string, chunks []chunk_retriever.Chunk) (string, error) {
	// Build retrieved context and assemble the prompt
	ctxText := buildContext(chunks)
	prompt := assemblePrompt("", query, ctxText)
	// Require a configured model
	if g == nil || g.client == nil {
		// Fallback: return the prompt preview if model not initialized
		return prompt, nil
	}

	temp := float32(0.2)
	topP := float32(0.9)
	topK := float32(15)
	maxTokens := int32(700)

	// Per-call timeout
	callCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	resp, err := g.client.Models.GenerateContent(callCtx, g.modelName, genai.Text(prompt), &genai.GenerateContentConfig{
		Temperature:     &temp,
		TopP:            &topP,
		TopK:            &topK,
		MaxOutputTokens: maxTokens,
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	// Check if response is nil
	if resp == nil {
		return "", fmt.Errorf("received nil response from API")
	}

	// Extract text from first candidate
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	if resp.Candidates[0] == nil || resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("first candidate or its content is nil")
	}

	if len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no parts in first candidate's content")
	}

	// Use resp.Text() to extract plain text
	text := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if part != nil && part.Text != "" {
			text += part.Text
		}
	}
	if text == "" {
		return "", fmt.Errorf("no text generated")
	}
	return strings.TrimSpace(resp.Text()), nil
}

// Helper: build context (no headers; blank-line separators)
func buildContext(chunks []chunk_retriever.Chunk) string {
	const charBudget = 40000
	if len(chunks) == 0 {

		return ""
	}

	// Sort by ascending distance (lower score = better)
	sort.SliceStable(chunks, func(i, j int) bool {
		return chunks[i].Score < chunks[j].Score
	})

	// Start building context
	var builder strings.Builder
	chunksAdded := 0

	for _, ch := range chunks {
		txt := strings.TrimSpace(ch.Text)
		if txt == "" {
			continue
		}
		// Calculate space needed (text + separator)
		separator := ""
		if builder.Len() > 0 {
			separator = "\n\n"
		}
		spaceNeeded := len(separator) + len(txt)

		// Check if adding this complete chunk would exceed budget
		if builder.Len()+spaceNeeded > charBudget {
			// Stop here, don't truncate mid-chunk
			break
		}

		// Add separator if not first chunk
		if separator != "" {
			builder.WriteString(separator)
		}

		// Add the complete chunk
		builder.WriteString(txt)
		chunksAdded++
	}
	return builder.String()
}

// Helper: prompt assembly
func assemblePrompt(system, query, ctxText string) string {
	var builder strings.Builder
	// System prompt included up front for non-system-aware callers
	if system != "" {
		builder.WriteString(system)
		builder.WriteString("\n\n")
	}
	builder.WriteString("USER QUESTION (answer this specifically):\n")
	builder.WriteString(query)
	builder.WriteString("\n\n")

	builder.WriteString("CONTEXT (use only what's relevant to the question):\n")
	builder.WriteString(ctxText)
	builder.WriteString("\n\n")

	builder.WriteString("CRITICAL INSTRUCTIONS:\n")
	builder.WriteString("1. Answer ONLY the specific question asked above\n")
	builder.WriteString("2. If the question asks about 'EmbedQuery', explain ONLY EmbedQuery\n")
	builder.WriteString("3. If the question asks about a specific function, explain ONLY that function\n")
	builder.WriteString("4. Do NOT explain other functions unless they are directly called by the target function\n")
	builder.WriteString("5. Use only information from the Context that is relevant to the question\n")
	builder.WriteString("6. If the Context doesn't contain the answer, say 'The provided context does not contain information about [topic]'\n")

	return builder.String()
}
