# Gemini Provider for GAI Framework

The Gemini provider implements Google's Gemini AI models for the GAI framework, offering unique features including file uploads, safety configuration, citations, and comprehensive multimodal support.

## Features

- 🚀 **Full Gemini Model Support**: Access to Gemini 1.5 Flash, Pro, and Ultra models
- 📁 **File Upload API**: Automatic handling of large media files via Gemini's file API
- 🛡️ **Safety Configuration**: Fine-grained content safety controls with event emission
- 📚 **Citations Support**: Automatic citation extraction and streaming
- 🎯 **Structured Outputs**: JSON Schema-based response generation
- 🔧 **Tool Calling**: Function calling with parallel execution support
- 📺 **Streaming**: Real-time SSE streaming with safety and citation events
- 🎨 **Multimodal**: Support for text, images, audio, video, and documents
- ♻️ **Automatic Retries**: Exponential backoff with jitter for transient failures
- 📊 **Usage Tracking**: Token counting and cost estimation

## Installation

```bash
go get github.com/recera/gai/providers/gemini
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/recera/gai/core"
    "github.com/recera/gai/providers/gemini"
)

func main() {
    // Create provider
    provider := gemini.New(
        gemini.WithAPIKey("your-api-key"),
        gemini.WithModel("gemini-1.5-flash"),
    )
    
    // Generate text
    ctx := context.Background()
    result, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Hello, Gemini!"},
                },
            },
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(result.Text)
}
```

## Configuration

### Provider Options

```go
provider := gemini.New(
    // Required: API key for authentication
    gemini.WithAPIKey("your-api-key"),
    
    // Optional: Custom base URL for proxies or regional endpoints
    gemini.WithBaseURL("https://custom.endpoint.com"),
    
    // Optional: Default model (defaults to gemini-1.5-flash)
    gemini.WithModel("gemini-1.5-pro"),
    
    // Optional: Custom HTTP client
    gemini.WithHTTPClient(customClient),
    
    // Optional: Retry configuration
    gemini.WithMaxRetries(3),
    gemini.WithRetryDelay(time.Second),
    
    // Optional: Default safety settings
    gemini.WithDefaultSafety(&core.SafetyConfig{
        Harassment: core.SafetyBlockMost,
        Hate:       core.SafetyBlockMost,
        Sexual:     core.SafetyBlockMost,
        Dangerous:  core.SafetyBlockFew,
    }),
    
    // Optional: Metrics collector for observability
    gemini.WithMetricsCollector(collector),
)
```

## Unique Gemini Features

### 1. File Upload Support

The Gemini provider automatically handles large files by uploading them to Gemini's file API:

```go
// Files are automatically uploaded when using BlobBytes or BlobURL
result, err := provider.GenerateText(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Analyze this video"},
                core.Video{
                    Source: core.BlobRef{
                        Kind: core.BlobURL,
                        URL:  "https://example.com/video.mp4",
                        MIME: "video/mp4",
                    },
                },
            },
        },
    },
})
```

Supported file types:
- **Images**: JPEG, PNG, GIF, WebP
- **Videos**: MP4, AVI, MOV, WebM
- **Audio**: MP3, WAV, FLAC, AAC
- **Documents**: PDF, TXT, HTML, CSS, JS, MD, CSV

### 2. Safety Configuration and Events

Configure safety thresholds and receive real-time safety events:

```go
// Configure safety per request
result, err := provider.GenerateText(ctx, core.Request{
    Messages: messages,
    Safety: &core.SafetyConfig{
        Harassment: core.SafetyBlockFew,    // Block only high probability
        Hate:       core.SafetyBlockSome,   // Block medium and above
        Sexual:     core.SafetyBlockMost,   // Block low and above
        Dangerous:  core.SafetyBlockNone,   // Don't block
    },
})

// Or stream with safety events
stream, err := provider.StreamText(ctx, request)
for event := range stream.Events() {
    switch event.Type {
    case core.EventSafety:
        fmt.Printf("Safety: %s - %s (score: %.2f)\n",
            event.Safety.Category,
            event.Safety.Action,
            event.Safety.Score)
    case core.EventTextDelta:
        fmt.Print(event.TextDelta)
    }
}
```

Safety levels:
- `SafetyBlockNone`: Don't block any content
- `SafetyBlockFew`: Block only high probability harmful content
- `SafetyBlockSome`: Block medium and high probability
- `SafetyBlockMost`: Block low, medium, and high probability

### 3. Citations Support

Gemini can provide citations for grounded responses:

```go
stream, err := provider.StreamText(ctx, request)
for event := range stream.Events() {
    switch event.Type {
    case core.EventCitations:
        for _, citation := range event.Citations {
            fmt.Printf("Citation: %s (%d-%d) - %s\n",
                citation.Title,
                citation.Start,
                citation.End,
                citation.URI)
        }
    case core.EventTextDelta:
        fmt.Print(event.TextDelta)
    }
}
```

### 4. System Instructions

Gemini handles system instructions separately from the message history:

```go
result, err := provider.GenerateText(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.System,
            Parts: []core.Part{
                core.Text{Text: "You are a helpful coding assistant."},
            },
        },
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Write a hello world in Go"},
            },
        },
    },
})
```

## Streaming

Real-time streaming with comprehensive event support:

```go
stream, err := provider.StreamText(ctx, request)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for event := range stream.Events() {
    switch event.Type {
    case core.EventStart:
        fmt.Println("Stream started")
    
    case core.EventTextDelta:
        fmt.Print(event.TextDelta)
    
    case core.EventSafety:
        fmt.Printf("Safety event: %+v\n", event.Safety)
    
    case core.EventCitations:
        fmt.Printf("Citations: %+v\n", event.Citations)
    
    case core.EventToolCall:
        fmt.Printf("Calling tool: %s\n", event.ToolName)
    
    case core.EventFinish:
        fmt.Printf("\nTokens used: %d\n", event.Usage.TotalTokens)
    
    case core.EventError:
        fmt.Printf("Error: %v\n", event.Err)
    }
}
```

## Structured Outputs

Generate typed JSON objects with schema validation:

```go
type Recipe struct {
    Name        string   `json:"name"`
    Ingredients []string `json:"ingredients"`
    Steps       []string `json:"steps"`
    PrepTime    int      `json:"prep_time_minutes"`
}

result, err := provider.GenerateObject(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Create a recipe for chocolate chip cookies"},
            },
        },
    },
}, Recipe{})

if err != nil {
    log.Fatal(err)
}

recipe := result.Value.(map[string]interface{})
fmt.Printf("Recipe: %s\n", recipe["name"])
```

## Tool Calling

Gemini supports function calling with automatic parallel execution:

```go
weatherTool := tools.New[WeatherInput, WeatherOutput](
    "get_weather",
    "Get current weather for a location",
    weatherHandler,
)

result, err := provider.GenerateText(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "What's the weather in Tokyo and Paris?"},
            },
        },
    },
    Tools: []core.ToolHandle{
        tools.NewCoreAdapter(weatherTool),
    },
    ToolChoice: core.ToolAuto,
})

// Tools are called automatically and results are included in the response
fmt.Println(result.Text)
```

## Error Handling

The provider maps Gemini errors to the GAI error taxonomy:

```go
result, err := provider.GenerateText(ctx, request)
if err != nil {
    if aiErr, ok := err.(*core.AIError); ok {
        switch aiErr.Code {
        case core.ErrorRateLimited:
            fmt.Printf("Rate limited, retry after: %v\n", aiErr.RetryAfter)
        case core.ErrorContextLengthExceeded:
            fmt.Println("Input too long")
        case core.ErrorContentFiltered:
            fmt.Println("Content blocked by safety filters")
        case core.ErrorUnauthorized:
            fmt.Println("Invalid API key")
        default:
            fmt.Printf("Error: %v\n", err)
        }
    }
}
```

## Supported Models

- **gemini-1.5-flash**: Fast, efficient model for most tasks
- **gemini-1.5-flash-8b**: Smaller, faster variant
- **gemini-1.5-pro**: Advanced reasoning and capabilities
- **gemini-2.0-flash-exp**: Experimental next-generation model

## Performance

Benchmark results on M1 MacBook Pro:

| Operation | Time | Memory | Allocations |
|-----------|------|--------|-------------|
| Provider Creation | 1.2μs | 1,536 B | 12 |
| Request Conversion | 850ns | 752 B | 8 |
| Response Conversion | 420ns | 384 B | 5 |
| Stream Processing | 3.2μs | 1,024 B | 14 |
| Error Mapping | 180ns | 192 B | 3 |
| File Store Operations | 95ns | 48 B | 1 |

## Best Practices

1. **File Uploads**: Use `BlobBytes` for small files (<20MB) and `BlobURL` for larger files
2. **Safety**: Configure safety based on your use case; stricter settings may block legitimate content
3. **Citations**: Enable when you need verifiable sources for generated content
4. **Streaming**: Use for real-time applications and long-form content generation
5. **Rate Limits**: Implement exponential backoff; the provider handles basic retries
6. **Context Length**: Gemini 1.5 models support up to 2M tokens; monitor usage
7. **Multimodal**: Combine different media types in a single request for best results

## Testing

Run tests with:

```bash
# Unit tests (with mock server)
go test ./providers/gemini

# Integration tests (requires GOOGLE_API_KEY)
GOOGLE_API_KEY=your-key go test ./providers/gemini -tags=integration

# Benchmarks
go test ./providers/gemini -bench=. -benchmem
```

## License

Apache 2.0 - See LICENSE file for details.