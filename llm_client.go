package gai

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/recera/gai/core"
	p "github.com/recera/gai/responseParser"

	"github.com/joho/godotenv"
)

// Type aliases for interfaces
type LLMClient = core.LLMClient
type ProviderClient = core.ProviderClient

// client is the concrete implementation of the LLMClient interface.
type client struct {
	openAI    ProviderClient
	anthropic ProviderClient
	gemini    ProviderClient
	groq      ProviderClient
	cerebras  ProviderClient
	cfg       clientOptions
}

// NewClient creates and initializes a new LLM client.
// By default, it reads API keys from environment variables.
// Use WithEnvFile to load from a specific .env file, or pass keys directly with WithOpenAIKey, etc.
func NewClient(opts ...ClientOption) (LLMClient, error) {
	// Apply default options
	cfg := getDefaultOptions()

	// Apply provided options
	for _, opt := range opts {
		opt(&cfg)
	}

	// Load environment variables from .env file if specified
	if !cfg.DisableEnvLoader && cfg.EnvFilePath != "" {
		if err := godotenv.Load(cfg.EnvFilePath); err != nil {
			return nil, fmt.Errorf("failed to load .env file from %s: %w", cfg.EnvFilePath, err)
		}
	}

	// Use provided keys or fall back to environment variables
	openAIKey := cfg.OpenAIKey
	if openAIKey == "" {
		openAIKey = os.Getenv("OPENAI_API_KEY")
	}

	anthropicKey := cfg.AnthropicKey
	if anthropicKey == "" {
		anthropicKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	geminiKey := cfg.GeminiKey
	if geminiKey == "" {
		geminiKey = os.Getenv("GEMINI_API_KEY")
	}

	groqKey := cfg.GroqKey
	if groqKey == "" {
		groqKey = os.Getenv("GROQ_API_KEY")
	}

	cerebrasKey := cfg.CerebrasKey
	if cerebrasKey == "" {
		cerebrasKey = os.Getenv("CEREBRAS_API_KEY")
	}

	cl := &client{
		openAI:    newOpenAIClient(openAIKey),
		anthropic: newAnthropicClient(anthropicKey),
		gemini:    newGeminiClient(geminiKey),
		groq:      newGroqClient(groqKey),
		cerebras:  newCerebrasClient(cerebrasKey),
		cfg:       cfg,
	}

	// Set global defaults for subsequent NewLLMCallParts()
	setGlobalDefaults(cfg.DefaultProvider, cfg.DefaultModel)

	return cl, nil
}

// GetCompletion routes the request to the appropriate provider based on the LLMCallParts.
func (c *client) GetCompletion(ctx context.Context, parts LLMCallParts) (LLMResponse, error) {
	parts.System.CoalesceTextContent()
	for i := range parts.Messages {
		parts.Messages[i].CoalesceTextContent()
	}

	// Emit initial trace if provided
	if parts.Trace != nil {
		parts.Trace(core.TraceInfo{Provider: parts.Provider, Model: parts.Model})
	}

	// Minimal retry loop honoring MaxRetries option if available
	attempts := 1
	if c.cfg.MaxRetries > 1 {
		attempts = c.cfg.MaxRetries
	}

	var resp LLMResponse
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		start := time.Now()
		switch parts.Provider {
		case "openai":
			resp, err = c.openAI.GetCompletion(ctx, parts)
		case "anthropic":
			resp, err = c.anthropic.GetCompletion(ctx, parts)
		case "gemini":
			resp, err = c.gemini.GetCompletion(ctx, parts)
		case "groq":
			resp, err = c.groq.GetCompletion(ctx, parts)
		case "cerebras":
			resp, err = c.cerebras.GetCompletion(ctx, parts)
		default:
			return LLMResponse{}, fmt.Errorf("unsupported provider: %s", parts.Provider)
		}
		if parts.Trace != nil {
			parts.Trace(core.TraceInfo{
				Provider:    parts.Provider,
				Model:       parts.Model,
				Attempt:     attempt,
				RawResponse: resp.Content,
				Elapsed:     time.Since(start),
			})
		}
		if err == nil {
			return resp, nil
		}
	}
	return resp, err
}

func (c *client) GetResponseObject(ctx context.Context, parts LLMCallParts, v any) error {
	return p.GetResponseObject(ctx, c, v, parts)
}

// StreamCompletion routes a streaming request to the appropriate provider
func (c *client) StreamCompletion(ctx context.Context, parts LLMCallParts, handler core.StreamHandler) error {
	parts.System.CoalesceTextContent()
	for i := range parts.Messages {
		parts.Messages[i].CoalesceTextContent()
	}

	switch parts.Provider {
	case "openai":
		return c.openAI.StreamCompletion(ctx, parts, handler)
	case "anthropic":
		return c.anthropic.StreamCompletion(ctx, parts, handler)
	case "gemini":
		return c.gemini.StreamCompletion(ctx, parts, handler)
	case "groq":
		return c.groq.StreamCompletion(ctx, parts, handler)
	case "cerebras":
		return c.cerebras.StreamCompletion(ctx, parts, handler)
	default:
		return fmt.Errorf("unsupported provider: %s", parts.Provider)
	}
}

// RunWithTools performs a tool-calling loop using provider-native tool calls when available.
func (c *client) RunWithTools(ctx context.Context, parts LLMCallParts, executor func(call core.ToolCall) (string, error)) (core.LLMResponse, error) {
	// Ensure tools are present
	if len(parts.Tools) == 0 {
		return core.LLMResponse{}, fmt.Errorf("no tools configured on LLMCallParts")
	}

	// We'll loop up to a small cap to avoid infinite recursion
	for step := 0; step < 8; step++ {
		resp, err := c.GetCompletion(ctx, parts)
		if err != nil {
			return resp, err
		}

		if len(resp.ToolCalls) == 0 {
			// Done: model returned final content
			return resp, nil
		}

		// Execute each tool call and append a tool response message in a provider-agnostic way
		for _, call := range resp.ToolCalls {
			out, err := executor(call)
			if err != nil {
				// Append failure result to the conversation to let model recover
				toolMsg := core.Message{Role: "tool"}
				toolMsg.AddTextContent(fmt.Sprintf("TOOL_RESPONSE_ERROR:%s:%v", call.Name, err))
				parts.AddMessage(toolMsg)
				continue
			}
			toolMsg := core.Message{Role: "tool"}
			toolMsg.AddTextContent(fmt.Sprintf("TOOL_RESPONSE:%s:%s", call.Name, out))
			parts.AddMessage(toolMsg)
		}
	}
	return core.LLMResponse{}, fmt.Errorf("tool loop exceeded max steps")
}

// Convenience helpers to accept pointer-based fluent builders without forcing callers
// to dereference. These improve ergonomics so users can stay entirely in gai.
func GetCompletionP(ctx context.Context, c LLMClient, parts *LLMCallParts) (LLMResponse, error) {
	if parts == nil {
		return LLMResponse{}, fmt.Errorf("nil parts")
	}
	return c.GetCompletion(ctx, parts.Value())
}

func GetResponseObjectP(ctx context.Context, c LLMClient, parts *LLMCallParts, v any) error {
	if parts == nil {
		return fmt.Errorf("nil parts")
	}
	return c.GetResponseObject(ctx, parts.Value(), v)
}
