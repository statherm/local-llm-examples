package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/statherm/local-llm-examples/shared/ollama"
	"github.com/statherm/local-llm-examples/shared/reporting"
	"github.com/statherm/local-llm-examples/shared/types"
)

// Schema describes the structure of data to generate.
type Schema struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Count       int           `json:"count"`
	Fields      []FieldDef    `json:"fields"`
	Example     json.RawMessage `json:"example"`
}

// FieldDef describes a single field in the schema.
type FieldDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Constraints defines validation rules for generated data.
type Constraints struct {
	Schema             string           `json:"schema"`
	Rules              []Rule           `json:"rules"`
	DistributionChecks []DistCheck      `json:"distribution_checks"`
	CrossFieldRules    []CrossFieldRule `json:"cross_field_rules"`
}

// Rule is a single validation rule for a field.
type Rule struct {
	Field        string   `json:"field"`
	RuleType     string   `json:"rule"`
	Min          *float64 `json:"min,omitempty"`
	Max          *float64 `json:"max,omitempty"`
	Exact        *int     `json:"exact,omitempty"`
	Regex        string   `json:"regex,omitempty"`
	Values       []string `json:"values,omitempty"`
	ExpectedType string   `json:"expected_type,omitempty"`
}

// DistCheck describes a distribution check.
type DistCheck struct {
	Field       string   `json:"field"`
	Check       string   `json:"check"`
	Values      []string `json:"values,omitempty"`
	Value       *bool    `json:"value,omitempty"`
	TargetRatio float64  `json:"target_ratio,omitempty"`
	Tolerance   float64  `json:"tolerance,omitempty"`
	Min         int      `json:"min,omitempty"`
}

// CrossFieldRule defines a relationship between fields.
type CrossFieldRule struct {
	RuleType  string      `json:"rule"`
	IfField   string      `json:"if_field"`
	IfValue   interface{} `json:"if_value"`
	ThenField string      `json:"then_field"`
	ThenValue interface{} `json:"then_value"`
}

// ScenarioResult stores the output for one schema.
type ScenarioResult struct {
	Schema     string                   `json:"schema"`
	Model      string                   `json:"model"`
	Records    []map[string]interface{} `json:"records"`
	Score      ScoreDetail              `json:"score"`
	Meta       types.ModelMetadata      `json:"metadata"`
}

// ScoreDetail breaks down the compliance score.
type ScoreDetail struct {
	SchemaCompliance float64 `json:"schema_compliance"`
	RuleCompliance   float64 `json:"rule_compliance"`
	Uniqueness       float64 `json:"uniqueness"`
	Overall          float64 `json:"overall"`
	Violations       []string `json:"violations,omitempty"`
}

const systemPrompt = `You are a test data generator. Given a schema definition with field types and constraints, generate realistic synthetic data records.

Requirements:
- Generate exactly the number of records requested
- Every field in the schema must be present in every record
- Values must be realistic and diverse — not placeholder text like "John Doe" repeated
- Follow all type constraints and value descriptions exactly
- IDs must be unique across records
- Dates must be valid ISO 8601 format

Respond with valid JSON in this exact format:
{"records": [<array of objects matching the schema>]}`

func buildPrompt(schema Schema) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Generate %d realistic %s records.\n\n", schema.Count, schema.Name))
	sb.WriteString(fmt.Sprintf("Description: %s\n\n", schema.Description))
	sb.WriteString("Schema:\n")
	for _, f := range schema.Fields {
		sb.WriteString(fmt.Sprintf("  - %s (%s): %s\n", f.Name, f.Type, f.Description))
	}
	sb.WriteString(fmt.Sprintf("\nExample record:\n%s\n", string(schema.Example)))
	sb.WriteString(fmt.Sprintf("\nGenerate exactly %d records. Each record must have all fields. Return JSON.", schema.Count))
	return sb.String()
}

// validateRecords checks generated records against schema and constraints.
func validateRecords(records []map[string]interface{}, schema Schema, constraints Constraints) ScoreDetail {
	var violations []string
	totalFieldChecks := 0
	passedFieldChecks := 0
	totalRuleChecks := 0
	passedRuleChecks := 0

	// Check record count
	if len(records) != schema.Count {
		violations = append(violations, fmt.Sprintf("expected %d records, got %d", schema.Count, len(records)))
	}

	// Schema compliance: every field must exist with correct type
	for i, rec := range records {
		for _, field := range schema.Fields {
			totalFieldChecks++
			val, ok := rec[field.Name]
			if !ok {
				violations = append(violations, fmt.Sprintf("record %d: missing field %q", i, field.Name))
				continue
			}
			if val == nil {
				violations = append(violations, fmt.Sprintf("record %d: field %q is null", i, field.Name))
				continue
			}

			// Basic type check
			typeOk := checkType(val, field.Type)
			if !typeOk {
				violations = append(violations, fmt.Sprintf("record %d: field %q has wrong type (expected %s)", i, field.Name, field.Type))
				continue
			}
			passedFieldChecks++
		}
	}

	// Rule compliance
	for _, rule := range constraints.Rules {
		for i, rec := range records {
			val, ok := rec[rule.Field]
			if !ok {
				continue
			}
			totalRuleChecks++
			if checkRule(val, rule) {
				passedRuleChecks++
			} else {
				violations = append(violations, fmt.Sprintf("record %d: field %q violates rule %q", i, rule.Field, rule.RuleType))
			}
		}
	}

	// Uniqueness check
	uniqueFields := make(map[string]bool)
	for _, rule := range constraints.Rules {
		if rule.RuleType == "unique" {
			uniqueFields[rule.Field] = true
		}
	}

	uniqueScore := 1.0
	for field := range uniqueFields {
		seen := make(map[string]bool)
		dupes := 0
		for _, rec := range records {
			val, ok := rec[field]
			if !ok {
				continue
			}
			key := fmt.Sprintf("%v", val)
			if seen[key] {
				dupes++
			}
			seen[key] = true
		}
		if len(records) > 0 && dupes > 0 {
			uniqueScore *= 1.0 - float64(dupes)/float64(len(records))
		}
	}

	schemaCompliance := 0.0
	if totalFieldChecks > 0 {
		schemaCompliance = float64(passedFieldChecks) / float64(totalFieldChecks)
	}
	ruleCompliance := 0.0
	if totalRuleChecks > 0 {
		ruleCompliance = float64(passedRuleChecks) / float64(totalRuleChecks)
	}

	overall := (schemaCompliance*0.4 + ruleCompliance*0.4 + uniqueScore*0.2)

	return ScoreDetail{
		SchemaCompliance: schemaCompliance,
		RuleCompliance:   ruleCompliance,
		Uniqueness:       uniqueScore,
		Overall:          overall,
		Violations:       violations,
	}
}

func checkType(val interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := val.(string)
		return ok
	case "integer":
		f, ok := val.(float64)
		if !ok {
			return false
		}
		return f == float64(int64(f))
	case "number":
		_, ok := val.(float64)
		return ok
	case "boolean":
		_, ok := val.(bool)
		return ok
	case "array":
		_, ok := val.([]interface{})
		return ok
	default:
		return true
	}
}

func checkRule(val interface{}, rule Rule) bool {
	switch rule.RuleType {
	case "range":
		f, ok := toFloat(val)
		if !ok {
			return false
		}
		if rule.Min != nil && f < *rule.Min {
			return false
		}
		if rule.Max != nil && f > *rule.Max {
			return false
		}
		return true

	case "length":
		s, ok := val.(string)
		if !ok {
			return false
		}
		if rule.Exact != nil && len(s) != *rule.Exact {
			return false
		}
		return true

	case "pattern":
		s, ok := val.(string)
		if !ok {
			return false
		}
		matched, err := regexp.MatchString(rule.Regex, s)
		return err == nil && matched

	case "enum":
		s := fmt.Sprintf("%v", val)
		for _, v := range rule.Values {
			if strings.EqualFold(s, v) {
				return true
			}
		}
		return false

	case "type":
		return checkType(val, rule.ExpectedType)

	case "array_length":
		arr, ok := val.([]interface{})
		if !ok {
			return false
		}
		if rule.Min != nil && len(arr) < int(*rule.Min) {
			return false
		}
		if rule.Max != nil && len(arr) > int(*rule.Max) {
			return false
		}
		return true

	case "unique":
		// Handled separately in validateRecords
		return true

	case "date_range":
		// Just check that it's a valid-looking date string
		s, ok := val.(string)
		if !ok {
			return false
		}
		matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, s)
		return matched

	default:
		return true
	}
}

func toFloat(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
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
	doScore := flag.Bool("score", false, "Score existing results against constraints")
	doReport := flag.Bool("report", false, "Generate benchmark report from results")
	flag.Parse()

	exampleDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	scenarios := []struct {
		name       string
		schema     string
		constraint string
	}{
		{"user_profiles", "schemas/user_profiles.json", "constraints/user_profiles.json"},
		{"transactions", "schemas/transactions.json", "constraints/transactions.json"},
		{"api_responses", "schemas/api_responses.json", "constraints/api_responses.json"},
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

		var schema Schema
		if err := loadJSON(filepath.Join(exampleDir, sc.schema), &schema); err != nil {
			log.Fatalf("load schema: %v", err)
		}

		var constraints Constraints
		if err := loadJSON(filepath.Join(exampleDir, sc.constraint), &constraints); err != nil {
			log.Fatalf("load constraints: %v", err)
		}

		prompt := buildPrompt(schema)
		response, meta, err := client.ChatCompletion(*model, systemPrompt, prompt, true)
		if err != nil {
			log.Fatalf("ollama: %v", err)
		}

		var output struct {
			Records []map[string]interface{} `json:"records"`
		}
		if err := json.Unmarshal([]byte(response), &output); err != nil {
			log.Printf("WARNING: failed to parse model output as JSON: %v", err)
			log.Printf("Raw response: %s", response)
			continue
		}

		score := validateRecords(output.Records, schema, constraints)

		result := ScenarioResult{
			Schema:  sc.name,
			Model:   *model,
			Records: output.Records,
			Score:   score,
			Meta:    meta,
		}

		fmt.Printf("  Records generated: %d / %d\n", len(output.Records), schema.Count)
		fmt.Printf("  Schema compliance: %.1f%%\n", score.SchemaCompliance*100)
		fmt.Printf("  Rule compliance:   %.1f%%\n", score.RuleCompliance*100)
		fmt.Printf("  Uniqueness:        %.1f%%\n", score.Uniqueness*100)
		fmt.Printf("  Overall score:     %.1f%%\n", score.Overall*100)
		fmt.Printf("  Tokens: %d in / %d out (%.1f tok/s)\n", meta.TokensIn, meta.TokensOut, meta.TokensPerSec)
		fmt.Printf("  Latency: %s (TTFT: %s)\n", meta.TotalTime, meta.TTFT)
		if len(score.Violations) > 0 {
			fmt.Printf("  Violations (%d):\n", len(score.Violations))
			limit := len(score.Violations)
			if limit > 5 {
				limit = 5
			}
			for _, v := range score.Violations[:limit] {
				fmt.Printf("    - %s\n", v)
			}
			if len(score.Violations) > 5 {
				fmt.Printf("    ... and %d more\n", len(score.Violations)-5)
			}
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
	name       string
	schema     string
	constraint string
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

		// Find matching schema and constraints
		var schemaPath, constraintPath string
		for _, sc := range scenarios {
			if sc.name == result.Schema {
				schemaPath = filepath.Join(exampleDir, sc.schema)
				constraintPath = filepath.Join(exampleDir, sc.constraint)
				break
			}
		}
		if schemaPath == "" {
			log.Printf("skip %s: no matching scenario", entry.Name())
			continue
		}

		var schema Schema
		if err := loadJSON(schemaPath, &schema); err != nil {
			log.Printf("skip %s: %v", entry.Name(), err)
			continue
		}
		var constraints Constraints
		if err := loadJSON(constraintPath, &constraints); err != nil {
			log.Printf("skip %s: %v", entry.Name(), err)
			continue
		}

		score := validateRecords(result.Records, schema, constraints)
		fmt.Printf("%s: schema=%.0f%%  rules=%.0f%%  unique=%.0f%%  overall=%.0f%%\n",
			entry.Name(), score.SchemaCompliance*100, score.RuleCompliance*100,
			score.Uniqueness*100, score.Overall*100)
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
			Example:      fmt.Sprintf("test-data-gen/%s", result.Schema),
			Model:        result.Model,
			Quality:      result.Score.Overall,
			QualityName:  "Compliance",
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
	if err := os.WriteFile(reportPath, []byte("# Test Data Generation Results\n\n"+report), 0644); err != nil {
		log.Printf("WARNING: could not write report: %v", err)
	}
}
