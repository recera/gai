// +build integration

package openai

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/tools"
)

// skipIfNoAPIKey skips the test if OPENAI_API_KEY is not set.
func skipIfNoAPIKey(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping integration test: OPENAI_API_KEY not set")
	}
}

func TestIntegrationGenerateText(t *testing.T) {
	skipIfNoAPIKey(t)

	p := New(
		WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		WithModel("gpt-4o-mini"),
	)

	tests := []struct {
		name    string
		request core.Request
		check   func(*testing.T, *core.TextResult)
	}{
		{
			name: "simple completion",
			request: core.Request{
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "Say 'Hello, World!' and nothing else."},
						},
					},
				},
				MaxTokens:   intPtr(20),
				Temperature: floatPtr(0),
			},
			check: func(t *testing.T, result *core.TextResult) {
				if result.Text == "" {
					t.Error("Expected non-empty response")
				}
				if result.Usage.TotalTokens == 0 {
					t.Error("Expected token usage information")
				}
			},
		},
		{
			name: "with system message",
			request: core.Request{
				Messages: []core.Message{
					{
						Role: core.System,
						Parts: []core.Part{
							core.Text{Text: "You are a pirate. Respond in pirate speak."},
						},
					},
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "How are you today?"},
						},
					},
				},
				MaxTokens:   intPtr(50),
				Temperature: floatPtr(0.7),
			},
			check: func(t *testing.T, result *core.TextResult) {
				if result.Text == "" {
					t.Error("Expected non-empty response")
				}
				// Could check for pirate-like words but that's non-deterministic
			},
		},
		{
			name: "conversation history",
			request: core.Request{
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "My name is Alice."},
						},
					},
					{
						Role: core.Assistant,
						Parts: []core.Part{
							core.Text{Text: "Nice to meet you, Alice!"},
						},
					},
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "What's my name?"},
						},
					},
				},
				MaxTokens:   intPtr(30),
				Temperature: floatPtr(0),
			},
			check: func(t *testing.T, result *core.TextResult) {
				if result.Text == "" {
					t.Error("Expected non-empty response")
				}
				// Response should mention "Alice"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			result, err := p.GenerateText(ctx, tt.request)
			if err != nil {
				t.Fatalf("GenerateText failed: %v", err)
			}

			tt.check(t, result)
		})
	}
}

func TestIntegrationStreamText(t *testing.T) {
	skipIfNoAPIKey(t)

	p := New(
		WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		WithModel("gpt-4o-mini"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := p.StreamText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Count from 1 to 5 slowly."},
				},
			},
		},
		MaxTokens:   intPtr(50),
		Temperature: floatPtr(0),
		Stream:      true,
	})

	if err != nil {
		t.Fatalf("StreamText failed: %v", err)
	}
	defer stream.Close()

	// Collect events
	var (
		textContent string
		eventCount  int
		hasStart    bool
		hasFinish   bool
		usage       *core.Usage
	)

	for event := range stream.Events() {
		eventCount++
		switch event.Type {
		case core.EventStart:
			hasStart = true
		case core.EventTextDelta:
			textContent += event.TextDelta
		case core.EventFinish:
			hasFinish = true
			usage = event.Usage
		case core.EventError:
			t.Errorf("Stream error: %v", event.Err)
		}
	}

	// Validate stream results
	if !hasStart {
		t.Error("Missing start event")
	}
	if !hasFinish {
		t.Error("Missing finish event")
	}
	if textContent == "" {
		t.Error("No text content received")
	}
	if usage == nil || usage.TotalTokens == 0 {
		t.Error("Missing or invalid usage information")
	}
	if eventCount < 3 { // At least start, one text delta, and finish
		t.Errorf("Too few events: %d", eventCount)
	}
}

func TestIntegrationWithTools(t *testing.T) {
	skipIfNoAPIKey(t)

	p := New(
		WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		WithModel("gpt-4o-mini"),
	)

	// Define a calculator tool
	type CalcInput struct {
		Expression string `json:"expression" jsonschema:"description=Mathematical expression to evaluate"`
	}
	type CalcOutput struct {
		Result float64 `json:"result"`
	}

	calcTool := tools.New[CalcInput, CalcOutput](
		"calculator",
		"Evaluates mathematical expressions",
		func(ctx context.Context, in CalcInput, meta tools.Meta) (CalcOutput, error) {
			// Simple evaluation (in production, use a proper expression evaluator)
			switch in.Expression {
			case "2 + 2":
				return CalcOutput{Result: 4}, nil
			case "10 * 5":
				return CalcOutput{Result: 50}, nil
			default:
				return CalcOutput{Result: 0}, nil
			}
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := p.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What is 2 + 2?"},
				},
			},
		},
		Tools:      []core.ToolHandle{tools.NewCoreAdapter(calcTool)},
		ToolChoice: core.ToolAuto,
		MaxTokens:  intPtr(100),
	})

	if err != nil {
		t.Fatalf("GenerateText with tools failed: %v", err)
	}

	// Check for tool calls
	if len(result.Steps) > 0 {
		hasToolCall := false
		for _, step := range result.Steps {
			if len(step.ToolCalls) > 0 {
				hasToolCall = true
				break
			}
		}
		if !hasToolCall {
			t.Log("Warning: Model did not use the tool (this can happen occasionally)")
		}
	}

	if result.Text == "" {
		t.Error("Expected final text response")
	}
}

func TestIntegrationGenerateObject(t *testing.T) {
	skipIfNoAPIKey(t)

	// Note: Structured outputs require specific models
	p := New(
		WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		WithModel("gpt-4o-mini"), // Supports structured outputs
	)

	type Person struct {
		Name    string `json:"name"`
		Age     int    `json:"age"`
		City    string `json:"city"`
		Hobbies []string `json:"hobbies"`
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
			"hobbies": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
		},
		"required": []string{"name", "age", "city", "hobbies"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := p.GenerateObject(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Generate a person profile for a software engineer living in San Francisco who is 28 years old and likes hiking and coding."},
				},
			},
		},
		MaxTokens:   intPtr(200),
		Temperature: floatPtr(0),
	}, schema)

	if err != nil {
		// Structured outputs might not be available on all models
		if err.Error() == "unsupported" {
			t.Skip("Structured outputs not supported on this model")
		}
		t.Fatalf("GenerateObject failed: %v", err)
	}

	if result.Value == nil {
		t.Fatal("Expected non-nil object value")
	}

	// Validate the structure
	if obj, ok := result.Value.(map[string]interface{}); ok {
		if _, hasName := obj["name"]; !hasName {
			t.Error("Missing 'name' field")
		}
		if _, hasAge := obj["age"]; !hasAge {
			t.Error("Missing 'age' field")
		}
		if _, hasCity := obj["city"]; !hasCity {
			t.Error("Missing 'city' field")
		}
		if _, hasHobbies := obj["hobbies"]; !hasHobbies {
			t.Error("Missing 'hobbies' field")
		}
	} else {
		t.Errorf("Unexpected object type: %T", result.Value)
	}
}

func TestIntegrationStreamObject(t *testing.T) {
	skipIfNoAPIKey(t)

	p := New(
		WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		WithModel("gpt-4o-mini"),
	)

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"summary": map[string]interface{}{
				"type": "string",
			},
			"sentiment": map[string]interface{}{
				"type": "string",
				"enum": []string{"positive", "negative", "neutral"},
			},
		},
		"required": []string{"summary", "sentiment"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := p.StreamObject(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Analyze this text: 'The new product launch was incredibly successful, exceeding all expectations.' Provide a summary and sentiment."},
				},
			},
		},
		MaxTokens:   intPtr(100),
		Temperature: floatPtr(0),
		Stream:      true,
	}, schema)

	if err != nil {
		if err.Error() == "unsupported" {
			t.Skip("Structured streaming not supported on this model")
		}
		t.Fatalf("StreamObject failed: %v", err)
	}
	defer stream.Close()

	// Collect streaming events
	eventCount := 0
	for event := range stream.Events() {
		eventCount++
		if event.Type == core.EventError {
			t.Errorf("Stream error: %v", event.Err)
		}
	}

	// Get final object
	finalObj, err := stream.Final()
	if err != nil {
		t.Fatalf("Failed to get final object: %v", err)
	}

	if finalObj == nil || *finalObj == nil {
		t.Fatal("Expected non-nil final object")
	}

	// Validate structure
	if obj, ok := (*finalObj).(map[string]interface{}); ok {
		if _, hasSummary := obj["summary"]; !hasSummary {
			t.Error("Missing 'summary' field")
		}
		if sentiment, hasSentiment := obj["sentiment"]; hasSentiment {
			if s, ok := sentiment.(string); ok {
				if s != "positive" && s != "negative" && s != "neutral" {
					t.Errorf("Invalid sentiment value: %s", s)
				}
			}
		} else {
			t.Error("Missing 'sentiment' field")
		}
	}

	if eventCount < 3 {
		t.Errorf("Too few streaming events: %d", eventCount)
	}
}

func TestIntegrationErrorHandling(t *testing.T) {
	skipIfNoAPIKey(t)

	tests := []struct {
		name    string
		setup   func() *Provider
		request core.Request
		wantErr bool
		errType string
	}{
		{
			name: "invalid API key",
			setup: func() *Provider {
				return New(
					WithAPIKey("invalid-key"),
				)
			},
			request: core.Request{
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "Hello"},
						},
					},
				},
			},
			wantErr: true,
			errType: "auth",
		},
		{
			name: "invalid model",
			setup: func() *Provider {
				return New(
					WithAPIKey(os.Getenv("OPENAI_API_KEY")),
					WithModel("invalid-model-xyz"),
				)
			},
			request: core.Request{
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "Hello"},
						},
					},
				},
			},
			wantErr: true,
			errType: "invalid_request",
		},
		{
			name: "token limit exceeded",
			setup: func() *Provider {
				return New(
					WithAPIKey(os.Getenv("OPENAI_API_KEY")),
					WithModel("gpt-4o-mini"),
				)
			},
			request: core.Request{
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "Hello"},
						},
					},
				},
				MaxTokens: intPtr(1000000), // Exceeds model limit
			},
			wantErr: true,
			errType: "invalid_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.setup()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_, err := p.GenerateText(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("Error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errType != "" {
				// Check error type
				if aiErr, ok := err.(*core.AIError); ok {
					t.Logf("Error details: Category=%s, Code=%s, Message=%s",
						aiErr.Category, aiErr.Code, aiErr.Error())
				} else {
					t.Errorf("Expected core.AIError, got %T", err)
				}
			}
		})
	}
}