package core

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func BenchmarkEventCreation(b *testing.B) {
	b.Run("TextDelta", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Event{
				Type:      EventTextDelta,
				TextDelta: "Hello world",
				Timestamp: time.Now(),
			}
		}
	})
	
	b.Run("ToolCall", func(b *testing.B) {
		input := json.RawMessage(`{"query":"test"}`)
		for i := 0; i < b.N; i++ {
			_ = Event{
				Type:      EventToolCall,
				ToolName:  "search",
				ToolID:    "call-123",
				ToolInput: input,
				Timestamp: time.Now(),
			}
		}
	})
	
	b.Run("Citations", func(b *testing.B) {
		citations := []Citation{
			{URI: "http://example.com", Start: 0, End: 10, Title: "Example"},
		}
		for i := 0; i < b.N; i++ {
			_ = Event{
				Type:      EventCitations,
				Citations: citations,
				Timestamp: time.Now(),
			}
		}
	})
}

func BenchmarkStopConditions(b *testing.B) {
	step := Step{
		Text: "response",
		ToolCalls: []ToolCall{
			{Name: "tool1", Input: json.RawMessage(`{}`)},
		},
	}
	
	b.Run("MaxSteps", func(b *testing.B) {
		cond := MaxSteps(3)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cond.ShouldStop(2, step)
		}
	})
	
	b.Run("NoMoreTools", func(b *testing.B) {
		cond := NoMoreTools()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cond.ShouldStop(1, step)
		}
	})
	
	b.Run("UntilToolSeen", func(b *testing.B) {
		cond := UntilToolSeen("search")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cond.ShouldStop(1, step)
		}
	})
	
	b.Run("CombineConditions", func(b *testing.B) {
		cond := CombineConditions(
			MaxSteps(10),
			NoMoreTools(),
			UntilToolSeen("done"),
		)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cond.ShouldStop(5, step)
		}
	})
}

func BenchmarkErrorCreation(b *testing.B) {
	b.Run("NewAIError", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewAIError(ErrorCategoryRateLimit, "provider", "Rate limit exceeded")
		}
	})
	
	b.Run("WithChaining", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewAIError(ErrorCategoryRateLimit, "provider", "Rate limit exceeded").
				WithCode("rate_limit").
				WithHTTPStatus(429).
				WithRetryAfter(30)
		}
	})
}

func BenchmarkErrorChecks(b *testing.B) {
	err := NewAIError(ErrorCategoryRateLimit, "provider", "Rate limited")
	
	b.Run("IsRateLimited", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = IsRateLimited(err)
		}
	})
	
	b.Run("IsRetryable", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = IsRetryable(err)
		}
	})
	
	b.Run("GetRetryAfter", func(b *testing.B) {
		errWithRetry := err.WithRetryAfter(30)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = GetRetryAfter(errWithRetry)
		}
	})
}

func BenchmarkMessageConstruction(b *testing.B) {
	b.Run("SimpleText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Message{
				Role: User,
				Parts: []Part{
					Text{Text: "Hello world"},
				},
			}
		}
	})
	
	b.Run("Multimodal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Message{
				Role: User,
				Parts: []Part{
					Text{Text: "Describe this image"},
					ImageURL{URL: "http://example.com/img.jpg", Detail: "high"},
					Audio{Source: BlobRef{Kind: BlobURL, URL: "http://example.com/audio.mp3"}},
				},
			}
		}
	})
}

func BenchmarkRequestCreation(b *testing.B) {
	messages := []Message{
		{Role: System, Parts: []Part{Text{Text: "You are helpful"}}},
		{Role: User, Parts: []Part{Text{Text: "Hello"}}},
	}
	
	b.Run("Basic", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Request{
				Model:       "gpt-4",
				Messages:    messages,
				Temperature: 0.7,
				MaxTokens:   1000,
			}
		}
	})
	
	b.Run("WithOptions", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Request{
				Model:       "gpt-4",
				Messages:    messages,
				Temperature: 0.7,
				MaxTokens:   1000,
				ProviderOptions: map[string]any{
					"top_p":    0.9,
					"presence": 0.5,
				},
				Metadata: map[string]any{
					"user_id": "123",
					"session": "abc",
				},
			}
		}
	})
}

// Benchmark parallel execution patterns
func BenchmarkParallelExecution(b *testing.B) {
	ctx := context.Background()
	
	b.Run("ContextCheck", func(b *testing.B) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			select {
			case <-ctx.Done():
			default:
			}
		}
	})
	
	b.Run("ChannelSend", func(b *testing.B) {
		ch := make(chan Event, 100)
		event := Event{Type: EventTextDelta, TextDelta: "test"}
		
		go func() {
			for range ch {
			}
		}()
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			select {
			case ch <- event:
			default:
			}
		}
		close(ch)
	})
}