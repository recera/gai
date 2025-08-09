package gai

import (
	"fmt"
	"os"
	"sync/atomic"

	"github.com/recera/gai/core"
)

// Type aliases for core types to maintain API compatibility
type LLMCallParts = core.LLMCallParts
type TraceInfo = core.TraceInfo
type Message = core.Message
type Content = core.Content
type TextContent = core.TextContent
type ImageContent = core.ImageContent
type LLMResponse = core.LLMResponse
type TokenUsage = core.TokenUsage
type ToolDefinition = core.ToolDefinition
type ToolCall = core.ToolCall
type StreamChunk = core.StreamChunk
type StreamHandler = core.StreamHandler

// NewLLMCallParts creates a new LLMCallParts with default values
func NewLLMCallParts() *LLMCallParts {
	provider := defaultProvider()
	model := defaultModel()
	return &LLMCallParts{
		Provider:    provider,
		Model:       model,
		System:      Message{Role: "system", Contents: []Content{}},
		Messages:    []Message{},
		MaxTokens:   1000,
		Temperature: 0.2,
	}
}

// AddFileAsContent reads a file and returns its content as TextContent
func AddFileAsContent(filePath string) TextContent {
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("failed to read file:", err)
		return TextContent{Text: "##ERROR: PROMPT FILE NOT READ##"}
	}
	return TextContent{Text: string(data)}
}

// Global defaults set by client creation. This preserves DX so NewLLMCallParts()
// can honor WithDefaultProvider/WithDefaultModel without forcing a client ref.
var (
	globalDefaultProvider atomic.Value // string
	globalDefaultModel    atomic.Value // string
)

func setGlobalDefaults(provider, model string) {
	if provider != "" {
		globalDefaultProvider.Store(provider)
	}
	if model != "" {
		globalDefaultModel.Store(model)
	}
}

func defaultProvider() string {
	if v := globalDefaultProvider.Load(); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func defaultModel() string {
	if v := globalDefaultModel.Load(); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return ""
}
