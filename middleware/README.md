# Middleware Package

The middleware package provides composable middleware for AI providers in the GAI framework. It includes retry logic, rate limiting, and safety filters that can be applied to any provider implementation.

## Features

- **Retry Middleware**: Automatic retry with exponential backoff for transient failures
- **Rate Limiting**: Token bucket algorithm for request throttling
- **Safety Filtering**: Content redaction and blocking for PII and sensitive data
- **Composable Chain**: Combine multiple middleware in a pipeline
- **Provider Agnostic**: Works with any provider implementing the core.Provider interface

## Installation

```go
import "github.com/recera/gai/middleware"
```

## Quick Start

```go
// Create a provider
provider := openai.New(openai.WithAPIKey(apiKey))

// Wrap with middleware
wrapped := middleware.Chain(
    middleware.WithRetry(middleware.DefaultRetryOpts()),
    middleware.WithRateLimit(middleware.DefaultRateLimitOpts()),
    middleware.WithSafety(middleware.DefaultSafetyOpts()),
)(provider)

// Use as normal
result, err := wrapped.GenerateText(ctx, request)
```

## Middleware Types

### Retry Middleware

Automatically retries failed requests with exponential backoff and jitter.

```go
provider = middleware.WithRetry(middleware.RetryOpts{
    MaxAttempts: 3,                    // Maximum retry attempts
    BaseDelay:   100 * time.Millisecond, // Initial delay
    MaxDelay:    10 * time.Second,      // Maximum delay between retries
    Multiplier:  2.0,                   // Exponential multiplier
    Jitter:      true,                  // Add randomization
    RetryIf: func(err error) bool {    // Custom retry logic (optional)
        return shouldRetry(err)
    },
})(provider)
```

**Features:**
- Exponential backoff with configurable multiplier
- Optional jitter to prevent thundering herd
- Respects rate limit retry-after headers
- Custom retry predicates
- Context cancellation support

**Default Behavior:**
- Retries on: transient errors, rate limits, timeouts
- Does not retry on: bad requests, auth errors, not found

### Rate Limiting Middleware

Enforces rate limits using a token bucket algorithm.

```go
provider = middleware.WithRateLimit(middleware.RateLimitOpts{
    RPS:         10,                    // Requests per second
    Burst:       20,                    // Maximum burst size
    WaitTimeout: 30 * time.Second,      // Max wait time for a token
    PerMethod: map[string]*middleware.RateLimitConfig{
        "GenerateText": {RPS: 5, Burst: 10},
        "StreamText":   {RPS: 2, Burst: 5},
    },
    OnRateLimited: func(method string, wait time.Duration) {
        log.Printf("Rate limited: %s, waiting %v", method, wait)
    },
})(provider)
```

**Features:**
- Token bucket algorithm for smooth rate limiting
- Per-method rate limits
- Configurable wait timeout
- Observable rate limit events
- Dynamic rate limit updates

### Safety Middleware

Filters and redacts sensitive content in requests and responses.

```go
provider = middleware.WithSafety(middleware.SafetyOpts{
    RedactPatterns: []string{
        `\b\d{3}-\d{2}-\d{4}\b`,        // SSN
        `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email
    },
    RedactReplacement: "[REDACTED]",
    BlockWords: []string{"forbidden", "inappropriate"},
    MaxContentLength: 10000,
    TransformRequest: func(messages []core.Message) ([]core.Message, error) {
        // Custom request transformation
        return messages, nil
    },
    TransformResponse: func(text string) (string, error) {
        // Custom response transformation
        return text, nil
    },
    OnBlocked: func(reason, content string) {
        log.Printf("Content blocked: %s", reason)
    },
    OnRedacted: func(pattern string, count int) {
        log.Printf("Redacted %d instances of %s", count, pattern)
    },
    StopOnSafetyEvent: true,  // Stop streaming on safety events
})(provider)
```

**Features:**
- Regex-based pattern redaction
- Word/phrase blocking
- Content length limits
- Custom transformation functions
- Stream filtering support
- Observable safety events

**Default PII Patterns:**
- Social Security Numbers (XXX-XX-XXXX)
- Email addresses
- Phone numbers
- Credit card numbers

## Middleware Composition

Use `Chain` to combine multiple middleware in order:

```go
// Middleware are applied in order: retry -> rate limit -> safety -> provider
provider = middleware.Chain(
    middleware.WithRetry(retryOpts),
    middleware.WithRateLimit(rateLimitOpts),
    middleware.WithSafety(safetyOpts),
)(provider)
```

The first middleware in the chain is the outermost layer, receiving requests first and responses last.

## Examples

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/recera/gai/core"
    "github.com/recera/gai/middleware"
    "github.com/recera/gai/providers/openai"
)

func main() {
    // Create provider
    provider := openai.New(
        openai.WithAPIKey("your-api-key"),
        openai.WithModel("gpt-4o-mini"),
    )
    
    // Add retry for resilience
    provider = middleware.WithRetry(middleware.RetryOpts{
        MaxAttempts: 3,
        BaseDelay:   500 * time.Millisecond,
    })(provider)
    
    // Use the wrapped provider
    ctx := context.Background()
    result, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {Role: core.User, Parts: []core.Part{
                core.Text{Text: "Hello, world!"},
            }},
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println(result.Text)
}
```

### Production Configuration

```go
// Production-ready middleware stack
provider = middleware.Chain(
    // Retry with exponential backoff
    middleware.WithRetry(middleware.RetryOpts{
        MaxAttempts: 3,
        BaseDelay:   200 * time.Millisecond,
        MaxDelay:    30 * time.Second,
        Jitter:      true,
    }),
    
    // Rate limiting per API limits
    middleware.WithRateLimit(middleware.RateLimitOpts{
        RPS:         50,  // 50 requests per second
        Burst:       100, // Allow bursts up to 100
        WaitTimeout: 60 * time.Second,
    }),
    
    // Safety filtering for PII
    middleware.WithSafety(middleware.SafetyOpts{
        RedactPatterns: []string{
            `\b\d{3}-\d{2}-\d{4}\b`,                     // SSN
            `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email
            `\b(?:\+?1[-.]?)?\(?[0-9]{3}\)?[-.]?[0-9]{3}[-.]?[0-9]{4}\b`, // Phone
        },
        RedactReplacement: "[PII_REMOVED]",
        MaxContentLength:  50000,
    }),
)(provider)
```

### Custom Retry Logic

```go
// Retry only specific errors
provider = middleware.WithRetry(middleware.RetryOpts{
    MaxAttempts: 5,
    BaseDelay:   1 * time.Second,
    RetryIf: func(err error) bool {
        // Custom logic: retry on specific error codes
        if aiErr, ok := err.(*core.AIError); ok {
            return aiErr.Code == "insufficient_quota" || 
                   aiErr.Code == "model_overloaded"
        }
        return core.IsTransient(err)
    },
})(provider)
```

### Per-Method Rate Limiting

```go
// Different limits for different operations
provider = middleware.WithRateLimit(middleware.RateLimitOpts{
    RPS:   10,  // Default for all methods
    Burst: 20,
    PerMethod: map[string]*middleware.RateLimitConfig{
        "GenerateText":   {RPS: 10, Burst: 20},  // Regular generation
        "StreamText":     {RPS: 5,  Burst: 10},  // Streaming (more expensive)
        "GenerateObject": {RPS: 20, Burst: 40},  // Structured (cheaper)
    },
})(provider)
```

### Streaming with Safety Filtering

```go
// Safety filtering works with streaming
provider = middleware.WithSafety(middleware.SafetyOpts{
    RedactPatterns: []string{`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`}, // Credit cards
    RedactReplacement: "[CARD]",
    StopOnSafetyEvent: true,  // Stop stream if safety event occurs
})(provider)

stream, err := provider.StreamText(ctx, request)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for event := range stream.Events() {
    switch event.Type {
    case core.EventTextDelta:
        // Text is already filtered
        fmt.Print(event.TextDelta)
    case core.EventSafety:
        log.Printf("Safety event: %+v", event.Safety)
    }
}
```

## Performance

Middleware overhead is minimal:

| Middleware | Overhead (no action) | Overhead (with action) |
|------------|---------------------|------------------------|
| Retry | ~50ns | +retry delay |
| Rate Limit | ~100ns | +wait time |
| Safety | ~500ns | +1-5μs per pattern |
| Chain (all 3) | ~700ns | Combined |

See [benchmark_test.go](benchmark_test.go) for detailed performance metrics.

## Testing

Run unit tests:
```bash
go test ./middleware
```

Run benchmarks:
```bash
go test -bench=. ./middleware
```

Run integration tests (requires API key):
```bash
OPENAI_API_KEY=your-key go test -tags=integration ./middleware
```

## Best Practices

1. **Order Matters**: Place retry as the outermost middleware so it can retry the entire chain
2. **Configure Appropriately**: Set rate limits based on your API tier and usage patterns
3. **Monitor Events**: Use OnRateLimited, OnBlocked, OnRedacted callbacks for observability
4. **Test Thoroughly**: Use mock providers to test middleware behavior without API calls
5. **Handle Errors**: Check error types using core.Is* functions for proper handling

## Thread Safety

All middleware implementations are thread-safe and can be used concurrently from multiple goroutines.

## License

Apache-2.0