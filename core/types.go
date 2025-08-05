// Package core contains the core types and interfaces used throughout the gai library.
// This package has no dependencies on other gai packages to avoid circular imports.
package core

import (
	"context"
	"time"
)

// LLMCallParts represents the parameters for an LLM API call
type LLMCallParts struct {
	Provider    string
	Model       string
	System      Message
	Messages    []Message
	MaxTokens   int
	Temperature float64
	
	// Trace is an optional function to receive trace information about the LLM call
	Trace func(TraceInfo)
}

// Message represents a message in a conversation with an LLM
type Message struct {
	Role     string
	Contents []Content
}

// Content is an interface for different types of content in a message
type Content interface {
	IsContent()
}

// TextContent represents text content in a message
type TextContent struct {
	Text string
}

func (t TextContent) IsContent() {}

// ImageContent represents image content in a message
type ImageContent struct {
	MIMEType string
	Data     []byte
	URL      string
}

func (i ImageContent) IsContent() {}

// LLMResponse represents a standardized response from any LLM provider
type LLMResponse struct {
	// The primary text content of the response.
	Content string

	// The reason the model stopped generating output.
	// Common values: "stop", "length", "tool_calls", etc.
	FinishReason string

	// Token usage statistics for the call.
	Usage TokenUsage
}

// TokenUsage holds the token count information for an LLM call
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// TraceInfo contains information about an LLM call for debugging and monitoring
type TraceInfo struct {
	Provider    string
	Model       string
	Attempt     int
	Prompt      string
	RawResponse string
	ParseErr    error
	Elapsed     time.Duration
}

// ProviderClient is the interface that all provider clients must implement
type ProviderClient interface {
	GetCompletion(ctx context.Context, parts LLMCallParts) (LLMResponse, error)
}

// LLMClient defines the main interface for an LLM client
type LLMClient interface {
	GetCompletion(ctx context.Context, parts LLMCallParts) (LLMResponse, error)
	GetResponseObject(ctx context.Context, parts LLMCallParts, v any) error
}

// LLMError represents a structured error from an LLM operation
type LLMError struct {
	// The underlying error
	Err error
	
	// Provider that generated the error
	Provider string
	
	// Model that was being used
	Model string
	
	// HTTP status code if applicable
	StatusCode int
	
	// Last raw response received
	LastRaw string
	
	// Additional context about the error
	Context map[string]interface{}
}