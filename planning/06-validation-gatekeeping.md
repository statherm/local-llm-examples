# Plan 06: Validation and Gatekeeping

**Status:** DRAFT
**Parent:** [00-project-overview.md](00-project-overview.md)
**Phase:** 2 (Expansion)

---

## 1. What This Demonstrates

A small local model acts as a validation layer — checking whether inputs are safe, well-formed, on-topic, or compliant with rules that are hard to express as regex but easy to describe in natural language. This is the "bouncer at the door" pattern: fast, cheap, and good enough to filter 95% of cases before expensive processing.

## 2. Why Small Models Should Shine Here

- **Binary or narrow output** — safe/unsafe, valid/invalid, on-topic/off-topic
- **Speed-critical** — validation gates the pipeline; latency directly affects user experience
- **Volume-critical** — every input hits the validator; $0 local vs. per-token API pricing
- **Fuzzy rules** — "does this look like a reasonable email?" is hard to regex but easy to prompt
- **Defense in depth** — a cheap local check before an expensive API call

## 3. Example Scenarios

### 3a. Prompt Injection Detection
**Input:** User prompt destined for a downstream LLM
**Output:** JSON: { safe: boolean, risk_category: "none" | "injection" | "jailbreak" | "data_exfiltration", confidence: float }
**Scoring:** Accuracy, false positive rate, false negative rate

### 3b. Schema Compliance Check
**Input:** JSON document + schema description (in natural language, not JSON Schema)
**Output:** JSON: { valid: boolean, issues: [{ field: string, problem: string }] }
**Scoring:** Issue detection rate, false positive rate

### 3c. Content Relevance Gate
**Input:** Text + topic description (e.g., "this forum is about Kubernetes")
**Output:** JSON: { on_topic: boolean, confidence: float, suggested_redirect: string | null }
**Scoring:** Accuracy on labeled dataset

### 3d. PII Detection
**Input:** Text that may contain personal information
**Output:** JSON: { contains_pii: boolean, pii_types: ["email" | "phone" | "ssn" | "address" | "name"], redacted_text: string }
**Scoring:** PII detection recall (must not miss), precision (false positives are annoying but safe)

## 4. Key Design Consideration

Validation has **asymmetric error costs**. A false negative (letting bad input through) is usually worse than a false positive (blocking good input). We need to measure and report both, and let users decide their threshold.

## 5. Open Questions

- Can small models detect prompt injection attempts that were designed to fool large models?
- What's the false positive rate at 99% recall? Is it usable?
- Should the validator explain its reasoning, or just give a verdict? (Explanation costs tokens/latency)
