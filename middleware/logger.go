package middleware

import (
	"context"
	"log"

	"github.com/recera/gai/core"
)

// Logger logs basic request/response info. Intended as example; apps can replace.
// Optional redactor can be provided to scrub sensitive fields in logs.
func Logger(redactors ...func(string) string) Wrapper {
	var redact func(string) string
	if len(redactors) > 0 && redactors[0] != nil {
		redact = redactors[0]
	}
	return func(next core.ProviderClient) core.ProviderClient {
		return &loggerProvider{next: next, redact: redact}
	}
}

type loggerProvider struct {
	next   core.ProviderClient
	redact func(string) string
}

func (l *loggerProvider) GetCompletion(ctx context.Context, parts core.LLMCallParts) (core.LLMResponse, error) {
	prov, model := parts.Provider, parts.Model
	if l.redact != nil {
		prov = l.redact(prov)
		model = l.redact(model)
	}
	log.Printf("llm start provider=%s model=%s", prov, model)
	resp, err := l.next.GetCompletion(ctx, parts)
	if err != nil {
		log.Printf("llm error: %v", err)
		return resp, err
	}
	finish := resp.FinishReason
	if l.redact != nil {
		finish = l.redact(finish)
	}
	log.Printf("llm done provider=%s model=%s finish=%s", prov, model, finish)
	return resp, nil
}

func (l *loggerProvider) StreamCompletion(ctx context.Context, parts core.LLMCallParts, handler core.StreamHandler) error {
	prov, model := parts.Provider, parts.Model
	if l.redact != nil {
		prov = l.redact(prov)
		model = l.redact(model)
	}
	log.Printf("llm stream start provider=%s model=%s", prov, model)
	return l.next.StreamCompletion(ctx, parts, func(ch core.StreamChunk) error {
		if ch.Type == "end" {
			finish := ch.FinishReason
			if l.redact != nil {
				finish = l.redact(finish)
			}
			log.Printf("llm stream end finish=%s", finish)
		}
		return handler(ch)
	})
}
