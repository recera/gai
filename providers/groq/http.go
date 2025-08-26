// Package groq - HTTP request handling and error mapping
package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/recera/gai/core"
)

// doRequest performs an HTTP request with Groq-optimized retry logic.
func (p *Provider) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var lastErr error
	
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			// Fast exponential backoff optimized for Groq's speed
			delay := p.retryDelay * time.Duration(1<<uint(attempt-1))
			
			// Add small jitter
			jitterMs := int64(delay.Nanoseconds()/1000000) / 10
			if jitterMs > 0 {
				delay += time.Duration(jitterMs) * time.Millisecond
			}
			
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := p.doRequestOnce(ctx, method, path, body)
		if err != nil {
			lastErr = err
			continue
		}

		// Check if we should retry based on status code
		if p.shouldRetry(resp.StatusCode) && attempt < p.maxRetries {
			// Read and close body before retry
			io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d (attempt %d)", resp.StatusCode, attempt+1)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("after %d retries: %w", p.maxRetries, lastErr)
}

// doRequestOnce performs a single HTTP request.
func (p *Provider) doRequestOnce(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := p.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "GAI-Groq/1.0")

	// Add custom headers
	for k, v := range p.customHeaders {
		req.Header.Set(k, v)
	}

	return p.client.Do(req)
}

// shouldRetry determines if a request should be retried based on status code.
func (p *Provider) shouldRetry(statusCode int) bool {
	switch statusCode {
	case 429: // Rate limited
		return true
	case 500, 502, 503, 504: // Server errors
		return true
	case 408: // Request timeout
		return true
	default:
		return false
	}
}

// parseError parses an error response from the Groq API.
func (p *Provider) parseError(resp *http.Response) error {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error response", resp.StatusCode)
	}

	// Try to parse as Groq error format
	var groqErr groqErrorResponse
	if err := json.Unmarshal(bodyBytes, &groqErr); err == nil && groqErr.Error.Message != "" {
		return p.mapGroqError(resp.StatusCode, groqErr.Error)
	}

	// Fallback to generic error
	return p.mapHTTPError(resp.StatusCode, string(bodyBytes))
}

// groqErrorResponse represents Groq's error response format.
type groqErrorResponse struct {
	Error groqError `json:"error"`
}

// groqError represents a Groq API error.
type groqError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Param   string `json:"param,omitempty"`
}

// mapGroqError maps Groq-specific errors to core error types.
func (p *Provider) mapGroqError(statusCode int, groqErr groqError) error {
	baseErr := &core.AIError{
		Code:     mapGroqErrorCode(groqErr.Type, groqErr.Code),
		Message:  groqErr.Message,
		Provider: "groq",
		Raw:      groqErr,
	}

	// Map specific error conditions
	switch statusCode {
	case 400:
		if groqErr.Type == "invalid_request_error" {
			baseErr.Code = core.ErrorInvalidRequest
			// Check for specific invalid request types
			if groqErr.Code == "context_length_exceeded" {
				baseErr.Code = core.ErrorContextLengthExceeded
			}
		}
	case 401:
		baseErr.Code = core.ErrorUnauthorized
	case 403:
		baseErr.Code = core.ErrorForbidden
	case 404:
		baseErr.Code = core.ErrorNotFound
	case 429:
		baseErr.Code = core.ErrorRateLimited
		// Parse retry-after header if present
		if retryAfter := parseRetryAfter(groqErr.Message); retryAfter > 0 {
			baseErr.RetryAfter = &retryAfter
		}
	case 500:
		baseErr.Code = core.ErrorInternal
	case 502, 503, 504:
		baseErr.Code = core.ErrorProviderUnavailable
	default:
		baseErr.Code = core.ErrorInternal
	}

	return baseErr
}

// mapGroqErrorCode maps Groq error types and codes to core error codes.
func mapGroqErrorCode(errorType, errorCode string) core.ErrorCode {
	switch errorType {
	case "invalid_request_error":
		switch errorCode {
		case "context_length_exceeded":
			return core.ErrorContextLengthExceeded
		case "invalid_model":
			return core.ErrorNotFound
		case "invalid_api_key":
			return core.ErrorUnauthorized
		default:
			return core.ErrorInvalidRequest
		}
	case "rate_limit_error":
		return core.ErrorRateLimited
	case "authentication_error":
		return core.ErrorUnauthorized
	case "permission_error":
		return core.ErrorForbidden
	case "not_found_error":
		return core.ErrorNotFound
	case "server_error":
		return core.ErrorInternal
	case "service_unavailable_error":
		return core.ErrorProviderUnavailable
	default:
		return core.ErrorInternal
	}
}

// mapHTTPError maps HTTP status codes to core error types when no Groq error format.
func (p *Provider) mapHTTPError(statusCode int, body string) error {
	baseErr := &core.AIError{
		Message:  fmt.Sprintf("HTTP %d: %s", statusCode, body),
		Provider: "groq",
	}

	switch statusCode {
	case 400:
		baseErr.Code = core.ErrorInvalidRequest
	case 401:
		baseErr.Code = core.ErrorUnauthorized
	case 403:
		baseErr.Code = core.ErrorForbidden
	case 404:
		baseErr.Code = core.ErrorNotFound
	case 408:
		baseErr.Code = core.ErrorTimeout
	case 429:
		baseErr.Code = core.ErrorRateLimited
	case 500:
		baseErr.Code = core.ErrorInternal
	case 502:
		baseErr.Code = core.ErrorNetwork
	case 503:
		baseErr.Code = core.ErrorProviderUnavailable
	case 504:
		baseErr.Code = core.ErrorTimeout
	default:
		if statusCode >= 500 {
			baseErr.Code = core.ErrorInternal
		} else {
			baseErr.Code = core.ErrorInternal
		}
	}

	return baseErr
}

// parseRetryAfter extracts retry-after duration from error message.
func parseRetryAfter(message string) time.Duration {
	// Groq typically includes retry information in the error message
	// This is a simplified implementation
	if message == "" {
		return 0
	}
	
	// Common patterns: "Try again in X seconds", "Rate limit reset in X minutes"
	// For now, return a reasonable default
	return 60 * time.Second
}

// Additional HTTP utilities

// setHeaders sets common headers for API requests.
func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "GAI-Groq/1.0")

	// Add custom headers
	for k, v := range p.customHeaders {
		req.Header.Set(k, v)
	}
}

// Health check functionality for connection testing.
func (p *Provider) HealthCheck(ctx context.Context) error {
	// Simple health check using models endpoint
	resp, err := p.doRequest(ctx, "GET", "/models", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetModels retrieves the list of available models.
func (p *Provider) GetModels(ctx context.Context) ([]ModelInfo, error) {
	resp, err := p.doRequest(ctx, "GET", "/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, p.parseError(resp)
	}

	var modelsResp struct {
		Object string      `json:"object"`
		Data   []ModelInfo `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("decoding models response: %w", err)
	}

	// Enrich with our model database
	for i, model := range modelsResp.Data {
		if info := p.getModelInfo(model.ID); info.ID != "" {
			// Merge API data with our database
			modelsResp.Data[i].SupportsVision = info.SupportsVision
			modelsResp.Data[i].SupportsTools = info.SupportsTools
			modelsResp.Data[i].SupportsJSON = info.SupportsJSON
			modelsResp.Data[i].SupportsStreaming = info.SupportsStreaming
			modelsResp.Data[i].RecommendedFor = info.RecommendedFor
			modelsResp.Data[i].PerformanceClass = info.PerformanceClass
			modelsResp.Data[i].IsDeprecated = info.IsDeprecated
		}
	}

	return modelsResp.Data, nil
}