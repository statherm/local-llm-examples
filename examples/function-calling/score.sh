#!/usr/bin/env bash
# score.sh — Score function calling results against expected outputs.
#
# Usage: ./score.sh <results-dir>
#
# Compares each result JSON in results-dir against expected/ ground truth.

set -euo pipefail

RESULTS_DIR="${1:?Usage: $0 <results-dir>}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
EXPECTED_DIR="$SCRIPT_DIR/expected"

echo "=== Scoring function calling results ==="
echo ""

for scenario in developer home-automation; do
    for result_file in "$RESULTS_DIR"/${scenario}-*.json; do
        [ -f "$result_file" ] || continue
        model=$(basename "$result_file" .json | sed "s/^${scenario}-//")
        expected_file="$EXPECTED_DIR/${scenario}.json"

        total=$(jq length "$expected_file")
        tool_correct=0
        param_correct=0
        both_correct=0

        for i in $(seq 0 $((total - 1))); do
            id=$(jq -r ".[$i].id" "$expected_file")
            exp_tool=$(jq -r ".[$i].tool" "$expected_file")
            act_tool=$(jq -r ".[] | select(.id==\"$id\") | .tool // \"\"" "$result_file" 2>/dev/null)

            if [ "$(echo "$exp_tool" | tr '[:upper:]' '[:lower:]')" = "$(echo "$act_tool" | tr '[:upper:]' '[:lower:]')" ]; then
                tool_correct=$((tool_correct + 1))
                tool_ok=true
            else
                echo "  MISS $id tool: expected=$exp_tool got=$act_tool"
                tool_ok=false
            fi

            # Check parameters (simplified: compare JSON representations)
            exp_params=$(jq -cS ".[$i].parameters // {}" "$expected_file")
            act_params=$(jq -cS ".[] | select(.id==\"$id\") | .parameters // {}" "$result_file" 2>/dev/null || echo "{}")

            if [ "$exp_params" = "$act_params" ]; then
                param_correct=$((param_correct + 1))
                param_ok=true
            else
                echo "  MISS $id params: expected=$exp_params got=$act_params"
                param_ok=false
            fi

            if [ "$tool_ok" = true ] && [ "$param_ok" = true ]; then
                both_correct=$((both_correct + 1))
            fi
        done

        echo ""
        echo "Function Calling: $scenario ($model):"
        echo "  Tool selection:  $tool_correct/$total"
        echo "  Parameters:      $param_correct/$total"
        echo "  Combined:        $both_correct/$total"
        echo ""
    done
done
