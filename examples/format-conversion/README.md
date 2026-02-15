# Example 05: Format Conversion

Demonstrates small local models converting data between structured formats -- deterministic-ish translation where the information content is preserved and only the representation changes.

## Why This Works for Small Models

- **Information preservation, not creation** -- input and output contain the same data
- **Well-defined source and target formats** -- the model knows exactly what to produce
- **Pattern-heavy** -- format conversions follow repeatable rules
- **Schema-constrainable** -- output can be validated against a target schema

## Scenarios

| Scenario | Input | Output | Scoring |
|----------|-------|--------|---------|
| Markdown table to JSON | Markdown table | JSON array of objects | Field-level exact match |
| Log lines to structured events | Free-form log lines | JSON event array | Field-level exact match |
| Natural language to YAML config | Config description in English | Valid YAML file | Key-value F1 |
| CSV to typed JSON | Raw CSV with headers | JSON with inferred types | Field-level exact match |

## Running

```bash
# Run with default model (qwen3:4b)
make run

# Run with a specific model
make run MODEL=llama3.2:3b

# Score existing results
make score

# Generate markdown report
make report
```

## Test Data

- `testdata/001-markdown-table.md` -- NPM package table (5 rows, 4 columns)
- `testdata/002-markdown-table.md` -- Service status table (5 rows, 5 columns)
- `testdata/003-log-lines.txt` -- Structured application log (5 lines, mixed levels)
- `testdata/004-log-lines.txt` -- Nginx access log (4 lines, syslog format)
- `testdata/005-nl-config.txt` -- Server + database + CORS configuration
- `testdata/006-nl-config.txt` -- Redis configuration with TLS
- `testdata/007-csv-data.csv` -- User records (5 rows, mixed types)
- `testdata/008-csv-data.csv` -- Product catalog (5 rows, numeric + boolean)

## Scoring

- **JSON output** (markdown, logs, CSV): Element-by-element field matching across the output array
- **YAML output** (configs): Key-value pair F1 score against reference YAML

Results are saved to `results/<model>.json`.
