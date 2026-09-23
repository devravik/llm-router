package llmrouter_test

import (
	"errors"
	"sync"
	"testing"

	llmrouter "github.com/devravik/llm-router"
)

func TestRoundRobin_Sequence(t *testing.T) {
	router := llmrouter.New(llmrouter.WithStrategy(llmrouter.RoundRobin()))

	for _, id := range []string{"a", "b", "c"} {
		if err := router.Add(llmrouter.Route{ID: id, Provider: "p", Model: "m", APIKey: "test-key"}); err != nil {
			t.Fatalf("Add(%s) failed: %v", id, err)
		}
	}

	want := []string{"a", "b", "c", "a", "b", "c", "a"}
	for i, id := range want {
		route, err := router.Next()
		if err != nil {
			t.Fatalf("Next() call %d failed: %v", i, err)
		}
		if route.ID != id {
			t.Fatalf("Next() call %d = %q, want %q", i, route.ID, id)
		}
	}
}

func TestRoundRobin_FairnessUnderConcurrency(t *testing.T) {
	router := llmrouter.New(llmrouter.WithStrategy(llmrouter.RoundRobin()))

	const routeCount = 5
	ids := make([]string, routeCount)
	for i := range ids {
		ids[i] = string(rune('a' + i))
		if err := router.Add(llmrouter.Route{ID: ids[i], Provider: "p", Model: "m", APIKey: "test-key"}); err != nil {
			t.Fatalf("Add(%s) failed: %v", ids[i], err)
		}
	}

	const goroutines = 50
	const callsPerGoroutine = 1000

	counts := make(map[string]int)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < callsPerGoroutine; i++ {
				route, err := router.Next()
				if err != nil {
					t.Errorf("Next() failed: %v", err)
					return
				}
				mu.Lock()
				counts[route.ID]++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// Each Next() call claims a unique counter value, so distribution
	// across N routes over a multiple of N calls is exact, not just
	// statistically close. Checking every known ID (not just the ones
	// that show up in counts) also catches a route that was never
	// selected at all.
	want := (goroutines * callsPerGoroutine) / routeCount
	for _, id := range ids {
		if counts[id] != want {
			t.Errorf("route %q selected %d times, want exactly %d", id, counts[id], want)
		}
	}
}

// TestRoundRobin_ConcurrentEnableDisable exercises Next() racing against
// concurrent Enable/Disable calls. It makes no distributional assertions;
// its purpose is to be run with -race and confirm neither operation
// corrupts router state under contention.
func TestRoundRobin_ConcurrentEnableDisable(t *testing.T) {
	router := llmrouter.New(llmrouter.WithStrategy(llmrouter.RoundRobin()))

	for i := 0; i < 5; i++ {
		id := string(rune('a' + i))
		if err := router.Add(llmrouter.Route{ID: id, Provider: "p", Model: "m", APIKey: "test-key"}); err != nil {
			t.Fatalf("Add(%s) failed: %v", id, err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			_, err := router.Next()
			if err != nil && !errors.Is(err, llmrouter.ErrNoRoutes) {
				t.Errorf("Next() unexpected error: %v", err)
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			_ = router.Disable("a")
			_ = router.Enable("a")
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			_ = router.Disable("b")
			_ = router.Enable("b")
		}
	}()

	wg.Wait()
}
