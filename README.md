# LLM Router

[![Go Reference](https://pkg.go.dev/badge/github.com/devravik/llm-router.svg)](https://pkg.go.dev/github.com/devravik/llm-router)
[![CI](https://github.com/devravik/llm-router/actions/workflows/ci.yml/badge.svg)](https://github.com/devravik/llm-router/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/devravik/llm-router)](https://goreportcard.com/report/github.com/devravik/llm-router)

A lightweight, provider-agnostic Go library for distributing LLM requests across multiple configured API keys and providers.

`llm-router` solves one specific problem: **efficiently rotating and selecting LLM credentials and endpoints without manually switching keys or writing ad-hoc routing logic.**

The router decides **which route to use**. Your application remains responsible for making the actual HTTP request or SDK call.

> **Status:** pre-v1.0 (`v0.x`). The public API is small and stable in practice, but may still change based on feedback before `v1.0.0`. See [Versioning](#versioning).

## Features

* **Provider & Model Agnostic**: Distribute requests across OpenAI, Anthropic, Gemini, OpenRouter, DeepSeek, Groq, self-hosted endpoints, or any custom provider.
* **Custom / Self-Hosted Endpoints**: Override the default endpoint per route via `BaseURL` — points at a proxy, self-hosted model server, or any OpenAI-compatible API.
* **Multiple API Keys**: Easily balance traffic and quotas across multiple keys for the same provider and model.
* **Built-in Routing Strategies**:
  * **Round Robin**: Fair sequential distribution across active routes.
  * **Random**: Uniform random selection via Go 1.22+ `math/rand/v2` without global lock contention.
  * **Weighted**: Proportional traffic distribution based on route weights.
* **Custom Strategies**: Implement your own selection logic with a single-method interface.
* **Dynamic Route Lifecycle**: Add, remove, enable, or disable routes at runtime.
* **Concurrency-Safe**: Designed for high-throughput concurrent use across multiple goroutines.
* **Zero External Dependencies**: Built strictly using the Go standard library.

---

## Installation

Requires **Go 1.22** or later:

```bash
go get github.com/devravik/llm-router
```

---

## Quick Start

```go
package main

import (
	"fmt"
	"log"
	"os"

	llmrouter "github.com/devravik/llm-router"
)

func main() {
	// Initialize router with round-robin strategy (the default)
	router := llmrouter.New(
		llmrouter.WithStrategy(llmrouter.RoundRobin()),
	)

	// Register routes with individual API keys
	err := router.Add(llmrouter.Route{
		ID:       "groq-primary",
		Provider: "groq",
		Model:    "llama-3.3-70b",
		APIKey:   os.Getenv("GROQ_API_KEY_1"),
	})
	if err != nil {
		log.Fatalf("failed to add route: %v", err)
	}

	err = router.Add(llmrouter.Route{
		ID:       "groq-secondary",
		Provider: "groq",
		Model:    "llama-3.3-70b",
		APIKey:   os.Getenv("GROQ_API_KEY_2"),
	})
	if err != nil {
		log.Fatalf("failed to add route: %v", err)
	}

	// Select the next route
	route, err := router.Next()
	if err != nil {
		log.Fatalf("no route available: %v", err)
	}

	// Use route.Provider, route.Model, and route.APIKey with your LLM client
	fmt.Printf("Selected route: %s (%s / %s)\n", route.ID, route.Provider, route.Model)
}
```

---

## Request Flow & Architectural Boundary

`llm-router` maintains a clean separation of concerns:

```text
Application
    │
    ▼
LLM Router  ──►  "Use route: groq-primary (key: ...)"
    │
    ▼
LLM Client / SDK
    │
    ▼
Provider API (Groq, Anthropic, OpenAI, etc.)
```

1. **Router**: Manages candidate routes, applies selection strategies, and returns a safe copy of the chosen `Route`.
2. **Application**: Receives the `Route` and executes the network request using whatever HTTP client or provider SDK it prefers.

The router never makes network calls, handles retries, or parses provider responses.

---

## Route Configuration

A route represents a selectable LLM endpoint configuration:

```go
type Route struct {
	ID       string // Unique route identifier
	Provider string // Provider name (e.g., "groq", "anthropic", "openai") — plain metadata, not validated
	Model    string // Model name (e.g., "llama-3.3-70b", "claude-3-5-sonnet")
	APIKey   string // Credential secret
	BaseURL  string // Optional: overrides the default endpoint for Provider
	Weight   int    // Relative weight for weighted selection (must be >= 0)
}
```

### Route Lifecycle

Routes can be added, removed, or temporarily toggled at runtime:

```go
// Temporarily take a route out of rotation (e.g., on HTTP 429)
err := router.Disable("groq-primary")

// Restore route eligibility
err = router.Enable("groq-primary")

// Remove route permanently
err = router.Remove("groq-primary")
```

---

## Providers

`Provider` and `BaseURL` are plain metadata — the router never calls a provider API itself, so any provider whose Go client takes a base URL and an API key works. A typical multi-provider setup:

```go
router := llmrouter.New(
	llmrouter.WithStrategy(llmrouter.RoundRobin()),
)

router.Add(llmrouter.Route{
	ID:       "anthropic-1",
	Provider: "anthropic",
	Model:    "claude-sonnet-4-5",
	APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
})

router.Add(llmrouter.Route{
	ID:       "openai-1",
	Provider: "openai",
	Model:    "gpt-5",
	APIKey:   os.Getenv("OPENAI_API_KEY"),
})

router.Add(llmrouter.Route{
	ID:       "gemini-1",
	Provider: "gemini",
	Model:    "gemini-2.5-pro",
	APIKey:   os.Getenv("GEMINI_API_KEY"),
})

router.Add(llmrouter.Route{
	ID:       "openrouter-1",
	Provider: "openrouter",
	Model:    "meta-llama/llama-3.3-70b-instruct",
	APIKey:   os.Getenv("OPENROUTER_API_KEY"),
})

router.Add(llmrouter.Route{
	ID:       "deepseek-1",
	Provider: "deepseek",
	Model:    "deepseek-chat",
	APIKey:   os.Getenv("DEEPSEEK_API_KEY"),
})

route, err := router.Next()
if err != nil {
	log.Fatal(err)
}

// route.Provider tells your application which client to use;
// route.Model, route.APIKey, and route.BaseURL (if set) configure it.
```

None of these routes set `BaseURL`, so each client falls back to its provider's normal default endpoint.

### Custom or Self-Hosted Endpoints

Set `BaseURL` to point a route at anything else: a self-hosted model server, an internal proxy, or any other OpenAI-compatible endpoint.

```go
router.Add(llmrouter.Route{
	ID:       "internal-proxy",
	Provider: "openai", // most self-hosted/proxy servers speak the OpenAI-compatible API
	Model:    "llama-3.3-70b",
	APIKey:   os.Getenv("INTERNAL_PROXY_KEY"),
	BaseURL:  "https://llm-proxy.internal.example.com/v1",
})
```

Routing across a mix like this works the same as routing across multiple keys for one provider — pick a strategy, add routes, call `Next()`. Enable/Disable lets you take a specific provider out of rotation (e.g., during an outage) without removing its route.

---

## Routing Strategies

### Round Robin

Selects eligible routes in round-robin sequence. Guarantees fair selection over $N \times k$ calls across $N$ eligible routes under concurrent load, provided the set of eligible routes stays the same for the duration of those calls.

```go
router := llmrouter.New(
	llmrouter.WithStrategy(llmrouter.RoundRobin()),
)
```

*Note: If no strategy is specified, `RoundRobin()` is used by default.*

### Random

Selects eligible routes randomly. Uses `math/rand/v2` from Go 1.22+, ensuring thread-safe selection without global lock contention.

```go
router := llmrouter.New(
	llmrouter.WithStrategy(llmrouter.Random()),
)
```

### Weighted

Selects routes proportionally according to their `Weight`. A route with weight `3` receives approximately three times the traffic of a route with weight `1`. Routes with weight `0` are ignored. `Weight` defaults to `0`, so it must be set explicitly on every route you want selectable when using this strategy.

```go
router := llmrouter.New(
	llmrouter.WithStrategy(llmrouter.Weighted()),
)

router.Add(llmrouter.Route{
	ID:       "high-capacity",
	Provider: "groq",
	Model:    "llama-3.3-70b",
	APIKey:   os.Getenv("GROQ_KEY_1"),
	Weight:   3,
})

router.Add(llmrouter.Route{
	ID:       "low-capacity",
	Provider: "groq",
	Model:    "llama-3.3-70b",
	APIKey:   os.Getenv("GROQ_KEY_2"),
	Weight:   1,
})
```

---

## Custom Strategies

Applications can implement custom routing strategies by satisfying the `Strategy` interface:

```go
type Strategy interface {
	Next(routes []Route) (Route, error)
}
```

### Example: Priority Fallback Strategy

```go
type PriorityStrategy struct{}

func (p *PriorityStrategy) Next(routes []llmrouter.Route) (llmrouter.Route, error) {
	if len(routes) == 0 {
		return llmrouter.Route{}, llmrouter.ErrNoRoutes
	}
	// Always select the first eligible route in the list
	return routes[0], nil
}

// Usage:
router := llmrouter.New(
	llmrouter.WithStrategy(&PriorityStrategy{}),
)
```

---

## Error Handling

Standard sentinel errors are exported for actionable conditions:

```go
route, err := router.Next()
if err != nil {
	switch {
	case errors.Is(err, llmrouter.ErrNoRoutes):
		// All routes are disabled or none registered
	default:
		// Unexpected error
	}
}
```

* `ErrNoRoutes`: No eligible routes are available for selection.
* `ErrNotFound`: Specified route ID does not exist on `Remove`, `Enable`, or `Disable`.
* `ErrDuplicateRoute`: Route ID already registered on `Add`.
* `ErrInvalidRoute`: Invalid configuration (empty ID or negative weight) on `Add`.

---

## Concurrency & Performance

`llm-router` is completely safe for concurrent access across goroutines:

* Router operations are synchronized using standard sync primitives (`sync.RWMutex`, `sync.Mutex`, or `sync/atomic`).
* `Next()` returns a value copy of `Route`, ensuring callers cannot mutate internal router state.
* The test suite enforces race-free execution with `go test -race ./...`.

---

## Future Scope & Possible Extensions

The core library intentionally focuses on fast, reliable, zero-dependency route selection. Higher-level extensions may be explored as optional packages in the future:

* Automatic cooldown managers (temporary exclusion after HTTP 429 rate limits)
* Request-time metadata filtering (tags, capability matching)
* Higher-level framework adapters (such as LangChain-Go integration)
* Latency and token accounting telemetry hooks

---

## Versioning

`llm-router` follows [Semantic Versioning](https://semver.org/) and stays on `v0.x` while the public API settles — minor versions may include breaking refinements during this stage if clearly justified. Once `v1.0.0` is tagged, backward compatibility is strictly maintained; any breaking change after that requires a new major version and module path (e.g. `github.com/devravik/llm-router/v2`).

---

## License

MIT License. See [LICENSE](LICENSE) for details.
