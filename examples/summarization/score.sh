#!/usr/bin/env bash
set -euo pipefail

# Score summarization results.
# Usage: ./score.sh [results-file]

cd "$(dirname "$0")"

if [ $# -ge 1 ]; then
    echo "Scoring $1..."
    go run . -score
else
    echo "Scoring all results..."
    go run . -score
fi
