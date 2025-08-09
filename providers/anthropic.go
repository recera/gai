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
	"github.com/recera/gai/observability"
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
	Type  string                 `json:"type"`
	Text  string                 `json:"text,omitempty"`
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
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

	// Map tools if provided (Anthropic: tools at top-level with input_schema)
	if len(parts.Tools) > 0 {
		tools := make([]anthropicTool, 0, len(parts.Tools))
		for _, t := range parts.Tools {
			tools = append(tools, anthropicTool{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.JSONSchema,
			})
		}
		reqBody.Tools = tools
	}
	if parts.ToolChoice != nil {
		reqBody.ToolChoice = parts.ToolChoice
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

	// Detect tool_use blocks; if present, emit ToolCalls for blocking flows
	var toolCalls []core.ToolCall
	content := ""
	for _, contentItem := range apiResponse.Content {
		switch contentItem.Type {
		case "text":
			content += contentItem.Text
		case "tool_use":
			// For blocking call, surface as ToolCalls to allow client to handle tool loop
			b, _ := json.Marshal(contentItem.Input)
			toolCalls = append(toolCalls, core.ToolCall{ID: contentItem.ID, Name: contentItem.Name, Arguments: string(b)})
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
		ToolCalls: toolCalls,
	}

	return unifiedResponse, nil
}

func (c *anthropicClient) transformMessages(messages []core.Message) []anthropicMessage {
	var out []anthropicMessage
	for _, msg := range messages {
		// For provider-native tool result, convert Role:"tool" into an Anthropic tool_result block
		if msg.Role == "tool" && msg.ToolCallID != "" {
			block := map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": msg.ToolCallID,
				"content":     msg.GetTextContent(),
			}
			out = append(out, anthropicMessage{Role: "user", Content: []interface{}{block}})
			continue
		}
		var blocks []interface{}
		for _, content := range msg.Contents {
			if textContent, ok := content.(core.TextContent); ok {
				blocks = append(blocks, anthropicTextContent{Type: "text", Text: textContent.Text})
			}
		}
		out = append(out, anthropicMessage{Role: msg.Role, Content: blocks})
	}
	return out
}

// StreamCompletion implements a minimal line-based streaming for Anthropic messages streaming API
func (c *anthropicClient) StreamCompletion(ctx context.Context, parts core.LLMCallParts, handler core.StreamHandler) error {
	if c.apiKey == "" {
		return core.NewLLMError(fmt.Errorf("API key is not set"), "anthropic", parts.Model)
	}
	ctx, span, metrics := observability.StartStream(ctx, "anthropic", parts.Model)
	// Anthropic streaming uses the messages API with header "anthropic-version", and stream=true via SSE.
	// We'll request text deltas by setting stream=true&stream_tokens=true equivalent JSON body.
	body := map[string]interface{}{
		"model":       parts.Model,
		"max_tokens":  parts.MaxTokens,
		"temperature": parts.Temperature,
		"messages":    c.transformMessages(parts.Messages),
		"stream":      true,
	}
	if len(parts.Tools) > 0 {
		tools := make([]anthropicTool, 0, len(parts.Tools))
		for _, t := range parts.Tools {
			tools = append(tools, anthropicTool{Name: t.Name, Description: t.Description, InputSchema: t.JSONSchema})
		}
		body["tools"] = tools
	}
	if parts.ToolChoice != nil {
		body["tool_choice"] = parts.ToolChoice
	}
	if len(parts.System.Contents) > 0 {
		if textContent, ok := parts.System.Contents[0].(core.TextContent); ok {
			body["system"] = textContent.Text
		}
	}

	reqBytes, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anthropic stream status %d: %s", resp.StatusCode, string(b))
	}

	// SSE lines that start with "data: {json}"
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
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
					observability.CloseStream(span, metrics, "")
					return nil
				}
				// Parse Anthropic events for text deltas and tool_use
				var generic map[string]interface{}
				if err := json.Unmarshal([]byte(payload), &generic); err == nil {
					if generic["type"] == "content_block_delta" {
						if delta, ok := generic["delta"].(map[string]interface{}); ok {
							if t, ok := delta["text"].(string); ok && t != "" {
								observability.MarkFirstToken(metrics)
								if err := handler(core.StreamChunk{Type: "content", Delta: t}); err != nil {
									return err
								}
							}
						}
					}
					if generic["type"] == "tool_use" {
						// tool_use: {type:"tool_use", id:"...", name:"...", input:{...}}
						id, _ := generic["id"].(string)
						name, _ := generic["name"].(string)
						// input may be object; keep raw json for arguments
						if input, ok := generic["input"]; ok && id != "" && name != "" {
							b, _ := json.Marshal(input)
							call := core.ToolCall{ID: id, Name: name, Arguments: string(b)}
							observability.MarkFirstToken(metrics)
							if err := handler(core.StreamChunk{Type: "tool_call", Call: &call}); err != nil {
								return err
							}
						}
					}
					if fr, ok := generic["stop_reason"].(string); ok && fr != "" {
						if err := handler(core.StreamChunk{Type: "end", FinishReason: fr}); err != nil {
							return err
						}
						observability.CloseStream(span, metrics, fr)
					}
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				observability.CloseStream(span, metrics, "")
				break
			}
			return err
		}
	}
	return nil
}
