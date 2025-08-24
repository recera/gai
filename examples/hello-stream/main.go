// Package main demonstrates streaming text generation with the GAI framework.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/middleware"
	"github.com/recera/gai/providers/openai"
)

func main() {
	// Check for API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set the OPENAI_API_KEY environment variable")
	}

	// Create the OpenAI provider
	var provider core.Provider = openai.New(
		openai.WithAPIKey(apiKey),
		openai.WithModel("gpt-4o-mini"),
	)

	// Apply middleware for production readiness
	provider = middleware.Chain(
		middleware.WithRetry(middleware.RetryOpts{
			MaxAttempts: 3,
			BaseDelay:   time.Second,
			MaxDelay:    10 * time.Second,
			Jitter:      true,
		}),
		middleware.WithRateLimit(middleware.RateLimitOpts{
			RPS:   5,
			Burst: 10,
		}),
	)(provider)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Example 1: Basic streaming
	fmt.Println("=== Example 1: Basic Streaming ===\n")
	basicStreamExample(ctx, provider)

	// Example 2: Streaming with event types
	fmt.Println("\n=== Example 2: Streaming with Event Types ===\n")
	eventStreamExample(ctx, provider)

	// Example 3: Streaming with real-time processing
	fmt.Println("\n=== Example 3: Real-time Processing ===\n")
	realtimeStreamExample(ctx, provider)

	// Example 4: Streaming with error handling
	fmt.Println("\n=== Example 4: Streaming with Error Handling ===\n")
	errorHandlingExample(ctx, provider)
}

func basicStreamExample(ctx context.Context, provider core.Provider) {
	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Tell me a short story about a curious cat exploring a mysterious garden. Make it engaging!"},
				},
			},
		},
		Stream: true,
	}

	stream, err := provider.StreamText(ctx, request)
	if err != nil {
		log.Printf("Error creating stream: %v", err)
		return
	}
	defer stream.Close()

	fmt.Print("Streaming response: ")
	
	// Collect the full text for display
	var fullText strings.Builder
	
	for event := range stream.Events() {
		switch event.Type {
		case core.EventTextDelta:
			fmt.Print(event.TextDelta)
			fullText.WriteString(event.TextDelta)
		case core.EventFinish:
			fmt.Println("\n\n[Stream completed]")
			if event.Usage != nil {
				fmt.Printf("Usage: %d input tokens, %d output tokens\n",
					event.Usage.InputTokens, event.Usage.OutputTokens)
			}
		case core.EventError:
			fmt.Printf("\nStream error: %v\n", event.Err)
		}
	}
}

func eventStreamExample(ctx context.Context, provider core.Provider) {
	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a helpful assistant. Structure your response with clear sections."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Explain the water cycle in simple terms."},
				},
			},
		},
		Stream: true,
	}

	stream, err := provider.StreamText(ctx, request)
	if err != nil {
		log.Printf("Error creating stream: %v", err)
		return
	}
	defer stream.Close()

	fmt.Println("Monitoring all event types:")
	fmt.Println(strings.Repeat("-", 50))

	eventCounts := make(map[core.EventType]int)
	var totalChars int

	for event := range stream.Events() {
		eventCounts[event.Type]++
		
		switch event.Type {
		case core.EventStart:
			fmt.Println("📝 Stream started")
		case core.EventTextDelta:
			fmt.Print(event.TextDelta)
			totalChars += len(event.TextDelta)
		case core.EventFinishStep:
			fmt.Println("\n⏸️  Step finished")
		case core.EventFinish:
			fmt.Println("\n✅ Stream completed")
			if event.Usage != nil {
				fmt.Printf("📊 Tokens - Input: %d, Output: %d, Total: %d\n",
					event.Usage.InputTokens,
					event.Usage.OutputTokens,
					event.Usage.TotalTokens)
			}
		case core.EventError:
			fmt.Printf("\n❌ Error: %v\n", event.Err)
		}
	}

	fmt.Println(strings.Repeat("-", 50))
	fmt.Println("Event Summary:")
	for eventType, count := range eventCounts {
		fmt.Printf("  %v: %d\n", eventType, count)
	}
	fmt.Printf("  Total characters streamed: %d\n", totalChars)
}

func realtimeStreamExample(ctx context.Context, provider core.Provider) {
	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Count from 1 to 10 slowly, with a brief description of each number's significance."},
				},
			},
		},
		Stream:      true,
		Temperature: 0.7,
	}

	stream, err := provider.StreamText(ctx, request)
	if err != nil {
		log.Printf("Error creating stream: %v", err)
		return
	}
	defer stream.Close()

	fmt.Println("Real-time processing example:")
	fmt.Println("(Detecting numbers as they stream)")
	fmt.Println(strings.Repeat("-", 50))

	var buffer strings.Builder
	numbersDetected := []string{}

	for event := range stream.Events() {
		switch event.Type {
		case core.EventTextDelta:
			// Display the text
			fmt.Print(event.TextDelta)
			buffer.WriteString(event.TextDelta)
			
			// Detect numbers in real-time
			text := buffer.String()
			for _, num := range []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"} {
				if strings.Contains(text, num) {
					found := false
					for _, detected := range numbersDetected {
						if detected == num {
							found = true
							break
						}
					}
					if !found {
						numbersDetected = append(numbersDetected, num)
						// Could trigger real-time actions here
					}
				}
			}
			
		case core.EventFinish:
			fmt.Println("\n" + strings.Repeat("-", 50))
			fmt.Printf("Numbers detected in order: %v\n", numbersDetected)
		}
	}
}

func errorHandlingExample(ctx context.Context, provider core.Provider) {
	// Create a context that will timeout quickly to demonstrate error handling
	shortCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Write a very long essay about the history of computing."},
				},
			},
		},
		Stream:    true,
		MaxTokens: 2000,
	}

	stream, err := provider.StreamText(shortCtx, request)
	if err != nil {
		log.Printf("Error creating stream: %v", err)
		// Check error type
		if aiErr, ok := err.(*core.AIError); ok {
			fmt.Printf("Error details:\n")
			fmt.Printf("  Code: %s\n", aiErr.Code)
			fmt.Printf("  Provider: %s\n", aiErr.Provider)
			fmt.Printf("  Retryable: %v\n", aiErr.Temporary)
		}
		return
	}
	defer stream.Close()

	fmt.Println("Attempting to stream (may timeout)...")
	
	var receivedText strings.Builder
	completed := false

	for event := range stream.Events() {
		select {
		case <-shortCtx.Done():
			fmt.Println("\n⏰ Context timeout - demonstrating graceful shutdown")
			stream.Close()
			completed = false
			break
		default:
			switch event.Type {
			case core.EventTextDelta:
				fmt.Print(event.TextDelta)
				receivedText.WriteString(event.TextDelta)
			case core.EventFinish:
				completed = true
				fmt.Println("\n✅ Stream completed successfully")
			case core.EventError:
				fmt.Printf("\n❌ Stream error: %v\n", event.Err)
				// Analyze error
				if err := event.Err; err != nil {
					if core.IsTransient(err) {
						fmt.Println("   This error is transient and can be retried")
					}
					if core.IsRateLimited(err) {
						fmt.Println("   Rate limited - should wait before retrying")
					}
				}
			}
		}
	}

	if !completed && receivedText.Len() > 0 {
		fmt.Printf("\nPartial response received (%d chars) before interruption\n", receivedText.Len())
	}

	// Demonstrate proper streaming with adequate timeout
	fmt.Println("\n--- Retrying with proper timeout ---")
	
	properCtx, properCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer properCancel()

	// Simpler request for quick success
	request.Messages[0].Parts[0] = core.Text{Text: "Say 'Hello, streaming works!'"}
	request.MaxTokens = 50

	stream2, err := provider.StreamText(properCtx, request)
	if err != nil {
		log.Printf("Error creating stream: %v", err)
		return
	}
	defer stream2.Close()

	fmt.Print("Response: ")
	for event := range stream2.Events() {
		if event.Type == core.EventTextDelta {
			fmt.Print(event.TextDelta)
		} else if event.Type == core.EventFinish {
			fmt.Println("\n✅ Success!")
		}
	}
}