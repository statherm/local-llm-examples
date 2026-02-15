package reporting

import (
	"fmt"
	"strings"

	"github.com/statherm/local-llm-examples/shared/types"
)

// GenerateReport produces a Markdown table summarizing benchmark results.
func GenerateReport(results []types.BenchmarkResult) string {
	if len(results) == 0 {
		return "_No results._\n"
	}

	var sb strings.Builder

	sb.WriteString("## Benchmark Results\n\n")
	sb.WriteString("| Model | Quality | Metric | Tokens In | Tokens Out | Tok/s | TTFT | Total | Cost |\n")
	sb.WriteString("|-------|---------|--------|-----------|------------|-------|------|-------|------|\n")

	for _, r := range results {
		qualityStr := fmt.Sprintf("%.1f%%", r.Quality*100)
		ttftStr := fmt.Sprintf("%.0fms", r.TTFT.Seconds()*1000)
		totalStr := fmt.Sprintf("%.2fs", r.TotalTime.Seconds())
		tokSecStr := fmt.Sprintf("%.1f", r.TokensPerSec)
		costStr := "$0.00"
		if r.CostUSD > 0 {
			costStr = fmt.Sprintf("$%.4f", r.CostUSD)
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %d | %s | %s | %s | %s |\n",
			r.Model, qualityStr, r.QualityName,
			r.TokensIn, r.TokensOut,
			tokSecStr, ttftStr, totalStr, costStr,
		))
	}

	sb.WriteString("\n")
	return sb.String()
}
