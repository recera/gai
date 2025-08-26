# API Reference

This comprehensive API reference covers all public interfaces, types, and functions in the GAI framework. Use this as your complete technical reference when building AI applications.

## Table of Contents
- [Core Package](#core-package)
- [Providers](#providers)
- [Tools Package](#tools-package)
- [Media Package](#media-package)
- [Middleware Package](#middleware-package)
- [Error Handling](#error-handling)
- [Configuration Types](#configuration-types)

## Core Package

The `core` package contains the fundamental types and interfaces used throughout GAI.

### Provider Interface

The main interface that all AI providers implement:

```go
type Provider interface {
    // Generate text with optional tool calling
    GenerateText(ctx context.Context, req Request) (*TextResult, error)
    
    // Stream text generation in real-time
    StreamText(ctx context.Context, req Request) (TextStream, error)
    
    // Generate structured objects with schema validation
    GenerateObject(ctx context.Context, req Request, schema any) (*ObjectResult[any], error)
    
    // Stream structured object generation
    StreamObject(ctx context.Context, req Request, schema any) (ObjectStream[any], error)
}
```

### Request Type

Configuration for AI generation requests:

```go
type Request struct {
    // Model selection
    Model       string `json:"model,omitempty"`
    
    // Conversation messages
    Messages    []Message `json:"messages"`
    
    // Generation parameters
    Temperature float32 `json:"temperature,omitempty"`  // 0.0-2.0, controls randomness
    MaxTokens   int     `json:"max_tokens,omitempty"`   // Maximum tokens to generate
    TopP        float32 `json:"top_p,omitempty"`        // 0.0-1.0, nucleus sampling
    TopK        int     `json:"top_k,omitempty"`        // Top-k sampling
    Stop        []string `json:"stop,omitempty"`        // Stop sequences
    Seed        int     `json:"seed,omitempty"`         // Reproducibility seed
    
    // Advanced features
    Tools       []tools.Handle `json:"tools,omitempty"`       // Available tools
    ToolChoice  ToolChoice     `json:"tool_choice,omitempty"` // Tool calling strategy
    StopWhen    StopCondition  `json:"-"`                     // Multi-step stop condition
    
    // Provider-specific options
    ProviderOptions map[string]any `json:"provider_options,omitempty"`
    
    // Metadata and observability
    Metadata map[string]any `json:"metadata,omitempty"`
}
```

### Message Types

Messages represent conversation turns:

```go
type Message struct {
    Role  Role     `json:"role"`           // Who is speaking
    Parts []Part   `json:"parts"`          // Content parts (multimodal)
    Name  string   `json:"name,omitempty"` // Named participant (for tools/agents)
}

type Role uint8
const (
    System Role = iota  // System instructions
    User               // Human user input  
    Assistant          // AI assistant response
    Tool               // Tool execution result
)
```

### Part Types (Multimodal Content)

Content parts for multimodal messages:

```go
// Sealed interface - only these types can be used as Parts
type Part interface {
    isPart()
    partType() string
}

// Text content
type Text struct {
    Text string `json:"text"`
}

// Image from URL
type ImageURL struct {
    URL    string `json:"url"`
    Detail string `json:"detail,omitempty"` // "low", "high", "auto"
}

// Audio content
type Audio struct {
    Source     BlobRef `json:"source"`
    SampleRate int     `json:"sample_rate,omitempty"`
    Channels   int     `json:"channels,omitempty"`
    Duration   float64 `json:"duration_seconds,omitempty"`
}

// Video content
type Video struct {
    Source   BlobRef `json:"source"`
    Duration float64 `json:"duration_seconds,omitempty"`
    Width    int     `json:"width,omitempty"`
    Height   int     `json:"height,omitempty"`
}

// File attachments
type File struct {
    Source  BlobRef `json:"source"`
    Name    string  `json:"name,omitempty"`
    Purpose string  `json:"purpose,omitempty"` // "assistants", "vision", etc.
}
```

### BlobRef System

Universal reference for binary content:

```go
type BlobRef struct {
    Kind   BlobKind `json:"kind"`
    URL    string   `json:"url,omitempty"`      // For BlobURL
    Bytes  []byte   `json:"bytes,omitempty"`    // For BlobBytes
    FileID string   `json:"file_id,omitempty"`  // For BlobProviderFile
    MIME   string   `json:"mime,omitempty"`     // MIME type
    Size   int64    `json:"size,omitempty"`     // Size in bytes
}

type BlobKind uint8
const (
    BlobURL          BlobKind = iota // Reference by URL
    BlobBytes                        // Inline bytes
    BlobProviderFile                 // Provider-specific file ID
)
```

### Response Types

Results from AI generation:

```go
type TextResult struct {
    Text  string `json:"text"`   // Generated text
    Steps []Step `json:"steps"`  // Multi-step execution history
    Usage Usage  `json:"usage"`  // Token consumption
    Raw   any    `json:"raw"`    // Provider-specific data
}

type ObjectResult[T any] struct {
    Value T     `json:"value"`  // Parsed structured object
    Steps []Step `json:"steps"` // Multi-step execution history  
    Usage Usage  `json:"usage"` // Token consumption
    Raw   any    `json:"raw"`   // Provider-specific data
}

// Execution step information
type Step struct {
    StepNumber  int           `json:"step_number"`
    ToolCalls   []ToolCall    `json:"tool_calls"`
    ToolResults []ToolResult  `json:"tool_results"`
    Text        string        `json:"text"`
    Usage       Usage         `json:"usage"`
    Duration    time.Duration `json:"duration"`
}

// Token usage tracking
type Usage struct {
    InputTokens  int `json:"input_tokens"`
    OutputTokens int `json:"output_tokens"`
    TotalTokens  int `json:"total_tokens"`
}
```

### Streaming Types

Real-time streaming interfaces:

```go
type TextStream interface {
    Events() <-chan Event  // Channel of streaming events
    Close() error          // Clean shutdown
}

type ObjectStream[T any] interface {
    Events() <-chan Event  // Channel of streaming events
    Close() error          // Clean shutdown
}

// Streaming events
type Event struct {
    Type       EventType      `json:"type"`
    TextDelta  string         `json:"text_delta,omitempty"`
    Object     any           `json:"object,omitempty"`
    ToolCall   *ToolCall     `json:"tool_call,omitempty"`
    ToolResult *ToolResult   `json:"tool_result,omitempty"`
    Citations  []Citation    `json:"citations,omitempty"`
    Safety     *SafetyEvent  `json:"safety,omitempty"`
    Usage      *Usage        `json:"usage,omitempty"`
    Error      error         `json:"error,omitempty"`
    Metadata   map[string]any `json:"metadata,omitempty"`
}

type EventType int
const (
    EventStart        EventType = iota // Stream started
    EventTextDelta                     // Text chunk received
    EventObjectDelta                   // Object chunk received
    EventToolCall                      // AI requested tool execution
    EventToolResult                    // Tool execution completed
    EventCitations                     // Source citations provided
    EventSafety                       // Safety/moderation event
    EventUsage                        // Token usage information
    EventError                        // Error occurred
    EventFinish                       // Stream completed
)
```

### Tool Calling Types

Types for function calling:

```go
// Tool execution request
type ToolCall struct {
    CallID string `json:"call_id"`
    Name   string `json:"name"`
    Input  []byte `json:"input"` // JSON-encoded input
}

// Tool execution result
type ToolResult struct {
    CallID string `json:"call_id"`
    Name   string `json:"name"`
    Result []byte `json:"result"` // JSON-encoded result
    Error  error  `json:"error,omitempty"`
}

// Tool calling strategy
type ToolChoice uint8
const (
    ToolAuto     ToolChoice = iota // Let AI decide
    ToolNone                       // Don't call tools
    ToolRequired                   // Must call at least one tool
    ToolSpecific                   // Call specific tool (use with ToolName)
)
```

### Stop Conditions

Control multi-step execution:

```go
type StopCondition func(stepNum int, step Step) bool

// Built-in stop conditions
func MaxSteps(n int) StopCondition
func NoMoreTools() StopCondition
func UntilToolSeen(toolName string) StopCondition
func CombineConditions(conditions ...StopCondition) StopCondition
```

### Citation and Safety Types

Additional metadata types:

```go
type Citation struct {
    Source    string `json:"source"`
    Title     string `json:"title,omitempty"`
    URL       string `json:"url,omitempty"`
    Snippet   string `json:"snippet,omitempty"`
    Relevance float64 `json:"relevance,omitempty"`
}

type SafetyEvent struct {
    Level    SafetyLevel `json:"level"`
    Category string      `json:"category"`
    Message  string      `json:"message"`
    Action   string      `json:"action"` // "blocked", "flagged", "passed"
}

type SafetyLevel uint8
const (
    SafetyLow SafetyLevel = iota
    SafetyMedium
    SafetyHigh
    SafetyCritical
)
```

## Providers

Each provider implements the core `Provider` interface with provider-specific configuration.

### OpenAI Provider

```go
package openai

// Create new OpenAI provider
func New(options ...Option) *Provider

// Configuration options
func WithAPIKey(key string) Option
func WithModel(model string) Option
func WithBaseURL(url string) Option
func WithOrganization(org string) Option
func WithProject(project string) Option
func WithTimeout(timeout time.Duration) Option
func WithMaxRetries(retries int) Option
func WithHTTPClient(client *http.Client) Option

// Provider struct (configuration only - methods are private)
type Provider struct {
    // Private fields
}

// Supported models (constants)
const (
    GPT5Turbo         = "gpt-5-turbo"
    GPT5Reasoning     = "gpt-5-reasoning"
    O1               = "o1"
    O1Mini           = "o1-mini"
    O1Preview        = "o1-preview"
    GPT4o            = "gpt-4o"
    GPT4oMini        = "gpt-4o-mini"
    GPT4Turbo        = "gpt-4-turbo"
    GPT4             = "gpt-4"
    GPT35Turbo       = "gpt-3.5-turbo"
)
```

### Anthropic Provider

```go
package anthropic

// Create new Anthropic provider
func New(options ...Option) *Provider

// Configuration options
func WithAPIKey(key string) Option
func WithModel(model string) Option
func WithVersion(version string) Option
func WithBaseURL(url string) Option
func WithTimeout(timeout time.Duration) Option
func WithMaxRetries(retries int) Option
func WithHTTPClient(client *http.Client) Option

// Supported models
const (
    ClaudeSonnet4         = "claude-sonnet-4-20250514"
    Claude35Haiku         = "claude-3-5-haiku-20241022"
    Claude35Sonnet        = "claude-3-5-sonnet-20241022"
    Claude3Opus           = "claude-3-opus-20240229"
    Claude3Sonnet         = "claude-3-sonnet-20240229"
    Claude3Haiku          = "claude-3-haiku-20240307"
)
```

### Google Gemini Provider

```go
package gemini

// Create new Gemini provider
func New(options ...Option) *Provider

// Configuration options
func WithAPIKey(key string) Option
func WithModel(model string) Option
func WithBaseURL(url string) Option
func WithProject(project string) Option
func WithLocation(location string) Option
func WithTimeout(timeout time.Duration) Option

// Supported models
const (
    Gemini20Flash    = "gemini-2.0-flash-exp"
    Gemini15Pro      = "gemini-1.5-pro"
    Gemini15Flash    = "gemini-1.5-flash"
    Gemini10Pro      = "gemini-1.0-pro"
)
```

### Groq Provider

```go
package groq

// Create new native Groq provider
func New(options ...Option) *Provider

// Configuration options
func WithAPIKey(key string) Option
func WithModel(model string) Option
func WithBaseURL(url string) Option
func WithTimeout(timeout time.Duration) Option
func WithMaxRetries(retries int) Option

// Supported models with LPU acceleration
const (
    Llama32Vision90B     = "llama-3.2-90b-vision-preview"
    Llama32Vision11B     = "llama-3.2-11b-vision-preview" 
    Llama3270B           = "llama-3.2-70b-versatile"
    Llama3290B           = "llama-3.2-90b-text-preview"
    Llama318BInstant     = "llama-3.1-8b-instant"
    Llama31405B          = "llama-3.1-405b-reasoning"
    Mixtral8x7BInstant   = "mixtral-8x7b-32768"
    Gemma29BIt           = "gemma2-9b-it"
)
```

### Ollama Provider

```go
package ollama

// Create new Ollama provider
func New(options ...Option) *Provider

// Configuration options
func WithBaseURL(url string) Option        // Default: http://localhost:11434
func WithModel(model string) Option
func WithTimeout(timeout time.Duration) Option
func WithKeepAlive(duration time.Duration) Option
func WithOptions(opts map[string]any) Option

// Model management
func (p *Provider) PullModel(ctx context.Context, model string) error
func (p *Provider) ListModels(ctx context.Context) ([]ModelInfo, error)
func (p *Provider) DeleteModel(ctx context.Context, model string) error

type ModelInfo struct {
    Name       string    `json:"name"`
    Size       int64     `json:"size"`
    ModifiedAt time.Time `json:"modified_at"`
    Digest     string    `json:"digest"`
}
```

### OpenAI Compatible Provider

```go
package openai_compat

// Create provider for OpenAI-compatible APIs
func New(options ...Option) *Provider

// Configuration options
func WithAPIKey(key string) Option
func WithBaseURL(url string) Option
func WithModel(model string) Option
func WithProvider(name string) Option // "groq", "together", etc.

// Predefined configurations
func NewGroqProvider(apiKey string) *Provider
func NewTogetherProvider(apiKey string) *Provider
func NewFireworksProvider(apiKey string) *Provider
func NewCerebrasProvider(apiKey string) *Provider
func NewXAIProvider(apiKey string) *Provider
```

## Tools Package

The `tools` package provides type-safe function calling capabilities.

### Core Types

```go
package tools

// Tool handle interface
type Handle interface {
    Name() string
    Description() string
    Schema() *JSONSchema
    Execute(ctx context.Context, input []byte, meta Meta) ([]byte, error)
}

// Tool execution metadata
type Meta struct {
    CallID    string            `json:"call_id"`
    MessageID string            `json:"message_id"`  
    Provider  string            `json:"provider"`
    Model     string            `json:"model"`
    Headers   map[string]string `json:"headers,omitempty"`
    Timeout   time.Duration     `json:"timeout"`
}
```

### Tool Creation

```go
// Create type-safe tool with generics
func New[Input, Output any](
    name string,
    description string,
    fn func(ctx context.Context, input Input, meta Meta) (Output, error),
) Handle

// Example usage
weatherTool := tools.New[WeatherInput, WeatherOutput](
    "get_weather",
    "Get current weather for a location",
    func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
        // Implementation
        return WeatherOutput{Temperature: 72}, nil
    },
)
```

### JSON Schema Generation

Schema generation from Go types using struct tags:

```go
type WeatherInput struct {
    Location string  `json:"location" jsonschema:"required,description=City and country"`
    Units    string  `json:"units,omitempty" jsonschema:"enum=metric,enum=imperial,default=metric"`
    Language string  `json:"language,omitempty" jsonschema:"pattern=^[a-z]{2}$,default=en"`
}

// Supported jsonschema tags:
// - required: Field is required
// - description: Field description
// - enum: Enumerated values
// - default: Default value
// - minimum/maximum: Numeric bounds
// - minLength/maxLength: String length bounds
// - pattern: Regex pattern
// - format: String format (email, uri, date, etc.)
// - minItems/maxItems: Array bounds
```

## Media Package

Audio and speech processing capabilities.

### Speech Provider Interface

```go
package media

type SpeechProvider interface {
    // Convert text to speech audio
    Synthesize(ctx context.Context, req SpeechRequest) (SpeechStream, error)
    
    // List available voices
    ListVoices(ctx context.Context) ([]Voice, error)
}

// Speech synthesis request
type SpeechRequest struct {
    Text            string  `json:"text"`
    Voice           string  `json:"voice,omitempty"`
    Model           string  `json:"model,omitempty"`
    Format          string  `json:"format,omitempty"`          // "mp3", "wav", "opus"
    Speed           float32 `json:"speed,omitempty"`           // 0.5-2.0
    Stability       float32 `json:"stability,omitempty"`       // 0.0-1.0
    SimilarityBoost float32 `json:"similarity_boost,omitempty"`// 0.0-1.0
    Options         map[string]any `json:"options,omitempty"`
}

// Streaming audio output
type SpeechStream interface {
    Chunks() <-chan []byte  // Audio data chunks
    Format() AudioFormat    // Audio format info
    Close() error          // Clean shutdown
    Error() error          // Stream error
}
```

### Transcription Provider Interface

```go
type TranscriptionProvider interface {
    // Convert audio to text
    Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResult, error)
    
    // Stream transcription for real-time audio
    TranscribeStream(ctx context.Context, audio io.Reader) (TranscriptionStream, error)
}

// Transcription request
type TranscriptionRequest struct {
    Audio            core.BlobRef `json:"audio"`
    Language         string       `json:"language,omitempty"`
    Model            string       `json:"model,omitempty"`
    Punctuate        bool         `json:"punctuate,omitempty"`
    Diarize          bool         `json:"diarize,omitempty"`
    FilterProfanity  bool         `json:"filter_profanity,omitempty"`
    Keywords         []string     `json:"keywords,omitempty"`
    MaxAlternatives  int          `json:"max_alternatives,omitempty"`
    Options          map[string]any `json:"options,omitempty"`
}

// Transcription result
type TranscriptionResult struct {
    Text         string                     `json:"text"`
    Alternatives []TranscriptionAlternative `json:"alternatives,omitempty"`
    Words        []WordTiming              `json:"words,omitempty"`
    Language     string                    `json:"language,omitempty"`
    Confidence   float32                   `json:"confidence"`
    Duration     time.Duration             `json:"duration"`
    Speakers     []SpeakerSegment          `json:"speakers,omitempty"`
}

// Word-level timing
type WordTiming struct {
    Word       string        `json:"word"`
    Start      time.Duration `json:"start"`
    End        time.Duration `json:"end"`
    Confidence float32       `json:"confidence"`
    Speaker    int           `json:"speaker"`
}
```

### Audio Format Types

```go
type AudioFormat struct {
    MIME       string `json:"mime"`        // "audio/mpeg", "audio/wav"
    SampleRate int    `json:"sample_rate"` // Hz (e.g. 44100)
    Channels   int    `json:"channels"`    // 1=mono, 2=stereo
    BitDepth   int    `json:"bit_depth"`   // e.g. 16, 24
    Encoding   string `json:"encoding"`    // "pcm", "mp3", "opus"
    Bitrate    int    `json:"bitrate"`     // bits per second
}

// Audio format constants
const (
    FormatMP3  = "mp3"
    FormatWAV  = "wav" 
    FormatOGG  = "ogg"
    FormatOpus = "opus"
    FormatFLAC = "flac"
    FormatPCM  = "pcm"
)

// MIME type constants
const (
    MimeMP3  = "audio/mpeg"
    MimeWAV  = "audio/wav"
    MimeOGG  = "audio/ogg"
    MimeOpus = "audio/opus"
    MimeFLAC = "audio/flac"
)
```

### Voice Information

```go
type Voice struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Description string   `json:"description,omitempty"`
    Languages   []string `json:"languages,omitempty"`
    Gender      string   `json:"gender,omitempty"`
    Age         string   `json:"age,omitempty"`
    Tags        []string `json:"tags,omitempty"`
    PreviewURL  string   `json:"preview_url,omitempty"`
    Premium     bool     `json:"premium,omitempty"`
}
```

## Middleware Package

Composable middleware for cross-cutting concerns.

### Middleware Interface

```go
package middleware

// Middleware wraps a Provider with additional functionality
type Middleware func(core.Provider) core.Provider

// Chain multiple middleware together
func Chain(middlewares ...Middleware) Middleware

// Apply middleware to a provider
provider = middleware.Chain(
    WithRetry(retryOpts),
    WithRateLimit(rateLimitOpts), 
    WithSafety(safetyOpts),
)(baseProvider)
```

### Built-in Middleware

#### Retry Middleware

```go
// Add retry logic with exponential backoff
func WithRetry(opts RetryOpts) Middleware

type RetryOpts struct {
    MaxAttempts    int           `json:"max_attempts"`
    InitialDelay   time.Duration `json:"initial_delay"`
    MaxDelay       time.Duration `json:"max_delay"`
    Multiplier     float64       `json:"multiplier"`
    RetryIf        func(error) bool `json:"-"`
}

// Default retry configuration
func DefaultRetryOpts() RetryOpts {
    return RetryOpts{
        MaxAttempts:  3,
        InitialDelay: time.Second,
        MaxDelay:     30 * time.Second,
        Multiplier:   2.0,
        RetryIf:      IsRetryableError,
    }
}
```

#### Rate Limiting Middleware

```go
// Add rate limiting
func WithRateLimit(opts RateLimitOpts) Middleware

type RateLimitOpts struct {
    RequestsPerSecond float64       `json:"requests_per_second"`
    BurstSize         int           `json:"burst_size"`
    WindowSize        time.Duration `json:"window_size"`
}
```

#### Safety Middleware

```go
// Add content safety filtering
func WithSafety(opts SafetyOpts) Middleware

type SafetyOpts struct {
    BlockHarmful     bool     `json:"block_harmful"`
    BlockPII         bool     `json:"block_pii"`
    AllowedTopics    []string `json:"allowed_topics,omitempty"`
    BlockedTopics    []string `json:"blocked_topics,omitempty"`
    MaxRequestLength int      `json:"max_request_length"`
}
```

#### Observability Middleware

```go
// Add metrics and tracing
func WithObservability(opts ObservabilityOpts) Middleware

type ObservabilityOpts struct {
    MetricsCollector MetricsCollector `json:"-"`
    TracingEnabled   bool            `json:"tracing_enabled"`
    LogRequests      bool            `json:"log_requests"`
    LogResponses     bool            `json:"log_responses"`
}

type MetricsCollector interface {
    RecordRequest(provider string, model string, duration time.Duration, tokens int, err error)
    RecordToolCall(toolName string, duration time.Duration, err error)
}
```

### Custom Middleware

```go
// Create custom middleware
func CustomLoggingMiddleware(logger Logger) Middleware {
    return func(provider core.Provider) core.Provider {
        return &loggingProvider{
            inner:  provider,
            logger: logger,
        }
    }
}

type loggingProvider struct {
    inner  core.Provider
    logger Logger
}

func (p *loggingProvider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
    start := time.Now()
    
    p.logger.Info("Starting text generation", "model", req.Model)
    
    result, err := p.inner.GenerateText(ctx, req)
    
    duration := time.Since(start)
    
    if err != nil {
        p.logger.Error("Text generation failed", "error", err, "duration", duration)
    } else {
        p.logger.Info("Text generation completed", 
            "duration", duration, 
            "tokens", result.Usage.TotalTokens)
    }
    
    return result, err
}
```

## Error Handling

Structured error types and utilities.

### Error Types

```go
// Base AI error with rich context
type AIError struct {
    Code       ErrorCode     `json:"code"`
    Message    string        `json:"message"`
    Provider   string        `json:"provider,omitempty"`
    Model      string        `json:"model,omitempty"`
    Temporary  bool          `json:"temporary"`
    RetryAfter time.Duration `json:"retry_after,omitempty"`
    Details    map[string]any `json:"details,omitempty"`
    Cause      error         `json:"-"`
}

func (e *AIError) Error() string { return e.Message }
func (e *AIError) Unwrap() error { return e.Cause }

// Error codes
type ErrorCode int
const (
    ErrorUnknown ErrorCode = iota
    ErrorAuth                    // Authentication failed
    ErrorRateLimit              // Rate limit exceeded  
    ErrorQuota                  // Quota exceeded
    ErrorModel                  // Model not available
    ErrorInput                  // Invalid input
    ErrorOutput                 // Invalid output
    ErrorNetwork               // Network error
    ErrorTimeout               // Request timeout
    ErrorSafety                // Content safety violation
    ErrorTool                  // Tool execution error
)
```

### Error Utilities

```go
// Check specific error types
func IsRateLimited(err error) bool
func IsQuotaExceeded(err error) bool
func IsTemporary(err error) bool
func IsRetryable(err error) bool

// Extract retry delay
func GetRetryAfter(err error) time.Duration

// Error wrapping
func WrapError(err error, code ErrorCode, message string) error
func WrapProviderError(err error, provider, model string) error

// Example usage
if IsRateLimited(err) {
    delay := GetRetryAfter(err)
    time.Sleep(delay)
    // retry
}
```

### Tool Errors

```go
// Tool-specific errors
type ToolError struct {
    ToolName string         `json:"tool_name"`
    CallID   string         `json:"call_id"`
    Code     ToolErrorCode  `json:"code"`
    Message  string         `json:"message"`
    Details  map[string]any `json:"details,omitempty"`
    Cause    error          `json:"-"`
}

type ToolErrorCode int
const (
    ToolErrorUnknown ToolErrorCode = iota
    ToolErrorNotFound           // Tool not found
    ToolErrorInvalidInput       // Invalid input
    ToolErrorExecutionFailed    // Execution failed
    ToolErrorTimeout           // Tool timeout
    ToolErrorPermission        // Permission denied
)
```

## Configuration Types

Common configuration patterns across the framework.

### Provider Configuration

```go
// Base configuration for all providers
type ProviderConfig struct {
    APIKey       string            `json:"api_key,omitempty"`
    BaseURL      string            `json:"base_url,omitempty"`
    Model        string            `json:"model,omitempty"`
    Timeout      time.Duration     `json:"timeout,omitempty"`
    MaxRetries   int               `json:"max_retries,omitempty"`
    HTTPClient   *http.Client      `json:"-"`
    Headers      map[string]string `json:"headers,omitempty"`
    UserAgent    string            `json:"user_agent,omitempty"`
}
```

### Generation Defaults

```go
// Default generation parameters
type GenerationDefaults struct {
    Temperature float32 `json:"temperature"`    // Default: 0.7
    MaxTokens   int     `json:"max_tokens"`     // Default: provider-specific
    TopP        float32 `json:"top_p"`          // Default: 1.0
    TopK        int     `json:"top_k"`          // Default: 0 (disabled)
    
    // Tool calling defaults
    ToolChoice      ToolChoice    `json:"tool_choice"`       // Default: ToolAuto
    MaxSteps        int           `json:"max_steps"`         // Default: 10
    ToolTimeout     time.Duration `json:"tool_timeout"`      // Default: 30s
    ParallelTools   int           `json:"parallel_tools"`    // Default: 5
}
```

### Validation

```go
// Request validation
func (r *Request) Validate() error {
    if len(r.Messages) == 0 {
        return fmt.Errorf("messages cannot be empty")
    }
    
    if r.Temperature < 0 || r.Temperature > 2.0 {
        return fmt.Errorf("temperature must be between 0.0 and 2.0")
    }
    
    if r.MaxTokens < 0 {
        return fmt.Errorf("max_tokens must be non-negative")
    }
    
    // Additional validation...
    return nil
}

// Message validation  
func (m *Message) Validate() error {
    if len(m.Parts) == 0 {
        return fmt.Errorf("message parts cannot be empty")
    }
    
    for i, part := range m.Parts {
        if err := validatePart(part); err != nil {
            return fmt.Errorf("part %d: %w", i, err)
        }
    }
    
    return nil
}
```

## Usage Examples

### Basic Text Generation

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
    provider := openai.New(
        openai.WithAPIKey("sk-..."),
        openai.WithModel(openai.GPT4oMini),
    )
    
    response, err := provider.GenerateText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Explain Go interfaces"},
                },
            },
        },
        Temperature: 0.7,
        MaxTokens:   500,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response.Text)
}
```

### Streaming with Tools

```go
func streamingWithTools() {
    provider := anthropic.New(
        anthropic.WithAPIKey("sk-ant-..."),
        anthropic.WithModel(anthropic.ClaudeSonnet4),
    )
    
    tools := []tools.Handle{
        createWeatherTool(),
        createCalculatorTool(),
    }
    
    stream, err := provider.StreamText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "What's the weather in Tokyo and calculate 15% tip on $47.50"},
                },
            },
        },
        Tools: tools,
        ToolChoice: core.ToolAuto,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            fmt.Print(event.TextDelta)
        case core.EventToolCall:
            fmt.Printf("\n🔧 Calling: %s\n", event.ToolCall.Name)
        case core.EventToolResult:
            fmt.Printf("✅ Result: %s\n", string(event.ToolResult.Result))
        case core.EventFinish:
            fmt.Println("\n[Complete]")
        }
    }
}
```

### Multi-Step Workflow

```go
func multiStepWorkflow() {
    provider := groq.New(
        groq.WithAPIKey("gsk-..."),
        groq.WithModel(groq.Llama318BInstant),
    )
    
    response, err := provider.GenerateText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are a research assistant. Work step by step."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Research AI trends, analyze data, create report, and email summary."},
                },
            },
        },
        Tools: []tools.Handle{
            createWebSearchTool(),
            createDataAnalysisTool(),
            createReportTool(),
            createEmailTool(),
        },
        StopWhen: core.CombineConditions(
            core.MaxSteps(20),
            core.UntilToolSeen("send_email"),
        ),
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Completed workflow in %d steps\n", len(response.Steps))
    fmt.Println(response.Text)
}
```

### Structured Output

```go
type Analysis struct {
    Summary    string   `json:"summary"`
    KeyPoints  []string `json:"key_points"`
    Confidence float64  `json:"confidence"`
    Tags       []string `json:"tags"`
}

func structuredOutput() {
    provider := openai.New(openai.WithAPIKey("sk-..."))
    
    result, err := provider.GenerateObject(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Analyze this text: 'The future of AI...'"},
                },
            },
        },
    }, Analysis{})
    
    if err != nil {
        log.Fatal(err)
    }
    
    analysis := result.Value.(Analysis)
    fmt.Printf("Summary: %s\n", analysis.Summary)
    fmt.Printf("Confidence: %.1f%%\n", analysis.Confidence*100)
}
```

This API reference provides comprehensive coverage of all public interfaces in GAI. Use it as your go-to reference when building applications with the framework.