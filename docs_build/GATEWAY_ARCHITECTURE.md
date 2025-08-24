# GAI Gateway Architecture: Production-Ready Improvements

## Overview

This document describes the architectural improvements made to the GAI framework to make it "operationally boring" at scale, as requested by the software architect. These changes ensure the framework is production-ready for building a Gateway product with enterprise-grade reliability.

## Three Core Improvements

### 1. Normalized Event Schema (`gai.events.v1`)

**Problem Solved**: Provider-specific event formats create client complexity and prevent seamless provider switching.

**Implementation**:
- **File**: `stream/normalize.go`
- **Wire Format**: Stable `gai.events.v1` schema versioning
- **Event Types**: Consistent string-based event types across all providers
  - `start`, `finish`, `error`
  - `text.delta`, `audio.delta`
  - `tool.call`, `tool.result`
  - `citations`, `safety`, `step.end`

**Key Features**:
- Dual-mode streaming: normalized vs passthrough
- Provider abstraction layer
- Compact JSON format for efficiency
- Golden wire format testing for regression prevention

**Usage Example**:
```go
// Normalized mode (default)
config := StreamConfig{
    Mode: ModeNormalized,
    RequestID: "req_123",
    Provider: "openai",
}
SSENormalized(w, stream, config)

// Passthrough mode for OpenAI compatibility
config.Mode = ModePassthrough
SSEPassthroughOpenAI(w, stream, config)
```

### 2. Idempotency Surfaces

**Problem Solved**: Duplicate requests in distributed systems cause billing issues and inconsistent state.

**Implementation**:
- **Core Types**: Added to `core/types.go`
  - `Request.IdempotencyKey`: Client-provided deduplication key
  - `Request.RequestID`: System-generated unique identifier
- **Tool Execution**: Enhanced `tools/tools.go`
  - `Meta.RequestID`: Propagated to tool calls
  - `Meta.IdempotencyScope`: Tool-specific deduplication
  - `Meta.Attempt`: Retry tracking

**Key Features**:
- Automatic RequestID generation (UUIDv7 format)
- Header support: `X-Idempotency-Key`, `Idempotency-Key`
- Request-level and tool-level deduplication
- Distributed tracing support via TraceID

**Usage Example**:
```go
req := core.Request{
    IdempotencyKey: "user-action-123",
    RequestID: "req_1234567890_1", // Auto-generated if not provided
    Model: "gpt-4",
    Messages: messages,
}
```

### 3. Stable Error Taxonomy

**Problem Solved**: Provider-specific error codes create inconsistent error handling and retry logic.

**Implementation**:
- **Core Errors**: New taxonomy in `core/errors.go`
  - String-based `ErrorCode` for wire stability
  - Consistent categorization across providers
  - Automatic retry hints
- **Provider Mapping**: `providers/openai/errors.go`
  - Provider-specific to stable error mapping
  - Preserves original error for debugging
  - Retry-after header support

**Error Codes**:
```go
// Client errors (4xx)
ErrorInvalidRequest        // Bad schema/params
ErrorUnauthorized          // Missing/invalid auth
ErrorForbidden             // No permission
ErrorNotFound              // Resource not found
ErrorContextLengthExceeded // Input too long
ErrorUnsupported           // Feature not available

// Rate limiting
ErrorRateLimited           // 429 Too Many Requests
ErrorOverloaded            // Provider at capacity

// Safety
ErrorSafetyBlocked         // Content filtered

// Network/Infrastructure
ErrorTimeout               // Request timeout
ErrorNetwork               // Connection error
ErrorProviderUnavailable   // Provider down

// Server errors
ErrorInternal              // Unexpected error
```

**Usage Example**:
```go
// Provider-specific error mapping
err := MapError(httpResponse)
if core.IsTransient(err) {
    // Retry with backoff
    delay := core.GetRetryAfter(err)
    time.Sleep(delay)
}
```

## Testing Strategy

### Golden Wire Format Tests
- **File**: `stream/normalize_test.go`
- **Purpose**: Prevent breaking changes to wire format
- **Coverage**: All event types, compact JSON format, schema validation

### Error Mapping Tests
- **File**: `providers/openai/errors_test.go`
- **Coverage**: All OpenAI error types and codes
- **Real-world**: Actual API error responses

### Integration Tests
- **Streaming**: SSE and NDJSON formats
- **Error handling**: Retry logic, rate limiting
- **Idempotency**: Deduplication verification

## Migration Guide

### For Gateway Developers

1. **Choose Streaming Mode**:
   ```go
   // For new clients: use normalized format
   config.Mode = ModeNormalized
   
   // For OpenAI compatibility: use passthrough
   config.Mode = ModePassthrough
   ```

2. **Implement Idempotency**:
   ```go
   // Client-side
   req.Header.Set("X-Idempotency-Key", userActionID)
   
   // Server-side
   if seen := cache.Get(req.IdempotencyKey); seen != nil {
       return seen // Return cached response
   }
   ```

3. **Handle Errors Consistently**:
   ```go
   switch {
   case core.IsRateLimited(err):
       // Back off and retry
   case core.IsAuth(err):
       // Refresh credentials
   case core.IsTransient(err):
       // Retry with exponential backoff
   default:
       // Log and fail
   }
   ```

### For Provider Implementers

1. **Create Error Mapper**:
   ```go
   // providers/{provider}/errors.go
   func MapError(resp *http.Response) error {
       // Map provider-specific to stable codes
       return core.NewError(code, message, opts...)
   }
   ```

2. **Use Error Mapper**:
   ```go
   func (p *Provider) parseError(resp *http.Response) error {
       return MapError(resp)
   }
   ```

3. **Support Streaming Modes**:
   ```go
   // Emit core.Event types
   stream.Send(core.Event{
       Type: core.EventTextDelta,
       TextDelta: chunk,
   })
   ```

## Performance Considerations

### Event Normalization
- **Overhead**: ~200ns per event (see benchmarks)
- **Memory**: Minimal allocation with channel buffering
- **Optimization**: Compact JSON format reduces bandwidth

### Error Mapping
- **Caching**: Error patterns cached per provider
- **Fast Path**: Common errors mapped in O(1)
- **Fallback**: HTTP status code mapping

### Idempotency
- **Storage**: Consider Redis/DynamoDB for distributed cache
- **TTL**: 24-hour default for idempotency keys
- **Scope**: Request-level vs tool-level deduplication

## Future Enhancements

1. **Additional Providers**:
   - Anthropic error mapping
   - Google Gemini error mapping
   - Cohere error mapping

2. **Advanced Features**:
   - Request replay from idempotency cache
   - Automatic retry with jitter
   - Circuit breaker per provider

3. **Observability**:
   - Error rate metrics by code
   - Idempotency hit rate
   - Wire format version tracking

## Conclusion

These architectural improvements transform the GAI framework into a production-ready foundation for building Gateway products. The changes prioritize:

- **Operational Simplicity**: Consistent behavior across providers
- **Reliability**: Proper error handling and deduplication
- **Compatibility**: Support for both normalized and passthrough modes
- **Maintainability**: Comprehensive testing and clear separation of concerns

The framework now provides the "boring" operational characteristics required for enterprise deployment while maintaining flexibility for future enhancements.