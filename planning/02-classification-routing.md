# Plan 02: Classification and Routing

**Status:** DRAFT
**Parent:** [00-project-overview.md](00-project-overview.md)
**Phase:** 1 (Foundation)

---

## 1. What This Demonstrates

A small local model classifies text into predefined categories and routes inputs to appropriate handlers. This is arguably the **simplest and most cost-effective** use case for local models — the output is a single label (or a short list), the decision space is bounded, and latency matters because classification is often a gateway step.

## 2. Why Small Models Should Shine Here

- **Tiny output space** — one label from a known set; generation length is trivial
- **Well-bounded problem** — the model isn't creating, it's choosing
- **Latency-sensitive** — classification gates downstream processing; 50ms local vs. 500ms API is meaningful
- **Volume-sensitive** — classifying 10,000 inputs/day at $0 vs. $3-30 adds up
- **Strong evidence** — SLM-Bench and Distillabs both show competitive classification from small models

## 3. Example Scenarios

### 3a. Issue Triage
**Input:** GitHub issue title + body text
**Output:** JSON: { category: "bug" | "feature" | "question" | "docs" | "performance", priority: "critical" | "high" | "medium" | "low", component: string }
**Scoring:** Accuracy per field, weighted F1

### 3b. Intent Detection (Support)
**Input:** Customer message (1-3 sentences)
**Output:** JSON: { intent: "billing" | "technical" | "account" | "cancellation" | "feedback" | "other", sentiment: "positive" | "neutral" | "negative", needs_human: boolean }
**Scoring:** Accuracy per field, confusion matrix

### 3c. Content Moderation
**Input:** User-generated text (comment, post, message)
**Output:** JSON: { safe: boolean, categories: ["spam" | "harassment" | "nsfw" | "misinformation" | "none"], confidence: float }
**Scoring:** Accuracy, false positive rate (critical — over-moderation is a real problem)

### 3d. Request Router
**Input:** Natural language request
**Output:** JSON: { route: "search" | "create" | "update" | "delete" | "navigate" | "help", entity: string, parameters: {} }
**Scoring:** Route accuracy, entity extraction accuracy

## 4. Implementation Approach

```
examples/classification-routing/
├── README.md
├── main.go
├── categories/               # Category definitions per scenario
│   ├── issue-triage.json
│   ├── intent-detection.json
│   ├── content-moderation.json
│   └── request-router.json
├── prompts/
│   ├── issue-triage.txt
│   ├── intent-detection.txt
│   ├── content-moderation.txt
│   └── request-router.txt
├── testdata/                 # 20-50 labeled examples per scenario
│   ├── issues/
│   ├── messages/
│   ├── content/
│   └── requests/
├── expected/                 # Ground truth labels
├── score.sh
└── RESULTS.md
```

### Key design decisions:
- **Categories are provided in the prompt** — the model picks from a defined list
- **Confidence calibration** — track whether model confidence correlates with accuracy
- **Batch mode** — measure throughput: how many classifications per second?
- **Few-shot vs. zero-shot** — test both to measure the uplift from examples

## 5. Metrics Captured

| Metric | How |
|--------|-----|
| Accuracy | Overall correct classification rate |
| Per-class F1 | Precision/recall per category |
| Confusion matrix | Where does the model get confused? |
| False positive rate | Critical for moderation scenario |
| Latency (per item) | Average ms per classification |
| Throughput | Classifications per second (batch) |
| Consistency | Same input 10 times → same output? |

## 6. Test Data Strategy

- **Issue triage:** Source from public GitHub repos (sanitized) or generate synthetic issues
- **Intent detection:** Adapt from public customer service datasets or generate synthetic
- **Content moderation:** Carefully curated synthetic examples (avoid real harmful content)
- **Request router:** Generate synthetic natural language commands

**Important:** Label quality matters more than quantity. 30 well-labeled examples beat 200 sloppy ones.

## 7. Models to Test

**Primary:** Qwen3-4B, Ministral-3-3B, Phi-3-mini, Llama3.2-3B
**Secondary:** Gemma-2-2B (test the floor — can 2B classify?), Mistral-7B-v0.3
**Baseline:** Claude Sonnet 4.5, Claude Opus 4.6

## 8. Hypothesis

Classification is where we expect the **strongest showing** from small models relative to large ones. A 3B model should achieve >95% of the accuracy of Opus on well-defined classification tasks, at 1/100th the cost and 1/5th the latency.

If this hypothesis fails, it tells us something important too.

## 9. Open Questions

- What's the maximum number of categories before small models degrade? (5? 20? 50?)
- Does chain-of-thought help small models classify, or does it just waste tokens?
- How sensitive are small models to prompt format? (numbered list vs. JSON enum vs. natural language description of categories)
