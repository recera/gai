package gai

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/recera/gai/core"
	p "github.com/recera/gai/responseParser"

	"strings"

	"github.com/joho/godotenv"
	"github.com/recera/gai/registry"
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
	reg       *registry.Registry
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

	reg := registry.New()
	cl := &client{
		openAI:    newOpenAIClient(openAIKey),
		anthropic: newAnthropicClient(anthropicKey),
		gemini:    newGeminiClient(geminiKey),
		groq:      newGroqClient(groqKey),
		cerebras:  newCerebrasClient(cerebrasKey),
		cfg:       cfg,
		reg:       reg,
	}
	// Register providers for model key resolution
	reg.Register("openai", cl.openAI)
	reg.Register("anthropic", cl.anthropic)
	reg.Register("gemini", cl.gemini)
	reg.Register("groq", cl.groq)
	reg.Register("cerebras", cl.cerebras)

	// Set global defaults for subsequent NewLLMCallParts()
	setGlobalDefaults(cfg.DefaultProvider, cfg.DefaultModel)

	return cl, nil
}

// ApplyModelKey parses "provider:model" and sets parts.Provider and parts.Model accordingly using the internal registry.
func (c *client) ApplyModelKey(parts *LLMCallParts, key string) error {
	if parts == nil {
		return fmt.Errorf("nil parts")
	}
	_, model, err := c.reg.Resolve(key)
	if err != nil {
		return err
	}
	pv := strings.SplitN(key, ":", 2)[0]
	parts.Provider = pv
	parts.Model = model
	return nil
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

// StreamWithTools orchestrates a streaming call that can pause for tool calls,
// execute them, append tool results, and resume generation until finish.
func (c *client) StreamWithTools(ctx context.Context, parts LLMCallParts, executor func(call core.ToolCall) (string, error), handler core.StreamHandler) error {
	if len(parts.Tools) == 0 {
		return fmt.Errorf("no tools configured on LLMCallParts")
	}

	// Internal buffer to capture emitted tool_call parts and then inject tool results
	toolCallHappened := false
	// Wrap user handler to detect tool_call and forward other parts
	wrapped := func(ch core.StreamChunk) error {
		switch ch.Type {
		case "tool_call":
			toolCallHappened = true
			if ch.Call != nil {
				out, err := executor(*ch.Call)
				// Append tool result message before resuming
				m := core.Message{Role: "tool"}
				m.AddTextContent(out)
				m.ToolCallID = ch.Call.ID
				if parts.Provider == "gemini" {
					m.ToolName = ch.Call.Name
				}
				parts.AddMessage(m)
				// After injecting result, we will resume by recursively streaming again.
				if err != nil {
					// Still continue to let model recover
				}
			}
			return nil
		default:
			return handler(ch)
		}
	}

	// Drive streaming; if a tool_call happens, we recursively resume until finish
	for step := 0; step < 8; step++ {
		toolCallHappened = false
		if err := c.StreamCompletion(ctx, parts, wrapped); err != nil {
			return err
		}
		if !toolCallHappened {
			return nil
		}
		// toolCall handled and tool result appended; loop to continue generation
	}
	return fmt.Errorf("tool streaming loop exceeded max steps")
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

		// Execute each tool call and append a provider-native tool response message
		for _, call := range resp.ToolCalls {
			out, err := executor(call)
			if err != nil {
				// On failure, still append a tool message with error text
				toolMsg := core.Message{Role: "tool"}
				toolMsg.ToolCallID = call.ID
				toolMsg.AddTextContent(fmt.Sprintf("error: %v", err))
				parts.AddMessage(toolMsg)
				continue
			}
			// Map to provider-native reply shape by using Role: "tool" and setting ToolCallID where supported
			toolMsg := core.Message{Role: "tool"}
			toolMsg.ToolCallID = call.ID
			// For providers like Gemini, set ToolName for functionResponse mapping
			if parts.Provider == "gemini" {
				toolMsg.ToolName = call.Name
			}
			toolMsg.AddTextContent(out)
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
