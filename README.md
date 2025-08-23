# GAI - Go AI Framework

[![CI](https://github.com/recera/gai/actions/workflows/ci.yml/badge.svg)](https://github.com/recera/gai/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/recera/gai.svg)](https://pkg.go.dev/github.com/recera/gai)
[![Go Report Card](https://goreportcard.com/badge/github.com/recera/gai)](https://goreportcard.com/report/github.com/recera/gai)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A production-grade, provider-agnostic Go framework for building AI applications with support for OpenAI, Anthropic, Google Gemini, Ollama, and any OpenAI-compatible API.

## Features

- 🔄 **Provider Agnostic** - Single interface for all AI providers
- 🏗️ **Typed APIs** - Full type safety with generics for structured outputs
- 🔧 **Tool Calling** - Type-safe tools with automatic JSON Schema generation
- 🌊 **Streaming** - First-class streaming support with SSE and NDJSON
- 📊 **Observability** - Built-in OpenTelemetry tracing and metrics
- 🔄 **Multi-Step** - Automatic multi-step execution with tool loops
- 🎯 **Router** - Intelligent routing based on cost, latency, and capabilities
- 🧰 **MCP Support** - Import and export tools via Model Context Protocol
- 🎙️ **Multimodal** - Support for text, images, audio, video, and files
- 🛡️ **Production Ready** - Comprehensive error handling, retries, and safety

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
    client := openai.New(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )

    // Generate text
    ctx := context.Background()
    result, err := client.GenerateText(ctx, core.Request{
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
stream, err := client.StreamText(ctx, core.Request{
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

result, err := client.GenerateObject[Recipe](ctx, core.Request{
    Messages: []core.Message{
        {Role: core.User, Parts: []core.Part{
            core.Text{Text: "Give me a recipe for chocolate chip cookies"},
        }},
    },
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Recipe: %s\n", result.Value.Name)
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

// Use in request
result, err := client.GenerateText(ctx, core.Request{
    Messages: messages,
    Tools:    []tools.Handle{weatherTool},
    StopWhen: core.MaxSteps(3),
})
```

## Supported Providers

- **OpenAI** - GPT-4, GPT-3.5, and other OpenAI models
- **Anthropic** - Claude 3 family
- **Google Gemini** - Gemini Pro and Flash
- **Ollama** - Local models
- **OpenAI Compatible** - Groq, xAI, Baseten, Cerebras, and any OpenAI-compatible endpoint

## Architecture

The framework is organized into focused packages:

- `core` - Core types and interfaces
- `providers/*` - Provider implementations
- `tools` - Tool definition and execution
- `stream` - Streaming utilities (SSE, NDJSON)
- `prompts` - Prompt management and versioning
- `router` - Multi-provider routing
- `middleware` - Retry, rate limiting, safety
- `mcp` - Model Context Protocol support

## Development

```bash
# Run tests
make test

# Run with coverage
make coverage

# Run linters
make lint

# Run benchmarks
make benchmark

# Run all CI checks locally
make ci
```

## Documentation

Full documentation is available at [pkg.go.dev/github.com/recera/gai](https://pkg.go.dev/github.com/recera/gai).

See the [examples](./examples) directory for more usage patterns.

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](./LICENSE) file for details.

## Status

This project is in active development (v0.x). APIs may change before v1.0.