package registry

import (
	"context"
	"testing"

	"github.com/recera/gai/core"
)

type fakeProv struct{}

func (f *fakeProv) GetCompletion(ctx context.Context, p core.LLMCallParts) (core.LLMResponse, error) {
	return core.LLMResponse{Content: "ok"}, nil
}
func (f *fakeProv) StreamCompletion(ctx context.Context, p core.LLMCallParts, h core.StreamHandler) error {
	return h(core.StreamChunk{Type: "end"})
}

func TestRegistry_Resolve(t *testing.T) {
	r := New()
	r.Register("openai", &fakeProv{})
	p, model, err := r.Resolve("openai:gpt-4o")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p == nil || model != "gpt-4o" {
		t.Fatalf("bad resolve")
	}
}
