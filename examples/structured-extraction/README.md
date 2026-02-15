# 01: Structured Extraction

Extract structured JSON from unstructured text using a small local LLM via Ollama.

## What This Demonstrates

A small model (2B-7B parameters) can parse unstructured text documents -- invoices, support tickets, log lines -- and produce clean JSON conforming to a predefined schema. This is one of the most common real-world LLM tasks and one where small models excel due to the constrained output space.

## Scenarios

| Scenario | Input | Output Schema |
|----------|-------|---------------|
| **Invoices** | Plain-text invoices with varying formats and complexity | vendor, line items, totals, currency, payment terms |
| **Support Tickets** | Customer email/form submissions | customer name, product, category, severity, requested action |
| **Log Events** | Mixed log formats (Apache, syslog, JSON, nginx) | timestamp, level, source, message, metadata |

## Prerequisites

- [Ollama](https://ollama.ai) running locally
- A model pulled (e.g., `ollama pull qwen3:4b`)
- Go 1.21+

## Usage

Run all scenarios with the default model:

```sh
make run
```

Run a specific scenario:

```sh
make run SCENARIO=invoices
```

Run with a different model:

```sh
make run MODEL=llama3.2:3b
```

Run directly with Go:

```sh
go run . -model=qwen3:4b -scenario=invoices
```

## Scoring

The output is scored by comparing JSON fields against ground truth expected outputs. Metrics:

- **Field-level exact match** -- per-field accuracy across all test cases
- **Schema compliance** -- whether the output is valid JSON matching the schema
- **Latency** -- time to first token and total generation time
- **Token usage** -- input and output token counts

## Test Data

All test data is hand-crafted to cover realistic scenarios:

- **Invoices**: Simple 2-line to complex multi-item with tax, multiple currencies (USD, EUR, GBP)
- **Tickets**: Billing disputes, technical outages, shipping delays, product defects, feature requests
- **Logs**: Apache access logs, syslog, structured JSON, nginx errors, PostgreSQL warnings

## File Structure

```
structured-extraction/
├── main.go              # Example implementation
├── schemas/             # JSON schemas for each scenario
├── prompts/             # Prompt templates (schema-in-prompt approach)
├── testdata/            # Input fixtures
│   ├── invoices/        # 5 invoices of varying complexity
│   ├── tickets/         # 5 support tickets across categories
│   └── logs/            # 5 log lines in different formats
├── expected/            # Ground truth JSON outputs
├── score.sh             # Standalone scoring script
├── Makefile             # Build and run targets
└── README.md            # This file
```
