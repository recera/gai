package openai_compat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/tools"
)

// mockServer creates a test server that simulates OpenAI-compatible API responses.
func mockServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log request for debugging
		t.Logf("Mock server received: %s %s", r.Method, r.URL.Path)
		
		// Check authorization
		if auth := r.Header.Get("Authorization"); auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(errorResponse{
				Error: struct {
					Message string `json:"message"`
					Type    string `json:"type"`
					Code    string `json:"code,omitempty"`
					Param   string `json:"param,omitempty"`
				}{
					Message: "Missing API key",
					Type:    "authentication_error",
				},
			})
			return
		}
		
		// Delegate to test handler
		handler(w, r)
	}))
}

func TestProviderCreation(t *testing.T) {
	tests := []struct {
		name    string
		opts    CompatOpts
		wantErr bool
	}{
		{
			name: "valid configuration",
			opts: CompatOpts{
				BaseURL: "https://api.example.com/v1",
				APIKey:  "test-key",
			},
			wantErr: false,
		},
		{
			name: "missing base URL",
			opts: CompatOpts{
				APIKey: "test-key",
			},
			wantErr: true,
		},
		{
			name: "invalid base URL",
			opts: CompatOpts{
				BaseURL: "not a url",
				APIKey:  "test-key",
			},
			wantErr: true,
		},
		{
			name: "URL without /v1 suffix",
			opts: CompatOpts{
				BaseURL: "https://api.example.com",
				APIKey:  "test-key",
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := New(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && p == nil {
				t.Error("New() returned nil provider without error")
			}
		})
	}
}

func TestGenerateText(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		
		// Parse request
		var req chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		
		// Return mock response
		resp := chatCompletionResponse{
			ID:      "chatcmpl-test",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []choice{
				{
					Index: 0,
					Message: chatMessage{
						Role:    "assistant",
						Content: "This is a test response.",
					},
					FinishReason: "stop",
				},
			},
			Usage: usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()
	
	provider, err := New(CompatOpts{
		BaseURL:      server.URL,
		APIKey:       "test-key",
		DefaultModel: "test-model",
		ProviderName: "test",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	
	ctx := context.Background()
	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Hello, world!"},
				},
			},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	})
	
	if err != nil {
		t.Fatalf("GenerateText failed: %v", err)
	}
	
	if result.Text != "This is a test response." {
		t.Errorf("Expected text 'This is a test response.', got %q", result.Text)
	}
	
	if result.Usage.TotalTokens != 15 {
		t.Errorf("Expected 15 total tokens, got %d", result.Usage.TotalTokens)
	}
}

func TestStreamText(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		
		// Send streaming chunks
		chunks := []string{
			`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
			`{"id":"2","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
			`{"id":"3","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":3,"total_tokens":13}}`,
		}
		
		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			w.(http.Flusher).Flush()
			time.Sleep(10 * time.Millisecond) // Simulate streaming delay
		}
		
		fmt.Fprintf(w, "data: [DONE]\n\n")
		w.(http.Flusher).Flush()
	})
	defer server.Close()
	
	provider, err := New(CompatOpts{
		BaseURL:      server.URL,
		APIKey:       "test-key",
		DefaultModel: "test-model",
		ProviderName: "test",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	
	ctx := context.Background()
	stream, err := provider.StreamText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Hello!"},
				},
			},
		},
		Stream: true,
	})
	
	if err != nil {
		t.Fatalf("StreamText failed: %v", err)
	}
	defer stream.Close()
	
	var fullText strings.Builder
	var usage *core.Usage
	
	for event := range stream.Events() {
		switch event.Type {
		case core.EventTextDelta:
			fullText.WriteString(event.TextDelta)
		case core.EventFinish:
			usage = event.Usage
		case core.EventError:
			t.Fatalf("Stream error: %v", event.Err)
		}
	}
	
	expectedText := "Hello world!"
	if fullText.String() != expectedText {
		t.Errorf("Expected text %q, got %q", expectedText, fullText.String())
	}
	
	if usage == nil || usage.TotalTokens != 13 {
		t.Errorf("Expected usage with 13 total tokens, got %v", usage)
	}
}

func TestToolCalling(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		
		// Parse request
		var req chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		
		// Check if tools are present
		if len(req.Tools) > 0 {
			// Return tool call response
			resp := chatCompletionResponse{
				ID:      "chatcmpl-test",
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   req.Model,
				Choices: []choice{
					{
						Index: 0,
						Message: chatMessage{
							Role:    "assistant",
							Content: "",
							ToolCalls: []toolCall{
								{
									ID:   "call_1",
									Type: "function",
									Function: functionCall{
										Name:      "get_weather",
										Arguments: `{"location":"San Francisco"}`,
									},
								},
							},
						},
						FinishReason: "tool_calls",
					},
				},
				Usage: usage{
					PromptTokens:     20,
					CompletionTokens: 10,
					TotalTokens:      30,
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		} else {
			// Return final response after tool execution
			resp := chatCompletionResponse{
				ID:      "chatcmpl-test2",
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   req.Model,
				Choices: []choice{
					{
						Index: 0,
						Message: chatMessage{
							Role:    "assistant",
							Content: "The weather in San Francisco is 72°F and sunny.",
						},
						FinishReason: "stop",
					},
				},
				Usage: usage{
					PromptTokens:     30,
					CompletionTokens: 15,
					TotalTokens:      45,
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	})
	defer server.Close()
	
	// Create a test tool
	type WeatherInput struct {
		Location string `json:"location"`
	}
	type WeatherOutput struct {
		Temperature float64 `json:"temperature"`
		Description string  `json:"description"`
	}
	
	weatherTool := tools.New[WeatherInput, WeatherOutput](
		"get_weather",
		"Get the current weather for a location",
		func(ctx context.Context, in WeatherInput, meta tools.Meta) (WeatherOutput, error) {
			return WeatherOutput{
				Temperature: 72.0,
				Description: "Sunny",
			}, nil
		},
	)
	
	provider, err := New(CompatOpts{
		BaseURL:      server.URL,
		APIKey:       "test-key",
		DefaultModel: "test-model",
		ProviderName: "test",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	
	ctx := context.Background()
	result, err := provider.GenerateText(ctx, core.Request{
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
		StopWhen:   core.MaxSteps(2),
	})
	
	if err != nil {
		t.Fatalf("GenerateText with tools failed: %v", err)
	}
	
	if !strings.Contains(result.Text, "72") || !strings.Contains(result.Text, "sunny") {
		t.Errorf("Expected text to contain weather info, got %q", result.Text)
	}
	
	if len(result.Steps) == 0 {
		t.Error("Expected at least one step with tool execution")
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		errorType    string
		errorCode    string
		expectedCode core.ErrorCode
	}{
		{
			name:         "rate limit error",
			statusCode:   429,
			errorType:    "rate_limit_error",
			errorCode:    "rate_limit_exceeded",
			expectedCode: core.ErrorRateLimited,
		},
		{
			name:         "authentication error",
			statusCode:   401,
			errorType:    "authentication_error",
			errorCode:    "invalid_api_key",
			expectedCode: core.ErrorUnauthorized,
		},
		{
			name:         "context length error",
			statusCode:   400,
			errorType:    "invalid_request_error",
			errorCode:    "context_length_exceeded",
			expectedCode: core.ErrorContextLengthExceeded,
		},
		{
			name:         "server error",
			statusCode:   500,
			errorType:    "server_error",
			errorCode:    "",
			expectedCode: core.ErrorInternal,
		},
		{
			name:         "overloaded error",
			statusCode:   503,
			errorType:    "engine_overloaded_error",
			errorCode:    "",
			expectedCode: core.ErrorOverloaded,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(errorResponse{
					Error: struct {
						Message string `json:"message"`
						Type    string `json:"type"`
						Code    string `json:"code,omitempty"`
						Param   string `json:"param,omitempty"`
					}{
						Message: "Test error",
						Type:    tt.errorType,
						Code:    tt.errorCode,
					},
				})
			})
			defer server.Close()
			
			provider, err := New(CompatOpts{
				BaseURL:      server.URL,
				APIKey:       "test-key",
				ProviderName: "test",
				MaxRetries:   0, // Don't retry for this test
			})
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}
			
			ctx := context.Background()
			_, err = provider.GenerateText(ctx, core.Request{
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "Test"},
						},
					},
				},
			})
			
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			
			aiErr, ok := err.(*core.AIError)
			if !ok {
				t.Fatalf("Expected core.AIError, got %T", err)
			}
			
			if aiErr.Code != tt.expectedCode {
				t.Errorf("Expected error code %v, got %v", tt.expectedCode, aiErr.Code)
			}
		})
	}
}

func TestProviderQuirks(t *testing.T) {
	tests := []struct {
		name       string
		opts       CompatOpts
		checkFunc  func(t *testing.T, req *chatCompletionRequest)
	}{
		{
			name: "disable parallel tool calls",
			opts: CompatOpts{
				DisableParallelToolCalls: true,
			},
			checkFunc: func(t *testing.T, req *chatCompletionRequest) {
				if req.ParallelToolCalls != nil {
					t.Error("Expected ParallelToolCalls to be nil")
				}
			},
		},
		{
			name: "disable tool choice",
			opts: CompatOpts{
				DisableToolChoice: true,
			},
			checkFunc: func(t *testing.T, req *chatCompletionRequest) {
				if len(req.Tools) != 0 {
					t.Error("Expected Tools to be empty when DisableToolChoice is true")
				}
			},
		},
		{
			name: "unsupported params stripped",
			opts: CompatOpts{
				UnsupportedParams: []string{"seed", "top_p"},
			},
			checkFunc: func(t *testing.T, req *chatCompletionRequest) {
				if req.Seed != nil {
					t.Error("Expected Seed to be nil")
				}
				if req.TopP != nil {
					t.Error("Expected TopP to be nil")
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedReq *chatCompletionRequest
			
			server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
				// Capture the request
				var req chatCompletionRequest
				json.NewDecoder(r.Body).Decode(&req)
				capturedReq = &req
				
				// Return minimal response
				resp := chatCompletionResponse{
					ID:      "test",
					Choices: []choice{{Message: chatMessage{Content: "test"}}},
				}
				json.NewEncoder(w).Encode(resp)
			})
			defer server.Close()
			
			tt.opts.BaseURL = server.URL
			tt.opts.APIKey = "test-key"
			
			provider, err := New(tt.opts)
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}
			
			// Create request with various options
			req := core.Request{
				Messages: []core.Message{
					{Role: core.User, Parts: []core.Part{core.Text{Text: "Test"}}},
				},
				ProviderOptions: map[string]any{
					"seed":  42,
					"top_p": 0.9,
				},
			}
			
			// Add tools if not disabled
			if !tt.opts.DisableToolChoice {
				type TestInput struct{}
				type TestOutput struct{}
				tool := tools.New[TestInput, TestOutput](
					"test_tool", "Test tool",
					func(ctx context.Context, in TestInput, meta tools.Meta) (TestOutput, error) {
						return TestOutput{}, nil
					},
				)
				req.Tools = []core.ToolHandle{tools.NewCoreAdapter(tool)}
			}
			
			ctx := context.Background()
			provider.GenerateText(ctx, req)
			
			if capturedReq != nil {
				tt.checkFunc(t, capturedReq)
			}
		})
	}
}

func TestRetryLogic(t *testing.T) {
	attemptCount := 0
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		
		if attemptCount < 3 {
			// Fail the first two attempts
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(errorResponse{
				Error: struct {
					Message string `json:"message"`
					Type    string `json:"type"`
					Code    string `json:"code,omitempty"`
					Param   string `json:"param,omitempty"`
				}{
					Message: "Service temporarily unavailable",
					Type:    "server_error",
				},
			})
			return
		}
		
		// Succeed on the third attempt
		resp := chatCompletionResponse{
			ID:      "test",
			Choices: []choice{{Message: chatMessage{Content: "Success after retry"}}},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()
	
	provider, err := New(CompatOpts{
		BaseURL:      server.URL,
		APIKey:       "test-key",
		MaxRetries:   3,
		RetryDelay:   10 * time.Millisecond, // Short delay for testing
		ProviderName: "test",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	
	ctx := context.Background()
	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Test"}}},
		},
	})
	
	if err != nil {
		t.Fatalf("GenerateText failed after retries: %v", err)
	}
	
	if result.Text != "Success after retry" {
		t.Errorf("Expected 'Success after retry', got %q", result.Text)
	}
	
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestCapabilityProbing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			// Return model list
			resp := modelsResponse{
				Object: "list",
				Data: []ModelInfo{
					{
						ID:            "test-model-1",
						Object:        "model",
						Created:       time.Now().Unix(),
						OwnedBy:       "test",
						ContextWindow: 8192,
					},
					{
						ID:            "test-model-vision",
						Object:        "model",
						Created:       time.Now().Unix(),
						OwnedBy:       "test",
						ContextWindow: 16384,
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		
		// Default response for other endpoints
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()
	
	provider, err := New(CompatOpts{
		BaseURL:      server.URL,
		APIKey:       "test-key",
		ProviderName: "test",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	
	// Wait for async probing to complete
	time.Sleep(100 * time.Millisecond)
	
	caps := provider.GetCapabilities()
	if caps == nil {
		t.Fatal("Expected capabilities to be set")
	}
	
	if len(caps.Models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(caps.Models))
	}
	
	if !caps.SupportsVision {
		t.Error("Expected SupportsVision to be true due to 'vision' in model name")
	}
	
	if caps.MaxContextWindow != 16384 {
		t.Errorf("Expected MaxContextWindow to be 16384, got %d", caps.MaxContextWindow)
	}
}