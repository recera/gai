package core

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestErrorCategoryString(t *testing.T) {
	tests := []struct {
		category ErrorCategory
		expected string
	}{
		{ErrorCategoryTransient, "transient"},
		{ErrorCategoryRateLimit, "rate_limit"},
		{ErrorCategoryContentFilter, "content_filtered"},
		{ErrorCategoryBadRequest, "bad_request"},
		{ErrorCategoryAuth, "auth"},
		{ErrorCategoryNotFound, "not_found"},
		{ErrorCategoryTimeout, "timeout"},
		{ErrorCategoryContextSize, "context_size"},
		{ErrorCategoryQuota, "quota"},
		{ErrorCategoryUnsupported, "unsupported"},
		{ErrorCategoryUnknown, "unknown"},
		{ErrorCategory(999), "unknown"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.category.String(); got != tt.expected {
				t.Errorf("ErrorCategory.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewAIError(t *testing.T) {
	err := NewAIError(ErrorCategoryRateLimit, "openai", "Rate limit exceeded")
	
	if err.Category != ErrorCategoryRateLimit {
		t.Errorf("Category = %v, want %v", err.Category, ErrorCategoryRateLimit)
	}
	
	if err.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", err.Provider, "openai")
	}
	
	if err.Message != "Rate limit exceeded" {
		t.Errorf("Message = %q, want %q", err.Message, "Rate limit exceeded")
	}
	
	if !err.Retryable {
		t.Error("Rate limit errors should be retryable")
	}
}

func TestAIErrorChaining(t *testing.T) {
	err := NewAIError(ErrorCategoryBadRequest, "anthropic", "Invalid request").
		WithCode("invalid_json").
		WithHTTPStatus(http.StatusBadRequest).
		WithCause(fmt.Errorf("underlying error"))
	
	if err.Code != "invalid_json" {
		t.Errorf("Code = %q, want %q", err.Code, "invalid_json")
	}
	
	if err.HTTPStatus != http.StatusBadRequest {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusBadRequest)
	}
	
	if err.Cause == nil {
		t.Error("Cause should not be nil")
	}
	
	if err.Retryable {
		t.Error("Bad request errors should not be retryable")
	}
}

func TestAIErrorWithRetryAfter(t *testing.T) {
	retryAfter := 30
	err := NewAIError(ErrorCategoryRateLimit, "openai", "Rate limited").
		WithRetryAfter(retryAfter)
	
	if err.RetryAfter == nil {
		t.Fatal("RetryAfter should not be nil")
	}
	
	if *err.RetryAfter != retryAfter {
		t.Errorf("RetryAfter = %d, want %d", *err.RetryAfter, retryAfter)
	}
}

func TestAIErrorString(t *testing.T) {
	tests := []struct {
		name     string
		err      *AIError
		contains []string
	}{
		{
			name: "Basic error",
			err: &AIError{
				Category: ErrorCategoryRateLimit,
				Provider: "openai",
				Message:  "Rate limit exceeded",
			},
			contains: []string{"[openai]", "rate_limit", "Rate limit exceeded"},
		},
		{
			name: "Error with code",
			err: &AIError{
				Category: ErrorCategoryBadRequest,
				Provider: "anthropic",
				Code:     "invalid_json",
				Message:  "Invalid JSON",
			},
			contains: []string{"[anthropic]", "bad_request", "(invalid_json)", "Invalid JSON"},
		},
		{
			name: "Error with HTTP status",
			err: &AIError{
				Category:   ErrorCategoryAuth,
				Provider:   "gemini",
				Message:    "Unauthorized",
				HTTPStatus: http.StatusUnauthorized,
			},
			contains: []string{"[gemini]", "auth", "Unauthorized", "(HTTP 401)"},
		},
		{
			name: "Error with retry after",
			err: &AIError{
				Category:   ErrorCategoryRateLimit,
				Provider:   "openai",
				Message:    "Too many requests",
				RetryAfter: intPtr(60),
			},
			contains: []string{"rate_limit", "Too many requests", "(retry after 60s)"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, substr := range tt.contains {
				if !strings.Contains(errStr, substr) {
					t.Errorf("Error string %q does not contain %q", errStr, substr)
				}
			}
		})
	}
}

func TestErrorClassificationFunctions(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		checkFn  func(error) bool
		expected bool
	}{
		{
			name:     "IsTransient with transient error",
			err:      NewAIError(ErrorCategoryTransient, "provider", "Temporary failure"),
			checkFn:  IsTransient,
			expected: true,
		},
		{
			name:     "IsRateLimited with rate limit error",
			err:      NewAIError(ErrorCategoryRateLimit, "provider", "Rate limited"),
			checkFn:  IsRateLimited,
			expected: true,
		},
		{
			name:     "IsContentFiltered with content filter error",
			err:      NewAIError(ErrorCategoryContentFilter, "provider", "Content blocked"),
			checkFn:  IsContentFiltered,
			expected: true,
		},
		{
			name:     "IsBadRequest with bad request error",
			err:      NewAIError(ErrorCategoryBadRequest, "provider", "Invalid input"),
			checkFn:  IsBadRequest,
			expected: true,
		},
		{
			name:     "IsAuth with auth error",
			err:      NewAIError(ErrorCategoryAuth, "provider", "Unauthorized"),
			checkFn:  IsAuth,
			expected: true,
		},
		{
			name:     "IsNotFound with not found error",
			err:      NewAIError(ErrorCategoryNotFound, "provider", "Model not found"),
			checkFn:  IsNotFound,
			expected: true,
		},
		{
			name:     "IsTimeout with timeout error",
			err:      NewAIError(ErrorCategoryTimeout, "provider", "Request timeout"),
			checkFn:  IsTimeout,
			expected: true,
		},
		{
			name:     "IsContextSize with context size error",
			err:      NewAIError(ErrorCategoryContextSize, "provider", "Context too large"),
			checkFn:  IsContextSize,
			expected: true,
		},
		{
			name:     "IsQuota with quota error",
			err:      NewAIError(ErrorCategoryQuota, "provider", "Quota exceeded"),
			checkFn:  IsQuota,
			expected: true,
		},
		{
			name:     "IsUnsupported with unsupported error",
			err:      NewAIError(ErrorCategoryUnsupported, "provider", "Feature not supported"),
			checkFn:  IsUnsupported,
			expected: true,
		},
		{
			name:     "Non-AI error returns false",
			err:      errors.New("generic error"),
			checkFn:  IsRateLimited,
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.checkFn(tt.err); got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, got, tt.expected)
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
			name:     "Transient error is retryable",
			err:      NewAIError(ErrorCategoryTransient, "provider", "Temporary failure"),
			expected: true,
		},
		{
			name:     "Rate limit error is retryable",
			err:      NewAIError(ErrorCategoryRateLimit, "provider", "Rate limited"),
			expected: true,
		},
		{
			name:     "Bad request is not retryable",
			err:      NewAIError(ErrorCategoryBadRequest, "provider", "Invalid input"),
			expected: false,
		},
		{
			name:     "Auth error is not retryable",
			err:      NewAIError(ErrorCategoryAuth, "provider", "Unauthorized"),
			expected: false,
		},
		{
			name:     "5xx status makes error retryable",
			err:      NewAIError(ErrorCategoryUnknown, "provider", "Server error").WithHTTPStatus(500),
			expected: true,
		},
		{
			name:     "429 status makes error retryable",
			err:      NewAIError(ErrorCategoryRateLimit, "provider", "Too many requests").WithHTTPStatus(429),
			expected: true,
		},
		{
			name:     "Non-AI error is not retryable",
			err:      errors.New("generic error"),
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetRetryAfter(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedSec  int
		expectedOk   bool
	}{
		{
			name:        "Error with retry after",
			err:         NewAIError(ErrorCategoryRateLimit, "provider", "Rate limited").WithRetryAfter(30),
			expectedSec: 30,
			expectedOk:  true,
		},
		{
			name:        "Error without retry after",
			err:         NewAIError(ErrorCategoryRateLimit, "provider", "Rate limited"),
			expectedSec: 0,
			expectedOk:  false,
		},
		{
			name:        "Non-AI error",
			err:         errors.New("generic error"),
			expectedSec: 0,
			expectedOk:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec, ok := GetRetryAfter(tt.err)
			if sec != tt.expectedSec {
				t.Errorf("GetRetryAfter() seconds = %d, want %d", sec, tt.expectedSec)
			}
			if ok != tt.expectedOk {
				t.Errorf("GetRetryAfter() ok = %v, want %v", ok, tt.expectedOk)
			}
		})
	}
}

func TestWrapProviderError(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		err          error
		httpStatus   int
		wantCategory ErrorCategory
		wantRetryable bool
	}{
		{
			name:         "429 becomes rate limit",
			provider:     "openai",
			err:          errors.New("too many requests"),
			httpStatus:   http.StatusTooManyRequests,
			wantCategory: ErrorCategoryRateLimit,
			wantRetryable: true,
		},
		{
			name:         "401 becomes auth",
			provider:     "anthropic",
			err:          errors.New("unauthorized"),
			httpStatus:   http.StatusUnauthorized,
			wantCategory: ErrorCategoryAuth,
			wantRetryable: false,
		},
		{
			name:         "400 becomes bad request",
			provider:     "gemini",
			err:          errors.New("invalid input"),
			httpStatus:   http.StatusBadRequest,
			wantCategory: ErrorCategoryBadRequest,
			wantRetryable: false,
		},
		{
			name:         "404 becomes not found",
			provider:     "openai",
			err:          errors.New("model not found"),
			httpStatus:   http.StatusNotFound,
			wantCategory: ErrorCategoryNotFound,
			wantRetryable: false,
		},
		{
			name:         "408 becomes timeout",
			provider:     "anthropic",
			err:          errors.New("request timeout"),
			httpStatus:   http.StatusRequestTimeout,
			wantCategory: ErrorCategoryTimeout,
			wantRetryable: true,
		},
		{
			name:         "413 becomes context size",
			provider:     "gemini",
			err:          errors.New("payload too large"),
			httpStatus:   http.StatusRequestEntityTooLarge,
			wantCategory: ErrorCategoryContextSize,
			wantRetryable: false,
		},
		{
			name:         "402 becomes quota",
			provider:     "openai",
			err:          errors.New("payment required"),
			httpStatus:   http.StatusPaymentRequired,
			wantCategory: ErrorCategoryQuota,
			wantRetryable: false,
		},
		{
			name:         "501 becomes unsupported",
			provider:     "anthropic",
			err:          errors.New("not implemented"),
			httpStatus:   http.StatusNotImplemented,
			wantCategory: ErrorCategoryUnsupported,
			wantRetryable: false,
		},
		{
			name:         "500 becomes transient",
			provider:     "gemini",
			err:          errors.New("internal server error"),
			httpStatus:   http.StatusInternalServerError,
			wantCategory: ErrorCategoryTransient,
			wantRetryable: true,
		},
		{
			name:         "503 becomes transient",
			provider:     "openai",
			err:          errors.New("service unavailable"),
			httpStatus:   http.StatusServiceUnavailable,
			wantCategory: ErrorCategoryTransient,
			wantRetryable: true,
		},
		{
			name:         "Already AIError is preserved",
			provider:     "new-provider",
			err:          NewAIError(ErrorCategoryRateLimit, "old-provider", "Rate limited"),
			httpStatus:   0,
			wantCategory: ErrorCategoryRateLimit,
			wantRetryable: true,
		},
		{
			name:         "Nil error returns nil",
			provider:     "provider",
			err:          nil,
			httpStatus:   200,
			wantCategory: ErrorCategoryUnknown,
			wantRetryable: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapProviderError(tt.provider, tt.err, tt.httpStatus)
			
			if tt.err == nil {
				if wrapped != nil {
					t.Errorf("WrapProviderError(nil) = %v, want nil", wrapped)
				}
				return
			}
			
			var aiErr *AIError
			if !errors.As(wrapped, &aiErr) {
				t.Fatalf("WrapProviderError did not return *AIError")
			}
			
			if aiErr.Category != tt.wantCategory {
				t.Errorf("Category = %v, want %v", aiErr.Category, tt.wantCategory)
			}
			
			if aiErr.Retryable != tt.wantRetryable {
				t.Errorf("Retryable = %v, want %v", aiErr.Retryable, tt.wantRetryable)
			}
			
			if aiErr.HTTPStatus != tt.httpStatus && tt.httpStatus != 0 {
				t.Errorf("HTTPStatus = %d, want %d", aiErr.HTTPStatus, tt.httpStatus)
			}
		})
	}
}

func TestAIErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewAIError(ErrorCategoryBadRequest, "provider", "Bad request").WithCause(cause)
	
	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
	
	// Test with errors.Is
	if !errors.Is(err, cause) {
		t.Error("errors.Is(err, cause) = false, want true")
	}
}

// Helper function for tests
func intPtr(i int) *int {
	return &i
}