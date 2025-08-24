package middleware

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/recera/gai/core"
)

// RetryOpts configures the retry middleware.
type RetryOpts struct {
	// MaxAttempts is the maximum number of retry attempts (0 = no retries).
	MaxAttempts int
	// BaseDelay is the initial delay between retries.
	BaseDelay time.Duration
	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration
	// Multiplier is the exponential backoff multiplier (typically 2).
	Multiplier float64
	// Jitter adds randomization to retry delays to avoid thundering herd.
	Jitter bool
	// RetryIf is a custom function to determine if an error should be retried.
	// If nil, uses default retry logic based on error classification.
	RetryIf func(error) bool
}

// DefaultRetryOpts returns sensible default retry options.
func DefaultRetryOpts() RetryOpts {
	return RetryOpts{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Multiplier:  2.0,
		Jitter:      true,
		RetryIf:     nil, // Use default logic
	}
}

// retryMiddleware implements retry logic with exponential backoff.
type retryMiddleware struct {
	baseMiddleware
	opts   RetryOpts
	rand   *rand.Rand
	mu     sync.Mutex
}

// WithRetry creates middleware that retries transient failures with exponential backoff.
func WithRetry(opts RetryOpts) Middleware {
	// Validate and set defaults
	if opts.MaxAttempts < 0 {
		opts.MaxAttempts = 0
	}
	if opts.BaseDelay <= 0 {
		opts.BaseDelay = 100 * time.Millisecond
	}
	if opts.MaxDelay <= 0 {
		opts.MaxDelay = 10 * time.Second
	}
	if opts.Multiplier <= 1 {
		opts.Multiplier = 2.0
	}

	return func(provider core.Provider) core.Provider {
		return &retryMiddleware{
			baseMiddleware: baseMiddleware{provider: provider},
			opts:           opts,
			rand:           rand.New(rand.NewSource(time.Now().UnixNano())),
		}
	}
}

// shouldRetry determines if an error should trigger a retry.
func (m *retryMiddleware) shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Use custom retry function if provided
	if m.opts.RetryIf != nil {
		return m.opts.RetryIf(err)
	}

	// Default retry logic based on error classification
	return core.IsTransient(err) || core.IsRateLimited(err) || core.IsTimeout(err)
}

// calculateDelay calculates the delay for the given attempt number.
func (m *retryMiddleware) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: delay = min(base * multiplier^attempt, maxDelay)
	delay := float64(m.opts.BaseDelay) * math.Pow(m.opts.Multiplier, float64(attempt))
	if delay > float64(m.opts.MaxDelay) {
		delay = float64(m.opts.MaxDelay)
	}

	// Add jitter if enabled (±25% randomization)
	if m.opts.Jitter {
		m.mu.Lock()
		jitter := 0.75 + m.rand.Float64()*0.5 // Range: [0.75, 1.25]
		m.mu.Unlock()
		delay *= jitter
	}

	// Handle rate limit retry-after header
	return time.Duration(delay)
}

// waitWithContext waits for the specified duration or until the context is cancelled.
func (m *retryMiddleware) waitWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// retryOperation executes an operation with retry logic.
func (m *retryMiddleware) retryOperation(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= m.opts.MaxAttempts; attempt++ {
		// Execute the operation
		err := operation()
		
		// Success - return immediately
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !m.shouldRetry(err) {
			return err
		}

		// Check if we've exhausted attempts
		if attempt >= m.opts.MaxAttempts {
			break
		}

		// Check for rate limit retry-after header
		delay := m.calculateDelay(attempt)
		if retryAfter := core.GetRetryAfter(err); retryAfter > 0 {
			// Use the retry-after value if it's provided
			delay = retryAfter
		}

		// Wait before retrying
		if err := m.waitWithContext(ctx, delay); err != nil {
			// Context cancelled during wait
			return lastErr
		}
	}

	return lastErr
}

// GenerateText implements the Provider interface with retry logic.
func (m *retryMiddleware) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
	var result *core.TextResult
	
	err := m.retryOperation(ctx, func() error {
		var err error
		result, err = m.provider.GenerateText(ctx, req)
		return err
	})
	
	return result, err
}

// StreamText implements the Provider interface with retry logic.
// Note: Streaming operations are only retried on initial connection failure,
// not on mid-stream errors.
func (m *retryMiddleware) StreamText(ctx context.Context, req core.Request) (core.TextStream, error) {
	var stream core.TextStream
	
	err := m.retryOperation(ctx, func() error {
		var err error
		stream, err = m.provider.StreamText(ctx, req)
		return err
	})
	
	return stream, err
}

// GenerateObject implements the Provider interface with retry logic.
func (m *retryMiddleware) GenerateObject(ctx context.Context, req core.Request, schema any) (*core.ObjectResult[any], error) {
	var result *core.ObjectResult[any]
	
	err := m.retryOperation(ctx, func() error {
		var err error
		result, err = m.provider.GenerateObject(ctx, req, schema)
		return err
	})
	
	return result, err
}

// StreamObject implements the Provider interface with retry logic.
// Note: Streaming operations are only retried on initial connection failure,
// not on mid-stream errors.
func (m *retryMiddleware) StreamObject(ctx context.Context, req core.Request, schema any) (core.ObjectStream[any], error) {
	var stream core.ObjectStream[any]
	
	err := m.retryOperation(ctx, func() error {
		var err error
		stream, err = m.provider.StreamObject(ctx, req, schema)
		return err
	})
	
	return stream, err
}