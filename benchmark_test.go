package llmrouter_test

import (
	"fmt"
	"testing"

	llmrouter "github.com/devravik/llm-router"
)

func benchmarkRouter(b *testing.B, strategy llmrouter.Strategy, routeCount int) *llmrouter.Router {
	b.Helper()
	router := llmrouter.New(llmrouter.WithStrategy(strategy))
	for i := 0; i < routeCount; i++ {
		err := router.Add(llmrouter.Route{
			ID:       fmt.Sprintf("route-%d", i),
			Provider: "groq",
			Model:    "llama-3.3-70b",
			APIKey:   "test-key",
			Weight:   1,
		})
		if err != nil {
			b.Fatalf("Add failed: %v", err)
		}
	}
	return router
}

func BenchmarkRoundRobin(b *testing.B) {
	router := benchmarkRouter(b, llmrouter.RoundRobin(), 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := router.Next(); err != nil {
			b.Fatalf("Next failed: %v", err)
		}
	}
}

func BenchmarkRandom(b *testing.B) {
	router := benchmarkRouter(b, llmrouter.Random(), 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := router.Next(); err != nil {
			b.Fatalf("Next failed: %v", err)
		}
	}
}

func BenchmarkWeighted(b *testing.B) {
	router := benchmarkRouter(b, llmrouter.Weighted(), 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := router.Next(); err != nil {
			b.Fatalf("Next failed: %v", err)
		}
	}
}
