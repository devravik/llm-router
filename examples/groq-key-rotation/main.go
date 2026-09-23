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
	"time"

	llmrouter "github.com/devravik/llm-router"
)

// This example demonstrates rotating multiple Groq API keys using Round Robin
// to multiply rate limits and distribute traffic fairly.

func main() {
	// Initialize router with round robin strategy
	router := llmrouter.New(
		llmrouter.WithStrategy(llmrouter.RoundRobin()),
	)

	// Fetch keys from environment, or use mock keys for demonstration
	keys := []string{
		getEnvOrDefault("GROQ_API_KEY_1", "gsk-mock-key-alpha-12345"),
		getEnvOrDefault("GROQ_API_KEY_2", "gsk-mock-key-bravo-67890"),
		getEnvOrDefault("GROQ_API_KEY_3", "gsk-mock-key-charlie-11223"),
	}

	for i, key := range keys {
		err := router.Add(llmrouter.Route{
			ID:       fmt.Sprintf("groq-key-%d", i+1),
			Provider: "groq",
			Model:    "llama-3.3-70b-versatile",
			APIKey:   key,
		})
		if err != nil {
			log.Fatalf("failed to add route %d: %v", i+1, err)
		}
	}

	fmt.Println("Registered 3 Groq API keys. Simulating request rotation:")

	// Simulate 6 consecutive requests rotating across the 3 keys
	for reqNum := 1; reqNum <= 6; reqNum++ {
		route, err := router.Next()
		if err != nil {
			log.Fatalf("no available route: %v", err)
		}

		fmt.Printf("Request #%d -> Selected Key: %s (Model: %s)\n",
			reqNum, route.ID, route.Model)

		// If real keys are present, we can call Groq's chat completion endpoint
		if os.Getenv("GROQ_API_KEY_1") != "" {
			resp, err := callGroq(context.Background(), route, "Say hello in one sentence.")
			if err != nil {
				log.Printf("Groq call failed: %v", err)
			} else {
				fmt.Printf("   Response: %s\n", resp)
			}
		}
	}
}

func callGroq(ctx context.Context, route llmrouter.Route, prompt string) (string, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"model": route.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
	})

	endpoint := "https://api.groq.com/openai/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+route.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq API status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	return result.Choices[0].Message.Content, nil
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
