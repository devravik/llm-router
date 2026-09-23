package main

import (
	"context"
	"fmt"
	"log"
	"os"

	llmrouter "github.com/devravik/llm-router"
)

// This recipe demonstrates routing across multiple heterogeneous LLM providers
// (OpenAI, Anthropic, Gemini, Groq, DeepSeek, OpenRouter) and dispatching
// the request to the correct provider client/endpoint.

type ClientDispatcher struct {
	router *llmrouter.Router
}

func main() {
	router := llmrouter.New(
		llmrouter.WithStrategy(llmrouter.RoundRobin()),
	)

	routes := []llmrouter.Route{
		{
			ID:       "groq-llama",
			Provider: "groq",
			Model:    "llama-3.3-70b-versatile",
			APIKey:   getEnvOrDefault("GROQ_API_KEY", "gsk-mock"),
		},
		{
			ID:       "openai-gpt4o",
			Provider: "openai",
			Model:    "gpt-4o-mini",
			APIKey:   getEnvOrDefault("OPENAI_API_KEY", "sk-mock"),
		},
		{
			ID:       "anthropic-claude",
			Provider: "anthropic",
			Model:    "claude-3-5-sonnet",
			APIKey:   getEnvOrDefault("ANTHROPIC_API_KEY", "sk-ant-mock"),
		},
		{
			ID:       "deepseek-chat",
			Provider: "deepseek",
			Model:    "deepseek-chat",
			APIKey:   getEnvOrDefault("DEEPSEEK_API_KEY", "sk-ds-mock"),
		},
		{
			ID:       "openrouter-mistral",
			Provider: "openrouter",
			Model:    "mistralai/mistral-large-2407",
			APIKey:   getEnvOrDefault("OPENROUTER_API_KEY", "sk-or-mock"),
		},
	}

	for _, r := range routes {
		if err := router.Add(r); err != nil {
			log.Fatalf("failed to add route %s: %v", r.ID, err)
		}
	}

	dispatcher := &ClientDispatcher{router: router}

	fmt.Println("Dispatching 5 requests across heterogeneous providers:")
	for i := 1; i <= 5; i++ {
		resp, err := dispatcher.SendPrompt(context.Background(), "Summarize key Go concurrency patterns.")
		if err != nil {
			log.Printf("Request #%d failed: %v", i, err)
			continue
		}
		fmt.Printf("Request #%d: %s\n", i, resp)
	}
}

func (d *ClientDispatcher) SendPrompt(_ context.Context, prompt string) (string, error) {
	route, err := d.router.Next()
	if err != nil {
		return "", err
	}

	// Dispatch by Provider metadata
	switch route.Provider {
	case "openai", "groq", "deepseek", "openrouter":
		// All OpenAI-compatible APIs share a common request schema and bearer auth
		return fmt.Sprintf("[Dispatched to %s API (%s) using model %s]: handled OpenAI-compatible prompt '%s'",
			route.Provider, route.ID, route.Model, prompt), nil

	case "anthropic":
		// Anthropic uses x-api-key header and /v1/messages endpoint
		return fmt.Sprintf("[Dispatched to Anthropic API (%s) using model %s]: handled Messages API prompt '%s'",
			route.ID, route.Model, prompt), nil

	default:
		return "", fmt.Errorf("unsupported provider: %s", route.Provider)
	}
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
