package llmrouter_test

import (
	"errors"
	"testing"

	llmrouter "github.com/devravik/llm-router"
)

func TestRouter_NextEmptyRouter(t *testing.T) {
	router := llmrouter.New()

	_, err := router.Next()
	if !errors.Is(err, llmrouter.ErrNoRoutes) {
		t.Fatalf("Next() error = %v, want ErrNoRoutes", err)
	}
}

func TestRouter_NextSingleRoute(t *testing.T) {
	router := llmrouter.New()

	if err := router.Add(llmrouter.Route{ID: "a", Provider: "p", Model: "m", APIKey: "test-key"}); err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	route, err := router.Next()
	if err != nil {
		t.Fatalf("Next() failed: %v", err)
	}
	if route.ID != "a" {
		t.Fatalf("Next().ID = %q, want %q", route.ID, "a")
	}
}

func TestRouter_NextMultipleRoutesDefaultsToRoundRobin(t *testing.T) {
	router := llmrouter.New()

	for _, id := range []string{"a", "b", "c"} {
		if err := router.Add(llmrouter.Route{ID: id, Provider: "p", Model: "m", APIKey: "test-key"}); err != nil {
			t.Fatalf("Add(%s) failed: %v", id, err)
		}
	}

	want := []string{"a", "b", "c", "a", "b", "c"}
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

func TestRouter_NextPreservesBaseURL(t *testing.T) {
	router := llmrouter.New()

	if err := router.Add(llmrouter.Route{
		ID:       "custom-proxy",
		Provider: "custom",
		Model:    "my-model",
		APIKey:   "test-key",
		BaseURL:  "https://proxy.example.com/v1",
	}); err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	route, err := router.Next()
	if err != nil {
		t.Fatalf("Next() failed: %v", err)
	}
	if route.BaseURL != "https://proxy.example.com/v1" {
		t.Fatalf("Next().BaseURL = %q, want %q", route.BaseURL, "https://proxy.example.com/v1")
	}
}

func TestRouter_AddValidation(t *testing.T) {
	tests := []struct {
		name    string
		route   llmrouter.Route
		wantErr error
	}{
		{
			name:    "empty ID",
			route:   llmrouter.Route{ID: "", Provider: "p", Model: "m", APIKey: "test-key"},
			wantErr: llmrouter.ErrInvalidRoute,
		},
		{
			name:    "negative weight",
			route:   llmrouter.Route{ID: "a", Provider: "p", Model: "m", APIKey: "test-key", Weight: -1},
			wantErr: llmrouter.ErrInvalidRoute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := llmrouter.New()

			err := router.Add(tt.route)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Add() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRouter_AddDuplicateID(t *testing.T) {
	router := llmrouter.New()

	route := llmrouter.Route{ID: "a", Provider: "p", Model: "m", APIKey: "test-key"}
	if err := router.Add(route); err != nil {
		t.Fatalf("first Add() failed: %v", err)
	}

	err := router.Add(route)
	if !errors.Is(err, llmrouter.ErrDuplicateRoute) {
		t.Fatalf("second Add() error = %v, want ErrDuplicateRoute", err)
	}
}

func TestRouter_RemoveUnknownID(t *testing.T) {
	router := llmrouter.New()

	err := router.Remove("missing")
	if !errors.Is(err, llmrouter.ErrNotFound) {
		t.Fatalf("Remove() error = %v, want ErrNotFound", err)
	}
}

func TestRouter_Remove(t *testing.T) {
	router := llmrouter.New()

	if err := router.Add(llmrouter.Route{ID: "a", Provider: "p", Model: "m", APIKey: "test-key"}); err != nil {
		t.Fatalf("Add() failed: %v", err)
	}
	if err := router.Remove("a"); err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}

	_, err := router.Next()
	if !errors.Is(err, llmrouter.ErrNoRoutes) {
		t.Fatalf("Next() error = %v, want ErrNoRoutes", err)
	}
}

func TestRouter_RemoveMiddleRoute(t *testing.T) {
	router := llmrouter.New()

	for _, id := range []string{"a", "b", "c", "d"} {
		if err := router.Add(llmrouter.Route{ID: id, Provider: "p", Model: "m", APIKey: "test-key"}); err != nil {
			t.Fatalf("Add(%s) failed: %v", id, err)
		}
	}

	if err := router.Remove("b"); err != nil {
		t.Fatalf("Remove(b) failed: %v", err)
	}
	if err := router.Disable("d"); err != nil {
		t.Fatalf("Disable(d) failed: %v", err)
	}

	want := []string{"a", "c", "a", "c"}
	for i, id := range want {
		route, err := router.Next()
		if err != nil {
			t.Fatalf("Next() call %d failed: %v", i, err)
		}
		if route.ID != id {
			t.Fatalf("Next() call %d = %q, want %q", i, route.ID, id)
		}
	}

	if err := router.Enable("d"); err != nil {
		t.Fatalf("Enable(d) failed: %v", err)
	}

	// A full cycle over the three eligible routes visits each exactly
	// once, regardless of where the round-robin counter currently sits.
	seen := make(map[string]bool)
	for i := 0; i < 3; i++ {
		route, err := router.Next()
		if err != nil {
			t.Fatalf("Next() call %d after Enable(d) failed: %v", i, err)
		}
		seen[route.ID] = true
	}
	for _, id := range []string{"a", "c", "d"} {
		if !seen[id] {
			t.Fatalf("Next() after Enable(d) never selected %q, seen = %v", id, seen)
		}
	}

	if err := router.Remove("d"); err != nil {
		t.Fatalf("Remove(d) after re-enable failed: %v", err)
	}
}

func TestRouter_EnableDisableUnknownID(t *testing.T) {
	router := llmrouter.New()

	if err := router.Enable("missing"); !errors.Is(err, llmrouter.ErrNotFound) {
		t.Fatalf("Enable() error = %v, want ErrNotFound", err)
	}
	if err := router.Disable("missing"); !errors.Is(err, llmrouter.ErrNotFound) {
		t.Fatalf("Disable() error = %v, want ErrNotFound", err)
	}
}

func TestRouter_DisableExcludesRoute(t *testing.T) {
	router := llmrouter.New()

	if err := router.Add(llmrouter.Route{ID: "a", Provider: "p", Model: "m", APIKey: "test-key"}); err != nil {
		t.Fatalf("Add() failed: %v", err)
	}
	if err := router.Disable("a"); err != nil {
		t.Fatalf("Disable() failed: %v", err)
	}

	_, err := router.Next()
	if !errors.Is(err, llmrouter.ErrNoRoutes) {
		t.Fatalf("Next() error = %v, want ErrNoRoutes", err)
	}
}

func TestRouter_EnableRestoresRoute(t *testing.T) {
	router := llmrouter.New()

	if err := router.Add(llmrouter.Route{ID: "a", Provider: "p", Model: "m", APIKey: "test-key"}); err != nil {
		t.Fatalf("Add() failed: %v", err)
	}
	if err := router.Disable("a"); err != nil {
		t.Fatalf("Disable() failed: %v", err)
	}
	if err := router.Enable("a"); err != nil {
		t.Fatalf("Enable() failed: %v", err)
	}

	route, err := router.Next()
	if err != nil {
		t.Fatalf("Next() failed: %v", err)
	}
	if route.ID != "a" {
		t.Fatalf("Next().ID = %q, want %q", route.ID, "a")
	}
}
