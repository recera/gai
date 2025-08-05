package gai

import (
	"context"
	"fmt"
	"os"

	"github.com/collinshill/gai/core"
	p "github.com/collinshill/gai/responseParser"

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

	return &client{
		openAI:    newOpenAIClient(openAIKey),
		anthropic: newAnthropicClient(anthropicKey),
		gemini:    newGeminiClient(geminiKey),
		groq:      newGroqClient(groqKey),
		cerebras:  newCerebrasClient(cerebrasKey),
	}, nil
}

// GetCompletion routes the request to the appropriate provider based on the LLMCallParts.
func (c *client) GetCompletion(ctx context.Context, parts LLMCallParts) (LLMResponse, error) {
	parts.System.CoalesceTextContent()
	for i := range parts.Messages {
		parts.Messages[i].CoalesceTextContent()
	}

	switch parts.Provider {
	case "openai":
		return c.openAI.GetCompletion(ctx, parts)
	case "anthropic":
		return c.anthropic.GetCompletion(ctx, parts)
	case "gemini":
		return c.gemini.GetCompletion(ctx, parts)
	case "groq":
		return c.groq.GetCompletion(ctx, parts)
	case "cerebras":
		return c.cerebras.GetCompletion(ctx, parts)
	default:
		return LLMResponse{}, fmt.Errorf("unsupported provider: %s", parts.Provider)
	}
}

func (c *client) GetResponseObject(ctx context.Context, parts LLMCallParts, v any) error {
	return p.GetResponseObject(ctx, c, v, parts)
}

