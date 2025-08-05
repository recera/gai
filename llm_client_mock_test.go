package gai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

)

// mockProviderClient is a test implementation of ProviderClient
type mockProviderClient struct {
	mockServer *httptest.Server
}

func (m *mockProviderClient) GetCompletion(ctx context.Context, parts LLMCallParts) (LLMResponse, error) {
	// For testing, we'll just return a predetermined response
	return LLMResponse{
		Content: `{"mammals": ["dog"], "birds": ["eagle"], "fish": ["salmon"]}`,
		FinishReason: "stop",
		Usage: TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 15,
			TotalTokens:      25,
		},
	}, nil
}

// TestLLMClientWithMock tests the LLM client without making real API calls
func TestLLMClientWithMock(t *testing.T) {
	// Create a mock client implementation
	mockClient := &struct {
		LLMClient
		mock *mockProviderClient
	}{
		mock: &mockProviderClient{},
	}

	// Override GetCompletion to use our mock
	mockClient.LLMClient = &testClient{
		getCompletionFunc: mockClient.mock.GetCompletion,
	}

	ctx := context.Background()
	
	type Animals struct {
		Mammals []string `json:"mammals"`
		Birds   []string `json:"birds"`
		Fish    []string `json:"fish"`
	}

	var result Animals

	// Configure LLM call
	callParts := NewLLMCallParts().
		WithProvider("openai").
		WithModel("gpt-4").
		WithUserMessage("List animals")

	// Make the call using our mock client
	if err := mockClient.GetResponseObject(ctx, *callParts, &result); err != nil {
		t.Fatalf("Failed to get response: %v", err)
	}

	// Verify the response
	if len(result.Mammals) != 1 || result.Mammals[0] != "dog" {
		t.Errorf("Expected mammals to be [dog], got %v", result.Mammals)
	}
	if len(result.Birds) != 1 || result.Birds[0] != "eagle" {
		t.Errorf("Expected birds to be [eagle], got %v", result.Birds)
	}
	if len(result.Fish) != 1 || result.Fish[0] != "salmon" {
		t.Errorf("Expected fish to be [salmon], got %v", result.Fish)
	}
}

// testClient is a test implementation that allows injecting behavior
type testClient struct {
	getCompletionFunc func(context.Context, LLMCallParts) (LLMResponse, error)
}

func (t *testClient) GetCompletion(ctx context.Context, parts LLMCallParts) (LLMResponse, error) {
	return t.getCompletionFunc(ctx, parts)
}

func (t *testClient) GetResponseObject(ctx context.Context, parts LLMCallParts, v any) error {
	response, err := t.GetCompletion(ctx, parts)
	if err != nil {
		return err
	}
	return ParseInto(response.Content, v)
}

// TestOpenAIProviderWithHTTPMock tests the OpenAI provider with a mock HTTP server
func TestOpenAIProviderWithHTTPMock(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Expected path /v1/chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization header 'Bearer test-key', got %s", r.Header.Get("Authorization"))
		}

		// Return mock response
		response := map[string]interface{}{
			"id": "test-id",
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Paris is the capital of France.",
					},
					"finish_reason": "stop",
					"index":         0,
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     5,
				"completion_tokens": 7,
				"total_tokens":      12,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create OpenAI client with mock server URL
	// Note: This would require modifying the provider to accept a custom base URL
	// For now, this demonstrates the pattern
	
	// This test demonstrates the pattern, but would need provider modifications
	// to actually inject the mock server URL
	t.Log("Mock server created at:", server.URL)
}

// TestErrorHandling tests error handling with mocks
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func() LLMClient
		wantErrType string
		wantErrMsg  string
	}{
		{
			name: "API error response",
			setupMock: func() LLMClient {
				return &testClient{
					getCompletionFunc: func(ctx context.Context, parts LLMCallParts) (LLMResponse, error) {
						err := NewLLMError(
							http.ErrBodyNotAllowed,
							"openai",
							"gpt-4",
						)
						err.StatusCode = 429
						err.LastRaw = `{"error": {"message": "Rate limit exceeded"}}`
						return LLMResponse{}, err
					},
				}
			},
			wantErrType: "*gai.LLMError",
			wantErrMsg:  "llm openai/gpt-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupMock()
			ctx := context.Background()
			parts := NewLLMCallParts().WithUserMessage("test")

			_, err := client.GetCompletion(ctx, *parts)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			// Check error type
			if _, ok := err.(*LLMError); !ok && tt.wantErrType == "*gai.LLMError" {
				t.Errorf("Expected error type %s, got %T", tt.wantErrType, err)
			}

			// Check error message contains expected string
			if tt.wantErrMsg != "" && !contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("Expected error message to contain %q, got %q", tt.wantErrMsg, err.Error())
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
		   len(s) > len(substr) && containsHelper(s[1:], substr)
}

func containsHelper(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s[:len(substr)] == substr {
		return true
	}
	return containsHelper(s[1:], substr)
}