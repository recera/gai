package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/tools"
)

// mockServer creates a test server for OpenAI API mocking.
type mockServer struct {
	*httptest.Server
	requests []interface{}
	mu       sync.Mutex
}

func newMockServer() *mockServer {
	m := &mockServer{
		requests: make([]interface{}, 0),
	}
	
	m.Server = httptest.NewServer(http.HandlerFunc(m.handler))
	return m
}

func (m *mockServer) handler(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Parse request body
	body, _ := io.ReadAll(r.Body)
	var req interface{}
	json.Unmarshal(body, &req)
	m.requests = append(m.requests, req)

	// Route based on path
	switch r.URL.Path {
	case "/chat/completions":
		m.handleChatCompletions(w, r, body)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (m *mockServer) handleChatCompletions(w http.ResponseWriter, r *http.Request, body []byte) {
	var req chatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Check authorization
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, `{"error": {"message": "Missing API key", "type": "auth_error"}}`, http.StatusUnauthorized)
		return
	}

	if req.Stream {
		m.handleStreamingResponse(w, req)
	} else {
		m.handleNonStreamingResponse(w, req)
	}
}

func (m *mockServer) handleNonStreamingResponse(w http.ResponseWriter, req chatCompletionRequest) {
	// Generate mock response
	resp := chatCompletionResponse{
		ID:      "chatcmpl-test123",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []choice{
			{
				Index: 0,
				Message: chatMessage{
					Role:    "assistant",
					Content: "This is a test response from the mock server.",
				},
				FinishReason: "stop",
			},
		},
		Usage: usage{
			PromptTokens:     10,
			CompletionTokens: 8,
			TotalTokens:      18,
		},
	}

	// Handle tool calls if tools are present
	if len(req.Tools) > 0 && req.ToolChoice != "none" {
		resp.Choices[0].Message.ToolCalls = []toolCall{
			{
				ID:   "call_test123",
				Type: "function",
				Function: functionCall{
					Name:      req.Tools[0].Function.Name,
					Arguments: `{"test": "value"}`,
				},
			},
		}
		resp.Choices[0].FinishReason = "tool_calls"
	}

	// Handle structured output
	if req.ResponseFormat != nil && req.ResponseFormat.Type == "json_schema" {
		resp.Choices[0].Message.Content = `{"result": "structured output test"}`
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (m *mockServer) handleStreamingResponse(w http.ResponseWriter, req chatCompletionRequest) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send streaming chunks
	chunks := []string{
		"This ",
		"is ",
		"a ",
		"streaming ",
		"test ",
		"response.",
	}

	for i, text := range chunks {
		chunk := streamChunk{
			ID:      "chatcmpl-test123",
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []deltaChoice{
				{
					Index: i,
					Delta: messageDelta{
						Content: &text,
					},
				},
			},
		}

		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		time.Sleep(10 * time.Millisecond) // Simulate delay
	}

	// Send usage in final chunk
	finalChunk := streamChunk{
		ID:      "chatcmpl-test123",
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []deltaChoice{},
		Usage: &usage{
			PromptTokens:     10,
			CompletionTokens: 6,
			TotalTokens:      16,
		},
	}
	data, _ := json.Marshal(finalChunk)
	fmt.Fprintf(w, "data: %s\n\n", data)

	// Send done marker
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func TestProviderCreation(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		want struct {
			apiKey  string
			baseURL string
			model   string
		}
	}{
		{
			name: "default configuration",
			opts: []Option{
				WithAPIKey("test-key"),
			},
			want: struct {
				apiKey  string
				baseURL string
				model   string
			}{
				apiKey:  "test-key",
				baseURL: defaultBaseURL,
				model:   "gpt-4o-mini",
			},
		},
		{
			name: "custom configuration",
			opts: []Option{
				WithAPIKey("custom-key"),
				WithBaseURL("https://custom.api.com"),
				WithModel("gpt-4"),
				WithOrganization("org-123"),
				WithProject("proj-456"),
			},
			want: struct {
				apiKey  string
				baseURL string
				model   string
			}{
				apiKey:  "custom-key",
				baseURL: "https://custom.api.com",
				model:   "gpt-4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.opts...)
			
			if p.apiKey != tt.want.apiKey {
				t.Errorf("apiKey = %v, want %v", p.apiKey, tt.want.apiKey)
			}
			if p.baseURL != tt.want.baseURL {
				t.Errorf("baseURL = %v, want %v", p.baseURL, tt.want.baseURL)
			}
			if p.model != tt.want.model {
				t.Errorf("model = %v, want %v", p.model, tt.want.model)
			}
		})
	}
}

func TestGenerateText(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	p := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
	)

	tests := []struct {
		name    string
		request core.Request
		wantErr bool
	}{
		{
			name: "simple text generation",
			request: core.Request{
				Model: "gpt-4o-mini",
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "Hello, how are you?"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "with system message",
			request: core.Request{
				Model: "gpt-4o-mini",
				Messages: []core.Message{
					{
						Role: core.System,
						Parts: []core.Part{
							core.Text{Text: "You are a helpful assistant."},
						},
					},
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "What is the capital of France?"},
						},
					},
				},
				Temperature: 0.7,
				MaxTokens:   100,
			},
			wantErr: false,
		},
		{
			name: "multimodal message",
			request: core.Request{
				Model: "gpt-4o",
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "What's in this image?"},
							core.ImageURL{URL: "https://example.com/image.jpg"},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := p.GenerateText(ctx, tt.request)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if result == nil {
					t.Error("GenerateText() returned nil result")
					return
				}
				if result.Text == "" {
					t.Error("GenerateText() returned empty text")
				}
				if result.Usage.TotalTokens == 0 {
					t.Error("GenerateText() returned zero usage")
				}
			}
		})
	}
}

func TestGenerateTextWithTools(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	p := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
	)

	// Create a test tool
	type WeatherInput struct {
		Location string `json:"location"`
	}
	type WeatherOutput struct {
		Temperature float64 `json:"temperature"`
		Conditions  string  `json:"conditions"`
	}

	weatherTool := tools.New[WeatherInput, WeatherOutput](
		"get_weather",
		"Get current weather for a location",
		func(ctx context.Context, in WeatherInput, meta tools.Meta) (WeatherOutput, error) {
			return WeatherOutput{
				Temperature: 72.5,
				Conditions:  "Sunny",
			}, nil
		},
	)

	ctx := context.Background()
	result, err := p.GenerateText(ctx, core.Request{
		Model: "gpt-4o-mini",
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What's the weather in San Francisco?"},
				},
			},
		},
		Tools:      []core.ToolHandle{tools.NewCoreAdapter(weatherTool)},
		ToolChoice: core.ToolAuto,
	})

	if err != nil {
		t.Fatalf("GenerateText with tools failed: %v", err)
	}

	if result == nil {
		t.Fatal("GenerateText with tools returned nil result")
	}

	// Check that tool calls were detected
	if len(result.Steps) > 0 && len(result.Steps[0].ToolCalls) == 0 {
		t.Error("Expected tool calls in result")
	}
}

func TestStreamText(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	p := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
	)

	ctx := context.Background()
	stream, err := p.StreamText(ctx, core.Request{
		Model: "gpt-4o-mini",
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Tell me a story."},
				},
			},
		},
		Stream: true,
	})

	if err != nil {
		t.Fatalf("StreamText failed: %v", err)
	}

	if stream == nil {
		t.Fatal("StreamText returned nil stream")
	}

	// Collect events
	var events []core.Event
	eventChan := stream.Events()
	done := make(chan struct{})

	go func() {
		defer close(done)
		for event := range eventChan {
			events = append(events, event)
		}
	}()

	// Wait for completion with timeout
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Stream timeout")
	}

	// Close stream
	if err := stream.Close(); err != nil {
		t.Errorf("Stream close error: %v", err)
	}

	// Verify events
	if len(events) == 0 {
		t.Fatal("No events received from stream")
	}

	// Check for expected event types
	hasStart := false
	hasTextDelta := false
	hasFinish := false

	for _, event := range events {
		switch event.Type {
		case core.EventStart:
			hasStart = true
		case core.EventTextDelta:
			hasTextDelta = true
		case core.EventFinish:
			hasFinish = true
		}
	}

	if !hasStart {
		t.Error("Missing start event")
	}
	if !hasTextDelta {
		t.Error("Missing text delta events")
	}
	if !hasFinish {
		t.Error("Missing finish event")
	}
}

func TestGenerateObject(t *testing.T) {
	server := newMockServer()
	defer server.Close()

	p := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
	)

	type TestResponse struct {
		Result string `json:"result"`
	}

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"result": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"result"},
	}

	ctx := context.Background()
	result, err := p.GenerateObject(ctx, core.Request{
		Model: "gpt-4o-mini",
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Generate a structured response."},
				},
			},
		},
	}, schema)

	if err != nil {
		t.Fatalf("GenerateObject failed: %v", err)
	}

	if result == nil {
		t.Fatal("GenerateObject returned nil result")
	}

	if result.Value == nil {
		t.Error("GenerateObject returned nil value")
	}

	// Check usage
	if result.Usage.TotalTokens == 0 {
		t.Error("GenerateObject returned zero usage")
	}
}

func TestRetryLogic(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 4 {
			// Return 503 for first three attempts
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}
		// Success on fourth attempt
		resp := chatCompletionResponse{
			ID:      "success",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
			Choices: []choice{
				{
					Message: chatMessage{
						Role:    "assistant",
						Content: "Success after retries",
					},
				},
			},
			Usage: usage{
				TotalTokens: 10,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
		WithMaxRetries(3),
		WithRetryDelay(10*time.Millisecond),
	)

	ctx := context.Background()
	result, err := p.GenerateText(ctx, core.Request{
		Model: "gpt-4o-mini",
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Test retry"},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("GenerateText with retries failed: %v", err)
	}

	if result == nil || result.Text != "Success after retries" {
		t.Error("Retry logic did not work correctly")
	}

	// maxRetries=3 means 1 initial attempt + 3 retries = 4 total attempts
	if attemptCount != 4 {
		t.Errorf("Expected 4 attempts (1 initial + 3 retries), got %d", attemptCount)
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   `{"error": {"message": "Invalid API key", "type": "auth_error", "code": "invalid_api_key"}}`,
			wantErr:    true,
		},
		{
			name:       "rate limit",
			statusCode: http.StatusTooManyRequests,
			response:   `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error", "code": "rate_limit"}}`,
			wantErr:    true,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			response:   `{"error": {"message": "Invalid request", "type": "invalid_request_error", "code": "invalid_request"}}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			p := New(
				WithAPIKey("test-key"),
				WithBaseURL(server.URL),
				WithMaxRetries(0), // Disable retries for error testing
			)

			ctx := context.Background()
			_, err := p.GenerateText(ctx, core.Request{
				Model: "gpt-4o-mini",
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "Test"},
						},
					},
				},
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("Error handling: error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				// Verify error is properly wrapped as core.AIError
				if _, ok := err.(*core.AIError); !ok {
					t.Errorf("Expected core.AIError, got %T", err)
				}
			}
		})
	}
}

// Helper functions
func floatPtr(f float32) *float32 {
	return &f
}

func intPtr(i int) *int {
	return &i
}