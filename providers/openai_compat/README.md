# OpenAI-Compatible Provider for GAI Framework

The `openai_compat` package provides a flexible adapter for any API that implements the OpenAI Chat Completions specification. This includes providers like Groq, xAI, Cerebras, Baseten, Together, Fireworks, and Anyscale.

## Features

- 🔌 **Universal Compatibility**: Works with any OpenAI-compatible API endpoint
- ⚙️ **Provider-Specific Quirks**: Automatic handling of provider limitations
- 🚀 **Preset Configurations**: Ready-to-use configurations for popular providers
- 🔄 **Automatic Retries**: Exponential backoff with jitter for transient failures
- 📊 **Capability Detection**: Automatic probing of provider capabilities
- 🛠️ **Flexible Configuration**: Fine-grained control over provider behavior
- 📈 **Full Observability**: Integrated metrics and tracing support
- 🔧 **Tool Support**: Complete tool calling with parallel execution
- 📝 **Structured Outputs**: JSON Schema-based structured generation

## Installation

```bash
go get github.com/recera/gai/providers/openai_compat
```

## Quick Start

### Using Preset Configurations

The easiest way to get started is using preset configurations for known providers:

```go
import "github.com/recera/gai/providers/openai_compat"

// Groq - Very fast inference
provider, err := openai_compat.Groq()

// xAI - Grok models
provider, err := openai_compat.XAI()

// Cerebras - Extremely fast but with some limitations
provider, err := openai_compat.Cerebras()

// Together - Wide range of open-source models
provider, err := openai_compat.Together()
```

### Custom Configuration

For custom deployments or unlisted providers:

```go
provider, err := openai_compat.New(openai_compat.CompatOpts{
    BaseURL:      "https://api.yourprovider.com/v1",
    APIKey:       os.Getenv("YOUR_API_KEY"),
    DefaultModel: "your-model-name",
    ProviderName: "yourprovider",
    
    // Optional quirks for provider limitations
    DisableJSONStreaming:     false,
    DisableParallelToolCalls: false,
    DisableStrictJSONSchema:  false,
    DisableToolChoice:        false,
})
```

## Provider-Specific Configurations

### Groq

Groq provides extremely fast inference with open-source models.

```go
provider, err := openai_compat.Groq(
    openai_compat.WithModel("llama-3.3-70b-versatile"), // default
    openai_compat.WithAPIKey(os.Getenv("GROQ_API_KEY")),
)

// Available models:
// - llama-3.3-70b-versatile (most capable)
// - llama-3.1-70b-versatile
// - llama-3.1-8b-instant (fastest)
// - mixtral-8x7b-32768
// - gemma2-9b-it
```

**Characteristics:**
- ✅ Very fast inference (< 1 second for most responses)
- ✅ Supports streaming, tools, and JSON mode
- ⚠️ Aggressive rate limits (use retry middleware)
- 💡 Best for: Real-time applications, chatbots

### xAI (Grok)

xAI provides access to the Grok family of models.

```go
provider, err := openai_compat.XAI(
    openai_compat.WithModel("grok-2-latest"), // default
)

// Available models:
// - grok-2-latest (most capable)
// - grok-2-1212
// - grok-beta
```

**Characteristics:**
- ✅ Large context windows
- ✅ Strong reasoning capabilities
- ✅ Full OpenAI compatibility
- 💡 Best for: Complex reasoning, analysis

### Cerebras

Cerebras offers the fastest inference available with custom hardware acceleration.

```go
provider, err := openai_compat.Cerebras(
    openai_compat.WithModel("llama-3.3-70b"), // default
)

// Available models:
// - llama-3.3-70b
// - llama-3.1-70b
// - llama-3.1-8b
```

**Characteristics:**
- ✅ Extremely fast inference (< 500ms typical)
- ❌ No JSON streaming support
- ❌ No parallel tool calls
- ⚠️ Strict rate limits
- 💡 Best for: Latency-critical applications

### Baseten

Baseten allows you to deploy and serve your own models.

```go
provider, err := openai_compat.Baseten(
    "https://model-abc123.api.baseten.co/v1",
    openai_compat.WithModel("your-deployed-model"),
    openai_compat.WithAPIKey(os.Getenv("BASETEN_API_KEY")),
)
```

**Characteristics:**
- ✅ Custom model deployments
- ✅ Full control over infrastructure
- ⚠️ Capabilities depend on deployed model
- 💡 Best for: Custom/fine-tuned models

### Together

Together provides access to a wide range of open-source models.

```go
provider, err := openai_compat.Together(
    openai_compat.WithModel("meta-llama/Llama-3.3-70B-Instruct-Turbo"),
)

// Popular models:
// - meta-llama/Llama-3.3-70B-Instruct-Turbo
// - meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo
// - mistralai/Mixtral-8x7B-Instruct-v0.1
// - NousResearch/Nous-Hermes-2-Mixtral-8x7B-DPO
```

**Characteristics:**
- ✅ Large model selection
- ✅ Good price/performance ratio
- ✅ Supports most OpenAI features
- 💡 Best for: Experimentation, cost optimization

## Usage Examples

### Basic Text Generation

```go
ctx := context.Background()

provider, err := openai_compat.Groq()
if err != nil {
    log.Fatal(err)
}

result, err := provider.GenerateText(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Explain quantum computing in simple terms."},
            },
        },
    },
    Temperature: 0.7,
    MaxTokens:   500,
})

if err != nil {
    log.Fatal(err)
}

fmt.Println(result.Text)
fmt.Printf("Tokens used: %d\n", result.Usage.TotalTokens)
```

### Streaming Responses

```go
stream, err := provider.StreamText(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Write a haiku about programming."},
            },
        },
    },
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
        fmt.Printf("\nTotal tokens: %d\n", event.Usage.TotalTokens)
    case core.EventError:
        log.Printf("Stream error: %v\n", event.Err)
    }
}
```

### Tool Calling

```go
// Define a weather tool
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
    func(ctx context.Context, in WeatherInput, meta tools.Meta) (WeatherOutput, error) {
        // Simulate weather API call
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
    ToolChoice: core.ToolAuto,
    StopWhen:   core.MaxSteps(3),
})
```

### Structured Outputs

```go
type Analysis struct {
    Summary      string   `json:"summary"`
    KeyPoints    []string `json:"key_points"`
    Sentiment    string   `json:"sentiment"`
    ActionItems  []string `json:"action_items,omitempty"`
}

result, err := provider.GenerateObject(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Analyze this customer feedback: The product is great but shipping was slow."},
            },
        },
    },
}, Analysis{})

if err != nil {
    log.Fatal(err)
}

analysis := result.Value.(*Analysis)
fmt.Printf("Sentiment: %s\n", analysis.Sentiment)
fmt.Printf("Key Points: %v\n", analysis.KeyPoints)
```

### Handling Provider Quirks

```go
// Custom configuration for a provider with limitations
provider, err := openai_compat.New(openai_compat.CompatOpts{
    BaseURL:      "https://api.limited-provider.com/v1",
    APIKey:       "your-key",
    ProviderName: "limited",
    
    // Disable unsupported features
    DisableJSONStreaming:     true,  // Provider doesn't support streaming JSON
    DisableParallelToolCalls: true,  // Can't execute tools in parallel
    DisableStrictJSONSchema:  true,  // No strict JSON schema support
    
    // Strip unsupported parameters
    UnsupportedParams: []string{"seed", "logit_bias"},
    
    // Custom retry behavior
    MaxRetries:  5,
    RetryDelay:  2 * time.Second,
})
```

## Error Handling

The adapter provides comprehensive error mapping to GAI's error taxonomy:

```go
result, err := provider.GenerateText(ctx, request)
if err != nil {
    // Check error type
    if core.IsRateLimited(err) {
        // Handle rate limiting
        if aiErr, ok := err.(*core.AIError); ok {
            fmt.Printf("Retry after: %v\n", aiErr.RetryAfter)
        }
    } else if core.IsContextLengthExceeded(err) {
        // Handle context length issues
        fmt.Println("Request too long, consider chunking")
    } else if core.IsTransient(err) {
        // Retry transient errors
        fmt.Println("Temporary error, retrying...")
    }
}
```

## Performance Optimization

### Connection Pooling

The adapter automatically manages connection pooling for optimal performance:

```go
provider, err := openai_compat.New(openai_compat.CompatOpts{
    BaseURL: "https://api.provider.com/v1",
    APIKey:  "key",
    HTTPClient: &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     90 * time.Second,
        },
    },
})
```

### Middleware Integration

Use GAI middleware for additional functionality:

```go
import "github.com/recera/gai/middleware"

// Add retry and rate limiting
provider = middleware.Chain(
    middleware.WithRetry(middleware.RetryOpts{
        MaxAttempts: 3,
        BaseDelay:   time.Second,
    }),
    middleware.WithRateLimit(middleware.RateLimitOpts{
        RPS:   10,
        Burst: 20,
    }),
)(provider)
```

## Benchmarks

Performance benchmarks on M1 MacBook Pro:

| Operation | Time | Memory | Allocations |
|-----------|------|--------|-------------|
| Provider Creation | 2.5μs | 1,856 B | 15 allocs |
| Request Conversion | 423ns | 624 B | 9 allocs |
| Message Conversion | 156ns | 384 B | 6 allocs |
| Error Mapping | 89ns | 128 B | 2 allocs |
| Parameter Stripping | 34ns | 0 B | 0 allocs |
| JSON Schema Generation | 1.2μs | 896 B | 12 allocs |

## Capabilities Detection

The adapter automatically probes provider capabilities:

```go
provider, err := openai_compat.New(opts)
// Wait for probing to complete (happens async)
time.Sleep(100 * time.Millisecond)

caps := provider.GetCapabilities()
fmt.Printf("Supports vision: %v\n", caps.SupportsVision)
fmt.Printf("Max context: %d tokens\n", caps.MaxContextWindow)
fmt.Printf("Available models: %v\n", caps.Models)
```

## Troubleshooting

### Common Issues

1. **401 Unauthorized**: Check your API key is set correctly
2. **404 Not Found**: Verify the base URL and model name
3. **429 Rate Limited**: Use retry middleware or reduce request rate
4. **413 Request Too Large**: Reduce prompt size or use a model with larger context
5. **500 Server Error**: Usually transient, automatic retry will handle it

### Debug Logging

Enable debug logging to see request/response details:

```go
provider, err := openai_compat.New(openai_compat.CompatOpts{
    BaseURL: "https://api.provider.com/v1",
    APIKey:  "key",
    CustomHeaders: map[string]string{
        "X-Debug": "true",
    },
})
```

## Contributing

Contributions are welcome! To add support for a new provider:

1. Test with the generic `New()` function first
2. Document any quirks or limitations
3. Add a preset configuration if the provider is popular
4. Submit a PR with tests

## License

This package is part of the GAI framework and follows the same license.