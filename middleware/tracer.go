package middleware

import (
	"context"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/observability"
)

// Tracer middleware records LLM spans for both blocking and streaming operations.
type Tracer struct {
	truncate int
}

func NewTracer(truncateLimit int) *Tracer { return &Tracer{truncate: truncateLimit} }

func (t *Tracer) Wrap() Wrapper {
	return func(next core.ProviderClient) core.ProviderClient {
		return &tracerProvider{next: next, truncate: t.truncate}
	}
}

type tracerProvider struct {
	next     core.ProviderClient
	truncate int
}

func (tp *tracerProvider) GetCompletion(ctx context.Context, parts core.LLMCallParts) (core.LLMResponse, error) {
	ctx, span := observability.StartLLMSpan(ctx, "generateText", parts)
	start := time.Now()
	resp, err := tp.next.GetCompletion(ctx, parts)
	// Would attach duration, usage, finishReason, error
	_ = start
	observability.EndLLMSpan(span, resp.FinishReason, &resp.Usage, err)
	return resp, err
}

func (tp *tracerProvider) StreamCompletion(ctx context.Context, parts core.LLMCallParts, handler core.StreamHandler) error {
	ctx, span := observability.StartLLMSpan(ctx, "streamText", parts)
	firstSeen := false
	err := tp.next.StreamCompletion(ctx, parts, func(ch core.StreamChunk) error {
		if !firstSeen && ch.Type == "content" {
			firstSeen = true
			observability.MarkFirstChunkLLM(ctx)
		}
		if ch.Type == "tool_call" && ch.Call != nil {
			observability.AddEventToolCall(ctx, ch.Call.Name, ch.Call.Arguments)
		}
		return handler(ch)
	})
	// Finish reason cannot always be known here; rely on End chunk’s FinishReason if needed.
	observability.EndLLMSpan(span, "", nil, err)
	return err
}
