package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/statherm/local-llm-examples/shared/ollama"
	"github.com/statherm/local-llm-examples/shared/reporting"
	"github.com/statherm/local-llm-examples/shared/types"
)

// SearchCandidate is a single search result to be reranked.
type SearchCandidate struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

// SearchQuery is the input: a query plus candidate results.
type SearchQuery struct {
	Query      string            `json:"query"`
	Candidates []SearchCandidate `json:"candidates"`
}

// GoldRanking is the expected relevance for a candidate.
type GoldRanking struct {
	ID        string `json:"id"`
	Relevance int    `json:"relevance"`
	Reason    string `json:"reason"`
}

// GoldStandard is the full ground truth for a query.
type GoldStandard struct {
	Query   string        `json:"query"`
	Ranking []GoldRanking `json:"ranking"`
}

// RankedResult is the model's output: a candidate ID with a relevance score.
type RankedResult struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

// RerankedOutput is the model's full response parsed from JSON.
type RerankedOutput struct {
	Rankings []RankedResult `json:"rankings"`
}

// ScenarioResult stores the outcome for one test scenario.
type ScenarioResult struct {
	Scenario string           `json:"scenario"`
	Model    string           `json:"model"`
	NDCG     float64          `json:"ndcg"`
	MRR      float64          `json:"mrr"`
	Rankings []RankedResult   `json:"rankings"`
	Meta     types.ModelMetadata `json:"metadata"`
}

const systemPrompt = `You are a search result reranking system. Given a search query and a list of candidate results, score each result's relevance to the query.

For each candidate, assign a relevance score from 0.0 to 1.0:
- 1.0 = directly and completely answers the query
- 0.7-0.9 = highly relevant, addresses the core topic
- 0.4-0.6 = somewhat relevant, related topic but not a direct answer
- 0.1-0.3 = tangentially related, shares some keywords
- 0.0 = completely irrelevant

Respond with valid JSON in this exact format:
{"rankings": [{"id": "<candidate-id>", "score": <0.0-1.0>}, ...]}`

func buildPrompt(query SearchQuery) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Query: %s\n\nCandidate results:\n\n", query.Query))
	for i, c := range query.Candidates {
		sb.WriteString(fmt.Sprintf("[%d] ID: %s\nTitle: %s\nSnippet: %s\n\n", i+1, c.ID, c.Title, c.Snippet))
	}
	sb.WriteString("Score each candidate's relevance to the query. Return JSON with all candidate IDs and their scores.")
	return sb.String()
}

// ndcg computes Normalized Discounted Cumulative Gain.
// modelRanking is the ordered list of IDs from the model.
// goldRelevance maps each ID to its gold-standard relevance grade.
func ndcg(modelRanking []string, goldRelevance map[string]int, k int) float64 {
	if k <= 0 || len(modelRanking) == 0 {
		return 0
	}
	if k > len(modelRanking) {
		k = len(modelRanking)
	}

	// DCG of model ranking
	dcg := 0.0
	for i := 0; i < k; i++ {
		rel := float64(goldRelevance[modelRanking[i]])
		dcg += (math.Pow(2, rel) - 1) / math.Log2(float64(i+2))
	}

	// Ideal DCG: sort by gold relevance descending
	idealOrder := make([]int, 0, len(goldRelevance))
	for _, rel := range goldRelevance {
		idealOrder = append(idealOrder, rel)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(idealOrder)))

	idcg := 0.0
	for i := 0; i < k && i < len(idealOrder); i++ {
		rel := float64(idealOrder[i])
		idcg += (math.Pow(2, rel) - 1) / math.Log2(float64(i+2))
	}

	if idcg == 0 {
		return 0
	}
	return dcg / idcg
}

// mrr computes Mean Reciprocal Rank. It finds the rank of the first
// highly relevant result (relevance >= threshold) in the model's ranking.
func mrr(modelRanking []string, goldRelevance map[string]int, threshold int) float64 {
	for i, id := range modelRanking {
		if goldRelevance[id] >= threshold {
			return 1.0 / float64(i+1)
		}
	}
	return 0
}

func loadJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return json.Unmarshal(data, v)
}

func main() {
	model := flag.String("model", "qwen3:4b", "Ollama model to use")
	doScore := flag.Bool("score", false, "Score existing results against gold standard")
	doReport := flag.Bool("report", false, "Generate benchmark report from results")
	flag.Parse()

	exampleDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	scenarios := []struct {
		name     string
		input    string
		gold     string
	}{
		{"doc_search", "testdata/doc_search.json", "baseline/doc_search_gold.json"},
		{"code_search", "testdata/code_search.json", "baseline/code_search_gold.json"},
		{"api_search", "testdata/api_search.json", "baseline/api_search_gold.json"},
	}

	if *doScore {
		scoreResults(exampleDir, scenarios)
		return
	}

	if *doReport {
		generateReport(exampleDir)
		return
	}

	client := ollama.NewClient()

	for _, sc := range scenarios {
		fmt.Printf("=== Scenario: %s (model: %s) ===\n", sc.name, *model)

		var query SearchQuery
		if err := loadJSON(filepath.Join(exampleDir, sc.input), &query); err != nil {
			log.Fatalf("load input: %v", err)
		}

		prompt := buildPrompt(query)
		response, meta, err := client.ChatCompletion(*model, systemPrompt, prompt, true)
		if err != nil {
			log.Fatalf("ollama: %v", err)
		}

		var output RerankedOutput
		if err := json.Unmarshal([]byte(response), &output); err != nil {
			log.Printf("WARNING: failed to parse model output as JSON: %v", err)
			log.Printf("Raw response: %s", response)
			continue
		}

		// Sort by score descending
		sort.Slice(output.Rankings, func(i, j int) bool {
			return output.Rankings[i].Score > output.Rankings[j].Score
		})

		result := ScenarioResult{
			Scenario: sc.name,
			Model:    *model,
			Rankings: output.Rankings,
			Meta:     meta,
		}

		// Score against gold standard
		var gold GoldStandard
		if err := loadJSON(filepath.Join(exampleDir, sc.gold), &gold); err != nil {
			log.Printf("WARNING: could not load gold standard: %v", err)
		} else {
			goldRel := make(map[string]int)
			for _, g := range gold.Ranking {
				goldRel[g.ID] = g.Relevance
			}
			modelOrder := make([]string, len(output.Rankings))
			for i, r := range output.Rankings {
				modelOrder[i] = r.ID
			}
			result.NDCG = ndcg(modelOrder, goldRel, 10)
			result.MRR = mrr(modelOrder, goldRel, 3)
		}

		fmt.Printf("  NDCG@10: %.3f\n", result.NDCG)
		fmt.Printf("  MRR:     %.3f\n", result.MRR)
		fmt.Printf("  Tokens:  %d in / %d out (%.1f tok/s)\n", meta.TokensIn, meta.TokensOut, meta.TokensPerSec)
		fmt.Printf("  Latency: %s (TTFT: %s)\n", meta.TotalTime, meta.TTFT)
		fmt.Println("  Top 5 results:")
		for i := 0; i < 5 && i < len(output.Rankings); i++ {
			r := output.Rankings[i]
			fmt.Printf("    %d. %s (score: %.2f)\n", i+1, r.ID, r.Score)
		}
		fmt.Println()

		// Save result
		resultPath := filepath.Join(exampleDir, "results", fmt.Sprintf("%s_%s.json", sc.name, sanitizeModelName(*model)))
		resultData, _ := json.MarshalIndent(result, "", "  ")
		if err := os.WriteFile(resultPath, resultData, 0644); err != nil {
			log.Printf("WARNING: could not write result: %v", err)
		}
	}
}

func sanitizeModelName(name string) string {
	r := strings.NewReplacer("/", "_", ":", "_", ".", "_")
	return r.Replace(name)
}

func scoreResults(exampleDir string, scenarios []struct {
	name  string
	input string
	gold  string
}) {
	entries, err := os.ReadDir(filepath.Join(exampleDir, "results"))
	if err != nil {
		log.Fatalf("read results dir: %v", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		var result ScenarioResult
		if err := loadJSON(filepath.Join(exampleDir, "results", entry.Name()), &result); err != nil {
			log.Printf("skip %s: %v", entry.Name(), err)
			continue
		}

		// Find matching gold standard
		var goldPath string
		for _, sc := range scenarios {
			if sc.name == result.Scenario {
				goldPath = filepath.Join(exampleDir, sc.gold)
				break
			}
		}
		if goldPath == "" {
			log.Printf("skip %s: no matching scenario", entry.Name())
			continue
		}

		var gold GoldStandard
		if err := loadJSON(goldPath, &gold); err != nil {
			log.Printf("skip %s: %v", entry.Name(), err)
			continue
		}

		goldRel := make(map[string]int)
		for _, g := range gold.Ranking {
			goldRel[g.ID] = g.Relevance
		}
		modelOrder := make([]string, len(result.Rankings))
		for i, r := range result.Rankings {
			modelOrder[i] = r.ID
		}

		ndcgScore := ndcg(modelOrder, goldRel, 10)
		mrrScore := mrr(modelOrder, goldRel, 3)

		fmt.Printf("%s: NDCG@10=%.3f  MRR=%.3f\n", entry.Name(), ndcgScore, mrrScore)
	}
}

func generateReport(exampleDir string) {
	entries, err := os.ReadDir(filepath.Join(exampleDir, "results"))
	if err != nil {
		log.Fatalf("read results dir: %v", err)
	}

	var benchmarks []types.BenchmarkResult
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		var result ScenarioResult
		if err := loadJSON(filepath.Join(exampleDir, "results", entry.Name()), &result); err != nil {
			continue
		}

		benchmarks = append(benchmarks, types.BenchmarkResult{
			Example:      fmt.Sprintf("search-reranking/%s", result.Scenario),
			Model:        result.Model,
			Quality:      result.NDCG,
			QualityName:  "NDCG@10",
			TokensIn:     result.Meta.TokensIn,
			TokensOut:    result.Meta.TokensOut,
			TTFT:         result.Meta.TTFT,
			TotalTime:    result.Meta.TotalTime,
			TokensPerSec: result.Meta.TokensPerSec,
			CostUSD:      0,
		})
	}

	report := reporting.GenerateReport(benchmarks)
	fmt.Print(report)

	reportPath := filepath.Join(exampleDir, "RESULTS.md")
	if err := os.WriteFile(reportPath, []byte("# Search Reranking Results\n\n"+report), 0644); err != nil {
		log.Printf("WARNING: could not write report: %v", err)
	}
}
