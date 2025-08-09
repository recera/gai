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

	// Tools defines provider-native tool/function definitions available to the model
	Tools []ToolDefinition

	// Unified, provider-agnostic settings (applied when supported; otherwise warned/ignored)
	StopSequences []string
	TopP          *float64
	TopK          *int
	Seed          *int64

	// Optional request headers (e.g., for gateways)
	Headers map[string]string

	// ProviderOpts is an escape hatch for provider-specific options
	ProviderOpts map[string]any

	// ToolChoice controls provider tool selection behavior (e.g., "auto", "none", or provider-specific struct)
	ToolChoice any

	// SessionID ties multi-turn conversations for observability/evaluation
	SessionID string

	// Metadata holds arbitrary context for tracing/evaluations
	Metadata map[string]any

	// Optional expected output used by evaluation tooling
	ExpectedText string
	ExpectedJSON any
}

// Message represents a message in a conversation with an LLM
type Message struct {
	Role     string
	Contents []Content

	// ToolCallID, when set and Role=="tool", is used by providers (e.g., OpenAI) to associate
	// a tool result message with a prior tool call id.
	ToolCallID string

	// ToolName, when set and Role=="tool", identifies the tool name for providers that associate
	// tool results by name (e.g., Gemini functionResponse).
	ToolName string
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

	// ToolCalls, when present, represents provider-native tool calls the assistant requested.
	// For OpenAI this maps to choices[].message.tool_calls; for Anthropic to content blocks of type tool_use.
	ToolCalls []ToolCall
}

// TokenUsage holds the token count information for an LLM call
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// ToolDefinition describes a provider-native tool/function available to the model
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	JSONSchema  map[string]interface{} `json:"json_schema"`
}

// ToolCall represents a single tool invocation requested by the model
type ToolCall struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// StreamChunk is a normalized unit emitted during streaming
type StreamChunk struct {
	// Type is one of: "content", "tool_call", "end"
	Type string

	// Delta is the incremental text (for Type=="content")
	Delta string

	// Call is set when Type=="tool_call"
	Call *ToolCall

	// Usage may be included on final/end chunks if available
	Usage *TokenUsage

	// FinishReason may be set on final chunks
	FinishReason string
}

// StreamHandler receives streaming chunks
type StreamHandler func(chunk StreamChunk) error

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
	// StreamCompletion streams the response, invoking handler for each chunk. Implementations
	// that don't support server-sent events may choose to emulate streaming by sending a single chunk.
	StreamCompletion(ctx context.Context, parts LLMCallParts, handler StreamHandler) error
}

// LLMClient defines the main interface for an LLM client
type LLMClient interface {
	GetCompletion(ctx context.Context, parts LLMCallParts) (LLMResponse, error)
	GetResponseObject(ctx context.Context, parts LLMCallParts, v any) error

	// StreamCompletion provides a unified streaming API independent of provider specifics
	StreamCompletion(ctx context.Context, parts LLMCallParts, handler StreamHandler) error

	// RunWithTools orchestrates provider-native tool calling until completion.
	// It will repeatedly call the provider, execute tool calls via executor, append tool results,
	// and stop when the model returns a non-tool response.
	RunWithTools(ctx context.Context, parts LLMCallParts, executor func(call ToolCall) (string, error)) (LLMResponse, error)

	// StreamWithTools streams and handles tool calls mid-stream, injecting tool results and resuming
	StreamWithTools(ctx context.Context, parts LLMCallParts, executor func(call ToolCall) (string, error), handler StreamHandler) error
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

	// RequestID is a provider-specific request identifier if available
	RequestID string

	// Rate limit related metadata when available (provider-specific)
	RateLimitLimit     string
	RateLimitRemaining string
	RateLimitReset     string
}
