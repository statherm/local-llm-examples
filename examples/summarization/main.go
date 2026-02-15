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

// scenario defines a summarization test case.
type scenario struct {
	Name         string // human-readable name
	Category     string // diff, commits, log, meeting
	InputFile    string // path to input fixture
	ExpectedFile string // path to expected output
	PromptFile   string // path to prompt template
	JSONMode     bool   // whether to request JSON output
}

// result stores model output for one scenario.
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
			Name:         "diff-changelog-retry",
			Category:     "diff",
			InputFile:    "testdata/diffs/001-add-retry-logic.diff",
			ExpectedFile: "expected/diff-001.txt",
			PromptFile:   "prompts/changelog.txt",
		},
		{
			Name:         "diff-changelog-nullfix",
			Category:     "diff",
			InputFile:    "testdata/diffs/002-fix-null-pointer.diff",
			ExpectedFile: "expected/diff-002.txt",
			PromptFile:   "prompts/changelog.txt",
		},
		{
			Name:         "diff-changelog-pagination",
			Category:     "diff",
			InputFile:    "testdata/diffs/003-add-pagination.diff",
			ExpectedFile: "expected/diff-003.txt",
			PromptFile:   "prompts/changelog.txt",
		},
		{
			Name:         "pr-description-websocket",
			Category:     "commits",
			InputFile:    "testdata/commits/001-feature-branch.txt",
			ExpectedFile: "expected/commits-001.txt",
			PromptFile:   "prompts/pr-description.txt",
		},
		{
			Name:         "pr-description-webhook-fix",
			Category:     "commits",
			InputFile:    "testdata/commits/002-bugfix-branch.txt",
			ExpectedFile: "expected/commits-002.txt",
			PromptFile:   "prompts/pr-description.txt",
		},
		{
			Name:         "log-summary-healthy",
			Category:     "log",
			InputFile:    "testdata/logs/001-healthy-deploy.log",
			ExpectedFile: "expected/log-001.txt",
			PromptFile:   "prompts/log-summary.txt",
		},
		{
			Name:         "log-summary-degraded",
			Category:     "log",
			InputFile:    "testdata/logs/002-degraded-database.log",
			ExpectedFile: "expected/log-002.txt",
			PromptFile:   "prompts/log-summary.txt",
		},
		{
			Name:         "log-summary-crash",
			Category:     "log",
			InputFile:    "testdata/logs/003-crash-loop.log",
			ExpectedFile: "expected/log-003.txt",
			PromptFile:   "prompts/log-summary.txt",
		},
		{
			Name:         "meeting-actions-sprint",
			Category:     "meeting",
			InputFile:    "testdata/meetings/001-sprint-planning.txt",
			ExpectedFile: "expected/meeting-001.json",
			PromptFile:   "prompts/action-items.txt",
			JSONMode:     true,
		},
		{
			Name:         "meeting-actions-retro",
			Category:     "meeting",
			InputFile:    "testdata/meetings/002-incident-retro.txt",
			ExpectedFile: "expected/meeting-002.json",
			PromptFile:   "prompts/action-items.txt",
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

	// Save results
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
			fmt.Printf("  %-30s  quality=%.3f\n", r.Scenario, sc)
		}
	}
}

func scoreScenario(r result, scenarios []scenario) float64 {
	// Find the matching scenario to determine category
	var cat string
	for _, s := range scenarios {
		if s.Name == r.Scenario {
			cat = s.Category
			break
		}
	}

	switch cat {
	case "meeting":
		return scoreMeetingActions(r.Expected, r.Output)
	default:
		return scoreTextSummary(r.Expected, r.Output)
	}
}

// scoreTextSummary computes token-level F1 between expected and actual text.
func scoreTextSummary(expected, actual string) float64 {
	expTokens := tokenize(expected)
	actTokens := tokenize(actual)
	return scoring.F1Score(expTokens, actTokens)
}

// scoreMeetingActions scores action item extraction by comparing JSON arrays.
func scoreMeetingActions(expected, actual string) float64 {
	type actionItem struct {
		Owner    string  `json:"owner"`
		Action   string  `json:"action"`
		Deadline *string `json:"deadline"`
	}

	var expItems, actItems []actionItem
	if err := json.Unmarshal([]byte(expected), &expItems); err != nil {
		return 0
	}
	if err := json.Unmarshal([]byte(actual), &actItems); err != nil {
		return 0
	}

	if len(expItems) == 0 {
		return 0
	}

	// Score by matching owners and checking action text overlap
	matched := 0
	for _, exp := range expItems {
		for _, act := range actItems {
			if scoring.ExactMatch(exp.Owner, act.Owner) {
				// Check if the action descriptions overlap meaningfully
				expWords := tokenize(exp.Action)
				actWords := tokenize(act.Action)
				f1 := scoring.F1Score(expWords, actWords)
				if f1 > 0.3 {
					matched++
					break
				}
			}
		}
	}

	return float64(matched) / float64(len(expItems))
}

// tokenize splits text into lowercase word tokens.
func tokenize(s string) []string {
	words := strings.Fields(strings.ToLower(s))
	var tokens []string
	for _, w := range words {
		// Strip common punctuation
		w = strings.Trim(w, ".,;:!?\"'()[]{}#*-")
		if w != "" {
			tokens = append(tokens, w)
		}
	}
	return tokens
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
			benchmarks = append(benchmarks, types.BenchmarkResult{
				Example:      r.Scenario,
				Model:        r.Model,
				Quality:      scoreTextSummary(r.Expected, r.Output),
				QualityName:  "F1",
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
