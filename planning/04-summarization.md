# Plan 04: Focused Summarization

**Status:** DRAFT
**Parent:** [00-project-overview.md](00-project-overview.md)
**Phase:** 2 (Expansion)

---

## 1. What This Demonstrates

A small local model produces concise, accurate summaries of focused inputs — not novels or research papers, but the kind of short-to-medium documents that appear constantly in software development workflows. The key insight: summarization of **bounded, domain-specific inputs** is a very different task from open-ended summarization.

## 2. Why Small Models Should Shine Here

- **Bounded input length** — git diffs, log windows, and PR descriptions are short enough to fit in small model context windows
- **Template-adjacent output** — changelog entries and commit summaries follow predictable patterns
- **Compression, not creation** — the model is reducing information, not inventing it
- **MLPerf evidence** — Llama3.1-8B benchmarks well on CNN-DailyMail summarization
- **High volume** — summarizing every commit or every log window adds up fast at API prices

## 3. Example Scenarios

### 3a. Git Diff → Changelog Entry
**Input:** `git diff` output (unified diff format, up to ~500 lines)
**Output:** 1-3 sentence changelog entry describing what changed and why it matters
**Scoring:** Rouge-1/2/L against human-written entries, plus manual quality spot-checks

### 3b. PR Description Generator
**Input:** Collection of commit messages + overall diff stats for a branch
**Output:** Structured PR description: summary, key changes (bulleted), testing notes
**Scoring:** Coverage (did it mention all significant changes?), accuracy (no hallucinated changes)

### 3c. Log Window Condensation
**Input:** 50-200 log lines from an application (mix of info, warn, error)
**Output:** 3-5 sentence summary: what happened, any errors, overall health assessment
**Scoring:** Error detection rate (did it catch all errors?), accuracy of health assessment

### 3d. Meeting Notes → Action Items
**Input:** Raw meeting transcript or notes (500-1500 words)
**Output:** JSON array of action items: { owner: string, action: string, deadline: string | null }
**Scoring:** Recall of action items, accuracy of owner/deadline extraction

## 4. Implementation Approach

```
examples/summarization/
├── README.md
├── main.go
├── prompts/
│   ├── changelog.txt
│   ├── pr-description.txt
│   ├── log-summary.txt
│   └── action-items.txt
├── testdata/
│   ├── diffs/            # Real or realistic git diffs
│   ├── commits/          # Commit message collections
│   ├── logs/             # Application log windows
│   └── meetings/         # Meeting note samples
├── expected/             # Human-written reference summaries
├── score.sh              # Rouge scoring + custom checks
└── RESULTS.md
```

### Key design decisions:
- **Focus on developer-workflow summarization** — not generic text summarization
- **Length constraints in prompts** — "summarize in 2-3 sentences" to keep output comparable
- **Error detection as a binary check** — for log summarization, did the model identify every ERROR-level event?

## 5. Metrics Captured

| Metric | How |
|--------|-----|
| Rouge-1/2/L | Standard summarization metrics against references |
| Coverage | % of significant changes/events mentioned |
| Hallucination rate | % of claims not supported by input |
| Length compliance | Did it respect the length constraint? |
| Latency | ms per summary |
| Tokens (in/out) | For cost comparison |

## 6. Open Questions

- How do small models handle large diffs (500+ lines)? May need chunking strategies.
- Is Rouge the right metric here, or do we need task-specific scoring?
- Should we test iterative summarization (summarize chunks, then summarize summaries)?
