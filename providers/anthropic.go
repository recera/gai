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

type anthropicClient struct {
	apiKey string
	client *http.Client
}

// NewAnthropicClient creates a new client for the Anthropic API.
func NewAnthropicClient(apiKey string) core.ProviderClient {
	return &anthropicClient{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

// anthropic response structs
type anthropicResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Content    []anthropicContent `json:"content"`
	StopReason string             `json:"stop_reason"`
	Usage      anthropicUsage     `json:"usage"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func (c *anthropicClient) GetCompletion(ctx context.Context, parts core.LLMCallParts) (core.LLMResponse, error) {
	emptyResponse := core.LLMResponse{}
	if c.apiKey == "" {
		return emptyResponse, core.NewLLMError(fmt.Errorf("API key is not set"), "anthropic", parts.Model)
	}

	reqBody := anthropicRequest{
		Model:       parts.Model,
		Messages:    c.transformMessages(parts.Messages),
		MaxTokens:   parts.MaxTokens,
		Temperature: parts.Temperature,
	}

	if len(parts.System.Contents) > 0 {
		if textContent, ok := parts.System.Contents[0].(core.TextContent); ok {
			reqBody.System = textContent.Text
		}
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error marshalling request: %w", err), "anthropic", parts.Model)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBytes))
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error creating request: %w", err), "anthropic", parts.Model)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error sending request: %w", err), "anthropic", parts.Model)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error reading response body: %w", err), "anthropic", parts.Model)
	}

	if resp.StatusCode != http.StatusOK {
		err := core.NewLLMError(fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes)), "anthropic", parts.Model)
		err.StatusCode = resp.StatusCode
		err.LastRaw = string(bodyBytes)
		return emptyResponse, err
	}

	// Parse the response into our provider-specific struct
	var apiResponse anthropicResponse
	if err := json.Unmarshal(bodyBytes, &apiResponse); err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error unmarshalling response: %w", err), "anthropic", parts.Model)
	}

	// Extract content from anthropic's response
	content := ""
	for _, contentItem := range apiResponse.Content {
		if contentItem.Type == "text" {
			content += contentItem.Text
		}
	}

	// Map to the unified LLMResponse
	unifiedResponse := core.LLMResponse{
		Content:      content,
		FinishReason: apiResponse.StopReason,
		Usage: core.TokenUsage{
			PromptTokens:     apiResponse.Usage.InputTokens,
			CompletionTokens: apiResponse.Usage.OutputTokens,
			TotalTokens:      apiResponse.Usage.InputTokens + apiResponse.Usage.OutputTokens,
		},
	}

	return unifiedResponse, nil
}

func (c *anthropicClient) transformMessages(messages []core.Message) []anthropicMessage {
	var anthropicMessages []anthropicMessage
	for _, msg := range messages {
		var contentStr string
		for _, content := range msg.Contents {
			if textContent, ok := content.(core.TextContent); ok {
				contentStr += textContent.Text
			}
		}
		anthropicMessages = append(anthropicMessages, anthropicMessage{
			Role:    msg.Role,
			Content: contentStr,
		})
	}
	return anthropicMessages
}
