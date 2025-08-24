// Package openai implements error mapping for OpenAI API responses.
// This file maps OpenAI-specific error responses to the stable GAI error taxonomy.
package openai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/recera/gai/core"
)

// ErrorResponse represents the structure of OpenAI API error responses.
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`    // e.g., "invalid_request_error", "rate_limit_error"
		Code    string `json:"code"`    // e.g., "context_length_exceeded", "model_not_found"
		Param   string `json:"param"`   // Optional: which parameter caused the error
	} `json:"error"`
}

// MapError converts an OpenAI API error response to a stable core.AIError.
// It handles both HTTP status codes and OpenAI-specific error codes to provide
// the most accurate error categorization.
func MapError(resp *http.Response) error {
	if resp == nil {
		return core.NewError(core.ErrorInternal, "nil response")
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return core.NewError(
			core.ErrorInternal,
			fmt.Sprintf("failed to read error response: %v", err),
			core.WithHTTPStatus(resp.StatusCode),
			core.WithProvider("openai"),
		)
	}

	// Try to parse the OpenAI error structure
	var apiErr ErrorResponse
	if err := json.Unmarshal(body, &apiErr); err != nil {
		// If we can't parse the error, return a generic error based on status
		return mapStatusCodeOnly(resp.StatusCode, string(body))
	}

	// Map the error based on both type and code
	code := mapErrorTypeAndCode(apiErr.Error.Type, apiErr.Error.Code, resp.StatusCode)
	
	// Build the error with all available context
	opts := []core.ErrorOption{
		core.WithProvider("openai"),
		core.WithHTTPStatus(resp.StatusCode),
		core.WithRaw(apiErr),
	}

	// Add retry-after for rate limiting
	if code == core.ErrorRateLimited {
		// OpenAI includes retry-after in headers
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			// Parse as seconds (OpenAI uses seconds)
			if seconds := parseRetryAfter(retryAfter); seconds > 0 {
				opts = append(opts, core.WithRetryAfter(time.Duration(seconds)*time.Second))
			}
		} else {
			// Default retry after for rate limiting
			opts = append(opts, core.WithRetryAfter(60*time.Second))
		}
	}

	// Add model information if available from headers
	if model := resp.Header.Get("X-Model"); model != "" {
		opts = append(opts, core.WithModel(model))
	}

	return core.NewError(code, apiErr.Error.Message, opts...)
}

// mapErrorTypeAndCode maps OpenAI error types and codes to stable error codes.
func mapErrorTypeAndCode(errorType, errorCode string, statusCode int) core.ErrorCode {
	// First check specific error codes (most precise)
	switch errorCode {
	case "context_length_exceeded":
		return core.ErrorContextLengthExceeded
	case "model_not_found":
		return core.ErrorNotFound
	case "insufficient_quota":
		return core.ErrorRateLimited
	case "rate_limit_exceeded":
		return core.ErrorRateLimited
	case "content_filter":
		return core.ErrorSafetyBlocked
	case "content_policy_violation":
		return core.ErrorSafetyBlocked
	case "invalid_api_key":
		return core.ErrorUnauthorized
	}

	// Then check error types
	switch errorType {
	case "invalid_request_error":
		// Further differentiate based on status code
		if statusCode == http.StatusNotFound {
			return core.ErrorNotFound
		}
		if statusCode == http.StatusRequestEntityTooLarge {
			return core.ErrorContextLengthExceeded
		}
		return core.ErrorInvalidRequest
	
	case "authentication_error":
		return core.ErrorUnauthorized
	
	case "permission_error":
		return core.ErrorForbidden
	
	case "rate_limit_error":
		return core.ErrorRateLimited
	
	case "server_error":
		if statusCode == http.StatusServiceUnavailable {
			return core.ErrorProviderUnavailable
		}
		if statusCode == http.StatusGatewayTimeout {
			return core.ErrorTimeout
		}
		if statusCode == http.StatusBadGateway {
			return core.ErrorNetwork
		}
		return core.ErrorInternal
	
	case "engine_overloaded_error":
		return core.ErrorOverloaded
	}

	// Fall back to status code mapping
	return mapStatusCode(statusCode)
}

// mapStatusCodeOnly handles cases where we only have an HTTP status code.
func mapStatusCodeOnly(statusCode int, body string) error {
	code := mapStatusCode(statusCode)
	
	// Try to extract a meaningful message from the body
	message := body
	if len(message) > 200 {
		message = message[:200] + "..."
	}
	if message == "" {
		message = fmt.Sprintf("HTTP %d error", statusCode)
	}

	opts := []core.ErrorOption{
		core.WithProvider("openai"),
		core.WithHTTPStatus(statusCode),
	}

	// Add retry hints for transient errors
	if code == core.ErrorRateLimited {
		opts = append(opts, core.WithRetryAfter(60*time.Second))
	} else if code == core.ErrorOverloaded {
		opts = append(opts, core.WithRetryAfter(10*time.Second))
	}

	return core.NewError(code, message, opts...)
}

// mapStatusCode maps HTTP status codes to error codes.
func mapStatusCode(status int) core.ErrorCode {
	switch status {
	case http.StatusBadRequest:
		return core.ErrorInvalidRequest
	case http.StatusUnauthorized:
		return core.ErrorUnauthorized
	case http.StatusForbidden:
		return core.ErrorForbidden
	case http.StatusNotFound:
		return core.ErrorNotFound
	case http.StatusRequestEntityTooLarge:
		return core.ErrorContextLengthExceeded
	case http.StatusTooManyRequests:
		return core.ErrorRateLimited
	case http.StatusRequestTimeout:
		return core.ErrorTimeout
	case http.StatusInternalServerError:
		return core.ErrorInternal
	case http.StatusBadGateway:
		return core.ErrorNetwork
	case http.StatusServiceUnavailable:
		return core.ErrorProviderUnavailable
	case http.StatusGatewayTimeout:
		return core.ErrorTimeout
	default:
		if status >= 400 && status < 500 {
			return core.ErrorInvalidRequest
		}
		if status >= 500 {
			return core.ErrorInternal
		}
		return core.ErrorInternal
	}
}

// parseRetryAfter parses the Retry-After header value (in seconds).
func parseRetryAfter(value string) int {
	// OpenAI uses seconds in Retry-After
	var seconds int
	if _, err := fmt.Sscanf(value, "%d", &seconds); err == nil {
		return seconds
	}
	return 0
}

// IsRetryable checks if an error from OpenAI should be retried.
// This is a convenience function that checks both the error code and OpenAI-specific conditions.
func IsRetryable(err error) bool {
	// Use the core helper which checks the Temporary flag
	if core.IsTransient(err) {
		return true
	}

	// Check for specific OpenAI conditions that might not be marked as transient
	var aiErr *core.AIError
	if e, ok := err.(*core.AIError); ok {
		aiErr = e
		// Check if it's an overloaded error (always retry)
		if aiErr.Code == core.ErrorOverloaded {
			return true
		}
		// Check HTTP status for additional retry signals
		if aiErr.HTTPStatus == 502 || aiErr.HTTPStatus == 504 {
			return true
		}
	}

	return false
}

// ExtractModelFromError attempts to extract the model name from an error response.
// OpenAI sometimes includes the model in error messages.
func ExtractModelFromError(err error) string {
	var aiErr *core.AIError
	if e, ok := err.(*core.AIError); ok {
		aiErr = e
		if aiErr.Model != "" {
			return aiErr.Model
		}
		// Try to extract from message (e.g., "The model `gpt-4` does not exist")
		if strings.Contains(aiErr.Message, "model `") {
			start := strings.Index(aiErr.Message, "model `") + 7
			if end := strings.Index(aiErr.Message[start:], "`"); end > 0 {
				return aiErr.Message[start : start+end]
			}
		}
	}
	return ""
}