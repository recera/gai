// Package openai_compat provides an adapter for OpenAI-compatible APIs.
// It supports providers like Groq, xAI, Baseten, and Cerebras that implement
// the OpenAI API specification with potential variations and limitations.
package openai_compat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/recera/gai/core"
)

const (
	defaultTimeout = 60 * time.Second
)

// Provider implements the core.Provider interface for OpenAI-compatible APIs.
// It adapts to various providers' quirks and limitations automatically.
type Provider struct {
	config      CompatOpts
	client      *http.Client
	capabilities *Capabilities
	mu          sync.RWMutex
	
	// Cached values
	baseURL     *url.URL
}

// CompatOpts configures the OpenAI-compatible provider.
type CompatOpts struct {
	// Required configuration
	BaseURL string // Base URL for the API (e.g., "https://api.groq.com/openai/v1")
	APIKey  string // API key for authentication
	
	// Model configuration
	DefaultModel string // Default model to use if not specified in request
	
	// Feature toggles for provider limitations
	DisableJSONStreaming      bool // Some providers don't support JSON streaming
	DisableParallelToolCalls  bool // Some providers don't support parallel tool execution
	DisableStrictJSONSchema   bool // Some providers don't support strict JSON schema mode
	DisableToolChoice         bool // Some providers don't support tool_choice parameter
	
	// Request modifications
	UnsupportedParams []string // Parameters to strip from requests
	ForceResponseFormat string  // Force a specific response format (e.g., "json_object")
	
	// Advanced options
	PreferResponsesAPI bool          // Use /responses endpoint if available
	CustomHeaders      map[string]string // Additional headers to send with requests
	MaxRetries         int           // Maximum retry attempts (default: 3)
	RetryDelay         time.Duration // Base delay between retries (default: 1s)
	HTTPClient         *http.Client  // Custom HTTP client
	
	// Observability
	MetricsCollector core.MetricsCollector
	
	// Provider identification (for error messages and telemetry)
	ProviderName string // e.g., "groq", "xai", "baseten", "cerebras"
}

// Capabilities represents the detected or configured capabilities of the provider.
type Capabilities struct {
	Models              []ModelInfo
	SupportsTools       bool
	SupportsStreaming   bool
	SupportsJSONMode    bool
	SupportsVision      bool
	MaxContextWindow    int
	DefaultTemperature  float32
	LastProbed          time.Time
}

// ModelInfo contains information about a supported model.
type ModelInfo struct {
	ID           string `json:"id"`
	Object       string `json:"object"`
	Created      int64  `json:"created"`
	OwnedBy      string `json:"owned_by"`
	ContextWindow int   `json:"context_window,omitempty"`
}

// New creates a new OpenAI-compatible provider with the given options.
func New(opts CompatOpts) (*Provider, error) {
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	
	// Parse and validate base URL
	baseURL, err := url.Parse(opts.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	
	// Validate that it's a proper URL with scheme
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("invalid base URL: must include scheme and host")
	}
	
	// Ensure base URL ends with /v1 or similar
	if !strings.HasSuffix(baseURL.Path, "/v1") && !strings.HasSuffix(baseURL.Path, "/v1/") {
		if !strings.HasSuffix(baseURL.Path, "/") {
			baseURL.Path += "/"
		}
		baseURL.Path += "v1"
	}
	
	// Set defaults
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 3
	}
	if opts.RetryDelay == 0 {
		opts.RetryDelay = time.Second
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{
			Timeout: defaultTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
		}
	}
	
	// Apply provider-specific defaults if name is provided
	if opts.ProviderName != "" {
		applyProviderDefaults(&opts)
	}
	
	p := &Provider{
		config:  opts,
		client:  opts.HTTPClient,
		baseURL: baseURL,
	}
	
	// Optionally probe capabilities on creation
	// This is done asynchronously to not block initialization
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = p.probeCapabilities(ctx)
	}()
	
	return p, nil
}

// applyProviderDefaults applies known defaults for specific providers.
func applyProviderDefaults(opts *CompatOpts) {
	switch strings.ToLower(opts.ProviderName) {
	case "groq":
		// Groq has very fast inference but some limitations
		if opts.DefaultModel == "" {
			opts.DefaultModel = "llama-3.3-70b-versatile"
		}
		// Groq supports most OpenAI features
		
	case "xai", "x.ai":
		// xAI (Grok) configuration
		if opts.DefaultModel == "" {
			opts.DefaultModel = "grok-2-latest"
		}
		
	case "cerebras":
		// Cerebras is very fast but has limitations
		opts.DisableJSONStreaming = true
		opts.DisableParallelToolCalls = true
		if opts.DefaultModel == "" {
			opts.DefaultModel = "llama-3.3-70b"
		}
		
	case "baseten":
		// Baseten is configurable, depends on deployed model
		// No specific defaults, user should configure based on their deployment
	}
}

// probeCapabilities attempts to detect provider capabilities.
func (p *Provider) probeCapabilities(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Try to fetch model list
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL.String()+"/models", nil)
	if err != nil {
		return fmt.Errorf("creating models request: %w", err)
	}
	
	p.setHeaders(req)
	
	resp, err := p.client.Do(req)
	if err != nil {
		// If we can't probe, use defaults
		p.capabilities = &Capabilities{
			SupportsTools:     !p.config.DisableToolChoice,
			SupportsStreaming: true,
			SupportsJSONMode:  !p.config.DisableJSONStreaming,
			MaxContextWindow:  128000, // Conservative default
			LastProbed:        time.Now(),
		}
		return fmt.Errorf("probing models: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		// Use defaults if probe fails
		p.capabilities = &Capabilities{
			SupportsTools:     !p.config.DisableToolChoice,
			SupportsStreaming: true,
			SupportsJSONMode:  !p.config.DisableJSONStreaming,
			MaxContextWindow:  128000,
			LastProbed:        time.Now(),
		}
		return fmt.Errorf("models endpoint returned %d", resp.StatusCode)
	}
	
	var modelsResp struct {
		Data []ModelInfo `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		// Use defaults if parsing fails
		p.capabilities = &Capabilities{
			SupportsTools:     !p.config.DisableToolChoice,
			SupportsStreaming: true,
			SupportsJSONMode:  !p.config.DisableJSONStreaming,
			MaxContextWindow:  128000,
			LastProbed:        time.Now(),
		}
		return fmt.Errorf("decoding models response: %w", err)
	}
	
	// Build capabilities from probe
	caps := &Capabilities{
		Models:            modelsResp.Data,
		SupportsTools:     !p.config.DisableToolChoice,
		SupportsStreaming: true,
		SupportsJSONMode:  !p.config.DisableJSONStreaming,
		LastProbed:        time.Now(),
	}
	
	// Detect capabilities from model names
	for _, model := range modelsResp.Data {
		// Check for vision models
		if strings.Contains(model.ID, "vision") || strings.Contains(model.ID, "gpt-4o") {
			caps.SupportsVision = true
		}
		// Update max context window
		if model.ContextWindow > caps.MaxContextWindow {
			caps.MaxContextWindow = model.ContextWindow
		}
	}
	
	// Set conservative defaults if not detected
	if caps.MaxContextWindow == 0 {
		caps.MaxContextWindow = 128000
	}
	
	p.capabilities = caps
	return nil
}

// setHeaders sets common headers for API requests.
func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	// Authentication
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}
	
	// Custom headers
	for k, v := range p.config.CustomHeaders {
		req.Header.Set(k, v)
	}
	
	// User agent
	req.Header.Set("User-Agent", "GAI/1.0 (OpenAI-Compatible)")
}

// doRequest performs an HTTP request with retry logic.
func (p *Provider) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}
	
	fullURL := p.baseURL.String() + endpoint
	
	var lastErr error
	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			delay := p.config.RetryDelay * time.Duration(1<<uint(attempt-1))
			jitter := time.Duration(float64(delay) * 0.1 * (0.5 - 0.5)) // Simplified jitter
			time.Sleep(delay + jitter)
			
			// Reset body reader for retry
			if body != nil {
				jsonBody, _ := json.Marshal(body)
				bodyReader = bytes.NewReader(jsonBody)
			}
		}
		
		req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		
		p.setHeaders(req)
		
		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http request: %w", err)
			continue
		}
		
		// Success or client error (don't retry client errors)
		if resp.StatusCode < 500 {
			return resp, nil
		}
		
		// Server error - read body for error details
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		// Check if it's a retryable error
		if resp.StatusCode == 503 || resp.StatusCode == 502 || resp.StatusCode == 504 {
			// For 503, check if this is the last attempt
			if attempt == p.config.MaxRetries {
				// Return the response so it can be properly mapped to an error
				resp.Body = io.NopCloser(bytes.NewReader(body))
				return resp, nil
			}
			lastErr = fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
			continue // Retry
		}
		
		// For 500 errors, check if the error message suggests retry
		if resp.StatusCode == 500 && strings.Contains(string(body), "temporarily") {
			if attempt == p.config.MaxRetries {
				resp.Body = io.NopCloser(bytes.NewReader(body))
				return resp, nil
			}
			lastErr = fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
			continue
		}
		
		// Non-retryable server error
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp, nil
	}
	
	return nil, fmt.Errorf("request failed after %d attempts: %w", p.config.MaxRetries+1, lastErr)
}

// getModel returns the model to use for a request.
func (p *Provider) getModel(req core.Request) string {
	if req.Model != "" {
		return req.Model
	}
	if p.config.DefaultModel != "" {
		return p.config.DefaultModel
	}
	// Fallback to a reasonable default
	return "gpt-3.5-turbo"
}

// GetCapabilities returns the provider's capabilities.
func (p *Provider) GetCapabilities() *Capabilities {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.capabilities
}