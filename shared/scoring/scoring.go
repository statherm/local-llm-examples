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
// Key lookup is case-insensitive and values are compared after JSON normalization
// (so "29451023" and 29451023 are considered equal).
func JSONFieldMatch(expected, actual json.RawMessage) (int, int, []types.FieldResult) {
	var expMap map[string]json.RawMessage
	var actMap map[string]json.RawMessage

	if err := json.Unmarshal(expected, &expMap); err != nil {
		return 0, 0, nil
	}
	if err := json.Unmarshal(actual, &actMap); err != nil {
		return 0, len(expMap), nil
	}

	// Build case-insensitive lookup for actual keys
	actLower := make(map[string]json.RawMessage)
	for k, v := range actMap {
		actLower[strings.ToLower(k)] = v
	}

	var matched int
	var details []types.FieldResult

	for field, expVal := range expMap {
		// Try exact key first, then case-insensitive
		actVal, ok := actMap[field]
		if !ok {
			actVal, ok = actLower[strings.ToLower(field)]
		}

		expStr := strings.TrimSpace(string(expVal))
		actStr := ""
		if ok {
			actStr = strings.TrimSpace(string(actVal))
		}

		isMatch := ok && jsonValuesEqual(expStr, actStr)
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

// jsonValuesEqual compares two raw JSON value strings with normalization.
// Handles: string/number equivalence ("29451023" == 29451023), case-insensitive
// string comparison, and whitespace-insensitive array/object comparison.
func jsonValuesEqual(a, b string) bool {
	if a == b {
		return true
	}

	// Try parsing both into generic interface{} and compare normalized
	var aVal, bVal interface{}
	if err := json.Unmarshal([]byte(a), &aVal); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bVal); err != nil {
		return false
	}

	// Re-marshal both to canonical JSON for comparison
	aNorm, err1 := json.Marshal(aVal)
	bNorm, err2 := json.Marshal(bVal)
	if err1 == nil && err2 == nil && string(aNorm) == string(bNorm) {
		return true
	}

	// Handle string/number cross-type: "42" vs 42
	aStr, aIsStr := aVal.(string)
	bStr, bIsStr := bVal.(string)
	if aIsStr && !bIsStr {
		bJSON, _ := json.Marshal(bVal)
		return aStr == strings.Trim(string(bJSON), "\"")
	}
	if bIsStr && !aIsStr {
		aJSON, _ := json.Marshal(aVal)
		return bStr == strings.Trim(string(aJSON), "\"")
	}

	// Case-insensitive string comparison
	if aIsStr && bIsStr {
		return strings.EqualFold(aStr, bStr)
	}

	return false
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
