package gai

import (
	"fmt"
	"os"

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

// NewLLMCallParts creates a new LLMCallParts with default values
func NewLLMCallParts() *LLMCallParts {
	return &LLMCallParts{
		Provider:    "cerebras",
		Model:       "llama-3.3-70b",
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