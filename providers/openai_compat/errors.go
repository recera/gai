package openai_compat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/recera/gai/core"
)

// MapError converts an OpenAI-compatible API error response to a stable core.AIError.
// It handles both HTTP status codes and provider-specific error codes.
func MapError(resp *http.Response, providerName string) error {
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
			core.WithProvider(providerName),
		)
	}
	
	// Try to parse the OpenAI-compatible error structure
	var apiErr errorResponse
	if err := json.Unmarshal(body, &apiErr); err != nil {
		// If we can't parse the error, return a generic error based on status
		return mapStatusCodeOnly(resp.StatusCode, string(body), providerName)
	}
	
	// Map the error based on both type and code
	code := mapErrorTypeAndCode(apiErr.Error.Type, apiErr.Error.Code, resp.StatusCode, providerName)
	
	// Build the error with all available context
	opts := []core.ErrorOption{
		core.WithProvider(providerName),
		core.WithHTTPStatus(resp.StatusCode),
		core.WithRaw(apiErr),
	}
	
	// Add retry-after for rate limiting
	if code == core.ErrorRateLimited {
		// Check for retry-after in headers
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			// Parse as seconds
			if seconds := parseRetryAfter(retryAfter); seconds > 0 {
				opts = append(opts, core.WithRetryAfter(time.Duration(seconds)*time.Second))
			}
		} else if retryAfter := resp.Header.Get("X-RateLimit-Reset"); retryAfter != "" {
			// Some providers use X-RateLimit-Reset
			if seconds := parseRetryAfter(retryAfter); seconds > 0 {
				opts = append(opts, core.WithRetryAfter(time.Duration(seconds)*time.Second))
			}
		} else {
			// Default retry after for rate limiting based on provider
			retryDuration := getDefaultRetryAfter(providerName, code)
			opts = append(opts, core.WithRetryAfter(retryDuration))
		}
	}
	
	// Add model information if available from headers
	if model := resp.Header.Get("X-Model"); model != "" {
		opts = append(opts, core.WithModel(model))
	}
	
	// Build error message
	message := apiErr.Error.Message
	if message == "" {
		message = fmt.Sprintf("HTTP %d error from %s", resp.StatusCode, providerName)
	}
	
	return core.NewError(code, message, opts...)
}

// mapErrorTypeAndCode maps provider error types and codes to stable error codes.
func mapErrorTypeAndCode(errorType, errorCode string, statusCode int, providerName string) core.ErrorCode {
	// First check specific error codes (most precise)
	switch errorCode {
	case "context_length_exceeded", "context_window_exceeded", "max_tokens_exceeded":
		return core.ErrorContextLengthExceeded
	case "model_not_found", "model_not_available":
		return core.ErrorNotFound
	case "insufficient_quota", "quota_exceeded":
		return core.ErrorRateLimited
	case "rate_limit_exceeded", "rate_limit_reached":
		return core.ErrorRateLimited
	case "content_filter", "content_blocked":
		return core.ErrorSafetyBlocked
	case "content_policy_violation", "safety_violation":
		return core.ErrorSafetyBlocked
	case "invalid_api_key", "authentication_failed":
		return core.ErrorUnauthorized
	case "service_overloaded", "capacity_exceeded":
		return core.ErrorOverloaded
	case "timeout", "request_timeout":
		return core.ErrorTimeout
	case "invalid_request", "bad_request":
		return core.ErrorInvalidRequest
	}
	
	// Provider-specific error handling
	switch strings.ToLower(providerName) {
	case "groq":
		// Groq-specific error codes
		if strings.Contains(errorCode, "rate") {
			return core.ErrorRateLimited
		}
		if strings.Contains(errorCode, "token") || strings.Contains(errorCode, "context") {
			return core.ErrorContextLengthExceeded
		}
		
	case "cerebras":
		// Cerebras-specific error codes
		if errorCode == "token_limit" {
			return core.ErrorContextLengthExceeded
		}
		if errorCode == "compute_limit" {
			return core.ErrorOverloaded
		}
		
	case "xai", "x.ai":
		// xAI-specific error codes
		if strings.Contains(errorCode, "capacity") {
			return core.ErrorOverloaded
		}
	}
	
	// Then check error types (OpenAI-compatible)
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
		
	case "engine_overloaded_error", "overloaded_error":
		return core.ErrorOverloaded
		
	case "timeout_error":
		return core.ErrorTimeout
		
	case "unsupported_error":
		return core.ErrorUnsupported
	}
	
	// Fall back to status code mapping
	return mapStatusCode(statusCode)
}

// mapStatusCodeOnly handles cases where we only have an HTTP status code.
func mapStatusCodeOnly(statusCode int, body string, providerName string) error {
	code := mapStatusCode(statusCode)
	
	// Try to extract a meaningful message from the body
	message := body
	if len(message) > 200 {
		message = message[:200] + "..."
	}
	if message == "" {
		message = fmt.Sprintf("HTTP %d error from %s", statusCode, providerName)
	}
	
	opts := []core.ErrorOption{
		core.WithProvider(providerName),
		core.WithHTTPStatus(statusCode),
	}
	
	// Add retry hints for transient errors
	if code == core.ErrorRateLimited || code == core.ErrorOverloaded {
		retryDuration := getDefaultRetryAfter(providerName, code)
		opts = append(opts, core.WithRetryAfter(retryDuration))
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
	// Try to parse as seconds
	var seconds int
	if _, err := fmt.Sscanf(value, "%d", &seconds); err == nil {
		return seconds
	}
	
	// Some providers might use timestamps
	// Try to parse as Unix timestamp and calculate difference
	var timestamp int64
	if _, err := fmt.Sscanf(value, "%d", &timestamp); err == nil && timestamp > 1000000000 {
		// Looks like a Unix timestamp
		now := time.Now().Unix()
		if timestamp > now {
			return int(timestamp - now)
		}
	}
	
	return 0
}

// getDefaultRetryAfter returns provider-specific default retry durations.
func getDefaultRetryAfter(providerName string, code core.ErrorCode) time.Duration {
	switch strings.ToLower(providerName) {
	case "groq":
		// Groq has aggressive rate limits but short retry windows
		if code == core.ErrorRateLimited {
			return 10 * time.Second
		}
		return 5 * time.Second
		
	case "cerebras":
		// Cerebras is very fast but has strict limits
		if code == core.ErrorRateLimited {
			return 30 * time.Second
		}
		return 10 * time.Second
		
	case "xai", "x.ai":
		// xAI standard retry
		if code == core.ErrorRateLimited {
			return 60 * time.Second
		}
		return 15 * time.Second
		
	case "baseten":
		// Baseten depends on deployment
		if code == core.ErrorOverloaded {
			return 20 * time.Second
		}
		return 10 * time.Second
		
	default:
		// Conservative defaults
		if code == core.ErrorRateLimited {
			return 60 * time.Second
		}
		if code == core.ErrorOverloaded {
			return 10 * time.Second
		}
		return 5 * time.Second
	}
}

// IsRetryable checks if an error from the provider should be retried.
func IsRetryable(err error) bool {
	// Use the core helper which checks the Temporary flag
	if core.IsTransient(err) {
		return true
	}
	
	// Check for specific conditions that might not be marked as transient
	var aiErr *core.AIError
	if e, ok := err.(*core.AIError); ok {
		aiErr = e
		// Check if it's an overloaded error (always retry)
		if aiErr.Code == core.ErrorOverloaded {
			return true
		}
		// Check HTTP status for additional retry signals
		if aiErr.HTTPStatus == 502 || aiErr.HTTPStatus == 503 || aiErr.HTTPStatus == 504 {
			return true
		}
		// Some providers return 500 for transient issues
		if aiErr.HTTPStatus == 500 && strings.Contains(aiErr.Message, "temporarily") {
			return true
		}
	}
	
	return false
}

// ExtractModelFromError attempts to extract the model name from an error response.
func ExtractModelFromError(err error) string {
	var aiErr *core.AIError
	if e, ok := err.(*core.AIError); ok {
		aiErr = e
		if aiErr.Model != "" {
			return aiErr.Model
		}
		// Try to extract from message (e.g., "The model `llama-3` does not exist")
		if strings.Contains(aiErr.Message, "model `") {
			start := strings.Index(aiErr.Message, "model `") + 7
			if end := strings.Index(aiErr.Message[start:], "`"); end > 0 {
				return aiErr.Message[start : start+end]
			}
		}
		// Also check for "model 'name'" format
		if strings.Contains(aiErr.Message, "model '") {
			start := strings.Index(aiErr.Message, "model '") + 7
			if end := strings.Index(aiErr.Message[start:], "'"); end > 0 {
				return aiErr.Message[start : start+end]
			}
		}
	}
	return ""
}