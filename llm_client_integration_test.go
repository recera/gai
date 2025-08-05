//go:build integration
// +build integration

package gai

import (
	"context"
	"testing"
)

type R_Animals struct {
	Mammals []string `json:"mammals"`
	Birds   []string `json:"birds"`
	Fish    []string `json:"fish"`
}

func TestLLMClient(t *testing.T) {
	// Setup test context and client
	ctx := context.Background()
	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create LLM client: %v", err)
	}

	var result R_Animals

	// Test data
	prompt := "Please list one animal in each category (mammals, birds, fish)."
	instructions, err := ResponseInstructions(&result)
	if err != nil {
		t.Fatalf("Failed to generate response instructions: %v", err)
	}

	// Configure LLM call
	callParts := LLMCallParts{
		Provider:    "openai",
		Model:       "gpt-4.1-nano",
		MaxTokens:   200,
		Temperature: 0.7,
	}

	// Add system message with instructions
	callParts.AddSystem(Message{
		Role:     "system",
		Contents: []Content{TextContent{Text: instructions}},
	})

	// Add user prompt
	callParts.AddMessage(Message{
		Role:     "user",
		Contents: []Content{TextContent{Text: prompt}},
	})

	// Make the LLM call
	t.Log("Sending request to LLM...")
	if err := client.GetResponseObject(ctx, callParts, &result); err != nil {
		t.Fatalf("Failed to get response from LLM: %v", err)
	}

	// Verify the response
	t.Logf("Received response: %+v", result)

	// Check that we got responses in each category
	if len(result.Mammals) == 0 {
		t.Error("Expected at least one mammal in response")
	}
	if len(result.Birds) == 0 {
		t.Error("Expected at least one bird in response")
	}
	if len(result.Fish) == 0 {
		t.Error("Expected at least one fish in response")
	}

	// Log the results for visibility
	t.Logf("Test passed! Animals received - Mammals: %v, Birds: %v, Fish: %v",
		result.Mammals, result.Birds, result.Fish)
}
