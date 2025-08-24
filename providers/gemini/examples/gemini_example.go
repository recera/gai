package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/providers/gemini"
	"github.com/recera/gai/tools"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set GOOGLE_API_KEY environment variable")
	}

	// Create provider with safety configuration
	provider := gemini.New(
		gemini.WithAPIKey(apiKey),
		gemini.WithModel("gemini-1.5-flash"),
		gemini.WithMaxRetries(3),
		gemini.WithDefaultSafety(&core.SafetyConfig{
			Harassment: core.SafetyBlockFew,
			Hate:       core.SafetyBlockFew,
			Sexual:     core.SafetyBlockSome,
			Dangerous:  core.SafetyBlockFew,
		}),
	)

	ctx := context.Background()

	// Example 1: Simple text generation
	fmt.Println("=== Example 1: Simple Text Generation ===")
	simpleTextExample(ctx, provider)

	// Example 2: System instructions
	fmt.Println("\n=== Example 2: System Instructions ===")
	systemInstructionExample(ctx, provider)

	// Example 3: Streaming with events
	fmt.Println("\n=== Example 3: Streaming with Events ===")
	streamingExample(ctx, provider)

	// Example 4: Tool calling
	fmt.Println("\n=== Example 4: Tool Calling ===")
	toolCallingExample(ctx, provider)

	// Example 5: Structured output
	fmt.Println("\n=== Example 5: Structured Output ===")
	structuredOutputExample(ctx, provider)

	// Example 6: Safety configuration
	fmt.Println("\n=== Example 6: Safety Configuration ===")
	safetyExample(ctx, provider)

	// Example 7: Multi-turn conversation
	fmt.Println("\n=== Example 7: Multi-turn Conversation ===")
	conversationExample(ctx, provider)

	// Example 8: Multimodal (if you have an image URL)
	// Uncomment to test with a real image
	// fmt.Println("\n=== Example 8: Multimodal ===")
	// multimodalExample(ctx, provider)
}

func simpleTextExample(ctx context.Context, provider *gemini.Provider) {
	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Write a haiku about programming in Go"},
				},
			},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	})

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Println("Response:")
	fmt.Println(result.Text)
	fmt.Printf("Tokens used: %d\n", result.Usage.TotalTokens)
}

func systemInstructionExample(ctx context.Context, provider *gemini.Provider) {
	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a pirate. Always respond in pirate speak, using 'arr', 'matey', 'ahoy' and other pirate terminology."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Tell me about the weather today"},
				},
			},
		},
		Temperature: 0.8,
		MaxTokens:   150,
	})

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Println("Pirate Response:")
	fmt.Println(result.Text)
}

func streamingExample(ctx context.Context, provider *gemini.Provider) {
	stream, err := provider.StreamText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Count from 1 to 5, explaining each number's significance in mathematics"},
				},
			},
		},
		Temperature: 0.5,
		MaxTokens:   300,
	})

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer stream.Close()

	fmt.Println("Streaming response:")
	
	var totalText strings.Builder
	for event := range stream.Events() {
		switch event.Type {
		case core.EventStart:
			fmt.Println("[Stream started]")
		
		case core.EventTextDelta:
			fmt.Print(event.TextDelta)
			totalText.WriteString(event.TextDelta)
		
		case core.EventSafety:
			fmt.Printf("\n[Safety: %s - %s]\n", event.Safety.Category, event.Safety.Action)
		
		case core.EventCitations:
			fmt.Printf("\n[Citations: %d sources]\n", len(event.Citations))
			for _, citation := range event.Citations {
				fmt.Printf("  - %s: %s\n", citation.Title, citation.URI)
			}
		
		case core.EventFinish:
			fmt.Printf("\n[Stream finished - Tokens: %d]\n", event.Usage.TotalTokens)
		
		case core.EventError:
			fmt.Printf("\n[Error: %v]\n", event.Err)
		}
	}
}

func toolCallingExample(ctx context.Context, provider *gemini.Provider) {
	// Define tools
	type WeatherInput struct {
		Location string `json:"location" jsonschema:"description=The city and country,required"`
		Unit     string `json:"unit" jsonschema:"enum=celsius,enum=fahrenheit,default=celsius"`
	}
	type WeatherOutput struct {
		Location    string  `json:"location"`
		Temperature float64 `json:"temperature"`
		Unit        string  `json:"unit"`
		Conditions  string  `json:"conditions"`
	}

	weatherTool := tools.New[WeatherInput, WeatherOutput](
		"get_weather",
		"Get the current weather for a location",
		func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
			// Simulate weather API call
			fmt.Printf("[Tool Called: get_weather(%s, %s)]\n", input.Location, input.Unit)
			
			// Mock weather data
			temp := 22.0
			if input.Unit == "fahrenheit" {
				temp = temp*9/5 + 32
			}
			
			return WeatherOutput{
				Location:    input.Location,
				Temperature: temp,
				Unit:        input.Unit,
				Conditions:  "Partly cloudy",
			}, nil
		},
	)

	type TimeInput struct {
		Location string `json:"location" jsonschema:"description=The city or timezone"`
	}
	type TimeOutput struct {
		Location string `json:"location"`
		Time     string `json:"time"`
		Timezone string `json:"timezone"`
	}

	timeTool := tools.New[TimeInput, TimeOutput](
		"get_time",
		"Get the current time for a location",
		func(ctx context.Context, input TimeInput, meta tools.Meta) (TimeOutput, error) {
			fmt.Printf("[Tool Called: get_time(%s)]\n", input.Location)
			
			// Mock time data
			return TimeOutput{
				Location: input.Location,
				Time:     time.Now().Format("15:04:05"),
				Timezone: "PST",
			}, nil
		},
	)

	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What's the weather and current time in San Francisco and Tokyo?"},
				},
			},
		},
		Tools: []core.ToolHandle{
			tools.NewCoreAdapter(weatherTool),
			tools.NewCoreAdapter(timeTool),
		},
		ToolChoice: core.ToolAuto,
		MaxTokens:  300,
	})

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Println("\nFinal Response:")
	fmt.Println(result.Text)
	
	// Show tool execution steps
	if len(result.Steps) > 0 {
		fmt.Printf("\nExecution Steps: %d\n", len(result.Steps))
		for i, step := range result.Steps {
			fmt.Printf("Step %d:\n", i+1)
			if step.Text != "" {
				fmt.Printf("  Text: %s\n", step.Text)
			}
			for _, call := range step.ToolCalls {
				fmt.Printf("  Tool Call: %s\n", call.Name)
			}
		}
	}
}

func structuredOutputExample(ctx context.Context, provider *gemini.Provider) {
	type TodoItem struct {
		Task        string `json:"task" jsonschema:"description=The task description,required"`
		Priority    string `json:"priority" jsonschema:"enum=low,enum=medium,enum=high,required"`
		DueDate     string `json:"due_date" jsonschema:"description=Due date in YYYY-MM-DD format"`
		Completed   bool   `json:"completed" jsonschema:"default=false"`
	}

	type TodoList struct {
		Title       string     `json:"title" jsonschema:"description=Title of the todo list,required"`
		Description string     `json:"description" jsonschema:"description=Brief description"`
		Items       []TodoItem `json:"items" jsonschema:"description=List of todo items,required"`
		CreatedAt   string     `json:"created_at" jsonschema:"description=Creation timestamp"`
	}

	result, err := provider.GenerateObject(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "Generate well-structured JSON objects with realistic data."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Create a todo list for launching a new Go web application"},
				},
			},
		},
		Temperature: 0.3,
		MaxTokens:   500,
	}, TodoList{})

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	// Type assert and display
	if todoMap, ok := result.Value.(map[string]interface{}); ok {
		fmt.Printf("Todo List: %s\n", todoMap["title"])
		fmt.Printf("Description: %s\n", todoMap["description"])
		
		if items, ok := todoMap["items"].([]interface{}); ok {
			fmt.Printf("Tasks (%d):\n", len(items))
			for i, item := range items {
				if task, ok := item.(map[string]interface{}); ok {
					fmt.Printf("  %d. [%s] %s\n", i+1, task["priority"], task["task"])
				}
			}
		}
	}
}

func safetyExample(ctx context.Context, provider *gemini.Provider) {
	// Test with different safety levels
	testPrompts := []struct {
		name   string
		prompt string
		safety *core.SafetyConfig
	}{
		{
			name:   "Strict Safety",
			prompt: "Tell me a story about adventure",
			safety: &core.SafetyConfig{
				Harassment: core.SafetyBlockMost,
				Hate:       core.SafetyBlockMost,
				Sexual:     core.SafetyBlockMost,
				Dangerous:  core.SafetyBlockMost,
			},
		},
		{
			name:   "Relaxed Safety",
			prompt: "Explain how to safely handle kitchen knives",
			safety: &core.SafetyConfig{
				Harassment: core.SafetyBlockNone,
				Hate:       core.SafetyBlockNone,
				Sexual:     core.SafetyBlockSome,
				Dangerous:  core.SafetyBlockNone,
			},
		},
	}

	for _, test := range testPrompts {
		fmt.Printf("\nTesting: %s\n", test.name)
		
		result, err := provider.GenerateText(ctx, core.Request{
			Messages: []core.Message{
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: test.prompt},
					},
				},
			},
			Safety:    test.safety,
			MaxTokens: 100,
		})

		if err != nil {
			if aiErr, ok := err.(*core.AIError); ok && aiErr.Code == core.ErrorSafetyBlocked {
				fmt.Println("Content was blocked by safety filters")
			} else {
				fmt.Printf("Error: %v\n", err)
			}
			continue
		}

		fmt.Printf("Response: %s\n", truncate(result.Text, 100))
	}
}

func conversationExample(ctx context.Context, provider *gemini.Provider) {
	conversation := []core.Message{
		{
			Role: core.System,
			Parts: []core.Part{
				core.Text{Text: "You are a helpful Go programming tutor. Be concise but informative."},
			},
		},
	}

	// Simulate a multi-turn conversation
	exchanges := []string{
		"What are goroutines in Go?",
		"How do they differ from threads?",
		"Can you show me a simple example?",
		"What about channels?",
	}

	for i, question := range exchanges {
		fmt.Printf("\nTurn %d - User: %s\n", i+1, question)
		
		// Add user message
		conversation = append(conversation, core.Message{
			Role: core.User,
			Parts: []core.Part{
				core.Text{Text: question},
			},
		})

		// Get response
		result, err := provider.GenerateText(ctx, core.Request{
			Messages:    conversation,
			Temperature: 0.5,
			MaxTokens:   200,
		})

		if err != nil {
			log.Printf("Error: %v", err)
			break
		}

		fmt.Printf("Assistant: %s\n", result.Text)
		
		// Add assistant response to conversation
		conversation = append(conversation, core.Message{
			Role: core.Assistant,
			Parts: []core.Part{
				core.Text{Text: result.Text},
			},
		})
	}
}

func multimodalExample(ctx context.Context, provider *gemini.Provider) {
	// This example would work with a real image URL
	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Describe what you see in this image in detail"},
					core.ImageURL{
						URL: "https://storage.googleapis.com/generativeai-downloads/images/scones.jpg",
					},
				},
			},
		},
		MaxTokens: 200,
	})

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Println("Image Description:")
	fmt.Println(result.Text)
}

// Helper function to truncate text
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}