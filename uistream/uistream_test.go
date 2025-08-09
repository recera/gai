package uistream

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/recera/gai/core"
)

func TestWrite_SSEFormat(t *testing.T) {
	w := httptest.NewRecorder()
	ch := make(chan core.StreamChunk, 3)
	ch <- core.StreamChunk{Type: "content", Delta: "Hi"}
	ch <- core.StreamChunk{Type: "tool_call", Call: &core.ToolCall{Name: "get_time"}}
	ch <- core.StreamChunk{Type: "end", FinishReason: "stop"}
	close(ch)
	Write(w, ch)
	resp := w.Result()
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("bad content type")
	}
	if resp.Header.Get("x-vercel-ai-ui-message-stream") != "v1" {
		t.Fatalf("missing ui message stream header")
	}
	body := w.Body.String()
	// At least 3 SSE data lines
	if strings.Count(body, "data:") < 3 {
		t.Fatalf("expected 3 data lines, got: %s", body)
	}
}
