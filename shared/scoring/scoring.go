package scoring

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/statherm/local-llm-examples/shared/types"
)

// ExactMatch returns true if expected and actual are identical after
// trimming whitespace and normalizing to lowercase.
func ExactMatch(expected, actual string) bool {
	return strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(actual))
}

// F1Score computes token-level F1 between expected and actual string slices.
// Returns 0 if both slices are empty.
func F1Score(expected, actual []string) float64 {
	if len(expected) == 0 && len(actual) == 0 {
		return 0
	}

	expectedSet := make(map[string]int)
	for _, v := range expected {
		expectedSet[strings.ToLower(strings.TrimSpace(v))]++
	}
	actualSet := make(map[string]int)
	for _, v := range actual {
		actualSet[strings.ToLower(strings.TrimSpace(v))]++
	}

	var tp float64
	for k, count := range actualSet {
		if expCount, ok := expectedSet[k]; ok {
			if count < expCount {
				tp += float64(count)
			} else {
				tp += float64(expCount)
			}
		}
	}

	precision := tp / float64(len(actual))
	recall := tp / float64(len(expected))

	if precision+recall == 0 {
		return 0
	}
	return 2 * precision * recall / (precision + recall)
}

// JSONFieldMatch compares two JSON objects field-by-field at the top level.
// It returns the number of matching fields, total expected fields, and per-field details.
func JSONFieldMatch(expected, actual json.RawMessage) (int, int, []types.FieldResult) {
	var expMap map[string]json.RawMessage
	var actMap map[string]json.RawMessage

	if err := json.Unmarshal(expected, &expMap); err != nil {
		return 0, 0, nil
	}
	if err := json.Unmarshal(actual, &actMap); err != nil {
		return 0, len(expMap), nil
	}

	var matched int
	var details []types.FieldResult

	for field, expVal := range expMap {
		actVal, ok := actMap[field]
		expStr := strings.TrimSpace(string(expVal))
		actStr := ""
		if ok {
			actStr = strings.TrimSpace(string(actVal))
		}

		isMatch := ok && expStr == actStr
		if isMatch {
			matched++
		}

		details = append(details, types.FieldResult{
			Field:    field,
			Expected: expStr,
			Actual:   actStr,
			Match:    isMatch,
		})
	}

	return matched, len(expMap), details
}

// AccuracyScore computes the fraction of predictions that exactly match labels.
// Both slices must have the same length.
func AccuracyScore(predictions, labels []string) (float64, error) {
	if len(predictions) != len(labels) {
		return 0, fmt.Errorf("length mismatch: %d predictions vs %d labels", len(predictions), len(labels))
	}
	if len(predictions) == 0 {
		return 0, nil
	}

	var correct int
	for i := range predictions {
		if ExactMatch(predictions[i], labels[i]) {
			correct++
		}
	}
	return float64(correct) / float64(len(predictions)), nil
}
