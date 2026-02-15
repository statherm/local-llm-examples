# Gap Analysis: Why This Project Needs to Exist

**Date:** 2026-02-14

---

## The Problem

The open-source / local LLM community primarily evaluates models on two axes:

1. **Code generation** — Can it write code? (HumanEval, SWE-Bench, LiveCodeBench)
2. **General reasoning** — How smart is it? (MMLU, ARC, HellaSwag, GSM8K)

This creates a distorted picture where small models always look inferior to large commercial models. A 3B model will never beat Opus 4.6 on SWE-Bench. But that's the wrong question.

The right question: **For a given task embedded in a pipeline, what is the smallest model that produces acceptable quality at the lowest cost and latency?**

## The Three Camps (and the missing fourth)

### Camp 1: Academic Benchmarks
- **Who:** Researchers, ML engineers
- **Output:** Papers, leaderboard scores
- **Limitation:** Nobody runs these benchmarks to make practical decisions. They prove a point but don't ship a product.

### Camp 2: Infrastructure / Runners
- **Who:** Ollama, llama.cpp, vLLM developers
- **Output:** "Run any model locally"
- **Limitation:** Tool-agnostic. They make it possible to run models but offer no opinion on what to run for what purpose.

### Camp 3: Framework Integrators
- **Who:** LangChain, Instructor, Outlines
- **Output:** "Plug any model into any pipeline"
- **Limitation:** Framework-focused. Compare libraries, not model capabilities for specific tasks.

### Camp 4: The Missing Piece (This Project)
- **Who:** Practitioners, developers, teams deciding what to deploy
- **Output:** Working examples showing "use this model for this task, here's why, here's proof"
- **Limitation:** Does not exist yet.

## What Practitioners Actually Need

1. **A runnable demo** — not a paper, not a framework, a thing I can `git clone` and run
2. **Task-specific recommendations** — "for invoice extraction, Qwen3-4B via Ollama with Instructor gives 94% accuracy at 180ms/doc"
3. **Cost/quality/latency tradeoff data** — the three-way comparison that actually drives deployment decisions
4. **Composability examples** — how to use a small model as one step in a larger pipeline, not as a standalone chatbot
5. **Honest baselines** — show the same task on a large commercial model so users can see where quality IS equivalent and where it ISN'T

## Success Criteria for This Project

The project succeeds if a developer can:

1. Clone the repo
2. Pick a use case from the catalog
3. Run it locally with Ollama
4. See results compared against a commercial model baseline
5. Understand the cost/quality/latency tradeoff
6. Copy the pattern into their own project

The project does NOT need to:
- Prove small models are "better" than large ones (they usually aren't, in general)
- Build production-ready tools (these are demonstrations, not products)
- Cover every possible use case (start with the strongest candidates)
