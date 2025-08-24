# GAI - Go AI Framework

[![Go Reference](https://pkg.go.dev/badge/github.com/recera/gai.svg)](https://pkg.go.dev/github.com/recera/gai)
[![Go Report Card](https://goreportcard.com/badge/github.com/recera/gai)](https://goreportcard.com/report/github.com/recera/gai)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A production-grade, provider-agnostic Go framework for building AI applications with OpenAI and OpenAI-compatible APIs.

## Features

- 🔄 **Provider Agnostic** - Single interface for AI providers
- 🏗️ **Typed APIs** - Full type safety with generics for structured outputs
- 🔧 **Tool Calling** - Type-safe tools with automatic JSON Schema generation
- 🌊 **Streaming** - First-class streaming support with SSE and NDJSON
- 📊 **Observability** - Built-in OpenTelemetry tracing and metrics
- 🔄 **Multi-Step** - Automatic multi-step execution with tool loops
- 🛡️ **Production Ready** - Comprehensive error handling, retries, and safety
- 🎯 **Structured Outputs** - Type-safe JSON generation with schema validation
- 📝 **Prompt Management** - Versioned templates with hot reload support
- 🚀 **Development Tools** - Built-in CLI with dev server and examples

## Installation

```bash
go get github.com/recera/gai
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
    // Create a provider
    var provider core.Provider = openai.New(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )

    // Generate text
    ctx := context.Background()
    result, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Explain quantum computing in simple terms"},
                },
            },
        },
        MaxTokens: 200,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Text)
}
```

## Streaming Example

```go
// Stream responses for real-time output
stream, err := provider.StreamText(ctx, core.Request{
    Messages: messages,
    Stream: true,
})
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for event := range stream.Events() {
    switch event.Type {
    case core.EventTextDelta:
        fmt.Print(event.TextDelta)
    case core.EventFinish:
        fmt.Println("\n\nDone!")
    }
}
```

## Structured Outputs

```go
type Recipe struct {
    Name        string   `json:"name"`
    Ingredients []string `json:"ingredients"`
    Steps       []string `json:"steps"`
}

result, err := provider.GenerateObject(ctx, core.Request{
    Messages: []core.Message{
        {Role: core.User, Parts: []core.Part{
            core.Text{Text: "Give me a recipe for chocolate chip cookies"},
        }},
    },
}, Recipe{})
if err != nil {
    log.Fatal(err)
}

recipe := result.Value.(*Recipe)
fmt.Printf("Recipe: %s\n", recipe.Name)
```

## Tool Calling

```go
// Define a typed tool
type WeatherInput struct {
    Location string `json:"location"`
}

type WeatherOutput struct {
    Temperature float64 `json:"temperature"`
    Conditions  string  `json:"conditions"`
}

weatherTool := tools.New[WeatherInput, WeatherOutput](
    "get_weather",
    "Get current weather for a location",
    func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
        // Implementation here
        return WeatherOutput{
            Temperature: 72.5,
            Conditions:  "Sunny",
        }, nil
    },
)

// Convert to core handles and use in request
coreTools := tools.ToCoreHandles([]tools.Handle{weatherTool})
result, err := provider.GenerateText(ctx, core.Request{
    Messages: messages,
    Tools:    coreTools,
    ToolChoice: core.ToolAuto,
})
```

## Middleware

Apply production-ready middleware for retries and rate limiting:

```go
import "github.com/recera/gai/middleware"

provider = middleware.Chain(
    middleware.WithRetry(middleware.RetryOpts{
        MaxAttempts: 3,
        BaseDelay:   time.Second,
        Jitter:      true,
    }),
    middleware.WithRateLimit(middleware.RateLimitOpts{
        RPS:   10,
        Burst: 20,
    }),
)(provider)
```

## Development Server

The framework includes a development server for testing:

```bash
# Install the CLI
go install ./cmd/ai

# Start the development server
ai dev serve

# Visit http://localhost:8080 for an interactive web interface
```

The dev server provides:
- Interactive web UI for testing
- SSE streaming endpoint (`/api/chat`)
- NDJSON streaming endpoint (`/api/chat/ndjson`)
- Traditional REST endpoint (`/api/generate`)
- Health check endpoint (`/api/health`)

## Examples

See the [examples](./examples) directory for comprehensive examples:

- **[hello-text](./examples/hello-text)** - Basic text generation
- **[hello-stream](./examples/hello-stream)** - Streaming responses
- **[hello-object](./examples/hello-object)** - Structured outputs
- **[hello-tool](./examples/hello-tool)** - Tool calling and multi-step workflows

## Architecture

The framework is organized into focused packages:

- `core` - Core types and interfaces
- `providers/openai` - OpenAI provider implementation
- `tools` - Tool definition and execution with JSON Schema
- `stream` - Streaming utilities (SSE, NDJSON)
- `prompts` - Prompt management and versioning
- `middleware` - Retry, rate limiting, and safety
- `obs` - Observability with OpenTelemetry

## Current Implementation Status

### ✅ Completed
- Core framework with provider abstraction
- OpenAI provider with full feature support
- Type-safe tool calling with JSON Schema generation
- Streaming support (SSE and NDJSON)
- Structured output generation
- Middleware (retry, rate limiting, safety)
- Prompt management with versioning
- Observability with OpenTelemetry
- CLI with development server
- Comprehensive examples

### 🚧 In Progress
- Anthropic provider
- Google Gemini provider
- Ollama provider
- OpenAI-compatible adapter for Groq, xAI, etc.
- Model routing and failover
- MCP (Model Context Protocol) support
- Audio/multimodal support

## Development

```bash
# Run tests
go test ./...

# Run with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./...

# Build the CLI
go build -o ai ./cmd/ai
```

## Documentation

Full documentation and API reference coming soon at pkg.go.dev.

See the [examples](./examples) directory for detailed usage patterns.

## Contributing

We welcome contributions! Please ensure:
- All tests pass
- Code follows Go idioms
- New features include tests
- Documentation is updated

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](./LICENSE) file for details.

## Status

This project is in active development (v0.8.x). The core API is stabilizing, but some features are still being implemented. Production use is possible with the completed features.