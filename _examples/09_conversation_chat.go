package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/recera/gai"
)

func main() {
	gai.FindAndLoadEnv()
	client, err := gai.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	conv := gai.NewLLMCallParts().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithSystem("You are a concise, helpful assistant.")

	ctx := context.Background()
	in := bufio.NewScanner(os.Stdin)
	fmt.Println("Chat (type 'quit' to exit)")
	for {
		fmt.Print("\nYou: ")
		if !in.Scan() {
			break
		}
		msg := strings.TrimSpace(in.Text())
		if msg == "quit" || msg == "exit" {
			break
		}
		conv.WithUserMessage(msg)
		resp, err := gai.GetCompletionP(ctx, client, conv)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		fmt.Println("\nAI:", resp.Content)
		conv.WithAssistantMessage(resp.Content)
		if len(conv.Messages) > 10 {
			conv.KeepLastMessages(10)
		}
	}
}
