# Example 04: Focused Summarization

Demonstrates small local models producing concise, accurate summaries of developer-workflow inputs: git diffs, commit histories, application logs, and meeting notes.

## Why This Works for Small Models

- **Bounded input length** -- git diffs, log windows, and meeting notes fit in small model context windows
- **Compression, not creation** -- the model reduces information rather than inventing it
- **Template-adjacent output** -- changelog entries and PR descriptions follow predictable patterns
- **High volume** -- summarizing every commit or log window adds up fast at API prices

## Scenarios

| Scenario | Input | Output | Scoring |
|----------|-------|--------|---------|
| Git diff to changelog | Unified diff | 1-3 sentence changelog entry | Token-level F1 vs reference |
| PR description | Commit messages + diff stats | Structured PR description | Token-level F1 vs reference |
| Log condensation | Application log window | 3-5 sentence health summary | Token-level F1 vs reference |
| Meeting action items | Meeting notes | JSON array of action items | Owner match + action overlap |

## Running

```bash
# Run with default model (qwen3:4b)
make run

# Run with a specific model
make run MODEL=llama3.2:3b

# Score existing results
make score

# Generate markdown report
make report
```

## Test Data

- `testdata/diffs/` -- Realistic git diffs (retry logic, null pointer fix, pagination)
- `testdata/commits/` -- Commit message collections (feature branch, bugfix branch)
- `testdata/logs/` -- Application logs (healthy deploy, degraded database, crash loop)
- `testdata/meetings/` -- Meeting notes (sprint planning, incident retrospective)

## Scoring

- **Text summaries** (diffs, logs, PRs): Token-level F1 against human-written reference summaries
- **Action items** (meetings): Owner matching with action description overlap (F1 > 0.3 threshold)

Results are saved to `results/<model>.json`.
