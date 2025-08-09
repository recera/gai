package providers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/recera/gai/core"
)

// Constructs a fake SSE stream where OpenAI emits a tool_calls delta followed by finish_reason=tool_calls
func TestOpenAI_StreamCompletion_EmitsToolCallChunks(t *testing.T) {
	c := &openAIClient{apiKey: "test", client: &http.Client{}}

	sse := bytes.NewBuffer(nil)
	// First delta: introduce tool_calls with id and name
	sse.WriteString("data: {\n")
	sse.WriteString("  \"choices\": [{\"index\":0, \"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"get_time\",\"arguments\":\"{\\\\\"tz\\\\\":\\\\\"UTC\\\\\"}\"}}]}}]\n")
	sse.WriteString("}\n\n")
	// Finish: reason is tool_calls
	sse.WriteString("data: {\"choices\":[{\"index\":0,\"finish_reason\":\"tool_calls\"}]}\n\n")
	// Done
	sse.WriteString("data: [DONE]\n\n")

	c.client.Transport = rtFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(sse.Bytes())), Header: make(http.Header)}, nil
	})

	parts := core.LLMCallParts{Provider: "openai", Model: "gpt-4o-mini", Tools: []core.ToolDefinition{{Name: "get_time", JSONSchema: map[string]any{"type": "object", "properties": map[string]any{}}}}}

	var toolCallSeen bool
	err := c.StreamCompletion(context.Background(), parts, func(ch core.StreamChunk) error {
		if ch.Type == "tool_call" {
			toolCallSeen = true
			if ch.Call == nil || ch.Call.Name != "get_time" || ch.Call.ID == "" || ch.Call.Arguments == "" {
				t.Fatalf("unexpected tool call content: %+v", ch.Call)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !toolCallSeen {
		t.Fatalf("expected a tool_call chunk")
	}
}

func TestOpenAI_StreamCompletion_CoalescesArgumentsByIndex(t *testing.T) {
	c := &openAIClient{apiKey: "test", client: &http.Client{}}

	// Simulate two deltas for the same tool_call index, splitting arguments
	sse := bytes.NewBuffer(nil)
	// First delta: tool_calls index 0 with partial args
	sse.WriteString("data: {\"choices\": [{\"index\":0, \"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"get_time\",\"arguments\":\"{\\\\\"tz\\\\\":\\\\\"UT\"}}}]}}]\n}\n\n")
	// Second delta: remaining args
	sse.WriteString("data: {\"choices\": [{\"index\":0, \"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"C\\\\\"}\"}}]}}]\n}\n\n")
	// Finish with tool_calls
	sse.WriteString("data: {\"choices\":[{\"index\":0,\"finish_reason\":\"tool_calls\"}]}\n\n")
	sse.WriteString("data: [DONE]\n\n")

	c.client.Transport = rtFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(sse.Bytes())), Header: make(http.Header)}, nil
	})

	parts := core.LLMCallParts{Provider: "openai", Model: "gpt-4o-mini", Tools: []core.ToolDefinition{{Name: "get_time", JSONSchema: map[string]any{"type": "object", "properties": map[string]any{}}}}}

	var calls []core.ToolCall
	err := c.StreamCompletion(context.Background(), parts, func(ch core.StreamChunk) error {
		if ch.Type == "tool_call" && ch.Call != nil {
			calls = append(calls, *ch.Call)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 coalesced tool call, got %d", len(calls))
	}
	if calls[0].Name != "get_time" {
		t.Fatalf("unexpected tool name: %s", calls[0].Name)
	}
	if calls[0].Arguments != "{\\\"tz\\\":\\\"UTC\\\"}" {
		t.Fatalf("unexpected arguments: %s", calls[0].Arguments)
	}
}

func TestOpenAI_StreamCompletion_MultiIndex_Interleaved(t *testing.T) {
	c := &openAIClient{apiKey: "test", client: &http.Client{}}

	sse := bytes.NewBuffer(nil)
	// index 1 first
	sse.WriteString("data: {\"choices\": [{\"index\":0, \"delta\":{\"tool_calls\":[{\"index\":1,\"id\":\"call_b\",\"type\":\"function\",\"function\":{\"name\":\"b\",\"arguments\":\"{\\\\\"x\\\\\":1\"}}}]}}]\n}\n\n")
	// index 0 appears
	sse.WriteString("data: {\"choices\": [{\"index\":0, \"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_a\",\"type\":\"function\",\"function\":{\"name\":\"a\",\"arguments\":\"{\\\\\"y\\\\\":2\"}}}]}}]\n}\n\n")
	// finish
	sse.WriteString("data: {\"choices\":[{\"index\":0,\"finish_reason\":\"tool_calls\"}]}\n\n")
	sse.WriteString("data: [DONE]\n\n")

	c.client.Transport = rtFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(sse.Bytes())), Header: make(http.Header)}, nil
	})

	parts := core.LLMCallParts{Provider: "openai", Model: "gpt-4o-mini", Tools: []core.ToolDefinition{{Name: "a", JSONSchema: map[string]any{"type": "object", "properties": map[string]any{}}}, {Name: "b", JSONSchema: map[string]any{"type": "object", "properties": map[string]any{}}}}}

	var calls []core.ToolCall
	err := c.StreamCompletion(context.Background(), parts, func(ch core.StreamChunk) error {
		if ch.Type == "tool_call" && ch.Call != nil {
			calls = append(calls, *ch.Call)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 coalesced calls, got %d", len(calls))
	}
	if calls[0].Name != "a" || calls[1].Name != "b" {
		t.Fatalf("expected calls in index order [a,b], got [%s,%s]", calls[0].Name, calls[1].Name)
	}
}
