package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/tools"
)

// skipIfNoAPI skips the test if no API key is provided
func skipIfNoAPI(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
	}
}

// newTestProvider creates a provider for integration tests
func newTestProvider(t *testing.T) *Provider {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Fatal("ANTHROPIC_API_KEY environment variable is required for integration tests")
	}
	
	return New(
		WithAPIKey(apiKey),
		WithModel("claude-3-haiku-20240307"), // Use a fast, cheap model for tests
	)
}

func TestIntegrationGenerateText(t *testing.T) {
	skipIfNoAPI(t)
	
	p := newTestProvider(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		request  core.Request
		validate func(t *testing.T, result *core.TextResult, err error)
	}{
		{
			name: "simple text generation",
			request: core.Request{
				Messages: []core.Message{
					{
						Role:  core.User,
						Parts: []core.Part{core.Text{Text: "Say hello in exactly 2 words."}},
					},
				},
				MaxTokens: 50,
			},
			validate: func(t *testing.T, result *core.TextResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result.Text == "" {
					t.Error("expected non-empty text response")
				}
				if result.Usage.InputTokens == 0 {
					t.Error("expected non-zero input tokens")
				}
				if result.Usage.OutputTokens == 0 {
					t.Error("expected non-zero output tokens")
				}
				if result.Usage.TotalTokens != result.Usage.InputTokens+result.Usage.OutputTokens {
					t.Error("total tokens should equal input + output tokens")
				}
				t.Logf("Response: %s", result.Text)
				t.Logf("Usage: %+v", result.Usage)
			},
		},
		{
			name: "with system prompt",
			request: core.Request{
				Messages: []core.Message{
					{
						Role:  core.System,
						Parts: []core.Part{core.Text{Text: "You are a helpful assistant that responds concisely."}},
					},
					{
						Role:  core.User,
						Parts: []core.Part{core.Text{Text: "What is the capital of France?"}},
					},
				},
				MaxTokens: 50,
			},
			validate: func(t *testing.T, result *core.TextResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result.Text == "" {
					t.Error("expected non-empty text response")
				}
				// Should mention Paris
				if !strings.Contains(strings.ToLower(result.Text), "paris") {
					t.Logf("Response: %s", result.Text)
					t.Error("expected response to mention Paris")
				}
				t.Logf("Response: %s", result.Text)
			},
		},
		{
			name: "conversation context",
			request: core.Request{
				Messages: []core.Message{
					{
						Role:  core.User,
						Parts: []core.Part{core.Text{Text: "My name is Alice."}},
					},
					{
						Role:  core.Assistant,
						Parts: []core.Part{core.Text{Text: "Hello Alice! Nice to meet you."}},
					},
					{
						Role:  core.User,
						Parts: []core.Part{core.Text{Text: "What's my name?"}},
					},
				},
				MaxTokens: 50,
			},
			validate: func(t *testing.T, result *core.TextResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result.Text == "" {
					t.Error("expected non-empty text response")
				}
				// Should remember the name Alice
				if !strings.Contains(result.Text, "Alice") {
					t.Logf("Response: %s", result.Text)
					t.Error("expected response to remember the name Alice")
				}
				t.Logf("Response: %s", result.Text)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.GenerateText(ctx, tt.request)
			tt.validate(t, result, err)
		})
	}
}

func TestIntegrationStreamText(t *testing.T) {
	skipIfNoAPI(t)
	
	p := newTestProvider(t)
	ctx := context.Background()

	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Count from 1 to 5, one number per line."}},
			},
		},
		MaxTokens: 100,
		Stream:    true,
	}

	stream, err := p.StreamText(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer stream.Close()

	var events []core.Event
	var textParts []string
	hasStart := false
	hasFinish := false

	for event := range stream.Events() {
		events = append(events, event)
		
		switch event.Type {
		case core.EventStart:
			hasStart = true
		case core.EventTextDelta:
			textParts = append(textParts, event.TextDelta)
			t.Logf("Text delta: %q", event.TextDelta)
		case core.EventFinish:
			hasFinish = true
			if event.Usage != nil {
				t.Logf("Final usage: %+v", *event.Usage)
			}
		case core.EventError:
			t.Fatalf("stream error: %v", event.Err)
		}
	}

	if !hasStart {
		t.Error("expected start event")
	}
	if !hasFinish {
		t.Error("expected finish event")
	}
	if len(events) == 0 {
		t.Error("expected to receive events")
	}
	if len(textParts) == 0 {
		t.Error("expected to receive text deltas")
	}

	fullText := strings.Join(textParts, "")
	t.Logf("Complete response: %s", fullText)
	
	// Should contain numbers 1-5
	for i := 1; i <= 5; i++ {
		if !strings.Contains(fullText, string(rune('0'+i))) {
			t.Errorf("expected response to contain number %d", i)
		}
	}
}

func TestIntegrationGenerateObject(t *testing.T) {
	skipIfNoAPI(t)
	
	p := newTestProvider(t)
	ctx := context.Background()

	// Define a schema for structured output
	type PersonInfo struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
		City string `json:"city"`
	}

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
			"age": map[string]interface{}{
				"type": "integer",
			},
			"city": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"name", "age", "city"},
	}

	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Generate information for a fictional person named John who is 30 years old and lives in New York."}},
			},
		},
		MaxTokens: 200,
	}

	result, err := p.GenerateObject(ctx, req, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Value == nil {
		t.Fatal("expected non-nil value")
	}

	// Try to parse the result as our expected structure
	jsonBytes, err := json.Marshal(result.Value)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}

	var person PersonInfo
	if err := json.Unmarshal(jsonBytes, &person); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	t.Logf("Generated person: %+v", person)
	t.Logf("Usage: %+v", result.Usage)

	if person.Name == "" {
		t.Error("expected non-empty name")
	}
	if person.Age == 0 {
		t.Error("expected non-zero age")
	}
	if person.City == "" {
		t.Error("expected non-empty city")
	}
}

func TestIntegrationWithTools(t *testing.T) {
	skipIfNoAPI(t)
	
	p := newTestProvider(t)
	ctx := context.Background()

	// Create a simple calculator tool
	calculatorTool := tools.New("calculator", "Performs basic arithmetic operations",
		func(ctx context.Context, input struct {
			Operation string  `json:"operation" description:"The operation to perform (add, subtract, multiply, divide)"`
			A         float64 `json:"a" description:"First number"`
			B         float64 `json:"b" description:"Second number"`
		}, meta tools.Meta) (map[string]interface{}, error) {
			var result float64
			switch input.Operation {
			case "add":
				result = input.A + input.B
			case "subtract":
				result = input.A - input.B
			case "multiply":
				result = input.A * input.B
			case "divide":
				if input.B == 0 {
					return nil, fmt.Errorf("cannot divide by zero")
				}
				result = input.A / input.B
			default:
				return nil, fmt.Errorf("unknown operation: %s", input.Operation)
			}
			
			return map[string]interface{}{
				"result":    result,
				"operation": input.Operation,
				"a":         input.A,
				"b":         input.B,
			}, nil
		})

	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "What is 15 + 27? Use the calculator tool to compute this."}},
			},
		},
		Tools:     tools.ToCoreHandles([]tools.Handle{calculatorTool}),
		MaxTokens: 300,
		StopWhen:  core.NoMoreTools(),
	}

	result, err := p.GenerateText(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Final response: %s", result.Text)
	t.Logf("Usage: %+v", result.Usage)
	t.Logf("Steps: %d", len(result.Steps))

	if len(result.Steps) == 0 {
		t.Error("expected at least one step with tool usage")
		return
	}

	// Check that tools were called
	hasToolCall := false
	hasToolResult := false
	for _, step := range result.Steps {
		t.Logf("Step %d: %s", step.StepNumber, step.Text)
		if len(step.ToolCalls) > 0 {
			hasToolCall = true
			for _, call := range step.ToolCalls {
				t.Logf("  Tool call: %s(%s)", call.Name, string(call.Input))
			}
		}
		if len(step.ToolResults) > 0 {
			hasToolResult = true
			for _, res := range step.ToolResults {
				t.Logf("  Tool result: %v", res.Result)
			}
		}
	}

	if !hasToolCall {
		t.Error("expected tool calls")
	}
	if !hasToolResult {
		t.Error("expected tool results")
	}

	// The result should mention the answer (42)
	if !strings.Contains(result.Text, "42") {
		t.Logf("Response: %s", result.Text)
		t.Error("expected response to contain the answer 42")
	}
}

func TestIntegrationErrorHandling(t *testing.T) {
	skipIfNoAPI(t)
	
	// Test with invalid API key
	invalidProvider := New(
		WithAPIKey("invalid-key-12345"),
		WithModel("claude-3-haiku-20240307"),
	)

	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Hello"}},
			},
		},
		MaxTokens: 50,
	}

	_, err := invalidProvider.GenerateText(ctx, req)
	if err == nil {
		t.Error("expected error with invalid API key")
		return
	}

	// Check that it's properly categorized
	if !core.IsAuth(err) {
		t.Errorf("expected authentication error, got: %v", err)
	}

	var aiErr *core.AIError
	if e, ok := err.(*core.AIError); ok {
		aiErr = e
		if aiErr.Provider != "anthropic" {
			t.Errorf("expected provider 'anthropic', got %q", aiErr.Provider)
		}
		if aiErr.Code != core.ErrorUnauthorized {
			t.Errorf("expected error code %v, got %v", core.ErrorUnauthorized, aiErr.Code)
		}
		t.Logf("Error properly categorized: %+v", aiErr)
	} else {
		t.Errorf("expected *core.AIError, got %T", err)
	}
}

func TestIntegrationRateLimiting(t *testing.T) {
	skipIfNoAPI(t)
	
	p := newTestProvider(t)
	ctx := context.Background()

	// Make multiple rapid requests to potentially trigger rate limiting
	// Note: This test might not always trigger rate limits depending on the account
	requests := 5
	errors := 0

	for i := 0; i < requests; i++ {
		req := core.Request{
			Messages: []core.Message{
				{
					Role:  core.User,
					Parts: []core.Part{core.Text{Text: fmt.Sprintf("Request %d: Say 'ok'", i+1)}},
				},
			},
			MaxTokens: 10,
		}

		_, err := p.GenerateText(ctx, req)
		if err != nil {
			errors++
			t.Logf("Request %d error: %v", i+1, err)
			
			// Check if it's a rate limit error
			if core.IsRateLimited(err) {
				t.Logf("Rate limit error properly detected")
				
				// Check retry-after
				retryAfter := core.GetRetryAfter(err)
				if retryAfter > 0 {
					t.Logf("Retry after: %v", retryAfter)
				}
			}
		} else {
			t.Logf("Request %d: success", i+1)
		}
		
		// Small delay between requests
		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("Completed %d requests with %d errors", requests, errors)
	// We don't fail the test for rate limiting since it depends on usage patterns
}