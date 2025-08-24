package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/middleware"
	"github.com/recera/gai/providers/openai_compat"
	"github.com/recera/gai/tools"
)

func main() {
	// Choose which example to run
	examples := []struct {
		name string
		fn   func()
	}{
		{"Basic Text Generation", basicTextGeneration},
		{"Streaming Responses", streamingExample},
		{"Tool Calling", toolCallingExample},
		{"Structured Outputs", structuredOutputExample},
		{"Multiple Providers", multipleProvidersExample},
		{"Error Handling", errorHandlingExample},
		{"Provider Quirks", providerQuirksExample},
		{"With Middleware", middlewareExample},
		{"Conversation Management", conversationExample},
		{"Vision Support", visionExample},
	}

	fmt.Println("OpenAI-Compatible Provider Examples")
	fmt.Println("====================================\n")

	for _, example := range examples {
		fmt.Printf("\n📝 %s\n", example.name)
		fmt.Println(strings.Repeat("-", 40))
		example.fn()
		fmt.Println()
	}
}

// basicTextGeneration demonstrates simple text generation with Groq
func basicTextGeneration() {
	ctx := context.Background()

	// Create Groq provider (very fast inference)
	provider, err := openai_compat.Groq()
	if err != nil {
		log.Printf("Failed to create Groq provider: %v", err)
		return
	}

	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a helpful assistant. Be concise."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What are the three primary colors?"},
				},
			},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	})

	if err != nil {
		log.Printf("Generation failed: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", result.Text)
	fmt.Printf("Tokens: Input=%d, Output=%d, Total=%d\n",
		result.Usage.InputTokens,
		result.Usage.OutputTokens,
		result.Usage.TotalTokens)
}

// streamingExample demonstrates streaming responses
func streamingExample() {
	ctx := context.Background()

	// Create provider (using Together for variety)
	provider, err := openai_compat.Together()
	if err != nil {
		log.Printf("Failed to create Together provider: %v", err)
		return
	}

	stream, err := provider.StreamText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Write a haiku about artificial intelligence."},
				},
			},
		},
		Stream:      true,
		Temperature: 0.9,
	})

	if err != nil {
		log.Printf("Stream creation failed: %v", err)
		return
	}
	defer stream.Close()

	fmt.Print("Streaming: ")
	var totalTokens int

	for event := range stream.Events() {
		switch event.Type {
		case core.EventTextDelta:
			fmt.Print(event.TextDelta)
		case core.EventFinish:
			if event.Usage != nil {
				totalTokens = event.Usage.TotalTokens
			}
		case core.EventError:
			log.Printf("\nStream error: %v", event.Err)
			return
		}
	}

	fmt.Printf("\n(Total tokens: %d)\n", totalTokens)
}

// toolCallingExample demonstrates function calling
func toolCallingExample() {
	ctx := context.Background()

	// Define tools
	type CalculatorInput struct {
		Expression string `json:"expression"`
	}
	type CalculatorOutput struct {
		Result float64 `json:"result"`
	}

	calculator := tools.New[CalculatorInput, CalculatorOutput](
		"calculate",
		"Evaluate a mathematical expression",
		func(ctx context.Context, in CalculatorInput, meta tools.Meta) (CalculatorOutput, error) {
			// Simple example - in production use a proper expression evaluator
			result := 42.0 // Placeholder
			fmt.Printf("  [Tool] Calculating: %s = %.2f\n", in.Expression, result)
			return CalculatorOutput{Result: result}, nil
		},
	)

	type WeatherInput struct {
		Location string `json:"location"`
	}
	type WeatherOutput struct {
		Temperature float64 `json:"temperature"`
		Conditions  string  `json:"conditions"`
	}

	weather := tools.New[WeatherInput, WeatherOutput](
		"get_weather",
		"Get current weather for a location",
		func(ctx context.Context, in WeatherInput, meta tools.Meta) (WeatherOutput, error) {
			fmt.Printf("  [Tool] Getting weather for: %s\n", in.Location)
			return WeatherOutput{
				Temperature: 72.5,
				Conditions:  "Sunny with light clouds",
			}, nil
		},
	)

	// Create provider
	provider, err := openai_compat.Groq()
	if err != nil {
		log.Printf("Failed to create provider: %v", err)
		return
	}

	// Make request with tools
	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What's the weather in San Francisco and what's 15 * 28?"},
				},
			},
		},
		Tools: []core.ToolHandle{
			tools.NewCoreAdapter(calculator),
			tools.NewCoreAdapter(weather),
		},
		ToolChoice: core.ToolAuto,
		StopWhen:   core.MaxSteps(3),
	})

	if err != nil {
		log.Printf("Tool calling failed: %v", err)
		return
	}

	fmt.Printf("Final answer: %s\n", result.Text)
	fmt.Printf("Steps taken: %d\n", len(result.Steps))
}

// structuredOutputExample demonstrates JSON schema-based outputs
func structuredOutputExample() {
	ctx := context.Background()

	type ProductReview struct {
		ProductName string   `json:"product_name"`
		Rating      int      `json:"rating"`
		Pros        []string `json:"pros"`
		Cons        []string `json:"cons"`
		Summary     string   `json:"summary"`
		Recommend   bool     `json:"recommend"`
	}

	provider, err := openai_compat.XAI()
	if err != nil {
		log.Printf("Failed to create xAI provider: %v", err)
		return
	}

	result, err := provider.GenerateObject(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "Extract structured product review information."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: `Review: The AirPods Pro 2 are fantastic! The noise cancellation 
					is incredible and the sound quality is crisp. Battery life could be better 
					and they're quite expensive, but overall I love them. Definitely worth it 
					if you can afford them. 4.5/5 stars.`},
				},
			},
		},
	}, ProductReview{})

	if err != nil {
		log.Printf("Structured output failed: %v", err)
		return
	}

	review := result.Value.(*ProductReview)
	fmt.Printf("Product: %s\n", review.ProductName)
	fmt.Printf("Rating: %d/5\n", review.Rating)
	fmt.Printf("Pros: %v\n", review.Pros)
	fmt.Printf("Cons: %v\n", review.Cons)
	fmt.Printf("Recommend: %v\n", review.Recommend)
}

// multipleProvidersExample shows using different providers
func multipleProvidersExample() {
	ctx := context.Background()

	providers := map[string]func(...openai_compat.Option) (*openai_compat.Provider, error){
		"Groq (Fast)":      openai_compat.Groq,
		"Cerebras (Ultra)": openai_compat.Cerebras,
		"Together (Flex)":  openai_compat.Together,
	}

	question := "What is 2+2?"

	for name, createFn := range providers {
		provider, err := createFn()
		if err != nil {
			fmt.Printf("%s: Failed to create - %v\n", name, err)
			continue
		}

		start := time.Now()
		result, err := provider.GenerateText(ctx, core.Request{
			Messages: []core.Message{
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: question},
					},
				},
			},
			MaxTokens: 10,
		})

		if err != nil {
			fmt.Printf("%s: Failed - %v\n", name, err)
			continue
		}

		fmt.Printf("%s: %s (%.2fs)\n", name, strings.TrimSpace(result.Text), time.Since(start).Seconds())
	}
}

// errorHandlingExample demonstrates error handling
func errorHandlingExample() {
	ctx := context.Background()

	// Create provider with invalid configuration to trigger errors
	provider, err := openai_compat.New(openai_compat.CompatOpts{
		BaseURL:      "https://api.groq.com/openai/v1",
		APIKey:       "invalid-key", // This will cause auth error
		ProviderName: "groq",
	})
	if err != nil {
		log.Printf("Failed to create provider: %v", err)
		return
	}

	_, err = provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Hello"},
				},
			},
		},
	})

	if err != nil {
		// Demonstrate error classification
		if core.IsAuth(err) {
			fmt.Println("Authentication failed - check API key")
		} else if core.IsRateLimited(err) {
			fmt.Println("Rate limited - slow down requests")
			if aiErr, ok := err.(*core.AIError); ok && aiErr.RetryAfter != nil {
				fmt.Printf("Retry after: %v\n", *aiErr.RetryAfter)
			}
		} else if core.IsTransient(err) {
			fmt.Println("Temporary error - can retry")
		} else {
			fmt.Printf("Other error: %v\n", err)
		}
	}
}

// providerQuirksExample shows handling provider limitations
func providerQuirksExample() {
	ctx := context.Background()

	// Cerebras has specific limitations
	fmt.Println("Cerebras limitations:")
	provider, err := openai_compat.Cerebras()
	if err != nil {
		log.Printf("Failed to create Cerebras provider: %v", err)
		return
	}

	// This will work despite Cerebras not supporting JSON streaming
	// The adapter handles it gracefully
	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Say 'Hello Cerebras'"},
				},
			},
		},
	})

	if err != nil {
		log.Printf("Cerebras request failed: %v", err)
		return
	}

	fmt.Printf("Cerebras response: %s\n", result.Text)

	// Custom provider with specific quirks
	fmt.Println("\nCustom provider with quirks:")
	customProvider, err := openai_compat.New(openai_compat.CompatOpts{
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "test-key",
		ProviderName: "custom",

		// Disable features this provider doesn't support
		DisableJSONStreaming:     true,
		DisableParallelToolCalls: true,
		DisableStrictJSONSchema:  true,

		// Strip parameters the provider doesn't understand
		UnsupportedParams: []string{"seed", "top_p", "logit_bias"},
	})

	if customProvider != nil {
		fmt.Println("Custom provider created with quirks handled")
	}
}

// middlewareExample demonstrates using middleware
func middlewareExample() {
	ctx := context.Background()

	// Create base provider
	provider, err := openai_compat.Groq()
	if err != nil {
		log.Printf("Failed to create provider: %v", err)
		return
	}

	// Add middleware for retry and rate limiting
	enhancedProvider := middleware.Chain(
		middleware.WithRetry(middleware.RetryOpts{
			MaxAttempts: 3,
			BaseDelay:   time.Second,
			Jitter:      true,
		}),
		middleware.WithRateLimit(middleware.RateLimitOpts{
			RPS:   5,
			Burst: 10,
		}),
		middleware.WithSafety(middleware.SafetyOpts{
			MaxContentLength: 10000,
			RedactPatterns: []string{
				`\b\d{3}-\d{2}-\d{4}\b`, // SSN
			},
			RedactReplacement: "[REDACTED]",
		}),
	)(provider)

	result, err := enhancedProvider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What is middleware in software?"},
				},
			},
		},
		MaxTokens: 50,
	})

	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}

	fmt.Printf("With middleware: %s\n", result.Text)
}

// conversationExample shows multi-turn conversations
func conversationExample() {
	ctx := context.Background()

	provider, err := openai_compat.Groq()
	if err != nil {
		log.Printf("Failed to create provider: %v", err)
		return
	}

	// Build conversation
	messages := []core.Message{
		{
			Role: core.System,
			Parts: []core.Part{
				core.Text{Text: "You are a helpful math tutor."},
			},
		},
	}

	// Turn 1
	messages = append(messages, core.Message{
		Role: core.User,
		Parts: []core.Part{
			core.Text{Text: "What is the Pythagorean theorem?"},
		},
	})

	result1, err := provider.GenerateText(ctx, core.Request{
		Messages:  messages,
		MaxTokens: 100,
	})
	if err != nil {
		log.Printf("Turn 1 failed: %v", err)
		return
	}

	messages = append(messages, core.Message{
		Role: core.Assistant,
		Parts: []core.Part{
			core.Text{Text: result1.Text},
		},
	})

	// Turn 2
	messages = append(messages, core.Message{
		Role: core.User,
		Parts: []core.Part{
			core.Text{Text: "Can you give me an example with numbers?"},
		},
	})

	result2, err := provider.GenerateText(ctx, core.Request{
		Messages:  messages,
		MaxTokens: 100,
	})
	if err != nil {
		log.Printf("Turn 2 failed: %v", err)
		return
	}

	fmt.Println("Conversation:")
	fmt.Printf("User: What is the Pythagorean theorem?\n")
	fmt.Printf("Assistant: %s\n", result1.Text)
	fmt.Printf("User: Can you give me an example with numbers?\n")
	fmt.Printf("Assistant: %s\n", result2.Text)
}

// visionExample demonstrates vision capabilities (if supported)
func visionExample() {
	ctx := context.Background()

	// Note: Not all providers support vision
	// This example shows how to use it when available
	provider, err := openai_compat.Together()
	if err != nil {
		log.Printf("Failed to create provider: %v", err)
		return
	}

	// Check capabilities
	caps := provider.GetCapabilities()
	if caps != nil && !caps.SupportsVision {
		fmt.Println("Provider does not support vision")
		return
	}

	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What's in this image?"},
					core.ImageURL{URL: "https://example.com/image.jpg"},
				},
			},
		},
		MaxTokens: 100,
	})

	if err != nil {
		// Many models don't support vision
		if strings.Contains(err.Error(), "not supported") {
			fmt.Println("Vision not supported by current model")
		} else {
			log.Printf("Vision request failed: %v", err)
		}
		return
	}

	fmt.Printf("Image description: %s\n", result.Text)
}

// init sets up environment variables if not already set
func init() {
	// Set dummy API keys if not present (for demo purposes)
	envVars := map[string]string{
		"GROQ_API_KEY":     "gsk_dummy_key_for_testing",
		"XAI_API_KEY":      "xai_dummy_key_for_testing",
		"CEREBRAS_API_KEY": "cbr_dummy_key_for_testing",
		"TOGETHER_API_KEY": "tog_dummy_key_for_testing",
	}

	for key, value := range envVars {
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}