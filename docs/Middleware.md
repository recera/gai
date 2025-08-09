# Middleware

GAI middlewares wrap a `ProviderClient` to add cross‑cutting behavior.

## Chain

```go
prov := middleware.Chain(base,
  middleware.Logger(),
  middleware.Defaults(func(p *gai.LLMCallParts){ if p.MaxTokens==0 { p.MaxTokens = 400 } }),
  middleware.SimulatedStreaming(30*time.Millisecond, 80),
)
```

## Defaults
Applies defaults to `LLMCallParts` when unset.

## Logger
Logs start/end of calls. Optionally accepts a redactor to scrub values.

```go
prov := middleware.Chain(base,
  middleware.Logger(func(s string) string { return "[redacted]" }),
)
```

## SimulatedStreaming
Emulates streaming by chunking blocking responses.

## ReasoningExtraction
Extracts `<think>...</think>` blocks and strips them from user‑visible output. Pass a callback to capture reasoning.

## Tracer
Records LLM spans (generate/stream) and tool call events via the observability API. Use with the `otel` build tag and `observability.Enable` to export spans.

