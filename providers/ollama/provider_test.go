package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		expected func(*Provider) bool
	}{
		{
			name: "default options",
			opts: nil,
			expected: func(p *Provider) bool {
				return p.baseURL == defaultBaseURL &&
					p.model == defaultModel &&
					p.maxRetries == 3 &&
					p.retryDelay == 100*time.Millisecond &&
					p.keepAlive == "5m" &&
					p.client != nil
			},
		},
		{
			name: "custom base URL",
			opts: []Option{WithBaseURL("http://custom:11434")},
			expected: func(p *Provider) bool {
				return p.baseURL == "http://custom:11434"
			},
		},
		{
			name: "custom model",
			opts: []Option{WithModel("llama3.1")},
			expected: func(p *Provider) bool {
				return p.model == "llama3.1"
			},
		},
		{
			name: "use generate API",
			opts: []Option{WithGenerateAPI(true)},
			expected: func(p *Provider) bool {
				return p.useGenerateAPI == true
			},
		},
		{
			name: "custom keep alive",
			opts: []Option{WithKeepAlive("10m")},
			expected: func(p *Provider) bool {
				return p.keepAlive == "10m"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.opts...)
			if !tt.expected(p) {
				t.Errorf("Provider configuration doesn't match expected")
			}
		})
	}
}

func TestProvider_getModel(t *testing.T) {
	p := New(WithModel("default-model"))

	tests := []struct {
		name     string
		req      core.Request
		expected string
	}{
		{
			name:     "use request model",
			req:      core.Request{Model: "request-model"},
			expected: "request-model",
		},
		{
			name:     "use provider default",
			req:      core.Request{},
			expected: "default-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.getModel(tt.req)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestProvider_convertMessages(t *testing.T) {
	p := New()

	tests := []struct {
		name     string
		messages []core.Message
		expected []chatMessage
		wantErr  bool
	}{
		{
			name: "simple text messages",
			messages: []core.Message{
				{Role: core.System, Parts: []core.Part{core.Text{Text: "You are helpful"}}},
				{Role: core.User, Parts: []core.Part{core.Text{Text: "Hello"}}},
				{Role: core.Assistant, Parts: []core.Part{core.Text{Text: "Hi there!"}}},
			},
			expected: []chatMessage{
				{Role: "system", Content: "You are helpful"},
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
			wantErr: false,
		},
		{
			name: "message with image",
			messages: []core.Message{
				{Role: core.User, Parts: []core.Part{
					core.Text{Text: "What's in this image?"},
					core.ImageURL{URL: "data:image/jpeg;base64,abc123"},
				}},
			},
			expected: []chatMessage{
				{
					Role:    "user",
					Content: "What's in this image?",
					Images:  []string{"data:image/jpeg;base64,abc123"},
				},
			},
			wantErr: false,
		},
		{
			name: "unsupported part type",
			messages: []core.Message{
				{Role: core.User, Parts: []core.Part{
					core.Audio{Source: core.BlobRef{URL: "audio.wav"}},
				}},
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.convertMessages(tt.messages)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got error: %v", tt.wantErr, err)
				return
			}
			if !tt.wantErr {
				if len(result) != len(tt.expected) {
					t.Errorf("expected %d messages, got %d", len(tt.expected), len(result))
					return
				}
				for i, expected := range tt.expected {
					if result[i].Role != expected.Role || result[i].Content != expected.Content {
						t.Errorf("message %d: expected {%s, %s}, got {%s, %s}",
							i, expected.Role, expected.Content, result[i].Role, result[i].Content)
					}
				}
			}
		})
	}
}

func TestProvider_convertTools(t *testing.T) {
	p := New()

	// Mock tool
	mockTool := &mockToolHandle{
		name: "test_tool",
		desc: "A test tool",
		inSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"input": {"type": "string"}
			}
		}`),
	}

	tools := []core.ToolHandle{mockTool}
	result := p.convertTools(tools)

	if len(result) != 1 {
		t.Errorf("expected 1 tool, got %d", len(result))
		return
	}

	tool := result[0]
	if tool.Type != "function" {
		t.Errorf("expected type 'function', got '%s'", tool.Type)
	}
	if tool.Function.Name != "test_tool" {
		t.Errorf("expected name 'test_tool', got '%s'", tool.Function.Name)
	}
	if tool.Function.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got '%s'", tool.Function.Description)
	}
}

func TestProvider_applyProviderOptions(t *testing.T) {
	p := New()
	req := &chatRequest{Options: &modelOptions{}}

	opts := map[string]interface{}{
		"top_k":             10,
		"top_p":             float32(0.9),
		"repeat_penalty":    float32(1.1),
		"seed":              42,
		"num_ctx":           2048,
		"num_gpu":           1,
		"low_vram":          true,
		"stop":              []string{".", "!", "?"},
		"frequency_penalty": float32(0.1),
		"presence_penalty":  float32(0.1),
		"mirostat":          1,
		"mirostat_eta":      float32(0.1),
		"mirostat_tau":      float32(5.0),
	}

	p.applyProviderOptions(req, opts)

	if *req.Options.TopK != 10 {
		t.Errorf("expected TopK 10, got %d", *req.Options.TopK)
	}
	if *req.Options.TopP != 0.9 {
		t.Errorf("expected TopP 0.9, got %f", *req.Options.TopP)
	}
	if *req.Options.RepeatPenalty != 1.1 {
		t.Errorf("expected RepeatPenalty 1.1, got %f", *req.Options.RepeatPenalty)
	}
	if *req.Options.Seed != 42 {
		t.Errorf("expected Seed 42, got %d", *req.Options.Seed)
	}
}

// Mock server tests

func createMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestProvider_GenerateText_Success(t *testing.T) {
	mockResp := chatResponse{
		Model:     "test-model",
		CreatedAt: time.Now(),
		Message: &chatMessage{
			Role:    "assistant",
			Content: "Hello, world!",
		},
		Done:               true,
		PromptEvalCount:    10,
		EvalCount:          5,
		TotalDuration:      1000000000, // 1 second in nanoseconds
		LoadDuration:       100000000,  // 100ms
		PromptEvalDuration: 500000000,  // 500ms
		EvalDuration:       400000000,  // 400ms
	}

	server := createMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("expected path /api/chat, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		// Verify request body
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		// Verify stream is disabled for non-streaming
		if req.Stream != nil && *req.Stream {
			t.Errorf("expected stream to be false for GenerateText")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	})
	defer server.Close()

	p := New(WithBaseURL(server.URL))
	
	req := core.Request{
		Model:       "test-model",
		Messages:    []core.Message{{Role: core.User, Parts: []core.Part{core.Text{Text: "Hello"}}}},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	result, err := p.GenerateText(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got '%s'", result.Text)
	}
	if result.Usage.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 5 {
		t.Errorf("expected 5 output tokens, got %d", result.Usage.OutputTokens)
	}
	if result.Usage.TotalTokens != 15 {
		t.Errorf("expected 15 total tokens, got %d", result.Usage.TotalTokens)
	}
}

func TestProvider_GenerateText_Error(t *testing.T) {
	server := createMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "model not found"})
	})
	defer server.Close()

	p := New(WithBaseURL(server.URL))
	
	req := core.Request{
		Model:    "nonexistent-model",
		Messages: []core.Message{{Role: core.User, Parts: []core.Part{core.Text{Text: "Hello"}}}},
	}

	result, err := p.GenerateText(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
	if !IsModelNotFoundError(err) {
		t.Errorf("expected model not found error, got %v", err)
	}
}

func TestProvider_StreamText(t *testing.T) {
	chunks := []chatResponse{
		{
			Model:     "test-model",
			CreatedAt: time.Now(),
			Message:   &chatMessage{Role: "assistant", Content: "Hello"},
			Done:      false,
		},
		{
			Model:     "test-model",
			CreatedAt: time.Now(),
			Message:   &chatMessage{Role: "assistant", Content: ", world"},
			Done:      false,
		},
		{
			Model:             "test-model",
			CreatedAt:         time.Now(),
			Message:           &chatMessage{Role: "assistant", Content: "!"},
			Done:              true,
			PromptEvalCount:   10,
			EvalCount:         8,
			TotalDuration:     2000000000, // 2 seconds
			LoadDuration:      100000000,  // 100ms
			PromptEvalDuration: 800000000, // 800ms
			EvalDuration:       1100000000, // 1.1s
		},
	}

	server := createMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("expected path /api/chat, got %s", r.URL.Path)
		}

		// Verify request body
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		// Verify stream is enabled
		if req.Stream == nil || !*req.Stream {
			t.Errorf("expected stream to be true for StreamText")
		}

		w.Header().Set("Content-Type", "application/x-ndjson")

		for _, chunk := range chunks {
			json.NewEncoder(w).Encode(chunk)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	})
	defer server.Close()

	p := New(WithBaseURL(server.URL))
	
	req := core.Request{
		Model:    "test-model",
		Messages: []core.Message{{Role: core.User, Parts: []core.Part{core.Text{Text: "Hello"}}}},
	}

	stream, err := p.StreamText(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer stream.Close()

	var events []core.Event
	for event := range stream.Events() {
		events = append(events, event)
	}

	// Verify we got the expected events
	var textDeltas []string
	var finalUsage *core.Usage
	
	for _, event := range events {
		switch event.Type {
		case core.EventTextDelta:
			textDeltas = append(textDeltas, event.TextDelta)
		case core.EventFinish:
			finalUsage = event.Usage
		case core.EventError:
			t.Errorf("unexpected error event: %v", event.Err)
		}
	}

	expectedText := strings.Join(textDeltas, "")
	if expectedText != "Hello, world!" {
		t.Errorf("expected concatenated text 'Hello, world!', got '%s'", expectedText)
	}

	if finalUsage == nil {
		t.Errorf("expected final usage, got nil")
	} else {
		if finalUsage.InputTokens != 10 {
			t.Errorf("expected 10 input tokens, got %d", finalUsage.InputTokens)
		}
		if finalUsage.OutputTokens != 8 {
			t.Errorf("expected 8 output tokens, got %d", finalUsage.OutputTokens)
		}
	}
}

func TestProvider_GenerateObject(t *testing.T) {
	mockResp := chatResponse{
		Model:     "test-model",
		CreatedAt: time.Now(),
		Message: &chatMessage{
			Role:    "assistant",
			Content: `{"name": "John", "age": 30, "city": "New York"}`,
		},
		Done:            true,
		PromptEvalCount: 15,
		EvalCount:       10,
	}

	server := createMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("expected path /api/chat, got %s", r.URL.Path)
		}

		// Verify request has format field
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if req.Format == "" {
			t.Errorf("expected format field to be set for structured output")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	})
	defer server.Close()

	p := New(WithBaseURL(server.URL))
	
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
			"age":  map[string]interface{}{"type": "integer"},
			"city": map[string]interface{}{"type": "string"},
		},
		"required": []string{"name", "age"},
	}

	req := core.Request{
		Model:    "test-model",
		Messages: []core.Message{{Role: core.User, Parts: []core.Part{core.Text{Text: "Generate a person"}}}},
	}

	result, err := p.GenerateObject(context.Background(), req, schema)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the parsed object
	obj, ok := result.Value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", result.Value)
	}

	if obj["name"] != "John" {
		t.Errorf("expected name 'John', got %v", obj["name"])
	}
	if obj["age"] != float64(30) { // JSON numbers are float64
		t.Errorf("expected age 30, got %v", obj["age"])
	}
	if obj["city"] != "New York" {
		t.Errorf("expected city 'New York', got %v", obj["city"])
	}
}

func TestProvider_ListModels(t *testing.T) {
	mockResp := modelsResponse{
		Models: []model{
			{
				Name:       "llama3.2",
				Size:       12345678,
				Digest:     "abc123",
				ModifiedAt: time.Now(),
				Details:    map[string]string{"format": "gguf"},
			},
			{
				Name:       "llama3.1:8b",
				Size:       87654321,
				Digest:     "def456",
				ModifiedAt: time.Now(),
				Details:    map[string]string{"format": "gguf"},
			},
		},
	}

	server := createMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("expected path /api/tags, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	})
	defer server.Close()

	p := New(WithBaseURL(server.URL))

	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(models) != 2 {
		t.Errorf("expected 2 models, got %d", len(models))
	}

	if models[0].Name != "llama3.2" {
		t.Errorf("expected first model name 'llama3.2', got '%s'", models[0].Name)
	}
	if models[1].Name != "llama3.1:8b" {
		t.Errorf("expected second model name 'llama3.1:8b', got '%s'", models[1].Name)
	}
}

func TestProvider_IsModelAvailable(t *testing.T) {
	mockResp := modelsResponse{
		Models: []model{
			{Name: "llama3.2"},
			{Name: "llama3.1:8b"},
			{Name: "mistral:7b"},
		},
	}

	server := createMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	})
	defer server.Close()

	p := New(WithBaseURL(server.URL))

	tests := []struct {
		modelName string
		expected  bool
	}{
		{"llama3.2", true},
		{"llama3.1", true}, // Should match "llama3.1:8b"
		{"mistral", true},  // Should match "mistral:7b"
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			available, err := p.IsModelAvailable(context.Background(), tt.modelName)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if available != tt.expected {
				t.Errorf("expected %v for model %s, got %v", tt.expected, tt.modelName, available)
			}
		})
	}
}

// Helper types for testing

type mockToolHandle struct {
	name     string
	desc     string
	inSchema json.RawMessage
}

func (m *mockToolHandle) Name() string                                           { return m.name }
func (m *mockToolHandle) Description() string                                    { return m.desc }
func (m *mockToolHandle) InSchemaJSON() []byte                                   { return m.inSchema }
func (m *mockToolHandle) OutSchemaJSON() []byte                                  { return []byte(`{}`) }
func (m *mockToolHandle) Exec(ctx context.Context, raw json.RawMessage, meta interface{}) (any, error) {
	return map[string]string{"result": "mock"}, nil
}