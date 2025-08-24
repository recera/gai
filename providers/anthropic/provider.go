// Package anthropic implements the Anthropic provider for the GAI framework.
// It supports Claude models, streaming, structured outputs, and tool calling
// with Anthropic's Messages API format and unique requirements.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/recera/gai/core"
)

const (
	defaultBaseURL = "https://api.anthropic.com"
	defaultTimeout = 60 * time.Second
	defaultVersion = "2023-06-01"
)

// Provider implements the core.Provider interface for Anthropic Claude.
type Provider struct {
	apiKey      string
	baseURL     string
	model       string
	client      *http.Client
	maxRetries  int
	retryDelay  time.Duration
	version     string
	collector   core.MetricsCollector
	mu          sync.RWMutex
}

// Option configures the Anthropic provider.
type Option func(*Provider)

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(p *Provider) {
		p.apiKey = key
	}
}

// WithBaseURL sets a custom base URL (useful for proxies or compatible APIs).
func WithBaseURL(url string) Option {
	return func(p *Provider) {
		p.baseURL = url
	}
}

// WithModel sets the default model to use.
func WithModel(model string) Option {
	return func(p *Provider) {
		p.model = model
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(p *Provider) {
		p.client = client
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(n int) Option {
	return func(p *Provider) {
		p.maxRetries = n
	}
}

// WithRetryDelay sets the base delay between retries.
func WithRetryDelay(d time.Duration) Option {
	return func(p *Provider) {
		p.retryDelay = d
	}
}

// WithVersion sets the Anthropic API version.
func WithVersion(version string) Option {
	return func(p *Provider) {
		p.version = version
	}
}

// WithMetricsCollector sets the metrics collector for observability.
func WithMetricsCollector(collector core.MetricsCollector) Option {
	return func(p *Provider) {
		p.collector = collector
	}
}

// New creates a new Anthropic provider with the given options.
func New(opts ...Option) *Provider {
	p := &Provider{
		baseURL:    defaultBaseURL,
		model:      "claude-sonnet-4-20250514",
		maxRetries: 3,
		retryDelay: 100 * time.Millisecond,
		version:    defaultVersion,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.client == nil {
		p.client = &http.Client{
			Timeout: defaultTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	}

	return p
}

// getModel returns the model to use for the request.
func (p *Provider) getModel(req core.Request) string {
	if req.Model != "" {
		return req.Model
	}
	return p.model
}

// convertRequest converts a core.Request to an Anthropic messages request.
func (p *Provider) convertRequest(req core.Request) (*messagesRequest, error) {
	ar := &messagesRequest{
		Model:     p.getModel(req),
		MaxTokens: req.MaxTokens,
	}

	// Set default max tokens if not specified
	if ar.MaxTokens == 0 {
		ar.MaxTokens = 4096
	}

	// Handle optional fields
	if req.Temperature > 0 {
		ar.Temperature = &req.Temperature
	}

	// Convert messages - Anthropic has special handling for system messages
	messages, system, err := p.convertMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("converting messages: %w", err)
	}
	ar.Messages = messages
	if system != "" {
		ar.System = system
	}

	// Convert tools if present
	if len(req.Tools) > 0 {
		ar.Tools = p.convertTools(req.Tools)
	}

	// Handle provider-specific options
	if opts, ok := req.ProviderOptions["anthropic"].(map[string]interface{}); ok {
		p.applyProviderOptions(ar, opts)
	}

	return ar, nil
}

// convertMessages converts core messages to Anthropic format.
// Anthropic requires system messages to be in a separate field, not in the messages array.
func (p *Provider) convertMessages(messages []core.Message) ([]message, string, error) {
	var result []message
	var systemPrompt string

	for _, msg := range messages {
		switch msg.Role {
		case core.System:
			// System messages go to the system field, not in messages array
			if len(msg.Parts) > 0 {
				if text, ok := msg.Parts[0].(core.Text); ok {
					if systemPrompt != "" {
						systemPrompt += "\n\n" + text.Text
					} else {
						systemPrompt = text.Text
					}
				}
			}
		case core.User, core.Assistant:
			// Convert message content
			content, err := p.convertParts(msg.Parts)
			if err != nil {
				return nil, "", err
			}

			am := message{
				Role:    string(msg.Role),
				Content: content,
			}

			result = append(result, am)
		case core.Tool:
			// Tool messages are handled differently in Anthropic
			// They become tool_result blocks in the previous assistant message
			// For now, we'll convert them to user messages with tool result content
			content, err := p.convertParts(msg.Parts)
			if err != nil {
				return nil, "", err
			}

			am := message{
				Role:    "user",
				Content: content,
			}

			result = append(result, am)
		}
	}

	return result, systemPrompt, nil
}

// convertParts converts message parts to Anthropic content format.
func (p *Provider) convertParts(parts []core.Part) (interface{}, error) {
	if len(parts) == 0 {
		return "", nil
	}

	// Single text part can be a string
	if len(parts) == 1 {
		if text, ok := parts[0].(core.Text); ok {
			return text.Text, nil
		}
	}

	// Multiple parts or non-text parts use content array
	var content []contentBlock
	for _, part := range parts {
		switch p := part.(type) {
		case core.Text:
			content = append(content, contentBlock{
				Type: "text",
				Text: p.Text,
			})
		case core.ImageURL:
			// Anthropic uses a different format for images
			content = append(content, contentBlock{
				Type: "image",
				Source: &imageSource{
					Type:      "base64",
					MediaType: "image/jpeg", // Default, should ideally parse from URL
					Data:      p.URL,        // This would need to be base64 encoded data
				},
			})
		case core.Audio, core.Video, core.File:
			// Anthropic doesn't support these content types in messages
			return nil, fmt.Errorf("unsupported part type for Anthropic: %T", p)
		default:
			return nil, fmt.Errorf("unknown part type: %T", p)
		}
	}

	return content, nil
}

// convertTools converts core tools to Anthropic format.
func (p *Provider) convertTools(tools []core.ToolHandle) []tool {
	result := make([]tool, 0, len(tools))

	for _, t := range tools {
		// Parse the JSON schema for input
		var inputSchema map[string]interface{}
		if err := json.Unmarshal(t.InSchemaJSON(), &inputSchema); err != nil {
			// If we can't parse the schema, use a basic object schema
			inputSchema = map[string]interface{}{
				"type": "object",
			}
		}

		result = append(result, tool{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: inputSchema,
		})
	}

	return result
}

// applyProviderOptions applies Anthropic-specific options.
func (p *Provider) applyProviderOptions(req *messagesRequest, opts map[string]interface{}) {
	if v, ok := opts["top_p"].(float32); ok {
		req.TopP = &v
	}
	if v, ok := opts["top_k"].(int); ok {
		req.TopK = &v
	}
	if v, ok := opts["stop_sequences"].([]string); ok {
		req.StopSequences = v
	}
}

// doRequest performs an HTTP request with retry logic.
func (p *Provider) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			delay := p.retryDelay * time.Duration(1<<uint(attempt-1))
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
		if p.shouldRetry(resp.StatusCode) {
			// Only retry if we have attempts left
			if attempt < p.maxRetries {
				bodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, bodyBytes)
				continue
			}
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

	// Set Anthropic-specific headers
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", p.version)
	req.Header.Set("content-type", "application/json")

	return p.client.Do(req)
}

// shouldRetry determines if a request should be retried based on status code.
func (p *Provider) shouldRetry(statusCode int) bool {
	// Don't retry on success codes
	if statusCode >= 200 && statusCode < 300 {
		return false
	}
	
	// Map the status code to our error taxonomy to determine if it's retryable
	code := mapStatusCode(statusCode)
	// Check if this error code is typically transient
	switch code {
	case core.ErrorRateLimited, core.ErrorOverloaded, core.ErrorTimeout,
		core.ErrorNetwork, core.ErrorProviderUnavailable, core.ErrorInternal:
		return true
	default:
		// Also retry on specific status codes that might not map cleanly
		return statusCode == 502 || statusCode == 504
	}
}

// parseError parses an error response from the API.
func (p *Provider) parseError(resp *http.Response) error {
	// Use the error mapper
	return MapError(resp)
}