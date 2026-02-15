# Example 08: Test Data Generation

A small local model generates synthetic but realistic test data from schema definitions and business rule constraints. Useful for development, testing, demos, and populating staging environments without using production data.

## Why Small Models Work Here

- **Pattern mimicry, not deep reasoning** -- generating plausible names, addresses, and transactions does not require large model capability
- **Schema-driven** -- the model fills in values guided by explicit field definitions
- **Volume** -- local generation at $0 is compelling when you need thousands of records
- **Privacy** -- no real data leaves your machine
- **Variety over perfection** -- slightly imperfect data is fine for testing; diversity matters more

## Scenarios

| Scenario | Description | Records | Key Constraints |
|----------|-------------|---------|-----------------|
| user_profiles | SaaS user profiles | 10 | US locations, ages 18-65, email format, plan enum |
| transactions | Payment transactions | 10 | UUID format, amount range, ~5% flagged, valid timestamps |
| api_responses | Product catalog API | 5 | PROD-XXXX IDs, stock_count=0 when out of stock, 2-4 tags |

## Running

```bash
# Prerequisites: Ollama running with a model pulled
ollama pull qwen3:4b

# Run with default model
make run

# Run with a specific model
make run MODEL=llama3.2:3b

# Score results against constraints
make score

# Generate comparison report
make report
```

## Scoring

Generated data is validated against three dimensions:

- **Schema Compliance** (40%) -- Every field present with the correct type (string, integer, number, boolean, array).
- **Rule Compliance** (40%) -- Field values pass constraint rules: range checks, regex patterns, enum membership, date formats, array lengths.
- **Uniqueness** (20%) -- Fields marked as unique (IDs) have no duplicate values across records.

Constraints are defined declaratively in `constraints/`. Cross-field rules (e.g., stock_count must be 0 when in_stock is false) are also checked.

## How It Works

1. Load a schema definition from `schemas/` (field names, types, descriptions, example)
2. Load business rule constraints from `constraints/`
3. Build a prompt with schema + constraints + example record
4. Send to model with JSON mode enabled
5. Parse the JSON array of generated records
6. Validate every record against schema types and constraint rules
7. Report compliance scores and specific violations

## File Structure

```
test-data-generation/
├── main.go                         # Generation and validation implementation
├── schemas/
│   ├── user_profiles.json          # User profile schema definition
│   ├── transactions.json           # Transaction schema definition
│   └── api_responses.json          # API response schema definition
├── constraints/
│   ├── user_profiles.json          # Validation rules for user profiles
│   ├── transactions.json           # Validation rules for transactions
│   └── api_responses.json          # Validation rules for API responses
├── results/                        # Model outputs (generated)
├── score.sh                        # Scoring script
├── Makefile
└── README.md
```
