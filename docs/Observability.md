# Observability (OpenTelemetry)

By default, observability is a no‑op. Enable a real OTLP exporter under the `otel` build tag.

## Enable

```go
//go:build otel

shutdown, _ := observability.Enable(ctx, observability.TracerConfig{
  Endpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
  Headers:  map[string]string{"Authorization": os.Getenv("OTEL_EXPORTER_OTLP_HEADERS")},
  SampleRatio: 1.0, TruncateLimit: 2048,
})
defer shutdown(ctx)

prov := middleware.Chain(baseProvider, middleware.NewTracer(2048).Wrap())
```

## Attributes

Spans include attributes like:
- ai.provider, ai.model, ai.session_id
- ai.settings.temperature, ai.settings.max_tokens
- ai.prompt.last_user (truncated/redacted)
- ai.usage.prompt_tokens/completion_tokens/total_tokens
- ai.finish_reason

Events:
- ai.first_chunk (streaming)
- ai.toolCall (name, arguments)

## Vendors
- Braintrust: point OTLP endpoint/headers to Braintrust collector
- Arize: point OTLP endpoint/headers to Arize collector (see their docs)

## Logging and Redaction

The example `Logger` middleware prints basic start/end messages. Pass a redaction function to scrub sensitive values:

```go
prov := middleware.Chain(base,
  middleware.Logger(func(s string) string { return strings.Repeat("*", len(s)) }),
)
```

