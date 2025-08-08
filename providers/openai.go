package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/recera/gai/core"
)

type openAIClient struct {
	apiKey string
	client *http.Client
}

// NewOpenAIClient creates a new client for the OpenAI API.
func NewOpenAIClient(apiKey string) core.ProviderClient {
	return &openAIClient{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

// OpenAI response structs
type openAIResponse struct {
	ID      string         `json:"id"`
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
	Model   string         `json:"model"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (c *openAIClient) GetCompletion(ctx context.Context, parts core.LLMCallParts) (core.LLMResponse, error) {
	emptyResponse := core.LLMResponse{}
	if c.apiKey == "" {
		return emptyResponse, core.NewLLMError(fmt.Errorf("API key is not set"), "openai", parts.Model)
	}

	// Include system message if present by prepending to messages
	transformed := c.transformMessagesWithSystem(parts.Messages, parts.System)
	reqBody := openAIRequest{
		Model:       parts.Model,
		Messages:    transformed,
		MaxTokens:   parts.MaxTokens,
		Temperature: parts.Temperature,
	}

	// Add tools if provided
	if len(parts.Tools) > 0 {
		tools := make([]openAITool, 0, len(parts.Tools))
		for _, t := range parts.Tools {
			tools = append(tools, openAITool{
				Type: "function",
				Function: openAIFunction{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.JSONSchema,
				},
			})
		}
		reqBody.Tools = tools
		// Let model decide tools automatically; caller can override in future API if needed
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error marshalling request: %w", err), "openai", parts.Model)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBytes))
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error creating request: %w", err), "openai", parts.Model)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error sending request: %w", err), "openai", parts.Model)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error reading response body: %w", err), "openai", parts.Model)
	}

	if resp.StatusCode != http.StatusOK {
		err := core.NewLLMError(fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes)), "openai", parts.Model)
		err.StatusCode = resp.StatusCode
		err.LastRaw = string(bodyBytes)
		return emptyResponse, err
	}

	// Parse the response into our provider-specific struct
	var apiResponse openAIResponse
	if err := json.Unmarshal(bodyBytes, &apiResponse); err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error unmarshalling response: %w", err), "openai", parts.Model)
	}

	// Check if we got any choices back
	if len(apiResponse.Choices) == 0 {
		return emptyResponse, core.NewLLMError(fmt.Errorf("response contained no choices"), "openai", parts.Model)
	}

	// Map tool calls if any
	var toolCalls []core.ToolCall
	if len(apiResponse.Choices[0].Message.ToolCalls) > 0 {
		for _, tc := range apiResponse.Choices[0].Message.ToolCalls {
			toolCalls = append(toolCalls, core.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
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
		ToolCalls: toolCalls,
	}

	return unifiedResponse, nil
}

func (c *openAIClient) transformMessages(messages []core.Message) []openAIMessage {
	var openAIMessages []openAIMessage
	for _, msg := range messages {
		var contentStr string
		for _, content := range msg.Contents {
			if textContent, ok := content.(core.TextContent); ok {
				contentStr += textContent.Text
			}
		}
		openAIMessages = append(openAIMessages, openAIMessage{
			Role:    msg.Role,
			Content: contentStr,
		})
	}
	return openAIMessages
}

func (c *openAIClient) transformMessagesWithSystem(messages []core.Message, system core.Message) []openAIMessage {
	result := make([]openAIMessage, 0, len(messages)+1)
	if len(system.Contents) > 0 {
		var sys string
		for _, content := range system.Contents {
			if textContent, ok := content.(core.TextContent); ok {
				sys += textContent.Text
			}
		}
		if sys != "" {
			result = append(result, openAIMessage{Role: "system", Content: sys})
		}
	}
	return append(result, c.transformMessages(messages)...)
}

// StreamCompletion implements SSE streaming for OpenAI chat.completions
func (c *openAIClient) StreamCompletion(ctx context.Context, parts core.LLMCallParts, handler core.StreamHandler) error {
	if c.apiKey == "" {
		return core.NewLLMError(fmt.Errorf("API key is not set"), "openai", parts.Model)
	}
	transformed := c.transformMessagesWithSystem(parts.Messages, parts.System)
	reqBody := map[string]interface{}{
		"model":       parts.Model,
		"messages":    transformed,
		"max_tokens":  parts.MaxTokens,
		"temperature": parts.Temperature,
		"stream":      true,
	}
	if len(parts.Tools) > 0 {
		tools := make([]openAITool, 0, len(parts.Tools))
		for _, t := range parts.Tools {
			tools = append(tools, openAITool{Type: "function", Function: openAIFunction{Name: t.Name, Description: t.Description, Parameters: t.JSONSchema}})
		}
		reqBody["tools"] = tools
	}

	reqBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai stream status %d: %s", resp.StatusCode, string(body))
	}

	// The OpenAI stream uses SSE where each line begins with 'data:'. We'll do a simple scan.
	// For simplicity here we decode JSON tokens directly if present, else treat as plain lines.
	// Minimal implementation: accumulate deltas only.
	type streamChoiceDelta struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
	}
	type streamEvent struct {
		Choices []streamChoiceDelta `json:"choices"`
	}
	// Fallback line-by-line reader to handle SSE "data: {json}"
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			// Split on newlines and parse any JSON after "data: "
			for {
				i := bytes.IndexByte(buf, '\n')
				if i < 0 {
					break
				}
				line := string(bytes.TrimSpace(buf[:i]))
				buf = buf[i+1:]
				if !strings.HasPrefix(line, "data:") {
					continue
				}
				payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if payload == "[DONE]" {
					_ = handler(core.StreamChunk{Type: "end"})
					return nil
				}
				var ev streamEvent
				if err := json.Unmarshal([]byte(payload), &ev); err == nil {
					for _, ch := range ev.Choices {
						if ch.Delta.Content != "" {
							if err := handler(core.StreamChunk{Type: "content", Delta: ch.Delta.Content}); err != nil {
								return err
							}
						}
						if ch.FinishReason != "" {
							if err := handler(core.StreamChunk{Type: "end", FinishReason: ch.FinishReason}); err != nil {
								return err
							}
						}
					}
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}
