package gai

import (
	"context"

	responseParser "github.com/recera/gai/responseParser"
)

// Action represents a type-safe LLM action that binds a request with its expected result type.
// This pattern ensures compile-time type safety and makes the API more ergonomic.
type Action[T any] struct {
	Parts LLMCallParts
}

// NewAction creates a new Action with the specified result type.
// The type parameter T represents the expected response structure.
func NewAction[T any]() *Action[T] {
	return &Action[T]{
		Parts: *NewLLMCallParts(),
	}
}

// Run executes the action using the provided LLM client and returns the typed result.
// It automatically handles the response parsing into the generic type T.
func (a *Action[T]) Run(ctx context.Context, c LLMClient) (*T, error) {
	var v T
	if err := c.GetResponseObject(ctx, a.Parts, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// RunWithTools executes this action with tool-calling support and returns the typed result.
// It requires that tools have been configured via WithTools on the underlying parts.
func (a *Action[T]) RunWithTools(ctx context.Context, c LLMClient, executor func(call ToolCall) (string, error)) (*T, error) {
	resp, err := c.RunWithTools(ctx, a.Parts, executor)
	if err != nil {
		return nil, err
	}
	var v T
	if err := responseParser.ParseInto(resp.Content, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Fluent builder methods that delegate to LLMCallParts methods

// WithProvider sets the LLM provider for this action
func (a *Action[T]) WithProvider(s string) *Action[T] {
	a.Parts.Provider = s
	return a
}

// WithModel sets the model name for this action
func (a *Action[T]) WithModel(s string) *Action[T] {
	a.Parts.Model = s
	return a
}

// WithTemp sets the temperature parameter for this action
func (a *Action[T]) WithTemp(t float64) *Action[T] {
	a.Parts.Temperature = t
	return a
}

// WithMaxTokens sets the maximum tokens for this action
func (a *Action[T]) WithMaxTokens(n int) *Action[T] {
	a.Parts.MaxTokens = n
	return a
}

// WithSystem sets the system message using a text string
func (a *Action[T]) WithSystem(text string) *Action[T] {
	a.Parts.System.Role = "system"
	a.Parts.System.AddTextContent(text)
	return a
}

// WithUserMessage adds a user message to the conversation
func (a *Action[T]) WithUserMessage(text string) *Action[T] {
	msg := Message{Role: "user"}
	msg.AddTextContent(text)
	a.Parts.AddMessage(msg)
	return a
}

// WithAssistantMessage adds an assistant message to the conversation
func (a *Action[T]) WithAssistantMessage(text string) *Action[T] {
	msg := Message{Role: "assistant"}
	msg.AddTextContent(text)
	a.Parts.AddMessage(msg)
	return a
}

// WithMessage adds a pre-constructed message to the conversation
func (a *Action[T]) WithMessage(msg Message) *Action[T] {
	a.Parts.AddMessage(msg)
	return a
}

// GetParts returns the underlying LLMCallParts for advanced manipulation
func (a *Action[T]) GetParts() *LLMCallParts {
	return &a.Parts
}
