//go:build integration
// +build integration

package gai_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/collinshill/gai"
)

// Weather represents a weather query response
type Weather struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`
	Conditions  string  `json:"conditions"`
	Unit        string  `json:"unit"`
}

// TestActionPattern tests the generic Action[T] pattern
func TestActionPattern(t *testing.T) {
	// Create a client with functional options
	client, err := gai.NewClient(
		gai.WithHTTPTimeout(30*time.Second),
		gai.WithMaxRetries(3),
		gai.WithDefaultProvider("openai"),
		gai.WithDefaultModel("gpt-4o-mini"),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create a typed action for weather queries
	weatherAction := gai.NewAction[Weather]().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithSystem("You are a weather API. Return weather data in JSON format.").
		WithUserMessage("What's the weather in San Francisco?")

	// Execute the action
	ctx := context.Background()
	weather, err := weatherAction.Run(ctx, client)
	if err != nil {
		t.Logf("Weather action failed (expected in test): %v", err)
	} else {
		t.Logf("Weather response: %+v", weather)
	}
}

// TestFluentBuilder tests the fluent builder pattern
func TestFluentBuilder(t *testing.T) {
	// Build a conversation using fluent methods
	parts := gai.NewLLMCallParts().
		WithProvider("anthropic").
		WithModel("claude-3-haiku").
		WithTemperature(0.7).
		WithMaxTokens(2000).
		WithSystem("You are a helpful coding assistant.").
		WithUserMessage("Explain Go interfaces").
		WithAssistantMessage("Go interfaces are...").
		WithUserMessage("Can you provide an example?")

	t.Logf("Built conversation with %d messages", len(parts.Messages))
	t.Logf("Provider: %s, Model: %s", parts.Provider, parts.Model)
}

// TestMessageConstructors tests first-class message constructors
func TestMessageConstructors(t *testing.T) {
	// Use various message constructors
	userMsg := gai.NewUserMessage("Hello, assistant!")
	assistantMsg := gai.NewAssistantMessage("Hello! How can I help you?")
	systemMsg := gai.NewSystemMessage("You are a helpful assistant.")
	
	// Build a message with multiple content types
	complexMsg := gai.NewMessageBuilder("user").
		WithText("Here's an image to analyze:").
		WithImageURL("image/png", "https://example.com/image.png").
		Build()

	t.Logf("Created %d simple messages", 3)
	t.Logf("Complex message has %d content items", len(complexMsg.Contents))
	
	// Test convenience functions
	msgWithImage := gai.NewUserMessageWithImageURL(
		"What's in this image?",
		"image/jpeg",
		"https://example.com/photo.jpg",
	)
	t.Logf("Message with image: role=%s, contents=%d", msgWithImage.Role, len(msgWithImage.Contents))
}

// TestPromptTemplates tests the prompt template functionality
func TestPromptTemplates(t *testing.T) {
	// Create a template
	tmpl, err := gai.NewPromptTemplate(`
You are analyzing code for {{.Language}}.
The file is: {{.Filename}}
Focus on: {{.Focus}}
`)
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	// Use the template with LLMCallParts
	data := map[string]string{
		"Language": "Go",
		"Filename": "main.go",
		"Focus":    "error handling",
	}

	parts := gai.NewLLMCallParts()
	if err := gai.RenderSystemTemplate(parts, tmpl, data); err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}
	parts.WithUserMessage("Please analyze the code")

	systemContent := parts.System.GetTextContent()
	t.Logf("System message from template: %s", systemContent)
}

// TestConversationUtilities tests conversation manipulation utilities
func TestConversationUtilities(t *testing.T) {
	// Build a conversation
	parts := gai.NewLLMCallParts().
		WithUserMessage("First question").
		WithAssistantMessage("First answer").
		WithUserMessage("Second question").
		WithAssistantMessage("Second answer").
		WithUserMessage("Third question")

	// Test finding messages
	lastUser, idx := parts.FindLastMessage("user")
	if lastUser != nil {
		t.Logf("Last user message at index %d: %s", idx, lastUser.GetTextContent())
	}

	// Test filtering
	filtered := parts.FilterMessages(func(m gai.Message) bool {
		return m.Role == "user"
	})
	t.Logf("Filtered to %d user messages", len(filtered.Messages))

	// Test trimming
	original := parts.Clone()
	parts.TrimMessages(3)
	t.Logf("Trimmed from %d to %d messages", len(original.Messages), len(parts.Messages))

	// Test transcript
	transcript := original.Transcript()
	t.Logf("Transcript:\n%s", transcript)
}

// TestErrorHandling tests structured error handling
func TestErrorHandling(t *testing.T) {
	parts := gai.NewLLMCallParts().
		WithProvider("invalid-provider").
		WithModel("invalid-model").
		WithUserMessage("Test")

	// Create an error with context
	err := gai.NewLLMError(
		fmt.Errorf("provider not found"),
		parts.Provider,
		parts.Model,
	).WithContext("attempted_at", time.Now()).
		WithContext("retry_count", 3)

	t.Logf("Structured error: %v", err)
	if llmErr, ok := err.(*gai.LLMError); ok {
		t.Logf("Error context: %+v", llmErr.Context)
	}
}

// TestTracing tests the trace functionality
func TestTracing(t *testing.T) {
	// Create a trace function
	traces := []gai.TraceInfo{}
	traceFunc := func(info gai.TraceInfo) {
		traces = append(traces, info)
		log.Printf("TRACE: Attempt=%d Provider=%s Model=%s Duration=%v",
			info.Attempt, info.Provider, info.Model, info.Duration)
	}

	// Create parts with tracing enabled
	parts := gai.NewLLMCallParts().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithTrace(traceFunc).
		WithUserMessage("Hello")

	t.Logf("Created parts with tracing enabled")
	// In real usage, executing this would trigger the trace function
}

// TestContextWindowManagement tests token management features
func TestContextWindowManagement(t *testing.T) {
	// Create a conversation with many messages
	parts := gai.NewLLMCallParts().
		WithSystem("You are a helpful assistant.")

	// Add many messages
	for i := 0; i < 20; i++ {
		parts.WithUserMessage(fmt.Sprintf("Question %d: Tell me about topic %d", i, i)).
			WithAssistantMessage(fmt.Sprintf("Answer %d: Here's information about topic %d...", i, i))
	}

	// Test token estimation
	tokenizer := gai.NewSimpleTokenizer()
	tokens := parts.EstimateTokens(tokenizer)
	t.Logf("Estimated tokens: %d", tokens)

	// Test pruning to token limit
	original := parts.Clone()
	removed, err := parts.PruneToTokens(1000, tokenizer)
	if err != nil {
		t.Logf("Pruning error: %v", err)
	} else {
		t.Logf("Pruned %d messages to fit in 1000 tokens", removed)
		t.Logf("Messages: %d -> %d", len(original.Messages), len(parts.Messages))
	}

	// Test pruning while keeping recent messages
	parts2 := original.Clone()
	removed2, err := parts2.PruneKeepingRecent(5, 1000, tokenizer)
	if err != nil {
		t.Logf("Pruning error: %v", err)
	} else {
		t.Logf("Pruned %d messages while keeping 5 recent", removed2)
	}

	// Test context window lookup
	window := gai.GetModelContextWindow("gpt-4o")
	t.Logf("GPT-4o context window: %d tokens", window)
}

// TestEndToEndWorkflow tests a complete agent workflow
func TestEndToEndWorkflow(t *testing.T) {
	// Skip if no API keys are available
	client, err := gai.NewClient()
	if err != nil {
		t.Skip("Skipping end-to-end test: no client available")
	}

	// Create a conversation action
	type Analysis struct {
		Summary     string   `json:"summary"`
		KeyPoints   []string `json:"key_points"`
		Sentiment   string   `json:"sentiment"`
		Recommended bool     `json:"recommended"`
	}

	// Build the action with all features
	action := gai.NewAction[Analysis]().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithTemperature(0.3).
		WithMaxTokens(500).
		WithSystem("You are a text analysis expert. Analyze the given text and return structured JSON.").
		WithUserMessage("Please analyze this product review: 'This product exceeded my expectations. Great quality!'")

	// Add tracing
	action.GetParts().WithTrace(func(info gai.TraceInfo) {
		t.Logf("Trace: %+v", info)
	})

	// Execute
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := action.Run(ctx, client)
	if err != nil {
		t.Logf("Analysis failed (expected in test): %v", err)
		return
	}

	t.Logf("Analysis result: %+v", result)
}

// ExampleWeatherAgent demonstrates the agent workflow from the docs
func ExampleWeatherAgent() {
	client, _ := gai.NewClient()
	ctx := context.Background()

	// Create an action for weather queries
	act := gai.NewAction[string]().
		WithProvider("openai").
		WithModel("gpt-4o").
		WithSystem("You can call tools.").
		WithUserMessage("What's the weather in Paris?")

	// First call - might request tool use
	resp1, _ := client.GetCompletion(ctx, act.Parts)
	
	// In a real implementation, check for tool calls
	// and execute them, then continue the conversation
	
	fmt.Println(resp1.Content)
}