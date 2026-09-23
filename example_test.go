package llmrouter_test

import (
	"fmt"
	"log"

	llmrouter "github.com/devravik/llm-router"
)

func ExampleRouter_Next() {
	router := llmrouter.New(
		llmrouter.WithStrategy(llmrouter.RoundRobin()),
	)

	if err := router.Add(llmrouter.Route{
		ID:       "groq-1",
		Provider: "groq",
		Model:    "llama-3.3-70b",
		APIKey:   "test-key",
	}); err != nil {
		log.Fatal(err)
	}

	route, err := router.Next()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(route.ID)
	// Output: groq-1
}

func ExampleRoundRobin() {
	router := llmrouter.New(
		llmrouter.WithStrategy(llmrouter.RoundRobin()),
	)

	for _, id := range []string{"groq-1", "groq-2"} {
		if err := router.Add(llmrouter.Route{
			ID:       id,
			Provider: "groq",
			Model:    "llama-3.3-70b",
			APIKey:   "test-key",
		}); err != nil {
			log.Fatal(err)
		}
	}

	for i := 0; i < 4; i++ {
		route, err := router.Next()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(route.ID)
	}
	// Output:
	// groq-1
	// groq-2
	// groq-1
	// groq-2
}

// Example_customStrategy uses firstEligible, defined in
// custom_strategy_test.go, to demonstrate implementing a custom Strategy.
func Example_customStrategy() {
	router := llmrouter.New(
		llmrouter.WithStrategy(firstEligible{}),
	)

	if err := router.Add(llmrouter.Route{
		ID:       "primary",
		Provider: "groq",
		Model:    "llama-3.3-70b",
		APIKey:   "test-key",
	}); err != nil {
		log.Fatal(err)
	}

	route, err := router.Next()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(route.ID)
	// Output: primary
}
