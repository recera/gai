package observability

import (
	"context"
	"strings"

	"github.com/recera/gai/core"
)

// TracerConfig controls OTel tracer bootstrapping (real impl behind build tag otel).
type TracerConfig struct {
	Preset        string            // "braintrust" | "arize" | "custom" (env fallback if empty)
	Endpoint      string            // OTLP endpoint override
	Headers       map[string]string // OTLP auth headers
	SampleRatio   float64           // 0..1
	TruncateLimit int               // max chars for prompt/response in attrs (0 = default)
	// Redact returns a redacted copy of a string before attaching to telemetry.
	Redact func(string) string
}

// Enable sets up telemetry. No-op by default unless built with the "otel" tag.
func Enable(ctx context.Context, cfg TracerConfig) (func(context.Context) error, error) {
	return func(context.Context) error { return nil }, nil
}

// LLMSpan is an opaque handle to an LLM span.
type LLMSpan interface{ End() }

// StartLLMSpan begins a span for an LLM operation ("generateText" | "streamText").
// Default implementation is no-op; real impl provided with the "otel" tag.
func StartLLMSpan(ctx context.Context, operation string, parts core.LLMCallParts) (context.Context, LLMSpan) {
	_ = operation
	return ctx, noopSpan{}
}

// MarkFirstChunk marks time to first token in streaming flows.
func MarkFirstChunkLLM(ctx context.Context) { _ = ctx }

// AddEventToolCall records a tool call event on the active LLM span.
func AddEventToolCall(ctx context.Context, name, arguments string) {
	_ = ctx
	_ = name
	_ = arguments
}

// EndLLMSpan finalizes the span with finish reason, usage, and error if present.
func EndLLMSpan(span LLMSpan, finishReason string, usage *core.TokenUsage, err error) {
	_ = finishReason
	_ = usage
	_ = err
	if span != nil {
		span.End()
	}
}

// SafeTruncate helper used by middleware to enforce truncate limits in absence of real tracer.
func SafeTruncate(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}
	if limit < 3 {
		return s[:limit]
	}
	return s[:limit-3] + "…"
}

// JoinStrings safely joins slice into a single attribute-friendly string.
func JoinStrings(ss []string) string { return strings.Join(ss, ",") }
