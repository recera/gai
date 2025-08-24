package middleware_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/middleware"
	"github.com/recera/gai/providers/openai"
)

// Example_basic demonstrates basic middleware usage
func Example_basic() {
	// Create a provider
	provider := openai.New(
		openai.WithAPIKey("your-api-key"),
		openai.WithModel("gpt-4o-mini"),
	)

	// Add retry middleware for resilience
	wrappedProvider := middleware.WithRetry(middleware.RetryOpts{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		Jitter:      true,
	})(provider)

	// Use the wrapped provider
	ctx := context.Background()
	result, err := wrappedProvider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What is 2+2?"},
				},
			},
		},
		MaxTokens: 50,
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Text)
}

// Example_chain demonstrates chaining multiple middleware
func Example_chain() {
	provider := openai.New(
		openai.WithAPIKey("your-api-key"),
		openai.WithModel("gpt-4o-mini"),
	)

	// Chain multiple middleware together
	// Order: retry (outermost) -> rate limit -> safety -> provider
	wrapped := middleware.Chain(
		middleware.WithRetry(middleware.RetryOpts{
			MaxAttempts: 3,
			BaseDelay:   200 * time.Millisecond,
		}),
		middleware.WithRateLimit(middleware.RateLimitOpts{
			RPS:   10,
			Burst: 20,
		}),
		middleware.WithSafety(middleware.SafetyOpts{
			RedactPatterns: []string{
				`\b\d{3}-\d{2}-\d{4}\b`, // SSN
			},
			RedactReplacement: "[REDACTED]",
		}),
	)(provider)

	ctx := context.Background()
	result, err := wrapped.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "My SSN is 123-45-6789, please remember it."},
				},
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	// The SSN will be redacted in both request and response
	fmt.Println(result.Text)
}

// Example_rateLimitWithCallbacks demonstrates rate limiting with observability
func Example_rateLimitWithCallbacks() {
	provider := openai.New(
		openai.WithAPIKey("your-api-key"),
		openai.WithModel("gpt-4o-mini"),
	)

	// Add rate limiting with callback for monitoring
	wrappedProvider := middleware.WithRateLimit(middleware.RateLimitOpts{
		RPS:         2, // Low limit for demonstration
		Burst:       2,
		WaitTimeout: 10 * time.Second,
		OnRateLimited: func(method string, waitTime time.Duration) {
			fmt.Printf("Rate limited on %s, waiting %v\n", method, waitTime)
		},
	})(provider)

	ctx := context.Background()
	
	// Make multiple rapid requests
	for i := 0; i < 5; i++ {
		fmt.Printf("Request %d...\n", i+1)
		
		_, err := wrappedProvider.GenerateText(ctx, core.Request{
			Messages: []core.Message{
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: fmt.Sprintf("Say 'Response %d'", i+1)},
					},
				},
			},
			MaxTokens: 10,
		})
		
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

// Example_safetyFiltering demonstrates content safety filtering
func Example_safetyFiltering() {
	provider := openai.New(
		openai.WithAPIKey("your-api-key"),
		openai.WithModel("gpt-4o-mini"),
	)

	// Configure safety filtering
	wrappedProvider := middleware.WithSafety(middleware.SafetyOpts{
		// Redact PII patterns
		RedactPatterns: []string{
			`\b\d{3}-\d{2}-\d{4}\b`,                          // SSN
			`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email
			`\b(?:\+?1[-.]?)?\(?[0-9]{3}\)?[-.]?[0-9]{3}[-.]?[0-9]{4}\b`, // Phone
		},
		RedactReplacement: "[PII]",
		
		// Block certain words
		BlockWords: []string{"confidential", "secret"},
		
		// Limit content length
		MaxContentLength: 1000,
		
		// Callbacks for monitoring
		OnRedacted: func(pattern string, count int) {
			fmt.Printf("Redacted %d instances matching pattern\n", count)
		},
		OnBlocked: func(reason, content string) {
			fmt.Printf("Blocked content: %s\n", reason)
		},
	})(provider)

	ctx := context.Background()

	// This request will have PII redacted
	result, err := wrappedProvider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Contact me at john@example.com or 555-1234"},
				},
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response:", result.Text)
}

// Example_customRetryLogic demonstrates custom retry conditions
func Example_customRetryLogic() {
	provider := openai.New(
		openai.WithAPIKey("your-api-key"),
		openai.WithModel("gpt-4o-mini"),
	)

	// Custom retry logic based on error type
	wrappedProvider := middleware.WithRetry(middleware.RetryOpts{
		MaxAttempts: 5,
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		Jitter:      true,
		RetryIf: func(err error) bool {
			// Always retry rate limits
			if core.IsRateLimited(err) {
				fmt.Println("Retrying due to rate limit")
				return true
			}
			
			// Retry transient errors up to a point
			if core.IsTransient(err) {
				fmt.Println("Retrying transient error")
				return true
			}
			
			// Don't retry other errors
			return false
		},
	})(provider)

	ctx := context.Background()
	_, err := wrappedProvider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Hello"},
				},
			},
		},
	})

	if err != nil {
		fmt.Printf("Final error: %v\n", err)
	}
}

// Example_streaming demonstrates middleware with streaming responses
func Example_streaming() {
	provider := openai.New(
		openai.WithAPIKey("your-api-key"),
		openai.WithModel("gpt-4o-mini"),
	)

	// Add safety filtering for streams
	wrappedProvider := middleware.WithSafety(middleware.SafetyOpts{
		RedactPatterns: []string{
			`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`, // Credit card
		},
		RedactReplacement: "[CARD]",
		StopOnSafetyEvent: true,
	})(provider)

	ctx := context.Background()
	stream, err := wrappedProvider.StreamText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Generate a story"},
				},
			},
		},
		Stream: true,
	})

	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	// Process stream events
	for event := range stream.Events() {
		switch event.Type {
		case core.EventTextDelta:
			// Text is already filtered for safety
			fmt.Print(event.TextDelta)
		case core.EventSafety:
			fmt.Printf("\n[Safety Event: %s]\n", event.Safety.Category)
		case core.EventFinish:
			fmt.Println("\n[Stream Complete]")
		case core.EventError:
			fmt.Printf("\n[Error: %v]\n", event.Err)
		}
	}
}

// Example_perMethodRateLimits demonstrates different rate limits per method
func Example_perMethodRateLimits() {
	provider := openai.New(
		openai.WithAPIKey("your-api-key"),
		openai.WithModel("gpt-4o-mini"),
	)

	// Configure different limits for different operations
	wrappedProvider := middleware.WithRateLimit(middleware.RateLimitOpts{
		RPS:   5,  // Default limit
		Burst: 10,
		PerMethod: map[string]*middleware.RateLimitConfig{
			"GenerateText": {
				RPS:   10, // Higher limit for regular generation
				Burst: 20,
			},
			"StreamText": {
				RPS:   2, // Lower limit for streaming (more expensive)
				Burst: 4,
			},
			"GenerateObject": {
				RPS:   15, // Higher limit for structured outputs
				Burst: 30,
			},
		},
	})(provider)

	ctx := context.Background()

	// These will use different rate limits
	_, _ = wrappedProvider.GenerateText(ctx, core.Request{
		Messages: []core.Message{{Role: core.User, Parts: []core.Part{core.Text{Text: "Hi"}}}},
	})

	_, _ = wrappedProvider.StreamText(ctx, core.Request{
		Messages: []core.Message{{Role: core.User, Parts: []core.Part{core.Text{Text: "Hi"}}}},
		Stream:   true,
	})

	fmt.Println("Different methods use different rate limits")
}

// Example_production demonstrates a production-ready configuration
func Example_production() {
	// Create base provider
	provider := openai.New(
		openai.WithAPIKey("your-api-key"),
		openai.WithModel("gpt-4o"),
		openai.WithOrganization("org-id"),
		openai.WithMaxRetries(0), // Disable provider's built-in retry
	)

	// Apply production middleware stack
	wrappedProvider := middleware.Chain(
		// Retry with exponential backoff and jitter
		middleware.WithRetry(middleware.RetryOpts{
			MaxAttempts: 3,
			BaseDelay:   500 * time.Millisecond,
			MaxDelay:    30 * time.Second,
			Multiplier:  2.0,
			Jitter:      true,
		}),
		
		// Rate limiting based on API tier
		middleware.WithRateLimit(middleware.RateLimitOpts{
			RPS:         100, // Adjust based on your tier
			Burst:       200,
			WaitTimeout: 60 * time.Second,
			OnRateLimited: func(method string, wait time.Duration) {
				// Log to monitoring system
				log.Printf("RATE_LIMITED method=%s wait=%v", method, wait)
			},
		}),
		
		// Safety filtering for PII protection
		middleware.WithSafety(middleware.SafetyOpts{
			RedactPatterns: []string{
				`\b\d{3}-\d{2}-\d{4}\b`,                          // SSN
				`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email
				`\b(?:\+?1[-.]?)?\(?[0-9]{3}\)?[-.]?[0-9]{3}[-.]?[0-9]{4}\b`, // Phone
				`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14})\b`, // Credit cards (partial)
			},
			RedactReplacement: "[PII_REMOVED]",
			MaxContentLength:  100000, // 100KB limit
			OnRedacted: func(pattern string, count int) {
				// Log to monitoring
				log.Printf("PII_REDACTED pattern=%s count=%d", pattern, count)
			},
			OnBlocked: func(reason, content string) {
				// Alert on blocked content
				log.Printf("CONTENT_BLOCKED reason=%s", reason)
			},
		}),
	)(provider)

	// Use in production
	ctx := context.Background()
	result, err := wrappedProvider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a helpful assistant."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "How can I improve my application's performance?"},
				},
			},
		},
		MaxTokens:   500,
		Temperature: 0.7,
	})

	if err != nil {
		// Log error with context
		log.Printf("ERROR generating text: %v", err)
		
		// Check error type for appropriate handling
		switch {
		case core.IsRateLimited(err):
			// Handle rate limiting
			if retryAfter, ok := core.GetRetryAfter(err); ok {
				fmt.Printf("Rate limited, retry after %d seconds\n", retryAfter)
			}
		case core.IsAuth(err):
			// Handle auth errors
			fmt.Println("Authentication failed, check API key")
		case core.IsTransient(err):
			// Already retried, log for investigation
			fmt.Println("Transient error persisted after retries")
		default:
			// Other errors
			fmt.Printf("Unexpected error: %v\n", err)
		}
		return
	}

	fmt.Println("Response:", result.Text)
	fmt.Printf("Tokens used: %d\n", result.Usage.TotalTokens)
}