package gemini

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/recera/gai/core"
)

// mapError converts Gemini API errors to GAI error types.
func mapError(errResp *ErrorResponse, statusCode int) error {
	if errResp == nil {
		return core.NewError(
			core.ErrorInternal,
			fmt.Sprintf("unknown error with status %d", statusCode),
			core.WithProvider("gemini"),
		)
	}

	message := errResp.Error.Message
	status := errResp.Error.Status
	
	// Check for specific error types in details
	var errorCode core.ErrorCode
	var retryAfter time.Duration
	temporary := false

	// Map by status code first
	switch statusCode {
	case http.StatusBadRequest:
		errorCode = core.ErrorInvalidRequest
		// Check for specific bad request types
		if strings.Contains(message, "context length") || strings.Contains(message, "token") {
			errorCode = core.ErrorContextLengthExceeded
		}
	case http.StatusUnauthorized:
		errorCode = core.ErrorUnauthorized
	case http.StatusForbidden:
		errorCode = core.ErrorForbidden
		// Check if it's safety-related
		if strings.Contains(message, "safety") || strings.Contains(message, "blocked") {
			errorCode = core.ErrorSafetyBlocked
		}
	case http.StatusNotFound:
		errorCode = core.ErrorNotFound
	case http.StatusTooManyRequests:
		errorCode = core.ErrorRateLimited
		temporary = true
		retryAfter = 30 * time.Second // Default retry after
		
		// Try to extract retry-after from details
		for _, detail := range errResp.Error.Details {
			if detail.Metadata != nil {
				if retryStr, ok := detail.Metadata["retry_after"]; ok {
					if duration, err := time.ParseDuration(retryStr); err == nil {
						retryAfter = duration
					}
				}
			}
		}
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		errorCode = core.ErrorProviderUnavailable
		temporary = true
		retryAfter = 5 * time.Second
	case http.StatusGatewayTimeout:
		errorCode = core.ErrorTimeout
		temporary = true
		retryAfter = 2 * time.Second
	default:
		if statusCode >= 500 {
			errorCode = core.ErrorProviderUnavailable
			temporary = true
		} else {
			errorCode = core.ErrorInternal
		}
	}

	// Check status string for additional context
	switch strings.ToUpper(status) {
	case "INVALID_ARGUMENT":
		errorCode = core.ErrorInvalidRequest
	case "FAILED_PRECONDITION":
		errorCode = core.ErrorInvalidRequest
	case "OUT_OF_RANGE":
		errorCode = core.ErrorContextLengthExceeded
	case "UNAUTHENTICATED":
		errorCode = core.ErrorUnauthorized
	case "PERMISSION_DENIED":
		errorCode = core.ErrorForbidden
	case "NOT_FOUND":
		errorCode = core.ErrorNotFound
	case "RESOURCE_EXHAUSTED":
		errorCode = core.ErrorRateLimited
		temporary = true
	case "CANCELLED":
		errorCode = core.ErrorTimeout
	case "DEADLINE_EXCEEDED":
		errorCode = core.ErrorTimeout
		temporary = true
	case "UNAVAILABLE":
		errorCode = core.ErrorProviderUnavailable
		temporary = true
	case "UNIMPLEMENTED":
		errorCode = core.ErrorUnsupported
	}

	// Check message content for specific errors
	messageLower := strings.ToLower(message)
	if strings.Contains(messageLower, "quota") || strings.Contains(messageLower, "rate limit") {
		errorCode = core.ErrorRateLimited
		temporary = true
	} else if strings.Contains(messageLower, "api key") || strings.Contains(messageLower, "authentication") {
		errorCode = core.ErrorUnauthorized
	} else if strings.Contains(messageLower, "permission") || strings.Contains(messageLower, "access denied") {
		errorCode = core.ErrorForbidden
	} else if strings.Contains(messageLower, "safety") || strings.Contains(messageLower, "blocked") || strings.Contains(messageLower, "harmful") {
		errorCode = core.ErrorSafetyBlocked
	} else if strings.Contains(messageLower, "context length") || strings.Contains(messageLower, "token limit") || strings.Contains(messageLower, "too long") {
		errorCode = core.ErrorContextLengthExceeded
	} else if strings.Contains(messageLower, "model") && strings.Contains(messageLower, "not found") {
		errorCode = core.ErrorNotFound
	} else if strings.Contains(messageLower, "timeout") || strings.Contains(messageLower, "deadline") {
		errorCode = core.ErrorTimeout
		temporary = true
	} else if strings.Contains(messageLower, "overloaded") || strings.Contains(messageLower, "capacity") {
		errorCode = core.ErrorOverloaded
		temporary = true
		retryAfter = 10 * time.Second
	}

	// Create error with appropriate options
	opts := []core.ErrorOption{
		core.WithProvider("gemini"),
		core.WithTemporary(temporary),
	}
	
	if retryAfter > 0 {
		opts = append(opts, core.WithRetryAfter(retryAfter))
	}
	
	if statusCode > 0 {
		opts = append(opts, core.WithHTTPStatus(statusCode))
	}

	// Add raw error response for debugging
	if errResp != nil {
		opts = append(opts, core.WithRaw(errResp))
	}

	return core.NewError(errorCode, message, opts...)
}