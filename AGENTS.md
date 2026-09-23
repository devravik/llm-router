# AGENTS.md

## Project & Purpose

`llm-router` is a lightweight, provider-agnostic Go library for distributing LLM requests across multiple configured API keys and providers.

The package exists to solve one specific problem:

> Allow a Go application to use multiple LLM API keys efficiently without manually switching between keys.

Scope boundaries:
* The router selects an eligible route.
* The router does **not** make HTTP requests, manage conversations, wrap provider SDKs, or persist state.
* Avoid speculative infrastructure: no databases, HTTP proxies, caching layers, token accounting systems, or agent frameworks.

Prioritize engineering quality and simplicity over feature count. The library should look and feel like it was written and maintained by an experienced Go developer.

---

## Go Version & Dependencies

* Target **Go 1.22 or later**.
* Go 1.22 introduces `math/rand/v2`, whose top-level functions (such as `rand.IntN`) are safe for concurrent use without global mutex contention, making it ideal for the `Random()` routing strategy.
* Rely exclusively on the Go standard library. Avoid external dependencies unless standard library support is genuinely impossible. Keep `go.mod` minimal.

---

## Versioning & Compatibility

* Follow Semantic Versioning (SemVer).
* Stay on `v0.x` (`v0.1.0`) while the public API settles. Minor versions may introduce breaking refinements during this stage if clearly justified.
* Once `v1.0.0` is tagged, backward compatibility is strictly maintained. Any subsequent breaking change requires a new major version and module path increment (e.g., `github.com/devravik/llm-router/v2`).

---

## Public API Design & Decisions

Keep the exported surface minimal, unsurprising, and idiomatic. Every exported type, function, and method is a long-term commitment.

```go
type Route struct {
    ID       string
    Provider string
    Model    string
    APIKey   string
    BaseURL  string
    Weight   int
}
```

`BaseURL` is optional and opaque to the router, exactly like `APIKey`: empty means "use `Provider`'s usual endpoint," non-empty means "use this URL instead" (self-hosted, proxied, or any other custom OpenAI-compatible endpoint). The router never inspects or validates it.

### Core API Rules & Decisions

1. **`Add(Route) error`**:
   * Empty `ID` is rejected with an error (`ErrInvalidRoute`).
   * Duplicate `ID` is rejected with an error (`ErrDuplicateRoute`).
   * Negative `Weight` is rejected with an error (`ErrInvalidRoute`). Zero weight is permitted, but the route will never be selected by `Weighted()`. All-zero weights across eligible routes cannot be detected at registration; it must be detected and rejected at `Next()` (returns `ErrNoRoutes`).
2. **`Remove(id string) error`**:
   * Removes a configured route by ID.
   * Returns `ErrNotFound` if the route does not exist.
3. **`Enable(id string) error` and `Disable(id string) error`**:
   * Toggles route eligibility.
   * Returns `ErrNotFound` if the route ID is unknown.
4. **Default Strategy**:
   * If no strategy option is passed to `New()`, the router defaults to `RoundRobin()`.
5. **Defensive Copies**:
   * `Next()` returns a copy of `Route` (by value), ensuring callers cannot mutate the router's internal state.

---

## Routing Strategies

A strategy answers exactly one question:

> Which eligible route should be selected next?

Strategies do not handle HTTP calls, retries, cooldowns, or route enablement.

### Strategy Interface

```go
type Strategy interface {
    Next(routes []Route) (Route, error)
}
```

### Predefined Strategies

Predefined strategies must be constructor functions returning fresh strategy instances, **never** package-level values:

* `RoundRobin()`: stateful sequential selection.
* `Random()`: random selection via `math/rand/v2`.
* `Weighted()`: weighted selection based on `Route.Weight`.

Because round robin maintains an internal counter, using a constructor ensures that multiple router instances never inadvertently share selection state.

Example usage:

```go
router := llmrouter.New(
    llmrouter.WithStrategy(llmrouter.RoundRobin()),
)

err := router.Add(llmrouter.Route{
    ID:       "groq-1",
    Provider: "groq",
    Model:    "llama-3.3-70b",
    APIKey:   os.Getenv("GROQ_KEY_1"),
})
if err != nil {
    return err
}

route, err := router.Next()
```

---

## Concurrency & Round Robin Semantics

The router must be completely safe for concurrent use across multiple goroutines.

* Protect mutable state with appropriate primitives (`sync.RWMutex`, `sync.Mutex`, or `sync/atomic`).
* Run all concurrency tests with `go test -race ./...`.

### Round Robin Semantics

* **Fairness under concurrency**: Under high concurrent load from multiple goroutines, deterministic execution order cannot be guaranteed due to runtime goroutine scheduling. What `RoundRobin()` guarantees is **fairness**: over $N \times k$ calls across $N$ eligible routes, each route is selected exactly $k$ times.
* **Dynamic membership transitions**: When routes are added, removed, enabled, or disabled concurrently, counter stepping modulo the updated eligible route count may cause minor skipping or repetition during the transition. This is acceptable and expected.

---

## Error Handling

Errors must be informative and testable:

* Use sentinel errors for actionable conditions:
  * `ErrNoRoutes`: no eligible routes are available.
  * `ErrNotFound`: specified route ID does not exist.
  * `ErrDuplicateRoute`: route ID already registered.
  * `ErrInvalidRoute`: invalid route configuration (e.g., empty ID, negative weight).
* Callers should be able to inspect errors using `errors.Is(err, target)`.
* Do not expose credentials or internal pointers in error strings.

---

## Documentation Standards

Documentation is part of the API contract:

* **`doc.go`**: Maintain a package-level comment in `doc.go` that outlines the package purpose, architecture, and primary workflows.
* **Runnable Examples (`example_test.go`)**: Provide runnable `Example...` functions (e.g., `ExampleRouter_Next`, `ExampleRoundRobin`, `Example_customStrategy`). These are compiled and executed by `go test` and rendered interactively on `pkg.go.dev`, ensuring examples never become outdated or uncompilable.
* **Code comments**: Explain **why**, not **what**. Avoid narrating obvious code (e.g., `// loop through routes`).

---

## Secrets & Security

* API keys are sensitive credentials owned by the caller.
* Never print, log, serialize, or include API keys in error messages, test outputs, or debug strings.
* The library must never persist API keys to disk.
* Test code and documentation examples must only use obvious mock values (e.g., `"test-key"`, `"fake-key"`).

---

## Code Style & Simplicity

* Write idiomatic, straightforward Go.
* Prefer simple functions and early returns (`if err != nil { return err }`).
* Avoid "AI slop": no boilerplate generators, unnecessary interfaces, speculative configuration fields, or enterprise design patterns (no `RouteManagerFactory`, `StrategyOrchestrator`, etc.).
* If a 30-line implementation meets requirements cleanly, do not replace it with a 150-line abstraction.

---

## Testing & Verification

Tests must communicate expected behavior clearly. Prefer table-driven tests.

Test matrix must cover:
* Empty router and no eligible routes (`ErrNoRoutes`)
* Single route and multiple routes
* Round-robin (sequence verification, fairness over $N \times k$ calls under concurrency, behavior during dynamic route addition/removal)
* Random strategy (distribution spread, concurrent safety)
* Weighted strategy (selection proportionality, zero-weight exclusion, all-zero weights rejection at `Next()`)
* Route lifecycle (`Add`, `Remove`, `Enable`, `Disable`, and corresponding error cases)
* Validation on `Add` (empty IDs, duplicate IDs, negative weights)
* Race detector clean: `go test -race ./...`
* Benchmarks for core selection methods (`BenchmarkRoundRobin`, `BenchmarkRandom`, `BenchmarkWeighted`)

### Continuous Integration

CI should run `go test ./...`, `go test -race ./...`, and `go vet ./...` on every pull request.

---

## Definition of Done

A task, feature, or PR is complete only when:

* [ ] Public API is minimal, idiomatic, and adheres to all API decisions.
* [ ] Concurrency is safe and verified with `go test -race ./...`.
* [ ] Package documentation in `doc.go` and runnable examples in `example_test.go` compile and pass.
* [ ] All tests pass without flakes or race conditions.
* [ ] CI checks (`go test`, `go test -race`, `go vet`) pass cleanly with zero warnings.
* [ ] `gofmt` produces zero diffs.
* [ ] README and examples accurately reflect working code.
* [ ] No credentials or sensitive data are leaked in code, errors, or tests.
* [ ] The implementation is simple, clear, and comfortable for an experienced Go engineer to maintain.
