//go:build examples

package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/recera/gai"
	"github.com/recera/gai/uistream"
)

func main() {
	gai.FindAndLoadEnv()
	client, _ := gai.NewClient()
	http.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Prompt string `json:"prompt"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		parts := gai.NewLLMCallParts().
			WithProvider("openai").
			WithModel("gpt-4o-mini").
			WithUserMessage(body.Prompt)

		ctx := context.Background()
		ch := make(chan gai.StreamChunk)
		go func() {
			defer close(ch)
			_ = client.StreamCompletion(ctx, parts.Value(), func(s gai.StreamChunk) error {
				ch <- s
				return nil
			})
		}()
		uistream.Write(w, ch)
	})
	_ = http.ListenAndServe(":8710", nil)
}
