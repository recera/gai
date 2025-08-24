package anthropic

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/recera/gai/core"
	"github.com/recera/gai/tools"
)

// skipIfNoAPIForBenchmark skips the benchmark if no API key is provided
func skipIfNoAPIForBenchmark(b *testing.B) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		b.Skip("ANTHROPIC_API_KEY not set, skipping benchmark")
	}
}

// newBenchmarkProvider creates a provider optimized for benchmarking
func newBenchmarkProvider(b *testing.B) *Provider {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		b.Fatal("ANTHROPIC_API_KEY environment variable is required for benchmarks")
	}
	
	return New(
		WithAPIKey(apiKey),
		WithModel("claude-3-haiku-20240307"), // Fast, cheap model for benchmarks
		WithMaxRetries(1),                    // Reduce retries for consistent timing
	)
}

func BenchmarkGenerateTextSimple(b *testing.B) {
	skipIfNoAPIForBenchmark(b)
	
	p := newBenchmarkProvider(b)
	ctx := context.Background()
	
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Say 'Hello' in exactly one word."}},
			},
		},
		MaxTokens: 10,
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := p.GenerateText(ctx, req)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkGenerateTextMedium(b *testing.B) {
	skipIfNoAPIForBenchmark(b)
	
	p := newBenchmarkProvider(b)
	ctx := context.Background()
	
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Write a short paragraph about the benefits of renewable energy."}},
			},
		},
		MaxTokens: 200,
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := p.GenerateText(ctx, req)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkGenerateTextLarge(b *testing.B) {
	skipIfNoAPIForBenchmark(b)
	
	p := newBenchmarkProvider(b)
	ctx := context.Background()
	
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Write a comprehensive essay about the impact of artificial intelligence on modern society, covering benefits, challenges, and future implications."}},
			},
		},
		MaxTokens: 1000,
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := p.GenerateText(ctx, req)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkStreamText(b *testing.B) {
	skipIfNoAPIForBenchmark(b)
	
	p := newBenchmarkProvider(b)
	ctx := context.Background()
	
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Count from 1 to 10."}},
			},
		},
		MaxTokens: 50,
		Stream:    true,
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		stream, err := p.StreamText(ctx, req)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		
		// Consume all events
		for range stream.Events() {
			// Just consume, don't process
		}
		
		stream.Close()
	}
}

func BenchmarkGenerateObject(b *testing.B) {
	skipIfNoAPIForBenchmark(b)
	
	p := newBenchmarkProvider(b)
	ctx := context.Background()
	
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
			"age": map[string]interface{}{
				"type": "integer",
			},
			"occupation": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"name", "age", "occupation"},
	}
	
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Generate information for a fictional person."}},
			},
		},
		MaxTokens: 100,
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := p.GenerateObject(ctx, req, schema)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkWithTools(b *testing.B) {
	skipIfNoAPIForBenchmark(b)
	
	p := newBenchmarkProvider(b)
	ctx := context.Background()
	
	// Create a simple math tool
	mathTool := tools.New("math", "Performs basic math operations",
		func(ctx context.Context, input struct {
			Expression string `json:"expression"`
		}, meta tools.Meta) (map[string]interface{}, error) {
			// Simple mock calculation
			return map[string]interface{}{
				"result": 42,
				"expression": input.Expression,
			}, nil
		})
	
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Calculate 20 + 22 using the math tool."}},
			},
		},
		Tools:     tools.ToCoreHandles([]tools.Handle{mathTool}),
		MaxTokens: 150,
		StopWhen:  core.MaxSteps(2),
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := p.GenerateText(ctx, req)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkConcurrentRequests(b *testing.B) {
	skipIfNoAPIForBenchmark(b)
	
	p := newBenchmarkProvider(b)
	ctx := context.Background()
	
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Say 'ok'."}},
			},
		},
		MaxTokens: 10,
	}

	b.ResetTimer()
	
	// Test different concurrency levels
	concurrencyLevels := []int{1, 2, 4, 8}
	
	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrent-%d", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, err := p.GenerateText(ctx, req)
					if err != nil {
						b.Fatalf("unexpected error: %v", err)
					}
				}
			})
		})
	}
}

func BenchmarkProviderCreation(b *testing.B) {
	apiKey := "test-key"
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = New(
			WithAPIKey(apiKey),
			WithModel("claude-3-haiku-20240307"),
		)
	}
}

func BenchmarkRequestConversion(b *testing.B) {
	p := New(WithAPIKey("test-key"))
	
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.System,
				Parts: []core.Part{core.Text{Text: "You are a helpful assistant."}},
			},
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Hello, how are you?"}},
			},
			{
				Role:  core.Assistant,
				Parts: []core.Part{core.Text{Text: "I'm doing well, thank you!"}},
			},
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "What can you help me with?"}},
			},
		},
		Temperature: 0.7,
		MaxTokens:   200,
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := p.convertRequest(req)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkMessageConversion(b *testing.B) {
	p := New()
	
	messages := []core.Message{
		{
			Role:  core.System,
			Parts: []core.Part{core.Text{Text: "You are a helpful assistant that provides accurate information."}},
		},
		{
			Role:  core.User,
			Parts: []core.Part{core.Text{Text: "What is artificial intelligence?"}},
		},
		{
			Role:  core.Assistant,
			Parts: []core.Part{core.Text{Text: "Artificial Intelligence (AI) refers to computer systems that can perform tasks typically requiring human intelligence."}},
		},
		{
			Role:  core.User,
			Parts: []core.Part{core.Text{Text: "Can you give me some examples?"}},
		},
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, _, err := p.convertMessages(messages)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkToolsConversion(b *testing.B) {
	p := New()
	
	// Create multiple tools
	toolHandles := []tools.Handle{
		tools.New("calculator", "Performs calculations",
			func(ctx context.Context, input struct{
				Expression string `json:"expression"`
			}, meta tools.Meta) (float64, error) {
				return 42.0, nil
			}),
		tools.New("weather", "Gets weather information",
			func(ctx context.Context, input struct{
				Location string `json:"location"`
			}, meta tools.Meta) (map[string]interface{}, error) {
				return map[string]interface{}{
					"temperature": 22,
					"condition": "sunny",
				}, nil
			}),
		tools.New("translator", "Translates text",
			func(ctx context.Context, input struct{
				Text   string `json:"text"`
				Target string `json:"target_language"`
			}, meta tools.Meta) (string, error) {
				return "translated text", nil
			}),
	}

	coreTools := tools.ToCoreHandles(toolHandles)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = p.convertTools(coreTools)
	}
}

func BenchmarkMemoryUsage(b *testing.B) {
	var providers []*Provider
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		p := New(
			WithAPIKey("test-key"),
			WithModel("claude-3-haiku-20240307"),
		)
		providers = append(providers, p)
	}
	
	// Keep reference to prevent GC during benchmark
	_ = providers
}

// BenchmarkLargeContext tests performance with large context windows
func BenchmarkLargeContext(b *testing.B) {
	skipIfNoAPIForBenchmark(b)
	
	p := newBenchmarkProvider(b)
	ctx := context.Background()
	
	// Create a large conversation context
	messages := []core.Message{
		{
			Role:  core.System,
			Parts: []core.Part{core.Text{Text: "You are a helpful assistant that maintains context throughout long conversations."}},
		},
	}
	
	// Add many back-and-forth messages
	for i := 0; i < 20; i++ {
		messages = append(messages, 
			core.Message{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: fmt.Sprintf("This is user message number %d. Please acknowledge and remember this number.", i+1)}},
			},
			core.Message{
				Role:  core.Assistant,
				Parts: []core.Part{core.Text{Text: fmt.Sprintf("I acknowledge user message number %d. I will remember this number.", i+1)}},
			},
		)
	}
	
	// Final question that requires context
	messages = append(messages, core.Message{
		Role:  core.User,
		Parts: []core.Part{core.Text{Text: "How many messages have I sent you?"}},
	})
	
	req := core.Request{
		Messages:  messages,
		MaxTokens: 50,
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := p.GenerateText(ctx, req)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkRetryMechanism tests the retry mechanism under simulated failures
func BenchmarkRetryMechanism(b *testing.B) {
	// This benchmark would need a mock server to simulate failures
	// For now, just benchmark the retry logic decision making
	p := New(WithAPIKey("test-key"))
	
	statusCodes := []int{200, 429, 500, 502, 503, 504}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		for _, code := range statusCodes {
			_ = p.shouldRetry(code)
		}
	}
}