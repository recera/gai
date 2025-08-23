// Package core provides error types and classification for the AI framework.
// This file defines the error taxonomy used across all providers.

package core

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ErrorCategory represents the category of an error.
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

// AIError represents an error from an AI provider.
type AIError struct {
	// Category classifies the error type
	Category ErrorCategory
	// Provider that generated the error
	Provider string
	// Code is the provider-specific error code
	Code string
	// Message is the human-readable error message
	Message string
	// HTTPStatus is the HTTP status code if applicable
	HTTPStatus int
	// Retryable indicates if the operation can be retried
	Retryable bool
	// RetryAfter suggests when to retry (for rate limits)
	RetryAfter *int
	// Cause is the underlying error if any
	Cause error
}

// Error implements the error interface.
func (e *AIError) Error() string {
	var parts []string
	
	if e.Provider != "" {
		parts = append(parts, fmt.Sprintf("[%s]", e.Provider))
	}
	
	parts = append(parts, e.Category.String())
	
	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("(%s)", e.Code))
	}
	
	parts = append(parts, e.Message)
	
	if e.HTTPStatus != 0 {
		parts = append(parts, fmt.Sprintf("(HTTP %d)", e.HTTPStatus))
	}
	
	if e.RetryAfter != nil {
		parts = append(parts, fmt.Sprintf("(retry after %ds)", *e.RetryAfter))
	}
	
	return strings.Join(parts, " ")
}

// Unwrap returns the underlying error.
func (e *AIError) Unwrap() error {
	return e.Cause
}

// String returns the string representation of an ErrorCategory.
func (c ErrorCategory) String() string {
	switch c {
	case ErrorCategoryTransient:
		return "transient"
	case ErrorCategoryRateLimit:
		return "rate_limit"
	case ErrorCategoryContentFilter:
		return "content_filtered"
	case ErrorCategoryBadRequest:
		return "bad_request"
	case ErrorCategoryAuth:
		return "auth"
	case ErrorCategoryNotFound:
		return "not_found"
	case ErrorCategoryTimeout:
		return "timeout"
	case ErrorCategoryContextSize:
		return "context_size"
	case ErrorCategoryQuota:
		return "quota"
	case ErrorCategoryUnsupported:
		return "unsupported"
	default:
		return "unknown"
	}
}

// NewAIError creates a new AIError with the given parameters.
func NewAIError(category ErrorCategory, provider, message string) *AIError {
	return &AIError{
		Category:  category,
		Provider:  provider,
		Message:   message,
		Retryable: category == ErrorCategoryTransient || category == ErrorCategoryRateLimit,
	}
}

// WithCode adds an error code to the error.
func (e *AIError) WithCode(code string) *AIError {
	e.Code = code
	return e
}

// WithHTTPStatus adds an HTTP status code to the error.
func (e *AIError) WithHTTPStatus(status int) *AIError {
	e.HTTPStatus = status
	// Update retryable based on status
	if status >= 500 || status == http.StatusTooManyRequests {
		e.Retryable = true
	} else if status >= 400 && status < 500 {
		e.Retryable = false
	}
	return e
}

// WithRetryAfter sets the retry-after duration in seconds.
func (e *AIError) WithRetryAfter(seconds int) *AIError {
	e.RetryAfter = &seconds
	return e
}

// WithCause wraps an underlying error.
func (e *AIError) WithCause(err error) *AIError {
	e.Cause = err
	return e
}

// IsTransient returns true if the error is transient and can be retried.
func IsTransient(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Category == ErrorCategoryTransient || aiErr.Retryable
	}
	return false
}

// IsRateLimited returns true if the error is due to rate limiting.
func IsRateLimited(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Category == ErrorCategoryRateLimit
	}
	return false
}

// IsContentFiltered returns true if the error is due to content filtering.
func IsContentFiltered(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Category == ErrorCategoryContentFilter
	}
	return false
}

// IsBadRequest returns true if the error is due to a bad request.
func IsBadRequest(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Category == ErrorCategoryBadRequest
	}
	return false
}

// IsAuth returns true if the error is due to authentication/authorization failure.
func IsAuth(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Category == ErrorCategoryAuth
	}
	return false
}

// IsNotFound returns true if the error is due to a resource not being found.
func IsNotFound(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Category == ErrorCategoryNotFound
	}
	return false
}

// IsTimeout returns true if the error is due to a timeout.
func IsTimeout(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Category == ErrorCategoryTimeout
	}
	return false
}

// IsContextSize returns true if the error is due to context size limits.
func IsContextSize(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Category == ErrorCategoryContextSize
	}
	return false
}

// IsQuota returns true if the error is due to quota exhaustion.
func IsQuota(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Category == ErrorCategoryQuota
	}
	return false
}

// IsUnsupported returns true if the error is due to an unsupported operation.
func IsUnsupported(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Category == ErrorCategoryUnsupported
	}
	return false
}

// GetRetryAfter returns the retry-after duration in seconds if available.
func GetRetryAfter(err error) (int, bool) {
	var aiErr *AIError
	if errors.As(err, &aiErr) && aiErr.RetryAfter != nil {
		return *aiErr.RetryAfter, true
	}
	return 0, false
}

// IsRetryable returns true if the error indicates the operation can be retried.
func IsRetryable(err error) bool {
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr.Retryable
	}
	// Default to not retryable for unknown errors
	return false
}

// WrapProviderError wraps a provider-specific error with classification.
// This is used by provider adapters to normalize errors.
func WrapProviderError(provider string, err error, httpStatus int) error {
	if err == nil {
		return nil
	}
	
	// If it's already an AIError, preserve it
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		if aiErr.Provider == "" {
			aiErr.Provider = provider
		}
		return aiErr
	}
	
	// Classify based on HTTP status
	category := ErrorCategoryUnknown
	retryable := false
	
	switch httpStatus {
	case http.StatusTooManyRequests:
		category = ErrorCategoryRateLimit
		retryable = true
	case http.StatusUnauthorized, http.StatusForbidden:
		category = ErrorCategoryAuth
	case http.StatusBadRequest:
		category = ErrorCategoryBadRequest
	case http.StatusNotFound:
		category = ErrorCategoryNotFound
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		category = ErrorCategoryTimeout
		retryable = true
	case http.StatusRequestEntityTooLarge:
		category = ErrorCategoryContextSize
	case http.StatusPaymentRequired:
		category = ErrorCategoryQuota
	case http.StatusNotImplemented:
		category = ErrorCategoryUnsupported
	default:
		if httpStatus >= 500 {
			category = ErrorCategoryTransient
			retryable = true
		} else if httpStatus >= 400 {
			category = ErrorCategoryBadRequest
		}
	}
	
	return &AIError{
		Category:   category,
		Provider:   provider,
		Message:    err.Error(),
		HTTPStatus: httpStatus,
		Retryable:  retryable,
		Cause:      err,
	}
}