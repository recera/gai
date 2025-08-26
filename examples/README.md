# GAI Framework Examples

This directory contains comprehensive examples demonstrating the capabilities of the GAI (Go AI) Framework. Each example is a standalone Go program that showcases different features of the framework.

## Prerequisites

Before running these examples, ensure you have:

1. **Go 1.23+** installed (required for generics and latest language features)
2. **API Keys** for the providers you want to use:
   - OpenAI: Set `OPENAI_API_KEY` environment variable
   - Anthropic: Set `ANTHROPIC_API_KEY` environment variable
   - Google Gemini: Set `GOOGLE_API_KEY` environment variable
   - Groq: Set `GROQ_API_KEY` environment variable
   - Ollama: No API key needed for local deployment

## Examples Overview

### 1. hello-text
**Basic Text Generation**

Demonstrates fundamental text generation capabilities including:
- Simple text generation
- System prompts for behavior control
- Multi-turn conversations
- Temperature control for creativity adjustment

```bash
cd hello-text
go run main.go
```

### 2. hello-stream
**Streaming Responses**

Shows real-time streaming capabilities:
- Basic streaming with text deltas
- Event type monitoring
- Real-time text processing
- Error handling in streams
- Graceful stream interruption

```bash
cd hello-stream
go run main.go
```

### 3. hello-object
**Structured Output Generation**

Demonstrates type-safe JSON generation:
- Recipe generation with ingredients and instructions
- Todo list creation with prioritized tasks
- Business analysis with SWOT framework
- Code review with structured feedback

```bash
cd hello-object
go run main.go
```

### 4. hello-tool
**Tool Calling & Function Execution**

Showcases advanced tool capabilities:
- Single tool execution
- Multiple tools in parallel
- Multi-step workflows
- Streaming with tool calls
- Type-safe tool definitions

```bash
cd hello-tool
go run main.go
```

### 5. advanced-workflows
**Complex Multi-Step Workflows**

Demonstrates sophisticated workflow patterns:
- Multi-provider comparison
- Research pipeline coordination
- Advanced stop conditions
- Streaming multi-step execution
- Production-ready patterns

```bash
cd advanced-workflows
go run main.go
```

### 6. observability
**Comprehensive Observability**

Showcases monitoring and telemetry:
- OpenTelemetry integration
- Distributed tracing
- Metrics collection
- Usage accounting and cost estimation
- Error tracking and performance monitoring

```bash
cd observability
go run main.go
```

### 7. Provider-Specific Examples

#### ollama/
**Local Model Deployment**

Demonstrates Ollama provider capabilities:
- Local model management
- Multimodal concepts
- Concurrent request handling
- Provider-specific options
- Structured output with local models

```bash
cd ollama
go run main.go
```

#### openai/
**OpenAI Provider Focus**

Focused OpenAI implementation:
- Latest model capabilities
- Observability integration
- Advanced tool calling patterns
- Cost tracking and optimization

```bash
cd openai
go run main.go
```

### 8. prompts_demo
**Template Management System**

Demonstrates the prompts package:
- Versioned template management
- Development overrides
- Template helpers and functions
- Observability integration
- Production deployment patterns

```bash
cd prompts_demo
go run main.go
```

## Quick Start

1. **Clone the repository:**
```bash
git clone https://github.com/recera/gai.git
cd gai/examples
```

2. **Set your API keys (choose providers you want to test):**
```bash
# Required for most examples
export OPENAI_API_KEY="your-api-key-here"

# Optional for additional providers
export ANTHROPIC_API_KEY="your-anthropic-key"
export GOOGLE_API_KEY="your-gemini-key"
export GROQ_API_KEY="your-groq-key"

# For local deployment (no API key needed)
# ollama serve  # Start Ollama server
# ollama pull llama3.2  # Pull a model
```

3. **Run an example:**
```bash
# Start with basic text generation
cd hello-text
go run main.go

# Or try advanced workflows
cd advanced-workflows
go run main.go

# Or test local deployment
cd ollama
go run main.go
```

## Features Demonstrated

### Core Features
- ✅ **Type Safety**: Strongly typed requests and responses
- ✅ **Provider Abstraction**: Swap providers without code changes
- ✅ **Middleware**: Automatic retry and rate limiting
- ✅ **Error Handling**: Comprehensive error classification
- ✅ **Context Support**: Proper cancellation and timeouts
- ✅ **Local Deployment**: Ollama integration for on-premises models

### Advanced Features
- ✅ **Streaming**: Real-time response streaming with SSE/NDJSON
- ✅ **Structured Outputs**: Type-safe JSON generation with schema validation
- ✅ **Tool Calling**: Typed tools with automatic execution and parallel processing
- ✅ **Multi-Step Workflows**: Complex agent behaviors with stop conditions
- ✅ **Observability**: OpenTelemetry tracing, metrics, and cost tracking
- ✅ **Template Management**: Versioned prompts with override support
- ✅ **Multi-Provider**: Compare and orchestrate across different providers
- ✅ **Production Patterns**: Error recovery, circuit breakers, and monitoring

## Example Patterns

### Basic Text Generation
```go
provider := openai.New(
    openai.WithAPIKey(apiKey),
    openai.WithModel("gpt-4o-mini"),
)

request := core.Request{
    Messages: []core.Message{
        {Role: core.User, Parts: []core.Part{
            core.Text{Text: "Hello, world!"},
        }},
    },
}

result, err := provider.GenerateText(ctx, request)
```

### Streaming
```go
stream, err := provider.StreamText(ctx, request)
defer stream.Close()

for event := range stream.Events() {
    switch event.Type {
    case core.EventTextDelta:
        fmt.Print(event.TextDelta)
    case core.EventFinish:
        fmt.Println("Done!")
    }
}
```

### Structured Output
```go
type Recipe struct {
    Name        string   `json:"name"`
    Ingredients []string `json:"ingredients"`
}

result, err := provider.GenerateObject(ctx, request, Recipe{})
recipe := result.Value.(*Recipe)
```

### Tool Calling
```go
weatherTool := tools.New[WeatherInput, WeatherOutput](
    "get_weather",
    "Get current weather",
    func(ctx context.Context, in WeatherInput, meta tools.Meta) (WeatherOutput, error) {
        // Implementation
    },
)

request.Tools = []core.ToolHandle{weatherTool}
request.ToolChoice = core.ToolAuto
```

## Middleware Configuration

All examples include production-ready middleware:

```go
provider = middleware.Chain(
    middleware.WithRetry(middleware.RetryOpts{
        MaxAttempts: 3,
        BaseDelay:   time.Second,
        MaxDelay:    10 * time.Second,
        Jitter:      true,
    }),
    middleware.WithRateLimit(middleware.RateLimitOpts{
        RPS:   5,
        Burst: 10,
    }),
)(provider)
```

## Error Handling

The framework provides comprehensive error classification:

```go
if err != nil {
    if aiErr, ok := err.(*core.AIError); ok {
        fmt.Printf("Error Code: %s\n", aiErr.Code)
        fmt.Printf("Retryable: %v\n", aiErr.Retryable)
        
        if core.IsRateLimited(err) {
            // Wait before retrying
        }
        if core.IsTransient(err) {
            // Can retry immediately
        }
    }
}
```

## Performance Considerations

- **Streaming**: Use for long responses to improve perceived latency
- **Middleware**: Configure retry and rate limits based on your use case
- **Context**: Always use contexts with appropriate timeouts
- **Token Limits**: Set `MaxTokens` to control costs and response length

## Troubleshooting

### Common Issues

1. **"OPENAI_API_KEY not set"**
   - Solution: Export your API key: `export OPENAI_API_KEY="sk-..."`
   - Each provider requires its own API key

2. **"context deadline exceeded"**
   - Solution: Increase timeout in context creation
   - Check network connectivity
   - For Ollama: Ensure `ollama serve` is running

3. **"rate limited"**
   - Solution: Middleware automatically handles retries
   - Adjust rate limit configuration if needed
   - Try different providers with higher limits

4. **"model not found"**
   - Solution: Ensure you're using a valid model name
   - For Ollama: Run `ollama pull <model-name>` first
   - Check provider documentation for available models

5. **"Ollama connection refused"**
   - Solution: Start Ollama server: `ollama serve`
   - Check if port 11434 is available
   - Verify Ollama installation

6. **"tool execution failed"**
   - Solution: Check tool input/output schema matching
   - Verify tool execution logic
   - Enable debug logging for detailed error info

## Contributing

We welcome contributions! To add a new example:

1. Create a new directory with a descriptive name
2. Include a self-contained `main.go` file
3. Add comprehensive comments explaining the features
4. Update this README with your example

## Resources

- [GAI Framework Documentation](../README.md)
- [API Reference](https://pkg.go.dev/github.com/recera/gai)
- [OpenAI API Documentation](https://platform.openai.com/docs)
- [Anthropic API Documentation](https://docs.anthropic.com)
- [Google AI Documentation](https://ai.google.dev)

## License

These examples are part of the GAI Framework and are licensed under the same terms. See the [LICENSE](../LICENSE) file for details.