package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/recera/gai/core"
	"golang.org/x/time/rate"
)

// RateLimitOpts configures the rate limiting middleware.
type RateLimitOpts struct {
	// RPS is the requests per second limit.
	RPS float64
	// Burst is the maximum burst size (tokens that can be consumed at once).
	Burst int
	// WaitTimeout is the maximum time to wait for a token.
	// If zero, waits indefinitely (respecting context deadline).
	WaitTimeout time.Duration
	// PerMethod allows different rate limits per method (GenerateText, StreamText, etc).
	// If nil, uses the global RPS/Burst settings for all methods.
	PerMethod map[string]*RateLimitConfig
	// OnRateLimited is called when a request is rate limited (for observability).
	OnRateLimited func(method string, waitTime time.Duration)
}

// RateLimitConfig specifies rate limit settings for a specific method.
type RateLimitConfig struct {
	RPS   float64
	Burst int
}

// DefaultRateLimitOpts returns sensible default rate limit options.
func DefaultRateLimitOpts() RateLimitOpts {
	return RateLimitOpts{
		RPS:         10,   // 10 requests per second
		Burst:       20,   // Allow bursts up to 20
		WaitTimeout: 30 * time.Second,
	}
}

// rateLimitMiddleware implements token bucket rate limiting.
type rateLimitMiddleware struct {
	baseMiddleware
	opts        RateLimitOpts
	globalLimit *rate.Limiter
	methodLimits map[string]*rate.Limiter
	mu          sync.RWMutex
}

// WithRateLimit creates middleware that enforces rate limits using a token bucket algorithm.
func WithRateLimit(opts RateLimitOpts) Middleware {
	// Validate options
	if opts.RPS <= 0 {
		opts.RPS = 10
	}
	if opts.Burst <= 0 {
		opts.Burst = int(opts.RPS * 2) // Default burst is 2x RPS
	}
	if opts.Burst < int(opts.RPS) {
		opts.Burst = int(opts.RPS) // Burst should be at least RPS
	}

	return func(provider core.Provider) core.Provider {
		m := &rateLimitMiddleware{
			baseMiddleware: baseMiddleware{provider: provider},
			opts:          opts,
			globalLimit:   rate.NewLimiter(rate.Limit(opts.RPS), opts.Burst),
			methodLimits:  make(map[string]*rate.Limiter),
		}

		// Initialize per-method limiters if configured
		if opts.PerMethod != nil {
			for method, config := range opts.PerMethod {
				if config.RPS > 0 && config.Burst > 0 {
					m.methodLimits[method] = rate.NewLimiter(rate.Limit(config.RPS), config.Burst)
				}
			}
		}

		return m
	}
}

// getLimiter returns the appropriate rate limiter for the given method.
func (m *rateLimitMiddleware) getLimiter(method string) *rate.Limiter {
	m.mu.RLock()
	limiter, exists := m.methodLimits[method]
	m.mu.RUnlock()

	if exists {
		return limiter
	}
	return m.globalLimit
}

// waitForToken waits for a rate limit token or returns an error if the wait times out.
func (m *rateLimitMiddleware) waitForToken(ctx context.Context, method string) error {
	limiter := m.getLimiter(method)

	// Create a context with timeout if configured
	waitCtx := ctx
	var cancel context.CancelFunc
	if m.opts.WaitTimeout > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, m.opts.WaitTimeout)
		defer cancel()
	}

	// Try to get a token immediately
	if limiter.Allow() {
		return nil
	}

	// Calculate wait time
	reservation := limiter.Reserve()
	waitTime := reservation.Delay()
	
	// If we can't wait that long, cancel the reservation and return error
	if waitTime > m.opts.WaitTimeout && m.opts.WaitTimeout > 0 {
		reservation.Cancel()
		return core.NewError(
			core.ErrorRateLimited,
			fmt.Sprintf("rate limit exceeded, would need to wait %v", waitTime),
			core.WithProvider("middleware"),
			core.WithRetryAfter(waitTime),
		)
	}

	// Notify observer if configured
	if m.opts.OnRateLimited != nil {
		m.opts.OnRateLimited(method, waitTime)
	}

	// Wait for the token
	timer := time.NewTimer(waitTime)
	defer timer.Stop()

	select {
	case <-waitCtx.Done():
		// Context cancelled or timed out
		reservation.Cancel()
		if waitCtx.Err() == context.DeadlineExceeded {
			return core.NewError(
				core.ErrorRateLimited,
				fmt.Sprintf("rate limit wait timeout after %v", m.opts.WaitTimeout),
				core.WithProvider("middleware"),
				core.WithRetryAfter(waitTime),
			)
		}
		return waitCtx.Err()
	case <-timer.C:
		// Successfully waited for token
		return nil
	}
}

// UpdateRateLimit dynamically updates the rate limit for a method or globally.
func (m *rateLimitMiddleware) UpdateRateLimit(method string, rps float64, burst int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if method == "" {
		// Update global limit
		m.globalLimit.SetLimit(rate.Limit(rps))
		m.globalLimit.SetBurst(burst)
	} else {
		// Update or create method-specific limit
		if limiter, exists := m.methodLimits[method]; exists {
			limiter.SetLimit(rate.Limit(rps))
			limiter.SetBurst(burst)
		} else {
			m.methodLimits[method] = rate.NewLimiter(rate.Limit(rps), burst)
		}
	}
}

// GenerateText implements the Provider interface with rate limiting.
func (m *rateLimitMiddleware) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
	if err := m.waitForToken(ctx, "GenerateText"); err != nil {
		return nil, err
	}
	return m.provider.GenerateText(ctx, req)
}

// StreamText implements the Provider interface with rate limiting.
func (m *rateLimitMiddleware) StreamText(ctx context.Context, req core.Request) (core.TextStream, error) {
	if err := m.waitForToken(ctx, "StreamText"); err != nil {
		return nil, err
	}
	return m.provider.StreamText(ctx, req)
}

// GenerateObject implements the Provider interface with rate limiting.
func (m *rateLimitMiddleware) GenerateObject(ctx context.Context, req core.Request, schema any) (*core.ObjectResult[any], error) {
	if err := m.waitForToken(ctx, "GenerateObject"); err != nil {
		return nil, err
	}
	return m.provider.GenerateObject(ctx, req, schema)
}

// StreamObject implements the Provider interface with rate limiting.
func (m *rateLimitMiddleware) StreamObject(ctx context.Context, req core.Request, schema any) (core.ObjectStream[any], error) {
	if err := m.waitForToken(ctx, "StreamObject"); err != nil {
		return nil, err
	}
	return m.provider.StreamObject(ctx, req, schema)
}

// TokenBucketRateLimiter is an interface that can be implemented by providers
// to expose their internal rate limiter for coordination with middleware.
type TokenBucketRateLimiter interface {
	// GetRateLimiter returns the internal rate limiter if available.
	GetRateLimiter() *rate.Limiter
	// SetRateLimiter sets a custom rate limiter.
	SetRateLimiter(*rate.Limiter)
}