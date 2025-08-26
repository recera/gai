// Package groq implements a native Groq provider for the GAI framework.
// It provides optimized support for Groq's LPU inference engine with proper
// tool calling, model-specific constraints, and high-performance features.
package groq

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/recera/gai/core"
)

const (
	defaultBaseURL = "https://api.groq.com/openai/v1"
	defaultTimeout = 30 * time.Second // Groq is fast, shorter timeout
	defaultModel   = "llama-3.3-70b-versatile"
)

// Provider implements the core.Provider interface for Groq.
type Provider struct {
	apiKey         string
	baseURL        string
	defaultModel   string
	client         *http.Client
	maxRetries     int
	retryDelay     time.Duration
	collector      core.MetricsCollector
	customHeaders  map[string]string
	serviceTier    string // "on_demand" or "flex"
	mu             sync.RWMutex
}

// Option configures the Groq provider.
type Option func(*Provider)

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(p *Provider) {
		p.apiKey = key
	}
}

// WithBaseURL sets a custom base URL (for testing or proxies).
func WithBaseURL(url string) Option {
	return func(p *Provider) {
		p.baseURL = url
	}
}

// WithModel sets the default model to use.
func WithModel(model string) Option {
	return func(p *Provider) {
		p.defaultModel = model
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

// WithCustomHeaders sets custom headers for requests.
func WithCustomHeaders(headers map[string]string) Option {
	return func(p *Provider) {
		if p.customHeaders == nil {
			p.customHeaders = make(map[string]string)
		}
		for k, v := range headers {
			p.customHeaders[k] = v
		}
	}
}

// WithServiceTier sets the service tier ("on_demand" or "flex").
func WithServiceTier(tier string) Option {
	return func(p *Provider) {
		p.serviceTier = tier
	}
}

// New creates a new Groq provider with the given options.
func New(opts ...Option) *Provider {
	p := &Provider{
		baseURL:      defaultBaseURL,
		defaultModel: defaultModel,
		maxRetries:   2, // Groq is usually reliable, fewer retries needed
		retryDelay:   200 * time.Millisecond, // Fast retries due to high speed
		serviceTier:  "on_demand", // Default tier
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.client == nil {
		p.client = &http.Client{
			Timeout: defaultTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     60 * time.Second,
				DisableCompression:  true, // Groq optimizes for speed
			},
		}
	}

	return p
}

// chatCompletionRequest represents the request structure for Groq's Chat Completions API.
type chatCompletionRequest struct {
	Model               string            `json:"model"`
	Messages            []chatMessage     `json:"messages"`
	Temperature         *float32          `json:"temperature,omitempty"`
	MaxTokens           *int              `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int              `json:"max_completion_tokens,omitempty"`
	Tools               []chatTool        `json:"tools,omitempty"`
	ToolChoice          interface{}       `json:"tool_choice,omitempty"`
	Stream              bool              `json:"stream,omitempty"`
	ResponseFormat      *responseFormat   `json:"response_format,omitempty"`
	StreamOptions       *streamOptions    `json:"stream_options,omitempty"`
	ParallelToolCalls   *bool             `json:"parallel_tool_calls,omitempty"`
	N                   int               `json:"n"` // Only n=1 supported
	Stop                []string          `json:"stop,omitempty"`
	PresencePenalty     *float32          `json:"presence_penalty,omitempty"`
	FrequencyPenalty    *float32          `json:"frequency_penalty,omitempty"`
	LogitBias           map[string]float32 `json:"logit_bias,omitempty"`
	User                string            `json:"user,omitempty"`
	Seed                *int              `json:"seed,omitempty"`
	TopP                *float32          `json:"top_p,omitempty"`
	TopK                *int              `json:"top_k,omitempty"`
	
	// Groq-specific parameters
	ServiceTier         *string           `json:"service_tier,omitempty"`
	StructuredOutputs   *bool             `json:"structured_outputs,omitempty"`
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
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
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
	Strict      *bool           `json:"strict,omitempty"`
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
	Type       string            `json:"type"` // "text", "json_object", "json_schema"
	JSONSchema *jsonSchemaFormat `json:"json_schema,omitempty"`
}

// jsonSchemaFormat specifies a JSON schema for structured outputs.
type jsonSchemaFormat struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema"`
	Strict      *bool           `json:"strict,omitempty"`
}

// streamOptions configures streaming behavior.
type streamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// ModelInfo contains model-specific information and constraints.
type ModelInfo struct {
	ID                   string   `json:"id"`
	OwnedBy              string   `json:"owned_by"`
	ContextWindow        int      `json:"context_window"`
	MaxCompletionTokens  int      `json:"max_completion_tokens"`
	SupportsVision       bool     `json:"supports_vision"`
	SupportsTools        bool     `json:"supports_tools"`
	SupportsJSON         bool     `json:"supports_json"`
	SupportsStreaming    bool     `json:"supports_streaming"`
	RecommendedFor       []string `json:"recommended_for"`
	PerformanceClass     string   `json:"performance_class"` // "ultra-fast", "fast", "balanced"
	IsDeprecated         bool     `json:"is_deprecated"`
}

// getModelInfo returns detailed information about a specific model.
func (p *Provider) getModelInfo(model string) ModelInfo {
	// Comprehensive model database based on Groq's 2025 catalog
	modelDB := map[string]ModelInfo{
		// Featured OpenAI Models
		"openai/gpt-oss-20b": {
			ID: "openai/gpt-oss-20b", OwnedBy: "OpenAI", ContextWindow: 131072, MaxCompletionTokens: 65536,
			SupportsVision: false, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"reasoning", "code", "search"}, PerformanceClass: "ultra-fast",
		},
		"openai/gpt-oss-120b": {
			ID: "openai/gpt-oss-120b", OwnedBy: "OpenAI", ContextWindow: 131072, MaxCompletionTokens: 65536,
			SupportsVision: false, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"reasoning", "code", "search"}, PerformanceClass: "fast",
		},
		
		// Meta Llama Models - Production
		"llama-3.3-70b-versatile": {
			ID: "llama-3.3-70b-versatile", OwnedBy: "Meta", ContextWindow: 131072, MaxCompletionTokens: 32768,
			SupportsVision: false, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"general", "reasoning", "code", "tools"}, PerformanceClass: "fast",
		},
		"llama-3.1-8b-instant": {
			ID: "llama-3.1-8b-instant", OwnedBy: "Meta", ContextWindow: 131072, MaxCompletionTokens: 131072,
			SupportsVision: false, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"general", "chat", "speed"}, PerformanceClass: "ultra-fast",
		},
		"llama3-8b-8192": {
			ID: "llama3-8b-8192", OwnedBy: "Meta", ContextWindow: 8192, MaxCompletionTokens: 8192,
			SupportsVision: false, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"general", "chat"}, PerformanceClass: "ultra-fast", IsDeprecated: true,
		},
		"llama3-70b-8192": {
			ID: "llama3-70b-8192", OwnedBy: "Meta", ContextWindow: 8192, MaxCompletionTokens: 8192,
			SupportsVision: false, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"general", "reasoning"}, PerformanceClass: "fast", IsDeprecated: true,
		},
		
		// Vision Models - Llama 4 Series
		"meta-llama/llama-4-scout-17b-16e-instruct": {
			ID: "meta-llama/llama-4-scout-17b-16e-instruct", OwnedBy: "Meta", ContextWindow: 131072, MaxCompletionTokens: 8192,
			SupportsVision: true, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"vision", "multimodal", "analysis"}, PerformanceClass: "fast",
		},
		"meta-llama/llama-4-maverick-17b-128e-instruct": {
			ID: "meta-llama/llama-4-maverick-17b-128e-instruct", OwnedBy: "Meta", ContextWindow: 131072, MaxCompletionTokens: 8192,
			SupportsVision: true, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"vision", "multimodal", "complex-analysis"}, PerformanceClass: "fast",
		},
		
		// Reasoning Models
		"deepseek-r1-distill-llama-70b": {
			ID: "deepseek-r1-distill-llama-70b", OwnedBy: "DeepSeek / Meta", ContextWindow: 131072, MaxCompletionTokens: 131072,
			SupportsVision: false, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"reasoning", "mathematics", "complex-problems"}, PerformanceClass: "balanced",
		},
		"moonshotai/kimi-k2-instruct": {
			ID: "moonshotai/kimi-k2-instruct", OwnedBy: "Moonshot AI", ContextWindow: 131072, MaxCompletionTokens: 16384,
			SupportsVision: false, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"reasoning", "tools", "agents"}, PerformanceClass: "ultra-fast",
		},
		"qwen/qwen3-32b": {
			ID: "qwen/qwen3-32b", OwnedBy: "Alibaba Cloud", ContextWindow: 131072, MaxCompletionTokens: 40960,
			SupportsVision: false, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"reasoning", "multilingual", "general"}, PerformanceClass: "fast",
		},
		
		// Google Models
		"gemma2-9b-it": {
			ID: "gemma2-9b-it", OwnedBy: "Google", ContextWindow: 8192, MaxCompletionTokens: 8192,
			SupportsVision: false, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"general", "chat", "instruction-following"}, PerformanceClass: "fast",
		},
		
		// Audio Models
		"whisper-large-v3": {
			ID: "whisper-large-v3", OwnedBy: "OpenAI", ContextWindow: 448, MaxCompletionTokens: 448,
			SupportsVision: false, SupportsTools: false, SupportsJSON: false, SupportsStreaming: false,
			RecommendedFor: []string{"speech-to-text", "transcription", "translation"}, PerformanceClass: "fast",
		},
		"whisper-large-v3-turbo": {
			ID: "whisper-large-v3-turbo", OwnedBy: "OpenAI", ContextWindow: 448, MaxCompletionTokens: 448,
			SupportsVision: false, SupportsTools: false, SupportsJSON: false, SupportsStreaming: false,
			RecommendedFor: []string{"speech-to-text", "transcription", "fast-audio"}, PerformanceClass: "ultra-fast",
		},
		
		// Groq Systems
		"compound-beta": {
			ID: "compound-beta", OwnedBy: "Groq", ContextWindow: 131072, MaxCompletionTokens: 8192,
			SupportsVision: false, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"experimental", "research", "beta-testing"}, PerformanceClass: "balanced",
		},
		"compound-beta-mini": {
			ID: "compound-beta-mini", OwnedBy: "Groq", ContextWindow: 131072, MaxCompletionTokens: 8192,
			SupportsVision: false, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"experimental", "fast-inference", "beta-testing"}, PerformanceClass: "ultra-fast",
		},
		
		// Guard Models
		"meta-llama/llama-guard-4-12b": {
			ID: "meta-llama/llama-guard-4-12b", OwnedBy: "Meta", ContextWindow: 131072, MaxCompletionTokens: 1024,
			SupportsVision: false, SupportsTools: false, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"safety", "content-moderation", "filtering"}, PerformanceClass: "fast",
		},
		"meta-llama/llama-prompt-guard-2-22m": {
			ID: "meta-llama/llama-prompt-guard-2-22m", OwnedBy: "Meta", ContextWindow: 512, MaxCompletionTokens: 512,
			SupportsVision: false, SupportsTools: false, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"prompt-injection-detection", "security"}, PerformanceClass: "ultra-fast",
		},
		"meta-llama/llama-prompt-guard-2-86m": {
			ID: "meta-llama/llama-prompt-guard-2-86m", OwnedBy: "Meta", ContextWindow: 512, MaxCompletionTokens: 512,
			SupportsVision: false, SupportsTools: false, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"prompt-injection-detection", "security"}, PerformanceClass: "ultra-fast",
		},
		
		// Other Models
		"allam-2-7b": {
			ID: "allam-2-7b", OwnedBy: "SDAIA", ContextWindow: 4096, MaxCompletionTokens: 4096,
			SupportsVision: false, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
			RecommendedFor: []string{"arabic", "multilingual", "regional"}, PerformanceClass: "fast",
		},
		
		// Text-to-Speech Models
		"playai-tts": {
			ID: "playai-tts", OwnedBy: "PlayAI", ContextWindow: 8192, MaxCompletionTokens: 8192,
			SupportsVision: false, SupportsTools: false, SupportsJSON: false, SupportsStreaming: true,
			RecommendedFor: []string{"text-to-speech", "voice-synthesis"}, PerformanceClass: "fast",
		},
		"playai-tts-arabic": {
			ID: "playai-tts-arabic", OwnedBy: "PlayAI", ContextWindow: 8192, MaxCompletionTokens: 8192,
			SupportsVision: false, SupportsTools: false, SupportsJSON: false, SupportsStreaming: true,
			RecommendedFor: []string{"text-to-speech", "arabic", "voice-synthesis"}, PerformanceClass: "fast",
		},
	}
	
	if info, exists := modelDB[model]; exists {
		return info
	}
	
	// Return conservative defaults for unknown models
	return ModelInfo{
		ID: model, OwnedBy: "Unknown", ContextWindow: 8192, MaxCompletionTokens: 8192,
		SupportsVision: false, SupportsTools: true, SupportsJSON: true, SupportsStreaming: true,
		RecommendedFor: []string{"general"}, PerformanceClass: "balanced",
	}
}

// isAudioModel determines if a model is for audio processing.
func (p *Provider) isAudioModel(model string) bool {
	return strings.Contains(model, "whisper") || strings.Contains(model, "tts")
}

// isVisionModel determines if a model supports vision/multimodal inputs.
func (p *Provider) isVisionModel(model string) bool {
	modelInfo := p.getModelInfo(model)
	return modelInfo.SupportsVision
}

// isReasoningModel determines if a model is optimized for reasoning tasks.
func (p *Provider) isReasoningModel(model string) bool {
	reasoningModels := []string{
		"deepseek-r1",
		"moonshotai/kimi-k2",
		"qwen/qwen3",
		"openai/gpt-oss",
	}
	
	for _, prefix := range reasoningModels {
		if strings.Contains(model, prefix) {
			return true
		}
	}
	return false
}

// getModel returns the model to use for the request.
func (p *Provider) getModel(req core.Request) string {
	if req.Model != "" {
		return req.Model
	}
	return p.defaultModel
}