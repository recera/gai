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

type geminiClient struct {
	apiKey string
	client *http.Client
}

// NewGeminiClient creates a new client for the Google Gemini API.
func NewGeminiClient(apiKey string) core.ProviderClient {
	return &geminiClient{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

// gemini response structs
type geminiResponse struct {
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata geminiUsage       `json:"usageMetadata"`
}

type geminiCandidate struct {
	Content      geminiResponseContent `json:"content"`
	FinishReason string                `json:"finishReason"`
}

type geminiResponseContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

func (c *geminiClient) GetCompletion(ctx context.Context, parts core.LLMCallParts) (core.LLMResponse, error) {
	emptyResponse := core.LLMResponse{}
	if c.apiKey == "" {
		return emptyResponse, core.NewLLMError(fmt.Errorf("API key is not set"), "gemini", parts.Model)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", parts.Model, c.apiKey)

	// Gemini tends to accept instructions as part of content. Prepend a system preamble if present.
	contents := c.transformMessages(parts.Messages)
	if len(parts.System.Contents) > 0 {
		var sys string
		for _, content := range parts.System.Contents {
			if textContent, ok := content.(core.TextContent); ok {
				sys += textContent.Text
			}
		}
		if sys != "" {
			pre := geminiContent{Role: "user", Parts: []geminiPart{{Text: sys}}}
			contents = append([]geminiContent{pre}, contents...)
		}
	}
	reqBody := geminiRequest{
		Contents: contents,
		GenerationConfig: generationConfig{
			Temperature:     parts.Temperature,
			MaxOutputTokens: parts.MaxTokens,
		},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error marshalling request: %w", err), "gemini", parts.Model)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error creating request: %w", err), "gemini", parts.Model)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error sending request: %w", err), "gemini", parts.Model)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error reading response body: %w", err), "gemini", parts.Model)
	}

	if resp.StatusCode != http.StatusOK {
		err := core.NewLLMError(fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes)), "gemini", parts.Model)
		err.StatusCode = resp.StatusCode
		err.LastRaw = string(bodyBytes)
		return emptyResponse, err
	}

	// Parse the response into our provider-specific struct
	var apiResponse geminiResponse
	if err := json.Unmarshal(bodyBytes, &apiResponse); err != nil {
		return emptyResponse, core.NewLLMError(fmt.Errorf("error unmarshalling response: %w", err), "gemini", parts.Model)
	}

	// Check if we got any candidates back
	if len(apiResponse.Candidates) == 0 {
		return emptyResponse, core.NewLLMError(fmt.Errorf("response contained no candidates"), "gemini", parts.Model)
	}

	// Extract content from gemini's response
	content := ""
	for _, part := range apiResponse.Candidates[0].Content.Parts {
		content += part.Text
	}

	// Map to the unified LLMResponse
	unifiedResponse := core.LLMResponse{
		Content:      content,
		FinishReason: apiResponse.Candidates[0].FinishReason,
		Usage: core.TokenUsage{
			PromptTokens:     apiResponse.UsageMetadata.PromptTokenCount,
			CompletionTokens: apiResponse.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      apiResponse.UsageMetadata.TotalTokenCount,
		},
	}

	return unifiedResponse, nil
}

func (c *geminiClient) transformMessages(messages []core.Message) []geminiContent {
	var geminiContents []geminiContent
	for _, msg := range messages {
		var parts []geminiPart
		for _, content := range msg.Contents {
			if textContent, ok := content.(core.TextContent); ok {
				parts = append(parts, geminiPart{Text: textContent.Text})
			}
		}
		geminiContents = append(geminiContents, geminiContent{
			Role:  c.formatRole(msg.Role),
			Parts: parts,
		})
	}
	return geminiContents
}

func (c *geminiClient) formatRole(role string) string {
	if role == "assistant" {
		return "model"
	}
	return role
}
