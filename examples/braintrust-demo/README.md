# Braintrust Integration Demo

This demo showcases comprehensive integration between the GAI framework and Braintrust for AI observability, tracing, and analytics.

## Features Demonstrated

🧠 **Braintrust Integration**
- OpenTelemetry traces sent to Braintrust via OTLP
- GenAI semantic conventions for LLM observability  
- Comprehensive span relationships and nesting
- Usage tracking and cost estimation

🔧 **GAI Framework Capabilities**
- Multi-step tool execution with automatic tracing
- Complex AI agent workflows
- Error handling and recovery patterns
- Performance monitoring and metrics

🤖 **Groq Provider**  
- High-speed inference with Llama 3.3 70B
- Tool calling and function execution
- Streaming and batched processing
- Model-specific optimizations

## Quick Start

1. **Setup Environment**:
   ```bash
   # Copy your existing .env or create from template
   cp .env.example .env
   # Edit .env with your actual API keys
   ```

2. **Install Dependencies**:
   ```bash
   go mod tidy
   ```

3. **Run Demo**:
   ```bash
   go run main.go
   ```

4. **View Results**:
   - Check your Braintrust dashboard at https://www.braintrust.dev/app
   - Look for traces in your configured project
   - Explore the span relationships and GenAI attributes

## Demo Scenarios

### 1. Simple Chat (💬)
- Basic text generation with full observability
- GenAI semantic conventions mapping
- Token usage and cost tracking

### 2. Multi-Step Tools (🔧) 
- Weather lookup and calculator tools
- Automatic tool execution tracing
- Step-by-step workflow visibility

### 3. Complex Agent (🤖)
- Research agent with knowledge retrieval
- Multi-turn conversations with context
- Advanced reasoning pattern tracking

### 4. Error Handling (⚠️)
- Deliberate tool failures and recovery
- Error categorization and metrics
- Resilient agent behavior patterns

### 5. Performance Monitoring (📊)
- Usage reports and cost analysis
- Cache hit/miss ratios
- Streaming event tracking

## Braintrust Features

The demo leverages these Braintrust capabilities:

- **LLM Span Recognition**: Automatic conversion of OpenTelemetry spans to Braintrust LLM spans
- **GenAI Semantic Conventions**: Proper mapping of `gen_ai.*` attributes
- **Trace Relationships**: Parent-child span relationships for multi-step workflows
- **Cost Tracking**: Token usage and estimated costs per request
- **Error Analysis**: Categorized error tracking and recovery patterns
- **Performance Insights**: Latency, throughput, and efficiency metrics

## Architecture

```
GAI Framework → OpenTelemetry → OTLP HTTP → Braintrust
     ↓              ↓              ↓           ↓
  Core Types → Trace Spans → GenAI Attrs → LLM Spans
```

## Key Integration Points

1. **Trace Export**: Uses `otlptracehttp` to send traces to `https://api.braintrust.dev/otel`
2. **Authentication**: Bearer token with project-specific headers
3. **Semantic Conventions**: Maps GAI framework concepts to GenAI standard attributes
4. **Span Relationships**: Maintains proper parent-child relationships across tool executions

## Troubleshooting

**No traces in Braintrust?**
- Verify API key and project ID are correct
- Check network connectivity to api.braintrust.dev
- Ensure spans are being created with `obs.StartRequestSpan()`

**Missing attributes?**
- Confirm GenAI semantic conventions are being set
- Check that spans are recorded before ending
- Verify OTLP exporter configuration

**High latency?**  
- Adjust batch timeout and size in trace provider
- Consider using async export patterns for production
- Monitor Groq API response times

## Production Considerations

- **Sampling**: Use probabilistic sampling for high-volume applications
- **Batching**: Configure appropriate batch sizes and timeouts  
- **Error Handling**: Implement robust error handling for trace export
- **Security**: Use secure credential management for API keys
- **Monitoring**: Set up alerts for trace export failures

## Next Steps

1. Explore the Braintrust dashboard features
2. Set up custom evaluations and scoring
3. Implement A/B testing with different models
4. Create custom metrics and dashboards
5. Integrate with CI/CD for automated evaluation