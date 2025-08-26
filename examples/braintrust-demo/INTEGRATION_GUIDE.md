# Braintrust + GAI Framework Integration Guide

This comprehensive guide explains how to integrate Braintrust observability with the GAI framework for production-ready AI applications.

## 🎯 Overview

The integration provides:
- **Real-time Tracing**: Every AI request, tool call, and multi-step workflow is tracked
- **GenAI Semantic Conventions**: Automatic mapping to OpenTelemetry GenAI standards
- **Cost Tracking**: Token usage and cost estimation per request and model
- **Performance Insights**: Latency, throughput, and efficiency metrics
- **Error Analytics**: Categorized error tracking and recovery patterns

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐    ┌──────────────────┐
│ GAI Application │────│ obs Package      │────│ OpenTelemetry   │────│ Braintrust       │
│                 │    │ (Observability)  │    │ OTLP Exporter   │    │ Dashboard        │
└─────────────────┘    └──────────────────┘    └─────────────────┘    └──────────────────┘
         │                        │                        │                        │
    ┌────▼────┐              ┌────▼────┐              ┌────▼────┐              ┌────▼────┐
    │ Core    │              │ Tracing │              │ HTTPS   │              │ LLM     │
    │ Request │              │ Metrics │              │ Export  │              │ Spans   │
    │ Tools   │              │ Usage   │              │ Batched │              │ Eval    │
    └─────────┘              └─────────┘              └─────────┘              └─────────┘
```

## 🚀 Quick Setup

### 1. Environment Configuration

Create a `.env` file in your project root:

```bash
# AI Provider Configuration
GROQ_API_KEY=your_groq_api_key_here

# Braintrust Configuration  
BRAINTRUST_API_KEY=your_braintrust_api_key_here
BRAINTRUST_PROJECT_ID=your_project_id_here
BRAINTRUST_PROJECT_NAME="Your Project Name"
```

### 2. Dependencies

Add to your `go.mod`:

```go
require (
    github.com/recera/gai v0.0.0
    go.opentelemetry.io/otel v1.37.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.37.0
    go.opentelemetry.io/otel/sdk v1.37.0
    go.opentelemetry.io/otel/trace v1.37.0
)
```

### 3. Initialize Braintrust Tracing

```go
func initBraintrustObservability() func() {
    // Create resource with service information
    res := resource.NewWithAttributes(
        "https://opentelemetry.io/schemas/1.26.0",
        attribute.String("service.name", "your-ai-service"),
        attribute.String("service.version", "1.0.0"),
        attribute.String("braintrust.project_id", os.Getenv("BRAINTRUST_PROJECT_ID")),
    )
    
    // Setup Braintrust OTLP trace exporter
    headers := map[string]string{
        "Authorization": "Bearer " + os.Getenv("BRAINTRUST_API_KEY"),
        "x-bt-parent":   "project_id:" + os.Getenv("BRAINTRUST_PROJECT_ID"),
    }
    
    exporter, err := otlptracehttp.New(
        context.Background(),
        otlptracehttp.WithEndpoint("api.braintrust.dev:443"),
        otlptracehttp.WithHeaders(headers),
        otlptracehttp.WithURLPath("/otel/v1/traces"),
    )
    if err != nil {
        log.Fatalf("Failed to create OTLP exporter: %v", err)
    }
    
    // Create trace provider with batching
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter,
            trace.WithBatchTimeout(5*time.Second), // Adjust for production
        ),
        trace.WithResource(res),
        trace.WithSampler(trace.TraceIDRatioBased(0.1)), // 10% sampling for production
    )
    
    // Set global providers
    otel.SetTracerProvider(tp)
    obs.SetGlobalTracerProvider(tp)
    
    // Return cleanup function
    return func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        tp.Shutdown(ctx)
    }
}
```

## 📊 Usage Patterns

### Basic Request Tracing

```go
func handleChatRequest(ctx context.Context, provider *groq.Provider, userMessage string) error {
    // Start request span with GenAI conventions
    ctx, span := obs.StartRequestSpan(ctx, obs.RequestSpanOptions{
        Provider:     "groq",
        Model:        "llama-3.3-70b-versatile",
        Temperature:  0.7,
        MessageCount: 2,
        SystemPrompt: true,
        Metadata: map[string]any{
            "user_id":    getCurrentUserID(),
            "session_id": getSessionID(),
        },
    })
    defer span.End()
    
    // Add GenAI semantic attributes
    span.SetAttributes(
        attribute.String("gen_ai.system", "gai-framework"),
        attribute.String("gen_ai.operation.name", "chat_completion"),
        attribute.String("gen_ai.request.model", "llama-3.3-70b-versatile"),
        attribute.String("gen_ai.prompt.0.role", "system"),
        attribute.String("gen_ai.prompt.0.content", "You are a helpful assistant"),
        attribute.String("gen_ai.prompt.1.role", "user"),
        attribute.String("gen_ai.prompt.1.content", userMessage),
    )
    
    // Execute request
    result, err := provider.GenerateText(ctx, core.Request{
        Model: "llama-3.3-70b-versatile",
        Temperature: 0.7,
        Messages: []core.Message{
            {Role: core.System, Parts: []core.Part{core.Text{Text: "You are a helpful assistant"}}},
            {Role: core.User, Parts: []core.Part{core.Text{Text: userMessage}}},
        },
    })
    
    if err != nil {
        obs.RecordError(span, err, "Chat completion failed")
        return err
    }
    
    // Record usage and completion
    obs.RecordUsage(span, result.Usage.InputTokens, result.Usage.OutputTokens, result.Usage.TotalTokens)
    span.SetAttributes(
        attribute.Int("gen_ai.usage.prompt_tokens", result.Usage.InputTokens),
        attribute.Int("gen_ai.usage.completion_tokens", result.Usage.OutputTokens),
        attribute.String("gen_ai.completion", result.Text),
    )
    
    return nil
}
```

### Multi-Step Tool Execution

```go
func executeAgentWorkflow(ctx context.Context, provider *groq.Provider) error {
    // Start agent workflow span
    ctx, span := obs.StartRequestSpan(ctx, obs.RequestSpanOptions{
        Provider:  "groq",
        Model:     "llama-3.3-70b-versatile", 
        ToolCount: 3,
        Metadata: map[string]any{
            "workflow_type": "research_agent",
            "max_steps":     5,
        },
    })
    defer span.End()
    
    // Set agent-specific attributes
    span.SetAttributes(
        attribute.String("agent.type", "research_assistant"),
        attribute.String("agent.task", "data_analysis"),
        attribute.StringSlice("gen_ai.tools", []string{"search", "calculator", "chart_generator"}),
    )
    
    // Create tools with observability
    searchTool := tools.New[SearchInput, SearchOutput]("search", "Search information", handleSearch)
    calcTool := tools.New[CalcInput, CalcOutput]("calculator", "Perform calculations", handleCalc)
    
    // Execute with automatic step tracing
    result, err := provider.GenerateText(ctx, core.Request{
        Model: "llama-3.3-70b-versatile",
        Messages: []core.Message{
            {Role: core.User, Parts: []core.Part{core.Text{Text: "Research market trends and analyze the data"}}},
        },
        Tools: []core.ToolHandle{
            NewToolAdapter(searchTool),
            NewToolAdapter(calcTool),
        },
        StopWhen: core.MaxSteps(5),
    })
    
    if err != nil {
        obs.RecordError(span, err, "Agent workflow failed")
        return err
    }
    
    // Record comprehensive metrics
    span.SetAttributes(
        attribute.Int("agent.steps_executed", len(result.Steps)),
        attribute.Bool("agent.task_completed", true),
        attribute.String("gen_ai.completion", result.Text),
    )
    
    return nil
}
```

### Error Handling and Recovery

```go
func handleWithRetry(ctx context.Context, provider *groq.Provider) error {
    ctx, span := obs.StartRequestSpan(ctx, obs.RequestSpanOptions{
        Provider: "groq",
        Model:    "llama-3.3-70b-versatile",
        Metadata: map[string]any{
            "retry_enabled": true,
            "max_retries":   3,
        },
    })
    defer span.End()
    
    for attempt := 1; attempt <= 3; attempt++ {
        result, err := provider.GenerateText(ctx, request)
        
        if err == nil {
            // Success - record metrics
            obs.RecordUsage(span, result.Usage.InputTokens, result.Usage.OutputTokens, result.Usage.TotalTokens)
            span.SetAttributes(
                attribute.Int("retry.successful_attempt", attempt),
                attribute.Bool("retry.success", true),
            )
            return nil
        }
        
        // Record error for this attempt
        obs.RecordError(span, err, fmt.Sprintf("Attempt %d failed", attempt))
        span.SetAttributes(
            attribute.Int("retry.failed_attempts", attempt),
            attribute.String("retry.last_error", err.Error()),
        )
        
        if attempt < 3 {
            backoffDuration := time.Duration(attempt) * time.Second
            time.Sleep(backoffDuration)
        }
    }
    
    span.SetAttributes(attribute.Bool("retry.exhausted", true))
    return fmt.Errorf("all retry attempts failed")
}
```

## 📈 Braintrust Dashboard Features

### LLM Span Recognition
Braintrust automatically converts spans with GenAI semantic conventions into LLM spans, providing:
- **Request/Response Pairs**: Clear input/output visualization
- **Token Usage**: Automatic cost calculation and trending
- **Model Performance**: Latency and throughput metrics
- **Quality Metrics**: Response quality and user feedback integration

### Trace Relationships  
Multi-step workflows show clear parent-child relationships:
- **Request Span**: Top-level AI request
- **Step Spans**: Individual reasoning steps
- **Tool Spans**: Function calls and executions
- **Error Spans**: Failure points and recovery attempts

### Custom Metrics
Track domain-specific metrics:
```go
span.SetAttributes(
    attribute.String("business.user_tier", "premium"),
    attribute.Float64("business.conversion_score", 0.85),
    attribute.StringSlice("business.features_used", []string{"search", "analysis"}),
)
```

## 🛠️ Production Considerations

### Sampling Strategy
```go
// Use probabilistic sampling for high volume
trace.WithSampler(trace.TraceIDRatioBased(0.01)) // 1% sampling

// Or use custom sampling logic
trace.WithSampler(trace.ParentBased(
    trace.TraceIDRatioBased(0.1), // 10% root sampling
))
```

### Batch Configuration
```go
trace.WithBatcher(exporter,
    trace.WithBatchTimeout(10*time.Second),     // Batch every 10s
    trace.WithExportTimeout(30*time.Second),    // Export timeout
    trace.WithMaxExportBatchSize(512),          // Max batch size
)
```

### Error Handling
```go
// Monitor export failures
go func() {
    for range time.Tick(1 * time.Minute) {
        // Check if traces are being exported successfully
        // Set up alerts if export failures exceed threshold
    }
}()
```

### Resource Management
```go
// Set appropriate resource limits
res := resource.NewWithAttributes(
    "https://opentelemetry.io/schemas/1.26.0",
    attribute.String("service.name", serviceName),
    attribute.String("service.version", version),
    attribute.String("deployment.environment", env),
    attribute.String("k8s.pod.name", podName),
    attribute.String("k8s.namespace.name", namespace),
)
```

## 🔍 Debugging and Monitoring

### Common Issues

**No traces in Braintrust:**
- Verify API key and project ID
- Check network connectivity to `api.braintrust.dev`
- Ensure spans have proper GenAI attributes
- Monitor export errors in logs

**Missing tool executions:**
- Confirm tools implement proper interfaces
- Check tool adapter configuration
- Verify step span creation

**High latency:**
- Adjust batch timeout and size
- Use async export patterns
- Monitor Braintrust API response times

### Verification Script
```go
func verifyIntegration() {
    ctx := context.Background()
    _, span := obs.StartRequestSpan(ctx, obs.RequestSpanOptions{
        Provider: "test",
        Model:    "test-model",
        Metadata: map[string]any{"test": true},
    })
    
    span.SetAttributes(
        attribute.String("gen_ai.system", "gai-framework"),
        attribute.String("gen_ai.operation.name", "integration_test"),
    )
    
    span.End()
    
    log.Println("Test span created - check Braintrust dashboard")
    time.Sleep(5 * time.Second) // Allow export
}
```

## 📚 Advanced Features

### A/B Testing Integration
```go
span.SetAttributes(
    attribute.String("experiment.name", "model_comparison_v2"),
    attribute.String("experiment.variant", "llama3.3_vs_gpt4"),
    attribute.String("experiment.user_segment", "premium_users"),
)
```

### Custom Evaluations
```go
// Add evaluation metadata for Braintrust scoring
span.SetAttributes(
    attribute.Float64("eval.accuracy_score", 0.92),
    attribute.Float64("eval.relevance_score", 0.88),
    attribute.String("eval.human_feedback", "helpful"),
)
```

### Cost Tracking
```go
// Detailed cost attribution
obs.RecordUsageData(ctx, "groq", "llama-3.3-70b-versatile", inputTokens, outputTokens)
span.SetAttributes(
    attribute.String("cost.center", "research_team"),
    attribute.String("cost.project", "q4_analysis"),
    attribute.Float64("cost.estimated_usd", estimatedCost),
)
```

## 🎉 Next Steps

1. **Deploy the Demo**: Run the included demo to verify integration
2. **Customize Metadata**: Add business-specific attributes and metrics  
3. **Set Up Alerts**: Configure alerts for error rates and cost thresholds
4. **Create Dashboards**: Build custom dashboards for your use cases
5. **Implement Evaluations**: Set up automated quality scoring
6. **Scale Configuration**: Adjust sampling and batching for your volume

The integration provides comprehensive observability for your AI applications, enabling you to monitor performance, track costs, and improve quality at scale.