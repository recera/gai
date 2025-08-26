// Package obs provides observability features for the GAI framework.
// It includes OpenTelemetry-based tracing, metrics, and usage accounting
// with zero overhead when observability is not configured.
//
// The package supports both custom GAI attributes and OpenTelemetry GenAI
// semantic conventions for interoperability with observability platforms
// like Braintrust, Langfuse, and others.
package obs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/recera/gai/core"
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

// ContentCaptureMode determines how to capture LLM input/output content
type ContentCaptureMode string

const (
	// ContentCaptureAttributes captures content as span attributes (default, compatible with Braintrust)
	ContentCaptureAttributes ContentCaptureMode = "attributes"
	// ContentCaptureEvents captures content as OpenTelemetry events
	ContentCaptureEvents ContentCaptureMode = "events"
	// ContentCaptureBoth captures content as both attributes and events
	ContentCaptureBoth ContentCaptureMode = "both"
	// ContentCaptureNone disables content capture (privacy mode)
	ContentCaptureNone ContentCaptureMode = "none"
)

// GenAI operation names following OpenTelemetry semantic conventions
const (
	GenAIOperationChat           = "chat"
	GenAIOperationChatCompletion = "chat_completion"
	GenAIOperationTextCompletion = "text_completion"
	GenAIOperationCompletion     = "completion"
	GenAIOperationGenerate       = "generate"
	GenAIOperationEmbedding      = "embedding"
)

// Provider name mapping to GenAI system identifiers
var providerSystemMap = map[string]string{
	"openai":      "openai",
	"anthropic":   "anthropic",
	"gemini":      "google",
	"google":      "google",
	"groq":        "groq",
	"ollama":      "ollama",
	"cohere":      "cohere",
	"huggingface": "huggingface",
	// OpenAI-compatible providers
	"xai":      "xai",
	"baseten":  "baseten",
	"cerebras": "cerebras",
}

// RequestSpanOptions contains options for creating a request span
type RequestSpanOptions struct {
	// Core request attributes (backward compatible)
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

	// GenAI semantic conventions support
	Operation      string             // GenAI operation name (e.g., "chat_completion")
	Messages       []core.Message     // Messages for automatic content capture
	ContentCapture ContentCaptureMode // How to capture input/output content
	GenAISystem    string             // Override for gen_ai.system (defaults to provider mapping)
	SpanNameFormat string             // Override span naming format

	// Advanced options
	ConversationID string // Unique conversation identifier
	UserID         string // User identifier for multi-tenant systems
}

// GenAIRequestSpanOptions contains options specifically for GenAI-compliant request spans
type GenAIRequestSpanOptions struct {
	// Required GenAI fields
	System    string // gen_ai.system (e.g., "openai", "groq")
	Model     string // gen_ai.request.model
	Operation string // gen_ai.operation.name

	// Optional GenAI fields
	Messages    []core.Message // For automatic prompt/completion capture
	Temperature *float32       // gen_ai.request.temperature
	MaxTokens   *int           // gen_ai.request.max_tokens
	TopP        *float32       // gen_ai.request.top_p
	TopK        *int           // gen_ai.request.top_k
	Tools       []string       // Tool names available

	// Content capture options
	ContentCapture ContentCaptureMode // How to capture content

	// Additional metadata
	ConversationID string         // gen_ai.conversation.id
	UserID         string         // User identifier
	Metadata       map[string]any // Additional custom attributes
}

// StartRequestSpan starts a new span for an AI request with automatic GenAI semantic conventions support
func StartRequestSpan(ctx context.Context, opts RequestSpanOptions) (context.Context, trace.Span) {
	// Determine span name - use GenAI convention if operation is specified
	spanName := "ai.request" // Default backward-compatible name
	if opts.Operation != "" && opts.Model != "" {
		if opts.SpanNameFormat != "" {
			spanName = fmt.Sprintf(opts.SpanNameFormat, opts.Operation, opts.Model)
		} else {
			spanName = fmt.Sprintf("%s %s", opts.Operation, opts.Model) // GenAI convention
		}
	}

	// Create span with basic attributes
	ctx, span := Tracer().Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			// Legacy GAI attributes (backward compatibility)
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

	// Add GenAI semantic conventions if operation is specified
	if opts.Operation != "" {
		system := opts.GenAISystem
		if system == "" {
			// Map provider to GenAI system identifier
			if mapped, ok := providerSystemMap[strings.ToLower(opts.Provider)]; ok {
				system = mapped
			} else {
				system = strings.ToLower(opts.Provider)
			}
		}

		span.SetAttributes(
			attribute.String("gen_ai.system", system),
			attribute.String("gen_ai.operation.name", opts.Operation),
			attribute.String("gen_ai.request.model", opts.Model),
		)

		// Add optional GenAI attributes
		if opts.Temperature > 0 {
			span.SetAttributes(attribute.Float64("gen_ai.request.temperature", float64(opts.Temperature)))
		}
		if opts.MaxTokens > 0 {
			span.SetAttributes(attribute.Int("gen_ai.request.max_tokens", opts.MaxTokens))
		}
		if opts.ConversationID != "" {
			span.SetAttributes(attribute.String("gen_ai.conversation.id", opts.ConversationID))
		}
		if opts.ToolCount > 0 {
			span.SetAttributes(attribute.StringSlice("gen_ai.tools", extractToolNames(opts.ToolCount)))
		}
	}

	// Capture message content if provided
	if len(opts.Messages) > 0 && opts.ContentCapture != ContentCaptureNone {
		captureMessageContent(span, opts.Messages, opts.ContentCapture, opts.Provider)
	}

	// Add provider-specific options as attributes
	for k, v := range opts.ProviderOptions {
		span.SetAttributes(attribute.String(fmt.Sprintf("llm.provider.%s", k), fmt.Sprint(v)))
	}

	// Add metadata as attributes
	for k, v := range opts.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata.%s", k), fmt.Sprint(v)))
	}

	// Add user context if provided
	if opts.UserID != "" {
		span.SetAttributes(attribute.String("user.id", opts.UserID))
	}

	return ctx, span
}

// StartGenAISpan starts a new span with pure GenAI semantic conventions
func StartGenAISpan(ctx context.Context, opts GenAIRequestSpanOptions) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("%s %s", opts.Operation, opts.Model)

	ctx, span := Tracer().Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("gen_ai.system", opts.System),
			attribute.String("gen_ai.operation.name", opts.Operation),
			attribute.String("gen_ai.request.model", opts.Model),
		),
	)

	// Add optional request parameters
	if opts.Temperature != nil {
		span.SetAttributes(attribute.Float64("gen_ai.request.temperature", float64(*opts.Temperature)))
	}
	if opts.MaxTokens != nil {
		span.SetAttributes(attribute.Int("gen_ai.request.max_tokens", *opts.MaxTokens))
	}
	if opts.TopP != nil {
		span.SetAttributes(attribute.Float64("gen_ai.request.top_p", float64(*opts.TopP)))
	}
	if opts.TopK != nil {
		span.SetAttributes(attribute.Int("gen_ai.request.top_k", *opts.TopK))
	}
	if len(opts.Tools) > 0 {
		span.SetAttributes(attribute.StringSlice("gen_ai.tools", opts.Tools))
	}
	if opts.ConversationID != "" {
		span.SetAttributes(attribute.String("gen_ai.conversation.id", opts.ConversationID))
	}
	if opts.UserID != "" {
		span.SetAttributes(attribute.String("user.id", opts.UserID))
	}

	// Capture message content if provided
	if len(opts.Messages) > 0 && opts.ContentCapture != ContentCaptureNone {
		captureMessageContent(span, opts.Messages, opts.ContentCapture, opts.System)
	}

	// Add custom metadata
	for k, v := range opts.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata.%s", k), fmt.Sprint(v)))
	}

	return ctx, span
}

// captureMessageContent captures message content as attributes and/or events
func captureMessageContent(span trace.Span, messages []core.Message, mode ContentCaptureMode, system string) {
	if span == nil || !span.IsRecording() {
		return
	}

	// Capture as attributes (Braintrust-compatible)
	if mode == ContentCaptureAttributes || mode == ContentCaptureBoth {
		captureMessageAttributes(span, messages)
	}

	// Capture as OpenTelemetry events
	if mode == ContentCaptureEvents || mode == ContentCaptureBoth {
		captureMessageEvents(span, messages, system)
	}
}

// captureMessageAttributes captures messages as span attributes
func captureMessageAttributes(span trace.Span, messages []core.Message) {
	// Support both individual attributes and JSON format
	for i, msg := range messages {
		span.SetAttributes(
			attribute.String(fmt.Sprintf("gen_ai.prompt.%d.role", i), string(msg.Role)),
		)

		// Extract text content from message parts
		content := extractTextContent(msg)
		if content != "" {
			span.SetAttributes(
				attribute.String(fmt.Sprintf("gen_ai.prompt.%d.content", i), content),
			)
		}
	}

	// Also provide JSON format for compatibility
	if jsonMessages, err := json.Marshal(convertMessagesToJSON(messages)); err == nil {
		span.SetAttributes(attribute.String("gen_ai.prompt_json", string(jsonMessages)))
	}
}

// captureMessageEvents captures messages as OpenTelemetry events
func captureMessageEvents(span trace.Span, messages []core.Message, system string) {
	for _, msg := range messages {
		var eventName string
		switch msg.Role {
		case core.System:
			eventName = "gen_ai.system.message"
		case core.User:
			eventName = "gen_ai.user.message"
		case core.Assistant:
			eventName = "gen_ai.assistant.message"
		case core.Tool:
			eventName = "gen_ai.tool.message"
		default:
			continue // Skip unknown roles
		}

		content := extractTextContent(msg)
		if content != "" {
			span.AddEvent(eventName, trace.WithAttributes(
				attribute.String("gen_ai.system", system),
				attribute.String("role", string(msg.Role)),
				attribute.String("content", content),
			))
		}
	}
}

// extractTextContent extracts text content from message parts
func extractTextContent(msg core.Message) string {
	var content strings.Builder
	for _, part := range msg.Parts {
		if text, ok := part.(core.Text); ok {
			if content.Len() > 0 {
				content.WriteString(" ")
			}
			content.WriteString(text.Text)
		}
	}
	return content.String()
}

// convertMessagesToJSON converts messages to JSON format for gen_ai.prompt_json
func convertMessagesToJSON(messages []core.Message) []map[string]string {
	result := make([]map[string]string, len(messages))
	for i, msg := range messages {
		result[i] = map[string]string{
			"role":    string(msg.Role),
			"content": extractTextContent(msg),
		}
	}
	return result
}

// extractToolNames creates a placeholder tool names list (to be enhanced by providers)
func extractToolNames(toolCount int) []string {
	if toolCount <= 0 {
		return nil
	}
	tools := make([]string, toolCount)
	for i := 0; i < toolCount; i++ {
		tools[i] = fmt.Sprintf("tool_%d", i+1)
	}
	return tools
}

// RecordGenAICompletion records the completion content and usage with GenAI semantic conventions
func RecordGenAICompletion(span trace.Span, text string, inputTokens, outputTokens, totalTokens int) {
	if span != nil && span.IsRecording() {
		// GenAI semantic conventions for completion and usage
		span.SetAttributes(
			attribute.String("gen_ai.completion", text),
			attribute.Int("gen_ai.usage.prompt_tokens", inputTokens),
			attribute.Int("gen_ai.usage.completion_tokens", outputTokens),
			attribute.Int("gen_ai.usage.total_tokens", totalTokens),
		)

		// Legacy attributes for backward compatibility
		RecordUsage(span, inputTokens, outputTokens, totalTokens)
	}
}

// RecordGenAICompletionEvent records a completion as an OpenTelemetry event
func RecordGenAICompletionEvent(span trace.Span, text string, system string) {
	if span != nil && span.IsRecording() && text != "" {
		span.AddEvent("gen_ai.choice", trace.WithAttributes(
			attribute.String("gen_ai.system", system),
			attribute.Int("index", 0),
			attribute.String("finish_reason", "stop"),
			attribute.String("content", text),
		))
	}
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
	ToolName   string
	ToolID     string
	InputSize  int
	StepNumber int
	Parallel   bool
	RetryCount int
	Timeout    time.Duration
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
	Provider   string
	Model      string
	EventCount int
	BytesCount int
	Duration   time.Duration
	EventType  string
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
// Supports both legacy and GenAI streaming metrics
func RecordStreamingMetrics(span trace.Span, eventCount, bytesCount int, duration time.Duration) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(
			// Legacy streaming attributes (backward compatibility)
			attribute.Int("streaming.event_count", eventCount),
			attribute.Int("streaming.bytes_count", bytesCount),
			attribute.Float64("streaming.duration_ms", float64(duration.Milliseconds())),
			// GenAI streaming attributes  
			attribute.Int("gen_ai.stream.total_chunks", eventCount),
			attribute.Int("gen_ai.stream.total_bytes", bytesCount),
			attribute.Float64("gen_ai.stream.duration_ms", float64(duration.Milliseconds())),
		)
		
		// Add bytes per second if duration > 0
		if duration.Seconds() > 0 {
			span.SetAttributes(
				attribute.Float64("gen_ai.stream.bytes_per_second", float64(bytesCount)/duration.Seconds()),
			)
		}
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

// RecordToolContent records tool input and output content for Braintrust display
func RecordToolContent(span trace.Span, toolName string, input json.RawMessage, output interface{}, err error) {
	if span == nil || !span.IsRecording() {
		return
	}

	// Set tool input content - use braintrust namespace for better compatibility
	if len(input) > 0 {
		span.SetAttributes(
			attribute.String("braintrust.input_json", string(input)),
			attribute.String("gen_ai.prompt", string(input)), // Also set GenAI format
		)
	}

	// Set tool output content
	if err != nil {
		errorOutput := map[string]interface{}{
			"error": err.Error(),
			"tool":  toolName,
		}
		if outputJSON, marshalErr := json.Marshal(errorOutput); marshalErr == nil {
			span.SetAttributes(
				attribute.String("braintrust.output_json", string(outputJSON)),
				attribute.String("gen_ai.completion", string(outputJSON)),
			)
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else if output != nil {
		if outputJSON, marshalErr := json.Marshal(output); marshalErr == nil {
			span.SetAttributes(
				attribute.String("braintrust.output_json", string(outputJSON)),
				attribute.String("gen_ai.completion", string(outputJSON)),
			)
		}
		span.SetStatus(codes.Ok, "Tool executed successfully")
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
		RecordError(span, err, name+" failed")
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

// BraintrustOptimizedSpanOptions returns span options optimized for Braintrust integration
// Following the investigation findings, this ensures maximum compatibility with Braintrust's
// automatic span recognition and content display
func BraintrustOptimizedSpanOptions(provider, model string) RequestSpanOptions {
	// Map provider to GenAI system identifier
	system := provider
	if mapped, ok := providerSystemMap[strings.ToLower(provider)]; ok {
		system = mapped
	}

	return RequestSpanOptions{
		Provider:       provider,
		Model:          model,
		Operation:      GenAIOperationChatCompletion, // Standard operation for Braintrust
		GenAISystem:    system,                       // Correct system mapping
		ContentCapture: ContentCaptureAttributes,     // Attributes work best with Braintrust
		SpanNameFormat: "%s %s",                      // GenAI convention: "{operation} {model}"
	}
}

// BraintrustGenAISpanOptions returns pure GenAI span options optimized for Braintrust
func BraintrustGenAISpanOptions(system, model string, messages []core.Message) GenAIRequestSpanOptions {
	return GenAIRequestSpanOptions{
		System:         system,
		Model:          model,
		Operation:      GenAIOperationChatCompletion,
		Messages:       messages,
		ContentCapture: ContentCaptureAttributes, // Braintrust prefers attributes over events
	}
}

// StartBraintrustSpan starts a span optimized for Braintrust integration
// This is a convenience function that applies all the findings from the investigation
func StartBraintrustSpan(ctx context.Context, provider, model string, messages []core.Message) (context.Context, trace.Span) {
	// Map provider to GenAI system
	system := provider
	if mapped, ok := providerSystemMap[strings.ToLower(provider)]; ok {
		system = mapped
	}

	opts := BraintrustGenAISpanOptions(system, model, messages)
	return StartGenAISpan(ctx, opts)
}

// RecordBraintrustCompletion records completion in the format most compatible with Braintrust
func RecordBraintrustCompletion(span trace.Span, result *core.TextResult, system string) {
	if span == nil || !span.IsRecording() || result == nil {
		return
	}

	// Record as GenAI completion (primary method for Braintrust)
	if result.Text != "" {
		span.SetAttributes(attribute.String("gen_ai.completion", result.Text))

		// Also add as completion event for comprehensive capture
		span.AddEvent("gen_ai.choice", trace.WithAttributes(
			attribute.String("gen_ai.system", system),
			attribute.Int("index", 0),
			attribute.String("finish_reason", "stop"),
			attribute.String("content", result.Text),
		))
	}

	// Record usage following GenAI semantic conventions
	if result.Usage.TotalTokens > 0 {
		span.SetAttributes(
			attribute.Int("gen_ai.usage.prompt_tokens", result.Usage.InputTokens),
			attribute.Int("gen_ai.usage.completion_tokens", result.Usage.OutputTokens),
			attribute.Int("gen_ai.usage.total_tokens", result.Usage.TotalTokens),
		)
	}

	// Determine finish reason from steps if available
	if len(result.Steps) > 0 {
		lastStep := result.Steps[len(result.Steps)-1]
		finishReason := "stop"
		if len(lastStep.ToolCalls) > 0 {
			finishReason = "tool_calls"
		}
		span.SetAttributes(attribute.String("gen_ai.completion.finish_reason", finishReason))
	}
}

// ConfigureBraintrustSpan configures an existing span for optimal Braintrust integration
// This applies all the critical fixes identified in the investigation
func ConfigureBraintrustSpan(span trace.Span, provider, model, operation string, messages []core.Message) {
	if span == nil || !span.IsRecording() {
		return
	}

	// Map provider to GenAI system identifier
	system := provider
	if mapped, ok := providerSystemMap[strings.ToLower(provider)]; ok {
		system = mapped
	}

	// Set GenAI span name (Critical Fix #3)
	if operation == "" {
		operation = GenAIOperationChatCompletion
	}
	spanName := fmt.Sprintf("%s %s", operation, model)
	span.SetName(spanName)

	// Set core GenAI attributes (Critical Fix #1 & #2)
	span.SetAttributes(
		attribute.String("gen_ai.system", system), // Correct system mapping
		attribute.String("gen_ai.operation.name", operation),
		attribute.String("gen_ai.request.model", model),
	)

	// Capture message content as attributes (Critical Fix #1)
	if len(messages) > 0 {
		captureMessageAttributes(span, messages)
	}
}

// GenAIOperation represents the type of GenAI operation being performed
type GenAIOperation struct {
	Name        string // Operation name for gen_ai.operation.name
	Description string // Human-readable description
}

// Standard GenAI operations for different provider functions
var (
	GenAIOpChatCompletion         = GenAIOperation{"chat_completion", "Chat completion with messages"}
	GenAIOpTextCompletion         = GenAIOperation{"text_completion", "Single text completion"}
	GenAIOpObjectCompletion       = GenAIOperation{"object_completion", "Structured object generation"}
	GenAIOpJSONCompletion         = GenAIOperation{"json_completion", "JSON object generation"}
	GenAIOpStreamCompletion       = GenAIOperation{"stream_completion", "Streaming text completion"}
	GenAIOpStreamObjectCompletion = GenAIOperation{"stream_object_completion", "Streaming object completion"}
	GenAIOpEmbedding              = GenAIOperation{"embedding", "Text embedding generation"}
	GenAIOpImageGeneration        = GenAIOperation{"image_generation", "Image generation"}
	GenAIOpAudioGeneration        = GenAIOperation{"audio_generation", "Audio generation"}
	GenAIOpSpeechToText           = GenAIOperation{"speech_to_text", "Speech recognition"}
	GenAIOpTextToSpeech           = GenAIOperation{"text_to_speech", "Text to speech synthesis"}
)

// GenAIExecutionFunc represents a function that performs the actual AI operation
type GenAIExecutionFunc func(context.Context) (*core.TextResult, error)

// GenAIStreamExecutionFunc represents a function that performs streaming AI operations
type GenAIStreamExecutionFunc func(context.Context) (interface{}, error)

// WithGenAIObservability wraps an AI operation with automatic GenAI observability
// This is the main function providers should use for consistent observability
func WithGenAIObservability(ctx context.Context, provider, model string, operation GenAIOperation, request core.Request, fn GenAIExecutionFunc) (*core.TextResult, error) {
	// Skip if observability is disabled (zero overhead)
	if !IsEnabled() {
		return fn(ctx)
	}

	// Map provider to GenAI system
	system := provider
	if mapped, ok := providerSystemMap[strings.ToLower(provider)]; ok {
		system = mapped
	}

	// Create GenAI span
	opts := GenAIRequestSpanOptions{
		System:         system,
		Model:          model,
		Operation:      operation.Name,
		Messages:       request.Messages,
		ContentCapture: ContentCaptureAttributes, // Optimized for compatibility
	}

	// Add request parameters
	if request.Temperature > 0 {
		temp := request.Temperature
		opts.Temperature = &temp
	}
	if request.MaxTokens > 0 {
		maxTokens := request.MaxTokens
		opts.MaxTokens = &maxTokens
	}

	// Add tools if present
	if len(request.Tools) > 0 {
		toolNames := make([]string, len(request.Tools))
		for i, tool := range request.Tools {
			toolNames[i] = tool.Name()
		}
		opts.Tools = toolNames
	}

	// Start span
	ctx, span := StartGenAISpan(ctx, opts)
	defer span.End()

	// Execute the operation
	result, err := fn(ctx)

	// Record error if occurred
	if err != nil {
		RecordError(span, err, fmt.Sprintf("%s failed", operation.Description))
		return nil, err
	}

	// Record successful completion
	if result != nil {
		RecordBraintrustCompletion(span, result, system)
	}

	return result, nil
}

// WithGenAIStreamingObservability wraps streaming AI operations with observability
func WithGenAIStreamingObservability(ctx context.Context, provider, model string, operation GenAIOperation, request core.Request, fn GenAIStreamExecutionFunc) (interface{}, error) {
	// Skip if observability is disabled
	if !IsEnabled() {
		return fn(ctx)
	}

	// Map provider to GenAI system
	system := provider
	if mapped, ok := providerSystemMap[strings.ToLower(provider)]; ok {
		system = mapped
	}

	// Create GenAI span for streaming
	opts := GenAIRequestSpanOptions{
		System:         system,
		Model:          model,
		Operation:      operation.Name,
		Messages:       request.Messages,
		ContentCapture: ContentCaptureAttributes,
	}

	// Add streaming-specific metadata
	opts.Metadata = map[string]any{
		"streaming":      true,
		"operation_type": "streaming",
	}

	ctx, span := StartGenAISpan(ctx, opts)
	defer span.End()

	// Record start of streaming
	span.AddEvent("gen_ai.stream.start", trace.WithAttributes(
		attribute.String("gen_ai.system", system),
		attribute.String("stream.type", operation.Name),
	))

	// Execute streaming operation
	result, err := fn(ctx)

	if err != nil {
		RecordError(span, err, fmt.Sprintf("Streaming %s failed", operation.Description))
		span.AddEvent("gen_ai.stream.error", trace.WithAttributes(
			attribute.String("error", err.Error()),
		))
		return nil, err
	}

	// Record successful stream completion
	span.AddEvent("gen_ai.stream.complete", trace.WithAttributes(
		attribute.String("gen_ai.system", system),
	))

	return result, nil
}

// RecordStreamingChunk records a chunk of streaming data
func RecordStreamingChunk(span trace.Span, chunkText string, chunkIndex int, system string) {
	if span == nil || !span.IsRecording() {
		return
	}

	span.AddEvent("gen_ai.stream.chunk", trace.WithAttributes(
		attribute.String("gen_ai.system", system),
		attribute.Int("chunk.index", chunkIndex),
		attribute.String("chunk.content", chunkText),
		attribute.Int("chunk.length", len(chunkText)),
	))
}


// GetProviderSystem maps a provider name to its GenAI system identifier
func GetProviderSystem(provider string) string {
	if mapped, ok := providerSystemMap[strings.ToLower(provider)]; ok {
		return mapped
	}
	return strings.ToLower(provider)
}

// CreateGenAISpanForProvider creates a provider-specific GenAI span
func CreateGenAISpanForProvider(ctx context.Context, provider, model string, operation GenAIOperation, request core.Request) (context.Context, trace.Span) {
	if !IsEnabled() {
		return ctx, trace.SpanFromContext(ctx) // Return noop span
	}

	system := GetProviderSystem(provider)

	opts := GenAIRequestSpanOptions{
		System:         system,
		Model:          model,
		Operation:      operation.Name,
		Messages:       request.Messages,
		ContentCapture: ContentCaptureAttributes,
	}

	return StartGenAISpan(ctx, opts)
}

// GenAIObservabilityConfig allows providers to customize observability behavior
type GenAIObservabilityConfig struct {
	Provider         string
	DefaultOperation GenAIOperation
	ContentCapture   ContentCaptureMode
	CustomAttributes map[string]any
	EnableStreaming  bool
}

// WithCustomGenAIObservability provides advanced customization for providers
func WithCustomGenAIObservability(ctx context.Context, config GenAIObservabilityConfig, model string, operation GenAIOperation, request core.Request, fn GenAIExecutionFunc) (*core.TextResult, error) {
	if !IsEnabled() {
		return fn(ctx)
	}

	system := GetProviderSystem(config.Provider)

	opts := GenAIRequestSpanOptions{
		System:         system,
		Model:          model,
		Operation:      operation.Name,
		Messages:       request.Messages,
		ContentCapture: config.ContentCapture,
		Metadata:       config.CustomAttributes,
	}

	ctx, span := StartGenAISpan(ctx, opts)
	defer span.End()

	// Add custom provider attributes
	for k, v := range config.CustomAttributes {
		span.SetAttributes(attribute.String(fmt.Sprintf("provider.%s", k), fmt.Sprint(v)))
	}

	result, err := fn(ctx)

	if err != nil {
		RecordError(span, err, operation.Description+" failed")
		return nil, err
	}

	if result != nil {
		RecordBraintrustCompletion(span, result, system)
	}

	return result, nil
}
