# Example 07: Semantic Search Reranking

A small local model reranks search results by semantic relevance to a query. The model receives N candidate results from an initial retrieval stage (vector search, BM25, grep) and reorders them by relevance. This is a common pattern in RAG pipelines.

## Why Small Models Work Here

- **Scoring, not generating** -- the model assigns relevance scores, not prose
- **Small context per item** -- each candidate is a title + snippet (50-100 tokens)
- **Bounded output** -- a JSON array of scores, one per candidate
- **Latency matters** -- reranking sits between retrieval and presentation

## Scenarios

| Scenario | Query | Candidates | Description |
|----------|-------|------------|-------------|
| doc_search | Database connection pooling in Go | 18 | Documentation snippets from vector search |
| code_search | Validate email address format | 15 | Code snippets from grep/AST search |
| api_search | Rate limiting middleware with sliding window | 16 | Mixed docs and code results |

## Running

```bash
# Prerequisites: Ollama running with a model pulled
ollama pull qwen3:4b

# Run with default model
make run

# Run with a specific model
make run MODEL=llama3.2:3b

# Score results against gold standard
make score

# Generate comparison report
make report
```

## Scoring

Results are evaluated with two ranking metrics:

- **NDCG@10** (Normalized Discounted Cumulative Gain) -- Measures overall ranking quality in the top 10 positions, weighting higher positions more heavily. Range: 0.0 (worst) to 1.0 (perfect).
- **MRR** (Mean Reciprocal Rank) -- 1 divided by the position of the first highly-relevant result. A model that puts the best result first scores 1.0.

Gold-standard relevance grades (0-3) are in `baseline/`. Each candidate has a human-assigned relevance score and justification.

## How It Works

1. Load a search query with candidate results from `testdata/`
2. Send the query + candidates to the model with a reranking prompt
3. Parse the model's JSON response containing relevance scores (0.0-1.0)
4. Sort candidates by score and compare to gold-standard ranking
5. Compute NDCG@10 and MRR metrics

## File Structure

```
search-reranking/
├── main.go                         # Reranking implementation
├── testdata/
│   ├── doc_search.json             # Documentation search scenario
│   ├── code_search.json            # Code search scenario
│   └── api_search.json             # API/mixed search scenario
├── baseline/
│   ├── doc_search_gold.json        # Gold-standard rankings
│   ├── code_search_gold.json
│   └── api_search_gold.json
├── results/                        # Model outputs (generated)
├── score.sh                        # Scoring script
├── Makefile
└── README.md
```
