package middleware

import (
	"context"
	"regexp"

	"github.com/recera/gai/core"
)

// ReasoningExtraction extracts <think>...</think> blocks and removes them from user-visible content.
// It forwards a clean stream while allowing apps to capture the hidden reasoning via a callback.
func ReasoningExtraction(onReason func(string)) Wrapper {
	re := regexp.MustCompile(`(?s)<think>(.*?)</think>`) // non-greedy
	return func(next core.ProviderClient) core.ProviderClient {
		return &reasoningProvider{next: next, re: re, cb: onReason}
	}
}

type reasoningProvider struct {
	next core.ProviderClient
	re   *regexp.Regexp
	cb   func(string)
}

func (r *reasoningProvider) GetCompletion(ctx context.Context, parts core.LLMCallParts) (core.LLMResponse, error) {
	resp, err := r.next.GetCompletion(ctx, parts)
	if err != nil {
		return resp, err
	}
	if r.cb != nil {
		matches := r.re.FindAllStringSubmatch(resp.Content, -1)
		for _, m := range matches {
			if len(m) > 1 {
				r.cb(m[1])
			}
		}
	}
	// Strip
	resp.Content = r.re.ReplaceAllString(resp.Content, "")
	return resp, nil
}

func (r *reasoningProvider) StreamCompletion(ctx context.Context, parts core.LLMCallParts, handler core.StreamHandler) error {
	var buf string
	return r.next.StreamCompletion(ctx, parts, func(ch core.StreamChunk) error {
		if ch.Type == "content" {
			buf += ch.Delta
		}
		if ch.Type == "end" {
			if r.cb != nil {
				matches := r.re.FindAllStringSubmatch(buf, -1)
				for _, m := range matches {
					if len(m) > 1 {
						r.cb(m[1])
					}
				}
			}
		}
		if ch.Type == "content" {
			// Forward with thinking stripped
			ch.Delta = r.re.ReplaceAllString(ch.Delta, "")
		}
		return handler(ch)
	})
}
