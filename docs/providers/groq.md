# Groq Provider Guide

The Groq provider enables ultra-fast AI inference through Groq's custom Language Processing Units (LPUs), delivering exceptional performance for text generation, tool calling, and streaming applications.

## Quick Start

```go
import "github.com/recera/gai/providers/groq"

provider := groq.New(
    groq.WithAPIKey("your-groq-api-key"),
    groq.WithModel("llama-3.1-8b-instant"),
)
```

## API Key Setup

Get your API key from [Groq Console](https://console.groq.com/keys):

```bash
export GROQ_API_KEY="gsk_..."
```

## Model Recommendations

### Production Models (Recommended)

| Model | Performance Class | Best For | Context Window | Tool Support |
|-------|-------------------|----------|----------------|--------------|
| `llama-3.1-8b-instant` | Ultra-Fast | General chat, simple tools | 131k | ✅ |
| `llama-3.3-70b-versatile` | Fast | Complex reasoning, analysis | 131k | ⚠️ Limited |
| `moonshotai/kimi-k2-instruct` | Ultra-Fast | Multi-step workflows, agents | 131k | ✅ |
| `deepseek-r1-distill-llama-70b` | Balanced | Mathematics, complex problems | 131k | ✅ |

### Specialized Models

| Model | Use Case | Performance |
|-------|----------|-------------|
| `meta-llama/llama-4-scout-17b-16e-instruct` | Vision & multimodal | Fast |
| `whisper-large-v3-turbo` | Speech-to-text | Ultra-Fast |
| `meta-llama/llama-guard-4-12b` | Content moderation | Fast |
| `compound-beta-mini` | Experimental inference | Ultra-Fast |

## Configuration Options

```go
provider := groq.New(
    groq.WithAPIKey("your-api-key"),
    groq.WithModel("llama-3.1-8b-instant"),
    groq.WithServiceTier("on_demand"),        // "on_demand" or "batch"
    groq.WithMaxRetries(3),
    groq.WithRetryDelay(500*time.Millisecond),
    groq.WithCustomHeaders(map[string]string{
        "X-Custom-Header": "value",
    }),
)
```

## Tool Calling

Groq supports function calling with proper tool_call_id handling:

```go
// Define your tool
weatherTool := tools.New[WeatherInput, WeatherOutput](
    "get_weather",
    "Get current weather for a location",
    func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
        // Your implementation
        return getWeatherData(input.Location)
    },
)

// Use with multi-step execution
request := core.Request{
    Messages: []core.Message{{
        Role: core.User,
        Parts: []core.Part{
            core.Text{Text: "What's the weather in Tokyo?"},
        },
    }},
    Tools:    tools.ToCoreHandles([]tools.Handle{weatherTool}),
    StopWhen: core.NoMoreTools(), // Continue until no more tool calls
}

result, err := provider.GenerateText(ctx, request)
```

### Tool Calling Best Practices

1. **Use Ultra-Fast Models**: `llama-3.1-8b-instant` and `moonshotai/kimi-k2-instruct` excel at tool calling
2. **Proper StopWhen Conditions**: Use `core.NoMoreTools()` or `core.MaxSteps(5)` to control execution
3. **Error Handling**: Tools that fail will be reported in the conversation flow
4. **Parallel Execution**: Ultra-fast models support parallel tool calls automatically

## Streaming

Real-time streaming with tool execution:

```go
request := core.Request{
    Messages: messages,
    Tools:    tools,
    Stream:   true,
}

stream, err := provider.StreamText(ctx, request)
defer stream.Close()

for event := range stream.Events() {
    switch event.Type {
    case core.EventStart:
        fmt.Print("Starting...")
    case core.EventTextDelta:
        fmt.Print(event.TextDelta)
    case core.EventToolCall:
        fmt.Printf("\\nCalling tool: %s", event.ToolName)
    case core.EventToolResult:
        fmt.Print("Tool completed")
    case core.EventFinish:
        fmt.Println("\\nDone")
    }
}
```

## Performance Characteristics

### Latency by Performance Class

- **Ultra-Fast** (< 1s): `llama-3.1-8b-instant`, `moonshotai/kimi-k2-instruct`
- **Fast** (1-3s): `llama-3.3-70b-versatile`, `meta-llama/llama-4-*`
- **Balanced** (3-5s): `deepseek-r1-distill-llama-70b`

### Context Windows

- **Long Context** (131k): Most modern models support up to 131,072 tokens
- **Legacy Models** (8k): `llama3-*-8192` models (deprecated)
- **Specialized** (448): Whisper models for audio transcription

## Error Handling

Common errors and solutions:

```go
result, err := provider.GenerateText(ctx, request)
if err != nil {
    var aiErr *core.AIError
    if errors.As(err, &aiErr) {
        switch aiErr.Code {
        case core.ErrorRateLimited:
            // Wait and retry, check rate limits
        case core.ErrorContextLengthExceeded:
            // Reduce input length or use larger context model
        case core.ErrorUnauthorized:
            // Check API key
        }
    }
}
```

### Rate Limits

- **Free Tier**: 30 requests/minute, 6,000 tokens/minute
- **Paid Tiers**: Higher limits based on usage tier
- **Best Practice**: Implement exponential backoff with the built-in retry logic

## Multi-Step Workflows

```go
// Complex agent workflow
request := core.Request{
    Messages: conversationHistory,
    Tools: []core.ToolHandle{
        weatherTool,
        calendarTool,
        emailTool,
    },
    StopWhen: core.CombineConditions(
        core.MaxSteps(10),
        core.UntilToolSeen("send_email"),
    ),
}

result, err := provider.GenerateText(ctx, request)

// Access the full execution trace
for i, step := range result.Steps {
    fmt.Printf("Step %d: %d tool calls", i+1, len(step.ToolCalls))
    if step.Text != "" {
        fmt.Printf(", text: %s", step.Text)
    }
    fmt.Println()
}
```

## Model-Specific Notes

### Kimi-K2 (Moonshot AI)
- **Strengths**: Excellent tool calling, fast inference, agent workflows
- **Context**: 131k tokens, 16k max completion
- **Best For**: Multi-step reasoning, tool orchestration

### Llama-3.1-8B-Instant
- **Strengths**: Blazingly fast (< 1s), reliable tool calls
- **Context**: 131k tokens, full context completion
- **Best For**: Chat applications, simple tools, high-throughput

### Llama-3.3-70B-Versatile
- **Limitations**: Some tool calling constraints in current version  
- **Strengths**: Strong reasoning when tools work
- **Best For**: Complex analysis without tools, or simple tool scenarios

### DeepSeek R1 Distill
- **Strengths**: Mathematics, complex problem solving
- **Performance**: More thoughtful but slower responses
- **Best For**: Research, analysis, mathematical reasoning

### Vision Models (Llama-4 Series)
- **Supports**: Image analysis, multimodal tasks
- **Performance**: Fast inference with vision capabilities
- **Best For**: Image analysis, document understanding, visual Q&A

## Integration Examples

### Basic Chat Bot
```go
provider := groq.New(
    groq.WithModel("llama-3.1-8b-instant"),
    groq.WithAPIKey(os.Getenv("GROQ_API_KEY")),
)
```

### Agent with Tools
```go
provider := groq.New(
    groq.WithModel("moonshotai/kimi-k2-instruct"),
    groq.WithAPIKey(os.Getenv("GROQ_API_KEY")),
    groq.WithMaxRetries(2),
)
```

### Production High-Volume
```go
provider := groq.New(
    groq.WithModel("llama-3.1-8b-instant"),
    groq.WithServiceTier("on_demand"),
    groq.WithMaxRetries(1), // Fast failure for high volume
    groq.WithRetryDelay(100*time.Millisecond),
)
```

## Troubleshooting

### Common Issues

1. **Tool Calls Not Working**
   - Ensure you're using a tool-compatible model
   - Check that tools are properly defined with JSON schema
   - Use recommended models: `llama-3.1-8b-instant`, `moonshotai/kimi-k2-instruct`

2. **Rate Limit Errors**
   - Implement proper backoff (built into provider)
   - Monitor your usage in Groq Console
   - Consider upgrading your tier for higher limits

3. **Context Length Exceeded**
   - Use models with larger context windows (131k token models)
   - Implement conversation summarization for long chats
   - Trim older messages from conversation history

4. **Slow Performance**
   - Use ultra-fast models for speed-critical applications
   - Enable streaming for better perceived performance
   - Consider parallel tool calls for complex workflows

### Debug Mode

Enable debug logging to see request/response details:

```go
provider := groq.New(
    groq.WithAPIKey("your-key"),
    groq.WithCustomHeaders(map[string]string{
        "X-Debug": "true",
    }),
)
```

## Advanced Configuration

### Custom Service Tiers
```go
// For batch processing
groq.WithServiceTier("batch")

// For real-time (default)
groq.WithServiceTier("on_demand")
```

### Provider-Specific Options
```go
request := core.Request{
    // ... other fields
    ProviderOptions: map[string]interface{}{
        "groq": map[string]interface{}{
            "top_p":             0.9,
            "frequency_penalty": 0.1,
            "presence_penalty":  0.1,
            "stop":              []string{"Human:", "AI:"},
            "seed":              42,
        },
    },
}
```

## Best Practices Summary

1. **Model Selection**: Use `llama-3.1-8b-instant` for speed, `moonshotai/kimi-k2-instruct` for tools
2. **Tool Design**: Keep tools simple and focused, use proper JSON schemas
3. **Error Handling**: Implement proper retry logic and error classification
4. **Performance**: Use streaming for better UX, enable parallel tool calls
5. **Monitoring**: Track token usage and response times
6. **Testing**: Test with multiple models to find optimal performance

The Groq provider delivers exceptional performance for AI applications requiring ultra-fast inference and reliable tool execution.