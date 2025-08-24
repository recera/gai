// Package anthropic implements error mapping for Anthropic API responses.
// This file maps Anthropic-specific error responses to the stable GAI error taxonomy.
package anthropic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/recera/gai/core"
)

// ErrorResponse represents the structure of Anthropic API error responses.
type ErrorResponse struct {
	Type    string `json:"type"`    // "error"
	Error   struct {
		Type    string `json:"type"`    // Error type (e.g., "invalid_request_error", "rate_limit_error")
		Message string `json:"message"` // Human-readable error message
	} `json:"error"`
}

// MapError converts an Anthropic API error response to a stable core.AIError.
// It handles both HTTP status codes and Anthropic-specific error types to provide
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
			core.WithProvider("anthropic"),
		)
	}

	// Try to parse the Anthropic error structure
	var apiErr ErrorResponse
	if err := json.Unmarshal(body, &apiErr); err != nil {
		// If we can't parse the error, return a generic error based on status
		return mapStatusCodeOnly(resp.StatusCode, string(body))
	}

	// Map the error based on both type and status code
	code := mapErrorType(apiErr.Error.Type, apiErr.Error.Message, resp.StatusCode)
	
	// Build the error with all available context
	opts := []core.ErrorOption{
		core.WithProvider("anthropic"),
		core.WithHTTPStatus(resp.StatusCode),
		core.WithRaw(apiErr),
	}

	// Add retry-after for rate limiting
	if code == core.ErrorRateLimited {
		// Check if Anthropic includes retry-after in headers
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			// Parse as seconds
			if seconds := parseRetryAfter(retryAfter); seconds > 0 {
				opts = append(opts, core.WithRetryAfter(time.Duration(seconds)*time.Second))
			}
		} else {
			// Default retry after for rate limiting
			opts = append(opts, core.WithRetryAfter(60*time.Second))
		}
	}

	// Add model information if available from headers or error message
	if model := extractModelFromError(apiErr.Error.Message); model != "" {
		opts = append(opts, core.WithModel(model))
	}

	return core.NewError(code, apiErr.Error.Message, opts...)
}

// mapErrorType maps Anthropic error types to stable error codes.
func mapErrorType(errorType, errorMessage string, statusCode int) core.ErrorCode {
	// First check specific error types from Anthropic
	switch errorType {
	case "invalid_request_error":
		// Further differentiate based on message content and status code
		messageLower := strings.ToLower(errorMessage)
		
		if strings.Contains(messageLower, "context") && strings.Contains(messageLower, "length") {
			return core.ErrorContextLengthExceeded
		}
		if strings.Contains(messageLower, "context") && strings.Contains(messageLower, "size") {
			return core.ErrorContextLengthExceeded
		}
		if strings.Contains(messageLower, "token") && strings.Contains(messageLower, "limit") {
			return core.ErrorContextLengthExceeded
		}
		if strings.Contains(messageLower, "model") && strings.Contains(messageLower, "not found") {
			return core.ErrorNotFound
		}
		if strings.Contains(messageLower, "unsupported") {
			return core.ErrorUnsupported
		}
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
	
	case "not_found_error":
		return core.ErrorNotFound
	
	case "rate_limit_error":
		return core.ErrorRateLimited
	
	case "api_error":
		// Generic API errors - differentiate by status code
		if statusCode == http.StatusServiceUnavailable {
			return core.ErrorProviderUnavailable
		}
		if statusCode == http.StatusBadGateway {
			return core.ErrorNetwork
		}
		if statusCode == http.StatusGatewayTimeout {
			return core.ErrorTimeout
		}
		if statusCode == 529 { // Some providers use 529 for overloaded
			return core.ErrorOverloaded
		}
		return core.ErrorInternal
	
	case "overloaded_error":
		return core.ErrorOverloaded
	
	case "internal_server_error":
		return core.ErrorInternal
	
	default:
		// Check message content for additional clues
		messageLower := strings.ToLower(errorMessage)
		
		if strings.Contains(messageLower, "rate limit") {
			return core.ErrorRateLimited
		}
		if strings.Contains(messageLower, "overload") {
			return core.ErrorOverloaded
		}
		if strings.Contains(messageLower, "context") && 
		   (strings.Contains(messageLower, "length") || strings.Contains(messageLower, "size")) {
			return core.ErrorContextLengthExceeded
		}
		if strings.Contains(messageLower, "safety") || 
		   strings.Contains(messageLower, "content policy") ||
		   strings.Contains(messageLower, "filtered") {
			return core.ErrorSafetyBlocked
		}
		if strings.Contains(messageLower, "unauthorized") || 
		   strings.Contains(messageLower, "api key") {
			return core.ErrorUnauthorized
		}
		
		// Fall back to status code mapping
		return mapStatusCode(statusCode)
	}
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
		core.WithProvider("anthropic"),
		core.WithHTTPStatus(statusCode),
	}

	// Add retry hints for transient errors
	if code == core.ErrorRateLimited {
		opts = append(opts, core.WithRetryAfter(60*time.Second))
	} else if code == core.ErrorOverloaded {
		opts = append(opts, core.WithRetryAfter(10*time.Second))
	} else if code == core.ErrorProviderUnavailable {
		opts = append(opts, core.WithRetryAfter(30*time.Second))
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
	case http.StatusUnprocessableEntity:
		// Anthropic sometimes uses 422 for validation errors
		return core.ErrorInvalidRequest
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
	case 529: // Some providers use 529 for overloaded
		return core.ErrorOverloaded
	default:
		if status >= 200 && status < 300 {
			// Success codes - not actually errors, but we need to return something
			// This should not happen in normal error processing
			return core.ErrorInternal // Will not be retried due to success status
		}
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
	// Anthropic typically uses seconds in Retry-After
	var seconds int
	if _, err := fmt.Sscanf(value, "%d", &seconds); err == nil {
		return seconds
	}
	return 0
}

// extractModelFromError attempts to extract the model name from an error message.
func extractModelFromError(message string) string {
	// Look for patterns like "model claude-3-sonnet-20240229 does not exist"
	patterns := []string{
		"model `",
		"model '",
		"model \"",
		"model ",
	}
	
	for _, pattern := range patterns {
		if idx := strings.Index(strings.ToLower(message), pattern); idx >= 0 {
			start := idx + len(pattern)
			if start >= len(message) {
				continue
			}
			
			// Find the end of the model name
			end := start
			for end < len(message) {
				char := message[end]
				if char == ' ' || char == '`' || char == '\'' || char == '"' || 
				   char == '.' || char == ',' || char == '\n' || char == '\r' {
					break
				}
				end++
			}
			
			if end > start {
				model := message[start:end]
				// Basic validation - model names typically contain hyphens and are reasonable length
				if len(model) > 3 && len(model) < 50 && strings.Contains(model, "-") {
					return model
				}
			}
		}
	}
	
	return ""
}

// IsRetryable checks if an error from Anthropic should be retried.
// This is a convenience function that checks both the error code and Anthropic-specific conditions.
func IsRetryable(err error) bool {
	// Use the core helper which checks the Temporary flag
	if core.IsTransient(err) {
		return true
	}

	// Check for specific Anthropic conditions that might not be marked as transient
	var aiErr *core.AIError
	if e, ok := err.(*core.AIError); ok {
		aiErr = e
		// Check if it's an overloaded error (always retry)
		if aiErr.Code == core.ErrorOverloaded {
			return true
		}
		// Check HTTP status for additional retry signals
		if aiErr.HTTPStatus == 502 || aiErr.HTTPStatus == 504 || aiErr.HTTPStatus == 529 {
			return true
		}
		// Check for specific Anthropic error messages that indicate transient issues
		if aiErr.Provider == "anthropic" {
			messageLower := strings.ToLower(aiErr.Message)
			if strings.Contains(messageLower, "temporarily unavailable") ||
			   strings.Contains(messageLower, "server error") ||
			   strings.Contains(messageLower, "timeout") {
				return true
			}
		}
	}

	return false
}

// GetRetryDelay returns the appropriate retry delay for an Anthropic error.
func GetRetryDelay(err error) time.Duration {
	// Use the core helper first
	if delay := core.GetRetryAfter(err); delay > 0 {
		return delay
	}

	// Check for Anthropic-specific retry patterns
	var aiErr *core.AIError
	if e, ok := err.(*core.AIError); ok {
		aiErr = e
		if aiErr.Provider == "anthropic" {
			switch aiErr.Code {
			case core.ErrorRateLimited:
				return 60 * time.Second // Anthropic rate limits tend to be longer
			case core.ErrorOverloaded:
				return 15 * time.Second // Overload recovery time
			case core.ErrorProviderUnavailable:
				return 45 * time.Second // Service downtime
			case core.ErrorNetwork, core.ErrorTimeout:
				return 5 * time.Second
			}
		}
	}

	return 0
}

// IsContextLengthExceeded checks if the error is specifically about context length.
func IsContextLengthExceeded(err error) bool {
	var aiErr *core.AIError
	if e, ok := err.(*core.AIError); ok {
		aiErr = e
		if aiErr.Code == core.ErrorContextLengthExceeded {
			return true
		}
		// Also check message content for additional context length indicators
		if aiErr.Provider == "anthropic" {
			messageLower := strings.ToLower(aiErr.Message)
			return strings.Contains(messageLower, "context") && 
			       (strings.Contains(messageLower, "length") || 
			        strings.Contains(messageLower, "size") ||
			        strings.Contains(messageLower, "limit"))
		}
	}
	return false
}

// IsSafetyFiltered checks if the error is due to content safety filtering.
func IsSafetyFiltered(err error) bool {
	var aiErr *core.AIError
	if e, ok := err.(*core.AIError); ok {
		aiErr = e
		if aiErr.Code == core.ErrorSafetyBlocked {
			return true
		}
		// Also check message content for safety-related terms
		if aiErr.Provider == "anthropic" {
			messageLower := strings.ToLower(aiErr.Message)
			return strings.Contains(messageLower, "safety") ||
			       strings.Contains(messageLower, "content policy") ||
			       strings.Contains(messageLower, "filtered") ||
			       strings.Contains(messageLower, "harmful")
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
		// Try to extract from message
		return extractModelFromError(aiErr.Message)
	}
	return ""
}