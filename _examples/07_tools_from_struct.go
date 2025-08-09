package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/recera/gai"
)

type GetTimeArgs struct {
	Timezone string `json:"timezone" desc:"IANA timezone, e.g. Europe/Paris"`
}

func main() {
	// Load .env from repo root if present
	gai.FindAndLoadEnv()

	client, err := gai.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	// Generate a tool schema from the struct and add it to the call
	tool, err := gai.ToolFromType[GetTimeArgs]("get_time", gai.ToolGenOptions{
		Description: "Get the current time for a timezone",
		Doc:         "Return an ISO-8601 time string.",
	})
	if err != nil {
		log.Fatal(err)
	}

	parts := gai.NewLLMCallParts().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithSystem("Use tools when necessary.").
		WithTools(tool).
		WithUserMessage("What time is it in New York?")

	ctx := context.Background()

	// Full loop using our own executor
	_, err = client.RunWithTools(ctx, parts.Value(), func(call gai.ToolCall) (string, error) {
		switch call.Name {
		case "get_time":
			var args GetTimeArgs
			if err := gai.ParseInto(call.Arguments, &args); err != nil {
				return "", err
			}
			// this is just a demo; you would use args.Timezone
			return time.Now().Format(time.RFC3339), nil
		default:
			return "", fmt.Errorf("unknown tool: %s", call.Name)
		}
	})
	if err != nil {
		log.Fatal(err)
	}

	// Alternatively, parse tool-call args directly into a struct result and stop
	var parsed GetTimeArgs
	parts2 := gai.NewLLMCallParts().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithSystem("Call the tool with the arguments.").
		WithUserMessage("Please call get_time for Tokyo.")
	if err := gai.GetResponseObjectViaTools(ctx, client, parts2.Value(), "get_time", &parsed); err != nil {
		log.Fatal(err)
	}
	fmt.Println("parsed args:", parsed)
}
