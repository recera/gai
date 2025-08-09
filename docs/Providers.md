# Providers

This page details how GAI maps requests and features for each provider.

## OpenAI

- Endpoints: Chat Completions (blocking & SSE streaming)
- Tools: `tools` + `tool_choice` mapped; native tool result messages `{role:"tool", tool_call_id, content}`
- Strict object mode: `response_format: {type: "json_schema"}` engaged via `GenerateObject[T]`
- Streaming: SSE parser emits `content` deltas; tool call arguments are coalesced by index and emitted at `finish_reason == "tool_calls"`. Optionally include token usage on the end chunk by enabling `WithOpenAIIncludeUsageInStream(true)`.

Settings honored: temperature, max_tokens, stop, top_p, seed, headers, provider opts.

## Anthropic

- Endpoints: Messages API (blocking & SSE streaming)
- Tools: `tools` + `tool_choice`; tool_use events (streaming) → `tool_call`; tool_result content block generated when replying to a tool call
- Strict object mode: use tolerant parser fallback (strict modes may vary across models)
- Streaming: emits text deltas via `content_block_delta` and `tool_use` events → `StreamChunk{tool_call}`

## Google Gemini

- Endpoints: Generate Content (blocking; streaming emulated)
- Tools: `tools.functionDeclarations` + functionCall mapping to `ToolCalls`; provider‑native tool results via functionResponse (set `Message.ToolName` in replies)
- Strict object mode: `response_mime_type: application/json` + `response_schema` set when provided in `ProviderOpts["response_schema"]`

## Groq & Cerebras

- Compatible with OpenAI‑like chat shapes; streaming may be emulated (use SimulatedStreaming middleware) depending on model capabilities.

## General notes

- Unsupported options are ignored; use `ProviderOpts` for provider‑specific features
- Use `Headers` to set gateway headers. You can also override base URLs per provider using `WithProviderBaseURL` in the client options. A shared `http.Client` with timeout is used for all providers; configure via `WithHTTPTimeout`.
- For request examples, see `_examples/` and the tests in `providers/*_test.go`
