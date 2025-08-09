# Tools & Structured Outputs

## Tools

Define tools with JSON Schemas and let models call them.

```go
// Schema from type
tool, _ := gai.ToolFromType[struct{ TZ string `json:"tz"` }]("get_time")
parts.WithTools(tool).WithSystem("Use tools when needed")

resp, err := client.RunWithTools(ctx, parts.Value(), func(call gai.ToolCall) (string, error) {
  if call.Name == "get_time" { return time.Now().Format(time.RFC3339), nil }
  return "", fmt.Errorf("unknown tool")
})
```

For streaming, use `StreamWithTools`.

### Provider mapping
- OpenAI: `tools` + `tool_choice`; replies with `{role:"tool", tool_call_id, content}`
- Anthropic: `tools`; streaming `tool_use` → `tool_call`; replies via `tool_result` content block referencing `tool_use_id`
- Gemini: `tools.functionDeclarations`; functionCall → `ToolCalls`; reply via functionResponse (set `Message.ToolName`)

## Structured outputs

Prefer strict modes when available, fallback to a tolerant parser.

```go
type City struct { Name string `json:"name"`; Pop int `json:"pop"` }
city, usage, err := gai.GenerateObject[City](ctx, client, parts.Value())
```

- OpenAI: `response_format: json_schema` engaged automatically
- Gemini: set `parts.ProviderOpts["response_schema"] = <json schema>`

