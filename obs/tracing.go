// Package obs provides observability features for the GAI framework.
// It includes OpenTelemetry-based tracing, metrics, and usage accounting
// with zero overhead when observability is not configured.
package obs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	// tracer is the global tracer instance
	tracer trace.Tracer
	// tracerOnce ensures tracer is initialized only once
	tracerOnce sync.Once
	// noopTracer is used when tracing is disabled
	noopTracer = trace.NewNoopTracerProvider().Tracer("")
)

// Tracer returns the configured tracer or a noop tracer if not configured.
// This ensures zero overhead when tracing is disabled.
func Tracer() trace.Tracer {
	tracerOnce.Do(func() {
		// Check if a global tracer provider is configured
		provider := otel.GetTracerProvider()
		if provider == nil {
			tracer = noopTracer
		} else {
			tracer = provider.Tracer(
				"github.com/recera/gai",
				trace.WithInstrumentationVersion("1.0.0"),
			)
		}
	})
	return tracer
}

// SpanKind represents the type of span being created
type SpanKind string

const (
	SpanKindRequest   SpanKind = "request"
	SpanKindStep      SpanKind = "step"
	SpanKindTool      SpanKind = "tool"
	SpanKindPrompt    SpanKind = "prompt"
	SpanKindProvider  SpanKind = "provider"
	SpanKindStreaming SpanKind = "streaming"
)

// RequestSpanOptions contains options for creating a request span
type RequestSpanOptions struct {
	Provider        string
	Model           string
	Temperature     float32
	MaxTokens       int
	Stream          bool
	ToolCount       int
	MessageCount    int
	SystemPrompt    bool
	ProviderOptions map[string]any
	Metadata        map[string]any
}

// StartRequestSpan starts a new span for an AI request
func StartRequestSpan(ctx context.Context, opts RequestSpanOptions) (context.Context, trace.Span) {
	ctx, span := Tracer().Start(ctx, "ai.request",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("llm.provider", opts.Provider),
			attribute.String("llm.model", opts.Model),
			attribute.Float64("llm.temperature", float64(opts.Temperature)),
			attribute.Int("llm.max_tokens", opts.MaxTokens),
			attribute.Bool("llm.stream", opts.Stream),
			attribute.Int("llm.tools.count", opts.ToolCount),
			attribute.Int("llm.messages.count", opts.MessageCount),
			attribute.Bool("llm.system_prompt", opts.SystemPrompt),
		),
	)

	// Add provider-specific options as attributes
	for k, v := range opts.ProviderOptions {
		span.SetAttributes(attribute.String(fmt.Sprintf("llm.provider.%s", k), fmt.Sprint(v)))
	}

	// Add metadata as attributes
	for k, v := range opts.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata.%s", k), fmt.Sprint(v)))
	}

	return ctx, span
}

// StepSpanOptions contains options for creating a step span
type StepSpanOptions struct {
	StepNumber   int
	HasToolCalls bool
	ToolCount    int
	TextLength   int
}

// StartStepSpan starts a new span for a multi-step execution step
func StartStepSpan(ctx context.Context, opts StepSpanOptions) (context.Context, trace.Span) {
	ctx, span := Tracer().Start(ctx, fmt.Sprintf("ai.step.%d", opts.StepNumber),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.Int("step.number", opts.StepNumber),
			attribute.Bool("step.has_tool_calls", opts.HasToolCalls),
			attribute.Int("step.tool_count", opts.ToolCount),
			attribute.Int("step.text_length", opts.TextLength),
		),
	)
	return ctx, span
}

// ToolSpanOptions contains options for creating a tool span
type ToolSpanOptions struct {
	ToolName    string
	ToolID      string
	InputSize   int
	StepNumber  int
	Parallel    bool
	RetryCount  int
	Timeout     time.Duration
}

// StartToolSpan starts a new span for a tool execution
func StartToolSpan(ctx context.Context, opts ToolSpanOptions) (context.Context, trace.Span) {
	ctx, span := Tracer().Start(ctx, fmt.Sprintf("ai.tool.%s", opts.ToolName),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("tool.name", opts.ToolName),
			attribute.String("tool.id", opts.ToolID),
			attribute.Int("tool.input_size", opts.InputSize),
			attribute.Int("tool.step_number", opts.StepNumber),
			attribute.Bool("tool.parallel", opts.Parallel),
			attribute.Int("tool.retry_count", opts.RetryCount),
			attribute.Float64("tool.timeout_seconds", opts.Timeout.Seconds()),
		),
	)
	return ctx, span
}

// PromptSpanOptions contains options for creating a prompt span
type PromptSpanOptions struct {
	Name        string
	Version     string
	Fingerprint string
	DataKeys    []string
	Override    bool
	CacheHit    bool
}

// StartPromptSpan starts a new span for prompt rendering
func StartPromptSpan(ctx context.Context, opts PromptSpanOptions) (context.Context, trace.Span) {
	ctx, span := Tracer().Start(ctx, fmt.Sprintf("ai.prompt.%s", opts.Name),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("prompt.name", opts.Name),
			attribute.String("prompt.version", opts.Version),
			attribute.String("prompt.fingerprint", opts.Fingerprint),
			attribute.StringSlice("prompt.data_keys", opts.DataKeys),
			attribute.Bool("prompt.override", opts.Override),
			attribute.Bool("prompt.cache_hit", opts.CacheHit),
		),
	)
	return ctx, span
}

// StreamingSpanOptions contains options for creating a streaming span
type StreamingSpanOptions struct {
	Provider    string
	Model       string
	EventCount  int
	BytesCount  int
	Duration    time.Duration
	EventType   string
}

// StartStreamingSpan starts a new span for streaming operations
func StartStreamingSpan(ctx context.Context, opts StreamingSpanOptions) (context.Context, trace.Span) {
	ctx, span := Tracer().Start(ctx, "ai.streaming",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("streaming.provider", opts.Provider),
			attribute.String("streaming.model", opts.Model),
			attribute.String("streaming.event_type", opts.EventType),
		),
	)
	return ctx, span
}

// RecordStreamingMetrics adds streaming metrics to an existing span
func RecordStreamingMetrics(span trace.Span, eventCount, bytesCount int, duration time.Duration) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(
			attribute.Int("streaming.event_count", eventCount),
			attribute.Int("streaming.bytes_count", bytesCount),
			attribute.Float64("streaming.duration_ms", float64(duration.Milliseconds())),
		)
	}
}

// RecordUsage adds usage metrics to a span
func RecordUsage(span trace.Span, inputTokens, outputTokens, totalTokens int) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(
			attribute.Int("usage.input_tokens", inputTokens),
			attribute.Int("usage.output_tokens", outputTokens),
			attribute.Int("usage.total_tokens", totalTokens),
		)
	}
}

// RecordError records an error on a span with proper status
func RecordError(span trace.Span, err error, description string) {
	if span != nil && span.IsRecording() && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, description)
		span.SetAttributes(
			attribute.String("error.type", fmt.Sprintf("%T", err)),
			attribute.String("error.message", err.Error()),
		)
	}
}

// RecordToolResult adds tool execution result to a span
func RecordToolResult(span trace.Span, success bool, outputSize int, duration time.Duration) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(
			attribute.Bool("tool.success", success),
			attribute.Int("tool.output_size", outputSize),
			attribute.Float64("tool.duration_ms", float64(duration.Milliseconds())),
		)
		if success {
			span.SetStatus(codes.Ok, "Tool executed successfully")
		}
	}
}

// RecordProviderLatency adds provider latency metrics to a span
func RecordProviderLatency(span trace.Span, latency time.Duration, firstTokenLatency *time.Duration) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(
			attribute.Float64("provider.latency_ms", float64(latency.Milliseconds())),
		)
		if firstTokenLatency != nil {
			span.SetAttributes(
				attribute.Float64("provider.first_token_latency_ms", float64(firstTokenLatency.Milliseconds())),
			)
		}
	}
}

// RecordCacheMetrics adds cache-related metrics to a span
func RecordCacheMetrics(span trace.Span, hit bool, key string, size int) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(
			attribute.Bool("cache.hit", hit),
			attribute.String("cache.key", key),
			attribute.Int("cache.size_bytes", size),
		)
	}
}

// RecordSafetyMetrics adds safety-related metrics to a span
func RecordSafetyMetrics(span trace.Span, category string, action string, score float32) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(
			attribute.String("safety.category", category),
			attribute.String("safety.action", action),
			attribute.Float64("safety.score", float64(score)),
		)
	}
}

// RecordCitations adds citation information to a span
func RecordCitations(span trace.Span, count int, sources []string) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(
			attribute.Int("citations.count", count),
			attribute.StringSlice("citations.sources", sources),
		)
	}
}

// RecordAudioMetrics adds audio processing metrics to a span
func RecordAudioMetrics(span trace.Span, provider string, format string, durationMs int, bytes int) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(
			attribute.String("audio.provider", provider),
			attribute.String("audio.format", format),
			attribute.Int("audio.duration_ms", durationMs),
			attribute.Int("audio.bytes", bytes),
		)
	}
}

// RecordTTSMetrics adds text-to-speech metrics to a span
func RecordTTSMetrics(span trace.Span, provider string, voice string, format string, bytes int) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(
			attribute.String("tts.provider", provider),
			attribute.String("tts.voice", voice),
			attribute.String("tts.format", format),
			attribute.Int("tts.bytes", bytes),
		)
	}
}

// RecordSTTMetrics adds speech-to-text metrics to a span
func RecordSTTMetrics(span trace.Span, provider string, durationMs int, wordCount int) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(
			attribute.String("stt.provider", provider),
			attribute.Int("stt.duration_ms", durationMs),
			attribute.Int("stt.word_count", wordCount),
		)
	}
}

// WithSpan is a helper to execute a function within a span
func WithSpan(ctx context.Context, name string, fn func(context.Context, trace.Span) error) error {
	ctx, span := Tracer().Start(ctx, name)
	defer span.End()
	
	err := fn(ctx, span)
	if err != nil {
		RecordError(span, err, name + " failed")
	}
	return err
}

// SpanFromContext retrieves the current span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ContextWithSpan returns a new context with the given span
func ContextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return trace.ContextWithSpan(ctx, span)
}

// IsEnabled returns true if tracing is enabled
func IsEnabled() bool {
	return Tracer() != noopTracer
}

// SetGlobalTracerProvider sets the global tracer provider
// This should be called once at application startup
func SetGlobalTracerProvider(provider trace.TracerProvider) {
	otel.SetTracerProvider(provider)
	// Reset the tracer to pick up the new provider
	tracerOnce = sync.Once{}
}