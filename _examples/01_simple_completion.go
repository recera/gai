package main

import (
	"context"
	"fmt"
	"log"

	"github.com/collinshill/gai"
)

func main() {
	// Option 1: Load .env file explicitly
	if err := gai.LoadEnvFromFile(".env"); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Option 2: Find and load .env file automatically
	// if _, err := gai.FindAndLoadEnv(); err != nil {
	//     log.Printf("Warning: Could not find .env file: %v", err)
	// }

	// Create client
	client, err := gai.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	// Create a simple completion request
	ctx := context.Background()
	
	// Using fluent builder pattern
	parts := gai.NewLLMCallParts().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithSystem("You are a helpful assistant.").
		WithUserMessage("What is the capital of France?")

	// Get completion
	response, err := client.GetCompletion(ctx, parts)
	if err != nil {
		// Check if it's an LLM error with more details
		if llmErr, ok := err.(*gai.LLMError); ok {
			log.Printf("LLM Error from %s/%s: %v", 
				llmErr.Provider, llmErr.Model, llmErr.Err)
			if llmErr.StatusCode != 0 {
				log.Printf("HTTP Status: %d", llmErr.StatusCode)
			}
		}
		log.Fatal(err)
	}

	// Print response
	fmt.Println("Response:", response.Content)
	fmt.Printf("Tokens used: %d\n", response.Usage.TotalTokens)
}