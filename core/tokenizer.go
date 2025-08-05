package core

import (
	"strings"
)

// Tokenizer is an interface for tokenizing text for different models
type Tokenizer interface {
	// CountTokens returns the approximate number of tokens in the text
	CountTokens(text string) int
	// GetModelMaxTokens returns the maximum tokens for a specific model
	GetModelMaxTokens(model string) int
}

// SimpleTokenizer provides basic token counting
// For production use, integrate with tiktoken-go or similar
type SimpleTokenizer struct{}

// CountTokens estimates tokens using a simple heuristic
func (t SimpleTokenizer) CountTokens(text string) int {
	// Simple estimation: ~4 characters per token
	// This is a rough approximation
	words := strings.Fields(text)
	chars := len(text)
	
	// Use the average of word count * 1.3 and character count / 4
	wordEstimate := int(float64(len(words)) * 1.3)
	charEstimate := chars / 4
	
	return (wordEstimate + charEstimate) / 2
}

// GetModelMaxTokens returns the context window size for known models
func (t SimpleTokenizer) GetModelMaxTokens(model string) int {
	// Common model context windows
	windows := map[string]int{
		"gpt-4":                8192,
		"gpt-4-32k":            32768,
		"gpt-4o":               128000,
		"gpt-4o-mini":          128000,
		"gpt-3.5-turbo":        4096,
		"gpt-3.5-turbo-16k":    16384,
		"claude-3-opus":        200000,
		"claude-3-sonnet":      200000,
		"claude-3-haiku":       200000,
		"claude-2.1":           200000,
		"claude-2":             100000,
		"gemini-pro":           32760,
		"gemini-2.0-flash-exp": 32760,
		"llama-3.3-70b":        8192,
		"mixtral-8x7b":         32768,
	}
	
	if w, ok := windows[model]; ok {
		return w
	}
	
	// Default context window
	return 4096
}

// Token management methods for LLMCallParts

// EstimateTokens estimates the total tokens in the conversation
func (p *LLMCallParts) EstimateTokens(tokenizer Tokenizer) int {
	total := 0
	
	// Count system message tokens
	if p.System.GetTextContent() != "" {
		total += tokenizer.CountTokens(p.System.GetTextContent())
		total += 4 // Message overhead
	}
	
	// Count message tokens
	for _, msg := range p.Messages {
		text := msg.GetTextContent()
		if text != "" {
			total += tokenizer.CountTokens(text)
			total += 4 // Message overhead
		}
	}
	
	// Add some buffer for response formatting
	total += 10
	
	return total
}

// PruneToTokens removes oldest messages to fit within token limit
func (p *LLMCallParts) PruneToTokens(maxTokens int, tokenizer Tokenizer) (int, error) {
	current := p.EstimateTokens(tokenizer)
	removed := 0
	
	// If already within limit, return
	if current <= maxTokens {
		return 0, nil
	}
	
	// Keep system message and remove from the beginning
	for len(p.Messages) > 0 && current > maxTokens {
		// Estimate tokens in first message
		firstMsgTokens := tokenizer.CountTokens(p.Messages[0].GetTextContent()) + 4
		
		// Remove first message
		p.Messages = p.Messages[1:]
		current -= firstMsgTokens
		removed++
	}
	
	return removed, nil
}

// PruneKeepingRecent removes older messages while keeping the most recent n messages
func (p *LLMCallParts) PruneKeepingRecent(keepRecent int, maxTokens int, tokenizer Tokenizer) (int, error) {
	if len(p.Messages) <= keepRecent {
		return 0, nil
	}
	
	// Temporarily remove recent messages
	recentMessages := p.Messages[len(p.Messages)-keepRecent:]
	p.Messages = p.Messages[:len(p.Messages)-keepRecent]
	
	// Prune older messages
	removed, err := p.PruneToTokens(maxTokens, tokenizer)
	
	// Add recent messages back
	p.Messages = append(p.Messages, recentMessages...)
	
	return removed, err
}