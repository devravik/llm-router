// Package llmrouter distributes LLM requests across multiple configured
// API keys and providers.
//
// A Router holds a set of Route values -- each a provider, model, and API
// key combination -- and selects one according to a Strategy each time
// Next is called. The router only selects a route; it does not make the
// HTTP request, manage conversations, or wrap a provider SDK.
//
// # Basic usage
//
//	router := llmrouter.New(
//		llmrouter.WithStrategy(llmrouter.RoundRobin()),
//	)
//
//	router.Add(llmrouter.Route{
//		ID:       "groq-1",
//		Provider: "groq",
//		Model:    "llama-3.3-70b",
//		APIKey:   os.Getenv("GROQ_KEY_1"),
//	})
//
//	route, err := router.Next()
//
// # Strategies
//
// RoundRobin, Random, and Weighted are the predefined strategies. Strategy
// is a single-method interface, so applications can implement their own
// selection logic without modifying this package.
package llmrouter
