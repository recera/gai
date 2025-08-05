package core

import (
	"fmt"
	"strings"
)

// Conversation utility methods for LLMCallParts

// CountMessages returns the total number of messages in the conversation
func (p *LLMCallParts) CountMessages() int {
	return len(p.Messages)
}

// GetLastUserMessage returns the last user message and its index, or ("", -1) if none found
func (p *LLMCallParts) GetLastUserMessage() (string, int) {
	for i := len(p.Messages) - 1; i >= 0; i-- {
		if p.Messages[i].Role == "user" {
			return p.Messages[i].GetTextContent(), i
		}
	}
	return "", -1
}

// FindLastMessage finds the last message with the given role
func (p *LLMCallParts) FindLastMessage(role string) (Message, int) {
	for i := len(p.Messages) - 1; i >= 0; i-- {
		if p.Messages[i].Role == role {
			return p.Messages[i], i
		}
	}
	return Message{}, -1
}

// FilterMessages returns messages that match the given predicate
func (p *LLMCallParts) FilterMessages(predicate func(Message) bool) []Message {
	var filtered []Message
	for _, msg := range p.Messages {
		if predicate(msg) {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// KeepLastMessages keeps only the last n messages
func (p *LLMCallParts) KeepLastMessages(n int) {
	if len(p.Messages) > n {
		p.Messages = p.Messages[len(p.Messages)-n:]
	}
}

// Transcript returns a string representation of the conversation
func (p *LLMCallParts) Transcript() string {
	var parts []string
	
	// Add system message if present
	if p.System.GetTextContent() != "" {
		parts = append(parts, fmt.Sprintf("System: %s", p.System.GetTextContent()))
	}
	
	// Add all messages
	for _, msg := range p.Messages {
		text := msg.GetTextContent()
		if text != "" {
			parts = append(parts, fmt.Sprintf("%s: %s", strings.Title(msg.Role), text))
		}
	}
	
	return strings.Join(parts, "\n\n")
}

// Clone creates a deep copy of LLMCallParts
func (p *LLMCallParts) Clone() LLMCallParts {
	clone := LLMCallParts{
		Provider:    p.Provider,
		Model:       p.Model,
		System:      p.System,
		MaxTokens:   p.MaxTokens,
		Temperature: p.Temperature,
		Trace:       p.Trace,
	}
	
	// Deep copy messages
	clone.Messages = make([]Message, len(p.Messages))
	copy(clone.Messages, p.Messages)
	
	return clone
}