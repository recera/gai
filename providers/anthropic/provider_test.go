package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/tools"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		expected *Provider
	}{
		{
			name: "default options",
			opts: nil,
			expected: &Provider{
				baseURL:    defaultBaseURL,
				model:      "claude-sonnet-4-20250514",
				maxRetries: 3,
				retryDelay: 100 * time.Millisecond,
				version:    defaultVersion,
			},
		},
		{
			name: "custom options",
			opts: []Option{
				WithAPIKey("test-key"),
				WithBaseURL("https://custom.api.com"),
				WithModel("claude-3-opus-20240229"),
				WithMaxRetries(5),
				WithRetryDelay(200 * time.Millisecond),
				WithVersion("2023-01-01"),
			},
			expected: &Provider{
				apiKey:     "test-key",
				baseURL:    "https://custom.api.com",
				model:      "claude-3-opus-20240229",
				maxRetries: 5,
				retryDelay: 200 * time.Millisecond,
				version:    "2023-01-01",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.opts...)
			
			if p.apiKey != tt.expected.apiKey {
				t.Errorf("apiKey = %v, expected %v", p.apiKey, tt.expected.apiKey)
			}
			if p.baseURL != tt.expected.baseURL {
				t.Errorf("baseURL = %v, expected %v", p.baseURL, tt.expected.baseURL)
			}
			if p.model != tt.expected.model {
				t.Errorf("model = %v, expected %v", p.model, tt.expected.model)
			}
			if p.maxRetries != tt.expected.maxRetries {
				t.Errorf("maxRetries = %v, expected %v", p.maxRetries, tt.expected.maxRetries)
			}
			if p.retryDelay != tt.expected.retryDelay {
				t.Errorf("retryDelay = %v, expected %v", p.retryDelay, tt.expected.retryDelay)
			}
			if p.version != tt.expected.version {
				t.Errorf("version = %v, expected %v", p.version, tt.expected.version)
			}
			if p.client == nil {
				t.Error("client should not be nil")
			}
		})
	}
}

func TestConvertMessages(t *testing.T) {
	p := New()

	tests := []struct {
		name           string
		input          []core.Message
		expectedMsgs   []message
		expectedSystem string
		expectError    bool
	}{
		{
			name: "simple user message",
			input: []core.Message{
				{
					Role:  core.User,
					Parts: []core.Part{core.Text{Text: "Hello"}},
				},
			},
			expectedMsgs: []message{
				{
					Role:    "user",
					Content: "Hello",
				},
			},
			expectedSystem: "",
		},
		{
			name: "system message separation",
			input: []core.Message{
				{
					Role:  core.System,
					Parts: []core.Part{core.Text{Text: "You are a helpful assistant"}},
				},
				{
					Role:  core.User,
					Parts: []core.Part{core.Text{Text: "Hello"}},
				},
			},
			expectedMsgs: []message{
				{
					Role:    "user",
					Content: "Hello",
				},
			},
			expectedSystem: "You are a helpful assistant",
		},
		{
			name: "multiple system messages",
			input: []core.Message{
				{
					Role:  core.System,
					Parts: []core.Part{core.Text{Text: "You are helpful"}},
				},
				{
					Role:  core.System,
					Parts: []core.Part{core.Text{Text: "Be concise"}},
				},
				{
					Role:  core.User,
					Parts: []core.Part{core.Text{Text: "Hello"}},
				},
			},
			expectedMsgs: []message{
				{
					Role:    "user",
					Content: "Hello",
				},
			},
			expectedSystem: "You are helpful\n\nBe concise",
		},
		{
			name: "multimodal content",
			input: []core.Message{
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: "Look at this image:"},
						core.ImageURL{URL: "data:image/jpeg;base64,/9j/4AAQ...", Detail: "high"},
					},
				},
			},
			expectedMsgs: []message{
				{
					Role: "user",
					Content: []contentBlock{
						{Type: "text", Text: "Look at this image:"},
						{
							Type: "image",
							Source: &imageSource{
								Type:      "base64",
								MediaType: "image/jpeg",
								Data:      "data:image/jpeg;base64,/9j/4AAQ...",
							},
						},
					},
				},
			},
			expectedSystem: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs, system, err := p.convertMessages(tt.input)
			
			if tt.expectError && err == nil {
				t.Error("expected error, got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if system != tt.expectedSystem {
				t.Errorf("system = %q, expected %q", system, tt.expectedSystem)
			}
			
			if len(msgs) != len(tt.expectedMsgs) {
				t.Errorf("messages length = %d, expected %d", len(msgs), len(tt.expectedMsgs))
				return
			}
			
			for i, msg := range msgs {
				expected := tt.expectedMsgs[i]
				if msg.Role != expected.Role {
					t.Errorf("message[%d].Role = %q, expected %q", i, msg.Role, expected.Role)
				}
				
				// Compare content - this is simplified for the test
				if expected.Content == nil {
					continue
				}
				
				switch expectedContent := expected.Content.(type) {
				case string:
					if msgContent, ok := msg.Content.(string); !ok {
						t.Errorf("message[%d].Content type mismatch", i)
					} else if msgContent != expectedContent {
						t.Errorf("message[%d].Content = %q, expected %q", i, msgContent, expectedContent)
					}
				case []contentBlock:
					msgContentBlocks, ok := msg.Content.([]contentBlock)
					if !ok {
						t.Errorf("message[%d].Content should be []contentBlock", i)
						continue
					}
					if len(msgContentBlocks) != len(expectedContent) {
						t.Errorf("message[%d].Content length = %d, expected %d", 
							i, len(msgContentBlocks), len(expectedContent))
					}
				}
			}
		})
	}
}

func TestConvertTools(t *testing.T) {
	p := New()
	
	// Create a mock tool
	mockTool := tools.New("test_tool", "A test tool", 
		func(ctx context.Context, input struct{
			Message string `json:"message"`
		}, meta tools.Meta) (string, error) {
			return "response", nil
		})
	
	coreTools := tools.ToCoreHandles([]tools.Handle{mockTool})
	
	result := p.convertTools(coreTools)
	
	if len(result) != 1 {
		t.Errorf("expected 1 tool, got %d", len(result))
		return
	}
	
	tool := result[0]
	if tool.Name != "test_tool" {
		t.Errorf("tool.Name = %q, expected %q", tool.Name, "test_tool")
	}
	if tool.Description != "A test tool" {
		t.Errorf("tool.Description = %q, expected %q", tool.Description, "A test tool")
	}
	if tool.InputSchema == nil {
		t.Error("tool.InputSchema should not be nil")
	}
}

func TestGenerateTextBasic(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("missing or incorrect API key header")
		}
		if r.Header.Get("anthropic-version") != defaultVersion {
			t.Errorf("missing or incorrect version header")
		}
		if r.Header.Get("content-type") != "application/json" {
			t.Errorf("missing or incorrect content type header")
		}
		
		// Mock response
		response := messagesResponse{
			ID:         "msg_123",
			Type:       "message",
			Role:       "assistant",
			Content:    []contentBlock{{Type: "text", Text: "Hello! How can I help you?"}},
			Model:      "claude-sonnet-4-20250514",
			StopReason: "end_turn",
			Usage: usage{
				InputTokens:  10,
				OutputTokens: 20,
			},
		}
		
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	p := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
	)
	
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Hello"}},
			},
		},
	}
	
	result, err := p.GenerateText(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if result.Text != "Hello! How can I help you?" {
		t.Errorf("result.Text = %q, expected %q", result.Text, "Hello! How can I help you?")
	}
	if result.Usage.InputTokens != 10 {
		t.Errorf("result.Usage.InputTokens = %d, expected %d", result.Usage.InputTokens, 10)
	}
	if result.Usage.OutputTokens != 20 {
		t.Errorf("result.Usage.OutputTokens = %d, expected %d", result.Usage.OutputTokens, 20)
	}
	if result.Usage.TotalTokens != 30 {
		t.Errorf("result.Usage.TotalTokens = %d, expected %d", result.Usage.TotalTokens, 30)
	}
}

func TestGenerateTextWithToolUse(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock response with tool use
		response := messagesResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{Type: "text", Text: "I'll help you with that calculation."},
				{
					Type: "tool_use",
					ID:   "tool_123",
					Name: "calculator",
					Input: map[string]interface{}{
						"expression": "2 + 2",
					},
				},
			},
			Model:      "claude-sonnet-4-20250514",
			StopReason: "tool_use",
			Usage: usage{
				InputTokens:  15,
				OutputTokens: 25,
			},
		}
		
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	p := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
	)
	
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "What's 2 + 2?"}},
			},
		},
	}
	
	result, err := p.GenerateText(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if result.Text != "I'll help you with that calculation." {
		t.Errorf("result.Text = %q, expected %q", result.Text, "I'll help you with that calculation.")
	}
	
	if len(result.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(result.Steps))
		return
	}
	
	step := result.Steps[0]
	if len(step.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(step.ToolCalls))
		return
	}
	
	toolCall := step.ToolCalls[0]
	if toolCall.ID != "tool_123" {
		t.Errorf("toolCall.ID = %q, expected %q", toolCall.ID, "tool_123")
	}
	if toolCall.Name != "calculator" {
		t.Errorf("toolCall.Name = %q, expected %q", toolCall.Name, "calculator")
	}
}

func TestStreamText(t *testing.T) {
	// Create a mock streaming server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}
		
		// Send streaming events
		events := []string{
			`data: {"type": "message_start", "message": {"id": "msg_123", "type": "message", "role": "assistant", "content": [], "model": "claude-sonnet-4-20250514", "stop_reason": null, "usage": {"input_tokens": 10, "output_tokens": 0}}}`,
			`data: {"type": "content_block_start", "index": 0, "content_block": {"type": "text", "text": ""}}`,
			`data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "Hello"}}`,
			`data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": " world"}}`,
			`data: {"type": "content_block_stop", "index": 0}`,
			`data: {"type": "message_delta", "delta": {"stop_reason": "end_turn"}, "usage": {"input_tokens": 10, "output_tokens": 15}}`,
			`data: {"type": "message_stop"}`,
		}
		
		for _, event := range events {
			w.Write([]byte(event + "\n\n"))
			flusher.Flush()
			time.Sleep(10 * time.Millisecond) // Small delay between events
		}
	}))
	defer server.Close()
	
	p := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
	)
	
	req := core.Request{
		Messages: []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: "Hello"}},
			},
		},
		Stream: true,
	}
	
	stream, err := p.StreamText(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer stream.Close()
	
	var events []core.Event
	var textParts []string
	
	// Collect all events
	for event := range stream.Events() {
		events = append(events, event)
		if event.Type == core.EventTextDelta {
			textParts = append(textParts, event.TextDelta)
		}
	}
	
	// Verify we got the expected events
	if len(events) == 0 {
		t.Error("expected to receive events")
		return
	}
	
	// Check that we received start and finish events
	hasStart := false
	hasFinish := false
	for _, event := range events {
		if event.Type == core.EventStart {
			hasStart = true
		}
		if event.Type == core.EventFinish {
			hasFinish = true
		}
	}
	
	if !hasStart {
		t.Error("expected start event")
	}
	if !hasFinish {
		t.Error("expected finish event")
	}
	
	// Verify text deltas
	expectedText := "Hello world"
	actualText := strings.Join(textParts, "")
	if actualText != expectedText {
		t.Errorf("accumulated text = %q, expected %q", actualText, expectedText)
	}
}

func TestErrorMapping(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedCode   core.ErrorCode
		expectedSubstr string
	}{
		{
			name:       "invalid request error",
			statusCode: 400,
			responseBody: `{
				"type": "error",
				"error": {
					"type": "invalid_request_error",
					"message": "Invalid request format"
				}
			}`,
			expectedCode:   core.ErrorInvalidRequest,
			expectedSubstr: "Invalid request format",
		},
		{
			name:       "context length exceeded",
			statusCode: 400,
			responseBody: `{
				"type": "error",
				"error": {
					"type": "invalid_request_error",
					"message": "Context length exceeded maximum allowed"
				}
			}`,
			expectedCode:   core.ErrorContextLengthExceeded,
			expectedSubstr: "Context length exceeded",
		},
		{
			name:       "authentication error",
			statusCode: 401,
			responseBody: `{
				"type": "error",
				"error": {
					"type": "authentication_error",
					"message": "Invalid API key"
				}
			}`,
			expectedCode:   core.ErrorUnauthorized,
			expectedSubstr: "Invalid API key",
		},
		{
			name:       "rate limit error",
			statusCode: 429,
			responseBody: `{
				"type": "error",
				"error": {
					"type": "rate_limit_error",
					"message": "Rate limit exceeded"
				}
			}`,
			expectedCode:   core.ErrorRateLimited,
			expectedSubstr: "Rate limit exceeded",
		},
		{
			name:       "overloaded error",
			statusCode: 529,
			responseBody: `{
				"type": "error",
				"error": {
					"type": "overloaded_error",
					"message": "Server is overloaded"
				}
			}`,
			expectedCode:   core.ErrorOverloaded,
			expectedSubstr: "Server is overloaded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock server that returns the error
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			p := New(
				WithAPIKey("test-key"),
				WithBaseURL(server.URL),
			)

			req := core.Request{
				Messages: []core.Message{
					{
						Role:  core.User,
						Parts: []core.Part{core.Text{Text: "Hello"}},
					},
				},
			}

			_, err := p.GenerateText(context.Background(), req)
			if err == nil {
				t.Error("expected error, got none")
				return
			}

			var aiErr *core.AIError
			var ok bool
			if aiErr, ok = err.(*core.AIError); !ok {
				t.Errorf("expected *core.AIError, got %T", err)
				return
			}

			if aiErr.Code != tt.expectedCode {
				t.Errorf("error code = %v, expected %v", aiErr.Code, tt.expectedCode)
			}
			if !strings.Contains(aiErr.Message, tt.expectedSubstr) {
				t.Errorf("error message %q does not contain %q", aiErr.Message, tt.expectedSubstr)
			}
			if aiErr.HTTPStatus != tt.statusCode {
				t.Errorf("HTTP status = %d, expected %d", aiErr.HTTPStatus, tt.statusCode)
			}
			if aiErr.Provider != "anthropic" {
				t.Errorf("provider = %q, expected %q", aiErr.Provider, "anthropic")
			}
		})
	}
}

func TestShouldRetry(t *testing.T) {
	p := New()

	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"200 OK", 200, false},
		{"400 Bad Request", 400, false},
		{"401 Unauthorized", 401, false},
		{"429 Rate Limited", 429, true},
		{"500 Internal Error", 500, true},
		{"502 Bad Gateway", 502, true},
		{"503 Service Unavailable", 503, true},
		{"504 Gateway Timeout", 504, true},
		{"529 Overloaded", 529, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.shouldRetry(tt.statusCode)
			if result != tt.expected {
				t.Errorf("shouldRetry(%d) = %v, expected %v", tt.statusCode, result, tt.expected)
			}
		})
	}
}