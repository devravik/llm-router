# llm-router

[![Go Reference](https://pkg.go.dev/badge/github.com/devravik/llm-router.svg)](https://pkg.go.dev/github.com/devravik/llm-router)
[![CI](https://github.com/devravik/llm-router/actions/workflows/ci.yml/badge.svg)](https://github.com/devravik/llm-router/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/devravik/llm-router)](https://goreportcard.com/report/github.com/devravik/llm-router)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**LLM API key rotation and provider routing for Go.**

Route requests across multiple API keys, models, providers, and custom OpenAI-compatible endpoints with **zero external dependencies**.

```go
router := llmrouter.New(
    llmrouter.WithStrategy(llmrouter.RoundRobin()),
)

router.Add(llmrouter.Route{
    ID:       "groq-1",
    Provider: "groq",
    Model:    "llama-3.3-70b",
    APIKey:   os.Getenv("GROQ_KEY_1"),
})
router.Add(llmrouter.Route{
    ID:       "groq-2",
    Provider: "groq",
    Model:    "llama-3.3-70b",
    APIKey:   os.Getenv("GROQ_KEY_2"),
})

route, err := router.Next()
// route.ID, route.Provider, route.Model, route.APIKey
```

`llm-router` solves one specific problem: **efficiently rotating and selecting LLM credentials and endpoints without manually switching keys or writing ad-hoc routing logic.**

The router chooses *which route to use*. Your application remains in complete control over how to execute the actual HTTP request or SDK call.

---

## Why llm-router?

| Capability | `llm-router` | Ad-hoc / DIY Slices | Heavy Proxies / Gateways |
| :--- | :---: | :---: | :---: |
| **Multiple API Key Rotation** | :white_check_mark: Yes | :warning: Manual tracking | :white_check_mark: Yes |
| **Multi-Provider Support** | :white_check_mark: Yes | :warning: Custom maps | :white_check_mark: Yes |
| **Round Robin Strategy** | :white_check_mark: Built-in | :warning: Needs atomics/locks | :white_check_mark: Yes |
| **Weighted Traffic Splitting** | :white_check_mark: Built-in | :x: Complex math | :white_check_mark: Yes |
| **Uniform Random Selection** | :white_check_mark: Built-in (`math/rand/v2`) | :warning: Lock contention | :white_check_mark: Yes |
| **Custom / Self-Hosted Endpoints** | :white_check_mark: Yes (`BaseURL`) | :warning: Ad-hoc | :white_check_mark: Yes |
| **Runtime Enable / Disable** | :white_check_mark: Yes (cooldowns/outages) | :x: Race prone | :white_check_mark: Yes |
| **Thread-Safe & Lock-Contention Free** | :white_check_mark: Concurrency-Safe | :x: Race prone | :white_check_mark: Yes |
| **External Dependencies** | **0 (Standard Library Only)** | 0 | Dozens + Docker |
| **HTTP Interception Overhead** | **None (In-process selection)** | None | Added network hop (1-5ms+) |

---

## Installation

Requires **Go 1.22** or later:

```bash
go get github.com/devravik/llm-router
```

---

## Real-World Recipes & Examples

Explore fully compilable, runnable examples in the [`examples/`](examples/) directory:

| Recipe | Description | Source |
| :--- | :--- | :--- |
| **Rotate Multiple Groq Keys** | Distribute requests across multiple Groq API keys using round-robin to multiply rate limits. | [`examples/groq-key-rotation`](examples/groq-key-rotation/) |
| **OpenAI + Anthropic Fallback** | Priority failover: route to primary provider (OpenAI) and automatically switch to backup (Anthropic) on errors. | [`examples/openai-fallback`](examples/openai-fallback/) |
| **Multi-Provider Routing** | Unified dispatcher across OpenAI, Anthropic, Gemini, Groq, DeepSeek, and OpenRouter. | [`examples/multi-provider`](examples/multi-provider/) |
| **Weighted Canary Routing** | Split traffic proportionally (e.g. 80% to economical Llama 3.3, 20% to deep reasoning GPT-4o). | [`examples/weighted-routing`](examples/weighted-routing/) |
| **Custom & Self-Hosted Endpoints** | Route to local Ollama (`localhost:11434`), vLLM, or corporate proxies with cloud failover via `BaseURL`. | [`examples/custom-openai-endpoint`](examples/custom-openai-endpoint/) |
| **HTTP 429 Cooldown & Backoff** | Dynamically disable a route upon hitting a rate limit and automatically re-enable it after a cooldown. | [`examples/runtime-disable`](examples/runtime-disable/) |

---

## Request Flow & Architectural Boundary

`llm-router` maintains a clean separation of concerns:

```text
Application
    │
    ▼
LLM Router  ──►  "Use route: groq-1 (key: gsk-...)"
    │
    ▼
LLM Client / SDK (net/http, official SDK, etc.)
    │
    ▼
Provider API (Groq, Anthropic, OpenAI, Ollama, etc.)
```

1. **Router**: Manages candidate routes, applies selection strategies, and returns a safe copy of the chosen `Route`.
2. **Application**: Receives the `Route` and executes the network request using whatever HTTP client or provider SDK it prefers.

The router never makes network calls, handles retries, or parses provider responses.

---

## Route Configuration & Lifecycle

A route represents an eligible LLM credential and endpoint target:

```go
type Route struct {
    ID       string // Unique route identifier (required)
    Provider string // Provider name (e.g. "groq", "anthropic", "openai") - metadata
    Model    string // Model name (e.g. "llama-3.3-70b", "gpt-4o")
    APIKey   string // Credential secret (never logged or exposed)
    BaseURL  string // Optional: overrides default endpoint (e.g. self-hosted Ollama/vLLM)
    Weight   int    // Proportional weight for Weighted strategy (>= 0)
}
```

### Runtime Enable, Disable & Remove

Routes can be dynamically modified at runtime without stopping your service:

```go
// Temporarily take a key out of rotation (e.g. on HTTP 429 rate limit)
err := router.Disable("groq-primary")

// Restore route eligibility after cooldown period
err = router.Enable("groq-primary")

// Permanently remove a decommissioned key
err = router.Remove("groq-primary")
```

---

## Routing Strategies

### 1. Round Robin (Default)

Selects active routes in sequential round-robin order. Guarantees fair distribution over $N \times k$ calls under high concurrency.

```go
router := llmrouter.New(
    llmrouter.WithStrategy(llmrouter.RoundRobin()),
)
```

### 2. Weighted

Selects routes proportionally according to their `Weight`. A route with weight `8` receives roughly 4x the requests of a route with weight `2`. Routes with weight `0` are excluded from selection.

```go
router := llmrouter.New(
    llmrouter.WithStrategy(llmrouter.Weighted()),
)

router.Add(llmrouter.Route{
    ID:       "fast-tier",
    Provider: "groq",
    Model:    "llama-3.3-70b",
    APIKey:   os.Getenv("GROQ_KEY"),
    Weight:   8, // 80%
})

router.Add(llmrouter.Route{
    ID:       "heavy-tier",
    Provider: "openai",
    Model:    "gpt-4o",
    APIKey:   os.Getenv("OPENAI_KEY"),
    Weight:   2, // 20%
})
```

### 3. Random

Selects eligible routes with uniform randomness. Uses `math/rand/v2` introduced in Go 1.22 for lock-free, concurrent randomness.

```go
router := llmrouter.New(
    llmrouter.WithStrategy(llmrouter.Random()),
)
```

### 4. Custom Strategies

Implement the single-method `Strategy` interface to build domain-specific routing:

```go
type Strategy interface {
    Next(routes []Route) (Route, error)
}
```

```go
// Example: Strict Priority Strategy
type PriorityStrategy struct{}

func (p *PriorityStrategy) Next(routes []llmrouter.Route) (llmrouter.Route, error) {
    if len(routes) == 0 {
        return llmrouter.Route{}, llmrouter.ErrNoRoutes
    }
    return routes[0], nil
}
```

---

## Performance Benchmarks

Measured on an Intel Core i5-12400 (12 cores) running Linux `amd64` across **100 registered routes**:

```text
goos: linux
goarch: amd64
pkg: github.com/devravik/llm-router
cpu: 12th Gen Intel(R) Core(TM) i5-12400
BenchmarkRoundRobin-12             877251        1318 ns/op        9472 B/op        1 allocs/op
BenchmarkRandom-12                 837199        1366 ns/op        9472 B/op        1 allocs/op
BenchmarkWeighted-12               682854        1841 ns/op        9472 B/op        1 allocs/op
BenchmarkRoundRobin_Parallel-12    792451        1508 ns/op        9472 B/op        1 allocs/op
BenchmarkRandom_Parallel-12        777202        1566 ns/op        9472 B/op        1 allocs/op
BenchmarkWeighted_Parallel-12      728376        1641 ns/op        9472 B/op        1 allocs/op
```

* **Sub-2 microsecond selection** across 100 candidate routes.
* **1 heap allocation** per selection (defensive slice copy preventing caller mutation).
* **Linear scaling** under saturated parallel goroutine execution.

---

## Error Handling

Sentinel errors allow explicit condition matching with `errors.Is`:

```go
route, err := router.Next()
if err != nil {
    if errors.Is(err, llmrouter.ErrNoRoutes) {
        // Handle all routes disabled or none registered
    }
}
```

| Error | Condition |
| :--- | :--- |
| `ErrNoRoutes` | No active routes available for selection |
| `ErrNotFound` | Target route ID does not exist on `Remove`, `Enable`, or `Disable` |
| `ErrDuplicateRoute` | Route ID already registered on `Add` |
| `ErrInvalidRoute` | Invalid configuration (empty `ID` or negative `Weight`) on `Add` |

---

## Community & Contributing

We welcome contributions! Please review our guidelines before submitting pull requests:

* [Contributing Guide](CONTRIBUTING.md)
* [Code of Conduct](CODE_OF_CONDUCT.md)
* [Security Policy](SECURITY.md)

---

## License

MIT License. See [LICENSE](LICENSE) for details.
