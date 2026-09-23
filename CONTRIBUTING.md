# Contributing to `llm-router`

Thank you for your interest in contributing to `llm-router`!

`llm-router` is a lightweight, provider-agnostic Go library for distributing LLM requests across multiple configured API keys and providers with **zero external dependencies**.

---

## Design Principles & Scope

Before proposing code changes, please review the scope boundaries defined in [AGENTS.md](AGENTS.md):

1. **Specific Focus**: The library solves one problem: selecting an eligible route from a configured set of API keys, models, and providers.
2. **Architectural Boundary**: The router does **not** make HTTP requests, wrap provider SDKs, manage conversations, or persist state.
3. **Zero External Dependencies**: The core package relies exclusively on the Go standard library (`go.mod` has zero external dependencies).
4. **Idiomatic Go**: Keep the public surface minimal, unsurprising, and concurrency-safe.

---

## Development Prerequisites

* **Go 1.22** or later (required for `math/rand/v2` and latest standard library capabilities).
* Git.

---

## Development Workflow

### 1. Clone & Verify

```bash
git clone https://github.com/devravik/llm-router.git
cd llm-router
go test -v -race ./...
```

### 2. Making Changes

* Write clear, idiomatic Go with simple functions and early returns.
* Ensure exported types, functions, and methods have clear GoDoc comments explaining *why*, not just *what*.
* Never log, print, or expose API keys in error strings, debug output, or tests. Always use mock keys in tests (e.g. `"mock-key"`).

### 3. Testing & Benchmarking

All changes must include tests:
* Test empty and single-route edge cases.
* Verify concurrent safety with the race detector:
  ```bash
  go test -race ./...
  ```
* Run linter / static analysis:
  ```bash
  go vet ./...
  ```
* Verify code formatting:
  ```bash
  gofmt -s -w .
  ```
* Run benchmarks if modifying selection or routing algorithms:
  ```bash
  go test -benchmem -bench=. .
  ```

---

## Submitting Pull Requests

1. Create a descriptive feature branch: `git checkout -b feature/my-feature`
2. Commit your changes with clear commit messages.
3. Push to your fork and submit a Pull Request to `main`.
4. Ensure all CI checks (unit tests, race detector, `go vet`) pass cleanly.

---

## Need Help or Have an Idea?

Please open an issue to discuss proposed features or architectural questions before investing significant implementation time.
