# Ollama Provider

The Ollama provider enables seamless integration with locally hosted language models through [Ollama](https://ollama.ai), providing high-performance local AI capabilities with full privacy and control.

## Features

- **Local Model Support**: Run popular models locally (Llama, Mistral, CodeLlama, etc.)
- **Streaming Generation**: Real-time text streaming for responsive applications
- **Tool Calling**: Execute functions and tools during model conversations
- **Structured Outputs**: Generate JSON objects conforming to schemas
- **Multimodal Support**: Handle text and images in conversations
- **Model Management**: List, check availability, and pull models
- **Flexible APIs**: Support for both Chat API and Generate API
- **High Performance**: Optimized for local inference with minimal latency

## Quick Start

### Prerequisites

1. Install [Ollama](https://ollama.ai) on your system
2. Start the Ollama server: `ollama serve`
3. Pull a model: `ollama pull llama3.2`

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/recera/gai/core"
    "github.com/recera/gai/providers/ollama"
)

func main() {
    // Create provider with default settings (http://localhost:11434)
    provider := ollama.New(
        ollama.WithModel("llama3.2"),
        ollama.WithBaseURL("http://localhost:11434"), // optional, this is the default
    )

    // Simple text generation
    req := core.Request{
        Messages: []core.Message{
            {Role: core.User, Parts: []core.Part{core.Text{Text: "Hello! How are you?"}}},
        },
        Temperature: 0.7,
        MaxTokens:   100,
    }

    result, err := provider.GenerateText(context.Background(), req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Response: %s\n", result.Text)
    fmt.Printf("Usage: %+v\n", result.Usage)
}
```

### Streaming Example

```go
// Stream text generation for real-time responses
stream, err := provider.StreamText(context.Background(), req)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for event := range stream.Events() {
    switch event.Type {
    case core.EventTextDelta:
        fmt.Print(event.TextDelta)
    case core.EventFinish:
        fmt.Printf("\nUsage: %+v\n", event.Usage)
    case core.EventError:
        fmt.Printf("Error: %v\n", event.Err)
    }
}
```

## Configuration Options

### Provider Options

```go
provider := ollama.New(
    // Basic configuration
    ollama.WithBaseURL("http://localhost:11434"),  // Ollama server URL
    ollama.WithModel("llama3.2"),                  // Default model
    ollama.WithKeepAlive("10m"),                   // Model memory duration
    
    // API selection
    ollama.WithGenerateAPI(false),                 // Use /api/chat (default) or /api/generate
    ollama.WithTemplate("custom template"),        // Custom prompt template
    
    // HTTP configuration
    ollama.WithHTTPClient(customClient),           // Custom HTTP client
    ollama.WithMaxRetries(3),                      // Retry attempts
    ollama.WithRetryDelay(100*time.Millisecond),   // Retry delay
    
    // Observability
    ollama.WithMetricsCollector(collector),        // Metrics collection
)
```

### Model Parameters

Configure model behavior using provider options:

```go
req := core.Request{
    Messages: messages,
    Temperature: 0.8,
    MaxTokens: 200,
    ProviderOptions: map[string]any{
        "ollama": map[string]any{
            // Sampling parameters
            "top_k":           40,
            "top_p":           0.9,
            "repeat_penalty":  1.1,
            "seed":           42,
            
            // Generation parameters
            "num_ctx":        4096,  // Context window size
            "num_gpu":        1,     // GPU layers to use
            "low_vram":       true,  // Low VRAM mode
            
            // Advanced parameters
            "frequency_penalty": 0.1,
            "presence_penalty":  0.1,
            "mirostat":         1,
            "mirostat_eta":     0.1,
            "mirostat_tau":     5.0,
            
            // Stop sequences
            "stop": []string{".", "!", "?"},
        },
    },
}
```

## Tool Calling

Enable function calling for enhanced model capabilities:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/recera/gai/core"
    "github.com/recera/gai/providers/ollama"
    "github.com/recera/gai/tools"
)

// Define tool input/output types
type WeatherInput struct {
    Location string `json:"location" description:"City name"`
}

type WeatherOutput struct {
    Temperature float64 `json:"temperature"`
    Condition   string  `json:"condition"`
    Humidity    float64 `json:"humidity"`
}

// Implement the weather tool
func getWeather(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
    // Simulate weather API call
    return WeatherOutput{
        Temperature: 22.5,
        Condition:   "sunny",
        Humidity:    0.65,
    }, nil
}

func main() {
    provider := ollama.New(ollama.WithModel("llama3.1")) // Tool calling requires compatible model
    
    // Create the tool
    weatherTool := tools.New(
        "get_weather",
        "Get current weather for a location",
        getWeather,
    )

    req := core.Request{
        Messages: []core.Message{
            {Role: core.User, Parts: []core.Part{core.Text{Text: "What's the weather like in Paris?"}}},
        },
        Tools: []core.ToolHandle{weatherTool},
        StopWhen: core.NoMoreTools(), // Stop when no more tool calls are made
    }

    result, err := provider.GenerateText(context.Background(), req)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Final response: %s\n", result.Text)
    for i, step := range result.Steps {
        fmt.Printf("Step %d: %s\n", i+1, step.Text)
        for _, toolCall := range step.ToolCalls {
            fmt.Printf("  Tool: %s\n", toolCall.Name)
        }
    }
}
```

## Structured Output

Generate JSON objects that conform to specific schemas:

```go
// Define the schema
schema := map[string]any{
    "type": "object",
    "properties": map[string]any{
        "name": map[string]any{"type": "string"},
        "age":  map[string]any{"type": "integer"},
        "skills": map[string]any{
            "type": "array",
            "items": map[string]any{"type": "string"},
        },
    },
    "required": []string{"name", "age"},
}

req := core.Request{
    Messages: []core.Message{
        {Role: core.User, Parts: []core.Part{core.Text{Text: "Generate a software developer profile"}}},
    },
}

result, err := provider.GenerateObject(context.Background(), req, schema)
if err != nil {
    panic(err)
}

// Access the parsed object
profile := result.Value.(map[string]any)
fmt.Printf("Name: %s\n", profile["name"])
fmt.Printf("Age: %.0f\n", profile["age"])
```

## Multimodal Support

Handle images alongside text in conversations:

```go
req := core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "What do you see in this image?"},
                core.ImageURL{URL: "data:image/jpeg;base64,/9j/4AAQ..."}, // Base64 encoded image
            },
        },
    },
}

result, err := provider.GenerateText(context.Background(), req)
if err != nil {
    panic(err)
}

fmt.Printf("Image description: %s\n", result.Text)
```

## Model Management

Manage your local Ollama models programmatically:

```go
// List available models
models, err := provider.ListModels(context.Background())
if err != nil {
    panic(err)
}

for _, model := range models {
    fmt.Printf("Model: %s (size: %d bytes)\n", model.Name, model.Size)
}

// Check if a model is available
available, err := provider.IsModelAvailable(context.Background(), "llama3.2")
if err != nil {
    panic(err)
}

if !available {
    fmt.Println("Model not found, pulling...")
    err = provider.PullModel(context.Background(), "llama3.2")
    if err != nil {
        panic(err)
    }
}
```

## Error Handling

The Ollama provider maps various error conditions to standardized error types:

```go
result, err := provider.GenerateText(context.Background(), req)
if err != nil {
    // Check specific error types
    if ollama.IsModelNotFoundError(err) {
        fmt.Println("Model not available locally")
    } else if ollama.IsInsufficientMemoryError(err) {
        fmt.Println("Not enough memory to load model")
    } else if ollama.IsContextLengthExceededError(err) {
        fmt.Println("Input too long for model context")
    } else {
        fmt.Printf("Other error: %v\n", err)
    }
}
```

## Performance Tuning

### Memory Management

```go
// Configure memory usage
req := core.Request{
    Messages: messages,
    ProviderOptions: map[string]any{
        "ollama": map[string]any{
            "num_gpu":   1,      // Use GPU acceleration
            "low_vram":  true,   // Enable low VRAM mode
            "num_ctx":   2048,   // Reduce context size if needed
        },
    },
}

// Set keep-alive for better performance with multiple requests
provider := ollama.New(
    ollama.WithKeepAlive("30m"), // Keep model in memory for 30 minutes
)
```

### Concurrent Requests

The provider is safe for concurrent use:

```go
// Multiple goroutines can use the same provider instance
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        req := core.Request{
            Messages: []core.Message{
                {Role: core.User, Parts: []core.Part{core.Text{Text: fmt.Sprintf("Request %d", id)}}},
            },
        }
        result, err := provider.GenerateText(context.Background(), req)
        if err != nil {
            fmt.Printf("Error in goroutine %d: %v\n", id, err)
            return
        }
        fmt.Printf("Response %d: %s\n", id, result.Text)
    }(i)
}
wg.Wait()
```

## API Reference

### Chat API vs Generate API

The provider supports both Ollama APIs:

- **Chat API** (`/api/chat`): Default, supports conversation history and tools
- **Generate API** (`/api/generate`): Simple text completion, useful for some models

```go
// Use Chat API (default)
provider := ollama.New()

// Use Generate API
provider := ollama.New(ollama.WithGenerateAPI(true))
```

### Request Types

| Type | Chat API | Generate API | Streaming | Tools | Structured Output |
|------|----------|--------------|-----------|-------|-------------------|
| GenerateText | ✓ | ✓ | ✗ | ✓ | ✗ |
| StreamText | ✓ | ✓ | ✓ | ✓ | ✗ |
| GenerateObject | ✓ | ✓ | ✗ | ✗ | ✓ |
| StreamObject | ✓ | ✓ | ✓ | ✗ | ✓ |

## Supported Models

The provider works with any model available in Ollama. Popular models include:

- **Llama 3.2**: `llama3.2`, `llama3.2:1b`, `llama3.2:3b`
- **Llama 3.1**: `llama3.1:8b`, `llama3.1:70b`, `llama3.1:405b`
- **Mistral**: `mistral`, `mistral-nemo`
- **Code Models**: `codellama`, `deepseek-coder`
- **Specialized**: `phi3`, `gemma2`, `qwen2.5`

For tool calling, use models that explicitly support it:
- Llama 3.1 and 3.2 series
- Mistral Nemo
- Firefunction v2

## Troubleshooting

### Common Issues

1. **Connection Refused**
   ```bash
   # Start Ollama server
   ollama serve
   ```

2. **Model Not Found**
   ```bash
   # Pull the model first
   ollama pull llama3.2
   ```

3. **Out of Memory**
   - Use smaller models or reduce context size
   - Enable low VRAM mode
   - Reduce `num_ctx` parameter

4. **Slow Performance**
   - Ensure GPU acceleration is working
   - Increase `keep_alive` duration
   - Use appropriate model size for your hardware

### Debug Mode

Enable verbose logging to debug issues:

```go
// Custom HTTP client with logging
client := &http.Client{
    Transport: &loggingTransport{http.DefaultTransport},
    Timeout:   60 * time.Second,
}

provider := ollama.New(
    ollama.WithHTTPClient(client),
    ollama.WithMaxRetries(1), // Reduce retries for faster debugging
)
```

## Contributing

We welcome contributions! Please see the main repository's contributing guidelines.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/recera/gai.git
cd gai/providers/ollama

# Run tests (requires Ollama running)
go test -v

# Run benchmarks
go test -bench=. -benchmem

# Run integration tests with live Ollama
OLLAMA_TEST_LIVE=1 go test -v -run TestIntegrationLive
```

## License

This package is part of the GAI framework and is licensed under the Apache 2.0 License.