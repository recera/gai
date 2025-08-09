package middleware

import (
	"context"

	"github.com/recera/gai/core"
)

// Defaults applies default settings when unset.
func Defaults(def func(*core.LLMCallParts)) Wrapper {
	return func(next core.ProviderClient) core.ProviderClient {
		return &wrappedProvider{next: next, apply: def}
	}
}

type wrappedProvider struct {
	next  core.ProviderClient
	apply func(*core.LLMCallParts)
}

func (w *wrappedProvider) GetCompletion(ctx context.Context, parts core.LLMCallParts) (core.LLMResponse, error) {
	if w.apply != nil {
		w.apply(&parts)
	}
	return w.next.GetCompletion(ctx, parts)
}

func (w *wrappedProvider) StreamCompletion(ctx context.Context, parts core.LLMCallParts, handler core.StreamHandler) error {
	if w.apply != nil {
		w.apply(&parts)
	}
	return w.next.StreamCompletion(ctx, parts, handler)
}
