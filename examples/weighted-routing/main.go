package main

import (
	"fmt"
	"log"
	"os"

	llmrouter "github.com/devravik/llm-router"
)

// This recipe demonstrates weighted traffic splitting (e.g., Canary testing,
// cost optimization, or quota-based distribution) between a fast/economical tier
// and a heavy/reasoning tier.

func main() {
	// Initialize router with Weighted strategy
	router := llmrouter.New(
		llmrouter.WithStrategy(llmrouter.Weighted()),
	)

	// 80% traffic to economical/fast tier (weight 8)
	err := router.Add(llmrouter.Route{
		ID:       "fast-tier-llama",
		Provider: "groq",
		Model:    "llama-3.3-70b",
		APIKey:   getEnvOrDefault("GROQ_API_KEY", "gsk-mock"),
		Weight:   8,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 20% traffic to premium/reasoning tier (weight 2)
	err = router.Add(llmrouter.Route{
		ID:       "premium-tier-gpt4o",
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   getEnvOrDefault("OPENAI_API_KEY", "sk-mock"),
		Weight:   2,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Route zero-weight route (e.g. disabled or test-only)
	err = router.Add(llmrouter.Route{
		ID:       "standby-route",
		Provider: "anthropic",
		Model:    "claude-3-opus",
		APIKey:   getEnvOrDefault("ANTHROPIC_API_KEY", "sk-mock"),
		Weight:   0, // Weight 0 routes are never selected by Weighted strategy
	})
	if err != nil {
		log.Fatal(err)
	}

	// Simulate 1,000 requests to measure distribution
	counts := make(map[string]int)
	totalRequests := 1000

	for i := 0; i < totalRequests; i++ {
		route, err := router.Next()
		if err != nil {
			log.Fatalf("Next failed: %v", err)
		}
		counts[route.ID]++
	}

	fmt.Printf("Simulated %d requests with 80/20 weights:\n", totalRequests)
	for id, count := range counts {
		pct := float64(count) / float64(totalRequests) * 100.0
		fmt.Printf(" - Route %-20s: %4d calls (%.1f%%)\n", id, count, pct)
	}
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
