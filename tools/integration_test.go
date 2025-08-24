package tools_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/recera/gai/core"
	"github.com/recera/gai/tools"
)

// TestIntegrationWithRunner tests the integration between tools and the core runner.
func TestIntegrationWithRunner(t *testing.T) {
	// Define test types
	type CalculatorInput struct {
		A        float64 `json:"a"`
		B        float64 `json:"b"`
		Operation string  `json:"operation"`
	}

	type CalculatorOutput struct {
		Result float64 `json:"result"`
		Expression string `json:"expression"`
	}

	// Create a calculator tool
	calcTool := tools.New[CalculatorInput, CalculatorOutput](
		"calculator",
		"Performs basic arithmetic operations",
		func(ctx context.Context, in CalculatorInput, meta tools.Meta) (CalculatorOutput, error) {
			var result float64
			var expression string

			switch in.Operation {
			case "add":
				result = in.A + in.B
				expression = fmt.Sprintf("%.2f + %.2f = %.2f", in.A, in.B, result)
			case "subtract":
				result = in.A - in.B
				expression = fmt.Sprintf("%.2f - %.2f = %.2f", in.A, in.B, result)
			case "multiply":
				result = in.A * in.B
				expression = fmt.Sprintf("%.2f * %.2f = %.2f", in.A, in.B, result)
			case "divide":
				if in.B == 0 {
					return CalculatorOutput{}, fmt.Errorf("division by zero")
				}
				result = in.A / in.B
				expression = fmt.Sprintf("%.2f / %.2f = %.2f", in.A, in.B, result)
			default:
				return CalculatorOutput{}, fmt.Errorf("unknown operation: %s", in.Operation)
			}

			return CalculatorOutput{
				Result:     result,
				Expression: expression,
			}, nil
		},
	)

	// Adapt the tool for use with core
	coreTool := tools.NewCoreAdapter(calcTool)

	// Test that it implements core.ToolHandle
	var _ core.ToolHandle = coreTool

	// Test tool properties
	if coreTool.Name() != "calculator" {
		t.Errorf("Expected name 'calculator', got '%s'", coreTool.Name())
	}

	if coreTool.Description() != "Performs basic arithmetic operations" {
		t.Errorf("Unexpected description: %s", coreTool.Description())
	}

	// Test schema generation
	inSchema := coreTool.InSchemaJSON()
	if len(inSchema) == 0 {
		t.Error("Input schema is empty")
	}

	outSchema := coreTool.OutSchemaJSON()
	if len(outSchema) == 0 {
		t.Error("Output schema is empty")
	}

	// Test execution through the core interface
	testCases := []struct {
		name     string
		input    json.RawMessage
		expected CalculatorOutput
		wantErr  bool
	}{
		{
			name:     "Addition",
			input:    json.RawMessage(`{"a": 5, "b": 3, "operation": "add"}`),
			expected: CalculatorOutput{Result: 8, Expression: "5.00 + 3.00 = 8.00"},
			wantErr:  false,
		},
		{
			name:     "Subtraction",
			input:    json.RawMessage(`{"a": 10, "b": 4, "operation": "subtract"}`),
			expected: CalculatorOutput{Result: 6, Expression: "10.00 - 4.00 = 6.00"},
			wantErr:  false,
		},
		{
			name:     "Multiplication",
			input:    json.RawMessage(`{"a": 7, "b": 6, "operation": "multiply"}`),
			expected: CalculatorOutput{Result: 42, Expression: "7.00 * 6.00 = 42.00"},
			wantErr:  false,
		},
		{
			name:     "Division",
			input:    json.RawMessage(`{"a": 15, "b": 3, "operation": "divide"}`),
			expected: CalculatorOutput{Result: 5, Expression: "15.00 / 3.00 = 5.00"},
			wantErr:  false,
		},
		{
			name:    "Division by zero",
			input:   json.RawMessage(`{"a": 10, "b": 0, "operation": "divide"}`),
			wantErr: true,
		},
		{
			name:    "Unknown operation",
			input:   json.RawMessage(`{"a": 1, "b": 2, "operation": "power"}`),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create meta as the runner would
			meta := map[string]interface{}{
				"call_id":     "test-call",
				"messages":    []core.Message{},
				"step_number": 1,
			}

			result, err := coreTool.Exec(context.Background(), tc.input, meta)

			if tc.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			output, ok := result.(CalculatorOutput)
			if !ok {
				t.Fatalf("Wrong output type: %T", result)
			}

			if output.Result != tc.expected.Result {
				t.Errorf("Result mismatch: got %f, want %f", output.Result, tc.expected.Result)
			}

			if output.Expression != tc.expected.Expression {
				t.Errorf("Expression mismatch: got %s, want %s", output.Expression, tc.expected.Expression)
			}
		})
	}
}

func TestMultipleToolsIntegration(t *testing.T) {
	// Create multiple tools
	type GreetInput struct {
		Name string `json:"name"`
		Lang string `json:"lang"`
	}

	type GreetOutput struct {
		Greeting string `json:"greeting"`
	}

	greetTool := tools.New[GreetInput, GreetOutput](
		"greet",
		"Greets a person in different languages",
		func(ctx context.Context, in GreetInput, meta tools.Meta) (GreetOutput, error) {
			var greeting string
			switch in.Lang {
			case "en":
				greeting = fmt.Sprintf("Hello, %s!", in.Name)
			case "es":
				greeting = fmt.Sprintf("¡Hola, %s!", in.Name)
			case "fr":
				greeting = fmt.Sprintf("Bonjour, %s!", in.Name)
			default:
				greeting = fmt.Sprintf("Hi, %s!", in.Name)
			}
			return GreetOutput{Greeting: greeting}, nil
		},
	)

	type TimeInput struct {
		Timezone string `json:"timezone"`
	}

	type TimeOutput struct {
		Time     string `json:"time"`
		Timezone string `json:"timezone"`
	}

	timeTool := tools.New[TimeInput, TimeOutput](
		"get_time",
		"Gets the current time in a timezone",
		func(ctx context.Context, in TimeInput, meta tools.Meta) (TimeOutput, error) {
			// Simulated time for testing
			return TimeOutput{
				Time:     "2024-01-15 10:30:00",
				Timezone: in.Timezone,
			}, nil
		},
	)

	// Register tools
	registry := tools.NewRegistry()
	if err := registry.Register(greetTool); err != nil {
		t.Fatalf("Failed to register greet tool: %v", err)
	}
	if err := registry.Register(timeTool); err != nil {
		t.Fatalf("Failed to register time tool: %v", err)
	}

	// Convert to core handles
	allTools := registry.All()
	coreHandles := tools.ToCoreHandles(allTools)

	// Verify we have the right number of tools
	if len(coreHandles) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(coreHandles))
	}

	// Test that we can find and execute tools by name
	for _, handle := range coreHandles {
		switch handle.Name() {
		case "greet":
			input := json.RawMessage(`{"name": "Alice", "lang": "es"}`)
			result, err := handle.Exec(context.Background(), input, nil)
			if err != nil {
				t.Errorf("Failed to execute greet tool: %v", err)
			}
			
			output, ok := result.(GreetOutput)
			if !ok {
				t.Errorf("Wrong output type for greet: %T", result)
			}
			
			if output.Greeting != "¡Hola, Alice!" {
				t.Errorf("Unexpected greeting: %s", output.Greeting)
			}

		case "get_time":
			input := json.RawMessage(`{"timezone": "UTC"}`)
			result, err := handle.Exec(context.Background(), input, nil)
			if err != nil {
				t.Errorf("Failed to execute time tool: %v", err)
			}
			
			output, ok := result.(TimeOutput)
			if !ok {
				t.Errorf("Wrong output type for time: %T", result)
			}
			
			if output.Timezone != "UTC" {
				t.Errorf("Unexpected timezone: %s", output.Timezone)
			}
		}
	}
}

func TestToolWithMetaAccess(t *testing.T) {
	type MetaTestInput struct {
		Query string `json:"query"`
	}

	type MetaTestOutput struct {
		CallID     string `json:"call_id"`
		StepNumber int    `json:"step_number"`
		Provider   string `json:"provider"`
		Query      string `json:"query"`
	}

	// Create a tool that uses meta information
	metaTool := tools.New[MetaTestInput, MetaTestOutput](
		"meta_test",
		"Tool that accesses meta information",
		func(ctx context.Context, in MetaTestInput, meta tools.Meta) (MetaTestOutput, error) {
			return MetaTestOutput{
				CallID:     meta.CallID,
				StepNumber: meta.StepNumber,
				Provider:   meta.Provider,
				Query:      in.Query,
			}, nil
		},
	)

	// Adapt for core
	coreTool := tools.NewCoreAdapter(metaTool)

	// Create rich meta information
	messages := []core.Message{
		{
			Role: core.User,
			Parts: []core.Part{
				core.Text{Text: "Test message 1"},
			},
		},
		{
			Role: core.Assistant,
			Parts: []core.Part{
				core.Text{Text: "Test response 1"},
			},
		},
	}

	meta := map[string]interface{}{
		"call_id":     "test-123",
		"messages":    messages,
		"step_number": 5,
		"provider":   "test-provider",
		"metadata": map[string]any{
			"session_id": "sess-456",
			"user_id":    "user-789",
		},
	}

	// Execute with meta
	input := json.RawMessage(`{"query": "test query"}`)
	result, err := coreTool.Exec(context.Background(), input, meta)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	output, ok := result.(MetaTestOutput)
	if !ok {
		t.Fatalf("Wrong output type: %T", result)
	}

	// Verify meta was passed correctly
	if output.CallID != "test-123" {
		t.Errorf("Expected CallID 'test-123', got '%s'", output.CallID)
	}

	if output.StepNumber != 5 {
		t.Errorf("Expected StepNumber 5, got %d", output.StepNumber)
	}

	if output.Provider != "test-provider" {
		t.Errorf("Expected Provider 'test-provider', got '%s'", output.Provider)
	}

	if output.Query != "test query" {
		t.Errorf("Expected Query 'test query', got '%s'", output.Query)
	}
}