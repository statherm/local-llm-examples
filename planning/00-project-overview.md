# Plan 00: Project Overview — Local LLM Embedded Tooling Examples

**Status:** DRAFT
**Date:** 2026-02-14
**Author:** Planning phase

---

## 1. Intent

Build a curated suite of working examples that demonstrate small, locally-run language models as **embedded tools within practical pipelines**. Each example showcases a task category where a model in the 2B-7B range, running via Ollama on consumer hardware, delivers results that are good enough — or surprisingly better — than defaulting to a large commercial model.

This is not a benchmark leaderboard. It's a **cookbook with proof**.

## 2. Who Is This For?

- **Developers** considering local LLMs but unsure what they're actually good at
- **Teams** evaluating build-vs-buy for AI-augmented tooling
- **The open-source community** that deserves better framing than "it scores 40% on SWE-Bench"
- **Anyone** who defaults to API calls for tasks that don't warrant them

## 3. Core Architecture

```
local-llm-examples/
├── docs/                          # Research, gap analysis, references
│   ├── research-findings.md
│   └── gap-analysis.md
├── planning/                      # This directory — planning docs
│   ├── README.md
│   ├── 00-project-overview.md     # ← You are here
│   ├── 01-structured-extraction.md
│   ├── 02-classification-routing.md
│   ├── 03-function-calling.md
│   ├── 04-summarization.md
│   ├── 05-format-conversion.md
│   ├── 06-validation-gatekeeping.md
│   ├── 07-search-reranking.md
│   └── 08-test-data-generation.md
├── examples/                      # Runnable examples (one dir per category)
│   ├── structured-extraction/
│   ├── classification-routing/
│   ├── function-calling/
│   ├── summarization/
│   ├── format-conversion/
│   ├── validation-gatekeeping/
│   ├── search-reranking/
│   └── test-data-generation/
├── shared/                        # Shared code (Ollama client, scoring, etc.)
├── results/                       # Benchmark results and comparisons
├── Makefile                       # Top-level orchestration
└── README.md                      # Project README
```

## 4. Each Example Follows a Standard Pattern

Every example directory contains:

```
examples/<category>/
├── README.md              # What this demonstrates, how to run it
├── main.go (or main.py)   # The example implementation
├── testdata/              # Input fixtures (invoices, logs, text, etc.)
├── baseline/              # Commercial model results for comparison
├── results/               # Local model results
├── score.sh               # Deterministic scoring script
└── RESULTS.md             # Summary with cost/quality/latency comparison
```

### Standard output for every example:
- **Quality score** — task-specific accuracy metric (F1, exact match, Rouge, etc.)
- **Latency** — time to first token + total generation time
- **Token usage** — input + output token counts
- **Cost** — $0 for local, estimated API cost for commercial baseline
- **Model metadata** — name, parameter count, quantization level

## 5. Language Choice

**Go** as the primary implementation language, matching specgen. Rationale:
- Specgen is Go — consistency across sibling projects
- Ollama has a native Go client
- Anthropic and OpenAI have Go SDKs for baseline comparisons
- Single binary distribution, no dependency management headaches
- M4 Pro Mac Mini is the reference platform (ARM64 native Go)

**Exception:** If a specific example benefits significantly from Python tooling (e.g., Instructor, Outlines), we document both, but the Go version is primary.

## 6. Phasing

### Phase 1: Foundation (Examples 01-03)
**Goal:** Prove the pattern works with the strongest candidates.

| # | Example | Why First |
|---|---------|-----------|
| 01 | Structured Extraction | Strongest academic evidence, constrained output |
| 02 | Classification / Routing | Simplest to implement, clearest win for small models |
| 03 | Function Calling | Directly supported by Ministral-3-3B, strong ToolBench evidence |

**Deliverables:**
- Shared Ollama client abstraction
- Shared scoring framework
- Shared comparison report generator
- 3 working examples with results

### Phase 2: Expansion (Examples 04-06)
**Goal:** Cover the "strong intuition" tier.

| # | Example | Notes |
|---|---------|-------|
| 04 | Summarization | PR diffs, changelogs, log condensation |
| 05 | Format Conversion | Markdown → JSON, log → structured event |
| 06 | Validation / Gatekeeping | Input safety, schema compliance |

### Phase 3: Exploration (Examples 07-08)
**Goal:** Test the boundaries.

| # | Example | Notes |
|---|---------|-------|
| 07 | Search Reranking | Semantic reordering of search results |
| 08 | Test Data Generation | Synthetic but realistic data |

## 7. Models Under Test

### Primary candidates (run every example):
| Model | Params | Why |
|-------|--------|-----|
| Qwen3-4B | 4B | Matches 120B teacher when fine-tuned; strong all-rounder |
| Ministral-3-3B | 3B | Purpose-built for function calling + JSON |
| Phi-3-mini | 3.8B | Strong reasoning for size |
| Llama3.2-3B | 3B | Meta baseline, widely used |

### Secondary candidates (run on best-fit examples):
| Model | Params | Why |
|-------|--------|-----|
| Gemma-2-2B | 2B | Smallest viable candidate |
| Mistral-7B-v0.3 | 7B | 7B class baseline |
| Qwen2.5-Coder-7B | 7B | Code-adjacent structured tasks |
| DeepSeek-R1-Distill-Qwen-7B | 7B | Reasoning in small form factor |

### Baselines (for comparison):
| Model | Provider | Purpose |
|-------|----------|---------|
| Claude Opus 4.6 | Anthropic API | Top-tier commercial baseline |
| Claude Sonnet 4.5 | Anthropic API | Mid-tier commercial baseline |
| GPT-4o | OpenAI API | Alternative commercial baseline |

## 8. Scoring Philosophy

Borrowed from specgen: **deterministic scoring, no LLM-as-judge**.

- Structured extraction → exact field match, F1 on field values
- Classification → accuracy, precision, recall, F1
- Function calling → correct tool + correct parameters (exact match)
- Summarization → Rouge-1/2/L, plus length compliance
- Format conversion → schema validation + field accuracy
- Validation → accuracy, false positive/negative rates

## 9. What We Don't Know Yet (Gaps to Fill During Implementation)

1. **Model-to-task mapping specifics** — Which quantization levels (Q4_K_M vs Q5_K_M vs Q8_0) hit the quality sweet spot per task?
2. **Prompt engineering requirements** — How much prompt tuning does each small model need vs. large models that "just work"?
3. **Consistency** — Large models give consistent output; do small models need retry/voting strategies?
4. **Context window limits** — Some tasks (summarization of large inputs) may hit context limits on small models
5. **Cold start vs. warm** — Ollama keeps models loaded; what's the real-world latency profile?

These are explicitly things we'll learn by building the examples, not prerequisites for starting.

## 10. Success Metrics

The project succeeds when:

1. At least 3 examples show a local model achieving **>90% of commercial model quality** at **>10x cost reduction**
2. Every example can be run with `make run` on the reference M4 Pro Mac Mini
3. Results are reproducible — same model + same input = same output (or within documented variance)
4. A developer unfamiliar with local LLMs can clone, run, and understand the value in under 15 minutes
