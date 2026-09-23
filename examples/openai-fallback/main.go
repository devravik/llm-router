package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	llmrouter "github.com/devravik/llm-router"
)

// PriorityFallbackStrategy selects the first eligible route in order.
// When an error occurs with a route, the caller disables it, causing the
// next call to select the secondary route.
type PriorityFallbackStrategy struct{}

func (s *PriorityFallbackStrategy) Next(routes []llmrouter.Route) (llmrouter.Route, error) {
	if len(routes) == 0 {
		return llmrouter.Route{}, llmrouter.ErrNoRoutes
	}
	return routes[0], nil
}

func main() {
	router := llmrouter.New(
		llmrouter.WithStrategy(&PriorityFallbackStrategy{}),
	)

	// Primary provider: OpenAI
	_ = router.Add(llmrouter.Route{
		ID:       "openai-primary",
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   getEnvOrDefault("OPENAI_API_KEY", "sk-mock-openai-key"),
	})

	// Fallback provider: Anthropic
	_ = router.Add(llmrouter.Route{
		ID:       "anthropic-fallback",
		Provider: "anthropic",
		Model:    "claude-3-5-sonnet-20241022",
		APIKey:   getEnvOrDefault("ANTHROPIC_API_KEY", "sk-mock-anthropic-key"),
	})

	// Execute with automatic fallback logic
	resp, err := executeWithFallback(context.Background(), router, "Explain Go channels concisely.")
	if err != nil {
		log.Fatalf("Request failed across all providers: %v", err)
	}

	fmt.Printf("Final Response: %s\n", resp)
}

func executeWithFallback(ctx context.Context, router *llmrouter.Router, prompt string) (string, error) {
	// Attempt routing until a route succeeds or routes are exhausted
	for {
		route, err := router.Next()
		if err != nil {
			if errors.Is(err, llmrouter.ErrNoRoutes) {
				return "", fmt.Errorf("all available LLM routes exhausted")
			}
			return "", err
		}

		fmt.Printf("Attempting request via: %s (%s / %s)\n", route.ID, route.Provider, route.Model)

		// Simulate client call (e.g. primary provider simulated 500/rate-limit error)
		resp, err := mockProviderCall(ctx, route, prompt)
		if err == nil {
			return resp, nil
		}

		fmt.Printf("Route %s failed: %v. Disabling and trying next eligible route...\n", route.ID, err)
		// Disable failing route so Next() selects the next priority fallback
		_ = router.Disable(route.ID)
	}
}

func mockProviderCall(_ context.Context, route llmrouter.Route, _ string) (string, error) {
	// Simulate OpenAI failing (outage/rate limit) and Anthropic succeeding
	if route.Provider == "openai" {
		return "", errors.New("upstream HTTP 503: OpenAI Service Unavailable")
	}
	return "Go channels provide typed, thread-safe communication and synchronization between goroutines.", nil
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
