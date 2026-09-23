package llmrouter_test

import (
	"testing"

	llmrouter "github.com/devravik/llm-router"
)

// firstEligible is a minimal custom Strategy: it always selects the
// first eligible route. It demonstrates that applications can satisfy
// llmrouter.Strategy without any changes to this package.
type firstEligible struct{}

func (firstEligible) Next(routes []llmrouter.Route) (llmrouter.Route, error) {
	if len(routes) == 0 {
		return llmrouter.Route{}, llmrouter.ErrNoRoutes
	}
	return routes[0], nil
}

func TestCustomStrategy(t *testing.T) {
	router := llmrouter.New(llmrouter.WithStrategy(firstEligible{}))

	if err := router.Add(llmrouter.Route{ID: "a", Provider: "p", Model: "m", APIKey: "test-key"}); err != nil {
		t.Fatalf("Add(a) failed: %v", err)
	}
	if err := router.Add(llmrouter.Route{ID: "b", Provider: "p", Model: "m", APIKey: "test-key"}); err != nil {
		t.Fatalf("Add(b) failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		route, err := router.Next()
		if err != nil {
			t.Fatalf("Next() failed: %v", err)
		}
		if route.ID != "a" {
			t.Fatalf("Next() = %q, want %q", route.ID, "a")
		}
	}
}
