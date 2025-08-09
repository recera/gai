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

func TestAnthropic_GetCompletion_ToolUseMappedToToolCalls(t *testing.T) {
	c := &anthropicClient{apiKey: "key", client: &http.Client{}}
	// Mock a tool_use content block
	resp := map[string]any{
		"id": "id", "type": "message", "role": "assistant",
		"content": []any{
			map[string]any{"type": "tool_use", "id": "tu_1", "name": "get_time", "input": map[string]any{"tz": "UTC"}},
		},
		"stop_reason": "tool_use",
		"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
	}
	body, _ := json.Marshal(resp)
	c.client.Transport = rtFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	})

	parts := core.LLMCallParts{Provider: "anthropic", Model: "claude-3-haiku"}
	out, err := c.GetCompletion(context.Background(), parts)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(out.ToolCalls))
	}
	if out.ToolCalls[0].Name != "get_time" {
		t.Fatalf("unexpected tool name: %s", out.ToolCalls[0].Name)
	}
}

func TestAnthropic_StreamCompletion_EmitsToolCall(t *testing.T) {
	c := &anthropicClient{apiKey: "key", client: &http.Client{}}
	sse := bytes.NewBuffer(nil)
	// Emit a tool_use event
	sse.WriteString("data: {\"type\":\"tool_use\",\"id\":\"tu_1\",\"name\":\"get_time\",\"input\":{\"tz\":\"UTC\"}}\n\n")
	sse.WriteString("data: [DONE]\n\n")
	c.client.Transport = rtFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(sse.Bytes())), Header: make(http.Header)}, nil
	})
	parts := core.LLMCallParts{Provider: "anthropic", Model: "claude-3-haiku"}
	var seen bool
	err := c.StreamCompletion(context.Background(), parts, func(ch core.StreamChunk) error {
		if ch.Type == "tool_call" {
			seen = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !seen {
		t.Fatalf("expected tool_call chunk")
	}
}

func TestAnthropic_ToolResult_Injection_FromToolMessage(t *testing.T) {
	c := &anthropicClient{apiKey: "key", client: &http.Client{}}
	var captured map[string]any
	c.client.Transport = rtFunc(func(req *http.Request) (*http.Response, error) {
		// capture body
		b, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(b, &captured)
		// return minimal text response
		resp := map[string]any{
			"id": "id", "type": "message", "role": "assistant",
			"content":     []any{map[string]any{"type": "text", "text": "ok"}},
			"stop_reason": "stop",
			"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
		}
		out, _ := json.Marshal(resp)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(out)), Header: make(http.Header)}, nil
	})

	parts := core.LLMCallParts{Provider: "anthropic", Model: "claude-3-haiku"}
	parts.Messages = append(parts.Messages, core.Message{Role: "user", Contents: []core.Content{core.TextContent{Text: "time"}}})
	tool := core.Message{Role: "tool", ToolCallID: "tu_1"}
	tool.AddTextContent("2025-01-01T00:00:00Z")
	parts.Messages = append(parts.Messages, tool)

	_, err := c.GetCompletion(context.Background(), parts)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// Ensure user message content included a tool_result block
	msgs, _ := captured["messages"].([]any)
	var hasToolResult bool
	for _, m := range msgs {
		if mm, ok := m.(map[string]any); ok {
			if mm["role"] == "user" {
				if arr, ok := mm["content"].([]any); ok {
					for _, cb := range arr {
						if cbm, ok := cb.(map[string]any); ok && cbm["type"] == "tool_result" {
							if cbm["tool_use_id"] == "tu_1" {
								hasToolResult = true
							}
						}
					}
				}
			}
		}
	}
	if !hasToolResult {
		t.Fatalf("expected tool_result block in outgoing user message")
	}
}
