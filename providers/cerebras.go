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

type cerebrasClient struct {
	apiKey string
	client *http.Client
}

// NewCerebrasClient creates a new client for the Cerebras API.
func NewCerebrasClient(apiKey string) core.ProviderClient {
	return &cerebrasClient{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

// cerebras response structs (follows OpenAI format)
type cerebrasResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []cerebrasChoice   `json:"choices"`
	Usage   cerebrasUsageTokens `json:"usage"`
}

type cerebrasChoice struct {
	Index        int             `json:"index"`
	Message      cerebrasMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

type cerebrasUsageTokens struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (c *cerebrasClient) GetCompletion(ctx context.Context, parts core.LLMCallParts) (core.LLMResponse, error) {
	emptyResponse := core.LLMResponse{}
	if c.apiKey == "" {
		return emptyResponse, core.NewLLMError(fmt.Errorf("API key is not set"), "cerebras", parts.Model)
	}

	reqBody := cerebrasRequest{
		Model:       parts.Model,
		Messages:    c.transformMessages(parts.Messages, parts.System),
		MaxTokens:   parts.MaxTokens,
		Temperature: parts.Temperature,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error marshalling request: %w", err), "cerebras", parts.Model)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.cerebras.ai/v1/chat/completions", bytes.NewBuffer(reqBytes))
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error creating request: %w", err), "cerebras", parts.Model)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error sending request: %w", err), "cerebras", parts.Model)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error reading response body: %w", err), "cerebras", parts.Model)
	}

	if resp.StatusCode != http.StatusOK {
		err := core.NewLLMError(fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes)), "cerebras", parts.Model)
		err.StatusCode = resp.StatusCode
		err.LastRaw = string(bodyBytes)
		return emptyResponse, err
	}

	// Parse the response into our provider-specific struct
	var apiResponse cerebrasResponse
	if err := json.Unmarshal(bodyBytes, &apiResponse); err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error unmarshalling response: %w", err), "cerebras", parts.Model)
	}

	// Check if we got any choices back
	if len(apiResponse.Choices) == 0 {
		return emptyResponse, core.NewLLMError(fmt.Errorf("response contained no choices"), "cerebras", parts.Model)
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

func (c *cerebrasClient) transformMessages(messages []core.Message, systemMessage core.Message) []cerebrasMessage {
	var cerebrasMessages []cerebrasMessage
	
	// Add system message first if it has content
	if len(systemMessage.Contents) > 0 {
		var systemContent string
		for _, content := range systemMessage.Contents {
			if textContent, ok := content.(core.TextContent); ok {
				systemContent += textContent.Text
			}
		}
		if systemContent != "" {
			cerebrasMessages = append(cerebrasMessages, cerebrasMessage{
				Role:    "system",
				Content: systemContent,
			})
		}
	}
	
	// Add regular messages
	for _, msg := range messages {
		var contentStr string
		for _, content := range msg.Contents {
			if textContent, ok := content.(core.TextContent); ok {
				contentStr += textContent.Text
			}
		}
		cerebrasMessages = append(cerebrasMessages, cerebrasMessage{
			Role:    msg.Role,
			Content: contentStr,
		})
	}
	return cerebrasMessages
}
