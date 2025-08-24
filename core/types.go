// Package core defines the fundamental types and interfaces for the AI framework.
// It provides provider-agnostic abstractions for messages, requests, responses,
// and streaming events that work across all AI providers.
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Role represents the role of a message participant in a conversation.
type Role string

const (
	// System represents system-level instructions or context
	System Role = "system"
	// User represents input from the user
	User Role = "user"
	// Assistant represents responses from the AI assistant
	Assistant Role = "assistant"
	// Tool represents results from tool executions
	Tool Role = "tool"
)

// Part represents a component of a multimodal message.
// It uses a sealed interface pattern for compile-time exhaustiveness.
type Part interface {
	isPart()
	// partType returns a string identifier for the part type (for JSON marshaling)
	partType() string
}

// Text represents textual content in a message.
type Text struct {
	Text string `json:"text"`
}

func (Text) isPart()          {}
func (Text) partType() string { return "text" }

// ImageURL represents an image by URL reference.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "low", "high", or "auto"
}

func (ImageURL) isPart()          {}
func (ImageURL) partType() string { return "image_url" }

// BlobKind represents the type of blob reference.
type BlobKind uint8

const (
	// BlobURL references content by URL
	BlobURL BlobKind = iota
	// BlobBytes contains inline content bytes
	BlobBytes
	// BlobProviderFile references a provider-specific file ID
	BlobProviderFile
)

// BlobRef is a universal reference for files and media content.
// It can represent URLs, inline bytes, or provider-specific file IDs.
type BlobRef struct {
	Kind   BlobKind `json:"kind"`
	URL    string   `json:"url,omitempty"`
	Bytes  []byte   `json:"bytes,omitempty"`
	FileID string   `json:"file_id,omitempty"`
	MIME   string   `json:"mime,omitempty"`
	Size   int64    `json:"size,omitempty"`
}

// Audio represents audio content in a message.
type Audio struct {
	Source     BlobRef `json:"source"`
	SampleRate int     `json:"sample_rate,omitempty"`
	Channels   int     `json:"channels,omitempty"`
	Duration   float64 `json:"duration_seconds,omitempty"`
}

func (Audio) isPart()          {}
func (Audio) partType() string { return "audio" }

// Video represents video content in a message.
type Video struct {
	Source   BlobRef `json:"source"`
	Duration float64 `json:"duration_seconds,omitempty"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
}

func (Video) isPart()          {}
func (Video) partType() string { return "video" }

// File represents a generic file attachment in a message.
type File struct {
	Source  BlobRef `json:"source"`
	Name    string  `json:"name,omitempty"`
	Purpose string  `json:"purpose,omitempty"`
}

func (File) isPart()          {}
func (File) partType() string { return "file" }

// Message represents a single message in a conversation.
type Message struct {
	Role  Role   `json:"role"`
	Parts []Part `json:"parts"`
	Name  string `json:"name,omitempty"` // Optional participant name
}

// ToolChoice specifies how the model should use tools.
type ToolChoice int

const (
	// ToolAuto lets the model decide whether to use tools
	ToolAuto ToolChoice = iota
	// ToolNone prevents the model from using tools
	ToolNone
	// ToolRequired forces the model to use at least one tool
	ToolRequired
	// ToolSpecific forces the model to use a specific tool
	ToolSpecific
)

// SafetyLevel represents content safety thresholds.
type SafetyLevel string

const (
	SafetyBlockNone   SafetyLevel = "block_none"
	SafetyBlockFew    SafetyLevel = "block_few"
	SafetyBlockSome   SafetyLevel = "block_some"
	SafetyBlockMost   SafetyLevel = "block_most"
	SafetyBlockAlways SafetyLevel = "block_always"
)

// SafetyConfig defines content safety thresholds for various categories.
type SafetyConfig struct {
	Harassment SafetyLevel `json:"harassment,omitempty"`
	Hate       SafetyLevel `json:"hate,omitempty"`
	Sexual     SafetyLevel `json:"sexual,omitempty"`
	Dangerous  SafetyLevel `json:"dangerous,omitempty"`
}

// Session represents a conversation session with potential caching.
type Session struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
}

// Request represents a unified request to any AI provider.
type Request struct {
	// RequestID is a unique identifier for this request (auto-generated if empty)
	RequestID string `json:"request_id,omitempty"`
	// IdempotencyKey enables request deduplication (client-supplied)
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	// Model specifies which model to use
	Model string `json:"model,omitempty"`
	// Messages contains the conversation history
	Messages []Message `json:"messages"`
	// Temperature controls randomness (0.0 = deterministic, 2.0 = very random)
	Temperature float32 `json:"temperature,omitempty"`
	// MaxTokens limits the response length
	MaxTokens int `json:"max_tokens,omitempty"`
	// Tools available for the model to use
	Tools []ToolHandle `json:"tools,omitempty"`
	// ToolChoice controls how tools are used
	ToolChoice ToolChoice `json:"tool_choice,omitempty"`
	// SpecificTool names a specific tool when ToolChoice is ToolSpecific
	SpecificTool string `json:"specific_tool,omitempty"`
	// StopWhen defines conditions to stop multi-step execution
	StopWhen StopCondition `json:"-"`
	// Safety configuration for content filtering
	Safety *SafetyConfig `json:"safety,omitempty"`
	// Session for conversation caching
	Session *Session `json:"session,omitempty"`
	// ProviderOptions for provider-specific settings
	ProviderOptions map[string]any `json:"provider_options,omitempty"`
	// Metadata for tracking and telemetry
	Metadata map[string]any `json:"metadata,omitempty"`
	// Stream enables streaming responses
	Stream bool `json:"stream"`
}

// ToolHandle represents a tool that can be executed by the AI.
// This is defined here to avoid circular dependencies, but the
// concrete implementation is in the tools package.
type ToolHandle interface {
	// Name returns the unique identifier for this tool
	Name() string
	// Description returns a human-readable description of what the tool does
	Description() string
	// InSchemaJSON returns the JSON Schema for the tool's input parameters
	InSchemaJSON() []byte
	// OutSchemaJSON returns the JSON Schema for the tool's output
	OutSchemaJSON() []byte
	// Exec executes the tool with raw JSON input and returns the result
	Exec(ctx context.Context, raw json.RawMessage, meta interface{}) (any, error)
}

// Usage tracks token consumption for a request.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ToolCall represents a request to execute a tool.
type ToolCall struct {
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolExecution represents the result of executing a tool.
type ToolExecution struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name"`
	Result any    `json:"result"`
	Error  string `json:"error,omitempty"`
}

// Step represents one step in a multi-step execution.
type Step struct {
	// Text output from this step
	Text string `json:"text,omitempty"`
	// ToolCalls made during this step
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// ToolResults from executing tool calls
	ToolResults []ToolExecution `json:"tool_results,omitempty"`
	// StepNumber for ordering
	StepNumber int `json:"step_number"`
	// Timestamp when the step completed
	Timestamp time.Time `json:"timestamp"`
}

// TextResult represents the complete result of a text generation request.
type TextResult struct {
	// Text is the final generated text
	Text string `json:"text"`
	// Steps contains the execution history for multi-step runs
	Steps []Step `json:"steps,omitempty"`
	// Usage tracks token consumption
	Usage Usage `json:"usage"`
	// Raw contains provider-specific response data
	Raw any `json:"raw,omitempty"`
}

// ObjectResult represents a structured output result with a typed value.
type ObjectResult[T any] struct {
	// Value is the parsed and validated object
	Value T `json:"value"`
	// Steps contains the execution history
	Steps []Step `json:"steps,omitempty"`
	// Usage tracks token consumption
	Usage Usage `json:"usage"`
	// Raw contains provider-specific response data
	Raw any `json:"raw,omitempty"`
}

// EventType identifies the type of streaming event.
type EventType int

const (
	// EventStart signals the beginning of a stream
	EventStart EventType = iota
	// EventTextDelta contains incremental text
	EventTextDelta
	// EventAudioDelta contains incremental audio data
	EventAudioDelta
	// EventToolCall indicates a tool is being called
	EventToolCall
	// EventToolResult contains a tool execution result
	EventToolResult
	// EventCitations provides source citations
	EventCitations
	// EventSafety contains content safety information
	EventSafety
	// EventFinishStep marks the end of a step
	EventFinishStep
	// EventFinish marks the end of the stream
	EventFinish
	// EventError indicates an error occurred
	EventError
	// EventRaw contains provider-specific event data
	EventRaw
)

// String returns the string representation of an EventType.
func (e EventType) String() string {
	switch e {
	case EventStart:
		return "start"
	case EventTextDelta:
		return "text_delta"
	case EventAudioDelta:
		return "audio_delta"
	case EventToolCall:
		return "tool_call"
	case EventToolResult:
		return "tool_result"
	case EventCitations:
		return "citations"
	case EventSafety:
		return "safety"
	case EventFinishStep:
		return "finish_step"
	case EventFinish:
		return "finish"
	case EventError:
		return "error"
	case EventRaw:
		return "raw"
	default:
		return fmt.Sprintf("unknown(%d)", e)
	}
}

// AudioFormat describes audio stream format.
type AudioFormat struct {
	MIME       string `json:"mime"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	BitDepth   int    `json:"bit_depth"`
}

// Citation represents a source reference in generated content.
type Citation struct {
	URI   string `json:"uri"`
	Start int    `json:"start"` // Start position in text
	End   int    `json:"end"`   // End position in text
	Title string `json:"title,omitempty"`
}

// SafetyEvent contains content safety information.
type SafetyEvent struct {
	Category string  `json:"category"`
	Action   string  `json:"action"` // "block", "warn", "pass"
	Score    float32 `json:"score"`
	Note     string  `json:"note,omitempty"`
}

// Event represents a streaming event from a provider.
// Using a single struct with optional fields to minimize allocations.
type Event struct {
	// Type identifies the event type
	Type EventType `json:"type"`
	// TextDelta contains incremental text (EventTextDelta)
	TextDelta string `json:"text_delta,omitempty"`
	// AudioChunk contains audio data (EventAudioDelta)
	AudioChunk []byte `json:"audio_chunk,omitempty"`
	// AudioFormat describes the audio format (EventAudioDelta)
	AudioFormat *AudioFormat `json:"audio_format,omitempty"`
	// Citations for the content (EventCitations)
	Citations []Citation `json:"citations,omitempty"`
	// Safety information (EventSafety)
	Safety *SafetyEvent `json:"safety,omitempty"`
	// ToolName being called (EventToolCall)
	ToolName string `json:"tool_name,omitempty"`
	// ToolID for the call (EventToolCall)
	ToolID string `json:"tool_id,omitempty"`
	// ToolInput arguments (EventToolCall)
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
	// ToolResult from execution (EventToolResult)
	ToolResult any `json:"tool_result,omitempty"`
	// StepNumber for multi-step execution (EventFinishStep)
	StepNumber int `json:"step_number,omitempty"`
	// Usage information (EventFinish)
	Usage *Usage `json:"usage,omitempty"`
	// Raw provider-specific data (EventRaw)
	Raw any `json:"raw,omitempty"`
	// Err contains error information (EventError)
	Err error `json:"error,omitempty"`
	// Timestamp of the event
	Timestamp time.Time `json:"timestamp"`
}

// TextStream represents a stream of events from a provider.
type TextStream interface {
	// Events returns a channel of events
	Events() <-chan Event
	// Close terminates the stream
	Close() error
}

// ObjectStream represents a stream that produces a typed object.
type ObjectStream[T any] interface {
	TextStream
	// Final returns the final validated object (blocks until complete)
	Final() (*T, error)
}

// Provider is the core interface that all AI providers must implement.
type Provider interface {
	// GenerateText generates text with optional multi-step tool execution
	GenerateText(ctx context.Context, req Request) (*TextResult, error)
	// StreamText streams text generation with events
	StreamText(ctx context.Context, req Request) (TextStream, error)
	// GenerateObject generates a structured object of type T
	GenerateObject(ctx context.Context, req Request, schema any) (*ObjectResult[any], error)
	// StreamObject streams generation of a structured object
	StreamObject(ctx context.Context, req Request, schema any) (ObjectStream[any], error)
}

// StopCondition defines when to stop multi-step execution.
type StopCondition interface {
	// ShouldStop returns true if execution should stop
	ShouldStop(stepCount int, lastStep Step) bool
}

// MaxSteps stops after a maximum number of steps.
type maxStepsCondition struct {
	max int
}

func (m maxStepsCondition) ShouldStop(stepCount int, _ Step) bool {
	return stepCount >= m.max
}

// MaxSteps returns a condition that stops after n steps.
func MaxSteps(n int) StopCondition {
	return maxStepsCondition{max: n}
}

// NoMoreTools stops when no tool calls are made in a step.
type noMoreToolsCondition struct{}

func (noMoreToolsCondition) ShouldStop(_ int, lastStep Step) bool {
	return len(lastStep.ToolCalls) == 0
}

// NoMoreTools returns a condition that stops when no more tools are called.
func NoMoreTools() StopCondition {
	return noMoreToolsCondition{}
}

// UntilToolSeen stops after a specific tool has been called.
type untilToolSeenCondition struct {
	toolName string
}

func (u untilToolSeenCondition) ShouldStop(_ int, lastStep Step) bool {
	for _, call := range lastStep.ToolCalls {
		if call.Name == u.toolName {
			return true
		}
	}
	return false
}

// UntilToolSeen returns a condition that stops after seeing a specific tool.
func UntilToolSeen(toolName string) StopCondition {
	return untilToolSeenCondition{toolName: toolName}
}

// CombineConditions creates a condition that stops if any sub-condition is met.
type combinedCondition struct {
	conditions []StopCondition
}

func (c combinedCondition) ShouldStop(stepCount int, lastStep Step) bool {
	for _, cond := range c.conditions {
		if cond.ShouldStop(stepCount, lastStep) {
			return true
		}
	}
	return false
}

// CombineConditions returns a condition that stops if any condition is met.
func CombineConditions(conditions ...StopCondition) StopCondition {
	return combinedCondition{conditions: conditions}
}