package middleware

import (
	"context"

	"github.com/recera/gai/core"
)

// Wrapper wraps a ProviderClient with cross-cutting behavior.
type Wrapper func(next core.ProviderClient) core.ProviderClient

// Chain composes multiple wrappers into one.
func Chain(pc core.ProviderClient, wrappers ...Wrapper) core.ProviderClient {
	wrapped := pc
	for i := len(wrappers) - 1; i >= 0; i-- {
		wrapped = wrappers[i](wrapped)
	}
	return wrapped
}

// StreamWrapper wraps a streaming call with cross-cutting behavior.
type StreamWrapper func(ctx context.Context, parts core.LLMCallParts, handler core.StreamHandler, next func(context.Context, core.LLMCallParts, core.StreamHandler) error) error
