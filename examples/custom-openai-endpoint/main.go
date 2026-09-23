package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	llmrouter "github.com/devravik/llm-router"
)

// This recipe demonstrates routing between an on-premise / self-hosted endpoint
// (such as local Ollama, vLLM, or LM Studio) and cloud endpoints using route.BaseURL.

func main() {
	router := llmrouter.New(
		llmrouter.WithStrategy(llmrouter.RoundRobin()),
	)

	// Route 1: Local Ollama instance (OpenAI-compatible endpoint at /v1)
	err := router.Add(llmrouter.Route{
		ID:       "local-ollama",
		Provider: "openai", // Speaks standard OpenAI chat completions format
		Model:    "llama3:8b",
		APIKey:   "ollama", // Ollama doesn't require a real key, but opaque token is fine
		BaseURL:  getEnvOrDefault("OLLAMA_BASE_URL", "http://localhost:11434/v1"),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Route 2: Self-hosted vLLM or internal corporate proxy
	err = router.Add(llmrouter.Route{
		ID:       "corp-vllm-proxy",
		Provider: "openai",
		Model:    "mistral-7b-instruct",
		APIKey:   getEnvOrDefault("INTERNAL_PROXY_KEY", "proxy-internal-token"),
		BaseURL:  "https://llm-proxy.corp.internal/v1",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Route 3: Public OpenAI cloud fallback (BaseURL is empty, so default api.openai.com is used)
	err = router.Add(llmrouter.Route{
		ID:       "openai-cloud-fallback",
		Provider: "openai",
		Model:    "gpt-4o-mini",
		APIKey:   getEnvOrDefault("OPENAI_API_KEY", "sk-mock-key"),
		BaseURL:  "", // Empty means standard provider endpoint
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Configured 3 routes with custom BaseURLs:")
	for i := 1; i <= 3; i++ {
		route, err := router.Next()
		if err != nil {
			log.Fatal(err)
		}

		resolvedURL := resolveEndpoint(route)
		fmt.Printf("Route #%d -> ID: %-22s Endpoint: %s (Model: %s)\n",
			i, route.ID, resolvedURL, route.Model)
	}
}

// resolveEndpoint uses route.BaseURL if present, falling back to provider standard URL
func resolveEndpoint(route llmrouter.Route) string {
	baseURL := strings.TrimRight(route.BaseURL, "/")
	if baseURL != "" {
		return baseURL + "/chat/completions"
	}

	switch route.Provider {
	case "openai":
		return "https://api.openai.com/v1/chat/completions"
	case "groq":
		return "https://api.groq.com/openai/v1/chat/completions"
	default:
		return "https://api.openai.com/v1/chat/completions"
	}
}

func sendRequest(ctx context.Context, route llmrouter.Route, prompt string) (string, error) {
	endpoint := resolveEndpoint(route)

	reqBody, _ := json.Marshal(map[string]any{
		"model": route.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}

	if route.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+route.APIKey)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
