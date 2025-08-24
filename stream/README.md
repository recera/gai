# Stream Package - Phase 7 Complete ✅

## Overview

The stream package provides production-grade **Server-Sent Events (SSE)** and **Newline-Delimited JSON (NDJSON)** streaming capabilities for the GAI framework. These implementations enable real-time AI response streaming to browsers and other HTTP clients with minimal overhead and maximum compatibility.

## Features

### SSE (Server-Sent Events)
- ✅ Full SSE protocol compliance
- ✅ Automatic heartbeat/keep-alive messages
- ✅ Event ID support for replay capability
- ✅ Retry hints for error recovery
- ✅ CORS headers for browser compatibility
- ✅ Nginx buffering bypass headers
- ✅ Thread-safe concurrent writes
- ✅ Backpressure handling

### NDJSON (Newline-Delimited JSON)
- ✅ Efficient line-based JSON streaming
- ✅ Buffered writing with periodic flush
- ✅ Compact JSON option for bandwidth savings
- ✅ Timestamp support for event ordering
- ✅ Reader implementation for bidirectional streams
- ✅ Channel-based event conversion
- ✅ Thread-safe operations

## API

### SSE Functions

```go
// Stream a TextStream as Server-Sent Events
func SSE(w http.ResponseWriter, stream core.TextStream, opts ...SSEOptions) error

// Create an HTTP handler for SSE streaming
func SSEHandler(provider core.Provider, prepareRequest func(*http.Request) (core.Request, error)) http.HandlerFunc

// Low-level SSE writer
type Writer struct {
    WriteEvent(event, data string) error
    WriteComment(comment string) error
}
```

### NDJSON Functions

```go
// Stream a TextStream as NDJSON
func NDJSON(w http.ResponseWriter, stream core.TextStream, opts ...NDJSONOptions) error

// Create an HTTP handler for NDJSON streaming
func NDJSONHandler(provider core.Provider, prepareRequest func(*http.Request) (core.Request, error)) http.HandlerFunc

// NDJSON reader for parsing streams
type Reader struct {
    Read(v any) error
}

// Convert NDJSON stream to event channel
func StreamToChannel(ctx context.Context, r io.Reader) (<-chan core.Event, error)
```

## Usage Examples

### Basic SSE Server

```go
package main

import (
    "net/http"
    "github.com/recera/gai/stream"
    "github.com/recera/gai/providers/openai"
)

func main() {
    // Create provider
    provider := openai.New(openai.WithAPIKey("..."))
    
    // Create SSE handler
    handler := stream.SSEHandler(provider, func(r *http.Request) (core.Request, error) {
        return core.Request{
            Messages: []core.Message{
                {Role: core.User, Parts: []core.Part{
                    core.Text{Text: r.URL.Query().Get("q")},
                }},
            },
            Stream: true,
        }, nil
    })
    
    http.HandleFunc("/chat", handler)
    http.ListenAndServe(":8080", nil)
}
```

### Browser Client (JavaScript)

```javascript
const eventSource = new EventSource('/chat?q=' + encodeURIComponent(question));

eventSource.addEventListener('text_delta', function(e) {
    const data = JSON.parse(e.data);
    document.getElementById('output').innerHTML += data.text;
});

eventSource.addEventListener('finish', function(e) {
    const data = JSON.parse(e.data);
    console.log('Usage:', data.usage);
});

eventSource.addEventListener('done', function(e) {
    eventSource.close();
});
```

### NDJSON with Fetch API

```javascript
async function streamChat(messages) {
    const response = await fetch('/api/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ messages })
    });
    
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    
    while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        
        for (let i = 0; i < lines.length - 1; i++) {
            const line = lines[i].trim();
            if (!line) continue;
            
            const event = JSON.parse(line);
            if (event.type === 'text_delta') {
                document.getElementById('output').innerHTML += event.text;
            }
        }
        
        buffer = lines[lines.length - 1];
    }
}
```

## Configuration

### SSE Options

```go
type SSEOptions struct {
    HeartbeatInterval time.Duration // Keep-alive interval (default: 15s)
    FlushAfterWrite   bool          // Force flush after each write (default: true)
    MaxRetries        int           // Client reconnection hints (default: 3)
    BufferSize        int           // Write buffer size (default: 4096)
    IncludeID         bool          // Add event IDs for replay (default: false)
}
```

### NDJSON Options

```go
type NDJSONOptions struct {
    BufferSize       int           // Write buffer size (default: 8192)
    FlushInterval    time.Duration // Periodic flush interval (default: 100ms)
    CompactJSON      bool          // Remove whitespace (default: true)
    IncludeTimestamp bool          // Add timestamps (default: false)
}
```

## Event Types

The streaming system handles all core event types:

- `start` - Stream initialization
- `text_delta` - Incremental text chunks
- `audio_delta` - Audio data chunks
- `tool_call` - Tool invocation events
- `tool_result` - Tool execution results
- `citations` - Source references
- `safety` - Content safety signals
- `finish_step` - Step completion
- `finish` - Stream completion with usage stats
- `error` - Error events
- `done` - Final completion signal

## Performance

### Benchmarks (M1 MacBook Pro)

| Operation | Time | Memory | Allocations |
|-----------|------|--------|-------------|
| SSE Writer | 79ns | 32B | 2 |
| NDJSON Writer | 234ns | 144B | 5 |
| SSE Stream (100 events) | 53μs | 93KB | 938 |
| NDJSON Stream (100 events) | 52μs | 103KB | 946 |
| HTTP Integration SSE | 296μs | 83KB | 1003 |
| HTTP Integration NDJSON | 302μs | 93KB | 1011 |

### Key Performance Features

- **Zero-allocation paths** in critical sections
- **Efficient buffering** to minimize syscalls
- **Concurrent-safe** operations without lock contention
- **Backpressure handling** to prevent memory bloat
- **Minimal overhead** compared to raw HTTP writes

## Browser Compatibility

### SSE Support
- ✅ Chrome/Edge 6+
- ✅ Firefox 6+
- ✅ Safari 5+
- ✅ Opera 11+
- ⚠️ IE/Edge Legacy (polyfill required)

### NDJSON Support
- ✅ All modern browsers (via Fetch API)
- ✅ Node.js
- ✅ Any HTTP client with streaming support

## HTTP Headers

### SSE Headers
```
Content-Type: text/event-stream
Cache-Control: no-cache, no-store, must-revalidate
Connection: keep-alive
X-Accel-Buffering: no
Access-Control-Allow-Origin: *
```

### NDJSON Headers
```
Content-Type: application/x-ndjson
Cache-Control: no-cache, no-store, must-revalidate
Connection: keep-alive
Transfer-Encoding: chunked
X-Accel-Buffering: no
Access-Control-Allow-Origin: *
```

## Testing

The package includes comprehensive test coverage:

- **Unit Tests**: All core functionality
- **Integration Tests**: Full HTTP server testing with httptest
- **Concurrent Tests**: Thread safety validation
- **Performance Tests**: Benchmarks for all operations
- **Browser Simulation**: EventSource behavior testing

### Run Tests
```bash
go test ./stream/...

# With coverage
go test ./stream/... -cover

# Benchmarks
go test ./stream/... -bench=. -benchmem
```

## Error Handling

The streaming implementations handle various error conditions:

- **Client disconnection**: Graceful cleanup of resources
- **Write failures**: Proper error propagation
- **Context cancellation**: Immediate stream termination
- **Provider errors**: Error events sent to client
- **Panic recovery**: In tool execution paths

## Production Considerations

1. **Proxy Configuration**: Ensure proxies (Nginx, etc.) don't buffer SSE streams
2. **Timeouts**: Configure appropriate keep-alive intervals
3. **CORS**: Adjust headers based on security requirements
4. **Rate Limiting**: Consider implementing rate limits for streaming endpoints
5. **Monitoring**: Track stream duration and completion rates
6. **Error Recovery**: Implement client-side reconnection logic

## Thread Safety

All streaming operations are thread-safe:
- Multiple goroutines can send events concurrently
- Writers use mutexes to ensure atomic writes
- Channel operations prevent race conditions
- Resource cleanup is properly synchronized

## Migration from Other Libraries

### From `github.com/r3labs/sse`
```go
// Before
sse.New().ServeHTTP(w, r)

// After
stream.SSE(w, textStream)
```

### From Manual SSE Implementation
```go
// Before
fmt.Fprintf(w, "data: %s\n\n", json)
w.(http.Flusher).Flush()

// After
writer := stream.NewWriter(w)
writer.WriteEvent("message", json)
```

## Best Practices

1. **Always defer stream.Close()** to prevent goroutine leaks
2. **Use heartbeats** for long-running streams
3. **Implement retry logic** on the client side
4. **Set appropriate buffer sizes** based on message size
5. **Monitor stream duration** to detect stuck connections
6. **Use event IDs** for critical streams requiring replay
7. **Compress responses** when bandwidth is limited
8. **Test with real browsers** not just curl

## Future Enhancements

Potential improvements for future phases:
- WebSocket support for bidirectional streaming
- gRPC streaming adapters
- Built-in compression (gzip/brotli)
- Stream multiplexing
- Event replay from persistent storage
- Metrics collection integration
- Stream encryption for sensitive data

## License

Part of the GAI framework under Apache-2.0 license.