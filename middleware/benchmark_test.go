package middleware

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

// Benchmark baseline provider without middleware
func BenchmarkBaseline(b *testing.B) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "response"}, nil
		},
	}
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mock.GenerateText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark retry middleware overhead (no retries needed)
func BenchmarkRetryMiddleware_NoRetries(b *testing.B) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "response"}, nil
		},
	}
	
	provider := WithRetry(RetryOpts{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
	})(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GenerateText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark retry middleware with one retry
func BenchmarkRetryMiddleware_OneRetry(b *testing.B) {
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempt := 0
		mock := &mockProvider{
			generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
				attempt++
				if attempt == 1 {
					return nil, core.NewAIError(core.ErrorCategoryTransient, "test", "transient")
				}
				return &core.TextResult{Text: "response"}, nil
			},
		}
		
		provider := WithRetry(RetryOpts{
			MaxAttempts: 3,
			BaseDelay:   1 * time.Microsecond, // Very short for benchmark
			Jitter:      false,
		})(mock)
		
		_, err := provider.GenerateText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark rate limit middleware without blocking
func BenchmarkRateLimitMiddleware_NoBlocking(b *testing.B) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "response"}, nil
		},
	}
	
	provider := WithRateLimit(RateLimitOpts{
		RPS:   10000, // Very high limit to avoid blocking
		Burst: 10000,
	})(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GenerateText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark rate limit middleware with blocking
func BenchmarkRateLimitMiddleware_WithBlocking(b *testing.B) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "response"}, nil
		},
	}
	
	provider := WithRateLimit(RateLimitOpts{
		RPS:   100, // Limit that will cause some blocking
		Burst: 10,
	})(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GenerateText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark safety middleware with no filtering
func BenchmarkSafetyMiddleware_NoFiltering(b *testing.B) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "clean response"}, nil
		},
	}
	
	provider := WithSafety(SafetyOpts{
		// No patterns or words to check
	})(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "clean input"}}},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GenerateText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark safety middleware with redaction
func BenchmarkSafetyMiddleware_WithRedaction(b *testing.B) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "Response with SSN 123-45-6789 and email test@example.com"}, nil
		},
	}
	
	provider := WithSafety(SafetyOpts{
		RedactPatterns: []string{
			`\b\d{3}-\d{2}-\d{4}\b`,                     // SSN
			`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email
		},
		RedactReplacement: "[REDACTED]",
	})(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Input with 555-55-5555 and user@domain.org"}}},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GenerateText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark safety middleware with block words
func BenchmarkSafetyMiddleware_BlockWords(b *testing.B) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "clean response"}, nil
		},
	}
	
	provider := WithSafety(SafetyOpts{
		BlockWords: []string{
			"forbidden", "blocked", "inappropriate", "offensive",
			"dangerous", "harmful", "illegal", "prohibited",
		},
	})(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: strings.Repeat("this is a clean message ", 10)}}},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GenerateText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark chain of all middleware
func BenchmarkChain_AllMiddleware(b *testing.B) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "Response with 123-45-6789"}, nil
		},
	}
	
	provider := Chain(
		WithRetry(RetryOpts{
			MaxAttempts: 3,
			BaseDelay:   10 * time.Millisecond,
		}),
		WithRateLimit(RateLimitOpts{
			RPS:   10000,
			Burst: 10000,
		}),
		WithSafety(SafetyOpts{
			RedactPatterns:    []string{`\d{3}-\d{2}-\d{4}`},
			RedactReplacement: "[SSN]",
		}),
	)(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GenerateText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark parallel requests with rate limiting
func BenchmarkRateLimitMiddleware_Parallel(b *testing.B) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "response"}, nil
		},
	}
	
	provider := WithRateLimit(RateLimitOpts{
		RPS:   1000,
		Burst: 100,
	})(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}},
		},
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := provider.GenerateText(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Benchmark streaming with safety middleware
func BenchmarkSafetyMiddleware_Streaming(b *testing.B) {
	mock := &mockProvider{
		streamTextFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			events := make(chan core.Event, 10)
			go func() {
				events <- core.Event{Type: core.EventStart}
				for i := 0; i < 10; i++ {
					events <- core.Event{Type: core.EventTextDelta, TextDelta: "chunk with 123-45-6789 "}
				}
				events <- core.Event{Type: core.EventFinish}
				close(events)
			}()
			return &mockTextStream{events: events}, nil
		},
	}
	
	provider := WithSafety(SafetyOpts{
		RedactPatterns:    []string{`\d{3}-\d{2}-\d{4}`},
		RedactReplacement: "[SSN]",
	})(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := provider.StreamText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
		
		// Consume the stream
		for range stream.Events() {
			// Just drain the events
		}
		
		stream.Close()
	}
}

// Benchmark memory allocations in middleware chain
func BenchmarkChain_Allocations(b *testing.B) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "response"}, nil
		},
	}
	
	provider := Chain(
		WithRetry(DefaultRetryOpts()),
		WithRateLimit(DefaultRateLimitOpts()),
		WithSafety(DefaultSafetyOpts()),
	)(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}},
		},
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GenerateText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark context cancellation handling
func BenchmarkRetryMiddleware_ContextCancellation(b *testing.B) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &core.TextResult{Text: "response"}, nil
			}
		},
	}
	
	provider := WithRetry(RetryOpts{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
	})(mock)
	
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		_, err := provider.GenerateText(ctx, req)
		cancel()
		if err != nil && err != context.Canceled {
			b.Fatal(err)
		}
	}
}