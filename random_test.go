package llmrouter_test

import (
	"errors"
	"sync"
	"testing"

	llmrouter "github.com/devravik/llm-router"
)

func TestRandom_NoRoutes(t *testing.T) {
	router := llmrouter.New(llmrouter.WithStrategy(llmrouter.Random()))

	_, err := router.Next()
	if !errors.Is(err, llmrouter.ErrNoRoutes) {
		t.Fatalf("Next() error = %v, want ErrNoRoutes", err)
	}
}

func TestRandom_DistributionSpread(t *testing.T) {
	router := llmrouter.New(llmrouter.WithStrategy(llmrouter.Random()))

	ids := []string{"a", "b", "c", "d"}
	for _, id := range ids {
		if err := router.Add(llmrouter.Route{ID: id, Provider: "p", Model: "m", APIKey: "test-key"}); err != nil {
			t.Fatalf("Add(%s) failed: %v", id, err)
		}
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

	want := trials / len(ids)
	tolerance := want / 5 // allow 20% deviation; generous enough to avoid flakes
	for _, id := range ids {
		count := counts[id]
		if count < want-tolerance || count > want+tolerance {
			t.Errorf("route %q selected %d times, want within %d of %d", id, count, tolerance, want)
		}
	}
}

func TestRandom_ConcurrentSafety(t *testing.T) {
	router := llmrouter.New(llmrouter.WithStrategy(llmrouter.Random()))

	for _, id := range []string{"a", "b", "c"} {
		if err := router.Add(llmrouter.Route{ID: id, Provider: "p", Model: "m", APIKey: "test-key"}); err != nil {
			t.Fatalf("Add(%s) failed: %v", id, err)
		}
	}

	var wg sync.WaitGroup
	for g := 0; g < 50; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				if _, err := router.Next(); err != nil {
					t.Errorf("Next() failed: %v", err)
				}
			}
		}()
	}
	wg.Wait()
}
