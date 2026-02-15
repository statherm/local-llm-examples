#!/usr/bin/env bash
# score.sh — Score structured extraction results against expected outputs.
#
# Usage: ./score.sh <results-dir> <expected-dir>
#
# Compares each JSON file in results-dir against its counterpart in expected-dir
# using top-level field matching. Prints per-file and aggregate scores.

set -euo pipefail

RESULTS_DIR="${1:?Usage: $0 <results-dir> <expected-dir>}"
EXPECTED_DIR="${2:?Usage: $0 <results-dir> <expected-dir>}"

total_fields=0
matched_fields=0
total_files=0
matched_files=0

for expected_file in "$EXPECTED_DIR"/*.json; do
    name=$(basename "$expected_file")
    result_file="$RESULTS_DIR/$name"

    if [ ! -f "$result_file" ]; then
        echo "SKIP  $name (no result file)"
        continue
    fi

    total_files=$((total_files + 1))

    # Extract top-level keys from expected and compare values.
    file_total=0
    file_matched=0

    for key in $(jq -r 'keys[]' "$expected_file"); do
        expected_val=$(jq -r --arg k "$key" '.[$k] | if type == "array" or type == "object" then tojson else tostring end' "$expected_file")
        actual_val=$(jq -r --arg k "$key" '.[$k] // "" | if type == "array" or type == "object" then tojson else tostring end' "$result_file" 2>/dev/null || echo "")

        file_total=$((file_total + 1))
        total_fields=$((total_fields + 1))

        if [ "$expected_val" = "$actual_val" ]; then
            file_matched=$((file_matched + 1))
            matched_fields=$((matched_fields + 1))
        else
            echo "  MISS $name.$key: expected='$expected_val' got='$actual_val'"
        fi
    done

    if [ "$file_total" -eq "$file_matched" ]; then
        echo "PASS  $name ($file_matched/$file_total fields)"
        matched_files=$((matched_files + 1))
    else
        echo "FAIL  $name ($file_matched/$file_total fields)"
    fi
done

echo ""
echo "=== Summary ==="
echo "Files: $matched_files/$total_files perfect match"
if [ "$total_fields" -gt 0 ]; then
    pct=$((matched_fields * 100 / total_fields))
    echo "Fields: $matched_fields/$total_fields ($pct%)"
else
    echo "Fields: 0/0"
fi
