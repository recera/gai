package gai

import (
	"fmt"
	"strings"
)

// ModelKeyResolver allows resolving "provider:model" to set Provider/Model on parts.
type ModelKeyResolver interface {
	ApplyModelKey(parts *LLMCallParts, key string) error
}

// ApplyModelKey applies a model key of form "provider:model" to LLMCallParts using the client's registry if available.
func ApplyModelKey(c LLMClient, parts *LLMCallParts, key string) error {
	if parts == nil {
		return fmt.Errorf("nil parts")
	}
	if r, ok := c.(ModelKeyResolver); ok {
		return r.ApplyModelKey(parts, key)
	}
	// Fallback: parse provider:model directly
	seg := strings.SplitN(key, ":", 2)
	if len(seg) != 2 {
		return fmt.Errorf("invalid model key: %s", key)
	}
	parts.Provider = seg[0]
	parts.Model = seg[1]
	return nil
}

// NewLLMCallPartsFor creates LLMCallParts and applies the given model key using the client's registry if present.
func NewLLMCallPartsFor(c LLMClient, key string) (*LLMCallParts, error) {
	p := NewLLMCallParts()
	if err := ApplyModelKey(c, p, key); err != nil {
		return nil, err
	}
	return p, nil
}
