# Model Registry

The client embeds a simple provider registry. Resolve `provider:model` keys and set parts quickly.

```go
parts, _ := gai.NewLLMCallPartsFor(client, "openai:gpt-4o-mini")
```

Under the hood the client registers "openai", "anthropic", "gemini", "groq", and "cerebras" providers for lookup.
You can also set `WithDefaultProvider` and `WithDefaultModel` to pre‑populate new call parts. By default, these are unset.

