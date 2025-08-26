# Frequently Asked Questions (FAQ)

This document answers common questions about the GAI (Go AI) Framework.

## Table of Contents
- [General Questions](#general-questions)
- [Getting Started](#getting-started)
- [Providers and Models](#providers-and-models)
- [Tool Calling](#tool-calling)
- [Performance and Optimization](#performance-and-optimization)
- [Production Deployment](#production-deployment)
- [Advanced Features](#advanced-features)
- [Troubleshooting](#troubleshooting)

## General Questions

### What is the GAI Framework?

The GAI (Go AI) Framework is a production-ready Go library for building AI-powered applications. It provides:
- **Unified API** across multiple AI providers (OpenAI, Anthropic, Google, Groq, Ollama)
- **Type-safe operations** with strongly typed requests and responses
- **Advanced tool calling** with multi-step execution capabilities
- **Production features** including observability, rate limiting, and error handling

### Why choose GAI over other frameworks?

**Key advantages:**
- **Go-native**: Built specifically for Go with idiomatic patterns
- **Type safety**: Compile-time guarantees and structured data handling
- **Provider flexibility**: Easy switching between AI providers without code changes
- **Production ready**: Built-in observability, error handling, and middleware
- **Local deployment**: Full support for on-premises models via Ollama

### Is GAI suitable for production use?

Yes! GAI is designed for production with:
- Comprehensive error handling and retry mechanisms
- OpenTelemetry observability integration
- Rate limiting and circuit breaker patterns
- Graceful degradation and failover support
- Extensive testing and validation

### What Go version is required?

**Go 1.23+ is required** for:
- Advanced generics support used throughout the framework
- Latest language features and performance improvements
- Compatibility with modern Go ecosystem libraries

## Getting Started

### How do I install GAI?

```bash
# Initialize your Go module
go mod init your-project

# Install GAI
go get github.com/recera/gai@latest

# Install specific provider packages as needed
go get github.com/recera/gai/providers/openai
go get github.com/recera/gai/providers/anthropic
```

### What's the quickest way to get started?

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/recera/gai/core"
    "github.com/recera/gai/providers/openai"
)

func main() {
    provider := openai.New(openai.WithAPIKey("your-key"))
    
    result, err := provider.GenerateText(context.Background(), core.Request{
        Messages: []core.Message{
            {Role: core.User, Parts: []core.Part{core.Text{Text: "Hello!"}}},
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(result.Text)
}
```

### Do I need API keys for all providers?

No, you only need API keys for the providers you plan to use:
- **Required for cloud providers**: OpenAI, Anthropic, Google Gemini, Groq
- **Not required for local deployment**: Ollama (runs locally)
- **Mix and match**: Use multiple providers in the same application

### Can I use GAI without any external dependencies?

For local-only deployment with Ollama, you have minimal external dependencies:
```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Use only local models
provider := ollama.New(ollama.WithModel("llama3.2"))
```

## Providers and Models

### Which providers are supported?

| Provider | Models | Features |
|----------|--------|----------|
| **OpenAI** | GPT-4o, GPT-4o-mini, o1-preview, o1-mini | Tools, vision, structured output |
| **Anthropic** | Claude 3.5 Sonnet, Claude 3 Opus/Haiku | Tools, large context, structured output |
| **Google** | Gemini 1.5 Pro/Flash, Gemini 2.0 Flash | Tools, vision, large context |
| **Groq** | Llama 3.1/3.2, Mixtral, Gemma 2 | Ultra-fast inference, tools |
| **Ollama** | Llama, Mistral, CodeLlama, etc. | Local deployment, privacy, customization |

### How do I choose the right provider/model?

**Consider these factors:**

**For speed**: Groq (ultra-fast inference) or OpenAI GPT-4o-mini
```go
provider := groq.New(groq.WithModel("llama-3.1-70b-versatile"))
```

**For reasoning**: OpenAI o1-preview or Anthropic Claude 3.5 Sonnet
```go
provider := openai.New(openai.WithModel("o1-preview"))
```

**For large contexts**: Anthropic Claude or Google Gemini
```go
provider := anthropic.New(anthropic.WithModel("claude-3-5-sonnet-20241022"))
```

**For privacy**: Ollama with local models
```go
provider := ollama.New(ollama.WithModel("llama3.2"))
```

**For cost efficiency**: OpenAI GPT-4o-mini or Groq models
```go
provider := openai.New(openai.WithModel("gpt-4o-mini"))
```

### Can I switch providers without changing code?

Yes! That's a core feature of GAI:
```go
// Configuration-driven provider selection
func createProvider(providerType string, apiKey string) core.Provider {
    switch providerType {
    case "openai":
        return openai.New(openai.WithAPIKey(apiKey))
    case "anthropic":
        return anthropic.New(anthropic.WithAPIKey(apiKey))
    case "groq":
        return groq.New(groq.WithAPIKey(apiKey))
    default:
        return ollama.New()  // Local fallback
    }
}
```

### How do I handle different model capabilities?

Use feature detection and graceful degradation:
```go
// Check if provider supports tools
if len(result.Steps) == 0 && len(request.Tools) > 0 {
    // Fallback for providers without tool support
    return handleWithoutTools(ctx, provider, request)
}

// Check for vision capabilities
if hasImageParts(request.Messages) {
    // Use vision-capable model
    request.Model = "gpt-4o"  // or gemini-1.5-pro-vision
}
```

## Tool Calling

### What is tool calling and why use it?

Tool calling allows AI models to execute functions and use external services:
- **Extend capabilities**: Web search, database queries, API calls
- **Real-world integration**: Connect AI to your systems
- **Multi-step workflows**: Chain operations for complex tasks

### How do I create a tool?

```go
// Define input/output types
type WeatherInput struct {
    Location string `json:"location" jsonschema:"required,description=City name"`
}

type WeatherOutput struct {
    Temperature float64 `json:"temperature"`
    Condition   string  `json:"condition"`
}

// Create type-safe tool
weatherTool := tools.New[WeatherInput, WeatherOutput](
    "get_weather",
    "Get current weather for a location",
    func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
        // Call weather API
        temp, condition := getWeatherFromAPI(input.Location)
        return WeatherOutput{
            Temperature: temp,
            Condition:   condition,
        }, nil
    },
)
```

### Can tools call other tools?

Yes! GAI supports multi-step execution:
```go
request := core.Request{
    Messages: messages,
    Tools:    []core.ToolHandle{weatherTool, bookingTool, emailTool},
    StopWhen: core.MaxSteps(10),  // Prevent infinite loops
}

result, err := provider.GenerateText(ctx, request)
// AI can use multiple tools in sequence
```

### How do I control tool execution?

Use stop conditions for fine-grained control:
```go
// Stop after maximum steps
request.StopWhen = core.MaxSteps(5)

// Stop when specific tool is called
request.StopWhen = core.UntilToolSeen("final_answer")

// Stop when no more tools are needed
request.StopWhen = core.NoMoreTools()

// Combine conditions
request.StopWhen = core.CombineConditions(
    core.MaxSteps(10),
    core.UntilToolSeen("complete"),
)
```

### How do I handle tool errors?

```go
weatherTool := tools.New[WeatherInput, WeatherOutput](
    "get_weather",
    "Get weather",
    func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
        // Validate input
        if input.Location == "" {
            return WeatherOutput{}, errors.New("location is required")
        }
        
        // Handle API errors
        weather, err := weatherAPI.Get(input.Location)
        if err != nil {
            // Return structured error for the AI to handle
            return WeatherOutput{}, fmt.Errorf("weather service unavailable: %w", err)
        }
        
        return WeatherOutput{
            Temperature: weather.Temp,
            Condition:   weather.Condition,
        }, nil
    },
)
```

## Performance and Optimization

### How do I optimize for speed?

**Use streaming for real-time responses:**
```go
stream, err := provider.StreamText(ctx, request)
for event := range stream.Events() {
    if event.Type == core.EventTextDelta {
        fmt.Print(event.TextDelta)  // Display immediately
    }
}
```

**Choose fast providers/models:**
```go
// Ultra-fast inference with Groq
provider := groq.New(groq.WithModel("llama-3.1-8b-instant"))

// Or cost-effective speed with OpenAI
provider := openai.New(openai.WithModel("gpt-4o-mini"))
```

**Implement caching:**
```go
provider = middleware.Chain(
    middleware.WithCache(middleware.CacheOpts{
        TTL:      time.Hour,
        MaxItems: 1000,
    }),
)(provider)
```

### How do I reduce costs?

**Choose cost-effective models:**
```go
// Most cost-effective for general tasks
provider := openai.New(openai.WithModel("gpt-4o-mini"))

// Free tier options
provider := groq.New(groq.WithModel("llama-3.1-8b-instant"))

// Local deployment (no API costs)
provider := ollama.New(ollama.WithModel("llama3.2"))
```

**Optimize token usage:**
```go
request := core.Request{
    Messages:    messages,
    MaxTokens:   500,      // Limit response length
    Temperature: 0.1,      // More focused, shorter responses
}
```

**Monitor and track usage:**
```go
collector := obs.NewCollector(ctx, "openai", "gpt-4o-mini")
provider := openai.New(
    openai.WithMetricsCollector(collector),
)

// Track costs
usage := collector.GetUsage()
fmt.Printf("Total cost: %d microcents", usage.TotalCostMicrocents)
```

### How do I handle high concurrency?

**Use semaphores to limit concurrent requests:**
```go
semaphore := make(chan struct{}, 10)  // Max 10 concurrent

func processRequest(ctx context.Context, request core.Request) error {
    semaphore <- struct{}{}
    defer func() { <-semaphore }()
    
    result, err := provider.GenerateText(ctx, request)
    // Process result
    return err
}
```

**Configure rate limiting:**
```go
provider = middleware.Chain(
    middleware.WithRateLimit(middleware.RateLimitOpts{
        RPS:   10,
        Burst: 20,
    }),
)(provider)
```

## Production Deployment

### How do I configure GAI for production?

**Use environment-based configuration:**
```go
type Config struct {
    OpenAIKey     string        `env:"OPENAI_API_KEY,required"`
    Provider      string        `env:"AI_PROVIDER" envDefault:"openai"`
    Model         string        `env:"AI_MODEL" envDefault:"gpt-4o-mini"`
    MaxConcurrent int           `env:"MAX_CONCURRENT" envDefault:"10"`
    Timeout       time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
}
```

**Set up comprehensive middleware:**
```go
provider = middleware.Chain(
    middleware.WithRetry(middleware.RetryOpts{
        MaxAttempts: 3,
        BaseDelay:   time.Second,
        MaxDelay:    10 * time.Second,
    }),
    middleware.WithRateLimit(middleware.RateLimitOpts{
        RPS:   10,
        Burst: 20,
    }),
    middleware.WithCircuitBreaker(middleware.CircuitBreakerOpts{
        MaxFailures: 5,
        Timeout:     time.Minute,
    }),
)(provider)
```

### How do I set up monitoring?

**OpenTelemetry integration:**
```go
// Initialize tracing
tp := trace.NewTracerProvider(
    trace.WithBatcher(jaegerExporter),
)
otel.SetTracerProvider(tp)

// Initialize metrics
mp := metric.NewMeterProvider(
    metric.WithReader(prometheusReader),
)
otel.SetMeterProvider(mp)

// Use observability-enabled provider
collector := obs.NewCollector(ctx, "openai", "gpt-4o")
provider := openai.New(
    openai.WithMetricsCollector(collector),
)
```

**Custom metrics:**
```go
// Track application-specific metrics
requestDuration := promauto.NewHistogramVec(prometheus.HistogramOpts{
    Name: "ai_request_duration_seconds",
    Help: "AI request duration",
}, []string{"provider", "model", "tool_used"})

start := time.Now()
result, err := provider.GenerateText(ctx, request)
requestDuration.WithLabelValues("openai", "gpt-4o", "true").Observe(time.Since(start).Seconds())
```

### How do I handle secrets securely?

**Use secure secret management:**
```go
// AWS Secrets Manager
func getAPIKey(secretName string) (string, error) {
    sess := session.Must(session.NewSession())
    svc := secretsmanager.New(sess)
    
    input := &secretsmanager.GetSecretValueInput{
        SecretId: aws.String(secretName),
    }
    
    result, err := svc.GetSecretValue(input)
    return *result.SecretString, err
}

// HashiCorp Vault
func getVaultSecret(path string) (string, error) {
    client, err := api.NewClient(api.DefaultConfig())
    if err != nil {
        return "", err
    }
    
    secret, err := client.Logical().Read(path)
    return secret.Data["api_key"].(string), err
}
```

**Environment variable validation:**
```go
func validateConfig() error {
    required := []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY"}
    for _, key := range required {
        if os.Getenv(key) == "" {
            return fmt.Errorf("required environment variable %s not set", key)
        }
    }
    return nil
}
```

### How do I implement graceful shutdown?

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Handle signals
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-c
        log.Println("Shutting down gracefully...")
        cancel()
    }()
    
    // Your application logic
    err := runApplication(ctx)
    if err != nil && !errors.Is(err, context.Canceled) {
        log.Fatal(err)
    }
}
```

## Advanced Features

### How do I use structured outputs?

```go
type AnalysisResult struct {
    Summary     string   `json:"summary" jsonschema:"required"`
    KeyPoints   []string `json:"key_points" jsonschema:"required"`
    Confidence  float64  `json:"confidence" jsonschema:"minimum=0,maximum=1"`
    Recommended bool     `json:"recommended"`
}

result, err := provider.GenerateObject(ctx, request, AnalysisResult{})
if err != nil {
    return err
}

// Type-safe access
analysis := result.Value.(*AnalysisResult)
fmt.Printf("Summary: %s\n", analysis.Summary)
```

### How do I implement multi-agent systems?

```go
// Define specialized agents
type Agent struct {
    Name     string
    Provider core.Provider
    Tools    []core.ToolHandle
    Prompt   string
}

func createResearchAgent() Agent {
    return Agent{
        Name:     "researcher",
        Provider: openai.New(openai.WithModel("gpt-4o")),
        Tools:    []core.ToolHandle{webSearchTool, databaseTool},
        Prompt:   "You are a research specialist...",
    }
}

func createAnalystAgent() Agent {
    return Agent{
        Name:     "analyst",
        Provider: anthropic.New(anthropic.WithModel("claude-3-5-sonnet")),
        Tools:    []core.ToolHandle{calculatorTool, chartTool},
        Prompt:   "You are a data analyst...",
    }
}

// Coordinate between agents
func coordinateAgents(ctx context.Context, task string) error {
    researcher := createResearchAgent()
    analyst := createAnalystAgent()
    
    // Research phase
    researchResults, err := researcher.Execute(ctx, task)
    if err != nil {
        return err
    }
    
    // Analysis phase
    analysisTask := fmt.Sprintf("Analyze this research: %s", researchResults)
    _, err = analyst.Execute(ctx, analysisTask)
    return err
}
```

### How do I implement custom middleware?

```go
func WithCustomLogging() middleware.Middleware {
    return func(next core.Provider) core.Provider {
        return &loggingProvider{next: next}
    }
}

type loggingProvider struct {
    next core.Provider
}

func (p *loggingProvider) GenerateText(ctx context.Context, req core.Request) (*core.Response, error) {
    start := time.Now()
    log.Printf("Starting request to %T", p.next)
    
    resp, err := p.next.GenerateText(ctx, req)
    
    duration := time.Since(start)
    if err != nil {
        log.Printf("Request failed after %v: %v", duration, err)
    } else {
        log.Printf("Request completed in %v, tokens: %d", duration, resp.Usage.TotalTokens)
    }
    
    return resp, err
}
```

## Troubleshooting

### Where can I get help?

1. **Documentation**: Check the [docs](.) and [examples](../examples/)
2. **Issues**: Search [GitHub Issues](https://github.com/recera/gai/issues)
3. **Discussions**: Join [GitHub Discussions](https://github.com/recera/gai/discussions)
4. **Debug logs**: Enable `GAI_DEBUG=true` for detailed logging

### What information should I include in bug reports?

```bash
# Version information
go version
go list -m github.com/recera/gai

# Environment details
echo "OS: $(uname -a)"
echo "Provider: $AI_PROVIDER"
echo "Model: $AI_MODEL"

# Minimal reproduction code
# Error messages with stack traces
# Expected vs actual behavior
```

### How do I enable debug logging?

```bash
export GAI_DEBUG=true
export GAI_LOG_LEVEL=debug
```

Or in code:
```go
import "log/slog"

logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

// Use with middleware
provider = middleware.Chain(
    middleware.WithLogging(middleware.LoggingOpts{
        Logger: logger,
        LogRequests: true,
        LogResponses: true,
    }),
)(provider)
```

---

## Still have questions?

If you don't find your answer here:
1. Check the [Troubleshooting Guide](./troubleshooting.md)
2. Browse the [examples](../examples/) for implementation patterns
3. Search [existing issues](https://github.com/recera/gai/issues)
4. Create a [new issue](https://github.com/recera/gai/issues/new) with detailed information

We're here to help make your AI integration successful!