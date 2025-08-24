# OpenAI Provider for GAI Framework

A production-grade OpenAI provider implementation for the GAI framework, supporting Chat Completions, streaming, structured outputs, and tool calling with the latest OpenAI API features.

## Features

- ✅ **Chat Completions API** - Full support for GPT-4, GPT-4 Turbo, GPT-4o, and GPT-3.5 models
- ✅ **Streaming** - Real-time streaming with Server-Sent Events (SSE)
- ✅ **Structured Outputs** - Type-safe JSON generation with schema validation
- ✅ **Tool Calling** - Parallel tool execution with multi-step support
- ✅ **Multimodal** - Support for text and image inputs
- ✅ **Retry Logic** - Automatic retry with exponential backoff
- ✅ **Error Handling** - Comprehensive error classification and recovery
- ✅ **Observability** - Built-in metrics and tracing support

## Installation

```go
import "github.com/recera/gai/providers/openai"
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "github.com/recera/gai/core"
    "github.com/recera/gai/providers/openai"
)

func main() {
    // Create provider
    provider := openai.New(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4o-mini"),
    )
    
    // Generate text
    result, err := provider.GenerateText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Hello, how are you?"},
                },
            },
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(result.Text)
    fmt.Printf("Tokens used: %d\n", result.Usage.TotalTokens)
}
```

## Configuration

### Provider Options

```go
provider := openai.New(
    // Required
    openai.WithAPIKey("your-api-key"),
    
    // Optional
    openai.WithModel("gpt-4o"),                      // Default model
    openai.WithBaseURL("https://api.openai.com/v1"), // Custom endpoint
    openai.WithOrganization("org-id"),               // Organization ID
    openai.WithProject("project-id"),                // Project ID
    openai.WithMaxRetries(3),                        // Retry attempts
    openai.WithRetryDelay(100*time.Millisecond),     // Base retry delay
    openai.WithHTTPClient(customClient),             // Custom HTTP client
    openai.WithMetricsCollector(collector),          // Observability
)
```

### Request Options

```go
result, err := provider.GenerateText(ctx, core.Request{
    Model:       "gpt-4o",        // Override default model
    Temperature: floatPtr(0.7),   // Control randomness (0-2)
    MaxTokens:   intPtr(1000),    // Maximum response length
    
    // OpenAI-specific options
    ProviderOptions: map[string]interface{}{
        "openai": map[string]interface{}{
            "presence_penalty":  0.5,
            "frequency_penalty": 0.5,
            "top_p":            0.9,
            "seed":             42,
            "user":             "user-123",
            "stop":             []string{"\n\n"},
        },
    },
})
```

## Usage Examples

### Basic Text Generation

```go
result, err := provider.GenerateText(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.System,
            Parts: []core.Part{
                core.Text{Text: "You are a helpful assistant."},
            },
        },
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Explain quantum computing in simple terms."},
            },
        },
    },
    Temperature: floatPtr(0.7),
    MaxTokens:   intPtr(500),
})
```

### Streaming Responses

```go
stream, err := provider.StreamText(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Write a short story about a robot."},
            },
        },
    },
    Stream: true,
})
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

// Process events
for event := range stream.Events() {
    switch event.Type {
    case core.EventTextDelta:
        fmt.Print(event.TextDelta)
    case core.EventFinish:
        fmt.Printf("\n\nTotal tokens: %d\n", event.Usage.TotalTokens)
    case core.EventError:
        log.Printf("Stream error: %v", event.Err)
    }
}
```

### Multimodal Input

```go
result, err := provider.GenerateText(ctx, core.Request{
    Model: "gpt-4o", // Vision-capable model
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "What's in this image?"},
                core.ImageURL{
                    URL:    "https://example.com/image.jpg",
                    Detail: "high", // "low", "high", or "auto"
                },
            },
        },
    },
})
```

### Tool Calling

```go
// Define a tool
type WeatherInput struct {
    Location string `json:"location" jsonschema:"description=City name"`
}

type WeatherOutput struct {
    Temperature float64 `json:"temperature"`
    Conditions  string  `json:"conditions"`
}

weatherTool := tools.New[WeatherInput, WeatherOutput](
    "get_weather",
    "Get current weather for a location",
    func(ctx context.Context, in WeatherInput, meta tools.Meta) (WeatherOutput, error) {
        // Implementation
        return WeatherOutput{
            Temperature: 72.5,
            Conditions:  "Sunny",
        }, nil
    },
)

// Use with provider
result, err := provider.GenerateText(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "What's the weather in San Francisco?"},
            },
        },
    },
    Tools:      []core.ToolHandle{tools.NewCoreAdapter(weatherTool)},
    ToolChoice: core.ToolAuto, // ToolNone, ToolRequired, or ToolSpecific
})

// The provider automatically executes tools and includes results
fmt.Println(result.Text) // "The weather in San Francisco is sunny with a temperature of 72.5°F."
```

### Structured Outputs

```go
type Recipe struct {
    Name        string   `json:"name"`
    Ingredients []string `json:"ingredients"`
    Steps       []string `json:"steps"`
    PrepTime    int      `json:"prep_time_minutes"`
}

schema := map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "name":        map[string]interface{}{"type": "string"},
        "ingredients": map[string]interface{}{
            "type":  "array",
            "items": map[string]interface{}{"type": "string"},
        },
        "steps": map[string]interface{}{
            "type":  "array",
            "items": map[string]interface{}{"type": "string"},
        },
        "prep_time_minutes": map[string]interface{}{"type": "integer"},
    },
    "required": []string{"name", "ingredients", "steps", "prep_time_minutes"},
}

result, err := provider.GenerateObject(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Create a recipe for chocolate chip cookies."},
            },
        },
    },
}, schema)

if err != nil {
    log.Fatal(err)
}

// Type assert and use the result
if recipe, ok := result.Value.(map[string]interface{}); ok {
    fmt.Printf("Recipe: %s\n", recipe["name"])
    fmt.Printf("Prep time: %v minutes\n", recipe["prep_time_minutes"])
}
```

### Streaming Structured Outputs

```go
stream, err := provider.StreamObject(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Analyze the sentiment of: 'I love this product!'"},
            },
        },
    },
    Stream: true,
}, schema)

if err != nil {
    log.Fatal(err)
}
defer stream.Close()

// Process streaming events
for event := range stream.Events() {
    switch event.Type {
    case core.EventTextDelta:
        // Show partial JSON as it streams
        fmt.Print(event.TextDelta)
    }
}

// Get final validated object
finalObj, err := stream.Final()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Final object: %+v\n", *finalObj)
```

## Error Handling

The provider returns typed errors that can be inspected:

```go
result, err := provider.GenerateText(ctx, request)
if err != nil {
    if aiErr, ok := err.(*core.AIError); ok {
        switch aiErr.Category {
        case core.ErrorCategoryRateLimit:
            // Handle rate limiting
            time.Sleep(time.Duration(aiErr.RetryAfter) * time.Second)
        case core.ErrorCategoryAuth:
            // Handle authentication errors
            log.Fatal("Invalid API key")
        case core.ErrorCategoryInvalidRequest:
            // Handle bad requests
            log.Printf("Invalid request: %s", aiErr.Message)
        default:
            // Handle other errors
            log.Printf("Error: %s", aiErr.Error())
        }
        
        // Check if retryable
        if aiErr.Retryable {
            // Retry logic
        }
    }
}
```

## Supported Models

### GPT-4 Series
- `gpt-4` - Original GPT-4
- `gpt-4-turbo` - GPT-4 Turbo with vision
- `gpt-4o` - GPT-4 Optimized
- `gpt-4o-mini` - Smaller, faster GPT-4 variant

### GPT-3.5 Series
- `gpt-3.5-turbo` - Fast, efficient model
- `gpt-3.5-turbo-16k` - Extended context window

## Performance

Benchmark results on M1 MacBook Pro:

| Operation | Time | Memory | Allocations |
|-----------|------|--------|-------------|
| Provider Creation | 2.5μs | 1,856 B | 15 allocs |
| GenerateText | 703μs | 11,432 B | 146 allocs |
| Parallel Requests | 144μs | 11,432 B | 146 allocs |
| Message Conversion | 423ns | 624 B | 9 allocs |
| Tool Execution | 1.9μs | 1,056 B | 22 allocs |
| Stream Processing | 6.5μs | 2,736 B | 52 allocs |

## Testing

### Unit Tests
```bash
go test ./providers/openai
```

### Integration Tests (requires API key)
```bash
OPENAI_API_KEY=your-key go test ./providers/openai -tags=integration
```

### Benchmarks
```bash
go test ./providers/openai -bench=. -benchmem
```

## Advanced Features

### Custom HTTP Client

```go
client := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        // Add proxy, TLS config, etc.
    },
}

provider := openai.New(
    openai.WithAPIKey(apiKey),
    openai.WithHTTPClient(client),
)
```

### Observability Integration

```go
// With OpenTelemetry
collector := obs.NewIntegratedCollector()

provider := openai.New(
    openai.WithAPIKey(apiKey),
    openai.WithMetricsCollector(collector),
)

// Metrics are automatically recorded:
// - Request duration
// - Token usage
// - Error rates
// - Tool execution metrics
```

### Rate Limit Handling

```go
for attempts := 0; attempts < 3; attempts++ {
    result, err := provider.GenerateText(ctx, request)
    if err != nil {
        if aiErr, ok := err.(*core.AIError); ok {
            if aiErr.Category == core.ErrorCategoryRateLimit {
                // Exponential backoff
                delay := time.Duration(math.Pow(2, float64(attempts))) * time.Second
                if aiErr.RetryAfter > 0 {
                    delay = time.Duration(aiErr.RetryAfter) * time.Second
                }
                time.Sleep(delay)
                continue
            }
        }
        return nil, err
    }
    return result, nil
}
```

## Limitations

- Audio transcription/generation not yet supported (use Whisper/TTS providers)
- File uploads require conversion to base64 images
- Assistants API not implemented (use chat completions)
- Fine-tuned models work but may have different capabilities

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## License

Apache-2.0 - See [LICENSE](../../LICENSE) for details.