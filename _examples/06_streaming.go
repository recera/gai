//go:build examples

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/recera/gai"
)

func main() {
	// Load .env from repo root if present
	gai.FindAndLoadEnv()

	client, err := gai.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	parts := gai.NewLLMCallParts().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithSystem("Respond in small chunks.").
		WithUserMessage("Give me 3 short facts about Go.")

	ctx := context.Background()
	err = client.StreamCompletion(ctx, parts.Value(), func(ch gai.StreamChunk) error {
		switch ch.Type {
		case "content":
			fmt.Print(ch.Delta)
		case "end":
			fmt.Printf("\n[done] reason=%s\n", ch.FinishReason)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
