package llmrouter_test

import (
	"errors"
	"testing"

	llmrouter "github.com/devravik/llm-router"
)

func TestWeighted_NoRoutes(t *testing.T) {
	router := llmrouter.New(llmrouter.WithStrategy(llmrouter.Weighted()))

	_, err := router.Next()
	if !errors.Is(err, llmrouter.ErrNoRoutes) {
		t.Fatalf("Next() error = %v, want ErrNoRoutes", err)
	}
}

func TestWeighted_AllZeroWeightsRejected(t *testing.T) {
	router := llmrouter.New(llmrouter.WithStrategy(llmrouter.Weighted()))

	for _, id := range []string{"a", "b"} {
		if err := router.Add(llmrouter.Route{ID: id, Provider: "p", Model: "m", APIKey: "test-key", Weight: 0}); err != nil {
			t.Fatalf("Add(%s) failed: %v", id, err)
		}
	}

	_, err := router.Next()
	if !errors.Is(err, llmrouter.ErrNoRoutes) {
		t.Fatalf("Next() error = %v, want ErrNoRoutes", err)
	}
}

func TestWeighted_ZeroWeightExcluded(t *testing.T) {
	router := llmrouter.New(llmrouter.WithStrategy(llmrouter.Weighted()))

	if err := router.Add(llmrouter.Route{ID: "unused", Provider: "p", Model: "m", APIKey: "test-key", Weight: 0}); err != nil {
		t.Fatalf("Add(unused) failed: %v", err)
	}
	if err := router.Add(llmrouter.Route{ID: "used", Provider: "p", Model: "m", APIKey: "test-key", Weight: 5}); err != nil {
		t.Fatalf("Add(used) failed: %v", err)
	}

	for i := 0; i < 200; i++ {
		route, err := router.Next()
		if err != nil {
			t.Fatalf("Next() failed: %v", err)
		}
		if route.ID != "used" {
			t.Fatalf("Next() = %q, want %q", route.ID, "used")
		}
	}
}

func TestWeighted_Proportionality(t *testing.T) {
	router := llmrouter.New(llmrouter.WithStrategy(llmrouter.Weighted()))

	if err := router.Add(llmrouter.Route{ID: "heavy", Provider: "p", Model: "m", APIKey: "test-key", Weight: 3}); err != nil {
		t.Fatalf("Add(heavy) failed: %v", err)
	}
	if err := router.Add(llmrouter.Route{ID: "light", Provider: "p", Model: "m", APIKey: "test-key", Weight: 1}); err != nil {
		t.Fatalf("Add(light) failed: %v", err)
	}

	const trials = 40000
	counts := make(map[string]int)
	for i := 0; i < trials; i++ {
		route, err := router.Next()
		if err != nil {
			t.Fatalf("Next() failed: %v", err)
		}
		counts[route.ID]++
	}

	wantHeavy := trials * 3 / 4
	wantLight := trials * 1 / 4
	tolerance := trials / 20 // allow 5% deviation

	if counts["heavy"] < wantHeavy-tolerance || counts["heavy"] > wantHeavy+tolerance {
		t.Errorf("heavy selected %d times, want within %d of %d", counts["heavy"], tolerance, wantHeavy)
	}
	if counts["light"] < wantLight-tolerance || counts["light"] > wantLight+tolerance {
		t.Errorf("light selected %d times, want within %d of %d", counts["light"], tolerance, wantLight)
	}
}
