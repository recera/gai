// Package core provides fundamental types and interfaces for the GAI framework.
// This file defines the stable error taxonomy for consistent error handling across providers.
package core

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ErrorCode represents a stable error category for consistent handling.
type ErrorCode string

const (
	// Client errors (4xx category)
	ErrorInvalidRequest        ErrorCode = "invalid_request"         // Bad schema/params
	ErrorUnauthorized          ErrorCode = "unauthorized"            // Missing/invalid auth
	ErrorForbidden             ErrorCode = "forbidden"               // No permission
	ErrorNotFound              ErrorCode = "not_found"               // Resource not found
	ErrorContextLengthExceeded ErrorCode = "context_length_exceeded" // Input too long
	ErrorUnsupported           ErrorCode = "unsupported"             // Feature not available

	// Rate limiting and capacity
	ErrorRateLimited ErrorCode = "rate_limited" // 429 Too Many Requests
	ErrorOverloaded  ErrorCode = "overloaded"   // Provider busy/at capacity

	// Safety and content filtering
	ErrorSafetyBlocked ErrorCode = "safety_blocked" // Content blocked by safety filters

	// Network and infrastructure
	ErrorTimeout             ErrorCode = "timeout"               // Request timed out
	ErrorNetwork             ErrorCode = "network"               // Network/connection error
	ErrorProviderUnavailable ErrorCode = "provider_unavailable" // Provider down/incident

	// Server errors (5xx category)
	ErrorInternal ErrorCode = "internal" // Unexpected server error
)

// AIError represents a normalized error from any AI provider.
type AIError struct {
	// Code is the stable error category
	Code ErrorCode `json:"code"`
	// Message is a human-readable error description
	Message string `json:"message"`
	// Temporary indicates if the error is likely transient
	Temporary bool `json:"temporary"`
	// RetryAfter suggests when to retry (optional)
	RetryAfter *time.Duration `json:"retry_after_ms,omitempty"`
	// Provider identifies which provider returned the error
	Provider string `json:"provider,omitempty"`
	// Model identifies which model was being used (if known)
	Model string `json:"model,omitempty"`
	// HTTPStatus is the original HTTP status code (if applicable)
	HTTPStatus int `json:"http_status,omitempty"`
	// Raw contains the original provider error for debugging
	Raw any `json:"raw,omitempty"`
	// wrapped allows error chaining
	wrapped error
}

// Error implements the error interface.
func (e *AIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return string(e.Code)
}

// Unwrap returns the wrapped error for error chain inspection.
func (e *AIError) Unwrap() error {
	return e.wrapped
}

// Is implements errors.Is for error comparison.
func (e *AIError) Is(target error) bool {
	var targetErr *AIError
	if errors.As(target, &targetErr) {
		return e.Code == targetErr.Code
	}
	return false
}

// RetryAfterSeconds returns the retry delay in seconds (0 if not set).
func (e *AIError) RetryAfterSeconds() int {
	if e.RetryAfter == nil {
		return 0
	}
	return int(e.RetryAfter.Seconds())
}

// NewError creates a new AIError with the given code and message.
func NewError(code ErrorCode, message string, opts ...ErrorOption) *AIError {
	err := &AIError{
		Code:      code,
		Message:   message,
		Temporary: isTemporaryCode(code),
	}
	
	for _, opt := range opts {
		opt(err)
	}
	
	return err
}

// ErrorOption configures an AIError.
type ErrorOption func(*AIError)

// WithProvider sets the provider that returned the error.
func WithProvider(provider string) ErrorOption {
	return func(e *AIError) {
		e.Provider = provider
	}
}

// WithModel sets the model that was being used.
func WithModel(model string) ErrorOption {
	return func(e *AIError) {
		e.Model = model
	}
}

// WithRetryAfter sets the retry delay.
func WithRetryAfter(d time.Duration) ErrorOption {
	return func(e *AIError) {
		e.RetryAfter = &d
	}
}

// WithHTTPStatus sets the original HTTP status code.
func WithHTTPStatus(status int) ErrorOption {
	return func(e *AIError) {
		e.HTTPStatus = status
	}
}

// WithRaw attaches the original provider error.
func WithRaw(raw any) ErrorOption {
	return func(e *AIError) {
		e.Raw = raw
	}
}

// WithWrapped wraps another error for chaining.
func WithWrapped(err error) ErrorOption {
	return func(e *AIError) {
		e.wrapped = err
	}
}

// WithTemporary overrides the temporary flag.
func WithTemporary(temporary bool) ErrorOption {
	return func(e *AIError) {
		e.Temporary = temporary
	}
}

// Helper functions to check error categories

// IsTransient returns true if the error is likely temporary and retryable.
func IsTransient(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Temporary
	}
	return false
}

// IsRateLimited returns true if the error is due to rate limiting.
func IsRateLimited(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Code == ErrorRateLimited
	}
	return false
}

// IsAuth returns true if the error is authentication/authorization related.
func IsAuth(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Code == ErrorUnauthorized || aiErr.Code == ErrorForbidden
	}
	return false
}

// IsBadRequest returns true if the error is due to invalid input.
func IsBadRequest(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Code == ErrorInvalidRequest || 
		       aiErr.Code == ErrorContextLengthExceeded ||
		       aiErr.Code == ErrorUnsupported
	}
	return false
}

// IsNotFound returns true if the error is due to a missing resource.
func IsNotFound(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Code == ErrorNotFound
	}
	return false
}

// IsSafetyBlocked returns true if the error is due to safety filtering.
func IsSafetyBlocked(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Code == ErrorSafetyBlocked
	}
	return false
}

// IsTimeout returns true if the error is due to a timeout.
func IsTimeout(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Code == ErrorTimeout
	}
	return false
}

// IsNetwork returns true if the error is network-related.
func IsNetwork(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Code == ErrorNetwork || aiErr.Code == ErrorProviderUnavailable
	}
	return false
}

// IsOverloaded returns true if the provider is at capacity.
func IsOverloaded(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Code == ErrorOverloaded
	}
	return false
}

// GetRetryAfter returns the suggested retry delay, or a default based on the error type.
func GetRetryAfter(err error) time.Duration {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		if aiErr.RetryAfter != nil {
			return *aiErr.RetryAfter
		}
		// Default retry delays by error type
		switch aiErr.Code {
		case ErrorRateLimited:
			return 60 * time.Second
		case ErrorOverloaded:
			return 10 * time.Second
		case ErrorProviderUnavailable:
			return 30 * time.Second
		case ErrorNetwork, ErrorTimeout:
			return 5 * time.Second
		}
	}
	return 0
}

// FromHTTPStatus creates an AIError from an HTTP status code.
func FromHTTPStatus(status int, message string, provider string) *AIError {
	code := httpStatusToErrorCode(status)
	opts := []ErrorOption{
		WithProvider(provider),
		WithHTTPStatus(status),
	}
	
	// Add retry hints for rate limiting
	if status == http.StatusTooManyRequests {
		opts = append(opts, WithRetryAfter(60*time.Second))
	}
	
	return NewError(code, message, opts...)
}

// httpStatusToErrorCode maps HTTP status codes to error codes.
func httpStatusToErrorCode(status int) ErrorCode {
	switch status {
	case http.StatusBadRequest:
		return ErrorInvalidRequest
	case http.StatusUnauthorized:
		return ErrorUnauthorized
	case http.StatusForbidden:
		return ErrorForbidden
	case http.StatusNotFound:
		return ErrorNotFound
	case http.StatusRequestEntityTooLarge:
		return ErrorContextLengthExceeded
	case http.StatusTooManyRequests:
		return ErrorRateLimited
	case http.StatusRequestTimeout:
		return ErrorTimeout
	case http.StatusInternalServerError, http.StatusBadGateway:
		return ErrorInternal
	case http.StatusServiceUnavailable:
		return ErrorProviderUnavailable
	case http.StatusGatewayTimeout:
		return ErrorTimeout
	default:
		if status >= 400 && status < 500 {
			return ErrorInvalidRequest
		}
		if status >= 500 {
			return ErrorInternal
		}
		return ErrorInternal
	}
}

// isTemporaryCode returns true if the error code represents a transient error.
func isTemporaryCode(code ErrorCode) bool {
	switch code {
	case ErrorRateLimited, ErrorOverloaded, ErrorTimeout, 
	     ErrorNetwork, ErrorProviderUnavailable, ErrorInternal:
		return true
	default:
		return false
	}
}

// WrapError wraps an existing error with AIError metadata.
func WrapError(err error, code ErrorCode, provider string) *AIError {
	if err == nil {
		return nil
	}
	
	// If already an AIError, preserve it
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr
	}
	
	return NewError(code, err.Error(),
		WithProvider(provider),
		WithWrapped(err),
	)
}

// Sentinel errors for common cases
var (
	ErrInvalidRequest        = NewError(ErrorInvalidRequest, "invalid request")
	ErrUnauthorized          = NewError(ErrorUnauthorized, "unauthorized")
	ErrForbidden             = NewError(ErrorForbidden, "forbidden")
	ErrNotFound              = NewError(ErrorNotFound, "not found")
	ErrContextLengthExceeded = NewError(ErrorContextLengthExceeded, "context length exceeded")
	ErrRateLimited           = NewError(ErrorRateLimited, "rate limited")
	ErrTimeout               = NewError(ErrorTimeout, "timeout")
	ErrNetwork               = NewError(ErrorNetwork, "network error")
	ErrInternal              = NewError(ErrorInternal, "internal error")
)

// Legacy compatibility - map old error categories to new error codes
// ErrorCategory represents the category of an error (deprecated).
type ErrorCategory int

const (
	// ErrorCategoryUnknown indicates an unclassified error
	ErrorCategoryUnknown ErrorCategory = iota
	// ErrorCategoryTransient indicates a temporary error that can be retried
	ErrorCategoryTransient
	// ErrorCategoryRateLimit indicates rate limiting
	ErrorCategoryRateLimit
	// ErrorCategoryContentFilter indicates content was filtered
	ErrorCategoryContentFilter
	// ErrorCategoryBadRequest indicates an invalid request
	ErrorCategoryBadRequest
	// ErrorCategoryAuth indicates authentication/authorization failure
	ErrorCategoryAuth
	// ErrorCategoryNotFound indicates a resource was not found
	ErrorCategoryNotFound
	// ErrorCategoryTimeout indicates a timeout occurred
	ErrorCategoryTimeout
	// ErrorCategoryContextSize indicates the context window was exceeded
	ErrorCategoryContextSize
	// ErrorCategoryQuota indicates a quota was exceeded
	ErrorCategoryQuota
	// ErrorCategoryUnsupported indicates an unsupported operation
	ErrorCategoryUnsupported
)

// IsContentFiltered returns true if the error is due to content filtering (legacy).
func IsContentFiltered(err error) bool {
	return IsSafetyBlocked(err)
}

// IsContextSizeExceeded returns true if the error is due to context size (legacy).
func IsContextSizeExceeded(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Code == ErrorContextLengthExceeded
	}
	return false
}

// IsQuotaExceeded returns true if the error is due to quota limits (legacy).
func IsQuotaExceeded(err error) bool {
	return IsRateLimited(err)
}

// IsUnsupported returns true if the operation is not supported (legacy).
func IsUnsupported(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Code == ErrorUnsupported
	}
	return false
}