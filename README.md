# GAI - Go AI Framework

<div align="center">
  <h1>🚀 GAI</h1>
  <p><strong>The Production-Ready, Type-Safe AI Integration Framework for Go</strong></p>
  
  [![Go Reference](https://pkg.go.dev/badge/github.com/recera/gai.svg)](https://pkg.go.dev/github.com/recera/gai)
  [![Go Report Card](https://goreportcard.com/badge/github.com/recera/gai)](https://goreportcard.com/report/github.com/recera/gai)
  [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
  [![Go Version](https://img.shields.io/badge/Go-1.22%2B-blue)](https://go.dev)
  
  <p>
    <a href="#features">Features</a> •
    <a href="#quick-start">Quick Start</a> •
    <a href="#providers">Providers</a> •
    <a href="#documentation">Documentation</a> •
    <a href="#examples">Examples</a> •
    <a href="#contributing">Contributing</a>
  </p>
</div>

---

## 🎯 Overview

GAI is a comprehensive, production-ready framework for building AI-powered applications in Go. It provides a unified, type-safe interface for interacting with multiple AI providers while maintaining operational simplicity and excellent performance.

Whether you're building a simple chatbot, a complex multi-agent system, or an enterprise-grade AI gateway, GAI provides the tools and abstractions you need to succeed.

### Why GAI?

- **🔄 Provider Agnostic**: Single interface for OpenAI, Anthropic, Google Gemini, Ollama, and OpenAI-compatible providers
- **🛡️ Type Safety**: Full compile-time type checking with Go generics
- **⚡ Performance**: Zero-allocation hot paths, efficient streaming
- **🔧 Production Ready**: Built-in retries, rate limiting, observability, and comprehensive error handling
- **🎯 Developer Experience**: Intuitive APIs, extensive documentation, and rich examples
- **🌐 Multimodal**: Native support for text, images, audio, video, and files
- **🔨 Tools**: Type-safe function calling with automatic JSON schema generation
- **📝 Structured Output**: Get typed responses with automatic validation
- **🎙️ Audio**: Built-in TTS and STT support with multiple providers
- **🏗️ Gateway Ready**: Normalized events, idempotency, and stable error taxonomy for building AI gateways

## ✨ Features

### Core Capabilities
- **Multi-Provider Support**: OpenAI, Anthropic, Google Gemini, Ollama, and any OpenAI-compatible API
- **Streaming**: First-class SSE and NDJSON streaming with backpressure
- **Tool Calling**: Type-safe function execution with automatic JSON Schema generation
- **Structured Outputs**: Generate and validate typed JSON responses
- **Multimodal Messages**: Mix text, images, audio, video, and files in conversations
- **Long Context**: Support for up to 200K+ tokens with providers like Claude
- **Vision**: Image analysis with GPT-4V, Claude 3, and Gemini

### Production Features
- **Middleware System**: Composable retry, rate limiting, and safety filters
- **Error Taxonomy**: Unified error classification across all providers
- **Observability**: OpenTelemetry integration for tracing and metrics
- **Prompt Management**: Versioned templates with hot reload
- **Idempotency**: Request-level and tool-level deduplication
- **Gateway Features**: Normalized event streams for provider abstraction

### Developer Tools
- **CLI**: Built-in development server and testing tools
- **Examples**: Comprehensive examples for all features
- **Documentation**: Extensive guides and API reference
- **Type Safety**: Compile-time checking with generics

## 🚀 Quick Start

### Installation

```bash
go get github.com/recera/gai
```

**Requirements**: Go 1.22+ (for generics support)

### Basic Usage

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
    // Create a provider (works with any supported provider)
    provider := openai.New(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4-turbo"),
    )
    
    // Generate text
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Explain quantum computing in simple terms"},
                },
            },
        },
        MaxTokens:   200,
        Temperature: 0.7,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response.Text)
    fmt.Printf("Tokens used: %d\n", response.Usage.TotalTokens)
}
```

### Streaming Example

```go
// Stream responses for real-time output
stream, err := provider.StreamText(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Write a story about AI"},
            },
        },
    },
    Stream: true,
})
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

// Process events as they arrive
for event := range stream.Events() {
    switch event.Type {
    case core.EventTextDelta:
        fmt.Print(event.TextDelta)
    case core.EventToolCall:
        fmt.Printf("\nCalling tool: %s\n", event.ToolName)
    case core.EventFinish:
        fmt.Println("\n\nComplete!")
    }
}
```

### Structured Output with Type Safety

```go
// Define your schema
type Analysis struct {
    Sentiment   string   `json:"sentiment" jsonschema:"enum=positive,enum=neutral,enum=negative"`
    Score       float64  `json:"score" jsonschema:"minimum=0,maximum=1"`
    Keywords    []string `json:"keywords"`
    Summary     string   `json:"summary"`
}

// Generate structured data
result, err := provider.GenerateObject[Analysis](ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Analyze this text: 'GAI makes AI integration in Go simple and powerful!'"},
            },
        },
    },
})

analysis := result.Value
fmt.Printf("Sentiment: %s (%.2f)\n", analysis.Sentiment, analysis.Score)
fmt.Printf("Keywords: %v\n", analysis.Keywords)
```

### Tool Calling

```go
import "github.com/recera/gai/tools"

// Define a typed tool
type WeatherInput struct {
    Location string `json:"location" jsonschema:"required,description=City name"`
    Unit     string `json:"unit,omitempty" jsonschema:"enum=celsius,enum=fahrenheit"`
}

type WeatherOutput struct {
    Temperature float64 `json:"temperature"`
    Conditions  string  `json:"conditions"`
    Humidity    int     `json:"humidity"`
}

// Create the tool
weatherTool := tools.New[WeatherInput, WeatherOutput](
    "get_weather",
    "Get current weather for a location",
    func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
        // Your weather API implementation
        return WeatherOutput{
            Temperature: 22.5,
            Conditions:  "Sunny",
            Humidity:    65,
        }, nil
    },
)

// Use with AI
response, err := provider.GenerateText(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "What's the weather in Tokyo?"},
            },
        },
    },
    Tools: []tools.Handle{weatherTool},
    ToolChoice: core.ToolAuto,
})
```

## 🔌 Providers

GAI supports multiple AI providers with a unified interface:

| Provider | Models | Context | Streaming | Tools | Vision | Audio | Status |
|----------|--------|---------|-----------|-------|--------|-------|--------|
| **OpenAI** | GPT-4, GPT-3.5 | 128K | ✅ | ✅ | ✅ | ✅ | ✅ Production |
| **Anthropic** | Claude 3 (Opus, Sonnet, Haiku) | 200K | ✅ | ✅ | ✅ | ❌ | ✅ Production |
| **Google Gemini** | Gemini 1.5 Pro/Flash | 1M+ | ✅ | ✅ | ✅ | ✅ | ✅ Production |
| **Ollama** | Llama, Mistral, etc. | Varies | ✅ | ✅ | ✅ | ❌ | ✅ Production |
| **OpenAI Compatible** | Any compatible API | Varies | ✅ | ✅ | Varies | Varies | ✅ Production |

### Provider Examples

```go
// OpenAI
provider := openai.New(
    openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    openai.WithModel("gpt-4-turbo"),
)

// Anthropic
provider := anthropic.New(
    anthropic.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
    anthropic.WithModel("claude-3-opus-20240229"),
)

// Google Gemini
provider := gemini.New(
    gemini.WithAPIKey(os.Getenv("GOOGLE_API_KEY")),
    gemini.WithModel("gemini-1.5-pro"),
)

// Ollama (Local)
provider := ollama.New(
    ollama.WithBaseURL("http://localhost:11434"),
    ollama.WithModel("llama3.2"),
)

// OpenAI Compatible (Groq, xAI, Together, etc.)
provider := openai_compat.New(openai_compat.CompatOpts{
    BaseURL: "https://api.groq.com/openai/v1",
    APIKey:  os.Getenv("GROQ_API_KEY"),
    DefaultModel: "llama-3.1-70b-versatile",
})
```

## 🎵 Audio Support (TTS/STT)

GAI includes comprehensive audio support through the media package:

```go
import "github.com/recera/gai/media"

// Text-to-Speech
tts := media.NewElevenLabs(
    media.WithElevenLabsAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
)

stream, err := tts.Synthesize(ctx, media.SpeechRequest{
    Text:   "Hello from GAI!",
    Voice:  "Rachel",
    Format: media.FormatMP3,
})

// Speech-to-Text
stt := media.NewWhisper(
    media.WithWhisperAPIKey(os.Getenv("OPENAI_API_KEY")),
)

result, err := stt.Transcribe(ctx, media.TranscriptionRequest{
    Audio: audioBlob,
    Language: "en",
})

fmt.Println("Transcript:", result.Text)
```

## 🛡️ Production Features

### Middleware

Apply production-ready middleware for reliability:

```go
import "github.com/recera/gai/middleware"

// Chain middleware for production use
provider = middleware.Chain(
    middleware.WithRetry(middleware.RetryOpts{
        MaxAttempts: 3,
        BaseDelay:   time.Second,
        MaxDelay:    10 * time.Second,
        Jitter:      true,
    }),
    middleware.WithRateLimit(middleware.RateLimitOpts{
        RPS:   10,
        Burst: 20,
    }),
    middleware.WithSafety(middleware.SafetyOpts{
        MaxPromptLength: 10000,
        BlockPatterns:   []string{"password", "credit card"},
    }),
)(provider)
```

### Observability

Built-in OpenTelemetry support:

```go
import "github.com/recera/gai/obs"

// Initialize observability
shutdown, err := obs.Init(obs.Config{
    ServiceName:    "my-ai-app",
    ServiceVersion: "1.0.0",
    Environment:    "production",
})
defer shutdown(context.Background())

// Traces and metrics are automatically collected
```

### Error Handling

Unified error taxonomy across all providers:

```go
response, err := provider.GenerateText(ctx, request)
if err != nil {
    switch {
    case core.IsRateLimited(err):
        // Handle rate limiting
        time.Sleep(core.GetRetryAfter(err))
        
    case core.IsContextSizeExceeded(err):
        // Reduce context size
        request.Messages = truncateMessages(request.Messages)
        
    case core.IsAuth(err):
        // Check API keys
        return fmt.Errorf("authentication failed: %w", err)
        
    case core.IsTransient(err):
        // Retry with backoff
        return retryWithBackoff(request)
        
    default:
        return fmt.Errorf("unexpected error: %w", err)
    }
}
```

## 📚 Documentation

Comprehensive documentation is available in the [docs](./docs) directory:

- **[Getting Started](./docs/getting-started/)** - Installation, configuration, and quick start
- **[Core Concepts](./docs/core-concepts/)** - Architecture, messages, streaming, and tools
- **[Provider Guides](./docs/providers/)** - Detailed guides for each provider
- **[Features](./docs/features/)** - Deep dives into specific features
- **[Tutorials](./docs/tutorials/)** - Step-by-step guides for common use cases
- **[API Reference](./docs/api-reference/)** - Complete API documentation
- **[Deployment](./docs/deployment/)** - Production deployment guides
- **[Troubleshooting](./docs/troubleshooting/)** - Common issues and solutions

## 💡 Examples

The [examples](./examples) directory contains runnable examples for all features:

- **[hello-text](./examples/hello-text)** - Basic text generation
- **[hello-stream](./examples/hello-stream)** - Streaming responses
- **[hello-object](./examples/hello-object)** - Structured outputs with type safety
- **[hello-tool](./examples/hello-tool)** - Tool calling and multi-step workflows
- **[hello-vision](./examples/hello-vision)** - Image analysis with vision models
- **[hello-audio](./examples/hello-audio)** - Speech synthesis and recognition
- **[multi-provider](./examples/multi-provider)** - Using multiple providers
- **[chat-app](./examples/chat-app)** - Complete chat application

## 🛠️ Development

### Prerequisites

- Go 1.22+ (required for generics)
- Git
- Make (optional, for convenience commands)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/recera/gai.git
cd gai

# Install dependencies
go mod download

# Run tests
go test ./...

# Run with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./...

# Build the CLI
go build -o ai ./cmd/ai

# Run the development server
./ai dev serve
```

### Development Server

GAI includes a development server for testing:

```bash
# Install the CLI globally
go install github.com/recera/gai/cmd/ai@latest

# Start the development server
ai dev serve

# The server provides:
# - Interactive web UI: http://localhost:8080
# - SSE streaming endpoint: /api/chat
# - NDJSON streaming endpoint: /api/chat/ndjson
# - REST endpoint: /api/generate
# - Health check: /api/health
```

### Testing

```bash
# Run all tests
make test

# Run specific package tests
go test ./core
go test ./providers/openai
go test ./tools

# Run integration tests (requires API keys)
OPENAI_API_KEY=sk-... go test -tags=integration ./providers/openai

# Run benchmarks
make bench

# Check code coverage
make coverage
```

## 🏗️ Architecture

GAI is built on solid architectural principles:

```
┌─────────────────────────────────────────────────┐
│                Your Application                  │
└─────────────────────────────────────────────────┘
                         │
┌─────────────────────────────────────────────────┐
│                  GAI Framework                   │
├─────────────────────────────────────────────────┤
│  Core Types │ Tools │ Streaming │ Middleware    │
├─────────────────────────────────────────────────┤
│              Provider Abstraction                │
├─────────────────────────────────────────────────┤
│ OpenAI │ Anthropic │ Gemini │ Ollama │ Others  │
└─────────────────────────────────────────────────┘
```

### Package Structure

- **`core`** - Core types, interfaces, and multi-step runner
- **`providers`** - Provider implementations
  - `openai` - OpenAI provider
  - `anthropic` - Anthropic Claude provider
  - `gemini` - Google Gemini provider
  - `ollama` - Local model provider
  - `openai_compat` - OpenAI-compatible adapter
- **`tools`** - Tool system with JSON Schema generation
- **`stream`** - Streaming utilities (SSE, NDJSON, normalization)
- **`middleware`** - Retry, rate limiting, safety filters
- **`prompts`** - Prompt template management
- **`media`** - Audio support (TTS/STT)
- **`obs`** - Observability with OpenTelemetry
- **`cmd/ai`** - CLI and development server

## 🚦 Implementation Status

### ✅ Production Ready

- Core framework with provider abstraction
- OpenAI provider with full features
- Anthropic Claude provider
- Google Gemini provider with multimodal support
- Ollama local model provider
- OpenAI-compatible adapter (Groq, xAI, Together, etc.)
- Type-safe tool calling with JSON Schema
- Streaming (SSE and NDJSON)
- Structured output generation
- Middleware system
- Prompt management
- Audio support (TTS/STT)
- Observability
- Gateway architecture improvements
- CLI with development server

### 🚧 Roadmap

- [ ] WebSocket streaming support
- [ ] Model routing and automatic failover
- [ ] MCP (Model Context Protocol) support
- [ ] Embedding APIs
- [ ] Fine-tuning management
- [ ] Batch processing
- [ ] Cost tracking and optimization
- [ ] Playground UI improvements
- [ ] Additional providers (Cohere, AI21, etc.)

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](./CONTRIBUTING.md) for details.

### How to Contribute

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Guidelines

- Follow Go idioms and best practices
- Add tests for new features
- Update documentation
- Ensure all tests pass
- Use conventional commits

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](./LICENSE) file for details.

## 🙏 Acknowledgments

GAI stands on the shoulders of giants:

- The Go team for an excellent language and toolchain
- AI provider teams for their powerful models
- The open-source community for inspiration and feedback
- Contributors who have helped improve the framework

## 📞 Support

- 📖 [Documentation](./docs)
- 💬 [Discussions](https://github.com/recera/gai/discussions)
- 🐛 [Issue Tracker](https://github.com/recera/gai/issues)
- 📧 [Email](mailto:support@recera.com)

## 🌟 Star History

If you find GAI useful, please consider giving it a star! It helps others discover the project.

[![Star History Chart](https://api.star-history.com/svg?repos=recera/gai&type=Date)](https://star-history.com/#recera/gai&Date)

---

<div align="center">
  <p><strong>Build Amazing AI Applications with Go!</strong></p>
  <p>Made with ❤️ by the GAI Team</p>
</div>