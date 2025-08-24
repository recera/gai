// Package main demonstrates the OpenAI provider capabilities of the GAI framework.
// This example showcases text generation, streaming, tool calling, and structured outputs.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/obs"
	"github.com/recera/gai/providers/openai"
	"github.com/recera/gai/tools"
)

func main() {
	// Check for API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set OPENAI_API_KEY environment variable")
	}

	// Create provider with observability
	collector := obs.NewIntegratedCollector()
	provider := openai.New(
		openai.WithAPIKey(apiKey),
		openai.WithModel("gpt-4o-mini"),
		openai.WithMaxRetries(3),
		openai.WithRetryDelay(100*time.Millisecond),
		openai.WithMetricsCollector(collector),
	)

	ctx := context.Background()

	// Example 1: Basic text generation
	fmt.Println("=== Example 1: Basic Text Generation ===")
	basicExample(ctx, provider)

	// Example 2: Streaming
	fmt.Println("\n=== Example 2: Streaming Response ===")
	streamingExample(ctx, provider)

	// Example 3: Tool calling
	fmt.Println("\n=== Example 3: Tool Calling ===")
	toolCallingExample(ctx, provider)

	// Example 4: Structured output
	fmt.Println("\n=== Example 4: Structured Output ===")
	structuredOutputExample(ctx, provider)

	// Example 5: Multimodal (if you have an image URL)
	fmt.Println("\n=== Example 5: Multimodal Input ===")
	multimodalExample(ctx, provider)

	// Example 6: Conversation with history
	fmt.Println("\n=== Example 6: Conversation History ===")
	conversationExample(ctx, provider)

	// Show usage statistics
	fmt.Println("\n=== Usage Statistics ===")
	showUsageStats(collector)
}

func basicExample(ctx context.Context, provider *openai.Provider) {
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
		Temperature: floatPtr(0),
		MaxTokens:   intPtr(50),
	})

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", result.Text)
	fmt.Printf("Tokens used: %d\n", result.Usage.TotalTokens)
}

func streamingExample(ctx context.Context, provider *openai.Provider) {
	stream, err := provider.StreamText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Count from 1 to 5 with a brief description of each number."},
				},
			},
		},
		MaxTokens:   intPtr(150),
		Temperature: floatPtr(0.7),
		Stream:      true,
	})

	if err != nil {
		log.Printf("Error: %v", err)
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
		}
	}
	fmt.Printf("\nTokens used: %d\n", totalTokens)
}

func toolCallingExample(ctx context.Context, provider *openai.Provider) {
	// Define tools
	type CalcInput struct {
		Expression string `json:"expression" jsonschema:"description=Mathematical expression like '2+2' or '10*5'"`
	}
	type CalcOutput struct {
		Result float64 `json:"result"`
	}

	calcTool := tools.New[CalcInput, CalcOutput](
		"calculator",
		"Performs basic mathematical calculations",
		func(ctx context.Context, in CalcInput, meta tools.Meta) (CalcOutput, error) {
			// Simple calculator (in production, use proper expression evaluator)
			var result float64
			switch in.Expression {
			case "2+2", "2 + 2":
				result = 4
			case "10*5", "10 * 5":
				result = 50
			case "100/4", "100 / 4":
				result = 25
			default:
				result = 42 // Default answer
			}
			return CalcOutput{Result: result}, nil
		},
	)

	type TimeInput struct {
		Location string `json:"location" jsonschema:"description=City name or timezone"`
	}
	type TimeOutput struct {
		Time string `json:"time"`
	}

	timeTool := tools.New[TimeInput, TimeOutput](
		"get_time",
		"Get current time in a location",
		func(ctx context.Context, in TimeInput, meta tools.Meta) (TimeOutput, error) {
			// Mock implementation
			currentTime := time.Now().Format("15:04:05")
			return TimeOutput{Time: fmt.Sprintf("%s in %s", currentTime, in.Location)}, nil
		},
	)

	// Request with tools
	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What's 10 * 5, and what time is it in Tokyo?"},
				},
			},
		},
		Tools: []core.ToolHandle{
			tools.NewCoreAdapter(calcTool),
			tools.NewCoreAdapter(timeTool),
		},
		ToolChoice: core.ToolAuto,
		MaxTokens:  intPtr(200),
	})

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", result.Text)
	
	// Show tool calls made
	if len(result.Steps) > 0 {
		fmt.Println("Tool calls made:")
		for _, step := range result.Steps {
			for _, call := range step.ToolCalls {
				fmt.Printf("  - %s: %s\n", call.Name, string(call.Input))
			}
		}
	}
}

func structuredOutputExample(ctx context.Context, provider *openai.Provider) {
	// Define schema for a book recommendation
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title": map[string]interface{}{
				"type": "string",
			},
			"author": map[string]interface{}{
				"type": "string",
			},
			"genre": map[string]interface{}{
				"type": "string",
				"enum": []string{"fiction", "non-fiction", "science", "history", "biography"},
			},
			"year_published": map[string]interface{}{
				"type": "integer",
			},
			"summary": map[string]interface{}{
				"type": "string",
			},
			"rating": map[string]interface{}{
				"type": "number",
				"minimum": 0,
				"maximum": 5,
			},
		},
		"required": []string{"title", "author", "genre", "summary", "rating"},
	}

	result, err := provider.GenerateObject(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Recommend a classic science fiction book."},
				},
			},
		},
		MaxTokens:   intPtr(200),
		Temperature: floatPtr(0.7),
	}, schema)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	// Pretty print the structured output
	jsonBytes, _ := json.MarshalIndent(result.Value, "", "  ")
	fmt.Printf("Structured Output:\n%s\n", string(jsonBytes))
}

func multimodalExample(ctx context.Context, provider *openai.Provider) {
	// Note: This requires a vision-capable model like gpt-4o
	result, err := provider.GenerateText(ctx, core.Request{
		Model: "gpt-4o-mini", // Vision-capable model
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Describe this image in one sentence:"},
					core.ImageURL{
						URL:    "https://upload.wikimedia.org/wikipedia/commons/thumb/3/3a/Cat03.jpg/1200px-Cat03.jpg",
						Detail: "auto",
					},
				},
			},
		},
		MaxTokens:   intPtr(50),
		Temperature: floatPtr(0),
	})

	if err != nil {
		// Some models don't support vision
		if err.Error() == "unsupported" {
			fmt.Println("Vision not supported on this model")
			return
		}
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Image Description: %s\n", result.Text)
}

func conversationExample(ctx context.Context, provider *openai.Provider) {
	// Build a conversation with context
	messages := []core.Message{
		{
			Role: core.System,
			Parts: []core.Part{
				core.Text{Text: "You are a helpful geography teacher."},
			},
		},
		{
			Role: core.User,
			Parts: []core.Part{
				core.Text{Text: "What's the capital of France?"},
			},
		},
		{
			Role: core.Assistant,
			Parts: []core.Part{
				core.Text{Text: "The capital of France is Paris."},
			},
		},
		{
			Role: core.User,
			Parts: []core.Part{
				core.Text{Text: "How many people live there?"},
			},
		},
	}

	result, err := provider.GenerateText(ctx, core.Request{
		Messages:    messages,
		MaxTokens:   intPtr(100),
		Temperature: floatPtr(0),
	})

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Contextual Response: %s\n", result.Text)
}

func showUsageStats(collector core.MetricsCollector) {
	// In a real application, you would export these to your observability platform
	// Here we just show that metrics are being collected
	fmt.Println("Metrics are being collected and can be exported to:")
	fmt.Println("  - Prometheus")
	fmt.Println("  - Grafana")
	fmt.Println("  - DataDog")
	fmt.Println("  - New Relic")
	fmt.Println("  - Or any OpenTelemetry-compatible backend")
}

// Helper functions
func floatPtr(f float32) *float32 {
	return &f
}

func intPtr(i int) *int {
	return &i
}