package openai

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/recera/gai/core"
)

func TestMapError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		headers        map[string]string
		expectedCode   core.ErrorCode
		expectedRetry  bool
		checkRetryTime bool
	}{
		{
			name:       "Context Length Exceeded",
			statusCode: http.StatusBadRequest,
			responseBody: `{
				"error": {
					"message": "This model's maximum context length is 4097 tokens",
					"type": "invalid_request_error",
					"code": "context_length_exceeded"
				}
			}`,
			expectedCode:  core.ErrorContextLengthExceeded,
			expectedRetry: false,
		},
		{
			name:       "Model Not Found",
			statusCode: http.StatusNotFound,
			responseBody: `{
				"error": {
					"message": "The model 'gpt-5' does not exist",
					"type": "invalid_request_error",
					"code": "model_not_found"
				}
			}`,
			expectedCode:  core.ErrorNotFound,
			expectedRetry: false,
		},
		{
			name:       "Rate Limited with Retry-After",
			statusCode: http.StatusTooManyRequests,
			responseBody: `{
				"error": {
					"message": "Rate limit reached for requests",
					"type": "rate_limit_error",
					"code": "rate_limit_exceeded"
				}
			}`,
			headers: map[string]string{
				"Retry-After": "30",
			},
			expectedCode:   core.ErrorRateLimited,
			expectedRetry:  true,
			checkRetryTime: true,
		},
		{
			name:       "Content Filter",
			statusCode: http.StatusBadRequest,
			responseBody: `{
				"error": {
					"message": "The content was filtered due to policy violations",
					"type": "invalid_request_error",
					"code": "content_filter"
				}
			}`,
			expectedCode:  core.ErrorSafetyBlocked,
			expectedRetry: false,
		},
		{
			name:       "Authentication Error",
			statusCode: http.StatusUnauthorized,
			responseBody: `{
				"error": {
					"message": "Incorrect API key provided",
					"type": "authentication_error",
					"code": "invalid_api_key"
				}
			}`,
			expectedCode:  core.ErrorUnauthorized,
			expectedRetry: false,
		},
		{
			name:       "Permission Error",
			statusCode: http.StatusForbidden,
			responseBody: `{
				"error": {
					"message": "You are not allowed to use this model",
					"type": "permission_error",
					"code": "model_permission_denied"
				}
			}`,
			expectedCode:  core.ErrorForbidden,
			expectedRetry: false,
		},
		{
			name:       "Server Error",
			statusCode: http.StatusInternalServerError,
			responseBody: `{
				"error": {
					"message": "The server had an error while processing your request",
					"type": "server_error",
					"code": null
				}
			}`,
			expectedCode:  core.ErrorInternal,
			expectedRetry: true,
		},
		{
			name:       "Engine Overloaded",
			statusCode: http.StatusServiceUnavailable,
			responseBody: `{
				"error": {
					"message": "The engine is currently overloaded, please try again later",
					"type": "engine_overloaded_error",
					"code": null
				}
			}`,
			expectedCode:  core.ErrorOverloaded,
			expectedRetry: true,
		},
		{
			name:       "Bad Gateway",
			statusCode: http.StatusBadGateway,
			responseBody: `{
				"error": {
					"message": "Bad gateway",
					"type": "server_error",
					"code": null
				}
			}`,
			expectedCode:  core.ErrorNetwork,
			expectedRetry: true,
		},
		{
			name:       "Gateway Timeout",
			statusCode: http.StatusGatewayTimeout,
			responseBody: `{
				"error": {
					"message": "Request timeout",
					"type": "server_error",
					"code": null
				}
			}`,
			expectedCode:  core.ErrorTimeout,
			expectedRetry: true,
		},
		{
			name:          "Malformed JSON Response",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `<!DOCTYPE html><html><body>Server Error</body></html>`,
			expectedCode:  core.ErrorInternal,
			expectedRetry: true,
		},
		{
			name:          "Empty Response Body",
			statusCode:    http.StatusServiceUnavailable,
			responseBody:  "",
			expectedCode:  core.ErrorProviderUnavailable,
			expectedRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock response
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(bytes.NewBufferString(tt.responseBody)),
				Header:     make(http.Header),
			}

			// Add headers if provided
			for k, v := range tt.headers {
				resp.Header.Set(k, v)
			}

			// Map the error
			err := MapError(resp)

			// Check that we got an AIError
			aiErr, ok := err.(*core.AIError)
			if !ok {
				t.Fatalf("Expected *core.AIError, got %T", err)
			}

			// Check error code
			if aiErr.Code != tt.expectedCode {
				t.Errorf("Expected error code %s, got %s", tt.expectedCode, aiErr.Code)
			}

			// Check provider
			if aiErr.Provider != "openai" {
				t.Errorf("Expected provider 'openai', got %s", aiErr.Provider)
			}

			// Check HTTP status
			if aiErr.HTTPStatus != tt.statusCode {
				t.Errorf("Expected HTTP status %d, got %d", tt.statusCode, aiErr.HTTPStatus)
			}

			// Check retryability
			isRetryable := IsRetryable(err)
			if isRetryable != tt.expectedRetry {
				t.Errorf("Expected retryable=%v, got %v", tt.expectedRetry, isRetryable)
			}

			// Check retry-after if specified
			if tt.checkRetryTime && tt.headers["Retry-After"] != "" {
				if aiErr.RetryAfter == nil {
					t.Error("Expected RetryAfter to be set, but it was nil")
				} else {
					expectedSeconds := 30
					if aiErr.RetryAfter.Seconds() != float64(expectedSeconds) {
						t.Errorf("Expected RetryAfter of %d seconds, got %v", expectedSeconds, aiErr.RetryAfter)
					}
				}
			}
		})
	}
}

func TestMapErrorTypeAndCode(t *testing.T) {
	tests := []struct {
		errorType    string
		errorCode    string
		statusCode   int
		expectedCode core.ErrorCode
	}{
		// Specific error codes take precedence
		{"invalid_request_error", "context_length_exceeded", 400, core.ErrorContextLengthExceeded},
		{"invalid_request_error", "model_not_found", 404, core.ErrorNotFound},
		{"rate_limit_error", "insufficient_quota", 429, core.ErrorRateLimited},
		{"invalid_request_error", "content_filter", 400, core.ErrorSafetyBlocked},
		
		// Error types when no specific code
		{"authentication_error", "", 401, core.ErrorUnauthorized},
		{"permission_error", "", 403, core.ErrorForbidden},
		{"rate_limit_error", "", 429, core.ErrorRateLimited},
		{"engine_overloaded_error", "", 503, core.ErrorOverloaded},
		
		// Server errors with different status codes
		{"server_error", "", 500, core.ErrorInternal},
		{"server_error", "", 503, core.ErrorProviderUnavailable},
		{"server_error", "", 504, core.ErrorTimeout},
		
		// Invalid request with different status codes
		{"invalid_request_error", "", 400, core.ErrorInvalidRequest},
		{"invalid_request_error", "", 404, core.ErrorNotFound},
		{"invalid_request_error", "", 413, core.ErrorContextLengthExceeded},
	}

	for _, tt := range tests {
		name := tt.errorType
		if tt.errorCode != "" {
			name += "/" + tt.errorCode
		}
		t.Run(name, func(t *testing.T) {
			code := mapErrorTypeAndCode(tt.errorType, tt.errorCode, tt.statusCode)
			if code != tt.expectedCode {
				t.Errorf("Expected %s, got %s", tt.expectedCode, code)
			}
		})
	}
}

func TestExtractModelFromError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedModel string
	}{
		{
			name: "Model in error struct",
			err: core.NewError(
				core.ErrorNotFound,
				"Model not found",
				core.WithModel("gpt-4"),
			),
			expectedModel: "gpt-4",
		},
		{
			name: "Model in message",
			err: core.NewError(
				core.ErrorNotFound,
				"The model `gpt-3.5-turbo` does not exist",
			),
			expectedModel: "gpt-3.5-turbo",
		},
		{
			name: "No model information",
			err: core.NewError(
				core.ErrorInternal,
				"Internal server error",
			),
			expectedModel: "",
		},
		{
			name:          "Non-AIError",
			err:           http.ErrBodyNotAllowed,
			expectedModel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := ExtractModelFromError(tt.err)
			if model != tt.expectedModel {
				t.Errorf("Expected model %q, got %q", tt.expectedModel, model)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		value    string
		expected int
	}{
		{"30", 30},
		{"60", 60},
		{"0", 0},
		{"invalid", 0},
		{"", 0},
		{"30.5", 30}, // Should parse integer part
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := parseRetryAfter(tt.value)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Rate limited error",
			err:      core.NewError(core.ErrorRateLimited, "Rate limited"),
			expected: true,
		},
		{
			name:     "Overloaded error",
			err:      core.NewError(core.ErrorOverloaded, "System overloaded"),
			expected: true,
		},
		{
			name:     "Network error",
			err:      core.NewError(core.ErrorNetwork, "Network error"),
			expected: true,
		},
		{
			name:     "Auth error (not retryable)",
			err:      core.NewError(core.ErrorUnauthorized, "Invalid API key"),
			expected: false,
		},
		{
			name:     "Bad request (not retryable)",
			err:      core.NewError(core.ErrorInvalidRequest, "Invalid parameters"),
			expected: false,
		},
		{
			name: "502 Bad Gateway",
			err: core.NewError(
				core.ErrorNetwork,
				"Bad gateway",
				core.WithHTTPStatus(502),
			),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestRealWorldErrorResponses tests actual error responses from OpenAI API
func TestRealWorldErrorResponses(t *testing.T) {
	// Real error responses captured from OpenAI API
	realResponses := []struct {
		name         string
		response     string
		statusCode   int
		expectedCode core.ErrorCode
	}{
		{
			name: "Real rate limit response",
			response: `{
				"error": {
					"message": "You exceeded your current quota, please check your plan and billing details.",
					"type": "insufficient_quota",
					"param": null,
					"code": "insufficient_quota"
				}
			}`,
			statusCode:   429,
			expectedCode: core.ErrorRateLimited,
		},
		{
			name: "Real context length error",
			response: `{
				"error": {
					"message": "This model's maximum context length is 4097 tokens. However, your messages resulted in 5000 tokens.",
					"type": "invalid_request_error",
					"param": "messages",
					"code": "context_length_exceeded"
				}
			}`,
			statusCode:   400,
			expectedCode: core.ErrorContextLengthExceeded,
		},
		{
			name: "Real invalid API key",
			response: `{
				"error": {
					"message": "Incorrect API key provided: sk-proj-****. You can find your API key at https://platform.openai.com/api-keys.",
					"type": "invalid_request_error",
					"param": null,
					"code": "invalid_api_key"
				}
			}`,
			statusCode:   401,
			expectedCode: core.ErrorUnauthorized,
		},
	}

	for _, tt := range realResponses {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(bytes.NewBufferString(tt.response)),
				Header:     make(http.Header),
			}

			err := MapError(resp)
			aiErr, ok := err.(*core.AIError)
			if !ok {
				t.Fatalf("Expected *core.AIError, got %T", err)
			}

			if aiErr.Code != tt.expectedCode {
				t.Errorf("Expected error code %s, got %s", tt.expectedCode, aiErr.Code)
			}

			// Verify the raw error is preserved
			if aiErr.Raw == nil {
				t.Error("Expected raw error to be preserved")
			}
		})
	}
}

// BenchmarkMapError measures the performance of error mapping
func BenchmarkMapError(b *testing.B) {
	responseBody := `{
		"error": {
			"message": "Rate limit reached",
			"type": "rate_limit_error",
			"code": "rate_limit_exceeded"
		}
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &http.Response{
			StatusCode: 429,
			Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
			Header:     make(http.Header),
		}
		_ = MapError(resp)
	}
}