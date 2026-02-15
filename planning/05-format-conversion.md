# Plan 05: Format Conversion

**Status:** DRAFT
**Parent:** [00-project-overview.md](00-project-overview.md)
**Phase:** 2 (Expansion)

---

## 1. What This Demonstrates

A small local model converts data between structured formats — not creative transformation, but deterministic-ish translation where the information content is preserved and only the representation changes. This sits at the intersection of extraction and generation, and small models should excel because the task is highly constrained.

## 2. Why Small Models Should Shine Here

- **Information preservation, not creation** — input and output contain the same data
- **Well-defined source and target formats** — the model knows exactly what to produce
- **Pattern-heavy** — format conversions follow repeatable rules
- **Schema-constrainable** — output can be validated against a target schema

## 3. Example Scenarios

### 3a. Markdown Table → JSON Array
**Input:** Markdown-formatted table
**Output:** JSON array of objects with column headers as keys
**Scoring:** Exact field match, schema compliance

### 3b. Unstructured Log → Structured Event
**Input:** Free-form log lines (mixed formats)
**Output:** Normalized JSON events: { timestamp, level, source, message, metadata }
**Scoring:** Field extraction accuracy, timestamp parsing correctness

### 3c. Natural Language Config → YAML
**Input:** "Set the server port to 8080, enable CORS for all origins, set the database connection pool to 10, and use INFO log level"
**Output:** Valid YAML configuration file
**Scoring:** Schema validation + field value accuracy

### 3d. CSV → Typed JSON (with inference)
**Input:** Raw CSV with headers
**Output:** JSON array with inferred types (numbers as numbers, dates as ISO strings, booleans as booleans)
**Scoring:** Type inference accuracy, value preservation

## 4. Implementation Approach

```
examples/format-conversion/
├── README.md
├── main.go
├── schemas/              # Target format schemas
├── prompts/
├── testdata/             # Source format inputs
├── expected/             # Expected target format outputs
├── score.sh              # Format validation + content comparison
└── RESULTS.md
```

## 5. Open Questions

- How complex can the source format be before small models break down?
- Is this task better served by traditional parsers for common formats, with LLMs only for ambiguous/messy inputs?
- Should we explicitly compare LLM conversion vs. regex/parser approach for cost-benefit?
