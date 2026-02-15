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
		generateReport(scenarios)
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

		// System prompt reinforces array output for JSON mode (meeting actions).
		var sysPrompt string
		if s.JSONMode {
			sysPrompt = "You respond only with valid JSON. When the user asks for action items, you MUST return a JSON array containing ALL items. Do not stop after the first item."
		}
		output, meta, err := client.ChatCompletion(*model, sysPrompt, prompt, s.JSONMode, 2048)
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

// scoreTextSummary scores a text summary using keyword recall: what fraction of
// meaningful keywords from the expected summary appear in the actual output.
// This is more appropriate than token F1 for summarization where phrasing varies
// but key concepts should be preserved.
func scoreTextSummary(expected, actual string) float64 {
	expTokens := tokenize(expected)
	actTokens := tokenize(actual)

	// Build set of actual tokens for lookup
	actSet := make(map[string]bool)
	for _, t := range actTokens {
		actSet[t] = true
	}

	// Filter expected tokens to meaningful keywords (skip stopwords)
	keywords := filterStopwords(expTokens)
	if len(keywords) == 0 {
		return scoring.F1Score(expTokens, actTokens) // fallback
	}

	// Keyword recall: what fraction of expected keywords appear in actual
	found := 0
	for _, kw := range keywords {
		if actSet[kw] {
			found++
		}
	}
	recall := float64(found) / float64(len(keywords))

	// Also compute brevity penalty: penalize outputs that are wildly longer
	lenRatio := float64(len(actTokens)) / float64(len(expTokens))
	brevity := 1.0
	if lenRatio > 3.0 {
		brevity = 3.0 / lenRatio
	}

	return recall * brevity
}

// Common English stopwords to skip when computing keyword recall.
var stopwords = map[string]bool{
	"a": true, "an": true, "the": true, "is": true, "are": true, "was": true,
	"were": true, "be": true, "been": true, "being": true, "have": true,
	"has": true, "had": true, "do": true, "does": true, "did": true,
	"will": true, "would": true, "could": true, "should": true, "may": true,
	"might": true, "shall": true, "can": true, "to": true, "of": true,
	"in": true, "for": true, "on": true, "with": true, "at": true,
	"by": true, "from": true, "as": true, "into": true, "through": true,
	"during": true, "before": true, "after": true, "above": true, "below": true,
	"between": true, "out": true, "off": true, "over": true, "under": true,
	"again": true, "further": true, "then": true, "once": true, "and": true,
	"but": true, "or": true, "nor": true, "not": true, "so": true, "yet": true,
	"both": true, "either": true, "neither": true, "each": true, "every": true,
	"all": true, "any": true, "few": true, "more": true, "most": true,
	"other": true, "some": true, "such": true, "no": true, "only": true,
	"own": true, "same": true, "than": true, "too": true, "very": true,
	"just": true, "because": true, "if": true, "when": true, "where": true,
	"how": true, "what": true, "which": true, "who": true, "whom": true,
	"this": true, "that": true, "these": true, "those": true, "it": true,
	"its": true, "i": true, "me": true, "my": true, "we": true, "our": true,
	"you": true, "your": true, "he": true, "him": true, "his": true,
	"she": true, "her": true, "they": true, "them": true, "their": true,
}

func filterStopwords(tokens []string) []string {
	var result []string
	for _, t := range tokens {
		if !stopwords[t] && len(t) > 2 {
			result = append(result, t)
		}
	}
	return result
}

// scoreMeetingActions scores action item extraction by comparing JSON arrays.
// Handles models that wrap the array in an object (e.g. {"actionItems": [...]})
// or return a single object instead of an array.
func scoreMeetingActions(expected, actual string) float64 {
	type actionItem struct {
		Owner    string  `json:"owner"`
		Action   string  `json:"action"`
		Deadline *string `json:"deadline"`
	}

	var expItems []actionItem
	if err := json.Unmarshal([]byte(expected), &expItems); err != nil {
		return 0
	}

	actItems := parseActionItems(actual)
	if len(expItems) == 0 || len(actItems) == 0 {
		return 0
	}

	// Score by matching owners and checking action text overlap
	matched := 0
	for _, exp := range expItems {
		for _, act := range actItems {
			if scoring.ExactMatch(exp.Owner, act.Owner) {
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

type actionItem struct {
	Owner    string  `json:"owner"`
	Action   string  `json:"action"`
	Deadline *string `json:"deadline"`
}

// parseActionItems tries to extract action items from various JSON formats:
// 1. Direct array: [{"owner": ...}, ...]
// 2. Wrapper object: {"actionItems": [{"owner": ...}, ...]}
// 3. Single object: {"owner": ...}
func parseActionItems(s string) []actionItem {
	// Try direct array
	var items []actionItem
	if err := json.Unmarshal([]byte(s), &items); err == nil {
		return items
	}

	// Try wrapper object — find the first array value
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &wrapper); err == nil {
		for _, v := range wrapper {
			var arr []actionItem
			if err := json.Unmarshal(v, &arr); err == nil && len(arr) > 0 {
				return arr
			}
		}
	}

	// Try single object
	var single actionItem
	if err := json.Unmarshal([]byte(s), &single); err == nil && single.Owner != "" {
		return []actionItem{single}
	}

	return nil
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

func generateReport(scenarios []scenario) {
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
			qName := "keyword-recall"
			if strings.Contains(r.Scenario, "meeting") {
				qName = "action-match"
			}
			benchmarks = append(benchmarks, types.BenchmarkResult{
				Example:      r.Scenario,
				Model:        r.Model,
				Quality:      scoreScenario(r, scenarios),
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
