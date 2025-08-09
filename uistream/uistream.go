package uistream

import (
	"bufio"
	"encoding/json"
	"net/http"

	"github.com/recera/gai/core"
)

// Write streams chunks to an SSE response compatible with AI SDK UI Message Stream.
// Sets Content-Type: text/event-stream and x-vercel-ai-ui-message-stream: v1.
func Write(w http.ResponseWriter, ch <-chan core.StreamChunk) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("x-vercel-ai-ui-message-stream", "v1")

	bw := bufio.NewWriter(w)
	defer bw.Flush()

	for c := range ch {
		payload, _ := json.Marshal(c)
		bw.WriteString("data: ")
		bw.Write(payload)
		bw.WriteString("\n\n")
		bw.Flush()
	}
}
