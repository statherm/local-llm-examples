package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/statherm/local-llm-examples/shared/ollama"
	"github.com/statherm/local-llm-examples/shared/reporting"
	"github.com/statherm/local-llm-examples/shared/scoring"
	"github.com/statherm/local-llm-examples/shared/types"
)

var (
	model    = flag.String("model", "qwen3:4b", "Ollama model name")
	scenario = flag.String("scenario", "all", "Scenario to run: invoices, tickets, logs, or all")
)

type scenarioConfig struct {
	Name       string
	PromptFile string
	InputDir   string
	ExpectedDir string
}

func main() {
	flag.Parse()

	scenarios := []scenarioConfig{
		{Name: "invoices", PromptFile: "prompts/invoice.txt", InputDir: "testdata/invoices", ExpectedDir: "expected/invoices"},
		{Name: "tickets", PromptFile: "prompts/support-ticket.txt", InputDir: "testdata/tickets", ExpectedDir: "expected/tickets"},
		{Name: "logs", PromptFile: "prompts/log-event.txt", InputDir: "testdata/logs", ExpectedDir: "expected/logs"},
	}

	// Filter to requested scenario.
	if *scenario != "all" {
		var filtered []scenarioConfig
		for _, s := range scenarios {
			if s.Name == *scenario {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) == 0 {
			fmt.Fprintf(os.Stderr, "unknown scenario: %s\n", *scenario)
			os.Exit(1)
		}
		scenarios = filtered
	}

	client := ollama.NewClient()
	var allResults []types.BenchmarkResult

	for _, sc := range scenarios {
		fmt.Printf("\n=== Scenario: %s (model: %s) ===\n\n", sc.Name, *model)

		results, err := runScenario(client, sc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scenario %s failed: %v\n", sc.Name, err)
			os.Exit(1)
		}
		allResults = append(allResults, results...)
	}

	fmt.Println()
	fmt.Print(reporting.GenerateReport(allResults))
}

func runScenario(client *ollama.Client, sc scenarioConfig) ([]types.BenchmarkResult, error) {
	promptTemplate, err := os.ReadFile(sc.PromptFile)
	if err != nil {
		return nil, fmt.Errorf("read prompt template: %w", err)
	}

	inputs, err := filepath.Glob(filepath.Join(sc.InputDir, "*.txt"))
	if err != nil {
		return nil, fmt.Errorf("glob inputs: %w", err)
	}
	if len(inputs) == 0 {
		return nil, fmt.Errorf("no input files found in %s", sc.InputDir)
	}

	var results []types.BenchmarkResult

	for _, inputPath := range inputs {
		name := strings.TrimSuffix(filepath.Base(inputPath), ".txt")
		expectedPath := filepath.Join(sc.ExpectedDir, name+".json")

		inputData, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, fmt.Errorf("read input %s: %w", inputPath, err)
		}

		expectedData, err := os.ReadFile(expectedPath)
		if err != nil {
			return nil, fmt.Errorf("read expected %s: %w", expectedPath, err)
		}

		prompt := strings.ReplaceAll(string(promptTemplate), "{{INPUT}}", string(inputData))

		fmt.Printf("  [%s/%s] calling model... ", sc.Name, name)

		response, meta, err := client.ChatCompletion(*model, "", prompt, true)
		if err != nil {
			return nil, fmt.Errorf("model call for %s: %w", name, err)
		}

		// Score: compare JSON fields.
		matched, total, details := scoring.JSONFieldMatch(
			json.RawMessage(expectedData),
			json.RawMessage(response),
		)

		quality := 0.0
		if total > 0 {
			quality = float64(matched) / float64(total)
		}

		fmt.Printf("score=%d/%d (%.0f%%) in %.2fs\n", matched, total, quality*100, meta.TotalTime.Seconds())

		for _, d := range details {
			status := "OK"
			if !d.Match {
				status = "MISS"
			}
			fmt.Printf("    %-20s [%s] expected=%-30s actual=%s\n", d.Field, status, d.Expected, d.Actual)
		}

		results = append(results, types.BenchmarkResult{
			Example:      fmt.Sprintf("%s/%s", sc.Name, name),
			Model:        meta.Model,
			Quality:      quality,
			QualityName:  "field_match",
			TokensIn:     meta.TokensIn,
			TokensOut:    meta.TokensOut,
			TTFT:         meta.TTFT,
			TotalTime:    meta.TotalTime,
			TokensPerSec: meta.TokensPerSec,
			CostUSD:      0,
		})
	}

	return results, nil
}
