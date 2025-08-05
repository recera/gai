package gai

import (
	"github.com/collinshill/gai/core"
)

// Type alias for LLMError
type LLMError = core.LLMError

// NewLLMError creates a new LLM error
func NewLLMError(err error, provider, model string) *LLMError {
	return &LLMError{
		Err:      err,
		Provider: provider,
		Model:    model,
	}
}

// WrapError wraps an error with LLM context
func WrapError(err error, parts LLMCallParts) *LLMError {
	return NewLLMError(err, parts.Provider, parts.Model)
}