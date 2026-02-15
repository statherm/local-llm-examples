#!/usr/bin/env bash
# Score test data generation results against schema constraints.
# Uses the Go program's built-in validation (schema compliance, rules, uniqueness).
#
# Usage: ./score.sh
set -euo pipefail
cd "$(dirname "$0")"

if [ ! -d results ] || [ -z "$(ls results/*.json 2>/dev/null)" ]; then
    echo "No results found. Run the example first:"
    echo "  go run . -model qwen3:4b"
    exit 1
fi

echo "=== Test Data Generation Scores ==="
echo ""
go run . -score
echo ""
echo "Metrics:"
echo "  Schema Compliance — Percentage of fields present with correct types"
echo "  Rule Compliance   — Percentage of field values passing constraint rules"
echo "                      (range checks, pattern matching, enum values, etc.)"
echo "  Uniqueness        — Percentage of unique values for fields marked unique"
echo "  Overall           — Weighted: 40% schema + 40% rules + 20% uniqueness"
