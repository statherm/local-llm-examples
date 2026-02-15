# Plan 07: Semantic Search Reranking

**Status:** DRAFT
**Parent:** [00-project-overview.md](00-project-overview.md)
**Phase:** 3 (Exploration)

---

## 1. What This Demonstrates

A small local model reranks a set of search results by semantic relevance to a query. The model doesn't perform the search — it receives N candidate results and reorders them. This is a common pattern in RAG pipelines where the initial retrieval (vector search, BM25) returns good-but-imperfect results.

## 2. Why Small Models Should Shine Here

- **Scoring, not generating** — the model assigns relevance scores, not creates text
- **Small context per item** — typically a title + snippet per result (50-100 tokens each)
- **Bounded output** — an ordered list or score per item
- **Latency matters** — reranking sits between retrieval and presentation; users are waiting

## 3. Example Scenarios

### 3a. Documentation Search Reranking
**Input:** Query + 20 candidate doc snippets from vector search
**Output:** Top-10 reranked by relevance, with confidence scores

### 3b. Code Search Reranking
**Input:** Natural language query + 15 code snippet results from grep/AST search
**Output:** Reranked results prioritizing semantic match over keyword match

## 4. Open Questions

- Is a generative model the right tool here, or are cross-encoder models (BERT-class) better?
- What's the tradeoff vs. dedicated reranking models like Cohere Rerank or bge-reranker?
- Does the model need to see all results at once, or can it score independently?
