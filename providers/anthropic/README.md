# Anthropic Provider for GAI Framework

A comprehensive, production-ready Anthropic Claude provider for the GAI (Go AI) framework. This provider implements full support for Anthropic's Messages API, including text generation, streaming, structured outputs, and tool calling.

## Features

### Core Capabilities
- ✅ **Text Generation**: Support for all Claude models with conversation context
- ✅ **Streaming**: Real-time Server-Sent Events (SSE) streaming with fine-grained events
- ✅ **Structured Outputs**: JSON object generation with schema guidance
- ✅ **Tool Calling**: Multi-step tool execution with automatic result handling
- ✅ **System Prompts**: Proper handling of Anthropic's system prompt format
- ✅ **Multimodal**: Support for text and image inputs

### Production Features
- 🔄 **Automatic Retries**: Exponential backoff with jitter for transient failures
- 🛡️ **Error Mapping**: Comprehensive error taxonomy mapping to GAI framework standards
- 📊 **Observability**: Full metrics and tracing integration
- 🚀 **Performance**: Optimized for low latency and high throughput
- 🔒 **Security**: Secure API key handling and request validation

## Installation

```bash
go get github.com/recera/gai/providers/anthropic
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/recera/gai/core"
    "github.com/recera/gai/providers/anthropic"
)

func main() {
    // Create provider
    provider := anthropic.New(
        anthropic.WithAPIKey("your-api-key"),
        anthropic.WithModel("claude-sonnet-4-20250514"),
    )

    // Simple text generation
    req := core.Request{
        Messages: []core.Message{
            {
                Role:  core.User,
                Parts: []core.Part{core.Text{Text: "What is artificial intelligence?"}},
            },
        },
        MaxTokens: 500,
    }

    result, err := provider.GenerateText(context.Background(), req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Text)
    fmt.Printf("Tokens used: %d input, %d output\n", 
        result.Usage.InputTokens, result.Usage.OutputTokens)
}
```

## Configuration Options

### Provider Options

```go
provider := anthropic.New(
    // Required: API key for authentication
    anthropic.WithAPIKey("your-api-key"),
    
    // Optional: Default model (defaults to claude-sonnet-4-20250514)
    anthropic.WithModel("claude-3-opus-20240229"),
    
    // Optional: Custom base URL (for proxies or testing)
    anthropic.WithBaseURL("https://api.anthropic.com"),
    
    // Optional: API version (defaults to 2023-06-01)
    anthropic.WithVersion("2023-06-01"),
    
    // Optional: Custom HTTP client
    anthropic.WithHTTPClient(customClient),
    
    // Optional: Retry configuration
    anthropic.WithMaxRetries(5),
    anthropic.WithRetryDelay(200 * time.Millisecond),
    
    // Optional: Observability
    anthropic.WithMetricsCollector(collector),
)
```

### Supported Models

| Model | Model ID | Context | Use Case |
|-------|----------|---------|----------|
| Claude Sonnet 4 | `claude-sonnet-4-20250514` | 200K tokens | Latest, best performance |
| Claude Haiku 3.5 | `claude-3-5-haiku-20241022` | 200K tokens | Fast, efficient |
| Claude Sonnet 3.5 | `claude-3-5-sonnet-20241022` | 200K tokens | Balanced performance |
| Claude Opus 3 | `claude-3-opus-20240229` | 200K tokens | Highest capability |

## Advanced Usage

### System Prompts

Anthropic handles system prompts differently from other providers - they go in a separate `system` field:

```go
req := core.Request{
    Messages: []core.Message{
        {
            Role:  core.System,
            Parts: []core.Part{core.Text{Text: "You are a helpful coding assistant. Be concise and accurate."}},
        },
        {
            Role:  core.User,
            Parts: []core.Part{core.Text{Text: "How do I reverse a string in Go?"}},
        },
    },
    MaxTokens: 300,
}
```

### Streaming Text Generation

```go
req := core.Request{
    Messages: []core.Message{
        {
            Role:  core.User,
            Parts: []core.Part{core.Text{Text: "Write a short story about a robot."}},
        },
    },
    MaxTokens: 1000,
    Stream:    true,
}

stream, err := provider.StreamText(context.Background(), req)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for event := range stream.Events() {
    switch event.Type {
    case core.EventStart:
        fmt.Println("Stream started")
    case core.EventTextDelta:
        fmt.Print(event.TextDelta) // Print text as it arrives
    case core.EventFinish:
        fmt.Printf("\nTokens used: %d\n", event.Usage.TotalTokens)
    case core.EventError:
        fmt.Printf("Error: %v\n", event.Err)
    }
}
```

### Structured Object Generation

```go
type Person struct {
    Name        string `json:"name"`
    Age         int    `json:"age"`
    Occupation  string `json:"occupation"`
    Hobbies     []string `json:"hobbies"`
}

schema := map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "name":       map[string]string{"type": "string"},
        "age":        map[string]string{"type": "integer"},
        "occupation": map[string]string{"type": "string"},
        "hobbies": map[string]interface{}{
            "type": "array",
            "items": map[string]string{"type": "string"},
        },
    },
    "required": []string{"name", "age", "occupation"},
}

req := core.Request{
    Messages: []core.Message{
        {
            Role:  core.User,
            Parts: []core.Part{core.Text{Text: "Generate a fictional person profile."}},
        },
    },
    MaxTokens: 200,
}

result, err := provider.GenerateObject(context.Background(), req, schema)
if err != nil {
    log.Fatal(err)
}

// Parse the result
var person Person
if err := json.Unmarshal(json.Marshal(result.Value), &person); err != nil {
    log.Fatal(err)
}

fmt.Printf("Generated person: %+v\n", person)
```

### Tool Calling

```go
import "github.com/recera/gai/tools"

// Define a calculator tool
calcTool := tools.New("calculator", "Performs arithmetic calculations",
    func(ctx context.Context, input struct {
        Expression string `json:"expression" description:"Mathematical expression to evaluate"`
    }, meta tools.Meta) (map[string]interface{}, error) {
        // Simple calculator logic here
        result := evaluateExpression(input.Expression)
        return map[string]interface{}{
            "result": result,
            "expression": input.Expression,
        }, nil
    })

req := core.Request{
    Messages: []core.Message{
        {
            Role:  core.User,
            Parts: []core.Part{core.Text{Text: "What's 15 * 23 + 47?"}},
        },
    },
    Tools:     []core.ToolHandle{calcTool},
    MaxTokens: 300,
    StopWhen:  core.NoMoreTools(), // Stop when no more tools are needed
}

result, err := provider.GenerateText(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Final answer:", result.Text)

// Examine the steps
for i, step := range result.Steps {
    fmt.Printf("Step %d: %s\n", i+1, step.Text)
    for _, call := range step.ToolCalls {
        fmt.Printf("  Called tool: %s\n", call.Name)
    }
    for _, result := range step.ToolResults {
        fmt.Printf("  Tool result: %v\n", result.Result)
    }
}
```

### Multimodal Inputs

```go
req := core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "What do you see in this image?"},
                core.ImageURL{
                    URL:    "data:image/jpeg;base64,/9j/4AAQSkZJRg...", // Base64 encoded image
                    Detail: "high", // "low", "high", or "auto"
                },
            },
        },
    },
    MaxTokens: 500,
}

result, err := provider.GenerateText(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

fmt.Println(result.Text)
```

## Error Handling

The provider maps all Anthropic API errors to the GAI framework's stable error taxonomy:

```go
result, err := provider.GenerateText(context.Background(), req)
if err != nil {
    // Check error type using GAI framework helpers
    switch {
    case core.IsAuth(err):
        fmt.Println("Authentication error - check your API key")
    case core.IsRateLimited(err):
        fmt.Printf("Rate limited - retry after %v\n", core.GetRetryAfter(err))
    case core.IsContextSizeExceeded(err):
        fmt.Println("Context too long - reduce input size")
    case core.IsSafetyBlocked(err):
        fmt.Println("Content blocked by safety filters")
    case core.IsTransient(err):
        fmt.Println("Temporary error - safe to retry")
    default:
        fmt.Printf("Other error: %v\n", err)
    }
}
```

### Anthropic-Specific Error Helpers

```go
import "github.com/recera/gai/providers/anthropic"

if anthropic.IsRetryable(err) {
    delay := anthropic.GetRetryDelay(err)
    time.Sleep(delay)
    // Retry the request
}

if anthropic.IsContextLengthExceeded(err) {
    fmt.Println("Request too long for Claude's context window")
}

if model := anthropic.ExtractModelFromError(err); model != "" {
    fmt.Printf("Error was for model: %s\n", model)
}
```

## Provider-Specific Options

You can pass Anthropic-specific options using the `ProviderOptions` field:

```go
req := core.Request{
    Messages: []core.Message{
        {
            Role:  core.User,
            Parts: []core.Part{core.Text{Text: "Write a poem."}},
        },
    },
    MaxTokens: 500,
    ProviderOptions: map[string]interface{}{
        "anthropic": map[string]interface{}{
            "top_p":          0.9,
            "top_k":          40,
            "stop_sequences": []string{"\n\n", "END"},
        },
    },
}
```

### Available Provider Options

- `top_p` (float): Nucleus sampling parameter (0.0 to 1.0)
- `top_k` (int): Top-k sampling parameter
- `stop_sequences` ([]string): Custom stop sequences

## Performance Considerations

### Best Practices

1. **Model Selection**: Use the fastest model that meets your quality needs
   - `claude-3-5-haiku-20241022` for simple tasks
   - `claude-sonnet-4-20250514` for complex reasoning

2. **Context Management**: Keep context length reasonable
   - Claude supports 200K tokens but shorter contexts are faster
   - Use conversation pruning for long interactions

3. **Concurrent Requests**: The provider is thread-safe and supports concurrency
   ```go
   var wg sync.WaitGroup
   for i := 0; i < 10; i++ {
       wg.Add(1)
       go func() {
           defer wg.Done()
           result, err := provider.GenerateText(ctx, req)
           // Handle result...
       }()
   }
   wg.Wait()
   ```

4. **Streaming**: Use streaming for long responses to improve perceived latency
   ```go
   stream, err := provider.StreamText(ctx, req)
   // Process events as they arrive
   ```

### Rate Limits

Anthropic has rate limits based on your usage tier:
- Free tier: 5 RPM, 25K TPM
- Pro tier: 50 RPM, 100K TPM
- Scale/Enterprise: Custom limits

The provider automatically handles rate limiting with exponential backoff.

## Testing

### Unit Tests
```bash
go test ./providers/anthropic
```

### Integration Tests (requires API key)
```bash
ANTHROPIC_API_KEY=your-key go test -tags=integration ./providers/anthropic
```

### Benchmarks
```bash
ANTHROPIC_API_KEY=your-key go test -bench=. ./providers/anthropic
```

## Contributing

1. Follow the existing code patterns
2. Add comprehensive tests for new features
3. Update documentation
4. Ensure error handling follows GAI framework patterns

## API Reference

### Types

- `Provider`: Main provider struct implementing `core.Provider`
- `Option`: Configuration option function
- Various internal types for API communication

### Methods

- `New(opts ...Option) *Provider`: Create new provider
- `GenerateText(ctx, req) (*core.TextResult, error)`: Generate text
- `StreamText(ctx, req) (core.TextStream, error)`: Stream text generation
- `GenerateObject(ctx, req, schema) (*core.ObjectResult[any], error)`: Generate structured object
- `StreamObject(ctx, req, schema) (core.ObjectStream[any], error)`: Stream object generation

### Configuration Options

- `WithAPIKey(key string)`: Set API key
- `WithModel(model string)`: Set default model
- `WithBaseURL(url string)`: Set custom base URL
- `WithVersion(version string)`: Set API version
- `WithHTTPClient(client *http.Client)`: Set custom HTTP client
- `WithMaxRetries(n int)`: Set retry count
- `WithRetryDelay(d time.Duration)`: Set retry delay
- `WithMetricsCollector(collector core.MetricsCollector)`: Set metrics collector

## License

This provider is part of the GAI framework and follows the same license terms.

## Support

- 📚 Documentation: [GAI Framework Docs](../../../docs/)
- 🐛 Issues: [GitHub Issues](https://github.com/recera/gai/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/recera/gai/discussions)