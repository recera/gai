# GAI: Go AI SDK

GAI is a production‑grade, provider‑agnostic Go SDK for building LLM‑powered applications.

It emphasizes three pillars:

- Developer Experience: fluent API, strong types, helpful defaults, clear docs
- Functionality: multi‑provider, tool calling (blocking/streaming), structured outputs, streaming + UI adapters, middleware
- Observability/Evals: optional OpenTelemetry spans, evaluation recorder and dataset builder

Deep dives live in the docs folder:

- docs/GettingStarted.md
- docs/Providers.md
- docs/Tools_and_StructuredOutputs.md
- docs/Streaming_and_UI.md
- docs/Middleware.md
- docs/Observability.md
- docs/Evaluation.md
- docs/Registry.md
- docs/Troubleshooting.md

---

## Install

```bash
go get github.com/recera/gai
```

Supported Go: 1.21+

---

## Quick Start

```go
package main

import (
  "context"
  "fmt"
  "log"
  "github.com/recera/gai"
)

func main() {
  // Optionally load .env (OPENAI_API_KEY, ANTHROPIC_API_KEY, ...)
  gai.FindAndLoadEnv()

  client, err := gai.NewClient()
  if err != nil { log.Fatal(err) }

  parts := gai.NewLLMCallParts().
    WithProvider("openai").
    WithModel("gpt-4o-mini").
    WithSystem("You are concise.").
    WithUserMessage("Explain Goroutines briefly.")

  resp, err := client.GetCompletion(context.Background(), parts.Value())
  if err != nil { log.Fatal(err) }
  fmt.Println(resp.Content)
}
```

Environment variables: `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`, `GROQ_API_KEY`, `CEREBRAS_API_KEY`.

See also: docs/GettingStarted.md

---

## Configuration & Client Options

```go
client, err := gai.NewClient(
  gai.WithHTTPTimeout(60*time.Second),
  gai.WithMaxRetries(3),
  gai.WithBackoff(200*time.Millisecond, 5*time.Second, 0.2), // jittered exponential backoff
  gai.WithOpenAIKey(os.Getenv("OPENAI_API_KEY")),
  // Defaults are UNSET unless you set them:
  gai.WithDefaultProvider("openai"),     // default for NewLLMCallParts
  gai.WithDefaultModel("gpt-4o-mini"),   // default for NewLLMCallParts
  // Observability/infra niceties
  gai.WithUserAgent("myapp/1.0 (+example.com)"),
  gai.WithProviderBaseURL("openai", os.Getenv("OPENAI_BASE_URL")), // e.g., gateway
  gai.WithOpenAIIncludeUsageInStream(true),
  // Tools loop cap (blocking/streaming)
  gai.WithToolLoopMaxSteps(8),
  // gai.WithEnvFile(".env"),            // load variables from a specific .env
  // gai.WithoutEnvFile(),                // disable .env loading
)
```

- Defaults set via the client are applied by `NewLLMCallParts()`. If you do not set defaults, you must set `Provider` and `Model` explicitly on each call.

---

## LLMCallParts (requests)

`LLMCallParts` is a fluent builder holding provider/model, system/user messages, and cross‑provider settings.

```go
parts := gai.NewLLMCallParts().
  WithProvider("anthropic").
  WithModel("claude-3-haiku-20240307").
  WithSystem("Be helpful.").
  WithUserMessage("Give me 3 tips to learn Go")
```

Advanced fields you can set on the value or struct:

- StopSequences `[]string`
- TopP `*float64`, TopK `*int`, Seed `*int64`
- Headers `map[string]string` (e.g., gateway headers)
- ProviderOpts `map[string]any` (escape hatch for provider‑specific options)
- ToolChoice `any` (e.g., "auto" or provider‑specific structure)
- SessionID `string` and Metadata `map[string]any` (for tracing/evals)
- ExpectedText `string`, ExpectedJSON `any` (for evals)

See docs/GettingStarted.md

---

## Providers

- OpenAI: native tools, strict object mode (json_schema), SSE streaming, arguments coalescer
- Anthropic: native tools, content‑block streaming, tool_use/tool_result mapping
- Gemini: functionDeclarations/calls/responses; request‑only strict schema hints
- Groq/Cerebras: OpenAI‑compatible chat shapes; streaming emulated (see middleware)

See docs/Providers.md for details on request/stream mappings and caveats.

---

## Structured Outputs (object mode)

Get typed JSON deterministically when supported, fallback to a robust tolerant parser otherwise.

```go
type City struct { Name string `json:"name"`; Pop int `json:"pop"` }
city, usage, err := gai.GenerateObject[City](ctx, client, parts.Value())
if err != nil { /* handle */ }
fmt.Println(city.Name, usage.TotalTokens)
```

- OpenAI: strict json_schema mode used transparently
- Gemini: pass `parts.ProviderOpts["response_schema"]` (provider consumes request hint)

See docs/Tools_and_StructuredOutputs.md

---

## Tool Calling

### Blocking tool loop

```go
// Build tool schema from a type (or a struct value)
tool, _ := gai.ToolFromType[struct{ TZ string `json:"tz"` }]("get_time")
parts.WithTools(tool).WithSystem("Use tools when needed")

resp, err := client.RunWithTools(ctx, parts.Value(), func(call gai.ToolCall) (string, error) {
  if call.Name == "get_time" { return time.Now().Format(time.RFC3339), nil }
  return "", fmt.Errorf("unknown tool")
})
```

### Streaming tools loop

```go
_ = client.StreamWithTools(ctx, parts.Value(), executorFn, func(ch gai.StreamChunk) error {
  switch ch.Type {
  case "content": fmt.Print(ch.Delta)
  case "tool_call": // visualise call
  case "end": fmt.Println("\n[done]", ch.FinishReason)
  }
  return nil
})
```

Provider‑native wiring:
- OpenAI: replies with `{role:"tool", tool_call_id, content}`
- Anthropic: replies with `tool_result` content block that references `tool_use_id`
- Gemini: replies with functionResponse (set `Message.ToolName`)

See docs/Tools_and_StructuredOutputs.md

---

## Streaming & UI (SSE)

Use the tiny adapter to expose an SSE endpoint consumable by modern UI hooks.

```go
http.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
  ch := make(chan gai.StreamChunk)
  go func(){
    defer close(ch)
    _ = client.StreamCompletion(ctx, parts.Value(), func(s gai.StreamChunk) error { ch <- s; return nil })
  }()
  uistream.Write(w, ch) // sets Content-Type: text/event-stream and x-vercel-ai-ui-message-stream: v1
})
```

Notes:
- Providers that support it may include token usage on the final end chunk. Enable for OpenAI with `WithOpenAIIncludeUsageInStream(true)`.
- React example (Vercel AI SDK `useChat`) is in docs/Streaming_and_UI.md

---

## Middleware

Chain cross‑cutting behavior around providers.

```go
prov := middleware.Chain(base,
  middleware.Logger(),
  middleware.Defaults(func(p *gai.LLMCallParts){ if p.MaxTokens==0 { p.MaxTokens = 400 } }),
  middleware.SimulatedStreaming(30*time.Millisecond, 80),
)
```

Shipped:
- Defaults: apply defaults to `LLMCallParts`
- Logger: log start/end
- SimulatedStreaming: emulate streaming by chunking blocking responses
- ReasoningExtraction: extract `<think>...</think>` and strip it from visible output
- Tracer: LLM spans and tool events via the observability API

See docs/Middleware.md

---

## Observability (OpenTelemetry, optional)

By default, no‑ops. With the `otel` build tag and an OTLP endpoint, spans are emitted for generate/stream calls and tool events.

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

Attributes include ai.provider, ai.model, ai.session_id, ai.settings.*, ai.first_chunk event, ai.toolCall events, ai.usage.*, ai.finish_reason.

See docs/Observability.md

---

## Evaluation & Datasets

Record interactions (NDJSON) and build datasets for external evals.

```go
rec, _ := eval.NewRecorder("eval_logs.ndjson")
prov := rec.Wrap(baseProvider) // wrap provider

_ = eval.BuildDataset("eval_logs.ndjson", "dataset.json", func(m map[string]any) *eval.Entry {
  return &eval.Entry{
    Provider: m["provider"].(string),
    Model:    m["model"].(string),
    Messages: nil, // or transform
    Response: m["response"].(string),
    Expected: m["expected_text"],
    Metadata: nil,
  }
})
```

See docs/Evaluation.md

---

## Model Registry

Resolve model keys like `provider:model` and apply them rapidly.

```go
parts, _ := gai.NewLLMCallPartsFor(client, "openai:gpt-4o-mini")
```

See docs/Registry.md

---

## Error Handling

Errors from providers come as `*gai.LLMError`:

```go
resp, err := client.GetCompletion(ctx, parts.Value())
if err != nil {
  if llmErr, ok := err.(*gai.LLMError); ok {
    fmt.Println("provider:", llmErr.Provider, "model:", llmErr.Model, "status:", llmErr.StatusCode)
    fmt.Println("raw:", llmErr.LastRaw)
    fmt.Println("request_id:", llmErr.RequestID)
    fmt.Println("ratelimit limit/remaining/reset:", llmErr.RateLimitLimit, llmErr.RateLimitRemaining, llmErr.RateLimitReset)
  }
}
```

Retries & backoff:
- Configure with `WithMaxRetries` and `WithBackoff`. The client retries on 429/5xx and transport errors, honoring `Retry-After` when available.

---

## Examples

All `_examples/` are tagged with `//go:build examples` to avoid multiple mains.

Run:

```bash
go run -tags examples _examples/06_streaming.go
```

---

## Troubleshooting

See docs/Troubleshooting.md for the most common issues and fixes.

---

## Contributing

PRs welcome. Please open an issue for larger features.

---

## License

MIT — see LICENSE
