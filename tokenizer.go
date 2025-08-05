package gai

import (
	"github.com/collinshill/gai/core"
)

// Type aliases for tokenizer types
type Tokenizer = core.Tokenizer
type SimpleTokenizer = core.SimpleTokenizer

// NewSimpleTokenizer creates a new simple tokenizer
func NewSimpleTokenizer() Tokenizer {
	return &SimpleTokenizer{}
}

// GetModelContextWindow returns the context window size for a model
func GetModelContextWindow(model string) int {
	tokenizer := SimpleTokenizer{}
	return tokenizer.GetModelMaxTokens(model)
}