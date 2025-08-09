# Getting Started

This guide walks you through installing GAI, configuring providers, and making your first calls.

## Install

```bash
go get github.com/recera/gai
```

Optional: create a `.env` with your API keys:

```
OPENAI_API_KEY=...
ANTHROPIC_API_KEY=...
GEMINI_API_KEY=...
GROQ_API_KEY=...
CEREBRAS_API_KEY=...
```

Load env and create a client:

```go
client, err := gai.NewClient()
```

Or pass keys directly:

```go
client, err := gai.NewClient(
  gai.WithOpenAIKey(os.Getenv("OPENAI_API_KEY")),
  // Defaults are unset by default; set them for convenience:
  gai.WithDefaultProvider("openai"),
  gai.WithDefaultModel("gpt-4o-mini"),
  gai.WithHTTPTimeout(60*time.Second),
  gai.WithMaxRetries(3),
  gai.WithBackoff(200*time.Millisecond, 5*time.Second, 0.2),
)
```

## First completion

```go
parts := gai.NewLLMCallParts().
  WithProvider("openai").WithModel("gpt-4o-mini").
  WithSystem("You are helpful.").
  WithUserMessage("What is Goroutine?")
resp, err := client.GetCompletion(ctx, parts.Value())
fmt.Println(resp.Content)
```

## Streaming

```go
_ = client.StreamCompletion(ctx, parts.Value(), func(ch gai.StreamChunk) error {
  if ch.Type == "content" { fmt.Print(ch.Delta) }
  if ch.Type == "end" {
    fmt.Println("\n[done]", ch.FinishReason)
    if ch.Usage != nil { fmt.Printf("usage: %+v\n", *ch.Usage) }
  }
  return nil
})
```

## Typed object mode

```go
type City struct { Name string `json:"name"`; Pop int `json:"pop"` }
city, usage, err := gai.GenerateObject[City](ctx, client, parts.Value())
```

## Tools (blocking)

```go
tool, _ := gai.ToolFromType[struct{ TZ string `json:"tz"` }]("get_time")
parts.WithTools(tool).WithSystem("Use tools when needed")
resp, _ := client.RunWithTools(ctx, parts.Value(), func(call gai.ToolCall) (string, error) {
  if call.Name == "get_time" { return time.Now().Format(time.RFC3339), nil }
  return "", fmt.Errorf("unknown tool")
})
```

## Streaming + tools

```go
_ = client.StreamWithTools(ctx, parts.Value(), execFn, handler)
```

See Providers.md for provider‑specific details.
