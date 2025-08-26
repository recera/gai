# Troubleshooting Guide

This comprehensive guide helps resolve common issues when using the GAI framework.

## Table of Contents
- [Installation Issues](#installation-issues)
- [Authentication Problems](#authentication-problems)
- [Provider-Specific Issues](#provider-specific-issues)
- [Tool Calling Problems](#tool-calling-problems)
- [Performance Issues](#performance-issues)
- [Error Handling](#error-handling)
- [Deployment Issues](#deployment-issues)
- [Debugging Tips](#debugging-tips)

## Installation Issues

### Go Version Compatibility

**Problem**: Build fails with generic-related errors
```
type instantiation error: type arguments do not satisfy constraints
```

**Solution**: Ensure you're using Go 1.23+
```bash
go version  # Should be 1.23.0 or higher
go get -u github.com/recera/gai@latest
```

**Problem**: Module resolution errors
```
go: cannot find main module; see 'go help modules'
```

**Solution**: Initialize Go module
```bash
go mod init your-project-name
go mod tidy
```

### Dependency Issues

**Problem**: Conflicting dependencies
```bash
go clean -modcache
go mod tidy
```

**Problem**: OpenTelemetry version conflicts
```bash
# Check versions
go list -m all | grep opentelemetry

# Update to compatible versions
go get go.opentelemetry.io/otel@latest
```

## Authentication Problems

### OpenAI API Issues

**Problem**: Invalid API key error
```
Error: invalid API key provided
```

**Solution**: Verify API key format and permissions
```bash
# Check key format (should start with sk-)
echo $OPENAI_API_KEY

# Test with curl
curl -H "Authorization: Bearer $OPENAI_API_KEY" \
  https://api.openai.com/v1/models
```

**Problem**: Insufficient credits/quota exceeded
```bash
# Check usage and billing in OpenAI dashboard
# Set usage limits programmatically
provider := openai.New(
    openai.WithAPIKey(apiKey),
    openai.WithMaxTokens(1000), // Limit per request
)
```

### Anthropic API Issues

**Problem**: Invalid API key format
```
Error: authentication_error
```

**Solution**: Verify Anthropic key format (starts with `sk-ant-`)
```bash
export ANTHROPIC_API_KEY="sk-ant-api03-..."
```

### Google Gemini Issues

**Problem**: API key not working
```bash
# Ensure key is enabled for Gemini API
# Check in Google Cloud Console
export GOOGLE_API_KEY="your-key"
```

### Groq API Issues

**Problem**: Rate limiting on free tier
```bash
# Use built-in rate limiting
provider = middleware.Chain(
    middleware.WithRateLimit(middleware.RateLimitOpts{
        RPS:   1,  // Lower rate for free tier
        Burst: 5,
    }),
)(provider)
```

## Provider-Specific Issues

### Ollama Issues

**Problem**: Connection refused
```
Error: connect: connection refused
```

**Solution**: Start Ollama service
```bash
# Start Ollama server
ollama serve

# Or if using Docker
docker run -d -v ollama:/root/.ollama -p 11434:11434 ollama/ollama

# Test connection
curl http://localhost:11434/api/tags
```

**Problem**: Model not found
```bash
# List available models
ollama list

# Pull required model
ollama pull llama3.2

# For specific model versions
ollama pull llama3.2:8b-instruct-q4_0
```

**Problem**: Out of memory errors
```bash
# Check system memory
free -h

# Use smaller models
ollama pull llama3.2:1b  # 1B parameter model

# Or configure memory usage
export OLLAMA_MAX_LOADED_MODELS=1
export OLLAMA_MAX_QUEUE=2
```

### OpenAI Issues

**Problem**: Model deprecation warnings
```bash
# Use latest models
provider := openai.New(
    openai.WithModel("gpt-4o-mini"),  # Latest stable
)
```

**Problem**: Context length exceeded
```bash
# Calculate token usage
inputTokens := countTokens(messages)
if inputTokens > 128000 {
    // Split into chunks or summarize
}

# Set explicit limits
request.MaxTokens = 4000  # Leave room for response
```

### Anthropic Issues

**Problem**: Claude model access
```bash
# Ensure you have access to Claude models
provider := anthropic.New(
    anthropic.WithModel("claude-3-5-sonnet-20241022"),
)
```

**Problem**: Message format issues
```
Error: invalid message format
```

**Solution**: Ensure proper message structure
```go
messages := []core.Message{
    {
        Role: core.System,
        Parts: []core.Part{core.Text{Text: "You are helpful assistant"}},
    },
    {
        Role: core.User,
        Parts: []core.Part{core.Text{Text: "Hello"}},
    },
}
```

## Tool Calling Problems

### Schema Validation Issues

**Problem**: Tool input schema mismatch
```
Error: invalid tool input schema
```

**Solution**: Verify JSON schema tags
```go
type WeatherInput struct {
    Location string `json:"location" jsonschema:"required,description=City name"`
    Units    string `json:"units,omitempty" jsonschema:"enum=celsius,enum=fahrenheit"`
}
```

**Problem**: Tool execution timeouts
```go
// Set explicit timeouts
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Or configure per-tool
weatherTool := tools.New[WeatherInput, WeatherOutput](
    "get_weather",
    "Get weather",
    func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
        // Add timeout handling
        select {
        case <-ctx.Done():
            return WeatherOutput{}, ctx.Err()
        default:
            // Tool logic
        }
    },
)
```

### Multi-Step Execution Issues

**Problem**: Infinite loops in multi-step execution
```go
// Use stop conditions
request := core.Request{
    Messages: messages,
    Tools:    tools,
    StopWhen: core.CombineConditions(
        core.MaxSteps(5),
        core.UntilToolSeen("final_answer"),
    ),
}
```

**Problem**: Tool results not being used
```bash
# Enable debug logging to trace execution
export GAI_DEBUG=true
```

## Performance Issues

### Slow Response Times

**Problem**: High latency responses

**Solution**: Use streaming and caching
```go
// Enable streaming
request.Stream = true
stream, err := provider.StreamText(ctx, request)

// Add caching middleware
provider = middleware.Chain(
    middleware.WithCache(cacheOpts),
    middleware.WithRetry(retryOpts),
)(provider)
```

### Memory Usage

**Problem**: High memory consumption
```go
// Limit concurrent requests
semaphore := make(chan struct{}, 5)  // Max 5 concurrent

// Use context cancellation
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
```

### Token Optimization

**Problem**: High token usage and costs
```go
// Set conservative limits
request := core.Request{
    Messages:    messages,
    MaxTokens:   1000,     // Limit response
    Temperature: 0.1,      // More focused responses
}

// Use cheaper models when appropriate
provider := openai.New(
    openai.WithModel("gpt-4o-mini"),  // Cost-effective
)
```

## Error Handling

### Comprehensive Error Handling

```go
result, err := provider.GenerateText(ctx, request)
if err != nil {
    // Check error type
    if aiErr, ok := err.(*core.AIError); ok {
        switch aiErr.Code {
        case core.ErrRateLimited:
            // Wait and retry
            backoff := time.Duration(aiErr.RetryAfter) * time.Second
            time.Sleep(backoff)
            return retryRequest(ctx, request)
            
        case core.ErrInvalidRequest:
            // Fix request and retry
            log.Printf("Invalid request: %s", aiErr.Message)
            return nil, fmt.Errorf("request validation failed: %w", err)
            
        case core.ErrProviderError:
            // Provider-specific error
            if aiErr.Temporary {
                return retryWithBackoff(ctx, request)
            }
            return nil, err
            
        case core.ErrContentFiltered:
            // Content policy violation
            log.Printf("Content filtered: %s", aiErr.Message)
            return nil, err
        }
    }
    
    // Handle context errors
    if errors.Is(err, context.DeadlineExceeded) {
        return nil, fmt.Errorf("request timeout: %w", err)
    }
    
    if errors.Is(err, context.Canceled) {
        return nil, fmt.Errorf("request canceled: %w", err)
    }
    
    return nil, err
}
```

### Network Issues

**Problem**: Intermittent network failures
```go
// Use robust retry configuration
provider = middleware.Chain(
    middleware.WithRetry(middleware.RetryOpts{
        MaxAttempts: 5,
        BaseDelay:   time.Second,
        MaxDelay:    30 * time.Second,
        Jitter:      true,
    }),
)(provider)
```

## Deployment Issues

### Production Configuration

**Problem**: Configuration management
```go
// Use environment-based configuration
type Config struct {
    OpenAIKey      string `env:"OPENAI_API_KEY,required"`
    AnthropicKey   string `env:"ANTHROPIC_API_KEY"`
    LogLevel       string `env:"LOG_LEVEL" envDefault:"info"`
    MaxConcurrent  int    `env:"MAX_CONCURRENT" envDefault:"10"`
    RequestTimeout time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
}
```

### Monitoring Setup

**Problem**: Lack of observability
```go
// Setup comprehensive monitoring
func setupObservability() func() {
    // Initialize OpenTelemetry
    tp := trace.NewTracerProvider(
        trace.WithBatcher(jaegerExporter),
    )
    otel.SetTracerProvider(tp)
    
    // Setup metrics
    mp := metric.NewMeterProvider(
        metric.WithReader(prometheusExporter),
    )
    otel.SetMeterProvider(mp)
    
    return func() {
        tp.Shutdown(context.Background())
        mp.Shutdown(context.Background())
    }
}
```

### Resource Management

**Problem**: Resource leaks
```go
// Always close streams
stream, err := provider.StreamText(ctx, request)
if err != nil {
    return err
}
defer stream.Close()  // Critical: always close

// Use context cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
```

## Debugging Tips

### Enable Debug Logging

```bash
# Environment variables
export GAI_DEBUG=true
export GAI_LOG_LEVEL=debug

# Or in code
import "log/slog"

logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```

### Request/Response Inspection

```go
// Log requests and responses
provider = middleware.Chain(
    middleware.WithLogging(middleware.LoggingOpts{
        LogRequests:  true,
        LogResponses: true,
        Logger:       logger,
    }),
)(provider)
```

### Performance Profiling

```go
import _ "net/http/pprof"
import "net/http"

// Enable pprof endpoint
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

// Usage: go tool pprof http://localhost:6060/debug/pprof/heap
```

### Tool Execution Tracing

```go
// Custom tool with detailed logging
weatherTool := tools.New[WeatherInput, WeatherOutput](
    "get_weather",
    "Get weather",
    func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
        start := time.Now()
        defer func() {
            log.Printf("Tool %s executed in %v", meta.CallID, time.Since(start))
        }()
        
        log.Printf("Tool called: %s with input: %+v", meta.CallID, input)
        
        // Tool logic
        result := WeatherOutput{...}
        
        log.Printf("Tool result: %+v", result)
        return result, nil
    },
)
```

## Getting Help

If you're still experiencing issues:

1. **Check the logs**: Enable debug logging for detailed error information
2. **Review examples**: Look at the [examples directory](../examples/) for working implementations
3. **Search issues**: Check [GitHub Issues](https://github.com/recera/gai/issues) for similar problems
4. **Create an issue**: Provide detailed information including:
   - Go version (`go version`)
   - GAI version (`go list -m github.com/recera/gai`)
   - Full error messages
   - Minimal reproduction code
   - Environment details (OS, provider used, etc.)

## Common Environment Variables

```bash
# Core configuration
export GAI_DEBUG=true
export GAI_LOG_LEVEL=debug
export GAI_DEFAULT_TIMEOUT=30s

# Provider API keys
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export GOOGLE_API_KEY="..."
export GROQ_API_KEY="gsk_..."

# Ollama configuration
export OLLAMA_HOST=http://localhost:11434
export OLLAMA_MAX_LOADED_MODELS=1

# Performance tuning
export GOMAXPROCS=4
export GOGC=100
```

Remember: Most issues are related to configuration, authentication, or network connectivity. Always verify these basics before diving into complex debugging.