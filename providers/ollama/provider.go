// Package ollama implements the Ollama provider for the GAI framework.
// It supports local models, streaming, structured outputs, and tool calling
// with Ollama's chat and generate API endpoints.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/recera/gai/core"
)

const (
	defaultBaseURL = "http://localhost:11434"
	defaultTimeout = 120 * time.Second // Longer timeout for local models
	defaultModel   = "llama3.2"
)

// Provider implements the core.Provider interface for Ollama.
type Provider struct {
	baseURL     string
	model       string
	client      *http.Client
	maxRetries  int
	retryDelay  time.Duration
	collector   core.MetricsCollector
	mu          sync.RWMutex
	
	// Ollama-specific options
	useGenerateAPI bool // Use /api/generate instead of /api/chat
	keepAlive      string
	template       string
}

// Option configures the Ollama provider.
type Option func(*Provider)

// WithBaseURL sets a custom base URL (default: http://localhost:11434).
func WithBaseURL(url string) Option {
	return func(p *Provider) {
		p.baseURL = strings.TrimSuffix(url, "/")
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

// WithMetricsCollector sets the metrics collector for observability.
func WithMetricsCollector(collector core.MetricsCollector) Option {
	return func(p *Provider) {
		p.collector = collector
	}
}

// WithGenerateAPI configures the provider to use /api/generate instead of /api/chat.
// This is useful for models that don't support the chat format.
func WithGenerateAPI(use bool) Option {
	return func(p *Provider) {
		p.useGenerateAPI = use
	}
}

// WithKeepAlive sets how long the model stays loaded in memory (default: 5m).
func WithKeepAlive(keepAlive string) Option {
	return func(p *Provider) {
		p.keepAlive = keepAlive
	}
}

// WithTemplate sets a custom prompt template for the model.
func WithTemplate(template string) Option {
	return func(p *Provider) {
		p.template = template
	}
}

// New creates a new Ollama provider with the given options.
func New(opts ...Option) *Provider {
	p := &Provider{
		baseURL:    defaultBaseURL,
		model:      defaultModel,
		maxRetries: 3,
		retryDelay: 100 * time.Millisecond,
		keepAlive:  "5m",
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

// convertRequest converts a core.Request to an Ollama chat request.
func (p *Provider) convertRequest(req core.Request) (*chatRequest, error) {
	model := p.getModel(req)
	
	// Convert messages
	messages, err := p.convertMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("converting messages: %w", err)
	}

	// Create chat request
	chatReq := NewChatRequest(model, messages)
	
	// Set template if configured
	if p.template != "" {
		chatReq.Template = p.template
	}
	
	// Set keep alive
	if p.keepAlive != "" {
		chatReq.KeepAlive = &p.keepAlive
	}

	// Handle optional fields
	if req.Temperature > 0 {
		chatReq = chatReq.WithTemperature(req.Temperature)
	}
	
	if req.MaxTokens > 0 {
		chatReq = chatReq.WithMaxTokens(req.MaxTokens)
	}

	// Convert tools if present
	if len(req.Tools) > 0 {
		tools := p.convertTools(req.Tools)
		chatReq = chatReq.WithTools(tools)
	}

	// Handle provider-specific options
	if opts, ok := req.ProviderOptions["ollama"].(map[string]interface{}); ok {
		p.applyProviderOptions(chatReq, opts)
	}

	return chatReq, nil
}

// convertMessages converts core messages to Ollama format.
func (p *Provider) convertMessages(messages []core.Message) ([]chatMessage, error) {
	result := make([]chatMessage, 0, len(messages))

	for _, msg := range messages {
		ollamaMsg := chatMessage{
			Role: string(msg.Role),
		}

		// Handle multimodal content
		var textContent strings.Builder
		var images []string
		
		for _, part := range msg.Parts {
			switch p := part.(type) {
			case core.Text:
				textContent.WriteString(p.Text)
			case core.ImageURL:
				// For Ollama, we need base64 encoded images
				// This is a placeholder - in practice, you'd need to fetch and encode the image
				images = append(images, p.URL)
			case core.Audio, core.Video, core.File:
				// Ollama doesn't directly support these in chat
				return nil, fmt.Errorf("unsupported part type for Ollama: %T", p)
			default:
				return nil, fmt.Errorf("unknown part type: %T", p)
			}
		}

		ollamaMsg.Content = textContent.String()
		if len(images) > 0 {
			ollamaMsg.Images = images
		}

		result = append(result, ollamaMsg)
	}

	return result, nil
}

// convertTools converts core tools to Ollama format.
func (p *Provider) convertTools(tools []core.ToolHandle) []chatTool {
	result := make([]chatTool, 0, len(tools))

	for _, tool := range tools {
		result = append(result, chatTool{
			Type: "function",
			Function: function{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.InSchemaJSON(),
			},
		})
	}

	return result
}

// applyProviderOptions applies Ollama-specific options.
func (p *Provider) applyProviderOptions(req *chatRequest, opts map[string]interface{}) {
	if req.Options == nil {
		req.Options = &modelOptions{}
	}

	if v, ok := opts["top_k"].(int); ok {
		req.Options.TopK = &v
	}
	if v, ok := opts["top_p"].(float32); ok {
		req.Options.TopP = &v
	}
	if v, ok := opts["repeat_penalty"].(float32); ok {
		req.Options.RepeatPenalty = &v
	}
	if v, ok := opts["seed"].(int); ok {
		req.Options.Seed = &v
	}
	if v, ok := opts["num_ctx"].(int); ok {
		req.Options.NumCtx = &v
	}
	if v, ok := opts["num_gpu"].(int); ok {
		req.Options.NumGPU = &v
	}
	if v, ok := opts["low_vram"].(bool); ok {
		req.Options.LowVRAM = &v
	}
	if v, ok := opts["stop"].([]string); ok {
		req.Options.Stop = v
	}
	if v, ok := opts["frequency_penalty"].(float32); ok {
		req.Options.FrequencyPenalty = &v
	}
	if v, ok := opts["presence_penalty"].(float32); ok {
		req.Options.PresencePenalty = &v
	}
	if v, ok := opts["mirostat"].(int); ok {
		req.Options.Mirostat = &v
	}
	if v, ok := opts["mirostat_eta"].(float32); ok {
		req.Options.MirostatEta = &v
	}
	if v, ok := opts["mirostat_tau"].(float32); ok {
		req.Options.MirostatTau = &v
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

	// Set headers (Ollama doesn't require authentication by default)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

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

// ListModels returns a list of available models from the Ollama server.
func (p *Provider) ListModels(ctx context.Context) ([]model, error) {
	resp, err := p.doRequest(ctx, "GET", "/api/tags", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var modelsResp modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("decoding models response: %w", err)
	}

	return modelsResp.Models, nil
}

// IsModelAvailable checks if a specific model is available on the server.
func (p *Provider) IsModelAvailable(ctx context.Context, modelName string) (bool, error) {
	models, err := p.ListModels(ctx)
	if err != nil {
		return false, err
	}

	for _, model := range models {
		if model.Name == modelName || strings.HasPrefix(model.Name, modelName+":") {
			return true, nil
		}
	}

	return false, nil
}

// PullModel pulls a model from the Ollama library.
// This method initiates a model download and returns immediately.
// Use context cancellation to abort the download.
func (p *Provider) PullModel(ctx context.Context, modelName string) error {
	pullReq := map[string]string{"name": modelName}
	
	resp, err := p.doRequest(ctx, "POST", "/api/pull", pullReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.parseError(resp)
	}

	// For now, we just start the pull. In a more complete implementation,
	// you might want to stream the progress updates.
	return nil
}