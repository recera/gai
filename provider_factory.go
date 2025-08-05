package gai

import (
	"github.com/recera/gai/providers"
)

// Provider factory functions to avoid circular imports

func newOpenAIClient(apiKey string) ProviderClient {
	return providers.NewOpenAIClient(apiKey)
}

func newAnthropicClient(apiKey string) ProviderClient {
	return providers.NewAnthropicClient(apiKey)
}

func newGeminiClient(apiKey string) ProviderClient {
	return providers.NewGeminiClient(apiKey)
}

func newGroqClient(apiKey string) ProviderClient {
	return providers.NewGroqClient(apiKey)
}

func newCerebrasClient(apiKey string) ProviderClient {
	return providers.NewCerebrasClient(apiKey)
}