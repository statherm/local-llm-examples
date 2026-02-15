#!/usr/bin/env bash
# score.sh — Score classification results against expected labels.
#
# Usage: ./score.sh <results-dir>
#
# Compares each result JSON in results-dir against expected/ ground truth.

set -euo pipefail

RESULTS_DIR="${1:?Usage: $0 <results-dir>}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
EXPECTED_DIR="$SCRIPT_DIR/expected"

echo "=== Scoring classification results ==="
echo ""

# Score issue triage results
for result_file in "$RESULTS_DIR"/issues-*.json; do
    [ -f "$result_file" ] || continue
    model=$(basename "$result_file" .json | sed 's/^issues-//')
    expected_file="$EXPECTED_DIR/issues.json"

    total=$(jq length "$expected_file")
    cat_correct=0
    pri_correct=0
    both_correct=0

    for i in $(seq 0 $((total - 1))); do
        id=$(jq -r ".[$i].id" "$expected_file")
        exp_cat=$(jq -r ".[$i].category" "$expected_file")
        exp_pri=$(jq -r ".[$i].priority" "$expected_file")
        act_cat=$(jq -r ".[] | select(.id==\"$id\") | .category" "$result_file" 2>/dev/null || echo "")
        act_pri=$(jq -r ".[] | select(.id==\"$id\") | .priority" "$result_file" 2>/dev/null || echo "")

        if [ "$exp_cat" = "$act_cat" ]; then
            cat_correct=$((cat_correct + 1))
        else
            echo "  MISS $id category: expected=$exp_cat got=$act_cat"
        fi
        if [ "$exp_pri" = "$act_pri" ]; then
            pri_correct=$((pri_correct + 1))
        else
            echo "  MISS $id priority: expected=$exp_pri got=$act_pri"
        fi
        if [ "$exp_cat" = "$act_cat" ] && [ "$exp_pri" = "$act_pri" ]; then
            both_correct=$((both_correct + 1))
        fi
    done

    echo ""
    echo "Issue Triage ($model):"
    echo "  Category accuracy: $cat_correct/$total"
    echo "  Priority accuracy: $pri_correct/$total"
    echo "  Combined accuracy: $both_correct/$total"
    echo ""
done

# Score intent detection results
for result_file in "$RESULTS_DIR"/messages-*.json; do
    [ -f "$result_file" ] || continue
    model=$(basename "$result_file" .json | sed 's/^messages-//')
    expected_file="$EXPECTED_DIR/messages.json"

    total=$(jq length "$expected_file")
    intent_correct=0
    sent_correct=0
    human_correct=0

    for i in $(seq 0 $((total - 1))); do
        id=$(jq -r ".[$i].id" "$expected_file")
        exp_intent=$(jq -r ".[$i].intent" "$expected_file")
        exp_sent=$(jq -r ".[$i].sentiment" "$expected_file")
        exp_human=$(jq -r ".[$i].needs_human" "$expected_file")
        act_intent=$(jq -r ".[] | select(.id==\"$id\") | .intent" "$result_file" 2>/dev/null || echo "")
        act_sent=$(jq -r ".[] | select(.id==\"$id\") | .sentiment" "$result_file" 2>/dev/null || echo "")
        act_human=$(jq -r ".[] | select(.id==\"$id\") | .needs_human" "$result_file" 2>/dev/null || echo "")

        [ "$exp_intent" = "$act_intent" ] && intent_correct=$((intent_correct + 1)) || echo "  MISS $id intent: expected=$exp_intent got=$act_intent"
        [ "$exp_sent" = "$act_sent" ] && sent_correct=$((sent_correct + 1)) || echo "  MISS $id sentiment: expected=$exp_sent got=$act_sent"
        [ "$exp_human" = "$act_human" ] && human_correct=$((human_correct + 1)) || echo "  MISS $id needs_human: expected=$exp_human got=$act_human"
    done

    echo ""
    echo "Intent Detection ($model):"
    echo "  Intent accuracy:     $intent_correct/$total"
    echo "  Sentiment accuracy:  $sent_correct/$total"
    echo "  Needs-human accuracy: $human_correct/$total"
    echo ""
done
