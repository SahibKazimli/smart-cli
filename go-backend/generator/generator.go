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

const SystemPrompt = `
<systemPrompt>
  <overview>You are an AI assistant integrated into a command-line interface. Explain things in plain text, without Markdown or special formatting.</overview>

  <job>
    <task>Review code snippets and highlight potential errors.</task>
    <task>Explain errors in clear, concise language.</task>
    <task>Suggest fixes or improvements.</task>
    <task>Optionally provide system or command-line suggestions.</task>
  </job>

  <responseFormat>
    <rule>Keep explanations short and actionable.</rule>
    <rule>Avoid unnecessary verbosity.</rule>
    <rule>Use plain text or simple structured output.</rule>
  </responseFormat>

  <constraints>
    <limit>Maximum 700 tokens.</limit>
    <limit>Be precise and relevant to the input provided.</limit>
    <limit>Output must be plain text only. Do not use Markdown.</limit>
  </constraints>
</systemPrompt>
`

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

// Helper: budget fit
func fitToBudget(chunks []chunk_retriever.Chunk, max int) []chunk_retriever.Chunk {
	// estimate per-chunk size and include until limit
	if len(chunks) > max {
		return chunks[:max]
	}
	return chunks
}

// Helper: build context (no headers; blank-line separators)
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
func assemblePrompt(system, query, ctxText string) string {
	var builder strings.Builder
	// System prompt included up front for non-system-aware callers
	if system != "" {
		builder.WriteString(system)
		builder.WriteString("\n\n")
	}
	builder.WriteString("Context:\n")
	builder.WriteString(ctxText)
	builder.WriteString("\n\nUser question:\n")
	builder.WriteString(query)
	builder.WriteString("\n\nInstructions:\n")
	builder.WriteString("- Use only the Context above; do not invent details.\n")
	builder.WriteString("- If the context is insufficient, say so briefly.\n")
	builder.WriteString("- Prefer precise, concise answers; include code when useful.\n")
	return builder.String()
}
