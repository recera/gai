//go:build examples

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/recera/gai"
)

func main() {
	gai.FindAndLoadEnv()
	client, err := gai.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	// Define a simple tool
	tools := []gai.ToolDefinition{{
		Name:        "get_time",
		Description: "Get current UTC time",
		JSONSchema:  map[string]any{"type": "object", "properties": map[string]any{}},
	}}

	parts := gai.NewLLMCallParts().
		WithProvider("anthropic").
		WithModel("claude-3-haiku-20240307").
		WithSystem("Use tools when appropriate; keep answers concise.").
		WithUserMessage("What time is it in UTC?")
	parts.WithTools(tools...)

	// Executor
	exec := func(call gai.ToolCall) (string, error) {
		if call.Name == "get_time" {
			return time.Now().UTC().Format(time.RFC3339), nil
		}
		return "", fmt.Errorf("unknown tool: %s", call.Name)
	}

	// Stream with tools orchestration
	ctx := context.Background()
	err = client.StreamWithTools(ctx, parts.Value(), exec, func(ch gai.StreamChunk) error {
		switch ch.Type {
		case "content":
			fmt.Print(ch.Delta)
		case "tool_call":
			// Already handled by StreamWithTools; you could visualize the call here
		case "end":
			fmt.Printf("\n[done] reason=%s\n", ch.FinishReason)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
