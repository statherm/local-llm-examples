# Plan 03: Function Calling and Tool Selection

**Status:** DRAFT
**Parent:** [00-project-overview.md](00-project-overview.md)
**Phase:** 1 (Foundation)

---

## 1. What This Demonstrates

A small local model receives a natural language request and a catalog of available tools/functions, then selects the correct tool and generates the correct parameters. This is the backbone of agentic workflows and one area where small models have **published evidence of outperforming large ones** on specific benchmarks.

## 2. Why Small Models Should Shine Here

- **The ToolBench result** — a 350M parameter SLM achieved 77.5% pass rate vs. ChatGPT-CoT at 26%. This is the single strongest evidence point in our research.
- **Bounded decision space** — choose from N tools, fill M parameters
- **Structured output** — function calls are JSON; models with JSON mode excel here
- **Ministral-3-3B** — specifically designed for function calling and structured output
- **MCP adoption** — the Model Context Protocol is becoming standard; demonstrating local models as MCP-compatible tool callers is timely

## 3. Example Scenarios

### 3a. Developer Toolbox
**Context:** A developer has 8 tools available (search_code, read_file, run_tests, create_file, edit_file, git_status, list_files, explain_code)
**Input:** Natural language request like "find all TODO comments in the auth module"
**Output:** JSON: { tool: "search_code", parameters: { pattern: "TODO", path: "auth/" } }
**Scoring:** Correct tool selection + correct parameter values

### 3b. Home Automation
**Context:** 10 smart home functions (set_thermostat, toggle_light, lock_door, set_alarm, play_music, check_weather, set_timer, send_notification, camera_snapshot, run_scene)
**Input:** Natural language command like "it's getting cold, bump the heat up to 72"
**Output:** JSON: { tool: "set_thermostat", parameters: { temperature: 72, unit: "fahrenheit" } }
**Scoring:** Correct tool + correct parameters (with type coercion tolerance)

### 3c. API Gateway Router
**Context:** 12 REST API endpoints with method, path, and required/optional parameters
**Input:** Natural language like "get the last 5 orders for customer 42"
**Output:** JSON: { method: "GET", path: "/customers/42/orders", query: { limit: 5, sort: "desc" } }
**Scoring:** Correct endpoint + correct parameters + correct method

### 3d. Multi-Step Tool Chains (Advanced)
**Context:** Same tool catalogs, but the request requires 2-3 sequential tool calls
**Input:** "Read the config file and update the port to 8080"
**Output:** Array of tool calls in order: [{ tool: "read_file", ... }, { tool: "edit_file", ... }]
**Scoring:** Correct tools in correct order + correct parameters

## 4. Implementation Approach

```
examples/function-calling/
├── README.md
├── main.go
├── tools/                    # Tool catalog definitions
│   ├── developer.json        # Tool schemas (name, description, parameters)
│   ├── home-automation.json
│   └── api-gateway.json
├── prompts/
│   ├── system.txt            # System prompt template (tool catalog injected)
│   └── variants/             # Prompt format variations to test
├── testdata/                 # Natural language requests + expected calls
│   ├── developer/
│   ├── home-automation/
│   └── api-gateway/
├── expected/
├── score.sh
└── RESULTS.md
```

### Key design decisions:
- **Tool schemas follow OpenAI function calling format** — widely understood, easy to compare
- **Test parameter extraction separately from tool selection** — a model might pick the right tool but botch the parameters
- **Ambiguity cases included** — requests that could map to multiple tools, to test model judgment
- **Distractor tools** — catalog includes tools that are plausible but wrong for the request

## 5. Metrics Captured

| Metric | How |
|--------|-----|
| Tool selection accuracy | % correct tool chosen |
| Parameter accuracy | % correct parameters (exact match per param) |
| Combined accuracy | Both tool AND all parameters correct |
| Ambiguity handling | Quality of choice when multiple tools could apply |
| Multi-step accuracy | Correct sequence for chained tool calls |
| Latency | ms per tool call decision |
| Schema compliance | % of outputs that are valid JSON matching expected format |

## 6. Test Data Strategy

- **30+ requests per scenario**, ranging from trivial ("turn off the lights") to ambiguous ("make the living room cozy")
- **Ground truth includes acceptable alternatives** — some requests legitimately map to multiple tools
- **Edge cases:** missing information (model should ask or use defaults), conflicting parameters, out-of-scope requests (model should decline)

## 7. Models to Test

**Primary:** Ministral-3-3B (purpose-built), Qwen3-4B, Phi-3-mini, Llama3.2-3B
**Secondary:** Mistral-7B-v0.3 (function calling support), Qwen2.5-Coder-7B
**Baseline:** Claude Sonnet 4.5, Claude Opus 4.6

**Special interest:** Ministral-3-3B is the headline candidate here. If a 3B model purpose-built for function calling can match commercial models on this task, that's a powerful story.

## 8. Open Questions

- How does catalog size affect accuracy? (5 tools vs. 20 vs. 50)
- Do small models handle nested parameters (objects within objects) well?
- How much does tool description quality affect small model performance vs. large?
- Should we test native function calling (Ollama's tool calling mode) vs. prompt-based?
