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

// toolAdapter adapts tools.Handle to core.ToolHandle
type toolAdapter struct {
	tool tools.Handle
}

func (ta *toolAdapter) Name() string {
	return ta.tool.Name()
}

func (ta *toolAdapter) Description() string {
	return ta.tool.Description()
}

func (ta *toolAdapter) InputSchemaJSON() json.RawMessage {
	return ta.tool.InSchemaJSON()
}

func (ta *toolAdapter) Exec(ctx context.Context, input json.RawMessage, meta interface{}) (any, error) {
	// Convert meta to tools.Meta
	toolMeta := tools.Meta{}
	if m, ok := meta.(map[string]interface{}); ok {
		if callID, ok := m["call_id"].(string); ok {
			toolMeta.CallID = callID
		}
		if stepNum, ok := m["step_number"].(int); ok {
			toolMeta.StepNumber = stepNum
		}
	}
	return ta.tool.Exec(ctx, input, toolMeta)
}

func main() {
	// Check for API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set OPENAI_API_KEY environment variable")
	}

	// Create context
	ctx := context.Background()

	// Create provider with observability
	collector := obs.NewCollector(ctx, "openai", "gpt-4o-mini")
	provider := openai.New(
		openai.WithAPIKey(apiKey),
		openai.WithModel("gpt-4o-mini"),
		openai.WithMaxRetries(3),
		openai.WithRetryDelay(100*time.Millisecond),
		openai.WithMetricsCollector(collector),
	)

	fmt.Println("=== OpenAI Provider Examples ===")
	fmt.Println()

	// Run examples
	basicExample(ctx, provider)
	fmt.Println()
	streamingExample(ctx, provider)
	fmt.Println()
	toolCallingExample(ctx, provider)
	fmt.Println()
	structuredOutputExample(ctx, provider)

	// Print usage summary
	fmt.Println("\n=== Usage Summary ===")
	usage := collector.GetUsage()
	fmt.Printf("Total Requests: %d\n", usage.TotalRequests)
	fmt.Printf("Total Input Tokens: %d\n", usage.TotalInputTokens)
	fmt.Printf("Total Output Tokens: %d\n", usage.TotalOutputTokens)
	fmt.Printf("Total Cost: %d microcents\n", usage.TotalCostMicrocents)
}

// basicExample demonstrates simple text generation
func basicExample(ctx context.Context, provider core.Provider) {
	fmt.Println("1. Basic Text Generation")
	fmt.Println("------------------------")

	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a helpful assistant."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What is the capital of France? Answer in one word."},
				},
			},
		},
		Temperature: 0,
		MaxTokens:   50,
	})

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", result.Text)
	fmt.Printf("Usage: Input=%d, Output=%d, Total=%d tokens\n",
		result.Usage.InputTokens,
		result.Usage.OutputTokens,
		result.Usage.TotalTokens)
}

// streamingExample demonstrates streaming text generation
func streamingExample(ctx context.Context, provider core.Provider) {
	fmt.Println("2. Streaming Text Generation")
	fmt.Println("---------------------------")

	stream, err := provider.StreamText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Count from 1 to 5 with explanations."},
				},
			},
		},
		MaxTokens:   150,
		Temperature: 0.7,
		Stream:      true,
	})

	if err != nil {
		log.Printf("Error starting stream: %v", err)
		return
	}
	defer stream.Close()

	fmt.Print("Streaming response: ")
	for event := range stream.Events() {
		switch event.Type {
		case core.EventTextDelta:
			fmt.Print(event.TextDelta)
		case core.EventFinish:
			fmt.Println("\n[Stream finished]")
			if event.Usage != nil {
				fmt.Printf("Usage: Input=%d, Output=%d tokens\n",
					event.Usage.InputTokens,
					event.Usage.OutputTokens)
			}
		case core.EventError:
			log.Printf("Stream error: %v", event.Err)
		}
	}
}

// toolCallingExample demonstrates function/tool calling
func toolCallingExample(ctx context.Context, provider core.Provider) {
	fmt.Println("3. Tool Calling")
	fmt.Println("---------------")

	// Define a weather tool
	type WeatherInput struct {
		Location string `json:"location" jsonschema:"required,description=City name"`
	}
	type WeatherOutput struct {
		Temperature int    `json:"temperature"`
		Condition   string `json:"condition"`
	}

	weatherTool := tools.New[WeatherInput, WeatherOutput](
		"get_weather",
		"Get the current weather for a location",
		func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
			// Simulate weather API call
			fmt.Printf("[Tool Called] get_weather(%s)\n", input.Location)
			return WeatherOutput{
				Temperature: 72,
				Condition:   "Sunny",
			}, nil
		},
	)

	// Make request with tool
	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What's the weather like in San Francisco?"},
				},
			},
		},
		Tools:      []core.ToolHandle{tools.NewCoreAdapter(weatherTool)},
		ToolChoice: core.ToolAuto,
		MaxTokens:  200,
	})

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Final response: %s\n", result.Text)
	fmt.Printf("Number of steps: %d\n", len(result.Steps))
	for i, step := range result.Steps {
		fmt.Printf("  Step %d: %d tool calls\n", i+1, len(step.ToolCalls))
	}
}

// structuredOutputExample demonstrates typed JSON output
func structuredOutputExample(ctx context.Context, provider core.Provider) {
	fmt.Println("4. Structured Output")
	fmt.Println("--------------------")

	// Define the output structure
	type Recipe struct {
		Name        string   `json:"name" jsonschema:"required"`
		Ingredients []string `json:"ingredients" jsonschema:"required"`
		PrepTime    int      `json:"prep_time_minutes" jsonschema:"required"`
		Difficulty  string   `json:"difficulty" jsonschema:"enum=easy,enum=medium,enum=hard"`
	}

	// Generate structured output
	result, err := provider.GenerateObject(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a helpful cooking assistant. Always provide valid JSON."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Give me a simple pasta recipe."},
				},
			},
		},
		MaxTokens: 200,
	}, Recipe{})

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	// Type assert the result
	if recipe, ok := result.Value.(*Recipe); ok {
		fmt.Printf("Recipe Name: %s\n", recipe.Name)
		fmt.Printf("Difficulty: %s\n", recipe.Difficulty)
		fmt.Printf("Prep Time: %d minutes\n", recipe.PrepTime)
		fmt.Printf("Ingredients: %v\n", recipe.Ingredients)
	} else {
		fmt.Printf("Raw result: %+v\n", result.Value)
	}
}
