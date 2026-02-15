# Plan 01: Structured Extraction

**Status:** DRAFT
**Parent:** [00-project-overview.md](00-project-overview.md)
**Phase:** 1 (Foundation)

---

## 1. What This Demonstrates

A small local model extracts structured data from unstructured text documents — invoices, receipts, emails, support tickets — and outputs clean JSON conforming to a predefined schema. This is one of the most common real-world LLM tasks and one where small models are expected to perform well due to the constrained output space.

## 2. Why Small Models Should Shine Here

- **Constrained output** — the schema defines exactly what fields to extract; the model isn't generating creative text
- **Pattern recognition** — invoices and receipts follow recognizable layouts
- **Short outputs** — a few dozen fields, not pages of text
- **Academic support** — StructEval and llm-structured-output-benchmarks both show strong SLM performance on extraction tasks
- **Production evidence** — extraction pipelines are 520x faster and 3700x cheaper than vision-LLM baselines when optimized

## 3. Example Scenarios

### 3a. Invoice Extraction
**Input:** Plain-text or markdown representation of an invoice
**Output:** JSON with fields: vendor_name, invoice_number, date, line_items[], subtotal, tax, total, currency, payment_terms
**Scoring:** Exact match per field, F1 across all fields

### 3b. Support Ticket Parsing
**Input:** Raw support ticket text (email body or form submission)
**Output:** JSON with fields: customer_name, product, issue_category, severity, requested_action, contact_info
**Scoring:** Exact match for structured fields, semantic similarity for free-text fields

### 3c. Log Event Structuring
**Input:** Unstructured log lines (mixed formats — syslog, JSON-ish, plain text)
**Output:** Normalized JSON events: timestamp, level, source, message, metadata{}
**Scoring:** Exact match on timestamp/level/source, partial credit on message extraction

## 4. Implementation Approach

```
examples/structured-extraction/
├── README.md
├── main.go
├── schemas/                  # JSON schemas for each scenario
│   ├── invoice.json
│   ├── support-ticket.json
│   └── log-event.json
├── prompts/                  # Prompt templates per scenario
│   ├── invoice.txt
│   ├── support-ticket.txt
│   └── log-event.txt
├── testdata/                 # Input fixtures (10-20 per scenario)
│   ├── invoices/
│   ├── tickets/
│   └── logs/
├── expected/                 # Ground truth outputs
│   ├── invoices/
│   ├── tickets/
│   └── logs/
├── score.sh                  # Deterministic JSON field comparison
└── RESULTS.md
```

### Key design decisions:
- **Prompt includes the JSON schema** — the model is told exactly what structure to produce
- **No framework dependency** — raw Ollama API calls with JSON mode enabled
- **Schema validation first** — before scoring fields, validate the output is valid JSON matching the schema
- **Baseline comparison** — same prompts sent to Claude Sonnet 4.5 and Opus 4.6 via API

## 5. Metrics Captured

| Metric | How |
|--------|-----|
| Schema compliance rate | % of outputs that are valid JSON matching the schema |
| Field-level exact match | Per-field accuracy across all test cases |
| F1 score | Precision/recall across all fields |
| Latency (TTFT) | Time to first token |
| Latency (total) | Total generation time |
| Tokens (in/out) | Token counts for cost estimation |
| Cost | $0 local, estimated API cost for baseline |

## 6. Test Data Strategy

- **Invoices:** Generate 15-20 synthetic invoices with varying complexity (simple 3-line-item to complex multi-page with discounts, shipping, tax variations)
- **Support tickets:** Adapt from public datasets or generate synthetic tickets covering different products, severity levels, and communication styles
- **Logs:** Mix of real-world log formats (Apache, nginx, application JSON logs, syslog)

Ground truth will be manually verified JSON for each input.

## 7. Models to Test

**Primary:** Qwen3-4B, Ministral-3-3B, Phi-3-mini, Llama3.2-3B
**Secondary:** Gemma-2-2B (push the lower bound), Mistral-7B-v0.3 (7B reference)
**Baseline:** Claude Sonnet 4.5, Claude Opus 4.6

## 8. Open Questions

- Should we test with and without few-shot examples in the prompt?
- How much does JSON mode (Ollama's `format: "json"`) vs. free generation + parsing differ in quality?
- Is schema-in-prompt sufficient or do we need grammar-constrained generation (Outlines-style)?
