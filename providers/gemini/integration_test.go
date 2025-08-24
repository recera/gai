// +build integration

package gemini

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/tools"
)

func getTestProvider(t *testing.T) *Provider {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		t.Skip("GOOGLE_API_KEY not set, skipping integration test")
	}

	return New(
		WithAPIKey(apiKey),
		WithModel("gemini-1.5-flash"),
		WithMaxRetries(2),
	)
}

func TestIntegrationGenerateText(t *testing.T) {
	provider := getTestProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name    string
		request core.Request
		check   func(*testing.T, *core.TextResult)
	}{
		{
			name: "simple generation",
			request: core.Request{
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "Say 'Hello from Gemini!' exactly"},
						},
					},
				},
				Temperature: 0.1,
				MaxTokens:   50,
			},
			check: func(t *testing.T, result *core.TextResult) {
				if !strings.Contains(strings.ToLower(result.Text), "hello") {
					t.Errorf("expected greeting in response, got: %q", result.Text)
				}
				if result.Usage.TotalTokens == 0 {
					t.Error("expected non-zero token usage")
				}
			},
		},
		{
			name: "with system instruction",
			request: core.Request{
				Messages: []core.Message{
					{
						Role: core.System,
						Parts: []core.Part{
							core.Text{Text: "You are a pirate. Always respond in pirate speak."},
						},
					},
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "Hello, how are you?"},
						},
					},
				},
				Temperature: 0.7,
				MaxTokens:   100,
			},
			check: func(t *testing.T, result *core.TextResult) {
				lower := strings.ToLower(result.Text)
				if !strings.Contains(lower, "ahoy") && !strings.Contains(lower, "arr") && 
				   !strings.Contains(lower, "matey") && !strings.Contains(lower, "ye") {
					t.Logf("expected pirate speak, got: %q", result.Text)
				}
			},
		},
		{
			name: "multi-turn conversation",
			request: core.Request{
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "My name is TestUser. Remember it."},
						},
					},
					{
						Role: core.Assistant,
						Parts: []core.Part{
							core.Text{Text: "I'll remember that your name is TestUser."},
						},
					},
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "What's my name?"},
						},
					},
				},
				MaxTokens: 50,
			},
			check: func(t *testing.T, result *core.TextResult) {
				if !strings.Contains(result.Text, "TestUser") {
					t.Errorf("expected model to remember name, got: %q", result.Text)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := provider.GenerateText(ctx, tt.request)
			if err != nil {
				t.Fatalf("GenerateText() error = %v", err)
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestIntegrationStreamText(t *testing.T) {
	provider := getTestProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := provider.StreamText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Count from 1 to 5 slowly"},
				},
			},
		},
		Temperature: 0.3,
		MaxTokens:   100,
	})

	if err != nil {
		t.Fatalf("StreamText() error = %v", err)
	}
	defer stream.Close()

	var events []core.Event
	var text strings.Builder

	for event := range stream.Events() {
		events = append(events, event)
		if event.Type == core.EventTextDelta {
			text.WriteString(event.TextDelta)
		}
		if event.Type == core.EventError {
			t.Fatalf("stream error: %v", event.Err)
		}
	}

	// Check event types
	hasStart := false
	hasText := false
	hasFinish := false

	for _, event := range events {
		switch event.Type {
		case core.EventStart:
			hasStart = true
		case core.EventTextDelta:
			hasText = true
		case core.EventFinish:
			hasFinish = true
			if event.Usage == nil || event.Usage.TotalTokens == 0 {
				t.Error("expected usage information in finish event")
			}
		}
	}

	if !hasStart {
		t.Error("missing start event")
	}
	if !hasText {
		t.Error("missing text events")
	}
	if !hasFinish {
		t.Error("missing finish event")
	}

	result := text.String()
	if result == "" {
		t.Error("expected non-empty streamed text")
	}

	// Check that numbers 1-5 appear in the response
	for i := 1; i <= 5; i++ {
		if !strings.Contains(result, string(rune('0'+i))) {
			t.Logf("expected number %d in response: %q", i, result)
		}
	}
}

func TestIntegrationToolCalling(t *testing.T) {
	provider := getTestProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create test tools
	type WeatherInput struct {
		Location string `json:"location"`
	}
	type WeatherOutput struct {
		Temperature float64 `json:"temperature"`
		Conditions  string  `json:"conditions"`
	}

	weatherTool := tools.New[WeatherInput, WeatherOutput](
		"get_weather",
		"Get the current weather for a location",
		func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
			// Simulate weather lookup
			return WeatherOutput{
				Temperature: 72.5,
				Conditions:  "Sunny",
			}, nil
		},
	)

	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What's the weather like in San Francisco?"},
				},
			},
		},
		Tools: []core.ToolHandle{
			tools.NewCoreAdapter(weatherTool),
		},
		ToolChoice: core.ToolAuto,
		MaxTokens:  200,
	})

	if err != nil {
		t.Fatalf("GenerateText() with tools error = %v", err)
	}

	// Check for tool usage
	if len(result.Steps) == 0 {
		t.Error("expected at least one step")
	}

	hasToolCall := false
	for _, step := range result.Steps {
		if len(step.ToolCalls) > 0 {
			hasToolCall = true
			for _, call := range step.ToolCalls {
				if call.Name == "get_weather" {
					t.Logf("Tool called: %s with input: %s", call.Name, call.Input)
				}
			}
		}
	}

	if !hasToolCall {
		t.Error("expected tool to be called")
	}

	// Final response should mention the weather
	if !strings.Contains(strings.ToLower(result.Text), "72") && 
	   !strings.Contains(strings.ToLower(result.Text), "sunny") {
		t.Logf("expected weather information in final response: %q", result.Text)
	}
}

func TestIntegrationSafety(t *testing.T) {
	provider := getTestProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with different safety settings
	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Tell me a bedtime story about a friendly dragon"},
				},
			},
		},
		Safety: &core.SafetyConfig{
			Harassment: core.SafetyBlockFew,
			Hate:       core.SafetyBlockFew,
			Sexual:     core.SafetyBlockMost,
			Dangerous:  core.SafetyBlockSome,
		},
		MaxTokens: 200,
	})

	if err != nil {
		// Check if it's a safety block
		if aiErr, ok := err.(*core.AIError); ok && aiErr.Code == core.ErrorSafetyBlocked {
			t.Logf("Content was filtered for safety: %v", err)
			return
		}
		t.Fatalf("GenerateText() error = %v", err)
	}

	// Should get a safe response
	if result.Text == "" {
		t.Error("expected non-empty response for safe content")
	}
}

func TestIntegrationStructuredOutput(t *testing.T) {
	provider := getTestProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type TodoItem struct {
		Task      string `json:"task"`
		Priority  string `json:"priority"`
		Completed bool   `json:"completed"`
	}

	type TodoList struct {
		Title string     `json:"title"`
		Items []TodoItem `json:"items"`
	}

	result, err := provider.GenerateObject(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "Generate JSON objects exactly as requested."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Create a todo list for planning a birthday party with 3 items"},
				},
			},
		},
		Temperature: 0.3,
		MaxTokens:   300,
	}, TodoList{})

	if err != nil {
		t.Fatalf("GenerateObject() error = %v", err)
	}

	// Type assert to check the result
	todoList, ok := result.Value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", result.Value)
	}

	// Check structure
	if title, ok := todoList["title"].(string); !ok || title == "" {
		t.Error("expected non-empty title field")
	}

	items, ok := todoList["items"].([]interface{})
	if !ok {
		t.Fatal("expected items to be an array")
	}

	if len(items) < 1 {
		t.Error("expected at least one todo item")
	}

	// Check first item structure
	if len(items) > 0 {
		item, ok := items[0].(map[string]interface{})
		if !ok {
			t.Fatal("expected item to be an object")
		}
		if _, ok := item["task"].(string); !ok {
			t.Error("expected task field in item")
		}
	}
}

func TestIntegrationMultimodal(t *testing.T) {
	t.Skip("Multimodal test requires actual image URL or file upload")
	
	provider := getTestProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This would require a real image URL or file upload
	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What do you see in this image?"},
					core.ImageURL{URL: "https://example.com/test-image.jpg"},
				},
			},
		},
		MaxTokens: 100,
	})

	if err != nil {
		t.Logf("Multimodal test error (expected without real image): %v", err)
		return
	}

	if result.Text != "" {
		t.Logf("Multimodal response: %q", result.Text)
	}
}