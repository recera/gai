# Observability Package

The `obs` package provides comprehensive observability features for the GAI framework, including OpenTelemetry-based distributed tracing, metrics collection, and usage accounting. It is designed with zero-overhead when observability is not configured.

## Features

- **Distributed Tracing**: OpenTelemetry-based tracing with automatic span creation for requests, steps, tools, and prompts
- **Metrics Collection**: Histograms, counters, and gauges for monitoring performance and usage
- **Usage Accounting**: Track token usage and estimate costs across providers and models
- **Zero Overhead**: When not configured, observability operations become no-ops with minimal performance impact
- **Provider Agnostic**: Works seamlessly with all AI providers in the framework

## Installation

The observability package is included with the GAI framework:

```go
import "github.com/recera/gai/obs"
```

## Quick Start

### Setting Up Tracing

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
)

// Create an exporter (stdout for development, Jaeger/OTLP for production)
exporter, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())

// Create a tracer provider
tp := trace.NewTracerProvider(
    trace.WithBatcher(exporter),
    trace.WithResource(resource.Default()),
)

// Set as global provider
otel.SetTracerProvider(tp)
obs.SetGlobalTracerProvider(tp)
```

### Setting Up Metrics

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
)

// Create a metric exporter
exporter, _ := stdoutmetric.New()

// Create a meter provider
mp := metric.NewMeterProvider(
    metric.WithReader(metric.NewPeriodicReader(exporter)),
)

// Set as global provider
otel.SetMeterProvider(mp)
obs.SetGlobalMeterProvider(mp)
```

### Using with Core Runner

The observability package integrates seamlessly with the core runner:

```go
// Create an integrated collector
collector := obs.NewIntegratedCollector(ctx, request)

// Use with runner
runner := core.NewRunner(provider, core.WithMetrics(collector))

// Execute request
result, err := runner.ExecuteRequest(ctx, request)

// Complete metrics collection
collector.Complete(err == nil, result.Usage, err)
```

## Tracing

### Request Spans

Track entire AI request lifecycles:

```go
ctx, span := obs.StartRequestSpan(ctx, obs.RequestSpanOptions{
    Provider:     "openai",
    Model:        "gpt-4",
    Temperature:  0.7,
    MaxTokens:    1000,
    Stream:       false,
    ToolCount:    2,
    MessageCount: 3,
})
defer span.End()

// Your AI request code here
obs.RecordUsage(span, inputTokens, outputTokens, totalTokens)
```

### Step Spans

Track individual steps in multi-step executions:

```go
ctx, span := obs.StartStepSpan(ctx, obs.StepSpanOptions{
    StepNumber:   1,
    HasToolCalls: true,
    ToolCount:    2,
    TextLength:   500,
})
defer span.End()
```

### Tool Spans

Track tool executions with detailed metrics:

```go
ctx, span := obs.StartToolSpan(ctx, obs.ToolSpanOptions{
    ToolName:   "get_weather",
    ToolID:     "tool_123",
    InputSize:  256,
    StepNumber: 1,
    Timeout:    30 * time.Second,
})
defer span.End()

// Execute tool
result, err := executeTool()

// Record result
obs.RecordToolResult(span, err == nil, len(result), duration)
```

### Prompt Spans

Track prompt rendering with caching information:

```go
ctx, span := obs.StartPromptSpan(ctx, obs.PromptSpanOptions{
    Name:        "assistant",
    Version:     "1.0.0",
    Fingerprint: "abc123",
    DataKeys:    []string{"user", "context"},
    Override:    false,
    CacheHit:    true,
})
defer span.End()
```

## Metrics

### Request Metrics

```go
// Record request completion
obs.RecordRequest(ctx, "openai", "gpt-4", success, duration)

// Record token usage
obs.RecordTokens(ctx, "openai", "gpt-4", inputTokens, outputTokens)
```

### Tool Metrics

```go
// Record tool execution
obs.RecordToolExecution(ctx, "get_weather", success, duration)
```

### Error Metrics

```go
// Record errors by type
obs.RecordErrorMetric(ctx, "rate_limited", "openai", "gpt-4")
```

### Streaming Metrics

```go
// Record streaming events
obs.RecordStreamEvent(ctx, "text_delta", "openai")
```

### Cache Metrics

```go
// Record cache hit/miss
obs.RecordCacheHit(ctx, "prompt", hit)
```

## Usage Accounting

Track and report token usage and costs:

```go
// Record usage data
obs.RecordUsageData(ctx, "openai", "gpt-4", inputTokens, outputTokens)

// Generate usage report
report := obs.GenerateReport()
fmt.Printf("Total requests: %d\n", report.TotalRequests)
fmt.Printf("Total tokens: %d\n", report.TotalTokens)
fmt.Printf("Estimated cost: %s\n", report.TotalCost)

for _, provider := range report.Providers {
    fmt.Printf("\n%s:\n", provider.Provider)
    fmt.Printf("  Requests: %d\n", provider.Requests)
    fmt.Printf("  Cost: %s\n", provider.Cost)
    
    for _, model := range provider.Models {
        fmt.Printf("    %s: %d requests, %s\n", 
            model.Model, model.Requests, model.Cost)
    }
}
```

### Cost Estimation

The package includes built-in cost estimation for popular models:

```go
// Estimate cost in microcents
cost := obs.EstimateCost("gpt-4", inputTokens, outputTokens)

// Format as dollar string
formatted := obs.FormatCost(cost) // e.g., "$0.09"
```

## Zero Overhead Design

When observability is not configured, all operations become no-ops:

```go
// Without provider configuration
obs.Tracer()  // Returns noop tracer
obs.Meter()   // Returns noop meter

// Benchmark results show minimal overhead:
// - Metrics disabled: 5.3ns, 0 allocations
// - Tracing disabled: 169ns, 6 allocations
```

## Integration Examples

### Complete Example with OpenAI

```go
package main

import (
    "context"
    "log"
    
    "github.com/recera/gai/core"
    "github.com/recera/gai/obs"
    "github.com/recera/gai/providers/openai"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
)

func main() {
    // Setup tracing
    exporter, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())
    tp := trace.NewTracerProvider(trace.WithBatcher(exporter))
    otel.SetTracerProvider(tp)
    obs.SetGlobalTracerProvider(tp)
    defer tp.Shutdown(context.Background())
    
    // Create provider
    provider := openai.New(openai.WithAPIKey("your-key"))
    
    // Create request
    request := core.Request{
        Model: "gpt-4",
        Messages: []core.Message{
            {Role: core.User, Parts: []core.Part{
                core.Text{Text: "Hello!"},
            }},
        },
    }
    
    // Create collector
    ctx := context.Background()
    collector := obs.NewIntegratedCollector(ctx, request)
    
    // Create runner with metrics
    runner := core.NewRunner(provider, core.WithMetrics(collector))
    
    // Execute
    result, err := runner.ExecuteRequest(ctx, request)
    
    // Complete collection
    collector.Complete(err == nil, result.Usage, err)
    
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Response: %s", result.Text)
    log.Printf("Tokens used: %d", result.Usage.TotalTokens)
}
```

### With Prompts Integration

```go
// Prompts automatically record observability data
reg := prompts.NewRegistry(embedFS)

// Render with automatic tracing
text, id, err := reg.Render(ctx, "assistant", "1.0.0", data)

// Prompt metadata is automatically attached to spans:
// - prompt.name
// - prompt.version  
// - prompt.fingerprint
// - prompt.cache_hit
```

### With Tools Integration

```go
// Tools automatically record execution metrics
tool := tools.New[Input, Output](
    "get_weather",
    "Get weather for a location",
    handler,
)

// Execution automatically creates spans and records metrics:
// - tool.name
// - tool.duration_ms
// - tool.success
// - tool.input_size
// - tool.output_size
```

## Performance

Benchmark results on Apple M4:

| Operation | Disabled | Enabled | Overhead |
|-----------|----------|---------|----------|
| Tracing | 169ns, 6 allocs | 1384ns, 20 allocs | 8.2x |
| Metrics | 5.3ns, 0 allocs | 2315ns, 28 allocs | 436x |

The large metrics overhead ratio is due to the extremely efficient no-op case (5ns). In absolute terms, the enabled overhead (2.3μs) is still minimal for production use.

## Best Practices

1. **Initialize Early**: Set up providers at application startup
2. **Use Context**: Always propagate context for proper span nesting
3. **Record Errors**: Use `RecordError()` for proper error tracking
4. **Complete Spans**: Always defer `span.End()` after starting a span
5. **Batch Metrics**: Use `RequestMetrics` and `ToolMetrics` for efficient recording
6. **Monitor Costs**: Use usage accounting to track and optimize costs

## Supported Metrics

### Request Metrics
- `ai.requests.total` - Total number of requests
- `ai.request.duration` - Request duration histogram
- `ai.requests.active` - Active requests gauge

### Token Metrics
- `ai.tokens.total` - Total tokens processed

### Tool Metrics
- `ai.tools.executions` - Tool execution counter
- `ai.tool.duration` - Tool duration histogram

### Error Metrics
- `ai.errors.total` - Error counter by type

### Cache Metrics
- `ai.cache.hit_ratio` - Cache hit ratio histogram

### Prompt Metrics
- `ai.prompt.render_duration` - Prompt rendering duration

## Troubleshooting

### No Traces Appearing

Ensure you've set the global tracer provider:
```go
obs.SetGlobalTracerProvider(tp)
```

### Missing Metrics

Initialize the meter before recording:
```go
obs.SetGlobalMeterProvider(mp)
_ = obs.Meter() // Initialize instruments
```

### High Memory Usage

Consider using batch exporters instead of simple exporters:
```go
tp := trace.NewTracerProvider(
    trace.WithBatcher(exporter), // Batch mode
)
```

## License

Part of the GAI framework - see main LICENSE file.