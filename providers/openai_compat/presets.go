// Package openai_compat provides preset configurations for known OpenAI-compatible providers.
package openai_compat

import (
	"os"
	"time"
)

// Groq creates a provider configured for Groq's API.
// Groq provides very fast inference with open-source models.
//
// Models available:
//   - llama-3.3-70b-versatile (default, most capable)
//   - llama-3.1-70b-versatile
//   - llama-3.1-8b-instant (fastest)
//   - mixtral-8x7b-32768
//   - gemma2-9b-it
//
// Example:
//
//	provider := openai_compat.Groq()
//	// or with custom model
//	provider := openai_compat.Groq(openai_compat.WithModel("llama-3.1-8b-instant"))
func Groq(opts ...Option) (*Provider, error) {
	config := CompatOpts{
		BaseURL:      "https://api.groq.com/openai/v1",
		APIKey:       os.Getenv("GROQ_API_KEY"),
		DefaultModel: "llama-3.3-70b-versatile",
		ProviderName: "groq",
		MaxRetries:   3,
		RetryDelay:   500 * time.Millisecond, // Groq is fast, short delays
		
		// Groq supports most OpenAI features
		DisableJSONStreaming:     false,
		DisableParallelToolCalls: false,
		DisableStrictJSONSchema:  false,
		DisableToolChoice:        false,
		
		// Custom headers for Groq
		CustomHeaders: map[string]string{
			"X-Provider": "groq",
		},
	}
	
	// Create provider
	provider, err := New(config)
	if err != nil {
		return nil, err
	}
	
	// Apply additional options
	for _, opt := range opts {
		opt(provider)
	}
	
	return provider, nil
}

// XAI creates a provider configured for xAI's API (Grok models).
// xAI provides access to the Grok family of models.
//
// Models available:
//   - grok-2-latest (default, most capable)
//   - grok-2-1212
//   - grok-beta
//
// Example:
//
//	provider := openai_compat.XAI()
func XAI(opts ...Option) (*Provider, error) {
	config := CompatOpts{
		BaseURL:      "https://api.x.ai/v1",
		APIKey:       os.Getenv("XAI_API_KEY"),
		DefaultModel: "grok-2-latest",
		ProviderName: "xai",
		MaxRetries:   3,
		RetryDelay:   time.Second,
		
		// xAI supports most OpenAI features
		DisableJSONStreaming:     false,
		DisableParallelToolCalls: false,
		DisableStrictJSONSchema:  false,
		DisableToolChoice:        false,
		
		CustomHeaders: map[string]string{
			"X-Provider": "xai",
		},
	}
	
	// Create provider
	provider, err := New(config)
	if err != nil {
		return nil, err
	}
	
	// Apply additional options
	for _, opt := range opts {
		opt(provider)
	}
	
	return provider, nil
}

// Cerebras creates a provider configured for Cerebras' API.
// Cerebras provides extremely fast inference with optimized hardware.
//
// Models available:
//   - llama-3.3-70b (default)
//   - llama-3.1-70b
//   - llama-3.1-8b
//
// Note: Cerebras has some limitations:
//   - JSON streaming is not supported
//   - Parallel tool calls are not supported
//   - Very fast but strict rate limits
//
// Example:
//
//	provider := openai_compat.Cerebras()
func Cerebras(opts ...Option) (*Provider, error) {
	config := CompatOpts{
		BaseURL:      "https://api.cerebras.ai/v1",
		APIKey:       os.Getenv("CEREBRAS_API_KEY"),
		DefaultModel: "llama-3.3-70b",
		ProviderName: "cerebras",
		MaxRetries:   3,
		RetryDelay:   2 * time.Second, // Cerebras has strict rate limits
		
		// Cerebras limitations
		DisableJSONStreaming:     true,  // Not supported
		DisableParallelToolCalls: true,  // Not supported
		DisableStrictJSONSchema:  false,
		DisableToolChoice:        false,
		
		CustomHeaders: map[string]string{
			"X-Provider": "cerebras",
		},
	}
	
	// Create provider
	provider, err := New(config)
	if err != nil {
		return nil, err
	}
	
	// Apply additional options
	for _, opt := range opts {
		opt(provider)
	}
	
	return provider, nil
}

// Baseten creates a provider configured for Baseten's API.
// Baseten allows you to deploy and serve your own models.
//
// The baseURL parameter should be your Baseten deployment URL.
// Model capabilities depend on what you've deployed.
//
// Example:
//
//	provider := openai_compat.Baseten(
//	    "https://model-abc123.api.baseten.co/v1",
//	    openai_compat.WithModel("llama-3-70b"),
//	)
func Baseten(baseURL string, opts ...Option) (*Provider, error) {
	config := CompatOpts{
		BaseURL:      baseURL,
		APIKey:       os.Getenv("BASETEN_API_KEY"),
		DefaultModel: "", // User should specify based on deployment
		ProviderName: "baseten",
		MaxRetries:   3,
		RetryDelay:   time.Second,
		
		// Baseten capabilities depend on the deployed model
		// Conservative defaults
		DisableJSONStreaming:     false,
		DisableParallelToolCalls: false,
		DisableStrictJSONSchema:  true, // Many models don't support strict mode
		DisableToolChoice:        false,
		
		CustomHeaders: map[string]string{
			"X-Provider": "baseten",
		},
	}
	
	// Create provider
	provider, err := New(config)
	if err != nil {
		return nil, err
	}
	
	// Apply additional options
	for _, opt := range opts {
		opt(provider)
	}
	
	return provider, nil
}

// Together creates a provider configured for Together.ai's API.
// Together provides access to a wide range of open-source models.
//
// Popular models:
//   - meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo
//   - meta-llama/Llama-3.3-70B-Instruct-Turbo (default)
//   - mistralai/Mixtral-8x7B-Instruct-v0.1
//   - NousResearch/Nous-Hermes-2-Mixtral-8x7B-DPO
//
// Example:
//
//	provider := openai_compat.Together()
func Together(opts ...Option) (*Provider, error) {
	config := CompatOpts{
		BaseURL:      "https://api.together.xyz/v1",
		APIKey:       os.Getenv("TOGETHER_API_KEY"),
		DefaultModel: "meta-llama/Llama-3.3-70B-Instruct-Turbo",
		ProviderName: "together",
		MaxRetries:   3,
		RetryDelay:   time.Second,
		
		// Together supports most OpenAI features
		DisableJSONStreaming:     false,
		DisableParallelToolCalls: false,
		DisableStrictJSONSchema:  true, // Most models don't support strict mode
		DisableToolChoice:        false,
		
		CustomHeaders: map[string]string{
			"X-Provider": "together",
		},
	}
	
	// Create provider
	provider, err := New(config)
	if err != nil {
		return nil, err
	}
	
	// Apply additional options
	for _, opt := range opts {
		opt(provider)
	}
	
	return provider, nil
}

// Fireworks creates a provider configured for Fireworks.ai's API.
// Fireworks provides fast inference for open-source models.
//
// Popular models:
//   - accounts/fireworks/models/llama-v3p3-70b-instruct (default)
//   - accounts/fireworks/models/llama-v3p1-8b-instruct
//   - accounts/fireworks/models/mixtral-8x7b-instruct
//
// Example:
//
//	provider := openai_compat.Fireworks()
func Fireworks(opts ...Option) (*Provider, error) {
	config := CompatOpts{
		BaseURL:      "https://api.fireworks.ai/inference/v1",
		APIKey:       os.Getenv("FIREWORKS_API_KEY"),
		DefaultModel: "accounts/fireworks/models/llama-v3p3-70b-instruct",
		ProviderName: "fireworks",
		MaxRetries:   3,
		RetryDelay:   time.Second,
		
		// Fireworks supports most OpenAI features
		DisableJSONStreaming:     false,
		DisableParallelToolCalls: false,
		DisableStrictJSONSchema:  true, // Most models don't support strict mode
		DisableToolChoice:        false,
		
		CustomHeaders: map[string]string{
			"X-Provider": "fireworks",
		},
	}
	
	// Create provider
	provider, err := New(config)
	if err != nil {
		return nil, err
	}
	
	// Apply additional options
	for _, opt := range opts {
		opt(provider)
	}
	
	return provider, nil
}

// Anyscale creates a provider configured for Anyscale's API.
// Anyscale provides scalable inference for open-source models.
//
// Popular models:
//   - meta-llama/Meta-Llama-3.1-70B-Instruct
//   - mistralai/Mixtral-8x7B-Instruct-v0.1
//
// Example:
//
//	provider := openai_compat.Anyscale()
func Anyscale(opts ...Option) (*Provider, error) {
	config := CompatOpts{
		BaseURL:      "https://api.endpoints.anyscale.com/v1",
		APIKey:       os.Getenv("ANYSCALE_API_KEY"),
		DefaultModel: "meta-llama/Meta-Llama-3.1-70B-Instruct",
		ProviderName: "anyscale",
		MaxRetries:   3,
		RetryDelay:   time.Second,
		
		// Anyscale supports most OpenAI features
		DisableJSONStreaming:     false,
		DisableParallelToolCalls: false,
		DisableStrictJSONSchema:  true, // Most models don't support strict mode
		DisableToolChoice:        false,
		
		CustomHeaders: map[string]string{
			"X-Provider": "anyscale",
		},
	}
	
	// Create provider
	provider, err := New(config)
	if err != nil {
		return nil, err
	}
	
	// Apply additional options
	for _, opt := range opts {
		opt(provider)
	}
	
	return provider, nil
}

// Option configures a provider.
type Option func(*Provider)

// WithModel sets the default model for the provider.
func WithModel(model string) Option {
	return func(p *Provider) {
		p.config.DefaultModel = model
	}
}

// WithAPIKey sets the API key for the provider.
func WithAPIKey(key string) Option {
	return func(p *Provider) {
		p.config.APIKey = key
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(n int) Option {
	return func(p *Provider) {
		p.config.MaxRetries = n
	}
}

// WithRetryDelay sets the base delay between retries.
func WithRetryDelay(d time.Duration) Option {
	return func(p *Provider) {
		p.config.RetryDelay = d
	}
}

// WithCustomHeader adds a custom header to all requests.
func WithCustomHeader(key, value string) Option {
	return func(p *Provider) {
		if p.config.CustomHeaders == nil {
			p.config.CustomHeaders = make(map[string]string)
		}
		p.config.CustomHeaders[key] = value
	}
}