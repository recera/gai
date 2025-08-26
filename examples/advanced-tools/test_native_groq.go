// Test the native Groq provider with proper tool call ID handling
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/providers/groq"
	"github.com/recera/gai/tools"
)

type WeatherInput struct {
	Location string `json:"location" jsonschema:"required,description=City name"`
}

type WeatherOutput struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature_celsius"`
	Condition   string  `json:"condition"`
	Humidity    int     `json:"humidity_percent"`
}

func testNativeGroqProvider() {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Fatal("GROQ_API_KEY not set")
	}

	// Create native Groq provider
	provider := groq.New(
		groq.WithAPIKey(apiKey),
		groq.WithModel("moonshotai/kimi-k2-instruct"), 
		groq.WithServiceTier("on_demand"),
		groq.WithMaxRetries(2),
		groq.WithRetryDelay(200*time.Millisecond),
	)

	// Test health check
	fmt.Println("🔍 Testing Groq provider health...")
	if err := provider.HealthCheck(context.Background()); err != nil {
		log.Printf("Health check failed: %v", err)
		return
	}
	fmt.Println("✅ Health check passed")

	// Test model listing
	fmt.Println("\n📋 Available models:")
	if models, err := provider.GetModels(context.Background()); err == nil {
		for _, model := range models[:5] { // Show first 5
			fmt.Printf("- %s (%s) - %s context window\n", 
				model.ID, model.OwnedBy, formatContextWindow(model.ContextWindow))
		}
	}

	// Create advanced weather tool
	weatherTool := tools.New[WeatherInput, WeatherOutput](
		"get_weather",
		"Get current weather information for a specific location",
		func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
			fmt.Printf("🌦️  [Native Groq Tool] Weather analysis for: %s\n", input.Location)
			
			// Simulate detailed weather data
			temp := 15.0 + float64(len(input.Location)%15) * 2.0
			conditions := []string{"Sunny", "Partly Cloudy", "Cloudy", "Light Rain", "Clear"}
			condition := conditions[len(input.Location)%len(conditions)]
			humidity := 40 + (len(input.Location) % 40)
			
			return WeatherOutput{
				Location:    input.Location,
				Temperature: temp,
				Condition:   condition,
				Humidity:    humidity,
			}, nil
		},
	)

	// Test cases with different models and configurations
	testCases := []struct {
		name        string
		model       string
		query       string
		stopWhen    core.StopCondition
		expectTools bool
	}{
		{
			name:        "Kimi-K2 Ultra Fast",
			model:       "moonshotai/kimi-k2-instruct",
			query:       "What's the weather in Tokyo? Please be detailed.",
			stopWhen:    core.MaxSteps(2),
			expectTools: true,
		},
		{
			name:        "Llama-3.3 70B Versatile",
			model:       "llama-3.3-70b-versatile",
			query:       "Check the weather in London and Paris, compare them.",
			stopWhen:    core.NoMoreTools(),
			expectTools: true,
		},
		{
			name:        "Llama-3.1 8B Instant",
			model:       "llama-3.1-8b-instant",
			query:       "Get weather info for New York.",
			stopWhen:    core.UntilToolSeen("get_weather"),
			expectTools: true,
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n🧪 Test: %s\n", tc.name)
		fmt.Printf("Model: %s\n", tc.model)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// Update provider model
		provider := groq.New(
			groq.WithAPIKey(apiKey),
			groq.WithModel(tc.model),
			groq.WithServiceTier("on_demand"),
		)

		start := time.Now()

		request := core.Request{
			Messages: []core.Message{
				{
					Role: core.System,
					Parts: []core.Part{
						core.Text{Text: "You are a helpful weather assistant. Use tools to get accurate information."},
					},
				},
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: tc.query},
					},
				},
			},
			Tools:    tools.ToCoreHandles([]tools.Handle{weatherTool}),
			StopWhen: tc.stopWhen,
		}

		result, err := provider.GenerateText(context.Background(), request)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		fmt.Printf("✅ Success in %.2fs\n", duration.Seconds())
		fmt.Printf("📊 Steps: %d, Tools called: %d, Tokens: %d\n", 
			len(result.Steps), countToolCalls(result.Steps), result.Usage.TotalTokens)
		
		if result.Text != "" {
			fmt.Printf("💬 Response: %s\n", truncateText(result.Text, 150))
		}

		// Show execution details
		if len(result.Steps) > 0 {
			fmt.Println("\n🔍 Execution trace:")
			for i, step := range result.Steps {
				fmt.Printf("  Step %d: %d tool calls", i+1, len(step.ToolCalls))
				if step.Text != "" {
					fmt.Printf(" + text output")
				}
				fmt.Println()
			}
		}
	}

	// Test streaming
	fmt.Printf("\n🌊 Testing Streaming with Native Groq Provider\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	testStreaming(provider, weatherTool)
}

func testStreaming(provider *groq.Provider, weatherTool tools.Handle) {
	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What's the weather like in San Francisco? Please use the weather tool and then give me a detailed response."},
				},
			},
		},
		Tools: tools.ToCoreHandles([]tools.Handle{weatherTool}),
		Stream: true,
	}

	stream, err := provider.StreamText(context.Background(), request)
	if err != nil {
		fmt.Printf("❌ Streaming failed: %v\n", err)
		return
	}
	defer stream.Close()

	fmt.Printf("📡 Streaming response: ")
	
	var totalText string
	toolCallsDetected := false
	
	for event := range stream.Events() {
		switch event.Type {
		case core.EventStart:
			fmt.Printf("[START] ")
		case core.EventTextDelta:
			fmt.Print(event.TextDelta)
			totalText += event.TextDelta
		case core.EventToolCall:
			if !toolCallsDetected {
				fmt.Printf("\n🔧 [TOOL CALL] %s ", event.ToolName)
				toolCallsDetected = true
			}
		case core.EventToolResult:
			fmt.Printf("✅ [RESULT] ")
		case core.EventFinish:
			fmt.Printf("\n[COMPLETE]")
			if event.Usage != nil {
				fmt.Printf(" (%d tokens)", event.Usage.TotalTokens)
			}
			fmt.Println()
		case core.EventError:
			fmt.Printf("\n❌ [ERROR] %v\n", event.Err)
		}
	}
	
	fmt.Printf("\nTotal streamed text length: %d characters\n", len(totalText))
}

func formatContextWindow(contextWindow int) string {
	if contextWindow >= 131072 {
		return "128k+"
	} else if contextWindow >= 32768 {
		return "32k+"
	} else if contextWindow >= 8192 {
		return "8k"
	} else {
		return fmt.Sprintf("%dk", contextWindow/1024)
	}
}

func countToolCalls(steps []core.Step) int {
	count := 0
	for _, step := range steps {
		count += len(step.ToolCalls)
	}
	return count
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func main() {
	fmt.Println("🚀 Native Groq Provider Test Suite")
	fmt.Println("=" + "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	testNativeGroqProvider()
	
	fmt.Println("\n🎉 All tests completed!")
}