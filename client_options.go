package gai

import (
	"time"
)

// ClientOption is a function that modifies client configuration
type ClientOption func(*clientOptions)

// clientOptions holds the configuration for the LLM client
type clientOptions struct {
	HTTPTimeout                time.Duration
	MaxRetries                 int
	BackoffInitial             time.Duration
	BackoffMax                 time.Duration
	BackoffJitter              float64
	OpenAIKey                  string
	AnthropicKey               string
	GeminiKey                  string
	GroqKey                    string
	CerebrasKey                string
	EnvFilePath                string
	DisableEnvLoader           bool
	DefaultProvider            string
	DefaultModel               string
	UserAgent                  string
	OpenAIBaseURL              string
	AnthropicBaseURL           string
	GeminiBaseURL              string
	GroqBaseURL                string
	CerebrasBaseURL            string
	OpenAIIncludeUsageInStream bool
	ToolLoopMaxSteps           int
}

// WithHTTPTimeout sets the HTTP timeout for API calls
func WithHTTPTimeout(d time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.HTTPTimeout = d
	}
}

// WithMaxRetries sets the maximum number of retries for API calls
func WithMaxRetries(n int) ClientOption {
	return func(o *clientOptions) {
		o.MaxRetries = n
	}
}

// WithBackoff configures exponential backoff parameters used during retries.
func WithBackoff(initial, max time.Duration, jitter float64) ClientOption {
	return func(o *clientOptions) {
		o.BackoffInitial = initial
		o.BackoffMax = max
		o.BackoffJitter = jitter
	}
}

// WithOpenAIKey sets the OpenAI API key
func WithOpenAIKey(key string) ClientOption {
	return func(o *clientOptions) {
		o.OpenAIKey = key
	}
}

// WithAnthropicKey sets the Anthropic API key
func WithAnthropicKey(key string) ClientOption {
	return func(o *clientOptions) {
		o.AnthropicKey = key
	}
}

// WithGeminiKey sets the Gemini API key
func WithGeminiKey(key string) ClientOption {
	return func(o *clientOptions) {
		o.GeminiKey = key
	}
}

// WithGroqKey sets the Groq API key
func WithGroqKey(key string) ClientOption {
	return func(o *clientOptions) {
		o.GroqKey = key
	}
}

// WithCerebrasKey sets the Cerebras API key
func WithCerebrasKey(key string) ClientOption {
	return func(o *clientOptions) {
		o.CerebrasKey = key
	}
}

// WithEnvFile sets a custom path for the .env file
func WithEnvFile(path string) ClientOption {
	return func(o *clientOptions) {
		o.EnvFilePath = path
	}
}

// WithoutEnvFile disables loading of environment variables from .env files
func WithoutEnvFile() ClientOption {
	return func(o *clientOptions) {
		o.DisableEnvLoader = true
	}
}

// WithDefaultProvider sets the default provider for new LLMCallParts
func WithDefaultProvider(provider string) ClientOption {
	return func(o *clientOptions) {
		o.DefaultProvider = provider
	}
}

// WithDefaultModel sets the default model for new LLMCallParts
func WithDefaultModel(model string) ClientOption {
	return func(o *clientOptions) {
		o.DefaultModel = model
	}
}

func WithUserAgent(ua string) ClientOption { return func(o *clientOptions) { o.UserAgent = ua } }

func WithProviderBaseURL(provider, baseURL string) ClientOption {
	return func(o *clientOptions) {
		switch provider {
		case "openai":
			o.OpenAIBaseURL = baseURL
		case "anthropic":
			o.AnthropicBaseURL = baseURL
		case "gemini":
			o.GeminiBaseURL = baseURL
		case "groq":
			o.GroqBaseURL = baseURL
		case "cerebras":
			o.CerebrasBaseURL = baseURL
		}
	}
}

func WithOpenAIIncludeUsageInStream(enable bool) ClientOption {
	return func(o *clientOptions) { o.OpenAIIncludeUsageInStream = enable }
}

func WithToolLoopMaxSteps(n int) ClientOption {
	return func(o *clientOptions) { o.ToolLoopMaxSteps = n }
}

// getDefaultOptions returns the default client options
func getDefaultOptions() clientOptions {
	return clientOptions{
		HTTPTimeout:      30 * time.Second,
		MaxRetries:       3,
		BackoffInitial:   200 * time.Millisecond,
		BackoffMax:       5 * time.Second,
		BackoffJitter:    0.2,
		DefaultProvider:  "",
		DefaultModel:     "",
		UserAgent:        "gai/0.1 (+github.com/recera/gai)",
		ToolLoopMaxSteps: 8,
	}
}
