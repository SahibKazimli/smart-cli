package generator

import (
	"context"
	"sort"
	"strings"

	"smart-cli/go-backend/chunk_retriever"

	"cloud.google.com/go/vertexai/genai"
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
	projectID string
	location  string
	modelName string

	// LLM client and model for generation
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewAgent(ctx context.Context, projectID, location, modelName string) (*Generator, error) {
	// Initialize Vertex AI Generative client and model
	client, err := genai.NewClient(ctx, projectID, location)
	if err != nil {
		return nil, err
	}
	m := client.GenerativeModel(modelName)
	// Configure generation params; enforce your token/style constraints
	temp := float32(0.2)
	topP := float32(0.9)
	topK := int32(32)
	maxTokens := int32(700)

	m.GenerationConfig = genai.GenerationConfig{
		Temperature:     &temp,
		TopP:            &topP,
		TopK:            &topK,
		MaxOutputTokens: &maxTokens,
	}
	// Set a stable system instruction matching your constraints
	m.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(SystemPrompt)},
	}

	return &Generator{
		projectID: projectID,
		location:  location,
		modelName: modelName,
		client:    client,
		model:     m,
	}, nil
}

func (g *Generator) Close() error {
	if g != nil && g.client != nil {
		return g.client.Close()
	}
	return nil
}

func (g *Generator) Answer(ctx context.Context, query string, chunks []chunk_retriever.Chunk) (string, error) {
	// Build retrieved context and assemble the prompt
	ctxText := buildContext(chunks)
	prompt := assemblePrompt("", query, ctxText)
	// Require a configured model
	if g == nil || g.model == nil {
		// Fallback: return the prompt preview if model not initialized
		return prompt, nil
	}

	// Call the model
	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	// Extract plain text from candidates
	var out strings.Builder
	if resp != nil {
		for _, c := range resp.Candidates {
			if c == nil || c.Content == nil {
				continue
			}
			for _, p := range c.Content.Parts {
				if t, ok := p.(genai.Text); ok {
					out.WriteString(string(t))
				}
			}
		}
	}
	return strings.TrimSpace(out.String()), nil
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
