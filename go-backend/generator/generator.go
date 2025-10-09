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

	temp := float32(0.7)
	topP := float32(0.95)
	topK := float32(15)
	maxTokens := int32(2048)

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

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	if resp.Candidates[0] == nil || resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("first candidate or its content is nil")
	}

	if len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no parts in first candidate's content")
	}

	// Extract text from the first valid candidate
	var textParts []string
	for _, candidate := range resp.Candidates {
		if candidate != nil && candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if part != nil && part.Text != "" {
					textParts = append(textParts, part.Text)
				}
			}
			if len(textParts) > 0 {
				break // Found text in this candidate, stop looking
			}
		}
	}

	if len(textParts) == 0 {
		return "", fmt.Errorf("no text parts found in response")
	}

	responseText := strings.Join(textParts, "")
	return strings.TrimSpace(responseText), nil
}

// ===== Helpers =====

// Helper: build context (no headers; blank-line separators)
func buildContext(chunks []chunk_retriever.Chunk) string {
	const charBudget = 50000
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
	return fmt.Sprintf(`<prompt>
  <meta>
    <version>1.0</version>
    <tool>smart-cli</tool>
    <purpose>Answer user questions based on retrieved context</purpose>
    <model_guidelines>
      <temperature>0.7</temperature>
      <top_p>0.95</top_p>
      <top_k>15</top_k>
      <max_tokens>2048</max_tokens>
    </model_guidelines>
  </meta>

  <system>
    %s
  </system>

  <user_query>
    %s
  </user_query>

  <context>
    %s
  </context>

  <instructions>
    <rule>Answer ONLY the specific question asked above.</rule>
    <rule>Give an overview of the file’s purpose if relevant.</rule>
    <rule>If the question refers to a function, explain ONLY that function.</rule>
    <rule>Do NOT explain unrelated code unless directly referenced.</rule>
    <rule>Use only information from the provided context.</rule>
    <rule>If the context lacks the answer, say: "The provided context does not contain information about [topic]."</rule>
    <format>Use concise, technical explanations suitable for CLI output.</format>
  </instructions>
</prompt>`, system, query, ctxText)
}
