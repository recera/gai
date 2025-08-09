//go:build otel

package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/recera/gai/core"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlphttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer     trace.Tracer = otel.Tracer("github.com/recera/gai")
	currentCfg TracerConfig
	tracerProv *sdktrace.TracerProvider
)

// Enable configures an OTLP HTTP exporter and global tracer provider.
func Enable(ctx context.Context, cfg TracerConfig) (func(context.Context) error, error) {
	currentCfg = cfg
	// Configure exporter
	opts := []otlphttp.Option{}
	if cfg.Endpoint != "" {
		opts = append(opts, otlphttp.WithEndpoint(cfg.Endpoint))
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlphttp.WithHeaders(cfg.Headers))
	}
	exp, err := otlphttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("otlp http exporter: %w", err)
	}

	// Sampler
	ratio := cfg.SampleRatio
	if ratio <= 0 {
		ratio = 1.0
	}
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))

	// Resource (basic)
	res, _ := resource.Merge(resource.Default(), resource.NewSchemaless())

	tracerProv = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProv)
	tracer = otel.Tracer("github.com/recera/gai")

	shutdown := func(ctx context.Context) error { return tracerProv.Shutdown(ctx) }
	return shutdown, nil
}

// StartLLMSpan starts a span and annotates with standard attributes.
func StartLLMSpan(ctx context.Context, operation string, parts core.LLMCallParts) (context.Context, LLMSpan) {
	ctx, sp := tracer.Start(ctx, "ai."+operation)
	// Attributes
	attrs := []attribute.KeyValue{
		attribute.String("ai.provider", parts.Provider),
		attribute.String("ai.model", parts.Model),
	}
	if parts.SessionID != "" {
		attrs = append(attrs, attribute.String("ai.session_id", parts.SessionID))
	}
	if parts.Temperature != 0 {
		attrs = append(attrs, attribute.Float64("ai.settings.temperature", parts.Temperature))
	}
	if parts.MaxTokens != 0 {
		attrs = append(attrs, attribute.Int("ai.settings.max_tokens", parts.MaxTokens))
	}
	// Attach a truncated prompt transcript as a hint (optional)
	if len(parts.Messages) > 0 {
		// naive: join last user content
		var last string
		for i := len(parts.Messages) - 1; i >= 0; i-- {
			if parts.Messages[i].Role == "user" {
				last = parts.Messages[i].GetTextContent()
				break
			}
		}
		if last != "" {
			tr := currentCfg.TruncateLimit
			if tr == 0 {
				tr = 2048
			}
			if currentCfg.Redact != nil {
				last = currentCfg.Redact(last)
			}
			last = SafeTruncate(last, tr)
			attrs = append(attrs, attribute.String("ai.prompt.last_user", last))
		}
	}
	sp.SetAttributes(attrs...)
	return ctx, sp
}

func MarkFirstChunkLLM(ctx context.Context) {
	if sp := trace.SpanFromContext(ctx); sp != nil {
		sp.AddEvent("ai.first_chunk", trace.WithTimestamp(time.Now()))
	}
}

func AddEventToolCall(ctx context.Context, name, arguments string) {
	if currentCfg.Redact != nil {
		arguments = currentCfg.Redact(arguments)
	}
	tr := currentCfg.TruncateLimit
	if tr == 0 {
		tr = 2048
	}
	arguments = SafeTruncate(arguments, tr)
	if sp := trace.SpanFromContext(ctx); sp != nil {
		sp.AddEvent("ai.toolCall", trace.WithAttributes(
			attribute.String("ai.tool.name", name),
			attribute.String("ai.tool.arguments", arguments),
		))
	}
}

func EndLLMSpan(span LLMSpan, finishReason string, usage *core.TokenUsage, err error) {
	if sp, ok := span.(trace.Span); ok {
		if finishReason != "" {
			sp.SetAttributes(attribute.String("ai.finish_reason", finishReason))
		}
		if usage != nil {
			sp.SetAttributes(
				attribute.Int("ai.usage.prompt_tokens", usage.PromptTokens),
				attribute.Int("ai.usage.completion_tokens", usage.CompletionTokens),
				attribute.Int("ai.usage.total_tokens", usage.TotalTokens),
			)
		}
		if err != nil {
			sp.RecordError(err)
		}
		sp.End()
		return
	}
	// fallback
	if span != nil {
		span.End()
	}
}
