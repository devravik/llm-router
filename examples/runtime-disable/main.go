package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	llmrouter "github.com/devravik/llm-router"
)

// This recipe demonstrates runtime cooldown management when hitting HTTP 429 (Rate Limited).
// When an API key hits a rate limit, the caller disables it for a cooldown period (e.g. 30s)
// using time.AfterFunc to automatically re-enable it.

type ResilientClient struct {
	router *llmrouter.Router
	mu     sync.Mutex
}

func main() {
	router := llmrouter.New(
		llmrouter.WithStrategy(llmrouter.RoundRobin()),
	)

	// Add 3 keys
	for i := 1; i <= 3; i++ {
		_ = router.Add(llmrouter.Route{
			ID:       fmt.Sprintf("provider-key-%d", i),
			Provider: "groq",
			Model:    "llama-3.3-70b",
			APIKey:   fmt.Sprintf("key-%d", i),
		})
	}

	client := &ResilientClient{router: router}

	fmt.Println("Initial rotation across 3 keys:")
	for i := 1; i <= 3; i++ {
		r, _ := router.Next()
		fmt.Printf(" - Route selected: %s\n", r.ID)
	}

	// Key 2 encounters an HTTP 429 Rate Limit!
	fmt.Println("\nSimulating HTTP 429 Rate Limit on provider-key-2:")
	client.handleRateLimit("provider-key-2", 200*time.Millisecond)

	// Calls while provider-key-2 is on cooldown
	fmt.Println("\nRouting during cooldown (key-2 should be excluded):")
	for i := 1; i <= 4; i++ {
		r, err := router.Next()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(" - Route selected: %s\n", r.ID)
	}

	// Wait for cooldown to expire and re-enable key-2
	fmt.Println("\nWaiting for cooldown to expire...")
	time.Sleep(300 * time.Millisecond)

	fmt.Println("Routing after cooldown expired (key-2 restored to rotation):")
	for i := 1; i <= 4; i++ {
		r, err := router.Next()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(" - Route selected: %s\n", r.ID)
	}
}

func (c *ResilientClient) handleRateLimit(routeID string, cooldown time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Disable the rate-limited route immediately
	if err := c.router.Disable(routeID); err != nil {
		log.Printf("Failed to disable route %s: %v", routeID, err)
		return
	}
	fmt.Printf("[Cooldown] Disabled %s for %v due to rate limit.\n", routeID, cooldown)

	// Schedule automatic re-enablement after cooldown expires
	time.AfterFunc(cooldown, func() {
		if err := c.router.Enable(routeID); err != nil {
			log.Printf("Failed to re-enable route %s: %v", routeID, err)
			return
		}
		fmt.Printf("[Cooldown Expired] Re-enabled %s in router.\n", routeID)
	})
}
