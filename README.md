# Local LLM Embedded Tooling Examples

A curated suite of working examples demonstrating small, locally-run language models (2B-7B parameters) as embedded tools within practical pipelines. Each example includes deterministic quality scoring and side-by-side comparison with commercial model baselines.

This is not a benchmark leaderboard. It is a cookbook with proof.

## Prerequisites

- **Go 1.22+** -- https://go.dev/dl/
- **Ollama** -- https://ollama.com with at least one model pulled (e.g. `ollama pull qwen3:4b`)

## Quick Start

```bash
# Pull a model
ollama pull qwen3:4b

# Run an example
make run-example EXAMPLE=structured-extraction MODEL=qwen3:4b

# Score the results
make score EXAMPLE=structured-extraction

# Generate a markdown report
make report EXAMPLE=structured-extraction
```

## Examples

| # | Example | Task | Phase |
|---|---------|------|-------|
| 01 | structured-extraction | Extract structured JSON from unstructured text | 1 |
| 02 | classification-routing | Classify and route inputs to handlers | 1 |
| 03 | function-calling | Select and parameterize tool calls | 1 |
| 04 | summarization | Condense text (PRs, changelogs, logs) | 2 |
| 05 | format-conversion | Convert between formats (Markdown, JSON, etc.) | 2 |
| 06 | validation-gatekeeping | Validate inputs for safety and schema compliance | 2 |
| 07 | search-reranking | Rerank search results by semantic relevance | 3 |
| 08 | test-data-generation | Generate synthetic but realistic test data | 3 |

## Models Under Test

**Primary (every example):** Qwen3-4B, Ministral-3-3B, Phi-3-mini, Llama3.2-3B

**Secondary (select examples):** Gemma-2-2B, Mistral-7B-v0.3, Qwen2.5-Coder-7B, DeepSeek-R1-Distill-Qwen-7B

**Baselines (for comparison):** Claude Opus 4.6, Claude Sonnet 4.5, GPT-4o

## Project Structure

```
local-llm-examples/
├── examples/          # One directory per example category
│   └── <category>/
│       ├── main.go    # Runnable example
│       ├── testdata/  # Input fixtures
│       └── results/   # Model output + scores
├── shared/            # Shared Go packages
│   ├── ollama/        # Ollama HTTP client
│   ├── scoring/       # Deterministic scoring functions
│   ├── reporting/     # Markdown report generator
│   └── types/         # Common types
├── results/           # Cross-example comparison reports
├── planning/          # Planning documents
├── docs/              # Research and gap analysis
├── Makefile           # Top-level orchestration
└── README.md          # This file
```

## Scoring Philosophy

All scoring is deterministic -- no LLM-as-judge. Each example uses task-appropriate metrics: exact match, F1, accuracy, field-level JSON comparison, or ROUGE scores.

## License

MIT
