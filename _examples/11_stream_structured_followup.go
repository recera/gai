//go:build examples

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/recera/gai"
)

// Simple structured response after a streamed intro
type Plan struct {
	Steps []string `json:"steps" desc:"Actionable steps"`
}

func main() {
	gai.FindAndLoadEnv()
	client, err := gai.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	// First, stream a quick intro
	parts := gai.NewLLMCallParts().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithSystem("Stream a brief intro, then we will request structured data.").
		WithUserMessage("Introduce the concept of time management in two sentences.")

	ctx := context.Background()
	if err := client.StreamCompletion(ctx, parts.Value(), func(ch gai.StreamChunk) error {
		if ch.Type == "content" {
			fmt.Print(ch.Delta)
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	// Then, ask for structured steps using typed action
	action := gai.NewAction[Plan]().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithSystem("Return a short plan as JSON.").
		WithUserMessage("Provide 3 steps to improve time management.")

	plan, err := action.Run(ctx, client)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\n\nPlan steps:")
	for i, s := range plan.Steps {
		fmt.Printf("%d. %s\n", i+1, s)
	}

	// Provider-native tool calling (blocking)
	parts := gai.NewLLMCallParts().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithSystem("Use tools when needed").
		WithUserMessage("What time is it in UTC?")
	parts.WithTools(gai.ToolDefinition{Name: "get_time", JSONSchema: map[string]any{"type": "object", "properties": map[string]any{}}})
	exec := func(call gai.ToolCall) (string, error) {
		if call.Name == "get_time" {
			return time.Now().UTC().Format(time.RFC3339), nil
		}
		return "", fmt.Errorf("unknown tool: %s", call.Name)
	}
	resp, err := client.RunWithTools(ctx, parts.Value(), exec)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nTool answer:", resp.Content)
}
