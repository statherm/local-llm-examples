package types

import "time"

// ModelMetadata captures performance metrics from a single model call.
type ModelMetadata struct {
	Model       string        `json:"model"`
	TokensIn    int           `json:"tokens_in"`
	TokensOut   int           `json:"tokens_out"`
	TTFT        time.Duration `json:"ttft"`
	TotalTime   time.Duration `json:"total_time"`
	TokensPerSec float64      `json:"tokens_per_sec"`
}

// BenchmarkResult holds the outcome of running one model on one example.
type BenchmarkResult struct {
	Example     string        `json:"example"`
	Model       string        `json:"model"`
	Quality     float64       `json:"quality"`
	QualityName string        `json:"quality_name"`
	TokensIn    int           `json:"tokens_in"`
	TokensOut   int           `json:"tokens_out"`
	TTFT        time.Duration `json:"ttft"`
	TotalTime   time.Duration `json:"total_time"`
	TokensPerSec float64      `json:"tokens_per_sec"`
	CostUSD     float64       `json:"cost_usd"`
}

// FieldResult describes the match outcome for a single JSON field.
type FieldResult struct {
	Field    string `json:"field"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Match    bool   `json:"match"`
}
