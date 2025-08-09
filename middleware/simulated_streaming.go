package middleware

import (
	"context"
	"time"

	"github.com/recera/gai/core"
)

// SimulatedStreaming wraps providers that do not support streaming, by chunking content.
func SimulatedStreaming(delay time.Duration, chunkSize int) Wrapper {
	if chunkSize <= 0 {
		chunkSize = 64
	}
	return func(next core.ProviderClient) core.ProviderClient {
		return &simStreamProvider{next: next, delay: delay, chunk: chunkSize}
	}
}

type simStreamProvider struct {
	next  core.ProviderClient
	delay time.Duration
	chunk int
}

func (s *simStreamProvider) GetCompletion(ctx context.Context, parts core.LLMCallParts) (core.LLMResponse, error) {
	return s.next.GetCompletion(ctx, parts)
}

func (s *simStreamProvider) StreamCompletion(ctx context.Context, parts core.LLMCallParts, handler core.StreamHandler) error {
	resp, err := s.next.GetCompletion(ctx, parts)
	if err != nil {
		return err
	}
	content := resp.Content
	for i := 0; i < len(content); i += s.chunk {
		end := i + s.chunk
		if end > len(content) {
			end = len(content)
		}
		if err := handler(core.StreamChunk{Type: "content", Delta: content[i:end]}); err != nil {
			return err
		}
		if s.delay > 0 {
			time.Sleep(s.delay)
		}
	}
	return handler(core.StreamChunk{Type: "end", FinishReason: resp.FinishReason})
}
