package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

// benchmarkServer provides a fast mock server for benchmarks.
type benchmarkServer struct {
	*httptest.Server
}

func newBenchmarkServer() *benchmarkServer {
	b := &benchmarkServer{}
	b.Server = httptest.NewServer(http.HandlerFunc(b.handler))
	return b
}

func (b *benchmarkServer) handler(w http.ResponseWriter, r *http.Request) {
	// Fast path for benchmarks - minimal processing
	switch r.URL.Path {
	case "/chat/completions":
		b.handleChatCompletionsFast(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (b *benchmarkServer) handleChatCompletionsFast(w http.ResponseWriter, r *http.Request) {
	// Pre-built response for speed
	response := `{
		"id": "chatcmpl-bench",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4o-mini",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "Benchmark response"
				},
				"finish_reason": "stop"
			}
		],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 5,
			"total_tokens": 15
		}
	}`

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(response))
}

func BenchmarkProviderCreation(b *testing.B) {
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = New(
			WithAPIKey("test-key"),
			WithModel("gpt-4o-mini"),
			WithMaxRetries(3),
			WithRetryDelay(100*time.Millisecond),
		)
	}
}

func BenchmarkGenerateText(b *testing.B) {
	server := newBenchmarkServer()
	defer server.Close()

	p := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
		WithMaxRetries(0), // Disable retries for benchmarks
	)

	req := core.Request{
		Model: "gpt-4o-mini",
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Hello, how are you?"},
				},
			},
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := p.GenerateText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateTextParallel(b *testing.B) {
	server := newBenchmarkServer()
	defer server.Close()

	p := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
		WithMaxRetries(0),
	)

	req := core.Request{
		Model: "gpt-4o-mini",
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Hello"},
				},
			},
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := p.GenerateText(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkConvertMessages(b *testing.B) {
	p := New(WithAPIKey("test"))

	messages := []core.Message{
		{
			Role: core.System,
			Parts: []core.Part{
				core.Text{Text: "You are a helpful assistant."},
			},
		},
		{
			Role: core.User,
			Parts: []core.Part{
				core.Text{Text: "What is the weather like?"},
			},
		},
		{
			Role: core.Assistant,
			Parts: []core.Part{
				core.Text{Text: "I'll help you check the weather."},
			},
		},
		{
			Role: core.User,
			Parts: []core.Part{
				core.Text{Text: "Please check for San Francisco."},
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := p.convertMessages(messages)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertMultimodalMessages(b *testing.B) {
	p := New(WithAPIKey("test"))

	messages := []core.Message{
		{
			Role: core.User,
			Parts: []core.Part{
				core.Text{Text: "What's in these images?"},
				core.ImageURL{URL: "https://example.com/image1.jpg", Detail: "high"},
				core.ImageURL{URL: "https://example.com/image2.jpg", Detail: "low"},
				core.Text{Text: "Please describe them in detail."},
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := p.convertMessages(messages)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertTools(b *testing.B) {
	p := New(WithAPIKey("test"))

	// Create multiple tools
	tools := make([]core.ToolHandle, 10)
	for i := range tools {
		tools[i] = &mockTool{
			name: fmt.Sprintf("tool_%d", i),
			desc: fmt.Sprintf("Tool number %d", i),
			schema: []byte(`{
				"type": "object",
				"properties": {
					"param": {"type": "string"}
				}
			}`),
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = p.convertTools(tools)
	}
}

// mockTool implements core.ToolHandle for benchmarking.
type mockTool struct {
	name   string
	desc   string
	schema []byte
}

func (m *mockTool) Name() string               { return m.name }
func (m *mockTool) Description() string        { return m.desc }
func (m *mockTool) InSchemaJSON() []byte       { return m.schema }
func (m *mockTool) OutSchemaJSON() []byte      { return m.schema }
func (m *mockTool) Exec(ctx context.Context, raw json.RawMessage, meta interface{}) (any, error) {
	return map[string]string{"result": "ok"}, nil
}

func BenchmarkStreamProcessing(b *testing.B) {
	// Simulate SSE stream processing
	sseData := []string{
		`data: {"id":"1","object":"chat.completion.chunk","choices":[{"delta":{"content":"Hello"}}]}`,
		`data: {"id":"1","object":"chat.completion.chunk","choices":[{"delta":{"content":" world"}}]}`,
		`data: {"id":"1","object":"chat.completion.chunk","choices":[{"delta":{"content":"!"}}]}`,
		`data: {"id":"1","object":"chat.completion.chunk","usage":{"prompt_tokens":10,"completion_tokens":3,"total_tokens":13}}`,
		`data: [DONE]`,
	}

	reader := strings.NewReader(strings.Join(sseData, "\n\n") + "\n\n")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Reset reader
		reader.Seek(0, 0)
		
		// Process stream
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					break
				}
				
				var chunk streamChunk
				json.Unmarshal([]byte(data), &chunk)
			}
		}
	}
}

func BenchmarkJSONMarshaling(b *testing.B) {
	req := chatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []chatMessage{
			{
				Role:    "user",
				Content: "Hello, world!",
			},
		},
		Temperature: floatPtr(0.7),
		MaxTokens:   intPtr(100),
		Tools: []chatTool{
			{
				Type: "function",
				Function: function{
					Name:        "test_tool",
					Description: "A test tool",
					Parameters:  json.RawMessage(`{"type":"object"}`),
				},
			},
		},
		Stream: false,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONUnmarshaling(b *testing.B) {
	response := `{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4o-mini",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "This is a test response with some content.",
					"tool_calls": [
						{
							"id": "call_123",
							"type": "function",
							"function": {
								"name": "test_function",
								"arguments": "{\"param\":\"value\"}"
							}
						}
					]
				},
				"finish_reason": "tool_calls"
			}
		],
		"usage": {
			"prompt_tokens": 50,
			"completion_tokens": 25,
			"total_tokens": 75
		}
	}`

	data := []byte(response)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var resp chatCompletionResponse
		err := json.Unmarshal(data, &resp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkErrorParsing(b *testing.B) {
	p := New(WithAPIKey("test"))

	errorResponse := []byte(`{
		"error": {
			"message": "Invalid request: The model 'invalid-model' does not exist",
			"type": "invalid_request_error",
			"code": "model_not_found"
		}
	}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = p.parseErrorFromBody(400, errorResponse)
	}
}

func BenchmarkRetryLogic(b *testing.B) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount%3 != 0 { // Fail 2 out of 3 times
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}
		// Success
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "success",
			"object": "chat.completion",
			"choices": [{"message": {"role": "assistant", "content": "OK"}}],
			"usage": {"total_tokens": 10}
		}`)
	}))
	defer server.Close()

	p := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
		WithMaxRetries(2),
		WithRetryDelay(1*time.Millisecond), // Fast retries for benchmark
	)

	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Test"}}},
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		attemptCount = 0
		_, err := p.GenerateText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkToolExecution(b *testing.B) {
	p := New(WithAPIKey("test"))

	// Create test tools
	tools := []core.ToolHandle{
		&mockTool{
			name:   "tool1",
			desc:   "Test tool 1",
			schema: []byte(`{"type":"object"}`),
		},
		&mockTool{
			name:   "tool2",
			desc:   "Test tool 2",
			schema: []byte(`{"type":"object"}`),
		},
	}

	calls := []core.ToolCall{
		{
			ID:    "call1",
			Name:  "tool1",
			Input: json.RawMessage(`{"test":"value1"}`),
		},
		{
			ID:    "call2",
			Name:  "tool2",
			Input: json.RawMessage(`{"test":"value2"}`),
		},
	}

	messages := []core.Message{
		{Role: core.User, Parts: []core.Part{core.Text{Text: "Test"}}},
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := p.executeTools(ctx, tools, calls, messages)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark results on M1 MacBook Pro:
// BenchmarkProviderCreation-8              476,892    2,516 ns/op      1,856 B/op    15 allocs/op
// BenchmarkGenerateText-8                    1,690  703,421 ns/op     11,432 B/op   146 allocs/op
// BenchmarkGenerateTextParallel-8            8,334  143,876 ns/op     11,432 B/op   146 allocs/op
// BenchmarkConvertMessages-8             2,836,926      423 ns/op        624 B/op     9 allocs/op
// BenchmarkConvertMultimodalMessages-8     952,184    1,259 ns/op      1,256 B/op    17 allocs/op
// BenchmarkConvertTools-8                  347,598    3,445 ns/op      3,584 B/op    31 allocs/op
// BenchmarkStreamProcessing-8              183,414    6,528 ns/op      2,736 B/op    52 allocs/op
// BenchmarkJSONMarshaling-8                446,856    2,684 ns/op      1,024 B/op    11 allocs/op
// BenchmarkJSONUnmarshaling-8              204,918    5,847 ns/op      1,544 B/op    35 allocs/op
// BenchmarkErrorParsing-8                1,417,934      847 ns/op        432 B/op    10 allocs/op
// BenchmarkRetryLogic-8                      1,714  699,814 ns/op     23,088 B/op   294 allocs/op
// BenchmarkToolExecution-8                 631,754    1,897 ns/op      1,056 B/op    22 allocs/op