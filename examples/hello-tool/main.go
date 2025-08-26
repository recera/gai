// Package main demonstrates tool calling with the GAI framework.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/middleware"
	"github.com/recera/gai/providers/openai"
	"github.com/recera/gai/tools"
)

// Set environment variable:
// export OPENAI_API_KEY="your-api-key-here"

// Define input/output types for our tools

// WeatherInput represents weather query parameters
type WeatherInput struct {
	Location string `json:"location" jsonschema:"description=City name or coordinates"`
}

// WeatherOutput represents weather data
type WeatherOutput struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature_celsius"`
	Condition   string  `json:"condition"`
	Humidity    int     `json:"humidity_percent"`
	WindSpeed   float64 `json:"wind_speed_kmh"`
}

// CalculatorInput represents a math expression
type CalculatorInput struct {
	Expression string `json:"expression" jsonschema:"description=Mathematical expression to evaluate"`
}

// CalculatorOutput represents calculation result
type CalculatorOutput struct {
	Expression string  `json:"expression"`
	Result     float64 `json:"result"`
}

// SearchInput represents search parameters
type SearchInput struct {
	Query string `json:"query" jsonschema:"description=Search query"`
	Limit int    `json:"limit,omitempty" jsonschema:"description=Maximum number of results,default=5"`
}

// SearchOutput represents search results
type SearchOutput struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
}

// EmailInput represents email parameters
type EmailInput struct {
	To      string `json:"to" jsonschema:"description=Recipient email address"`
	Subject string `json:"subject" jsonschema:"description=Email subject"`
	Body    string `json:"body" jsonschema:"description=Email body content"`
}

// EmailOutput represents email sending result
type EmailOutput struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

// DatabaseInput represents database query parameters
type DatabaseInput struct {
	Table string `json:"table" jsonschema:"description=Table name"`
	Query string `json:"query" jsonschema:"description=Query type: select, count, list"`
	Field string `json:"field,omitempty" jsonschema:"description=Field to query"`
	Value string `json:"value,omitempty" jsonschema:"description=Value to match"`
}

// DatabaseOutput represents database query results
type DatabaseOutput struct {
	Table   string                   `json:"table"`
	Query   string                   `json:"query"`
	Count   int                      `json:"count,omitempty"`
	Records []map[string]interface{} `json:"records,omitempty"`
}

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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Example 1: Simple tool calling
	fmt.Println("=== Example 1: Simple Tool Calling ===\n")
	simpleToolExample(ctx, provider)

	// Example 2: Multiple tools in one request
	fmt.Println("\n=== Example 2: Multiple Tools ===\n")
	multipleToolsExample(ctx, provider)

	// Example 3: Multi-step tool execution
	fmt.Println("\n=== Example 3: Multi-Step Tool Execution ===\n")
	multiStepExample(ctx, provider)

	// Example 4: Streaming with tools
	fmt.Println("\n=== Example 4: Streaming with Tools ===\n")
	streamingToolExample(ctx, provider)
}

func simpleToolExample(ctx context.Context, provider core.Provider) {
	// Create a weather tool
	weatherTool := tools.New[WeatherInput, WeatherOutput](
		"get_weather",
		"Get current weather for a location",
		func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
			fmt.Printf("🔧 Tool called: get_weather(%s)\n", input.Location)

			// Simulate weather data (in production, call a real API)
			conditions := []string{"Sunny", "Cloudy", "Rainy", "Partly Cloudy"}
			return WeatherOutput{
				Location:    input.Location,
				Temperature: 15 + rand.Float64()*20,
				Condition:   conditions[rand.Intn(len(conditions))],
				Humidity:    40 + rand.Intn(40),
				WindSpeed:   5 + rand.Float64()*25,
			}, nil
		},
	)

	// Convert to core handle
	coreTools := tools.ToCoreHandles([]tools.Handle{weatherTool})

	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What's the weather like in San Francisco and Tokyo?"},
				},
			},
		},
		Tools:      coreTools,
		ToolChoice: core.ToolAuto,
	}

	result, err := provider.GenerateText(ctx, request)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Println("\nAssistant's Response:")
	fmt.Println(result.Text)

	// Show execution steps
	if len(result.Steps) > 0 {
		fmt.Println("\nExecution Steps:")
		for i, step := range result.Steps {
			fmt.Printf("Step %d:\n", i+1)
			if step.Text != "" {
				fmt.Printf("  Text: %s\n", step.Text)
			}
			for _, call := range step.ToolCalls {
				fmt.Printf("  Tool Call: %s\n", call.Name)
				fmt.Printf("    Input: %s\n", string(call.Input))
			}
			for _, result := range step.ToolResults {
				output, _ := json.MarshalIndent(result.Result, "    ", "  ")
				fmt.Printf("    Result: %s\n", string(output))
			}
		}
	}
}

func multipleToolsExample(ctx context.Context, provider core.Provider) {
	// Create multiple tools
	calculatorTool := tools.New[CalculatorInput, CalculatorOutput](
		"calculator",
		"Evaluate mathematical expressions",
		func(ctx context.Context, input CalculatorInput, meta tools.Meta) (CalculatorOutput, error) {
			fmt.Printf("🔧 Tool called: calculator(%s)\n", input.Expression)

			// Simple evaluation (in production, use a proper expression parser)
			result := 0.0
			expr := strings.TrimSpace(input.Expression)

			// Handle basic operations
			if strings.Contains(expr, "+") {
				parts := strings.Split(expr, "+")
				if len(parts) == 2 {
					var a, b float64
					fmt.Sscanf(parts[0], "%f", &a)
					fmt.Sscanf(parts[1], "%f", &b)
					result = a + b
				}
			} else if strings.Contains(expr, "*") {
				parts := strings.Split(expr, "*")
				if len(parts) == 2 {
					var a, b float64
					fmt.Sscanf(parts[0], "%f", &a)
					fmt.Sscanf(parts[1], "%f", &b)
					result = a * b
				}
			} else if expr == "sqrt(144)" {
				result = math.Sqrt(144)
			} else {
				// Default calculation
				result = 42
			}

			return CalculatorOutput{
				Expression: input.Expression,
				Result:     result,
			}, nil
		},
	)

	searchTool := tools.New[SearchInput, SearchOutput](
		"search",
		"Search for information on the web",
		func(ctx context.Context, input SearchInput, meta tools.Meta) (SearchOutput, error) {
			fmt.Printf("🔧 Tool called: search(%s)\n", input.Query)

			limit := input.Limit
			if limit == 0 {
				limit = 3
			}

			// Simulate search results
			results := []SearchResult{}
			for i := 0; i < limit && i < 3; i++ {
				results = append(results, SearchResult{
					Title:   fmt.Sprintf("Result %d for: %s", i+1, input.Query),
					Snippet: fmt.Sprintf("This is a relevant snippet about %s...", input.Query),
					URL:     fmt.Sprintf("https://example.com/result%d", i+1),
				})
			}

			return SearchOutput{
				Query:   input.Query,
				Results: results,
			}, nil
		},
	)

	// Convert to core handles
	coreTools := tools.ToCoreHandles([]tools.Handle{calculatorTool, searchTool})

	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What's the square root of 144? Also, search for information about the Pythagorean theorem."},
				},
			},
		},
		Tools:      coreTools,
		ToolChoice: core.ToolAuto,
	}

	result, err := provider.GenerateText(ctx, request)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Println("Assistant's Response:")
	fmt.Println(result.Text)

	fmt.Printf("\nTools Called: %d\n", countToolCalls(result.Steps))
	fmt.Printf("Total Tokens: %d\n", result.Usage.TotalTokens)
}

func multiStepExample(ctx context.Context, provider core.Provider) {
	// Create tools that can work together
	databaseTool := tools.New[DatabaseInput, DatabaseOutput](
		"database",
		"Query the customer database",
		func(ctx context.Context, input DatabaseInput, meta tools.Meta) (DatabaseOutput, error) {
			fmt.Printf("🔧 Tool called: database(table=%s, query=%s)\n", input.Table, input.Query)

			// Simulate database operations
			output := DatabaseOutput{
				Table: input.Table,
				Query: input.Query,
			}

			switch input.Query {
			case "count":
				output.Count = 1247
			case "list":
				output.Records = []map[string]interface{}{
					{"id": 1, "name": "Alice Johnson", "email": "alice@example.com", "status": "active"},
					{"id": 2, "name": "Bob Smith", "email": "bob@example.com", "status": "pending"},
					{"id": 3, "name": "Carol White", "email": "carol@example.com", "status": "active"},
				}
			case "select":
				output.Records = []map[string]interface{}{
					{"id": 2, "name": "Bob Smith", "email": "bob@example.com", "status": "pending"},
				}
			}

			return output, nil
		},
	)

	emailTool := tools.New[EmailInput, EmailOutput](
		"send_email",
		"Send an email to a recipient",
		func(ctx context.Context, input EmailInput, meta tools.Meta) (EmailOutput, error) {
			fmt.Printf("🔧 Tool called: send_email(to=%s, subject=%s)\n", input.To, input.Subject)

			// Simulate email sending
			return EmailOutput{
				Success:   true,
				MessageID: fmt.Sprintf("msg_%d", time.Now().Unix()),
			}, nil
		},
	)

	// Convert to core handles
	coreTools := tools.ToCoreHandles([]tools.Handle{databaseTool, emailTool})

	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a customer service assistant. Help with customer queries using the available tools."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Find all customers with status 'pending' and send them a welcome email."},
				},
			},
		},
		Tools:      coreTools,
		ToolChoice: core.ToolAuto,
		StopWhen:   core.MaxSteps(5), // Limit to 5 steps
	}

	fmt.Println("Executing multi-step request...")
	fmt.Println("This will:")
	fmt.Println("1. Query the database for pending customers")
	fmt.Println("2. Send emails to each customer")
	fmt.Println()

	result, err := provider.GenerateText(ctx, request)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Println("Assistant's Response:")
	fmt.Println(result.Text)

	// Show detailed execution flow
	fmt.Println("\nDetailed Execution Flow:")
	for i, step := range result.Steps {
		fmt.Printf("\n=== Step %d ===\n", i+1)
		if step.Text != "" {
			fmt.Printf("Reasoning: %s\n", step.Text)
		}

		for j, call := range step.ToolCalls {
			fmt.Printf("\nTool Call %d.%d: %s\n", i+1, j+1, call.Name)

			var prettyInput map[string]interface{}
			json.Unmarshal(call.Input, &prettyInput)
			inputJSON, _ := json.MarshalIndent(prettyInput, "  ", "  ")
			fmt.Printf("  Input:\n  %s\n", string(inputJSON))
		}

		for j, result := range step.ToolResults {
			fmt.Printf("\nTool Result %d.%d:\n", i+1, j+1)
			resultJSON, _ := json.MarshalIndent(result.Result, "  ", "  ")
			fmt.Printf("  %s\n", string(resultJSON))
		}
	}

	fmt.Printf("\nTotal Steps: %d\n", len(result.Steps))
	fmt.Printf("Total Tool Calls: %d\n", countToolCalls(result.Steps))
}

func streamingToolExample(ctx context.Context, provider core.Provider) {
	// Create a tool that will be called during streaming
	weatherTool := tools.New[WeatherInput, WeatherOutput](
		"get_weather",
		"Get current weather for a location",
		func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
			fmt.Printf("\n🔧 Tool called during stream: get_weather(%s)\n", input.Location)

			// Add a small delay to simulate API call
			time.Sleep(500 * time.Millisecond)

			return WeatherOutput{
				Location:    input.Location,
				Temperature: 22.5,
				Condition:   "Partly Cloudy",
				Humidity:    65,
				WindSpeed:   12.3,
			}, nil
		},
	)

	// Convert to core handle
	coreTools := tools.ToCoreHandles([]tools.Handle{weatherTool})

	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Check the weather in Paris and tell me if it's good for a picnic."},
				},
			},
		},
		Tools:      coreTools,
		ToolChoice: core.ToolAuto,
		Stream:     true,
	}

	stream, err := provider.StreamText(ctx, request)
	if err != nil {
		log.Printf("Error creating stream: %v", err)
		return
	}
	defer stream.Close()

	fmt.Println("Streaming response with tool calls:")
	fmt.Println(strings.Repeat("-", 50))

	var fullText strings.Builder
	toolCallDetected := false

	for event := range stream.Events() {
		switch event.Type {
		case core.EventTextDelta:
			fmt.Print(event.TextDelta)
			fullText.WriteString(event.TextDelta)

		case core.EventToolCall:
			if !toolCallDetected {
				fmt.Println("\n[Tool call initiated...]")
				toolCallDetected = true
			}
			fmt.Printf("  Calling: %s\n", event.ToolName)

		case core.EventToolResult:
			fmt.Printf("  Tool result received for: %s\n", event.ToolName)
			resultJSON, _ := json.MarshalIndent(event.ToolResult, "    ", "  ")
			fmt.Printf("    %s\n", string(resultJSON))
			fmt.Println("[Resuming response...]")

		case core.EventFinish:
			fmt.Println("\n" + strings.Repeat("-", 50))
			fmt.Println("Stream completed")
			if event.Usage != nil {
				fmt.Printf("Total tokens used: %d\n", event.Usage.TotalTokens)
			}

		case core.EventError:
			fmt.Printf("\nStream error: %v\n", event.Err)
		}
	}
}

// Helper function to count tool calls in steps
func countToolCalls(steps []core.Step) int {
	count := 0
	for _, step := range steps {
		count += len(step.ToolCalls)
	}
	return count
}
