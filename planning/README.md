# Planning: Local LLM Embedded Tooling Examples

This directory contains planning documents for building a suite of practical examples that demonstrate where small, locally-run language models shine as embedded tools.

## Project Vision

Demonstrate through working, runnable examples that small local LLMs are the **right tool for many jobs** — not a compromise, but a deliberate, optimal choice for focused tasks embedded in larger workflows.

## Guiding Principles

1. **Practical over theoretical** — every plan must result in something you can `git clone && make run`
2. **Honest comparisons** — always show the commercial model baseline; we're not here to mislead
3. **Right-sized** — each example should use the smallest model that achieves acceptable quality
4. **Pipeline-first** — models are embedded as tools, not standalone chatbots
5. **Reproducible** — Ollama-based, deterministic where possible, documented hardware requirements

## Planning Documents

| Document | Purpose |
|----------|---------|
| [00-project-overview.md](00-project-overview.md) | High-level intent, scope, architecture, and phasing |
| [01-structured-extraction.md](01-structured-extraction.md) | Invoice/receipt/document extraction examples |
| [02-classification-routing.md](02-classification-routing.md) | Text classification and request routing examples |
| [03-function-calling.md](03-function-calling.md) | Tool selection and function calling examples |
| [04-summarization.md](04-summarization.md) | Focused summarization (changelogs, PR diffs, logs) |
| [05-format-conversion.md](05-format-conversion.md) | Structured format transformation examples |
| [06-validation-gatekeeping.md](06-validation-gatekeeping.md) | Input validation and safety classification |
| [07-search-reranking.md](07-search-reranking.md) | Semantic reranking of search results |
| [08-test-data-generation.md](08-test-data-generation.md) | Synthetic but realistic test data creation |

## Relationship to specgen

This project is a sibling to [specgen](../../../specgen/), which benchmarks models on code generation. Where specgen asks "how well can this model write code?", this project asks "what tasks can this model do better than you'd expect, embedded as a tool?"

Both projects share:
- Ollama as the local model runner
- Deterministic scoring where possible
- Provider abstraction (local vs. API models for comparison)
- The same M4 Pro Mac Mini as the reference platform

## Status

- [x] Research findings documented (see `docs/research-findings.md`)
- [x] Gap analysis complete (see `docs/gap-analysis.md`)
- [ ] Planning documents drafted
- [ ] Example implementations begun
- [ ] Benchmark results collected
