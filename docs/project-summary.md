# Project Summary: Local LLM Embedded Tooling Examples

A short overview of what this repo is, what each example does, who it’s for, and how the code is structured.

---

## What This Repo Is

A **cookbook of runnable examples** that show small, local LLMs (roughly 2B–7B parameters) used as **embedded tools** inside real pipelines. Each example:

1. Calls a model via **Ollama** (and optionally commercial APIs for comparison)
2. Scores the output with **deterministic metrics** (no LLM-as-judge)
3. Records quality, tokens, latency, and cost

The goal is **proof by running code**: right-size the model — use a small local model where it’s good enough instead of defaulting to a large API.

---

## What Each Example Does

| # | Example | What it does |
|---|---------|--------------|
| 01 | **structured-extraction** | Takes unstructured text (invoices, support tickets, log lines), sends it to the model with a prompt, expects JSON. Scores by field-level match against expected JSON. |
| 02 | **classification-routing** | Classifies inputs (e.g. GitHub issues → category/priority, support messages → intent/sentiment/needs_human). Parses JSON from the model and scores with exact match or F1 vs expected labels. |
| 03 | **function-calling** | Given a user request and a tool schema (e.g. developer tools, home automation), the model chooses a tool and parameters. Output is JSON `{tool, parameters}`. Scored by correct tool and correct params. |
| 04 | **summarization** | Condenses text (e.g. PRs, changelogs, logs). Quality via ROUGE (or similar) and length. |
| 05 | **format-conversion** | Converts between formats (e.g. Markdown ↔ JSON). Scored with schema validation and field accuracy. |
| 06 | **validation-gatekeeping** | Validates inputs (e.g. PII detection, prompt safety). Scored by accuracy and false positive/negative rates. |
| 07 | **search-reranking** | Reranks search results (API, code, docs) by relevance. Quality is position/ranking metrics. |
| 08 | **test-data-generation** | Generates synthetic but realistic test data (e.g. API responses, transactions, user profiles). Scored for schema and plausibility. |

Common pattern: load testdata and prompts → call Ollama (and optionally baseline APIs) → parse output (usually JSON) → score with shared helpers → record metadata and optionally print a report table.

---

## Who This Is For

- **Developers** evaluating local LLMs and wanting to see what they’re good at in practice
- **Teams** deciding build-vs-buy for AI-augmented tooling (triage, extraction, validation)
- **Practitioners** who want “run this, see score and latency” rather than benchmark numbers alone
- **The broader OSS community** — framing local models as practical tools, not only “40% on SWE-Bench”

Success looks like: a developer can clone, run an example, and understand the value in under about 15 minutes, with reproducible results.

---

## How the Code Is Structured

- **One directory per example** under `examples/<category>/`: `main.go`, `testdata/`, prompts, `results/`. Each `main.go` is flag-driven (`-model`, `-scenario`, `-score`, `-report`), reads from files, calls shared client and scoring, writes results.
- **Shared packages** under `shared/`:
  - **ollama** — HTTP client for the Ollama API; JSON request/response; token counts and timings; optional JSON mode and output token cap.
  - **scoring** — Deterministic helpers: `JSONFieldMatch`, `ExactMatch`, `F1Score`, etc., with per-field details for debugging.
  - **reporting** — Produces a Markdown table (model, quality, tokens, tok/s, TTFT, total time, cost).
  - **types** — Common types (e.g. benchmark result, model metadata).
- **No LLM-as-judge** — all scoring is deterministic and task-appropriate (exact match, F1, field match, ROUGE, etc.).

The code is consistent, runnable, and focused on proving that small local models can be embedded in pipelines with measurable quality at zero marginal cost and clear latency.

---

## More Detail

- **Intent, phasing, models:** [planning/00-project-overview.md](../planning/00-project-overview.md)
- **Vision and principles:** [planning/README.md](../planning/README.md)
- **Why this project exists:** [docs/gap-analysis.md](gap-analysis.md)
- **Research and benchmarks:** [docs/research-findings.md](research-findings.md)
