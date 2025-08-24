# GAI Architecture Overview

This document provides a comprehensive understanding of GAI's architecture, design principles, and how all the components work together to provide a unified AI integration framework.

## Table of Contents
- [Design Philosophy](#design-philosophy)
- [System Architecture](#system-architecture)
- [Core Components](#core-components)
- [Data Flow](#data-flow)
- [Provider Abstraction](#provider-abstraction)
- [Extensibility Points](#extensibility-points)
- [Performance Considerations](#performance-considerations)
- [Security Architecture](#security-architecture)

## Design Philosophy

GAI is built on several key architectural principles that guide every design decision:

### 1. Provider Agnosticism

The framework treats all AI providers as implementations of a common interface. This means:

```go
// Any provider can be used interchangeably
var provider core.Provider

provider = openai.New(...)      // OpenAI
provider = anthropic.New(...)   // Anthropic  
provider = gemini.New(...)      // Google Gemini
provider = ollama.New(...)      // Local models

// The same code works with any provider
response, err := provider.GenerateText(ctx, request)
```

This abstraction enables:
- **Vendor Independence**: Switch providers without code changes
- **Multi-Provider Strategies**: Use different providers for different tasks
- **Fallback Mechanisms**: Automatically failover to backup providers
- **Cost Optimization**: Route requests to the most cost-effective provider

### 2. Type Safety First

GAI leverages Go's type system and generics to provide compile-time safety:

```go
// Structured outputs with compile-time type checking
type Analysis struct {
    Sentiment string   `json:"sentiment"`
    Score     float64  `json:"score"`
    Keywords  []string `json:"keywords"`
}

// The compiler ensures type safety
result, err := provider.GenerateObject[Analysis](ctx, request)
// result.Value is guaranteed to be of type Analysis
```

### 3. Streaming as a First-Class Citizen

Streaming is not an afterthought but a core design consideration:

```go
// Streaming is built into the provider interface
type Provider interface {
    GenerateText(ctx context.Context, req Request) (*TextResult, error)
    StreamText(ctx context.Context, req Request) (TextStream, error)
    // ...
}

// Event-driven streaming with backpressure
type TextStream interface {
    Events() <-chan Event  // Channel provides natural backpressure
    Close() error
}
```

### 4. Zero-Allocation Hot Paths

Performance-critical paths are optimized to minimize allocations:

```go
// Event creation uses a single struct with optional fields
type Event struct {
    Type       EventType
    TextDelta  string      // Only allocated when needed
    ToolCall   *ToolCall   // Pointer to avoid allocation when nil
    // ... other optional fields
}

// Stop conditions check with zero allocations
func MaxSteps(n int) StopCondition {
    return func(step int, _ Step) bool {
        return step >= n  // Simple comparison, no allocations
    }
}
```

### 5. Explicit Error Handling

GAI embraces Go's explicit error handling with rich error types:

```go
// Errors carry context for better debugging
type AIError struct {
    Code       ErrorCode
    Message    string
    Provider   string
    Temporary  bool
    RetryAfter time.Duration
    Cause      error
}

// Semantic error checking
if errors.IsRateLimited(err) {
    time.Sleep(errors.GetRetryAfter(err))
    // retry...
}
```

## System Architecture

### High-Level Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      Application Layer                       │
│  (Your AI Application - Chatbots, Agents, Analytics, etc.)   │
└──────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                         GAI Framework                        │
├──────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────┐   │
│  │                    Core Package                      │   │
│  │  • Types (Message, Request, Response, Event)         │   │
│  │  • Provider Interface                                │   │
│  │  • Error Taxonomy                                    │   │
│  │  • Runner (Multi-step Execution)                     │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │   Tools     │ │  Streaming  │ │     Middleware      │   │
│  │  • Schema   │ │  • SSE      │ │  • Retry            │   │
│  │  • Typing   │ │  • NDJSON   │ │  • Rate Limit       │   │
│  │  • Registry │ │  • Events   │ │  • Safety           │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
│                                                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │   Prompts   │ │    Media    │ │   Observability     │   │
│  │  • Templates│ │  • TTS/STT  │ │  • Tracing          │   │
│  │  • Versions │ │  • Audio    │ │  • Metrics          │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                      Provider Adapters                       │
├──────────────────────────────────────────────────────────────┤
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   │
│  │  OpenAI  │ │Anthropic │ │  Gemini  │ │    Ollama    │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              OpenAI Compatible Adapter               │   │
│  │     (Groq, xAI, Together, Cerebras, Baseten, ...)    │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                        AI Providers                          │
│   (OpenAI API, Anthropic API, Google AI, Local Models, ...)  │
└──────────────────────────────────────────────────────────────┘
```

### Layer Responsibilities

#### Application Layer
Your application code that uses GAI to build AI-powered features:
- Chatbots and conversational interfaces
- Content generation systems
- Data analysis and extraction
- Multi-agent systems
- Voice applications

#### GAI Framework Layer
The core framework providing unified abstractions:
- **Core Package**: Fundamental types and interfaces
- **Tools Package**: Function calling with type safety
- **Streaming Package**: Real-time response handling
- **Middleware Package**: Cross-cutting concerns
- **Prompts Package**: Template management
- **Media Package**: Audio/speech capabilities
- **Observability Package**: Monitoring and tracing

#### Provider Adapter Layer
Implementations that adapt provider-specific APIs to GAI's interface:
- Protocol translation (REST, SSE, WebSocket)
- Error mapping to unified taxonomy
- Feature normalization
- Capability detection

#### AI Provider Layer
The actual AI services:
- Cloud providers (OpenAI, Anthropic, Google)
- Local inference (Ollama)
- Custom deployments

## Core Components

### 1. The Provider Interface

The heart of GAI is the `Provider` interface:

```go
type Provider interface {
    // Text generation
    GenerateText(ctx context.Context, req Request) (*TextResult, error)
    StreamText(ctx context.Context, req Request) (TextStream, error)
    
    // Structured output
    GenerateObject(ctx context.Context, req Request, schema any) (*ObjectResult[any], error)
    StreamObject(ctx context.Context, req Request, schema any) (ObjectStream[any], error)
}
```

This interface is:
- **Minimal**: Only essential methods
- **Composable**: Providers can be wrapped with middleware
- **Extensible**: New methods can be added without breaking existing code

### 2. Message System

GAI uses a flexible message system that supports multimodal content:

```go
type Message struct {
    Role  Role    // System, User, Assistant, Tool
    Parts []Part  // Multiple content parts
    Name  string  // Optional: for named participants
}

type Part interface {
    isPart()      // Sealed interface pattern
    partType() string
}

// Concrete part types
type Text struct { Text string }
type ImageURL struct { URL string }
type Audio struct { Source BlobRef }
type Video struct { Source BlobRef }
type File struct { Source BlobRef; Name string }
```

This design enables:
- **Multimodal Messages**: Mix text, images, audio in a single message
- **Type Safety**: Compile-time checking of part types
- **Extensibility**: New part types can be added

### 3. Request/Response Model

Requests encapsulate all parameters for AI generation:

```go
type Request struct {
    // Model configuration
    Model       string
    Messages    []Message
    
    // Generation parameters
    Temperature float32
    MaxTokens   int
    TopP        float32
    
    // Advanced features
    Tools       []tools.Handle
    ToolChoice  ToolChoice
    StopWhen    StopCondition
    
    // Provider-specific
    ProviderOptions map[string]any
    
    // Observability
    Metadata    map[string]any
}
```

Responses provide comprehensive results:

```go
type TextResult struct {
    Text  string      // Generated text
    Steps []Step      // Multi-step execution history
    Usage Usage       // Token consumption
    Raw   any         // Provider-specific data
}

type Usage struct {
    InputTokens  int
    OutputTokens int
    TotalTokens  int
}
```

### 4. Streaming System

Streaming uses channels for natural Go concurrency:

```go
type TextStream interface {
    Events() <-chan Event  // Read-only channel for events
    Close() error          // Clean shutdown
}

type Event struct {
    Type       EventType
    TextDelta  string
    ToolCall   *ToolCall
    Citations  []Citation
    Safety     *SafetyEvent
    Error      error
}
```

Channel-based streaming provides:
- **Backpressure**: Automatic flow control
- **Cancellation**: Context-based cancellation
- **Multiplexing**: Multiple consumers can process events

### 5. Tool System

Tools are type-safe functions that AI can call:

```go
// Define a tool with typed input/output
func CreateWeatherTool() tools.Handle {
    return tools.New[WeatherInput, WeatherOutput](
        "get_weather",
        "Get current weather for a location",
        func(ctx context.Context, in WeatherInput, meta tools.Meta) (WeatherOutput, error) {
            // Implementation
            return WeatherOutput{
                Temperature: 72,
                Condition:   "Sunny",
            }, nil
        },
    )
}

// The framework handles:
// - JSON Schema generation from types
// - Marshaling/unmarshaling
// - Error handling
// - Parallel execution
```

### 6. Multi-Step Runner

The runner orchestrates complex multi-step executions:

```go
type Runner struct {
    provider    Provider
    maxParallel int
    toolTimeout time.Duration
    metrics     MetricsCollector
}

// Execution flow:
// 1. Send request to provider
// 2. Receive response with potential tool calls
// 3. Execute tools in parallel
// 4. Add results to message history
// 5. Check stop condition
// 6. Repeat if needed
```

## Data Flow

### Request Processing Flow

```
Application Request
        │
        ▼
┌──────────────┐
│   Request    │ ◄── User constructs request with messages, tools, etc.
└──────────────┘
        │
        ▼
┌──────────────┐
│  Middleware  │ ◄── Apply retry, rate limiting, safety filters
└──────────────┘
        │
        ▼
┌──────────────┐
│   Provider   │ ◄── Provider adapter translates to API format
│   Adapter    │
└──────────────┘
        │
        ▼
┌──────────────┐
│   HTTP/WS    │ ◄── Network communication with AI service
│   Transport  │
└──────────────┘
        │
        ▼
┌──────────────┐
│  AI Service  │ ◄── Actual AI processing
└──────────────┘
        │
        ▼
┌──────────────┐
│   Response   │ ◄── Parse and normalize response
│   Parsing    │
└──────────────┘
        │
        ▼
┌──────────────┐
│    Result    │ ◄── Typed result returned to application
└──────────────┘
```

### Streaming Flow

```
Stream Request
        │
        ▼
┌──────────────┐
│ SSE/WebSocket│ ◄── Establish streaming connection
│  Connection  │
└──────────────┘
        │
        ▼
┌──────────────┐
│Event Pipeline│
├──────────────┤
│ Parse Chunk  │ ◄── Parse SSE/NDJSON chunks
│      ▼       │
│  Normalize   │ ◄── Convert to GAI events
│      ▼       │
│   Emit to    │ ◄── Send via channel
│   Channel    │
└──────────────┘
        │
        ▼
┌──────────────┐
│  Application │ ◄── Process events as they arrive
│   for-range  │
└──────────────┘
```

### Tool Execution Flow

```
AI Response with Tool Calls
        │
        ▼
┌──────────────────┐
│ Extract Tool     │
│     Calls        │
└──────────────────┘
        │
        ▼
┌──────────────────┐
│ Parallel         │
│ Execution        │
├──────────────────┤
│  ┌────────────┐  │
│  │   Tool 1   │  │ ◄── Execute each tool
│  └────────────┘  │      in parallel
│  ┌────────────┐  │      with timeout
│  │   Tool 2   │  │
│  └────────────┘  │
│  ┌────────────┐  │
│  │   Tool N   │  │
│  └────────────┘  │
└──────────────────┘
        │
        ▼
┌──────────────────┐
│ Collect Results  │ ◄── Wait for all tools
└──────────────────┘
        │
        ▼
┌──────────────────┐
│ Add to Message   │ ◄── Append tool results
│    History       │      to conversation
└──────────────────┘
        │
        ▼
┌──────────────────┐
│ Check Stop       │ ◄── Evaluate stop condition
│   Condition      │
└──────────────────┘
        │
        ▼
    Continue?
    Yes │ No
        │  └──► Return Result
        ▼
    Next Step
```

## Provider Abstraction

### Provider Adapter Pattern

Each provider adapter follows a consistent pattern:

```go
package openai

type Provider struct {
    apiKey     string
    baseURL    string
    model      string
    client     *http.Client
    // Provider-specific fields
}

// Implement core.Provider interface
func (p *Provider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
    // 1. Convert core.Request to provider format
    apiReq := p.convertRequest(req)
    
    // 2. Make API call
    apiResp, err := p.callAPI(ctx, apiReq)
    if err != nil {
        return nil, p.mapError(err)
    }
    
    // 3. Convert response to core.TextResult
    return p.convertResponse(apiResp), nil
}

// Provider-specific conversion
func (p *Provider) convertRequest(req core.Request) *openaiRequest {
    // Map GAI types to OpenAI types
}

func (p *Provider) mapError(err error) error {
    // Map to GAI error taxonomy
}
```

### Capability Detection

Providers can expose their capabilities:

```go
type Capabilities struct {
    SupportsTools        bool
    SupportsStreaming    bool
    SupportsJSONMode     bool
    SupportsVision       bool
    MaxContextTokens     int
    SupportedModalities  []string
}

func (p *Provider) Capabilities() Capabilities {
    return Capabilities{
        SupportsTools:     true,
        SupportsStreaming: true,
        SupportsVision:    true,
        MaxContextTokens:  128000,
    }
}
```

## Extensibility Points

GAI is designed to be extended at multiple points:

### 1. Custom Providers

Implement the `Provider` interface for new AI services:

```go
type CustomProvider struct {
    // Your fields
}

func (p *CustomProvider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
    // Your implementation
}

func (p *CustomProvider) StreamText(ctx context.Context, req core.Request) (core.TextStream, error) {
    // Your implementation
}
```

### 2. Middleware

Wrap providers with cross-cutting concerns:

```go
// Retry middleware
func WithRetry(provider core.Provider, opts RetryOpts) core.Provider {
    return &retryProvider{
        inner: provider,
        opts:  opts,
    }
}

// Rate limiting middleware
func WithRateLimit(provider core.Provider, rps int) core.Provider {
    return &rateLimitProvider{
        inner:   provider,
        limiter: rate.NewLimiter(rate.Limit(rps), rps),
    }
}

// Compose middleware
provider = WithRetry(
    WithRateLimit(
        WithObservability(
            openai.New(...),
        ),
    ),
)
```

### 3. Custom Tools

Create domain-specific tools:

```go
// Database query tool
dbTool := tools.New[QueryInput, QueryOutput](
    "query_database",
    "Execute SQL queries",
    func(ctx context.Context, in QueryInput, meta tools.Meta) (QueryOutput, error) {
        rows, err := db.QueryContext(ctx, in.SQL)
        // ... process and return results
    },
)

// API integration tool
apiTool := tools.New[APIRequest, APIResponse](
    "call_api",
    "Make HTTP API calls",
    func(ctx context.Context, in APIRequest, meta tools.Meta) (APIResponse, error) {
        resp, err := http.Get(in.URL)
        // ... process and return
    },
)
```

### 4. Custom Streaming Formats

Implement custom streaming protocols:

```go
type CustomStream struct {
    reader io.Reader
    events chan core.Event
}

func (s *CustomStream) Events() <-chan core.Event {
    return s.events
}

func (s *CustomStream) processStream() {
    // Parse your custom format
    // Emit events to channel
}
```

## Performance Considerations

### Memory Management

GAI is designed to minimize allocations:

```go
// Event struct reuse via sync.Pool
var eventPool = sync.Pool{
    New: func() interface{} {
        return &Event{}
    },
}

// String builder reuse for streaming
var builderPool = sync.Pool{
    New: func() interface{} {
        return &strings.Builder{}
    },
}
```

### Concurrency Patterns

Efficient concurrency throughout:

```go
// Parallel tool execution with semaphore
sem := make(chan struct{}, maxParallel)
var wg sync.WaitGroup

for _, tool := range tools {
    wg.Add(1)
    go func(t Tool) {
        defer wg.Done()
        sem <- struct{}{}        // Acquire
        defer func() { <-sem }() // Release
        
        result := t.Execute(ctx)
        // ... handle result
    }(tool)
}
wg.Wait()
```

### Streaming Optimizations

Efficient streaming with buffered channels:

```go
// Buffered channel for smooth streaming
events := make(chan Event, 100)

// Non-blocking send with overflow handling
select {
case events <- event:
    // Sent successfully
default:
    // Channel full, apply backpressure
}
```

## Security Architecture

### API Key Management

GAI never logs or stores API keys:

```go
// API keys are only held in memory
type Provider struct {
    apiKey string // Never logged
}

// Sanitized error messages
func (p *Provider) String() string {
    return fmt.Sprintf("Provider{model=%s}", p.model)
    // Note: apiKey is not included
}
```

### Input Validation

All inputs are validated:

```go
func (p *Provider) GenerateText(ctx context.Context, req Request) (*TextResult, error) {
    // Validate request
    if err := req.Validate(); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }
    
    // Sanitize inputs
    req = p.sanitizeRequest(req)
    
    // Size limits
    if req.MaxTokens > p.maxAllowedTokens {
        req.MaxTokens = p.maxAllowedTokens
    }
    
    // ... continue processing
}
```

### Tool Execution Sandboxing

Tools run with constraints:

```go
type ToolExecutor struct {
    timeout    time.Duration
    maxOutput  int
    allowlist  []string
}

func (e *ToolExecutor) Execute(ctx context.Context, tool Tool) (any, error) {
    // Apply timeout
    ctx, cancel := context.WithTimeout(ctx, e.timeout)
    defer cancel()
    
    // Check allowlist
    if !e.isAllowed(tool.Name()) {
        return nil, ErrToolNotAllowed
    }
    
    // Execute with size limits
    output, err := tool.Execute(ctx)
    if len(output) > e.maxOutput {
        return nil, ErrOutputTooLarge
    }
    
    return output, err
}
```

## Best Practices

### 1. Always Use Context

Pass context through all operations for cancellation and tracing:

```go
// Good
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
response, err := provider.GenerateText(ctx, request)

// Bad
response, err := provider.GenerateText(context.Background(), request)
```

### 2. Handle Streaming Properly

Always close streams and handle errors:

```go
stream, err := provider.StreamText(ctx, request)
if err != nil {
    return err
}
defer stream.Close() // Always close

for event := range stream.Events() {
    switch event.Type {
    case core.EventError:
        return event.Err // Handle errors
    case core.EventTextDelta:
        // Process text
    }
}
```

### 3. Use Middleware for Cross-Cutting Concerns

Don't implement retry/rate-limiting in application code:

```go
// Good - use middleware
provider = middleware.WithRetry(
    middleware.WithRateLimit(
        openai.New(...),
        10, // 10 RPS
    ),
    middleware.RetryOpts{MaxAttempts: 3},
)

// Bad - implementing retry in application
for i := 0; i < 3; i++ {
    response, err := provider.GenerateText(ctx, request)
    if err == nil {
        break
    }
    time.Sleep(time.Second * time.Duration(i+1))
}
```

### 4. Validate Structured Outputs

Always validate structured outputs:

```go
type Config struct {
    MaxRetries int    `json:"max_retries" validate:"min=1,max=10"`
    Timeout    string `json:"timeout" validate:"required"`
}

result, err := provider.GenerateObject[Config](ctx, request)
if err != nil {
    return err
}

// Validate the result
if err := validator.Struct(result.Value); err != nil {
    return fmt.Errorf("invalid config: %w", err)
}
```

## Summary

GAI's architecture provides:

1. **Unified Interface**: Single API for all AI providers
2. **Type Safety**: Compile-time checking with generics
3. **Performance**: Zero-allocation hot paths
4. **Extensibility**: Multiple extension points
5. **Production Ready**: Built-in retry, rate limiting, observability
6. **Security**: Secure by default with validation and sandboxing

The architecture enables you to:
- Build provider-agnostic AI applications
- Switch providers without code changes
- Handle errors consistently
- Stream responses efficiently
- Execute tools safely
- Monitor and observe all operations

Next, explore:
- [Messages and Parts](./messages.md) - Deep dive into the message system
- [Providers](./providers.md) - Understanding provider abstraction
- [Streaming](./streaming.md) - Real-time response handling
- [Tools](./tools.md) - Function calling system