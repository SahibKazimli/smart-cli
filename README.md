## Smart CLI
A modular, AI-enhanced command-line interface that provides code review, error explanation, system suggestions, and more, powered by RAG/Agent modules and Vertex AI.

## Configuring the Generation Model

Smart CLI uses `gemini-2.5-flash` as the default generation model. You can override this by setting the `SMARTCLI_MODEL` environment variable:

```bash
# Use a different model
export SMARTCLI_MODEL=gemini-1.5-pro

# Run with the custom model
./smartcli explain -e "undefined variable"

# Or set it for a single command
SMARTCLI_MODEL=gemini-1.5-pro ./smartcli review -f main.go -q "what does this do?"
```

The model selection priority is:
1. Explicitly passed model parameter (in code)
2. `SMARTCLI_MODEL` environment variable
3. Default: `gemini-2.5-flash`



## Project Goals

- Create an intelligent CLI that enhances standard terminal workflows.
- Provide AI-powered insights for code, system diagnostics, and command suggestions.
- Showcase integration of Go CLI, and cloud LLMs (Vertex AI).

## Current Development
- Repo scanning to enable LLM responses dynamically (works but a bit inconsistent)
- Concurrent index & reindexing of codebases
- Colored/progress-enhanced terminal output
- Context/history tracking for smarter suggestions

## Completed Objectives
- Parallel chunk retrieval and embedding for performance
- Basic CLI command (code-review, error-explanation)
- - AI integration using Vertex AI

## Sidelined objectives 
- Additional skills: system-check.
- Plugin system for modular skill addition.
- Python code is not going to be used, including skills.
- Routing internally based on user input, instead of creating tools (unsure, might do tools)


