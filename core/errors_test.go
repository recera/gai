package core

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestNewError(t *testing.T) {
	err := NewError(ErrorRateLimited, "Rate limit exceeded", WithProvider("openai"))
	
	if err.Code != ErrorRateLimited {
		t.Errorf("Code = %v, want %v", err.Code, ErrorRateLimited)
	}
	
	if err.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", err.Provider, "openai")
	}
	
	if err.Message != "Rate limit exceeded" {
		t.Errorf("Message = %q, want %q", err.Message, "Rate limit exceeded")
	}
	
	if !err.Temporary {
		t.Error("Rate limit errors should be temporary")
	}
}

func TestErrorOptions(t *testing.T) {
	retryDuration := 30 * time.Second
	wrappedErr := fmt.Errorf("underlying error")
	
	err := NewError(ErrorInvalidRequest, "Invalid request",
		WithProvider("anthropic"),
		WithModel("claude-3"),
		WithHTTPStatus(http.StatusBadRequest),
		WithRetryAfter(retryDuration),
		WithRaw(map[string]any{"code": "invalid_json"}),
		WithWrapped(wrappedErr),
		WithTemporary(false),
	)
	
	if err.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", err.Provider, "anthropic")
	}
	
	if err.Model != "claude-3" {
		t.Errorf("Model = %q, want %q", err.Model, "claude-3")
	}
	
	if err.HTTPStatus != http.StatusBadRequest {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusBadRequest)
	}
	
	if err.RetryAfter == nil || *err.RetryAfter != retryDuration {
		t.Errorf("RetryAfter = %v, want %v", err.RetryAfter, retryDuration)
	}
	
	if err.Raw == nil {
		t.Error("Raw should not be nil")
	}
	
	if err.wrapped != wrappedErr {
		t.Errorf("wrapped = %v, want %v", err.wrapped, wrappedErr)
	}
	
	if err.Temporary {
		t.Error("Should not be temporary when explicitly set to false")
	}
}

func TestErrorString(t *testing.T) {
	tests := []struct {
		name     string
		err      *AIError
		expected string
	}{
		{
			name: "Basic error",
			err: &AIError{
				Code:    ErrorRateLimited,
				Message: "Rate limit exceeded",
			},
			expected: "rate_limited: Rate limit exceeded",
		},
		{
			name: "Error without message",
			err: &AIError{
				Code: ErrorUnauthorized,
			},
			expected: "unauthorized",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrorUnwrap(t *testing.T) {
	underlying := fmt.Errorf("underlying error")
	err := NewError(ErrorInternal, "Internal error", WithWrapped(underlying))
	
	unwrapped := err.Unwrap()
	if unwrapped != underlying {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlying)
	}
}

func TestErrorIs(t *testing.T) {
	err1 := NewError(ErrorRateLimited, "Rate limited")
	err2 := NewError(ErrorRateLimited, "Different message")
	err3 := NewError(ErrorUnauthorized, "Unauthorized")
	
	if !err1.Is(err2) {
		t.Error("Errors with same code should match with Is()")
	}
	
	if err1.Is(err3) {
		t.Error("Errors with different codes should not match with Is()")
	}
	
	if err1.Is(fmt.Errorf("random error")) {
		t.Error("Should not match non-AIError")
	}
}

func TestRetryAfterSeconds(t *testing.T) {
	tests := []struct {
		name     string
		err      *AIError
		expected int
	}{
		{
			name:     "No retry after",
			err:      &AIError{Code: ErrorInternal},
			expected: 0,
		},
		{
			name: "With retry after",
			err: &AIError{
				Code:       ErrorRateLimited,
				RetryAfter: durationPtr(30 * time.Second),
			},
			expected: 30,
		},
		{
			name: "With fractional seconds",
			err: &AIError{
				Code:       ErrorRateLimited,
				RetryAfter: durationPtr(45500 * time.Millisecond),
			},
			expected: 45,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.RetryAfterSeconds(); got != tt.expected {
				t.Errorf("RetryAfterSeconds() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestIsTransient(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{NewError(ErrorRateLimited, "Rate limited"), true},
		{NewError(ErrorOverloaded, "Overloaded"), true},
		{NewError(ErrorTimeout, "Timeout"), true},
		{NewError(ErrorNetwork, "Network error"), true},
		{NewError(ErrorProviderUnavailable, "Provider down"), true},
		{NewError(ErrorInternal, "Internal error"), true},
		{NewError(ErrorInvalidRequest, "Bad request"), false},
		{NewError(ErrorUnauthorized, "Unauthorized"), false},
		{fmt.Errorf("random error"), false},
		{nil, false},
	}
	
	for _, tt := range tests {
		name := "nil"
		if tt.err != nil {
			name = tt.err.Error()
		}
		t.Run(name, func(t *testing.T) {
			if got := IsTransient(tt.err); got != tt.expected {
				t.Errorf("IsTransient(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestIsRateLimited(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{NewError(ErrorRateLimited, "Rate limited"), true},
		{NewError(ErrorOverloaded, "Overloaded"), false},
		{NewError(ErrorUnauthorized, "Unauthorized"), false},
		{fmt.Errorf("random error"), false},
		{nil, false},
	}
	
	for _, tt := range tests {
		name := "nil"
		if tt.err != nil {
			name = tt.err.Error()
		}
		t.Run(name, func(t *testing.T) {
			if got := IsRateLimited(tt.err); got != tt.expected {
				t.Errorf("IsRateLimited(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestIsAuth(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{NewError(ErrorUnauthorized, "Unauthorized"), true},
		{NewError(ErrorForbidden, "Forbidden"), true},
		{NewError(ErrorRateLimited, "Rate limited"), false},
		{fmt.Errorf("random error"), false},
		{nil, false},
	}
	
	for _, tt := range tests {
		name := "nil"
		if tt.err != nil {
			name = tt.err.Error()
		}
		t.Run(name, func(t *testing.T) {
			if got := IsAuth(tt.err); got != tt.expected {
				t.Errorf("IsAuth(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestIsBadRequest(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{NewError(ErrorInvalidRequest, "Invalid"), true},
		{NewError(ErrorContextLengthExceeded, "Too long"), true},
		{NewError(ErrorUnsupported, "Unsupported"), true},
		{NewError(ErrorRateLimited, "Rate limited"), false},
		{fmt.Errorf("random error"), false},
		{nil, false},
	}
	
	for _, tt := range tests {
		name := "nil"
		if tt.err != nil {
			name = tt.err.Error()
		}
		t.Run(name, func(t *testing.T) {
			if got := IsBadRequest(tt.err); got != tt.expected {
				t.Errorf("IsBadRequest(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{NewError(ErrorNotFound, "Not found"), true},
		{NewError(ErrorInvalidRequest, "Invalid"), false},
		{fmt.Errorf("random error"), false},
		{nil, false},
	}
	
	for _, tt := range tests {
		name := "nil"
		if tt.err != nil {
			name = tt.err.Error()
		}
		t.Run(name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.expected {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestIsSafetyBlocked(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{NewError(ErrorSafetyBlocked, "Content filtered"), true},
		{NewError(ErrorInvalidRequest, "Invalid"), false},
		{fmt.Errorf("random error"), false},
		{nil, false},
	}
	
	for _, tt := range tests {
		name := "nil"
		if tt.err != nil {
			name = tt.err.Error()
		}
		t.Run(name, func(t *testing.T) {
			if got := IsSafetyBlocked(tt.err); got != tt.expected {
				t.Errorf("IsSafetyBlocked(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestIsTimeout(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{NewError(ErrorTimeout, "Timeout"), true},
		{NewError(ErrorNetwork, "Network"), false},
		{fmt.Errorf("random error"), false},
		{nil, false},
	}
	
	for _, tt := range tests {
		name := "nil"
		if tt.err != nil {
			name = tt.err.Error()
		}
		t.Run(name, func(t *testing.T) {
			if got := IsTimeout(tt.err); got != tt.expected {
				t.Errorf("IsTimeout(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestIsNetwork(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{NewError(ErrorNetwork, "Network error"), true},
		{NewError(ErrorProviderUnavailable, "Provider down"), true},
		{NewError(ErrorTimeout, "Timeout"), false},
		{fmt.Errorf("random error"), false},
		{nil, false},
	}
	
	for _, tt := range tests {
		name := "nil"
		if tt.err != nil {
			name = tt.err.Error()
		}
		t.Run(name, func(t *testing.T) {
			if got := IsNetwork(tt.err); got != tt.expected {
				t.Errorf("IsNetwork(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestIsOverloaded(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{NewError(ErrorOverloaded, "Overloaded"), true},
		{NewError(ErrorRateLimited, "Rate limited"), false},
		{fmt.Errorf("random error"), false},
		{nil, false},
	}
	
	for _, tt := range tests {
		name := "nil"
		if tt.err != nil {
			name = tt.err.Error()
		}
		t.Run(name, func(t *testing.T) {
			if got := IsOverloaded(tt.err); got != tt.expected {
				t.Errorf("IsOverloaded(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestGetRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected time.Duration
	}{
		{
			name: "Rate limited with retry after",
			err: NewError(ErrorRateLimited, "Rate limited",
				WithRetryAfter(45 * time.Second)),
			expected: 45 * time.Second,
		},
		{
			name:     "Rate limited without retry after",
			err:      NewError(ErrorRateLimited, "Rate limited"),
			expected: 60 * time.Second, // Default for rate limited
		},
		{
			name:     "Overloaded",
			err:      NewError(ErrorOverloaded, "Overloaded"),
			expected: 10 * time.Second,
		},
		{
			name:     "Provider unavailable",
			err:      NewError(ErrorProviderUnavailable, "Down"),
			expected: 30 * time.Second,
		},
		{
			name:     "Network error",
			err:      NewError(ErrorNetwork, "Network"),
			expected: 5 * time.Second,
		},
		{
			name:     "Timeout",
			err:      NewError(ErrorTimeout, "Timeout"),
			expected: 5 * time.Second,
		},
		{
			name:     "Non-retryable error",
			err:      NewError(ErrorInvalidRequest, "Bad request"),
			expected: 0,
		},
		{
			name:     "Non-AIError",
			err:      fmt.Errorf("random error"),
			expected: 0,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: 0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRetryAfter(tt.err); got != tt.expected {
				t.Errorf("GetRetryAfter() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFromHTTPStatus(t *testing.T) {
	tests := []struct {
		status   int
		message  string
		provider string
		expected ErrorCode
	}{
		{http.StatusBadRequest, "Bad request", "openai", ErrorInvalidRequest},
		{http.StatusUnauthorized, "Unauthorized", "anthropic", ErrorUnauthorized},
		{http.StatusForbidden, "Forbidden", "gemini", ErrorForbidden},
		{http.StatusNotFound, "Not found", "openai", ErrorNotFound},
		{http.StatusRequestEntityTooLarge, "Too large", "openai", ErrorContextLengthExceeded},
		{http.StatusTooManyRequests, "Rate limited", "openai", ErrorRateLimited},
		{http.StatusRequestTimeout, "Timeout", "openai", ErrorTimeout},
		{http.StatusInternalServerError, "Server error", "openai", ErrorInternal},
		{http.StatusBadGateway, "Bad gateway", "openai", ErrorInternal},
		{http.StatusServiceUnavailable, "Service unavailable", "openai", ErrorProviderUnavailable},
		{http.StatusGatewayTimeout, "Gateway timeout", "openai", ErrorTimeout},
		{418, "I'm a teapot", "openai", ErrorInvalidRequest}, // Unknown 4xx
		{599, "Unknown", "openai", ErrorInternal},            // Unknown 5xx
	}
	
	for _, tt := range tests {
		name := fmt.Sprintf("HTTP_%d", tt.status)
		t.Run(name, func(t *testing.T) {
			err := FromHTTPStatus(tt.status, tt.message, tt.provider)
			if err.Code != tt.expected {
				t.Errorf("FromHTTPStatus(%d) Code = %v, want %v", tt.status, err.Code, tt.expected)
			}
			if err.Message != tt.message {
				t.Errorf("Message = %q, want %q", err.Message, tt.message)
			}
			if err.Provider != tt.provider {
				t.Errorf("Provider = %q, want %q", err.Provider, tt.provider)
			}
			if err.HTTPStatus != tt.status {
				t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, tt.status)
			}
			// Check retry after for rate limiting
			if tt.status == http.StatusTooManyRequests && err.RetryAfter == nil {
				t.Error("Rate limited errors should have RetryAfter set")
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		code         ErrorCode
		provider     string
		expectedCode ErrorCode
		expectedMsg  string
	}{
		{
			name:         "Wrap standard error",
			err:          fmt.Errorf("connection refused"),
			code:         ErrorNetwork,
			provider:     "openai",
			expectedCode: ErrorNetwork,
			expectedMsg:  "connection refused",
		},
		{
			name:         "Wrap nil error",
			err:          nil,
			code:         ErrorInternal,
			provider:     "openai",
			expectedCode: ErrorCode(""),
			expectedMsg:  "",
		},
		{
			name: "Wrap existing AIError",
			err: NewError(ErrorRateLimited, "Rate limited",
				WithProvider("anthropic")),
			code:         ErrorInternal,
			provider:     "openai",
			expectedCode: ErrorRateLimited,
			expectedMsg:  "Rate limited",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapError(tt.err, tt.code, tt.provider)
			if wrapped == nil && tt.err == nil {
				// Expected behavior for nil input
				return
			}
			if wrapped == nil && tt.err != nil {
				t.Fatal("WrapError returned nil for non-nil error")
			}
			if wrapped.Code != tt.expectedCode {
				t.Errorf("Code = %v, want %v", wrapped.Code, tt.expectedCode)
			}
			if tt.expectedMsg != "" && wrapped.Message != tt.expectedMsg {
				t.Errorf("Message = %q, want %q", wrapped.Message, tt.expectedMsg)
			}
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	// Test that sentinel errors are properly initialized
	sentinels := []struct {
		err  *AIError
		code ErrorCode
	}{
		{ErrInvalidRequest, ErrorInvalidRequest},
		{ErrUnauthorized, ErrorUnauthorized},
		{ErrForbidden, ErrorForbidden},
		{ErrNotFound, ErrorNotFound},
		{ErrContextLengthExceeded, ErrorContextLengthExceeded},
		{ErrRateLimited, ErrorRateLimited},
		{ErrTimeout, ErrorTimeout},
		{ErrNetwork, ErrorNetwork},
		{ErrInternal, ErrorInternal},
	}
	
	for _, s := range sentinels {
		t.Run(string(s.code), func(t *testing.T) {
			if s.err.Code != s.code {
				t.Errorf("Sentinel error has wrong code: %v, want %v", s.err.Code, s.code)
			}
		})
	}
}

// Legacy compatibility tests
func TestLegacyHelpers(t *testing.T) {
	t.Run("IsContentFiltered", func(t *testing.T) {
		if !IsContentFiltered(NewError(ErrorSafetyBlocked, "Blocked")) {
			t.Error("IsContentFiltered should return true for safety blocked errors")
		}
		if IsContentFiltered(NewError(ErrorInvalidRequest, "Invalid")) {
			t.Error("IsContentFiltered should return false for non-safety errors")
		}
	})
	
	t.Run("IsContextSizeExceeded", func(t *testing.T) {
		if !IsContextSizeExceeded(NewError(ErrorContextLengthExceeded, "Too long")) {
			t.Error("IsContextSizeExceeded should return true for context length errors")
		}
		if IsContextSizeExceeded(NewError(ErrorInvalidRequest, "Invalid")) {
			t.Error("IsContextSizeExceeded should return false for other errors")
		}
	})
	
	t.Run("IsQuotaExceeded", func(t *testing.T) {
		if !IsQuotaExceeded(NewError(ErrorRateLimited, "Rate limited")) {
			t.Error("IsQuotaExceeded should return true for rate limited errors")
		}
		if IsQuotaExceeded(NewError(ErrorInvalidRequest, "Invalid")) {
			t.Error("IsQuotaExceeded should return false for other errors")
		}
	})
	
	t.Run("IsUnsupported", func(t *testing.T) {
		if !IsUnsupported(NewError(ErrorUnsupported, "Unsupported")) {
			t.Error("IsUnsupported should return true for unsupported errors")
		}
		if IsUnsupported(NewError(ErrorInvalidRequest, "Invalid")) {
			t.Error("IsUnsupported should return false for other errors")
		}
	})
}

// Helper function for tests
func durationPtr(d time.Duration) *time.Duration {
	return &d
}