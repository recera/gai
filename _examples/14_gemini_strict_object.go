//go:build examples

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/recera/gai"
)

type Weather struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_c"`
}

func main() {
	gai.FindAndLoadEnv()
	client, err := gai.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	// Use strict object mode hints for Gemini via ProviderOpts
	parts := gai.NewLLMCallParts().
		WithProvider("gemini").
		WithModel("gemini-pro").
		WithUserMessage("Return current weather for Tokyo as JSON")
	// Set response schema
	schema := map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}, "temp_c": map[string]any{"type": "number"}}, "required": []string{"city", "temp_c"}}
	parts.ProviderOpts = map[string]any{"response_schema": schema}

	ctx := context.Background()
	w, usage, err := gai.GenerateObject[Weather](ctx, client, parts.Value())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Weather: %s %.1fC (tokens=%d)\n", w.City, w.TempC, usage.TotalTokens)
}
