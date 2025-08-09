package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/recera/gai/core"
)

// rtFunc is a helper RoundTripper for mocking http.Client
type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOpenAI_GetCompletion_MapsToolChoiceAndResponseFormat(t *testing.T) {
	c := &openAIClient{apiKey: "test", client: &http.Client{}}

	var capturedBody map[string]any
	c.client.Transport = rtFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(body, &capturedBody)
		// Return a minimal valid response
		resp := map[string]any{
			"id":      "id",
			"model":   "gpt-4o-mini",
			"choices": []any{map[string]any{"index": 0, "finish_reason": "stop", "message": map[string]any{"role": "assistant", "content": "ok"}}},
			"usage":   map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		}
		b, _ := json.Marshal(resp)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(b)), Header: make(http.Header)}, nil
	})

	parts := core.LLMCallParts{
		Provider:      "openai",
		Model:         "gpt-4o-mini",
		MaxTokens:     32,
		Temperature:   0,
		StopSequences: []string{"END"},
		Tools:         []core.ToolDefinition{{Name: "get_time", JSONSchema: map[string]any{"type": "object", "properties": map[string]any{}}}},
		ProviderOpts:  map[string]any{"response_format": map[string]any{"type": "json_object"}},
		ToolChoice:    map[string]any{"type": "function", "function": map[string]any{"name": "get_time"}},
	}

	resp, err := c.GetCompletion(context.Background(), parts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("unexpected content: %q", resp.Content)
	}

	// Assertions on captured request body
	if v, ok := capturedBody["tool_choice"]; !ok || v == nil {
		t.Fatalf("tool_choice not present in request body")
	}
	if v, ok := capturedBody["response_format"]; !ok || v == nil {
		t.Fatalf("response_format not present in request body")
	}
	if v, ok := capturedBody["stop"]; !ok || v == nil {
		t.Fatalf("stop not present in request body")
	}
}
