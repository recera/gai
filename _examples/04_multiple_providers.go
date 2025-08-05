package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/collinshill/gai"
)

// Question structure for comparing responses
type ProviderComparison struct {
	Provider string
	Model    string
	Response string
	Duration time.Duration
	Tokens   int
}

func main() {
	// Load environment
	gai.FindAndLoadEnv()

	// Create client
	client, err := gai.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	// Test question
	question := "Explain quantum computing in one paragraph."

	// Providers to test
	providers := []struct {
		Provider string
		Model    string
	}{
		{"openai", "gpt-4o-mini"},
		{"anthropic", "claude-3-haiku-20240307"},
		{"gemini", "gemini-2.0-flash-exp"},
		{"groq", "llama-3.3-70b-versatile"},
		{"cerebras", "llama-3.3-70b"},
	}

	ctx := context.Background()
	results := make([]ProviderComparison, 0, len(providers))

	fmt.Println("Comparing LLM Providers")
	fmt.Println("======================")
	fmt.Printf("Question: %s\n\n", question)

	// Test each provider
	for _, p := range providers {
		fmt.Printf("Testing %s/%s... ", p.Provider, p.Model)
		
		start := time.Now()
		
		parts := gai.NewLLMCallParts().
			WithProvider(p.Provider).
			WithModel(p.Model).
			WithUserMessage(question).
			WithMaxTokens(150).
			WithTemperature(0.3)

		response, err := client.GetCompletion(ctx, parts)
		
		duration := time.Since(start)
		
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}
		
		fmt.Printf("✅ (%.2fs)\n", duration.Seconds())
		
		results = append(results, ProviderComparison{
			Provider: p.Provider,
			Model:    p.Model,
			Response: response.Content,
			Duration: duration,
			Tokens:   response.Usage.TotalTokens,
		})
	}

	// Display results
	fmt.Println("\nResults:")
	fmt.Println("========")
	
	for _, r := range results {
		fmt.Printf("\n%s/%s (%.2fs, %d tokens):\n", 
			r.Provider, r.Model, r.Duration.Seconds(), r.Tokens)
		fmt.Printf("%s\n", r.Response)
		fmt.Println(strings.Repeat("-", 60))
	}
}