// Package openai implements the OpenAI provider for the GAI framework.
// It supports Chat Completions, streaming, structured outputs, and tool calling
// with both the latest API features and fallback compatibility modes.
package openai

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
	defaultBaseURL = "https://api.openai.com/v1"
	defaultTimeout = 60 * time.Second
)

// Provider implements the core.Provider interface for OpenAI.
type Provider struct {
	apiKey      string
	baseURL     string
	model       string
	client      *http.Client
	maxRetries  int
	retryDelay  time.Duration
	org         string
	project     string
	collector   core.MetricsCollector
	mu          sync.RWMutex
}

// Option configures the OpenAI provider.
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

// WithOrganization sets the organization ID for requests.
func WithOrganization(org string) Option {
	return func(p *Provider) {
		p.org = org
	}
}

// WithProject sets the project ID for requests.
func WithProject(project string) Option {
	return func(p *Provider) {
		p.project = project
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

// WithMetricsCollector sets the metrics collector for observability.
func WithMetricsCollector(collector core.MetricsCollector) Option {
	return func(p *Provider) {
		p.collector = collector
	}
}

// New creates a new OpenAI provider with the given options.
func New(opts ...Option) *Provider {
	p := &Provider{
		baseURL:    defaultBaseURL,
		model:      "gpt-4o-mini",
		maxRetries: 3,
		retryDelay: 100 * time.Millisecond,
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

// chatCompletionRequest represents the request structure for OpenAI's Chat Completions API.
type chatCompletionRequest struct {
	Model            string                   `json:"model"`
	Messages         []chatMessage            `json:"messages"`
	Temperature      *float32                 `json:"temperature,omitempty"`
	MaxTokens        *int                     `json:"max_tokens,omitempty"`
	Tools            []chatTool               `json:"tools,omitempty"`
	ToolChoice       interface{}              `json:"tool_choice,omitempty"`
	Stream           bool                     `json:"stream,omitempty"`
	ResponseFormat   *responseFormat          `json:"response_format,omitempty"`
	StreamOptions    *streamOptions           `json:"stream_options,omitempty"`
	ParallelToolCalls *bool                   `json:"parallel_tool_calls,omitempty"`
	N                int                      `json:"n,omitempty"`
	Stop             []string                 `json:"stop,omitempty"`
	PresencePenalty  *float32                 `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32                 `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]float32       `json:"logit_bias,omitempty"`
	User             string                   `json:"user,omitempty"`
	Seed             *int                     `json:"seed,omitempty"`
	TopP             *float32                 `json:"top_p,omitempty"`
}

// chatMessage represents a message in the chat conversation.
type chatMessage struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"` // string or []contentPart
	Name       string      `json:"name,omitempty"`
	ToolCalls  []toolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// contentPart represents a part of multimodal content.
type contentPart struct {
	Type     string       `json:"type"`
	Text     string       `json:"text,omitempty"`
	ImageURL *imageURLPart `json:"image_url,omitempty"`
}

// imageURLPart represents an image URL in content.
type imageURLPart struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// chatTool represents a tool available to the model.
type chatTool struct {
	Type     string   `json:"type"`
	Function function `json:"function"`
}

// function represents a function tool.
type function struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	Strict      bool            `json:"strict,omitempty"`
}

// toolCall represents a tool call made by the model.
type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

// functionCall represents the function call details.
type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// responseFormat specifies the output format.
type responseFormat struct {
	Type       string          `json:"type"` // "text", "json_object", "json_schema"
	JSONSchema *jsonSchemaFormat `json:"json_schema,omitempty"`
}

// jsonSchemaFormat specifies a JSON schema for structured outputs.
type jsonSchemaFormat struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema"`
	Strict      bool            `json:"strict"`
}

// streamOptions configures streaming behavior.
type streamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// chatCompletionResponse represents the response from the Chat Completions API.
type chatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

// choice represents a completion choice.
type choice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
	LogProbs     interface{} `json:"logprobs"`
}

// usage represents token usage information.
type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// streamChunk represents a chunk in the streaming response.
type streamChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []deltaChoice `json:"choices"`
	Usage   *usage        `json:"usage,omitempty"`
}

// deltaChoice represents a streaming choice with delta content.
type deltaChoice struct {
	Index        int          `json:"index"`
	Delta        messageDelta `json:"delta"`
	FinishReason *string      `json:"finish_reason"`
	LogProbs     interface{}  `json:"logprobs"`
}

// messageDelta represents incremental message content.
type messageDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   *string    `json:"content,omitempty"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
}

// convertRequest converts a core.Request to an OpenAI chat completion request.
func (p *Provider) convertRequest(req core.Request) (*chatCompletionRequest, error) {
	ocr := &chatCompletionRequest{
		Model: p.getModel(req),
		N:     1,
	}

	// Handle optional fields
	if req.Temperature > 0 {
		ocr.Temperature = &req.Temperature
	}
	if req.MaxTokens > 0 {
		ocr.MaxTokens = &req.MaxTokens
	}

	// Convert messages
	messages, err := p.convertMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("converting messages: %w", err)
	}
	ocr.Messages = messages

	// Convert tools if present
	if len(req.Tools) > 0 {
		ocr.Tools = p.convertTools(req.Tools)
		ocr.ToolChoice = p.convertToolChoice(req.ToolChoice)
		parallelCalls := true
		ocr.ParallelToolCalls = &parallelCalls
	}

	// Handle provider-specific options
	if opts, ok := req.ProviderOptions["openai"].(map[string]interface{}); ok {
		p.applyProviderOptions(ocr, opts)
	}

	return ocr, nil
}

// getModel returns the model to use for the request.
func (p *Provider) getModel(req core.Request) string {
	if req.Model != "" {
		return req.Model
	}
	return p.model
}

// convertMessages converts core messages to OpenAI format.
func (p *Provider) convertMessages(messages []core.Message) ([]chatMessage, error) {
	result := make([]chatMessage, 0, len(messages))
	
	for _, msg := range messages {
		cm := chatMessage{
			Role: string(msg.Role),
			Name: msg.Name,
		}

		// Handle multimodal content
		if len(msg.Parts) == 1 {
			// Single part - use string content for text
			if text, ok := msg.Parts[0].(core.Text); ok {
				cm.Content = text.Text
			} else {
				// Convert to content parts for other types
				parts, err := p.convertParts(msg.Parts)
				if err != nil {
					return nil, err
				}
				cm.Content = parts
			}
		} else if len(msg.Parts) > 1 {
			// Multiple parts - use content array
			parts, err := p.convertParts(msg.Parts)
			if err != nil {
				return nil, err
			}
			cm.Content = parts
		}

		result = append(result, cm)
	}

	return result, nil
}

// convertParts converts message parts to OpenAI content parts.
func (p *Provider) convertParts(parts []core.Part) ([]contentPart, error) {
	result := make([]contentPart, 0, len(parts))
	
	for _, part := range parts {
		switch p := part.(type) {
		case core.Text:
			result = append(result, contentPart{
				Type: "text",
				Text: p.Text,
			})
		case core.ImageURL:
			result = append(result, contentPart{
				Type: "image_url",
				ImageURL: &imageURLPart{
					URL:    p.URL,
					Detail: p.Detail,
				},
			})
		case core.Audio, core.Video, core.File:
			// OpenAI doesn't directly support these in chat completions
			// Would need to handle via assistants API or convert to supported format
			return nil, fmt.Errorf("unsupported part type: %T", p)
		default:
			return nil, fmt.Errorf("unknown part type: %T", p)
		}
	}
	
	return result, nil
}

// convertTools converts core tools to OpenAI format.
func (p *Provider) convertTools(tools []core.ToolHandle) []chatTool {
	result := make([]chatTool, 0, len(tools))
	
	for _, tool := range tools {
		result = append(result, chatTool{
			Type: "function",
			Function: function{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.InSchemaJSON(),
				Strict:      false, // Can be made configurable
			},
		})
	}
	
	return result
}

// convertToolChoice converts core tool choice to OpenAI format.
func (p *Provider) convertToolChoice(choice core.ToolChoice) interface{} {
	switch choice {
	case core.ToolAuto:
		return "auto"
	case core.ToolNone:
		return "none"
	case core.ToolRequired:
		return "required"
	default:
		return "auto"
	}
}

// applyProviderOptions applies OpenAI-specific options.
func (p *Provider) applyProviderOptions(req *chatCompletionRequest, opts map[string]interface{}) {
	if v, ok := opts["presence_penalty"].(float32); ok {
		req.PresencePenalty = &v
	}
	if v, ok := opts["frequency_penalty"].(float32); ok {
		req.FrequencyPenalty = &v
	}
	if v, ok := opts["top_p"].(float32); ok {
		req.TopP = &v
	}
	if v, ok := opts["stop"].([]string); ok {
		req.Stop = v
	}
	if v, ok := opts["seed"].(int); ok {
		req.Seed = &v
	}
	if v, ok := opts["user"].(string); ok {
		req.User = v
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
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
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

	// Set headers
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if p.org != "" {
		req.Header.Set("OpenAI-Organization", p.org)
	}
	if p.project != "" {
		req.Header.Set("OpenAI-Project", p.project)
	}

	return p.client.Do(req)
}

// shouldRetry determines if a request should be retried based on status code.
func (p *Provider) shouldRetry(statusCode int) bool {
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
	// Use the new error mapper
	return MapError(resp)
}