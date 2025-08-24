// Package middleware provides composable middleware for AI providers.
// It includes retry logic, rate limiting, and safety filters that can be
// applied to any provider implementation.
package middleware

import (
	"context"

	"github.com/recera/gai/core"
)

// Middleware is a function that wraps a Provider with additional functionality.
type Middleware func(core.Provider) core.Provider

// Chain composes multiple middleware functions into a single middleware.
// The middleware are applied in the order they are provided, with the first
// middleware being the outermost layer.
//
// Example:
//
//	provider = middleware.Chain(
//	    middleware.WithRetry(retryOpts),
//	    middleware.WithRateLimit(rateLimitOpts),
//	    middleware.WithSafety(safetyOpts),
//	)(provider)
func Chain(middlewares ...Middleware) Middleware {
	return func(provider core.Provider) core.Provider {
		// Apply middleware in reverse order so the first middleware
		// is the outermost layer
		for i := len(middlewares) - 1; i >= 0; i-- {
			provider = middlewares[i](provider)
		}
		return provider
	}
}

// baseMiddleware provides a base implementation that delegates all methods to the wrapped provider.
// Specific middleware can embed this and override only the methods they need to modify.
type baseMiddleware struct {
	provider core.Provider
}

// GenerateText delegates to the wrapped provider.
func (m *baseMiddleware) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
	return m.provider.GenerateText(ctx, req)
}

// StreamText delegates to the wrapped provider.
func (m *baseMiddleware) StreamText(ctx context.Context, req core.Request) (core.TextStream, error) {
	return m.provider.StreamText(ctx, req)
}

// GenerateObject delegates to the wrapped provider.
func (m *baseMiddleware) GenerateObject(ctx context.Context, req core.Request, schema any) (*core.ObjectResult[any], error) {
	return m.provider.GenerateObject(ctx, req, schema)
}

// StreamObject delegates to the wrapped provider.
func (m *baseMiddleware) StreamObject(ctx context.Context, req core.Request, schema any) (core.ObjectStream[any], error) {
	return m.provider.StreamObject(ctx, req, schema)
}