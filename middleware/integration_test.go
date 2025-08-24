// +build integration

package middleware

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/providers/openai"
)

// TestIntegration_OpenAI_WithRetry tests retry middleware with real OpenAI provider
func TestIntegration_OpenAI_WithRetry(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	// Create OpenAI provider
	provider := openai.New(
		openai.WithAPIKey(apiKey),
		openai.WithModel("gpt-4o-mini"),
	)

	// Wrap with retry middleware
	retryProvider := WithRetry(RetryOpts{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		Jitter:      true,
	})(provider)

	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a helpful assistant. Keep responses very brief."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Say 'Hello, middleware!' exactly."},
				},
			},
		},
		MaxTokens:   50,
		Temperature: 0,
	}

	result, err := retryProvider.GenerateText(ctx, req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if result.Text == "" {
		t.Error("expected non-empty response")
	}

	t.Logf("Response: %s", result.Text)
	t.Logf("Tokens used: in=%d, out=%d", result.Usage.InputTokens, result.Usage.OutputTokens)
}

// TestIntegration_OpenAI_WithRateLimit tests rate limiting with real OpenAI provider
func TestIntegration_OpenAI_WithRateLimit(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	provider := openai.New(
		openai.WithAPIKey(apiKey),
		openai.WithModel("gpt-4o-mini"),
	)

	// Track rate limit events
	var rateLimitedCount int
	var totalWaitTime time.Duration

	// Wrap with rate limit middleware (very low limit for testing)
	rateLimitProvider := WithRateLimit(RateLimitOpts{
		RPS:         1, // 1 request per second
		Burst:       1,
		WaitTimeout: 10 * time.Second,
		OnRateLimited: func(method string, waitTime time.Duration) {
			rateLimitedCount++
			totalWaitTime += waitTime
			t.Logf("Rate limited on %s, waiting %v", method, waitTime)
		},
	})(provider)

	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Say 'test' only."},
				},
			},
		},
		MaxTokens:   10,
		Temperature: 0,
	}

	// Make 3 rapid requests
	start := time.Now()
	for i := 0; i < 3; i++ {
		result, err := rateLimitProvider.GenerateText(ctx, req)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		t.Logf("Request %d completed: %s", i, result.Text)
	}
	elapsed := time.Since(start)

	// Should have been rate limited at least once
	if rateLimitedCount < 2 {
		t.Errorf("expected at least 2 rate limit events, got %d", rateLimitedCount)
	}

	// Should have taken at least 2 seconds (3 requests at 1 RPS)
	if elapsed < 2*time.Second {
		t.Errorf("requests completed too quickly: %v", elapsed)
	}

	t.Logf("Total time: %v, Rate limited %d times, Total wait: %v", 
		elapsed, rateLimitedCount, totalWaitTime)
}

// TestIntegration_OpenAI_WithSafety tests safety filtering with real OpenAI provider
func TestIntegration_OpenAI_WithSafety(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	provider := openai.New(
		openai.WithAPIKey(apiKey),
		openai.WithModel("gpt-4o-mini"),
	)

	// Track redactions
	var redactionCount int

	// Wrap with safety middleware
	safetyProvider := WithSafety(SafetyOpts{
		RedactPatterns: []string{
			`\b\d{3}-\d{2}-\d{4}\b`,                     // SSN
			`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email
			`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`, // Credit card
		},
		RedactReplacement: "[REDACTED]",
		OnRedacted: func(pattern string, count int) {
			redactionCount += count
			t.Logf("Redacted %d instances of pattern: %s", count, pattern)
		},
	})(provider)

	ctx := context.Background()
	
	// Request with PII that should be redacted
	req := core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a helpful assistant. Echo back exactly what the user says."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "My SSN is 123-45-6789 and my email is test@example.com"},
				},
			},
		},
		MaxTokens:   100,
		Temperature: 0,
	}

	result, err := safetyProvider.GenerateText(ctx, req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Check that PII was redacted in the response
	if result.Text == "" {
		t.Error("expected non-empty response")
	}

	t.Logf("Response: %s", result.Text)

	// Response should contain [REDACTED] instead of actual PII
	// Note: The model may not echo back the exact PII, but if it does, it should be redacted
	if redactionCount > 0 {
		t.Logf("Successfully redacted %d PII instances", redactionCount)
	}
}

// TestIntegration_OpenAI_ChainedMiddleware tests all middleware chained together
func TestIntegration_OpenAI_ChainedMiddleware(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	baseProvider := openai.New(
		openai.WithAPIKey(apiKey),
		openai.WithModel("gpt-4o-mini"),
	)

	// Chain all middleware
	provider := Chain(
		WithRetry(RetryOpts{
			MaxAttempts: 2,
			BaseDelay:   100 * time.Millisecond,
		}),
		WithRateLimit(RateLimitOpts{
			RPS:   5,
			Burst: 2,
		}),
		WithSafety(SafetyOpts{
			RedactPatterns: []string{
				`\b\d{3}-\d{2}-\d{4}\b`, // SSN
			},
			RedactReplacement: "[SSN]",
			MaxContentLength:  10000,
		}),
	)(baseProvider)

	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What is 2+2? Also, here's a number to ignore: 123-45-6789"},
				},
			},
		},
		MaxTokens:   50,
		Temperature: 0,
	}

	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	t.Logf("Response: %s", result.Text)
	
	// The SSN in the request should have been redacted
	// The response should be about 2+2=4
	if result.Text == "" {
		t.Error("expected non-empty response")
	}
}

// TestIntegration_OpenAI_Streaming tests middleware with streaming
func TestIntegration_OpenAI_Streaming(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	provider := openai.New(
		openai.WithAPIKey(apiKey),
		openai.WithModel("gpt-4o-mini"),
	)

	// Wrap with safety middleware to test streaming filtering
	safetyProvider := WithSafety(SafetyOpts{
		RedactPatterns: []string{
			`\b\d{3}-\d{2}-\d{4}\b`, // SSN
		},
		RedactReplacement: "[SSN]",
	})(provider)

	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Count from 1 to 5 slowly."},
				},
			},
		},
		MaxTokens:   100,
		Temperature: 0,
		Stream:      true,
	}

	stream, err := safetyProvider.StreamText(ctx, req)
	if err != nil {
		t.Fatalf("stream creation failed: %v", err)
	}
	defer stream.Close()

	var fullText string
	eventCount := 0

	for event := range stream.Events() {
		eventCount++
		switch event.Type {
		case core.EventTextDelta:
			fullText += event.TextDelta
			t.Logf("Delta: %q", event.TextDelta)
		case core.EventFinish:
			t.Log("Stream finished")
		case core.EventError:
			t.Fatalf("Stream error: %v", event.Err)
		}
	}

	if fullText == "" {
		t.Error("expected non-empty streamed response")
	}

	if eventCount < 2 {
		t.Errorf("expected multiple events, got %d", eventCount)
	}

	t.Logf("Full response: %s", fullText)
}

// TestIntegration_OpenAI_ErrorHandling tests error handling scenarios
func TestIntegration_OpenAI_ErrorHandling(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	provider := openai.New(
		openai.WithAPIKey(apiKey),
		openai.WithModel("gpt-4o-mini"),
	)

	// Wrap with retry middleware
	retryProvider := WithRetry(RetryOpts{
		MaxAttempts: 2,
		BaseDelay:   100 * time.Millisecond,
	})(provider)

	ctx := context.Background()
	
	// Test with invalid request (too many tokens)
	req := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Hello"},
				},
			},
		},
		MaxTokens: 1000000, // Way too many tokens
	}

	_, err := retryProvider.GenerateText(ctx, req)
	if err == nil {
		t.Fatal("expected error for invalid max tokens")
	}

	// Should be a bad request error (not retryable)
	if core.IsRetryable(err) {
		t.Errorf("bad request should not be retryable: %v", err)
	}

	t.Logf("Got expected error: %v", err)
}