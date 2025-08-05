package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/collinshill/gai"
)

func main() {
	// Load environment
	gai.FindAndLoadEnv()

	// Create client
	client, err := gai.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	// Create a conversation
	conversation := gai.NewLLMCallParts().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithSystem("You are a helpful AI assistant. Keep responses concise.")

	// Interactive conversation loop
	ctx := context.Background()
	scanner := bufio.NewScanner(os.Stdin)
	
	fmt.Println("Chat with AI (type 'quit' to exit)")
	fmt.Println("=====================================")

	for {
		fmt.Print("\nYou: ")
		if !scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		if input == "quit" || input == "exit" {
			break
		}

		// Add user message
		conversation.WithUserMessage(input)

		// Get response
		response, err := client.GetCompletion(ctx, *conversation)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Add assistant response to conversation
		conversation.WithAssistantMessage(response.Content)

		fmt.Printf("\nAI: %s\n", response.Content)

		// Manage conversation length (keep last 10 messages)
		if len(conversation.Messages) > 10 {
			conversation.KeepLastMessages(10)
		}
	}

	fmt.Println("\nGoodbye!")
}