#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

echo "=== Scoring Validation & Gatekeeping Results ==="
echo ""

if [ -z "$(ls results/ 2>/dev/null)" ]; then
    echo "No results found. Run the example first:"
    echo "  go run . -model qwen3:4b"
    exit 1
fi

go run . -score
