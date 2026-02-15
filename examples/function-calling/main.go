package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/statherm/local-llm-examples/shared/ollama"
	"github.com/statherm/local-llm-examples/shared/reporting"
	"github.com/statherm/local-llm-examples/shared/types"
)

// --- Data types ---

type ToolDef struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Parameters  map[string]ToolParam         `json:"parameters"`
	Required    []string                     `json:"required"`
}

type ToolParam struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type TestCase struct {
	ID      string `json:"id"`
	Request string `json:"request"`
}

type ExpectedCall struct {
	ID         string         `json:"id"`
	Tool       string         `json:"tool"`
	Parameters map[string]any `json:"parameters"`
}

type ActualCall struct {
	ID         string         `json:"id"`
	Tool       string         `json:"tool"`
	Parameters map[string]any `json:"parameters"`
	RawOutput  string         `json:"raw_output,omitempty"`
}

// --- System prompt builder ---

func buildSystemPrompt(tools []ToolDef) string {
	var sb strings.Builder
	sb.WriteString("You are a function calling assistant. Given a user request, select the most appropriate tool and provide the correct parameters.\n\n")
	sb.WriteString("Available tools:\n\n")

	for _, t := range tools {
		sb.WriteString(fmt.Sprintf("### %s\n%s\n", t.Name, t.Description))
		if len(t.Parameters) > 0 {
			sb.WriteString("Parameters:\n")
			for name, param := range t.Parameters {
				req := ""
				for _, r := range t.Required {
					if r == name {
						req = " (required)"
						break
					}
				}
				sb.WriteString(fmt.Sprintf("  - %s (%s): %s%s\n", name, param.Type, param.Description, req))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Respond with JSON only: {\"tool\": \"tool_name\", \"parameters\": {...}}\n")
	sb.WriteString("Only include parameters that are relevant to the request. Use the exact tool names shown above.")

	return sb.String()
}

func main() {
	model := flag.String("model", "qwen3:4b", "Ollama model to use")
	scenario := flag.String("scenario", "all", "Scenario: developer, home, or all")
	scoreOnly := flag.Bool("score", false, "Score existing results")
	reportOnly := flag.Bool("report", false, "Generate report from existing results")
	flag.Parse()

	exampleDir := filepath.Dir(os.Args[0])
	if abs, err := filepath.Abs("."); err == nil {
		exampleDir = abs
	}

	if *scoreOnly {
		scoreResults(exampleDir, *scenario)
		return
	}
	if *reportOnly {
		generateReport(exampleDir)
		return
	}

	client := ollama.NewClient()

	if *scenario == "all" || *scenario == "developer" {
		runScenario(client, *model, exampleDir, "developer")
	}
	if *scenario == "all" || *scenario == "home" {
		runScenario(client, *model, exampleDir, "home-automation")
	}
}

func runScenario(client *ollama.Client, model, dir, scenario string) {
	tools := loadJSON[[]ToolDef](filepath.Join(dir, "tools", scenario+".json"))
	cases := loadJSON[[]TestCase](filepath.Join(dir, "testdata", scenario+".json"))
	systemPrompt := buildSystemPrompt(tools)

	fmt.Printf("=== Function Calling: %s (%s) — %d requests ===\n", scenario, model, len(cases))

	var results []ActualCall
	var totalTokensIn, totalTokensOut int
	var totalDuration time.Duration

	for i, tc := range cases {
		resp, meta, err := client.ChatCompletion(model, systemPrompt, tc.Request, true)
		if err != nil {
			log.Printf("  [%d/%d] %s: ERROR: %v", i+1, len(cases), tc.ID, err)
			results = append(results, ActualCall{ID: tc.ID, RawOutput: err.Error()})
			continue
		}

		var call ActualCall
		if err := json.Unmarshal([]byte(resp), &call); err != nil {
			log.Printf("  [%d/%d] %s: JSON parse error: %v (raw: %s)", i+1, len(cases), tc.ID, err, resp)
			results = append(results, ActualCall{ID: tc.ID, RawOutput: resp})
			continue
		}
		call.ID = tc.ID

		totalTokensIn += meta.TokensIn
		totalTokensOut += meta.TokensOut
		totalDuration += meta.TotalTime

		paramStr, _ := json.Marshal(call.Parameters)
		fmt.Printf("  [%d/%d] %s → %s(%s) (%.0fms, %.1f tok/s)\n",
			i+1, len(cases), tc.ID, call.Tool, string(paramStr),
			meta.TotalTime.Seconds()*1000, meta.TokensPerSec)

		results = append(results, call)
	}

	outPath := filepath.Join(dir, "results", fmt.Sprintf("%s-%s.json", scenario, sanitizeModelName(model)))
	writeJSON(outPath, results)
	fmt.Printf("  Wrote %s (%d results, %d tok in, %d tok out, %.1fs total)\n\n",
		outPath, len(results), totalTokensIn, totalTokensOut, totalDuration.Seconds())
}

func scoreResults(dir, scenario string) {
	if scenario == "all" || scenario == "developer" {
		scoreScenario(dir, "developer")
	}
	if scenario == "all" || scenario == "home" {
		scoreScenario(dir, "home-automation")
	}
}

func scoreScenario(dir, scenario string) {
	expected := loadJSON[[]ExpectedCall](filepath.Join(dir, "expected", scenario+".json"))
	expectedMap := make(map[string]ExpectedCall)
	for _, e := range expected {
		expectedMap[e.ID] = e
	}

	resultFiles, _ := filepath.Glob(filepath.Join(dir, "results", scenario+"-*.json"))
	for _, rf := range resultFiles {
		actual := loadJSON[[]ActualCall](rf)
		modelName := strings.TrimPrefix(filepath.Base(rf), scenario+"-")
		modelName = strings.TrimSuffix(modelName, ".json")

		var toolCorrect, paramCorrect, bothCorrect, total int

		for _, a := range actual {
			e, ok := expectedMap[a.ID]
			if !ok {
				continue
			}
			total++

			toolMatch := strings.EqualFold(strings.TrimSpace(a.Tool), strings.TrimSpace(e.Tool))
			if toolMatch {
				toolCorrect++
			}

			pMatch := parametersMatch(e.Parameters, a.Parameters)
			if pMatch {
				paramCorrect++
			}

			if toolMatch && pMatch {
				bothCorrect++
			}
		}

		fmt.Printf("=== Function Calling Scores: %s / %s ===\n", scenario, modelName)
		fmt.Printf("  Tool selection:  %.1f%% (%d/%d)\n", pct(toolCorrect, total), toolCorrect, total)
		fmt.Printf("  Parameters:      %.1f%% (%d/%d)\n", pct(paramCorrect, total), paramCorrect, total)
		fmt.Printf("  Combined:        %.1f%% (%d/%d)\n\n", pct(bothCorrect, total), bothCorrect, total)
	}
}

func generateReport(dir string) {
	var results []types.BenchmarkResult

	for _, scenario := range []string{"developer", "home-automation"} {
		expected := loadJSON[[]ExpectedCall](filepath.Join(dir, "expected", scenario+".json"))
		expectedMap := make(map[string]ExpectedCall)
		for _, e := range expected {
			expectedMap[e.ID] = e
		}

		resultFiles, _ := filepath.Glob(filepath.Join(dir, "results", scenario+"-*.json"))
		for _, rf := range resultFiles {
			actual := loadJSON[[]ActualCall](rf)
			modelName := strings.TrimPrefix(filepath.Base(rf), scenario+"-")
			modelName = strings.TrimSuffix(modelName, ".json")

			var toolCorrect, total int
			for _, a := range actual {
				if e, ok := expectedMap[a.ID]; ok {
					total++
					if strings.EqualFold(strings.TrimSpace(a.Tool), strings.TrimSpace(e.Tool)) {
						toolCorrect++
					}
				}
			}

			quality := 0.0
			if total > 0 {
				quality = float64(toolCorrect) / float64(total)
			}

			results = append(results, types.BenchmarkResult{
				Example:     fmt.Sprintf("Function Calling (%s)", scenario),
				Model:       modelName,
				Quality:     quality,
				QualityName: "Tool Acc",
			})
		}
	}

	report := reporting.GenerateReport(results)
	fmt.Print(report)
}

// parametersMatch checks whether actual parameters satisfy the expected ones.
// It only checks keys present in expected -- extra keys in actual are ignored.
// Empty expected parameters means only the tool name matters.
func parametersMatch(expected, actual map[string]any) bool {
	if len(expected) == 0 {
		return true
	}
	for key, expVal := range expected {
		actVal, ok := actual[key]
		if !ok {
			return false
		}
		if !valuesMatch(expVal, actVal) {
			return false
		}
	}
	return true
}

func valuesMatch(expected, actual any) bool {
	// Normalize numbers: JSON unmarshals all numbers as float64
	expStr := fmt.Sprintf("%v", expected)
	actStr := fmt.Sprintf("%v", actual)
	return strings.EqualFold(strings.TrimSpace(expStr), strings.TrimSpace(actStr))
}

// --- Helpers ---

func loadJSON[T any](path string) T {
	var v T
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, &v); err != nil {
		log.Fatalf("Failed to parse %s: %v", path, err)
	}
	return v
}

func writeJSON(path string, v any) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Fatalf("Failed to create directory for %s: %v", path, err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Fatalf("Failed to write %s: %v", path, err)
	}
}

func sanitizeModelName(model string) string {
	r := strings.NewReplacer("/", "-", ":", "-", " ", "-")
	return r.Replace(model)
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}
