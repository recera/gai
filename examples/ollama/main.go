package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/providers/ollama"
)

func main() {
	// Create Ollama provider
	provider := ollama.New(
		ollama.WithModel("llama3.2"),
		ollama.WithKeepAlive("5m"),
	)

	fmt.Println("🦙 Ollama Provider Examples")
	fmt.Println("=========================================")

	// Check if models are available
	checkModels(provider)

	// Run examples
	fmt.Println("\n1. Basic Text Generation")
	basicTextGeneration(provider)

	fmt.Println("\n2. Streaming Text Generation")
	streamingTextGeneration(provider)

	fmt.Println("\n3. Structured Output Generation")
	structuredOutputGeneration(provider)

	fmt.Println("\n4. Tool Calling Example")
	toolCallingExample(provider)

	fmt.Println("\n5. Multimodal Example (Conceptual)")
	multimodalExample(provider)

	fmt.Println("\n6. Concurrent Requests Example")
	concurrentRequestsExample(provider)

	fmt.Println("\n7. Provider Options Example")
	providerOptionsExample(provider)

	fmt.Println("\n✅ All examples completed!")
}

func checkModels(provider *ollama.Provider) {
	fmt.Println("\n📋 Checking Available Models...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := provider.ListModels(ctx)
	if err != nil {
		log.Printf("⚠️  Warning: Could not list models: %v", err)
		log.Println("   Make sure Ollama is running: `ollama serve`")
		return
	}

	fmt.Printf("Found %d models:\n", len(models))
	for _, model := range models {
		fmt.Printf("  - %s (%.2f GB)\n", model.Name, float64(model.Size)/(1024*1024*1024))
	}

	// Check if our default model is available
	available, err := provider.IsModelAvailable(ctx, "llama3.2")
	if err != nil {
		log.Printf("Error checking model availability: %v", err)
		return
	}

	if !available {
		fmt.Println("\n📥 Model 'llama3.2' not found. You can pull it with:")
		fmt.Println("   ollama pull llama3.2")
	}
}

func basicTextGeneration(provider *ollama.Provider) {
	ctx := context.Background()

	req := core.Request{
		Messages: []core.Message{
			{Role: core.System, Parts: []core.Part{core.Text{Text: "You are a helpful and friendly AI assistant."}}},
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Tell me a fun fact about llamas!"}}},
		},
		Temperature: 0.7,
		MaxTokens:   150,
	}

	fmt.Println("Generating response...")
	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("🤖 Response: %s\n", result.Text)
	fmt.Printf("📊 Usage: Input=%d, Output=%d, Total=%d tokens\n", 
		result.Usage.InputTokens, result.Usage.OutputTokens, result.Usage.TotalTokens)
}

func streamingTextGeneration(provider *ollama.Provider) {
	ctx := context.Background()

	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Write a short poem about programming with AI."}}},
		},
		Temperature: 0.8,
		MaxTokens:   200,
	}

	fmt.Println("Streaming response...")
	stream, err := provider.StreamText(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer stream.Close()

	fmt.Print("🤖 Streaming: ")
	var fullText string
	var finalUsage *core.Usage

	for event := range stream.Events() {
		switch event.Type {
		case core.EventStart:
			// Stream started
		case core.EventTextDelta:
			fmt.Print(event.TextDelta)
			fullText += event.TextDelta
		case core.EventFinish:
			finalUsage = event.Usage
		case core.EventError:
			fmt.Printf("\n❌ Error: %v\n", event.Err)
			return
		}
	}

	fmt.Println() // New line after streaming
	if finalUsage != nil {
		fmt.Printf("📊 Stream Usage: Input=%d, Output=%d tokens\n", 
			finalUsage.InputTokens, finalUsage.OutputTokens)
	}
}

func structuredOutputGeneration(provider *ollama.Provider) {
	ctx := context.Background()

	// Define schema for a character profile
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type": "string",
				"description": "Character's full name",
			},
			"age": map[string]any{
				"type": "integer",
				"minimum": 0,
				"maximum": 150,
			},
			"occupation": map[string]any{
				"type": "string",
			},
			"personality": map[string]any{
				"type": "array",
				"items": map[string]any{"type": "string"},
				"description": "List of personality traits",
			},
			"background": map[string]any{
				"type": "string",
				"description": "Brief character background",
			},
		},
		"required": []string{"name", "age", "occupation", "personality"},
	}

	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Create a fantasy character profile for a wise old wizard."}}},
		},
	}

	fmt.Println("Generating structured character profile...")
	result, err := provider.GenerateObject(ctx, req, schema)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	// Parse the result
	character, ok := result.Value.(map[string]any)
	if !ok {
		log.Printf("Error: Expected object result")
		return
	}

	fmt.Println("🧙‍♂️ Generated Character:")
	fmt.Printf("  Name: %v\n", character["name"])
	fmt.Printf("  Age: %.0f years\n", character["age"])
	fmt.Printf("  Occupation: %v\n", character["occupation"])
	
	if personality, ok := character["personality"].([]any); ok {
		fmt.Printf("  Personality: ")
		for i, trait := range personality {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(trait)
		}
		fmt.Println()
	}
	
	if background := character["background"]; background != nil {
		fmt.Printf("  Background: %v\n", background)
	}
}

func toolCallingExample(provider *ollama.Provider) {
	ctx := context.Background()

	// Create a mock weather tool (simpler version)
	weatherTool := &mockWeatherTool{}

	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "What's the weather like?"}}},
		},
		Tools:    []core.ToolHandle{weatherTool},
		StopWhen: core.MaxSteps(3), // Limit to 3 steps
	}

	fmt.Println("Generating response with tool calls...")
	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("🤖 Final Response: %s\n", result.Text)
	
	if len(result.Steps) > 0 {
		fmt.Printf("🔧 Tool Execution Steps:\n")
		for i, step := range result.Steps {
			fmt.Printf("  Step %d: %s\n", i+1, step.Text)
			for _, toolCall := range step.ToolCalls {
				fmt.Printf("    🛠️  Called: %s\n", toolCall.Name)
			}
			for _, toolResult := range step.ToolResults {
				if toolResult.Error != "" {
					fmt.Printf("    ❌ Tool Error: %s\n", toolResult.Error)
				} else {
					fmt.Printf("    ✅ Tool Result: %v\n", toolResult.Result)
				}
			}
		}
	}
}

func multimodalExample(provider *ollama.Provider) {
	// Note: This is a conceptual example. In practice, you would need:
	// 1. A multimodal model (like llama3.2-vision)
	// 2. Actual base64-encoded image data

	fmt.Println("🖼️  Multimodal Example (Conceptual):")
	fmt.Println("This example shows how to structure a multimodal request.")
	fmt.Println("For actual usage:")
	fmt.Println("  1. Use a vision-capable model: ollama pull llama3.2-vision")
	fmt.Println("  2. Encode your image to base64")
	fmt.Println("  3. Include it in the ImageURL part")

	// Example structure (commented out to avoid errors)
	/*
	ctx := context.Background()
	
	req := core.Request{
		Model: "llama3.2-vision", // Vision-capable model
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What do you see in this image?"},
					core.ImageURL{URL: "data:image/jpeg;base64,/9j/4AAQ..."}, // Your base64 image
				},
			},
		},
	}

	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("🤖 Image Description: %s\n", result.Text)
	*/
}

func concurrentRequestsExample(provider *ollama.Provider) {
	fmt.Println("Running 5 concurrent requests...")

	var wg sync.WaitGroup
	results := make(chan string, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx := context.Background()
			req := core.Request{
				Messages: []core.Message{
					{Role: core.User, Parts: []core.Part{core.Text{Text: fmt.Sprintf("Tell me about number %d in one sentence.", id+1)}}},
				},
				MaxTokens: 50,
			}

			result, err := provider.GenerateText(ctx, req)
			if err != nil {
				results <- fmt.Sprintf("Request %d: Error - %v", id+1, err)
				return
			}

			results <- fmt.Sprintf("Request %d: %s", id+1, result.Text)
		}(i)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and display results
	fmt.Println("🚀 Concurrent Results:")
	for result := range results {
		fmt.Printf("  %s\n", result)
	}
}

func providerOptionsExample(provider *ollama.Provider) {
	ctx := context.Background()

	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Write a creative short story about a robot."}}},
		},
		Temperature: 0.9, // High creativity
		MaxTokens:   100,
		ProviderOptions: map[string]any{
			"ollama": map[string]any{
				"top_k":           20,    // Limit vocabulary
				"top_p":           0.9,   // Nucleus sampling
				"repeat_penalty":  1.2,   // Reduce repetition
				"seed":           42,     // Reproducible output
				"stop":           []string{"THE END"}, // Custom stop sequence
			},
		},
	}

	fmt.Println("Generating with custom parameters...")
	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("🤖 Creative Story: %s\n", result.Text)
}

// Mock weather tool for example
type mockWeatherTool struct{}

func (m *mockWeatherTool) Name() string {
	return "get_weather"
}

func (m *mockWeatherTool) Description() string {
	return "Get current weather information for a location"
}

func (m *mockWeatherTool) InSchemaJSON() []byte {
	return []byte(`{
		"type": "object",
		"properties": {
			"location": {
				"type": "string",
				"description": "The location to get weather for"
			}
		},
		"required": ["location"]
	}`)
}

func (m *mockWeatherTool) OutSchemaJSON() []byte {
	return []byte(`{
		"type": "object",
		"properties": {
			"temperature": {"type": "number"},
			"condition": {"type": "string"},
			"humidity": {"type": "number"}
		}
	}`)
}

func (m *mockWeatherTool) Exec(ctx context.Context, raw json.RawMessage, meta interface{}) (any, error) {
	var input struct {
		Location string `json:"location"`
	}
	
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"temperature": 22.5,
		"condition":   "partly cloudy",
		"humidity":    0.65,
		"location":    input.Location,
	}, nil
}