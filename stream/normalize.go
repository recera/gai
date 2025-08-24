// Package stream provides streaming utilities for AI responses.
// This file implements event normalization for stable wire format.
package stream

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/recera/gai/core"
)

// SchemaVersion defines the current wire format version.
const SchemaVersion = "gai.events.v1"

// NormalizedEventType represents event types as strings for wire format.
type NormalizedEventType string

const (
	// Stream lifecycle events
	EventTypeStart  NormalizedEventType = "start"
	EventTypeFinish NormalizedEventType = "finish"
	EventTypeError  NormalizedEventType = "error"

	// Content events
	EventTypeTextDelta  NormalizedEventType = "text.delta"
	EventTypeAudioDelta NormalizedEventType = "audio.delta"

	// Tool events
	EventTypeToolCall   NormalizedEventType = "tool.call"
	EventTypeToolResult NormalizedEventType = "tool.result"

	// Metadata events
	EventTypeCitations NormalizedEventType = "citations"
	EventTypeSafety    NormalizedEventType = "safety"
	EventTypeStepEnd   NormalizedEventType = "step.end"
)

// NormalizedEvent represents a normalized event for wire transmission.
// This format is stable across all providers and versions.
type NormalizedEvent struct {
	// Schema identifies the wire format version
	Schema string `json:"schema"`
	// Type identifies the event type as a string
	Type NormalizedEventType `json:"type"`
	// Timestamp when the event was created
	Timestamp int64 `json:"ts"`
	// Sequence number for ordering
	Sequence int64 `json:"seq,omitempty"`
	// TraceID for distributed tracing
	TraceID string `json:"trace_id,omitempty"`
	// RequestID uniquely identifies the request
	RequestID string `json:"request_id,omitempty"`
	// Step number in multi-step execution
	Step int `json:"step,omitempty"`
	// CallID for tool calls
	CallID string `json:"call_id,omitempty"`
	// Provider (only in start/finish events)
	Provider string `json:"provider,omitempty"`
	// Model (only in start/finish events)
	Model string `json:"model,omitempty"`

	// Event-specific data fields
	// Text delta content
	Text string `json:"text,omitempty"`
	// Audio delta data
	Audio *AudioData `json:"audio,omitempty"`
	// Tool call information
	ToolCall *ToolCallData `json:"tool_call,omitempty"`
	// Tool result data
	ToolResult any `json:"tool_result,omitempty"`
	// Citations list
	Citations []Citation `json:"citations,omitempty"`
	// Safety information
	Safety *SafetyData `json:"safety,omitempty"`
	// Usage statistics (finish event)
	Usage *UsageData `json:"usage,omitempty"`
	// Finish reason
	FinishReason string `json:"finish_reason,omitempty"`
	// Error information
	Error *ErrorData `json:"error,omitempty"`
}

// AudioData contains audio chunk information.
type AudioData struct {
	Chunk  []byte `json:"chunk,omitempty"`
	Format string `json:"format,omitempty"`
}

// ToolCallData contains tool invocation details.
type ToolCallData struct {
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// Citation represents a source reference.
type Citation struct {
	URI   string `json:"uri"`
	Title string `json:"title,omitempty"`
	Start int    `json:"start,omitempty"`
	End   int    `json:"end,omitempty"`
}

// SafetyData contains content safety information.
type SafetyData struct {
	Category string  `json:"category"`
	Action   string  `json:"action"`
	Score    float32 `json:"score"`
}

// UsageData contains token usage statistics.
type UsageData struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens,omitempty"`
}

// ErrorData contains error information.
type ErrorData struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Temporary  bool   `json:"temporary,omitempty"`
	RetryAfter int    `json:"retry_after_ms,omitempty"`
}

// Normalizer converts provider events to normalized wire format.
type Normalizer struct {
	schema    string
	traceID   string
	requestID string
	provider  string
	model     string
	sequence  atomic.Int64
}

// NewNormalizer creates a new event normalizer.
func NewNormalizer(requestID, traceID string) *Normalizer {
	return &Normalizer{
		schema:    SchemaVersion,
		traceID:   traceID,
		requestID: requestID,
	}
}

// WithProvider sets the provider name for start/finish events.
func (n *Normalizer) WithProvider(provider string) *Normalizer {
	n.provider = provider
	return n
}

// WithModel sets the model name for start/finish events.
func (n *Normalizer) WithModel(model string) *Normalizer {
	n.model = model
	return n
}

// Normalize converts a core.Event to normalized wire format.
func (n *Normalizer) Normalize(event core.Event) NormalizedEvent {
	// Increment sequence counter
	seq := n.sequence.Add(1)

	// Base event with common fields
	normalized := NormalizedEvent{
		Schema:    n.schema,
		Timestamp: event.Timestamp.UnixMilli(),
		Sequence:  seq,
		TraceID:   n.traceID,
		RequestID: n.requestID,
	}

	// Map event type and populate specific fields
	switch event.Type {
	case core.EventStart:
		normalized.Type = EventTypeStart
		normalized.Provider = n.provider
		normalized.Model = n.model

	case core.EventTextDelta:
		normalized.Type = EventTypeTextDelta
		normalized.Text = event.TextDelta

	case core.EventAudioDelta:
		normalized.Type = EventTypeAudioDelta
		if event.AudioFormat != nil {
			normalized.Audio = &AudioData{
				Chunk:  event.AudioChunk,
				Format: event.AudioFormat.MIME,
			}
		}

	case core.EventToolCall:
		normalized.Type = EventTypeToolCall
		normalized.CallID = event.ToolID
		normalized.ToolCall = &ToolCallData{
			Name:  event.ToolName,
			Input: event.ToolInput,
		}

	case core.EventToolResult:
		normalized.Type = EventTypeToolResult
		normalized.CallID = event.ToolID
		normalized.ToolResult = event.ToolResult

	case core.EventCitations:
		normalized.Type = EventTypeCitations
		citations := make([]Citation, len(event.Citations))
		for i, c := range event.Citations {
			citations[i] = Citation{
				URI:   c.URI,
				Title: c.Title,
				Start: c.Start,
				End:   c.End,
			}
		}
		normalized.Citations = citations

	case core.EventSafety:
		normalized.Type = EventTypeSafety
		if event.Safety != nil {
			normalized.Safety = &SafetyData{
				Category: event.Safety.Category,
				Action:   event.Safety.Action,
				Score:    event.Safety.Score,
			}
		}

	case core.EventFinishStep:
		normalized.Type = EventTypeStepEnd
		normalized.Step = event.StepNumber

	case core.EventFinish:
		normalized.Type = EventTypeFinish
		if event.Usage != nil {
			normalized.Usage = &UsageData{
				InputTokens:  event.Usage.InputTokens,
				OutputTokens: event.Usage.OutputTokens,
				TotalTokens:  event.Usage.TotalTokens,
			}
		}
		normalized.Provider = n.provider
		normalized.Model = n.model
		// TODO: Add finish_reason when available in core.Event

	case core.EventError:
		normalized.Type = EventTypeError
		if event.Err != nil {
			// Check if it's an AIError with structured data
			if aiErr, ok := event.Err.(*core.AIError); ok {
				normalized.Error = &ErrorData{
					Code:      string(aiErr.Code),
					Message:   aiErr.Message,
					Temporary: aiErr.Temporary,
				}
				if aiErr.RetryAfter != nil {
					normalized.Error.RetryAfter = int(aiErr.RetryAfter.Milliseconds())
				}
			} else {
				// Generic error
				normalized.Error = &ErrorData{
					Code:    "internal",
					Message: event.Err.Error(),
				}
			}
		}

	default:
		// Unknown event type - use raw passthrough
		normalized.Type = NormalizedEventType(fmt.Sprintf("raw.%d", event.Type))
	}

	return normalized
}

// NormalizedStream wraps a TextStream to emit normalized events.
type NormalizedStream struct {
	source     core.TextStream
	normalizer *Normalizer
	events     chan NormalizedEvent
	done       chan struct{}
}

// NewNormalizedStream creates a stream that emits normalized events.
func NewNormalizedStream(source core.TextStream, normalizer *Normalizer) *NormalizedStream {
	ns := &NormalizedStream{
		source:     source,
		normalizer: normalizer,
		events:     make(chan NormalizedEvent, 100),
		done:       make(chan struct{}),
	}

	// Start normalization goroutine
	go ns.normalize()

	return ns
}

// normalize processes source events and emits normalized versions.
func (ns *NormalizedStream) normalize() {
	defer close(ns.events)
	defer close(ns.done)

	for event := range ns.source.Events() {
		normalized := ns.normalizer.Normalize(event)
		select {
		case ns.events <- normalized:
		case <-ns.done:
			return
		}
	}
}

// Events returns the channel of normalized events.
func (ns *NormalizedStream) Events() <-chan NormalizedEvent {
	return ns.events
}

// Close stops the normalization process.
func (ns *NormalizedStream) Close() error {
	select {
	case <-ns.done:
		// Already closed
		return nil
	default:
		close(ns.done)
		return ns.source.Close()
	}
}

// JSONMarshal marshals a normalized event to JSON.
// This ensures consistent field ordering and format.
func (e NormalizedEvent) JSONMarshal() ([]byte, error) {
	return json.Marshal(e)
}

// CompactJSON returns a compact JSON representation.
func (e NormalizedEvent) CompactJSON() []byte {
	// Build minimal object with only present fields
	obj := map[string]any{
		"type": e.Type,
	}

	// Only include schema in start event
	if e.Type == EventTypeStart {
		obj["schema"] = e.Schema
		if e.RequestID != "" {
			obj["request_id"] = e.RequestID
		}
		if e.TraceID != "" {
			obj["trace_id"] = e.TraceID
		}
		if e.Provider != "" {
			obj["provider"] = e.Provider
		}
		if e.Model != "" {
			obj["model"] = e.Model
		}
	}

	// Add sequence for ordering (except start/finish)
	if e.Type != EventTypeStart && e.Type != EventTypeFinish {
		obj["seq"] = e.Sequence
	}

	// Add event-specific fields
	switch e.Type {
	case EventTypeTextDelta:
		obj["text"] = e.Text
	case EventTypeToolCall:
		obj["call_id"] = e.CallID
		obj["name"] = e.ToolCall.Name
		obj["input"] = e.ToolCall.Input
	case EventTypeToolResult:
		obj["call_id"] = e.CallID
		obj["output"] = e.ToolResult
	case EventTypeFinish:
		if e.Usage != nil {
			obj["usage"] = e.Usage
		}
		if e.FinishReason != "" {
			obj["finish_reason"] = e.FinishReason
		}
	case EventTypeError:
		if e.Error != nil {
			obj["code"] = e.Error.Code
			obj["message"] = e.Error.Message
			if e.Error.RetryAfter > 0 {
				obj["retry_after_ms"] = e.Error.RetryAfter
			}
		}
	}

	// Marshal to compact JSON
	data, _ := json.Marshal(obj)
	return data
}

// ParseNormalizedEvent parses a JSON byte slice into a NormalizedEvent.
func ParseNormalizedEvent(data []byte) (*NormalizedEvent, error) {
	var event NormalizedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to parse normalized event: %w", err)
	}
	return &event, nil
}

// ValidateSchema checks if an event has the expected schema version.
func ValidateSchema(event NormalizedEvent) error {
	if event.Type == EventTypeStart && event.Schema != SchemaVersion {
		return fmt.Errorf("unsupported schema version: %s (expected %s)", event.Schema, SchemaVersion)
	}
	return nil
}

// RequestIDGenerator generates unique request IDs if not provided.
type RequestIDGenerator interface {
	Generate() string
}

// DefaultRequestIDGenerator uses UUIDv7 format (time-ordered).
type DefaultRequestIDGenerator struct{}

// Generate creates a new request ID.
func (g *DefaultRequestIDGenerator) Generate() string {
	// Simple implementation - in production use a proper UUIDv7 library
	return fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), seq())
}

var requestSeq atomic.Int64

func seq() int64 {
	return requestSeq.Add(1)
}