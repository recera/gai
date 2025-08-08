package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/recera/gai/core"
)

type groqClient struct {
	apiKey string
	client *http.Client
}

// NewGroqClient creates a new client for the Groq API.
func NewGroqClient(apiKey string) core.ProviderClient {
	return &groqClient{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

// groq response structs (follows OpenAI format)
type groqResponse struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int64           `json:"created"`
	Model   string          `json:"model"`
	Choices []groqChoice    `json:"choices"`
	Usage   groqUsageTokens `json:"usage"`
}

type groqChoice struct {
	Index        int         `json:"index"`
	Message      groqMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type groqUsageTokens struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (c *groqClient) GetCompletion(ctx context.Context, parts core.LLMCallParts) (core.LLMResponse, error) {
	emptyResponse := core.LLMResponse{}
	if c.apiKey == "" {
		return emptyResponse, core.NewLLMError(fmt.Errorf("API key is not set"), "groq", parts.Model)
	}

	transformed := c.transformMessagesWithSystem(parts.Messages, parts.System)
	reqBody := groqRequest{
		Model:       parts.Model,
		Messages:    transformed,
		MaxTokens:   parts.MaxTokens,
		Temperature: parts.Temperature,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error marshalling request: %w", err), "groq", parts.Model)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(reqBytes))
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error creating request: %w", err), "groq", parts.Model)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error sending request: %w", err), "groq", parts.Model)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error reading response body: %w", err), "groq", parts.Model)
	}

	if resp.StatusCode != http.StatusOK {
		err := core.NewLLMError(fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes)), "groq", parts.Model)
		err.StatusCode = resp.StatusCode
		err.LastRaw = string(bodyBytes)
		return emptyResponse, err
	}

	// Parse the response into our provider-specific struct
	var apiResponse groqResponse
	if err := json.Unmarshal(bodyBytes, &apiResponse); err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error unmarshalling response: %w", err), "groq", parts.Model)
	}

	// Check if we got any choices back
	if len(apiResponse.Choices) == 0 {
		return emptyResponse, core.NewLLMError(fmt.Errorf("response contained no choices"), "groq", parts.Model)
	}

	// Map to the unified LLMResponse
	unifiedResponse := core.LLMResponse{
		Content:      apiResponse.Choices[0].Message.Content,
		FinishReason: apiResponse.Choices[0].FinishReason,
		Usage: core.TokenUsage{
			PromptTokens:     apiResponse.Usage.PromptTokens,
			CompletionTokens: apiResponse.Usage.CompletionTokens,
			TotalTokens:      apiResponse.Usage.TotalTokens,
		},
	}

	return unifiedResponse, nil
}

func (c *groqClient) transformMessages(messages []core.Message) []groqMessage {
	var groqMessages []groqMessage
	for _, msg := range messages {
		var contentStr string
		for _, content := range msg.Contents {
			if textContent, ok := content.(core.TextContent); ok {
				contentStr += textContent.Text
			}
		}
		groqMessages = append(groqMessages, groqMessage{
			Role:    msg.Role,
			Content: contentStr,
		})
	}
	return groqMessages
}

func (c *groqClient) transformMessagesWithSystem(messages []core.Message, system core.Message) []groqMessage {
	result := make([]groqMessage, 0, len(messages)+1)
	if len(system.Contents) > 0 {
		var sys string
		for _, content := range system.Contents {
			if textContent, ok := content.(core.TextContent); ok {
				sys += textContent.Text
			}
		}
		if sys != "" {
			result = append(result, groqMessage{Role: "system", Content: sys})
		}
	}
	return append(result, c.transformMessages(messages)...)
}

// StreamCompletion uses the OpenAI-compatible streaming endpoint if available; here we emulate with one-shot
func (c *groqClient) StreamCompletion(ctx context.Context, parts core.LLMCallParts, handler core.StreamHandler) error {
	resp, err := c.GetCompletion(ctx, parts)
	if err != nil {
		return err
	}
	if resp.Content != "" {
		if err := handler(core.StreamChunk{Type: "content", Delta: resp.Content}); err != nil {
			return err
		}
	}
	return handler(core.StreamChunk{Type: "end", FinishReason: resp.FinishReason})
}
