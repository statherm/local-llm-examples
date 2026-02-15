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
	"github.com/statherm/local-llm-examples/shared/scoring"
	"github.com/statherm/local-llm-examples/shared/types"
)

// --- Data types ---

type Issue struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type IssueLabel struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Priority string `json:"priority"`
}

type Message struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type MessageLabel struct {
	ID         string `json:"id"`
	Intent     string `json:"intent"`
	Sentiment  string `json:"sentiment"`
	NeedsHuman bool   `json:"needs_human"`
}

// --- Prompt templates ---

const issueTriageSystem = `You are an issue triage classifier. Classify the given GitHub issue into exactly one category and one priority level.

Categories: bug, feature, question, docs, performance
Priorities: critical, high, medium, low

Guidelines:
- "bug": something is broken or not working as expected
- "feature": a request for new functionality
- "question": the user is asking how to do something
- "docs": documentation is missing, wrong, or unclear
- "performance": the system is slow or resource-intensive

Priority guidelines:
- "critical": data loss, security issue, complete breakage, or affects all users
- "high": significant impact, no workaround, or affects many users
- "medium": moderate impact, workaround exists
- "low": minor inconvenience, cosmetic, or affects few users

Respond with JSON only: {"category": "...", "priority": "..."}`

const intentDetectionSystem = `You are a customer support intent classifier. Classify the customer message into exactly one intent, one sentiment, and whether it needs a human agent.

Intents: billing, technical, account, cancellation, feedback, other
Sentiments: positive, neutral, negative

Guidelines for needs_human:
- true: refund requests, cancellation requests, complaints requiring action, compliance/legal
- false: general questions, positive feedback, technical issues solvable with docs, informational requests

Respond with JSON only: {"intent": "...", "sentiment": "...", "needs_human": true/false}`

func main() {
	model := flag.String("model", "qwen3:4b", "Ollama model to use")
	scenario := flag.String("scenario", "all", "Scenario to run: issues, messages, or all")
	scoreOnly := flag.Bool("score", false, "Score existing results instead of running models")
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

	if *scenario == "all" || *scenario == "issues" {
		runIssueTriage(client, *model, exampleDir)
	}
	if *scenario == "all" || *scenario == "messages" {
		runIntentDetection(client, *model, exampleDir)
	}
}

func runIssueTriage(client *ollama.Client, model, dir string) {
	issues := loadJSON[[]Issue](filepath.Join(dir, "testdata", "issues.json"))
	fmt.Printf("=== Issue Triage (%s) — %d issues ===\n", model, len(issues))

	var results []IssueLabel
	var totalTokensIn, totalTokensOut int
	var totalDuration time.Duration

	for i, issue := range issues {
		prompt := fmt.Sprintf("Title: %s\n\nBody: %s", issue.Title, issue.Body)
		resp, meta, err := client.ChatCompletion(model, issueTriageSystem, prompt, true)
		if err != nil {
			log.Printf("  [%d/%d] %s: ERROR: %v", i+1, len(issues), issue.ID, err)
			results = append(results, IssueLabel{ID: issue.ID})
			continue
		}

		var label IssueLabel
		if err := json.Unmarshal([]byte(resp), &label); err != nil {
			log.Printf("  [%d/%d] %s: JSON parse error: %v (raw: %s)", i+1, len(issues), issue.ID, err, resp)
			results = append(results, IssueLabel{ID: issue.ID})
			continue
		}
		label.ID = issue.ID
		label.Category = strings.ToLower(strings.TrimSpace(label.Category))
		label.Priority = strings.ToLower(strings.TrimSpace(label.Priority))

		totalTokensIn += meta.TokensIn
		totalTokensOut += meta.TokensOut
		totalDuration += meta.TotalTime

		fmt.Printf("  [%d/%d] %s → category=%s priority=%s (%.0fms, %.1f tok/s)\n",
			i+1, len(issues), issue.ID, label.Category, label.Priority,
			meta.TotalTime.Seconds()*1000, meta.TokensPerSec)

		results = append(results, label)
	}

	outPath := filepath.Join(dir, "results", fmt.Sprintf("issues-%s.json", sanitizeModelName(model)))
	writeJSON(outPath, results)
	fmt.Printf("  Wrote %s (%d results, %d tok in, %d tok out, %.1fs total)\n\n",
		outPath, len(results), totalTokensIn, totalTokensOut, totalDuration.Seconds())
}

func runIntentDetection(client *ollama.Client, model, dir string) {
	messages := loadJSON[[]Message](filepath.Join(dir, "testdata", "messages.json"))
	fmt.Printf("=== Intent Detection (%s) — %d messages ===\n", model, len(messages))

	var results []MessageLabel
	var totalTokensIn, totalTokensOut int
	var totalDuration time.Duration

	for i, msg := range messages {
		resp, meta, err := client.ChatCompletion(model, intentDetectionSystem, msg.Text, true)
		if err != nil {
			log.Printf("  [%d/%d] %s: ERROR: %v", i+1, len(messages), msg.ID, err)
			results = append(results, MessageLabel{ID: msg.ID})
			continue
		}

		var label MessageLabel
		if err := json.Unmarshal([]byte(resp), &label); err != nil {
			log.Printf("  [%d/%d] %s: JSON parse error: %v (raw: %s)", i+1, len(messages), msg.ID, err, resp)
			results = append(results, MessageLabel{ID: msg.ID})
			continue
		}
		label.ID = msg.ID
		label.Intent = strings.ToLower(strings.TrimSpace(label.Intent))
		label.Sentiment = strings.ToLower(strings.TrimSpace(label.Sentiment))

		totalTokensIn += meta.TokensIn
		totalTokensOut += meta.TokensOut
		totalDuration += meta.TotalTime

		fmt.Printf("  [%d/%d] %s → intent=%s sentiment=%s needs_human=%v (%.0fms, %.1f tok/s)\n",
			i+1, len(messages), msg.ID, label.Intent, label.Sentiment, label.NeedsHuman,
			meta.TotalTime.Seconds()*1000, meta.TokensPerSec)

		results = append(results, label)
	}

	outPath := filepath.Join(dir, "results", fmt.Sprintf("messages-%s.json", sanitizeModelName(model)))
	writeJSON(outPath, results)
	fmt.Printf("  Wrote %s (%d results, %d tok in, %d tok out, %.1fs total)\n\n",
		outPath, len(results), totalTokensIn, totalTokensOut, totalDuration.Seconds())
}

func scoreResults(dir, scenario string) {
	if scenario == "all" || scenario == "issues" {
		scoreIssues(dir)
	}
	if scenario == "all" || scenario == "messages" {
		scoreMessages(dir)
	}
}

func scoreIssues(dir string) {
	expected := loadJSON[[]IssueLabel](filepath.Join(dir, "expected", "issues.json"))
	expectedMap := make(map[string]IssueLabel)
	for _, e := range expected {
		expectedMap[e.ID] = e
	}

	resultFiles, _ := filepath.Glob(filepath.Join(dir, "results", "issues-*.json"))
	for _, rf := range resultFiles {
		actual := loadJSON[[]IssueLabel](rf)
		modelName := strings.TrimPrefix(filepath.Base(rf), "issues-")
		modelName = strings.TrimSuffix(modelName, ".json")

		var catPred, catLabel, priPred, priLabel []string
		for _, a := range actual {
			if e, ok := expectedMap[a.ID]; ok {
				catPred = append(catPred, a.Category)
				catLabel = append(catLabel, e.Category)
				priPred = append(priPred, a.Priority)
				priLabel = append(priLabel, e.Priority)
			}
		}

		catAcc, _ := scoring.AccuracyScore(catPred, catLabel)
		priAcc, _ := scoring.AccuracyScore(priPred, priLabel)

		fmt.Printf("=== Issue Triage Scores: %s ===\n", modelName)
		fmt.Printf("  Category accuracy: %.1f%% (%d/%d)\n", catAcc*100, countMatches(catPred, catLabel), len(catLabel))
		fmt.Printf("  Priority accuracy: %.1f%% (%d/%d)\n", priAcc*100, countMatches(priPred, priLabel), len(priLabel))
		fmt.Printf("  Combined accuracy: %.1f%%\n\n", combinedAccuracy(catPred, catLabel, priPred, priLabel)*100)
	}
}

func scoreMessages(dir string) {
	expected := loadJSON[[]MessageLabel](filepath.Join(dir, "expected", "messages.json"))
	expectedMap := make(map[string]MessageLabel)
	for _, e := range expected {
		expectedMap[e.ID] = e
	}

	resultFiles, _ := filepath.Glob(filepath.Join(dir, "results", "messages-*.json"))
	for _, rf := range resultFiles {
		actual := loadJSON[[]MessageLabel](rf)
		modelName := strings.TrimPrefix(filepath.Base(rf), "messages-")
		modelName = strings.TrimSuffix(modelName, ".json")

		var intentPred, intentLabel, sentPred, sentLabel []string
		var humanPred, humanLabel []string
		for _, a := range actual {
			if e, ok := expectedMap[a.ID]; ok {
				intentPred = append(intentPred, a.Intent)
				intentLabel = append(intentLabel, e.Intent)
				sentPred = append(sentPred, a.Sentiment)
				sentLabel = append(sentLabel, e.Sentiment)
				humanPred = append(humanPred, fmt.Sprintf("%v", a.NeedsHuman))
				humanLabel = append(humanLabel, fmt.Sprintf("%v", e.NeedsHuman))
			}
		}

		intentAcc, _ := scoring.AccuracyScore(intentPred, intentLabel)
		sentAcc, _ := scoring.AccuracyScore(sentPred, sentLabel)
		humanAcc, _ := scoring.AccuracyScore(humanPred, humanLabel)

		fmt.Printf("=== Intent Detection Scores: %s ===\n", modelName)
		fmt.Printf("  Intent accuracy:     %.1f%% (%d/%d)\n", intentAcc*100, countMatches(intentPred, intentLabel), len(intentLabel))
		fmt.Printf("  Sentiment accuracy:  %.1f%% (%d/%d)\n", sentAcc*100, countMatches(sentPred, sentLabel), len(sentLabel))
		fmt.Printf("  Needs-human accuracy: %.1f%% (%d/%d)\n\n", humanAcc*100, countMatches(humanPred, humanLabel), len(humanLabel))
	}
}

func generateReport(dir string) {
	// Collect all result files and build benchmark results
	var results []types.BenchmarkResult

	issueFiles, _ := filepath.Glob(filepath.Join(dir, "results", "issues-*.json"))
	for _, rf := range issueFiles {
		modelName := strings.TrimPrefix(filepath.Base(rf), "issues-")
		modelName = strings.TrimSuffix(modelName, ".json")

		expected := loadJSON[[]IssueLabel](filepath.Join(dir, "expected", "issues.json"))
		actual := loadJSON[[]IssueLabel](rf)

		expectedMap := make(map[string]IssueLabel)
		for _, e := range expected {
			expectedMap[e.ID] = e
		}
		var catPred, catLabel, priPred, priLabel []string
		for _, a := range actual {
			if e, ok := expectedMap[a.ID]; ok {
				catPred = append(catPred, a.Category)
				catLabel = append(catLabel, e.Category)
				priPred = append(priPred, a.Priority)
				priLabel = append(priLabel, e.Priority)
			}
		}
		combined := combinedAccuracy(catPred, catLabel, priPred, priLabel)
		results = append(results, types.BenchmarkResult{
			Example:     "Issue Triage",
			Model:       modelName,
			Quality:     combined,
			QualityName: "Combined Acc",
		})
	}

	msgFiles, _ := filepath.Glob(filepath.Join(dir, "results", "messages-*.json"))
	for _, rf := range msgFiles {
		modelName := strings.TrimPrefix(filepath.Base(rf), "messages-")
		modelName = strings.TrimSuffix(modelName, ".json")

		expected := loadJSON[[]MessageLabel](filepath.Join(dir, "expected", "messages.json"))
		actual := loadJSON[[]MessageLabel](rf)

		expectedMap := make(map[string]MessageLabel)
		for _, e := range expected {
			expectedMap[e.ID] = e
		}
		var intentPred, intentLabel []string
		for _, a := range actual {
			if e, ok := expectedMap[a.ID]; ok {
				intentPred = append(intentPred, a.Intent)
				intentLabel = append(intentLabel, e.Intent)
			}
		}
		intentAcc, _ := scoring.AccuracyScore(intentPred, intentLabel)
		results = append(results, types.BenchmarkResult{
			Example:     "Intent Detection",
			Model:       modelName,
			Quality:     intentAcc,
			QualityName: "Intent Acc",
		})
	}

	report := reporting.GenerateReport(results)
	fmt.Print(report)
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

func countMatches(pred, label []string) int {
	n := 0
	for i := range pred {
		if scoring.ExactMatch(pred[i], label[i]) {
			n++
		}
	}
	return n
}

func combinedAccuracy(catPred, catLabel, priPred, priLabel []string) float64 {
	if len(catPred) == 0 {
		return 0
	}
	n := 0
	for i := range catPred {
		if scoring.ExactMatch(catPred[i], catLabel[i]) && scoring.ExactMatch(priPred[i], priLabel[i]) {
			n++
		}
	}
	return float64(n) / float64(len(catPred))
}
