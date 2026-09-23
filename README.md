# llm-router

[![Go Reference](https://pkg.go.dev/badge/github.com/devravik/llm-router.svg)](https://pkg.go.dev/github.com/devravik/llm-router)
[![CI](https://github.com/devravik/llm-router/actions/workflows/ci.yml/badge.svg)](https://github.com/devravik/llm-router/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/devravik/llm-router)](https://goreportcard.com/report/github.com/devravik/llm-router)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

LLM API key rotation and provider routing for Go with zero external dependencies.

Route requests across multiple API keys, models, providers, and custom OpenAI-compatible endpoints.

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

`llm-router` solves one specific problem: rotating and selecting LLM credentials and endpoints without manually switching keys or maintaining custom synchronization logic.

The router selects which route to use. The caller remains responsible for executing the network request using their preferred HTTP client or provider SDK.

---

## Comparison

| Feature | `llm-router` | Ad-hoc Slice | HTTP Proxy / Gateway |
| :--- | :--- | :--- | :--- |
| Multiple API key rotation | Yes | Manual index tracking | Yes |
| Multi-provider support | Yes | Custom maps | Yes |
| Round-robin selection | Built-in | Requires mutex/atomics | Yes |
| Weighted distribution | Built-in | Custom implementation | Yes |
| Concurrent random selection | Built-in (`math/rand/v2`) | Mutex contention | Yes |
| Custom endpoints (`BaseURL`) | Supported | Manual routing | Yes |
| Runtime route enable/disable | Supported | Prone to data races | Yes |
| Concurrency safety | Yes (`sync.RWMutex`) | Caller responsibility | Yes |
| External dependencies | 0 (standard library only) | 0 | External daemon/containers |
| Network latency overhead | None (in-process selection) | None | Network hop (1-5ms+) |

---

## Installation

Requires Go 1.22 or later:

```bash
go get github.com/devravik/llm-router
```

---

## Runnable Examples

Complete, runnable examples are provided in the [`examples/`](examples/) directory:

| Directory | Description |
| :--- | :--- |
| [`examples/groq-key-rotation`](examples/groq-key-rotation/) | Distribute requests across multiple Groq API keys using round-robin. |
| [`examples/openai-fallback`](examples/openai-fallback/) | Fallback to a secondary provider (Anthropic) when the primary (OpenAI) fails. |
| [`examples/multi-provider`](examples/multi-provider/) | Dispatch requests across OpenAI, Anthropic, Gemini, Groq, DeepSeek, and OpenRouter. |
| [`examples/weighted-routing`](examples/weighted-routing/) | Split traffic proportionally between different models (e.g., 80% fast, 20% reasoning). |
| [`examples/custom-openai-endpoint`](examples/custom-openai-endpoint/) | Route to self-hosted Ollama (`localhost:11434`), vLLM, or corporate proxies via `BaseURL`. |
| [`examples/runtime-disable`](examples/runtime-disable/) | Temporarily remove a key on HTTP 429 rate limit and re-enable it after a cooldown. |

---

## Architecture & Boundaries

`llm-router` maintains a clear boundary:

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

1. **Router**: Tracks registered routes, checks eligibility, applies the configured selection strategy, and returns a copy of the selected `Route`.
2. **Application**: Receives the `Route` and executes the network call.

The library intentionally does not make HTTP calls, handle retries, parse provider payloads, or manage tokens.

---

## Route Configuration

A route represents a selectable configuration:

```go
type Route struct {
    ID       string // Unique identifier (required)
    Provider string // Provider label (e.g., "groq", "anthropic", "openai")
    Model    string // Model name (e.g., "llama-3.3-70b", "gpt-4o")
    APIKey   string // Credential secret
    BaseURL  string // Optional endpoint override (e.g., self-hosted Ollama/vLLM)
    Weight   int    // Proportional weight for Weighted strategy (>= 0)
}
```

### Runtime Route Lifecycle

Routes can be modified dynamically while the application is serving traffic:

```go
// Temporarily exclude a route from selection (e.g., during cooldown)
err := router.Disable("groq-primary")

// Restore route eligibility
err = router.Enable("groq-primary")

// Permanently remove a route
err = router.Remove("groq-primary")
```

---

## Routing Strategies

### Round Robin (Default)

Selects active routes sequentially. Ensures fair distribution over $N \times k$ calls under concurrent execution:

```go
router := llmrouter.New(
    llmrouter.WithStrategy(llmrouter.RoundRobin()),
)
```

### Weighted

Distributes requests proportionally based on `Route.Weight`. A route with weight `8` receives four times the traffic of a route with weight `2`. Routes with weight `0` are ignored.

```go
router := llmrouter.New(
    llmrouter.WithStrategy(llmrouter.Weighted()),
)

router.Add(llmrouter.Route{
    ID:       "fast-tier",
    Provider: "groq",
    Model:    "llama-3.3-70b",
    APIKey:   os.Getenv("GROQ_KEY"),
    Weight:   8,
})

router.Add(llmrouter.Route{
    ID:       "heavy-tier",
    Provider: "openai",
    Model:    "gpt-4o",
    APIKey:   os.Getenv("OPENAI_KEY"),
    Weight:   2,
})
```

### Random

Selects eligible routes uniformly at random. Uses `math/rand/v2` to avoid global lock contention.

```go
router := llmrouter.New(
    llmrouter.WithStrategy(llmrouter.Random()),
)
```

### Custom Strategies

Custom selection logic can be implemented via the `Strategy` interface:

```go
type Strategy interface {
    Next(routes []Route) (Route, error)
}
```

Example priority fallback strategy:

```go
type PriorityStrategy struct{}

func (p *PriorityStrategy) Next(routes []llmrouter.Route) (llmrouter.Route, error) {
    if len(routes) == 0 {
        return llmrouter.Route{}, llmrouter.ErrNoRoutes
    }
    return routes[0], nil
}
```

---

## Benchmarks

Benchmark results on Linux amd64 (12th Gen Intel Core i5-12400) across 100 configured routes:

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

* Selection overhead is ~1.3-1.8 microseconds with 100 routes.
* One heap allocation per selection (defensive slice copy preventing caller mutation).
* Scales linearly under saturated parallel goroutine load.

---

## Error Handling

Sentinel errors can be inspected using `errors.Is`:

```go
route, err := router.Next()
if err != nil {
    if errors.Is(err, llmrouter.ErrNoRoutes) {
        // No routes are registered or all routes are disabled
    }
}
```

| Error | Cause |
| :--- | :--- |
| `ErrNoRoutes` | No active routes available for selection |
| `ErrNotFound` | Route ID not found on `Remove`, `Enable`, or `Disable` |
| `ErrDuplicateRoute` | Route ID already registered on `Add` |
| `ErrInvalidRoute` | Empty `ID` or negative `Weight` on `Add` |

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines, testing instructions, and code standards.

* [Code of Conduct](CODE_OF_CONDUCT.md)
* [Security Policy](SECURITY.md)

---

## License

MIT License. See [LICENSE](LICENSE) for details.
