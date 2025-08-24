// Package main demonstrates basic text generation with the GAI framework.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/middleware"
	"github.com/recera/gai/providers/openai"
)

func main() {
	// Check for API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set the OPENAI_API_KEY environment variable")
	}

	// Create the OpenAI provider
	var provider core.Provider = openai.New(
		openai.WithAPIKey(apiKey),
		openai.WithModel("gpt-4o-mini"),
	)

	// Apply middleware for production readiness
	provider = middleware.Chain(
		middleware.WithRetry(middleware.RetryOpts{
			MaxAttempts: 3,
			BaseDelay:   time.Second,
			MaxDelay:    10 * time.Second,
			Jitter:      true,
		}),
		middleware.WithRateLimit(middleware.RateLimitOpts{
			RPS:   5,
			Burst: 10,
		}),
	)(provider)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Example 1: Simple text generation
	fmt.Println("=== Example 1: Simple Text Generation ===\n")
	simpleExample(ctx, provider)

	// Example 2: Text generation with system prompt
	fmt.Println("\n=== Example 2: Text Generation with System Prompt ===\n")
	systemPromptExample(ctx, provider)

	// Example 3: Multi-turn conversation
	fmt.Println("\n=== Example 3: Multi-turn Conversation ===\n")
	conversationExample(ctx, provider)

	// Example 4: Controlled generation with parameters
	fmt.Println("\n=== Example 4: Controlled Generation ===\n")
	controlledExample(ctx, provider)
}

func simpleExample(ctx context.Context, provider core.Provider) {
	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Write a haiku about Go programming."},
				},
			},
		},
	}

	result, err := provider.GenerateText(ctx, request)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Println("Response:")
	fmt.Println(result.Text)
	fmt.Printf("\nUsage: %d input tokens, %d output tokens\n", 
		result.Usage.InputTokens, result.Usage.OutputTokens)
}

func systemPromptExample(ctx context.Context, provider core.Provider) {
	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a helpful assistant who speaks like a pirate."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Explain what a database is."},
				},
			},
		},
	}

	result, err := provider.GenerateText(ctx, request)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Println("Response:")
	fmt.Println(result.Text)
}

func conversationExample(ctx context.Context, provider core.Provider) {
	// Build a conversation history
	messages := []core.Message{
		{
			Role: core.User,
			Parts: []core.Part{
				core.Text{Text: "My name is Alice. Remember it."},
			},
		},
		{
			Role: core.Assistant,
			Parts: []core.Part{
				core.Text{Text: "Hello Alice! I'll remember your name. How can I help you today?"},
			},
		},
		{
			Role: core.User,
			Parts: []core.Part{
				core.Text{Text: "What's my name?"},
			},
		},
	}

	request := core.Request{
		Messages: messages,
	}

	result, err := provider.GenerateText(ctx, request)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Println("Conversation:")
	for _, msg := range messages {
		role := string(msg.Role)
		if len(msg.Parts) > 0 {
			if text, ok := msg.Parts[0].(core.Text); ok {
				fmt.Printf("%s: %s\n", role, text.Text)
			}
		}
	}
	fmt.Printf("assistant: %s\n", result.Text)
}

func controlledExample(ctx context.Context, provider core.Provider) {
	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Generate a creative story about a robot learning to paint."},
				},
			},
		},
		Temperature: 0.9,  // High creativity
		MaxTokens:   200,  // Limit response length
		Model:       "gpt-4o-mini",
	}

	fmt.Println("Generating with high creativity (temperature=0.9)...")
	result, err := provider.GenerateText(ctx, request)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Println("Creative Story:")
	fmt.Println(result.Text)

	// Now generate with low creativity
	request.Temperature = 0.1
	fmt.Println("\nGenerating with low creativity (temperature=0.1)...")
	
	result, err = provider.GenerateText(ctx, request)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Println("Less Creative Story:")
	fmt.Println(result.Text)
}