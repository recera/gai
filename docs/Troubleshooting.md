# Troubleshooting

## I get no streaming output
- Ensure your provider supports SSE (OpenAI, Anthropic). For others, wrap with `SimulatedStreaming`.
- Check network/proxy issues; SSE requires long‑lived HTTP connections.
- For OpenAI token usage on the final chunk, enable `WithOpenAIIncludeUsageInStream(true)`.

## Tool calling does nothing
- Confirm `parts.WithTools(...)` was called and the system prompt instructs the model to use tools.
- Inspect `resp.ToolCalls` (blocking) or stream `tool_call` chunks.

## Structured outputs parse errors
- Prefer strict object mode when supported; otherwise the tolerant parser will recover in many cases.
- Consider lowering temperature for strict JSON.

## Telemetry shows nothing
- You must build with the `otel` tag and call `observability.Enable(...)` with a valid OTLP endpoint/headers. Without that, spans are no‑ops.

## Provider returned 401/403
- Check API keys and any required headers in `parts.Headers`. Gateways often require additional headers.
- You can override base URLs with `WithProviderBaseURL(provider, url)` to target gateways.
- Increase HTTP timeout via `WithHTTPTimeout` if your gateway is slow.

## Model key resolution fails
- Ensure the key is `provider:model`. The default registry registers: openai, anthropic, gemini, groq, cerebras.

## Getting rate limited (429) or transient errors (5xx)
- The client retries automatically with exponential backoff and jitter (configure with `WithMaxRetries` and `WithBackoff`).
- Honor provider `Retry-After` headers where present.

