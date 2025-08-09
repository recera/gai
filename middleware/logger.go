package middleware

import (
	"context"
	"log"

	"github.com/recera/gai/core"
)

// Logger logs basic request/response info. Intended as example; apps can replace.
func Logger() Wrapper {
	return func(next core.ProviderClient) core.ProviderClient {
		return &loggerProvider{next: next}
	}
}

type loggerProvider struct{ next core.ProviderClient }

func (l *loggerProvider) GetCompletion(ctx context.Context, parts core.LLMCallParts) (core.LLMResponse, error) {
	log.Printf("llm start provider=%s model=%s", parts.Provider, parts.Model)
	resp, err := l.next.GetCompletion(ctx, parts)
	if err != nil {
		log.Printf("llm error: %v", err)
		return resp, err
	}
	log.Printf("llm done provider=%s model=%s finish=%s", parts.Provider, parts.Model, resp.FinishReason)
	return resp, nil
}

func (l *loggerProvider) StreamCompletion(ctx context.Context, parts core.LLMCallParts, handler core.StreamHandler) error {
	log.Printf("llm stream start provider=%s model=%s", parts.Provider, parts.Model)
	return l.next.StreamCompletion(ctx, parts, func(ch core.StreamChunk) error {
		if ch.Type == "end" {
			log.Printf("llm stream end finish=%s", ch.FinishReason)
		}
		return handler(ch)
	})
}
