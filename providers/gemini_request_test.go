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

func TestGemini_FunctionDeclarations_And_ResponseSchema(t *testing.T) {
	c := &geminiClient{apiKey: "key", client: &http.Client{}}
	var captured map[string]any
	c.client.Transport = rtFunc(func(req *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(b, &captured)
		resp := map[string]any{
			"candidates":    []any{map[string]any{"content": map[string]any{"role": "model", "parts": []any{map[string]any{"text": "ok"}}}, "finishReason": "stop"}},
			"usageMetadata": map[string]any{"promptTokenCount": 1, "candidatesTokenCount": 1, "totalTokenCount": 2},
		}
		out, _ := json.Marshal(resp)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(out)), Header: make(http.Header)}, nil
	})

	parts := core.LLMCallParts{
		Provider: "gemini", Model: "gemini-pro",
		Tools:        []core.ToolDefinition{{Name: "get_time", JSONSchema: map[string]any{"type": "object", "properties": map[string]any{}}}},
		ProviderOpts: map[string]any{"response_schema": map[string]any{"type": "object", "properties": map[string]any{"ok": map[string]any{"type": "boolean"}}}},
	}

	_, err := c.GetCompletion(context.Background(), parts)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("tools not present in request")
	}
	if v, ok := captured["response_mime_type"].(string); !ok || v == "" {
		t.Fatalf("response_mime_type missing")
	}
	if _, ok := captured["response_schema"].(map[string]any); !ok {
		t.Fatalf("response_schema missing")
	}
}

func TestGemini_FunctionResponse_Injection_FromToolMessage(t *testing.T) {
	c := &geminiClient{apiKey: "key", client: &http.Client{}}
	var captured map[string]any
	c.client.Transport = rtFunc(func(req *http.Request) (*http.Response, error) {
		// capture request body
		b, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(b, &captured)
		// Return a minimal OK response
		resp := map[string]any{
			"candidates":    []any{map[string]any{"content": map[string]any{"role": "model", "parts": []any{map[string]any{"text": "ok"}}}, "finishReason": "stop"}},
			"usageMetadata": map[string]any{"promptTokenCount": 1, "candidatesTokenCount": 1, "totalTokenCount": 2},
		}
		out, _ := json.Marshal(resp)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(out)), Header: make(http.Header)}, nil
	})

	// Build a conversation that includes a tool result message with ToolName set
	parts := core.LLMCallParts{Provider: "gemini", Model: "gemini-pro"}
	// One user message, then a tool response
	parts.Messages = append(parts.Messages, core.Message{Role: "user", Contents: []core.Content{core.TextContent{Text: "time please"}}})
	toolMsg := core.Message{Role: "tool", ToolCallID: "id1"}
	toolMsg.ToolName = "get_time"
	toolMsg.AddTextContent("2025-01-01T00:00:00Z")
	parts.Messages = append(parts.Messages, toolMsg)

	_, err := c.GetCompletion(context.Background(), parts)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// Ensure last content part is a functionResponse with the tool name
	contents, _ := captured["contents"].([]any)
	if len(contents) < 2 {
		t.Fatalf("expected at least 2 contents entries")
	}
	last := contents[len(contents)-1].(map[string]any)
	partsArr := last["parts"].([]any)
	found := false
	for _, p := range partsArr {
		if m, ok := p.(map[string]any); ok {
			if fr, ok := m["functionResponse"].(map[string]any); ok {
				if fr["name"] == "get_time" {
					found = true
					break
				}
			}
		}
	}
	if !found {
		t.Fatalf("functionResponse not present for tool name")
	}
}
