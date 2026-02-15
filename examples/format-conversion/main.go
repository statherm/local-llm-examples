package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/statherm/local-llm-examples/shared/ollama"
	"github.com/statherm/local-llm-examples/shared/reporting"
	"github.com/statherm/local-llm-examples/shared/scoring"
	"github.com/statherm/local-llm-examples/shared/types"
)

var (
	model      = flag.String("model", "qwen3:4b", "Ollama model to use")
	scoreOnly  = flag.Bool("score", false, "Score existing results without running the model")
	reportOnly = flag.Bool("report", false, "Generate a report from existing results")
)

type scenario struct {
	Name         string
	Category     string // markdown-to-json, log-to-structured, nl-to-yaml, csv-to-json
	InputFile    string
	ExpectedFile string
	PromptFile   string
	JSONMode     bool
}

type result struct {
	Scenario string             `json:"scenario"`
	Model    string             `json:"model"`
	Input    string             `json:"input"`
	Output   string             `json:"output"`
	Expected string             `json:"expected"`
	Meta     types.ModelMetadata `json:"metadata"`
}

func main() {
	flag.Parse()

	scenarios := []scenario{
		{
			Name:         "markdown-packages",
			Category:     "markdown-to-json",
			InputFile:    "testdata/001-markdown-table.md",
			ExpectedFile: "expected/001-markdown.json",
			PromptFile:   "prompts/markdown-to-json.txt",
			JSONMode:     true,
		},
		{
			Name:         "markdown-services",
			Category:     "markdown-to-json",
			InputFile:    "testdata/002-markdown-table.md",
			ExpectedFile: "expected/002-markdown.json",
			PromptFile:   "prompts/markdown-to-json.txt",
			JSONMode:     true,
		},
		{
			Name:         "log-structured-app",
			Category:     "log-to-structured",
			InputFile:    "testdata/003-log-lines.txt",
			ExpectedFile: "expected/003-log.json",
			PromptFile:   "prompts/log-to-structured.txt",
			JSONMode:     true,
		},
		{
			Name:         "log-structured-nginx",
			Category:     "log-to-structured",
			InputFile:    "testdata/004-log-lines.txt",
			ExpectedFile: "expected/004-log.json",
			PromptFile:   "prompts/log-to-structured.txt",
			JSONMode:     true,
		},
		{
			Name:         "nl-config-server",
			Category:     "nl-to-yaml",
			InputFile:    "testdata/005-nl-config.txt",
			ExpectedFile: "expected/005-config.yaml",
			PromptFile:   "prompts/nl-to-yaml.txt",
		},
		{
			Name:         "nl-config-redis",
			Category:     "nl-to-yaml",
			InputFile:    "testdata/006-nl-config.txt",
			ExpectedFile: "expected/006-config.yaml",
			PromptFile:   "prompts/nl-to-yaml.txt",
		},
		{
			Name:         "csv-users",
			Category:     "csv-to-json",
			InputFile:    "testdata/007-csv-data.csv",
			ExpectedFile: "expected/007-csv.json",
			PromptFile:   "prompts/csv-to-json.txt",
			JSONMode:     true,
		},
		{
			Name:         "csv-products",
			Category:     "csv-to-json",
			InputFile:    "testdata/008-csv-data.csv",
			ExpectedFile: "expected/008-csv.json",
			PromptFile:   "prompts/csv-to-json.txt",
			JSONMode:     true,
		},
	}

	if *reportOnly {
		generateReport()
		return
	}

	if *scoreOnly {
		scoreResults(scenarios)
		return
	}

	runScenarios(scenarios)
}

func runScenarios(scenarios []scenario) {
	client := ollama.NewClient()
	var results []result

	for _, s := range scenarios {
		fmt.Printf("Running: %s\n", s.Name)

		input, err := os.ReadFile(s.InputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR reading input: %v\n", err)
			continue
		}

		promptTmpl, err := os.ReadFile(s.PromptFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR reading prompt: %v\n", err)
			continue
		}

		prompt, err := renderPrompt(string(promptTmpl), string(input))
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR rendering prompt: %v\n", err)
			continue
		}

		expected, err := os.ReadFile(s.ExpectedFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR reading expected: %v\n", err)
			continue
		}

		output, meta, err := client.ChatCompletion(*model, "", prompt, s.JSONMode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR from model: %v\n", err)
			continue
		}

		r := result{
			Scenario: s.Name,
			Model:    *model,
			Input:    string(input),
			Output:   output,
			Expected: string(expected),
			Meta:     meta,
		}
		results = append(results, r)

		fmt.Printf("  Model: %s | Tokens: %d in, %d out | %.1f tok/s | %v\n",
			meta.Model, meta.TokensIn, meta.TokensOut, meta.TokensPerSec, meta.TotalTime)
	}

	resultsFile := filepath.Join("results", *model+".json")
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR marshaling results: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll("results", 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR creating results dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(resultsFile, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR writing results: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nResults saved to %s\n", resultsFile)
}

func scoreResults(scenarios []scenario) {
	files, err := filepath.Glob("results/*.json")
	if err != nil || len(files) == 0 {
		fmt.Println("No result files found in results/")
		return
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR reading %s: %v\n", f, err)
			continue
		}

		var results []result
		if err := json.Unmarshal(data, &results); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR parsing %s: %v\n", f, err)
			continue
		}

		fmt.Printf("=== Scores for %s ===\n", filepath.Base(f))
		for _, r := range results {
			sc := scoreScenario(r, scenarios)
			fmt.Printf("  %-25s  quality=%.3f\n", r.Scenario, sc)
		}
	}
}

func scoreScenario(r result, scenarios []scenario) float64 {
	var cat string
	for _, s := range scenarios {
		if s.Name == r.Scenario {
			cat = s.Category
			break
		}
	}

	switch cat {
	case "markdown-to-json", "log-to-structured", "csv-to-json":
		return scoreJSONArray(r.Expected, r.Output)
	case "nl-to-yaml":
		return scoreYAMLConfig(r.Expected, r.Output)
	default:
		return 0
	}
}

// scoreJSONArray compares two JSON arrays element by element using field matching.
func scoreJSONArray(expected, actual string) float64 {
	var expArr []json.RawMessage
	var actArr []json.RawMessage

	if err := json.Unmarshal([]byte(strings.TrimSpace(expected)), &expArr); err != nil {
		return 0
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(actual)), &actArr); err != nil {
		return 0
	}

	if len(expArr) == 0 {
		return 0
	}

	var totalMatched, totalFields int
	limit := len(expArr)
	if len(actArr) < limit {
		limit = len(actArr)
	}

	for i := 0; i < limit; i++ {
		matched, total, _ := scoring.JSONFieldMatch(expArr[i], actArr[i])
		totalMatched += matched
		totalFields += total
	}
	// Count missing rows as unmatched fields
	for i := limit; i < len(expArr); i++ {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(expArr[i], &obj); err == nil {
			totalFields += len(obj)
		}
	}

	if totalFields == 0 {
		return 0
	}
	return float64(totalMatched) / float64(totalFields)
}

// scoreYAMLConfig does a simple key-value token overlap between expected and actual YAML.
func scoreYAMLConfig(expected, actual string) float64 {
	expTokens := extractKeyValues(expected)
	actTokens := extractKeyValues(actual)
	return scoring.F1Score(expTokens, actTokens)
}

// extractKeyValues pulls out "key: value" pairs from YAML-like text.
func extractKeyValues(s string) []string {
	var pairs []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			// Include list items as values
			if strings.HasPrefix(line, "-") {
				pairs = append(pairs, strings.TrimSpace(strings.TrimPrefix(line, "-")))
			}
			continue
		}
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			if val != "" {
				pairs = append(pairs, key+":"+val)
			} else {
				pairs = append(pairs, key)
			}
		}
	}
	return pairs
}

func renderPrompt(tmpl, input string) (string, error) {
	t, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	err = t.Execute(&sb, struct{ Input string }{Input: input})
	return sb.String(), err
}

func generateReport() {
	files, err := filepath.Glob("results/*.json")
	if err != nil || len(files) == 0 {
		fmt.Println("No result files found in results/")
		return
	}

	var benchmarks []types.BenchmarkResult
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var results []result
		if err := json.Unmarshal(data, &results); err != nil {
			continue
		}
		for _, r := range results {
			qName := "field-match"
			if strings.Contains(r.Scenario, "config") {
				qName = "F1"
			}
			benchmarks = append(benchmarks, types.BenchmarkResult{
				Example:      r.Scenario,
				Model:        r.Model,
				Quality:      scoreJSONArray(r.Expected, r.Output),
				QualityName:  qName,
				TokensIn:     r.Meta.TokensIn,
				TokensOut:    r.Meta.TokensOut,
				TTFT:         r.Meta.TTFT,
				TotalTime:    r.Meta.TotalTime,
				TokensPerSec: r.Meta.TokensPerSec,
			})
		}
	}

	report := reporting.GenerateReport(benchmarks)
	fmt.Print(report)
}
