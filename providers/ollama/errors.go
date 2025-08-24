package ollama

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/recera/gai/core"
)

// MapError maps an HTTP response to a standardized error type.
func MapError(resp *http.Response) error {
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading error response: %w", err)
	}

	// Try to parse as Ollama error response
	var ollamaErr errorResponse
	if err := json.Unmarshal(body, &ollamaErr); err == nil && ollamaErr.Error != "" {
		return mapOllamaError(resp.StatusCode, ollamaErr.Error, body)
	}

	// Fallback to generic HTTP error
	return mapStatusCodeError(resp.StatusCode, string(body))
}

// mapOllamaError maps Ollama-specific error messages to standardized error types.
func mapOllamaError(statusCode int, errorMsg string, body []byte) error {
	// Normalize error message for pattern matching
	lowerMsg := strings.ToLower(errorMsg)
	
	// Map based on error message content
	switch {
	case strings.Contains(lowerMsg, "model not found") || strings.Contains(lowerMsg, "model") && strings.Contains(lowerMsg, "not found"):
		return core.NewError(core.ErrorNotFound, errorMsg,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
			core.WithRaw(string(body)),
		)
		
	case strings.Contains(lowerMsg, "out of memory") || strings.Contains(lowerMsg, "insufficient memory"):
		return core.NewError(core.ErrorProviderUnavailable, errorMsg,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
			core.WithRaw(string(body)),
		)
		
	case strings.Contains(lowerMsg, "context length") || strings.Contains(lowerMsg, "context too long"):
		return core.NewError(core.ErrorContextLengthExceeded, errorMsg,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
			core.WithRaw(string(body)),
		)
		
	case strings.Contains(lowerMsg, "invalid") && strings.Contains(lowerMsg, "request"):
		return core.NewError(core.ErrorInvalidRequest, errorMsg,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
			core.WithRaw(string(body)),
		)
		
	case strings.Contains(lowerMsg, "timeout") || strings.Contains(lowerMsg, "deadline exceeded"):
		return core.NewError(core.ErrorTimeout, errorMsg,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
			core.WithRaw(string(body)),
		)
		
	case strings.Contains(lowerMsg, "connection") && (strings.Contains(lowerMsg, "refused") || strings.Contains(lowerMsg, "failed")):
		return core.NewError(core.ErrorNetwork, errorMsg,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
			core.WithRaw(string(body)),
		)
		
	case strings.Contains(lowerMsg, "server") && strings.Contains(lowerMsg, "unavailable"):
		return core.NewError(core.ErrorProviderUnavailable, errorMsg,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
			core.WithRaw(string(body)),
		)
		
	default:
		// Fall back to status code mapping
		return mapStatusCodeError(statusCode, errorMsg)
	}
}

// mapStatusCodeError maps HTTP status codes to standardized error types.
func mapStatusCodeError(statusCode int, message string) error {
	switch statusCode {
	case http.StatusBadRequest:
		return core.NewError(core.ErrorInvalidRequest, message,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
		)
		
	case http.StatusUnauthorized:
		return core.NewError(core.ErrorUnauthorized, message,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
		)
		
	case http.StatusForbidden:
		return core.NewError(core.ErrorForbidden, message,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
		)
		
	case http.StatusNotFound:
		return core.NewError(core.ErrorNotFound, message,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
		)
		
	case http.StatusRequestTimeout:
		return core.NewError(core.ErrorTimeout, message,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
		)
		
	case http.StatusTooManyRequests:
		return core.NewError(core.ErrorRateLimited, message,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
		)
		
	case http.StatusInternalServerError:
		return core.NewError(core.ErrorInternal, message,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
		)
		
	case http.StatusBadGateway:
		return core.NewError(core.ErrorNetwork, message,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
		)
		
	case http.StatusServiceUnavailable:
		return core.NewError(core.ErrorProviderUnavailable, message,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
		)
		
	case http.StatusGatewayTimeout:
		return core.NewError(core.ErrorTimeout, message,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
		)
		
	default:
		if statusCode >= 400 && statusCode < 500 {
			return core.NewError(core.ErrorInvalidRequest, message,
				core.WithHTTPStatus(statusCode),
				core.WithProvider("ollama"),
			)
		} else if statusCode >= 500 {
			return core.NewError(core.ErrorInternal, message,
				core.WithHTTPStatus(statusCode),
				core.WithProvider("ollama"),
			)
		}
		
		return core.NewError(core.ErrorInternal, message,
			core.WithHTTPStatus(statusCode),
			core.WithProvider("ollama"),
		)
	}
}

// mapStatusCode maps HTTP status codes to core error codes for retry logic.
func mapStatusCode(statusCode int) core.ErrorCode {
	switch statusCode {
	case http.StatusBadRequest:
		return core.ErrorInvalidRequest
	case http.StatusUnauthorized:
		return core.ErrorUnauthorized
	case http.StatusForbidden:
		return core.ErrorForbidden
	case http.StatusNotFound:
		return core.ErrorNotFound
	case http.StatusRequestTimeout:
		return core.ErrorTimeout
	case http.StatusTooManyRequests:
		return core.ErrorRateLimited
	case http.StatusInternalServerError:
		return core.ErrorInternal
	case http.StatusBadGateway:
		return core.ErrorNetwork
	case http.StatusServiceUnavailable:
		return core.ErrorProviderUnavailable
	case http.StatusGatewayTimeout:
		return core.ErrorTimeout
	default:
		if statusCode >= 400 && statusCode < 500 {
			return core.ErrorInvalidRequest
		} else if statusCode >= 500 {
			return core.ErrorInternal
		}
		return core.ErrorInternal
	}
}

// IsRetriableError returns true if the error is retriable.
func IsRetriableError(err error) bool {
	var coreErr *core.AIError
	if errors.As(err, &coreErr) {
		return coreErr.Temporary
	}
	return false
}

// IsModelNotFoundError returns true if the error indicates a model was not found.
func IsModelNotFoundError(err error) bool {
	var coreErr *core.AIError
	if errors.As(err, &coreErr) {
		return coreErr.Code == core.ErrorNotFound
	}
	return false
}

// IsInsufficientMemoryError returns true if the error indicates insufficient memory.
func IsInsufficientMemoryError(err error) bool {
	var coreErr *core.AIError
	if errors.As(err, &coreErr) {
		return coreErr.Code == core.ErrorProviderUnavailable && 
			   strings.Contains(strings.ToLower(coreErr.Message), "memory")
	}
	return false
}

// IsContextLengthExceededError returns true if the error indicates context length was exceeded.
func IsContextLengthExceededError(err error) bool {
	var coreErr *core.AIError
	if errors.As(err, &coreErr) {
		return coreErr.Code == core.ErrorContextLengthExceeded
	}
	return false
}