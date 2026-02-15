#!/usr/bin/env bash
# Score search reranking results against gold-standard rankings.
# Uses the Go program's built-in scoring (NDCG@10, MRR).
#
# Usage: ./score.sh
set -euo pipefail
cd "$(dirname "$0")"

if [ ! -d results ] || [ -z "$(ls results/*.json 2>/dev/null)" ]; then
    echo "No results found. Run the example first:"
    echo "  go run . -model qwen3:4b"
    exit 1
fi

echo "=== Search Reranking Scores ==="
echo ""
go run . -score
echo ""
echo "Metrics:"
echo "  NDCG@10 — Normalized Discounted Cumulative Gain at rank 10"
echo "           Measures ranking quality, weighting top positions more heavily"
echo "           1.0 = perfect ranking, 0.0 = worst possible ranking"
echo ""
echo "  MRR     — Mean Reciprocal Rank (threshold: relevance >= 3)"
echo "           1/position of first highly-relevant result"
echo "           1.0 = best result is rank 1, 0.5 = rank 2, etc."
