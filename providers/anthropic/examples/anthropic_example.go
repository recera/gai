package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/recera/gai/core"
	"github.com/recera/gai/providers/anthropic"
	"github.com/recera/gai/tools"
)

// Example demonstrating comprehensive usage of the Anthropic provider
func main() {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	// Create provider with configuration
	provider := anthropic.New(
		anthropic.WithAPIKey(apiKey),
		anthropic.WithModel("claude-3-haiku-20240307"), // Using faster model for examples
	)

	ctx := context.Background()

	fmt.Println("=== Anthropic Provider Examples ===")

	// Example 1: Basic text generation
	fmt.Println("1. Basic Text Generation")
	fmt.Println("------------------------")
	basicTextGeneration(ctx, provider)

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Example 2: System prompts
	fmt.Println("2. System Prompts")
	fmt.Println("-----------------")
	systemPromptExample(ctx, provider)

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Example 3: Conversation context
	fmt.Println("3. Conversation Context")
	fmt.Println("-----------------------")
	conversationExample(ctx, provider)

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Example 4: Streaming
	fmt.Println("4. Streaming Text")
	fmt.Println("-----------------")
	streamingExample(ctx, provider)

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Example 5: Structured output
	fmt.Println("5. Structured Object Generation")
	fmt.Println("-------------------------------")
	structuredOutputExample(ctx, provider)

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Example 6: Tool calling
	fmt.Println("6. Tool Calling")
	fmt.Println("---------------")
	toolCallingExample(ctx, provider)

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Example 7: Error handling
	fmt.Println("7. Error Handling")
	fmt.Println("-----------------")
	errorHandlingExample(ctx, provider)

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Example 8: Provider-specific options
	fmt.Println("8. Provider-Specific Options")
	fmt.Println("----------------------------")
	providerOptionsExample(ctx, provider)

	fmt.Println("\nAll examples completed!")
}

// basicTextGeneration demonstrates simple text generation
func basicTextGeneration(ctx context.Context, provider *anthropic.Provider) {
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Explain quantum computing in simple terms."}},
			},
		},
		MaxTokens: 200,
	}

	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", result.Text)
	fmt.Printf("Usage: %d input tokens, %d output tokens\n", 
		result.Usage.InputTokens, result.Usage.OutputTokens)
}

// systemPromptExample demonstrates system prompt usage
func systemPromptExample(ctx context.Context, provider *anthropic.Provider) {
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.System,
				Parts: []core.Part{core.Text{Text: "You are a helpful coding assistant. Provide concise, practical answers with code examples when appropriate."}},
			},
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "How do I reverse a string in Go?"}},
			},
		},
		MaxTokens: 300,
	}

	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", result.Text)
}

// conversationExample demonstrates multi-turn conversation
func conversationExample(ctx context.Context, provider *anthropic.Provider) {
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "I'm planning a trip to Japan. What should I know?"}},
			},
			{
				Role:  core.Assistant,
				Parts: []core.Part{core.Text{Text: "Japan is a fascinating destination! Here are some key things to know: You'll need a passport, consider getting a JR Pass for trains, learn basic Japanese phrases, and try the amazing food. The best times to visit are spring (cherry blossoms) or fall (autumn colors)."}},
			},
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "What about cultural etiquette?"}},
			},
		},
		MaxTokens: 250,
	}

	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", result.Text)
}

// streamingExample demonstrates real-time streaming
func streamingExample(ctx context.Context, provider *anthropic.Provider) {
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Write a short poem about artificial intelligence."}},
			},
		},
		MaxTokens: 200,
		Stream:    true,
	}

	stream, err := provider.StreamText(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer stream.Close()

	fmt.Print("Streaming response: ")
	var totalTokens int

	for event := range stream.Events() {
		switch event.Type {
		case core.EventStart:
			fmt.Print("\n")
		case core.EventTextDelta:
			fmt.Print(event.TextDelta)
		case core.EventFinish:
			if event.Usage != nil {
				totalTokens = event.Usage.TotalTokens
			}
			fmt.Printf("\n\nStream finished. Total tokens: %d\n", totalTokens)
		case core.EventError:
			fmt.Printf("\nStream error: %v\n", event.Err)
			return
		}
	}
}

// structuredOutputExample demonstrates JSON object generation
func structuredOutputExample(ctx context.Context, provider *anthropic.Provider) {
	type MovieReview struct {
		Title   string `json:"title"`
		Rating  int    `json:"rating"`
		Summary string `json:"summary"`
		Genres  []string `json:"genres"`
	}

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title": map[string]interface{}{
				"type": "string",
				"description": "Movie title",
			},
			"rating": map[string]interface{}{
				"type": "integer",
				"minimum": 1,
				"maximum": 10,
				"description": "Rating out of 10",
			},
			"summary": map[string]interface{}{
				"type": "string",
				"description": "Brief summary of the movie",
			},
			"genres": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "Movie genres",
			},
		},
		"required": []string{"title", "rating", "summary", "genres"},
	}

	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Create a fictional movie review for a sci-fi thriller about time travel."}},
			},
		},
		MaxTokens: 300,
	}

	result, err := provider.GenerateObject(ctx, req, schema)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	// Parse the result
	jsonBytes, err := json.Marshal(result.Value)
	if err != nil {
		log.Printf("Error marshaling result: %v", err)
		return
	}

	var review MovieReview
	if err := json.Unmarshal(jsonBytes, &review); err != nil {
		log.Printf("Error parsing review: %v", err)
		return
	}

	fmt.Printf("Generated Movie Review:\n")
	fmt.Printf("Title: %s\n", review.Title)
	fmt.Printf("Rating: %d/10\n", review.Rating)
	fmt.Printf("Summary: %s\n", review.Summary)
	fmt.Printf("Genres: %s\n", strings.Join(review.Genres, ", "))
}

// toolCallingExample demonstrates tool usage
func toolCallingExample(ctx context.Context, provider *anthropic.Provider) {
	// Create a weather tool
	weatherTool := tools.New("get_weather", "Get current weather information for a city",
		func(ctx context.Context, input struct {
			City    string `json:"city" description:"Name of the city"`
			Country string `json:"country,omitempty" description:"Country code (optional)"`
		}, meta tools.Meta) (map[string]interface{}, error) {
			// Mock weather data
			weather := map[string]interface{}{
				"city":        input.City,
				"temperature": 22,
				"condition":   "Sunny",
				"humidity":    60,
				"wind_speed":  10,
			}
			return weather, nil
		})

	// Create a calculator tool
	calcTool := tools.New("calculate", "Perform mathematical calculations",
		func(ctx context.Context, input struct {
			Expression string `json:"expression" description:"Mathematical expression to evaluate"`
		}, meta tools.Meta) (map[string]interface{}, error) {
			// Simple calculation (in real implementation, use a proper expression parser)
			result := 42.0 // Mock result
			return map[string]interface{}{
				"expression": input.Expression,
				"result":     result,
			}, nil
		})

	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "What's the weather like in Tokyo? Also, what's 15 * 27?"}},
			},
		},
		Tools:     tools.ToCoreHandles([]tools.Handle{weatherTool, calcTool}),
		MaxTokens: 400,
		StopWhen:  core.MaxSteps(3), // Allow up to 3 steps
	}

	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Final Response: %s\n", result.Text)
	fmt.Printf("Steps taken: %d\n", len(result.Steps))

	// Show the execution steps
	for i, step := range result.Steps {
		fmt.Printf("\nStep %d:\n", i+1)
		fmt.Printf("  Text: %s\n", step.Text)
		
		if len(step.ToolCalls) > 0 {
			fmt.Printf("  Tool Calls:\n")
			for _, call := range step.ToolCalls {
				fmt.Printf("    - %s: %s\n", call.Name, string(call.Input))
			}
		}
		
		if len(step.ToolResults) > 0 {
			fmt.Printf("  Tool Results:\n")
			for _, res := range step.ToolResults {
				if res.Error != "" {
					fmt.Printf("    - Error: %s\n", res.Error)
				} else {
					fmt.Printf("    - Result: %v\n", res.Result)
				}
			}
		}
	}
}

// errorHandlingExample demonstrates error handling patterns
func errorHandlingExample(ctx context.Context, provider *anthropic.Provider) {
	// Create a request that might trigger various errors
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Generate a very long response about the history of computing, covering every major development from the abacus to modern quantum computers, with detailed explanations of each era, key figures, technological breakthroughs, and their societal impacts."}},
			},
		},
		MaxTokens: 10000, // Very high token limit that might exceed context
	}

	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		fmt.Printf("An error occurred: %v\n\n", err)
		
		// Demonstrate error type checking
		fmt.Println("Error analysis:")
		switch {
		case core.IsAuth(err):
			fmt.Println("❌ Authentication error - check your API key")
		case core.IsRateLimited(err):
			retryAfter := core.GetRetryAfter(err)
			fmt.Printf("⏰ Rate limited - retry after %v\n", retryAfter)
		case core.IsContextSizeExceeded(err):
			fmt.Println("📏 Context too long - reduce input size or max tokens")
		case core.IsSafetyBlocked(err):
			fmt.Println("🛡️  Content blocked by safety filters")
		case core.IsTransient(err):
			fmt.Println("🔄 Temporary error - safe to retry")
		case core.IsNetwork(err):
			fmt.Println("🌐 Network error - check connection")
		case core.IsTimeout(err):
			fmt.Println("⏱️  Request timed out")
		default:
			fmt.Printf("❓ Other error type: %T\n", err)
		}

		// Show error details
		if aiErr, ok := err.(*core.AIError); ok {
			fmt.Printf("\nError details:\n")
			fmt.Printf("  Code: %s\n", aiErr.Code)
			fmt.Printf("  Provider: %s\n", aiErr.Provider)
			fmt.Printf("  HTTP Status: %d\n", aiErr.HTTPStatus)
			fmt.Printf("  Temporary: %v\n", aiErr.Temporary)
			if aiErr.RetryAfter != nil {
				fmt.Printf("  Retry After: %v\n", *aiErr.RetryAfter)
			}
		}
		return
	}

	fmt.Printf("Success! Generated %d tokens\n", result.Usage.TotalTokens)
}

// providerOptionsExample demonstrates Anthropic-specific options
func providerOptionsExample(ctx context.Context, provider *anthropic.Provider) {
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Write a creative short story ending. Be imaginative!"}},
			},
		},
		MaxTokens:   250,
		Temperature: 0.9, // High creativity
		ProviderOptions: map[string]interface{}{
			"anthropic": map[string]interface{}{
				"top_p":          0.95,  // Nucleus sampling
				"top_k":          50,    // Top-k sampling
				"stop_sequences": []string{"\n---\n", "THE END"}, // Custom stop sequences
			},
		},
	}

	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Creative story ending:\n%s\n", result.Text)
	fmt.Printf("Generated with high temperature (%.1f) and nucleus sampling (top_p=%.2f)\n", 
		0.9, 0.95)
}