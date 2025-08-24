# GAI Documentation

<div align="center">
  <h1>🚀 GAI - Go AI Framework</h1>
  <p><strong>The Production-Ready, Type-Safe AI Integration Framework for Go</strong></p>
  <p>
    <a href="#quick-start">Quick Start</a> •
    <a href="#features">Features</a> •
    <a href="#documentation">Documentation</a> •
    <a href="#providers">Providers</a> •
    <a href="#examples">Examples</a> •
    <a href="#contributing">Contributing</a>
  </p>
</div>

---

## Welcome to GAI

GAI (Go AI) is a comprehensive, production-ready framework for building AI-powered applications in Go. It provides a unified, type-safe interface for interacting with multiple AI providers while maintaining operational simplicity and excellent performance.

Whether you're building a simple chatbot, a complex multi-agent system, or an enterprise-grade AI gateway, GAI provides the tools and abstractions you need to succeed.

## 🎯 Why GAI?

### The Challenge

Building AI applications in Go presents unique challenges:
- Each AI provider has different APIs, authentication methods, and capabilities
- Switching providers requires significant code changes
- Error handling is inconsistent across providers
- Streaming, tool calling, and structured outputs have different implementations
- Type safety is often sacrificed for flexibility
- Production concerns like retries, rate limiting, and observability are afterthoughts

### The GAI Solution

GAI solves these problems with:

- **🔄 Unified Interface**: One API for all providers - switch providers with a single line change
- **🛡️ Type Safety**: Full compile-time type checking with Go generics
- **⚡ Performance**: Zero-allocation hot paths, efficient streaming
- **🔧 Production Ready**: Built-in retries, rate limiting, observability, and error handling
- **🎯 Developer Experience**: Intuitive APIs, comprehensive docs, and examples
- **🌐 Multi-Provider**: Support for OpenAI, Anthropic, Google Gemini, Ollama, and OpenAI-compatible providers
- **🎵 Multimodal**: Text, images, audio, video, and file support
- **🔨 Tools**: Type-safe tool calling with automatic JSON schema generation
- **📝 Structured Output**: Get typed responses with automatic validation
- **🎙️ Audio**: Built-in TTS and STT support with multiple providers

## 📚 Documentation Structure

Our documentation is organized to help you quickly find what you need:

### 🚀 [Getting Started](./getting-started/)
- [Installation Guide](./getting-started/installation.md) - Set up GAI in your project
- [Quick Start Tutorial](./getting-started/quickstart.md) - Build your first AI app in 5 minutes
- [Basic Examples](./getting-started/basic-examples.md) - Simple, focused examples
- [Configuration](./getting-started/configuration.md) - Provider setup and configuration

### 🧠 [Core Concepts](./core-concepts/)
- [Architecture Overview](./core-concepts/architecture.md) - Understand GAI's design
- [Messages and Parts](./core-concepts/messages.md) - Multimodal message system
- [Providers](./core-concepts/providers.md) - Provider abstraction and switching
- [Streaming](./core-concepts/streaming.md) - Real-time response streaming
- [Error Handling](./core-concepts/errors.md) - Unified error taxonomy
- [Tools](./core-concepts/tools.md) - Function calling and tool execution

### 🔌 [Provider Guides](./providers/)
Complete guides for each supported provider:
- [OpenAI](./providers/openai.md) - GPT-4, GPT-3.5, DALL-E integration
- [Anthropic](./providers/anthropic.md) - Claude 3 models
- [Google Gemini](./providers/gemini.md) - Gemini Pro, multimodal features
- [Ollama](./providers/ollama.md) - Local model execution
- [OpenAI Compatible](./providers/openai-compatible.md) - Groq, xAI, Cerebras, and more

### ⚡ [Features](./features/)
Deep dives into GAI's powerful features:
- [Structured Outputs](./features/structured-outputs.md) - Type-safe JSON responses
- [Tool Calling](./features/tool-calling.md) - Multi-step function execution
- [Prompt Management](./features/prompts.md) - Versioned prompt templates
- [Audio (TTS/STT)](./features/audio.md) - Speech synthesis and recognition
- [Streaming](./features/streaming.md) - SSE and NDJSON streaming
- [Middleware](./features/middleware.md) - Retry, rate limit, safety
- [Observability](./features/observability.md) - Metrics and tracing

### 📖 [Tutorials](./tutorials/)
Step-by-step guides for common use cases:
- [Building a Chatbot](./tutorials/chatbot.md) - Complete chat application
- [Multi-Agent Systems](./tutorials/multi-agent.md) - Coordinate multiple AI agents
- [RAG Implementation](./tutorials/rag.md) - Retrieval-augmented generation
- [Voice Assistant](./tutorials/voice-assistant.md) - Speech-enabled AI
- [API Gateway](./tutorials/gateway.md) - Build an AI gateway service

### 🔧 [API Reference](./api-reference/)
Complete API documentation:
- [Core Package](./api-reference/core.md) - Core types and interfaces
- [Providers Package](./api-reference/providers.md) - Provider implementations
- [Tools Package](./api-reference/tools.md) - Tool system
- [Stream Package](./api-reference/stream.md) - Streaming utilities
- [Media Package](./api-reference/media.md) - Audio/TTS/STT

### 📘 [Guides](./guides/)
Best practices and advanced topics:
- [Migration Guide](./guides/migration.md) - Migrate from other frameworks
- [Best Practices](./guides/best-practices.md) - Production recommendations
- [Performance Tuning](./guides/performance.md) - Optimization guide
- [Security](./guides/security.md) - Security best practices
- [Testing](./guides/testing.md) - Testing AI applications

### 🚢 [Deployment](./deployment/)
Production deployment guidance:
- [Docker](./deployment/docker.md) - Containerization
- [Kubernetes](./deployment/kubernetes.md) - K8s deployment
- [Cloud Platforms](./deployment/cloud.md) - AWS, GCP, Azure
- [Monitoring](./deployment/monitoring.md) - Observability setup
- [Scaling](./deployment/scaling.md) - High-availability patterns

### 🔍 [Troubleshooting](./troubleshooting/)
- [Common Issues](./troubleshooting/common-issues.md) - Frequent problems and solutions
- [Error Reference](./troubleshooting/errors.md) - Error code explanations
- [FAQ](./troubleshooting/faq.md) - Frequently asked questions
- [Debug Guide](./troubleshooting/debugging.md) - Debugging techniques

## Quick Start

Get started with GAI in seconds:

```bash
# Install GAI
go get github.com/yourusername/gai

# Set up your API keys
export OPENAI_API_KEY="your-key-here"
```

Create your first AI application:

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/yourusername/gai/core"
    "github.com/yourusername/gai/providers/openai"
)

func main() {
    // Create a provider
    provider := openai.New(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4"),
    )
    
    // Generate text
    response, err := provider.GenerateText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a haiku about Go programming"},
                },
            },
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response.Text)
}
```

## 🌟 Key Features

### Multi-Provider Support
```go
// Switch providers with one line
provider := openai.New(...)      // OpenAI
provider := anthropic.New(...)   // Anthropic
provider := gemini.New(...)      // Google Gemini
provider := ollama.New(...)      // Local models
```

### Type-Safe Structured Outputs
```go
type Analysis struct {
    Sentiment string   `json:"sentiment"`
    Score     float64  `json:"score"`
    Keywords  []string `json:"keywords"`
}

result, err := provider.GenerateObject[Analysis](ctx, request)
fmt.Printf("Sentiment: %s (%.2f)\n", result.Value.Sentiment, result.Value.Score)
```

### Streaming Responses
```go
stream, err := provider.StreamText(ctx, request)
for event := range stream.Events() {
    switch event.Type {
    case core.EventTextDelta:
        fmt.Print(event.TextDelta)
    case core.EventToolCall:
        fmt.Printf("Calling tool: %s\n", event.ToolName)
    }
}
```

### Tool Calling
```go
weatherTool := tools.New[WeatherInput, WeatherOutput](
    "get_weather",
    "Get current weather for a location",
    func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
        // Your weather API logic here
        return WeatherOutput{Temperature: 72, Condition: "Sunny"}, nil
    },
)

response, err := provider.GenerateText(ctx, core.Request{
    Messages: messages,
    Tools: []tools.Handle{weatherTool},
})
```

## Supported Providers

| Provider | Models | Streaming | Tools | Structured Output | Multimodal |
|----------|--------|-----------|-------|-------------------|------------|
| OpenAI | GPT-4, GPT-3.5 | ✅ | ✅ | ✅ | ✅ (Images) |
| Anthropic | Claude 3 | ✅ | ✅ | ✅ | ✅ (Images) |
| Google Gemini | Gemini Pro | ✅ | ✅ | ✅ | ✅ (All) |
| Ollama | Local Models | ✅ | ✅ | ✅ | ✅ (Images) |
| Groq | Llama, Mixtral | ✅ | ✅ | ✅ | ❌ |
| xAI | Grok | ✅ | ✅ | ✅ | ❌ |
| Together | Many Models | ✅ | ✅ | ✅ | ✅ |

## Use Cases

GAI is perfect for:

- **🤖 Chatbots & Assistants**: Build conversational AI with memory and context
- **📊 Data Analysis**: Extract insights and generate reports
- **🔍 Content Generation**: Create articles, code, and creative content
- **🎵 Voice Applications**: Build voice assistants with TTS/STT
- **🏢 Enterprise AI Gateway**: Unified API for all AI services
- **🔬 Research Applications**: Experiment with different models
- **📱 Multi-Modal Apps**: Process text, images, audio, and video
- **⚙️ Automation**: Build AI-powered automation workflows

## Architecture

GAI is built on solid architectural principles:

```
┌─────────────────────────────────────────────────┐
│                 Your Application                 │
└─────────────────────────────────────────────────┘
                          │
┌─────────────────────────────────────────────────┐
│                   GAI Framework                  │
├─────────────────────────────────────────────────┤
│  Core Types │ Tools │ Streaming │ Middleware    │
├─────────────────────────────────────────────────┤
│              Provider Abstraction                │
├─────────────────────────────────────────────────┤
│ OpenAI │ Anthropic │ Gemini │ Ollama │ Others  │
└─────────────────────────────────────────────────┘
```

## 🔄 Version Compatibility

| GAI Version | Go Version | Status |
|-------------|------------|--------|
| v1.0.x | 1.22+ | Current |
| v0.9.x | 1.22+ | Maintenance |

## Contributing

We welcome contributions! See our [Contributing Guide](../CONTRIBUTING.md) for details.

### Development Setup
```bash
# Clone the repository
git clone https://github.com/yourusername/gai.git
cd gai

# Install dependencies
go mod download

# Run tests
make test

# Run benchmarks
make bench
```

## 📄 License

GAI is released under the Apache 2.0 License. See [LICENSE](../LICENSE) for details.

## Acknowledgments

GAI stands on the shoulders of giants:
- The Go team for an excellent language and toolchain
- AI provider teams for their powerful models
- The open-source community for inspiration and feedback

## 📞 Support

- [Documentation](https://gai.dev/docs)
- [Discord Community](https://discord.gg/gai)
- [Issue Tracker](https://github.com/yourusername/gai/issues)
- [Email Support](mailto:support@gai.dev)

## What's Next?

- Explore the [Getting Started Guide](./getting-started/installation.md)
- Check out [Examples](./examples/)
- Read about [Core Concepts](./core-concepts/)
- Join our [Community](https://discord.gg/gai)

---

<div align="center">
  <p><strong>Build Amazing AI Applications with Go!</strong></p>
  <p>⭐ Star us on GitHub • 🐦 Follow us on Twitter • 💬 Join our Discord</p>
</div>