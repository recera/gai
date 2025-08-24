// +build integration

package openai_compat

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/tools"
)

// Integration tests for OpenAI-compatible providers.
// Run with: go test -tags=integration -v ./providers/openai_compat
// Requires valid API keys in environment variables.

func skipIfNoAPIKey(t *testing.T, envVar string) {
	if os.Getenv(envVar) == "" {
		t.Skipf("Skipping test: %s not set", envVar)
	}
}

func TestGroqIntegration(t *testing.T) {
	skipIfNoAPIKey(t, "GROQ_API_KEY")
	
	ctx := context.Background()
	provider, err := Groq()
	if err != nil {
		t.Fatalf("Failed to create Groq provider: %v", err)
	}
	
	t.Run("TextGeneration", func(t *testing.T) {
		result, err := provider.GenerateText(ctx, core.Request{
			Messages: []core.Message{
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: "Say 'Hello from Groq' and nothing else."},
					},
				},
			},
			Temperature: 0.1,
			MaxTokens:   20,
		})
		
		if err != nil {
			t.Fatalf("GenerateText failed: %v", err)
		}
		
		if !strings.Contains(strings.ToLower(result.Text), "groq") {
			t.Errorf("Expected response to contain 'groq', got: %s", result.Text)
		}
		
		t.Logf("Response: %s", result.Text)
		t.Logf("Tokens: %+v", result.Usage)
	})
	
	t.Run("Streaming", func(t *testing.T) {
		stream, err := provider.StreamText(ctx, core.Request{
			Messages: []core.Message{
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: "Count from 1 to 5."},
					},
				},
			},
			Stream:    true,
			MaxTokens: 50,
		})
		
		if err != nil {
			t.Fatalf("StreamText failed: %v", err)
		}
		defer stream.Close()
		
		var fullText strings.Builder
		eventCount := 0
		
		for event := range stream.Events() {
			switch event.Type {
			case core.EventTextDelta:
				fullText.WriteString(event.TextDelta)
				eventCount++
			case core.EventError:
				t.Fatalf("Stream error: %v", event.Err)
			}
		}
		
		if eventCount == 0 {
			t.Error("No streaming events received")
		}
		
		t.Logf("Streamed text: %s", fullText.String())
		t.Logf("Event count: %d", eventCount)
	})
	
	t.Run("ToolCalling", func(t *testing.T) {
		type MathInput struct {
			A int `json:"a"`
			B int `json:"b"`
		}
		type MathOutput struct {
			Sum int `json:"sum"`
		}
		
		addTool := tools.New[MathInput, MathOutput](
			"add",
			"Add two numbers",
			func(ctx context.Context, in MathInput, meta tools.Meta) (MathOutput, error) {
				return MathOutput{Sum: in.A + in.B}, nil
			},
		)
		
		result, err := provider.GenerateText(ctx, core.Request{
			Messages: []core.Message{
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: "What is 42 plus 58?"},
					},
				},
			},
			Tools:      []core.ToolHandle{tools.NewCoreAdapter(addTool)},
			ToolChoice: core.ToolAuto,
			StopWhen:   core.MaxSteps(2),
		})
		
		if err != nil {
			t.Fatalf("Tool calling failed: %v", err)
		}
		
		if !strings.Contains(result.Text, "100") {
			t.Errorf("Expected result to contain '100', got: %s", result.Text)
		}
		
		if len(result.Steps) == 0 {
			t.Error("Expected at least one step with tool execution")
		}
		
		t.Logf("Final answer: %s", result.Text)
		t.Logf("Steps: %d", len(result.Steps))
	})
}

func TestXAIIntegration(t *testing.T) {
	skipIfNoAPIKey(t, "XAI_API_KEY")
	
	ctx := context.Background()
	provider, err := XAI()
	if err != nil {
		t.Fatalf("Failed to create xAI provider: %v", err)
	}
	
	t.Run("TextGeneration", func(t *testing.T) {
		result, err := provider.GenerateText(ctx, core.Request{
			Messages: []core.Message{
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: "What is xAI? Answer in one sentence."},
					},
				},
			},
			MaxTokens: 50,
		})
		
		if err != nil {
			t.Fatalf("GenerateText failed: %v", err)
		}
		
		t.Logf("Response: %s", result.Text)
		t.Logf("Model used: grok model")
	})
}

func TestCerebrasIntegration(t *testing.T) {
	skipIfNoAPIKey(t, "CEREBRAS_API_KEY")
	
	ctx := context.Background()
	provider, err := Cerebras()
	if err != nil {
		t.Fatalf("Failed to create Cerebras provider: %v", err)
	}
	
	t.Run("FastInference", func(t *testing.T) {
		start := time.Now()
		
		result, err := provider.GenerateText(ctx, core.Request{
			Messages: []core.Message{
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: "Say 'Speed test complete' and nothing else."},
					},
				},
			},
			Temperature: 0,
			MaxTokens:   10,
		})
		
		duration := time.Since(start)
		
		if err != nil {
			t.Fatalf("GenerateText failed: %v", err)
		}
		
		t.Logf("Response: %s", result.Text)
		t.Logf("Inference time: %v", duration)
		
		// Cerebras is known for very fast inference
		if duration > 2*time.Second {
			t.Logf("Warning: Cerebras took longer than expected: %v", duration)
		}
	})
	
	t.Run("NoJSONStreaming", func(t *testing.T) {
		// Cerebras doesn't support JSON streaming
		// The adapter should handle this gracefully
		stream, err := provider.StreamText(ctx, core.Request{
			Messages: []core.Message{
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: "Count to 3."},
					},
				},
			},
			Stream:    true,
			MaxTokens: 20,
		})
		
		if err != nil {
			t.Fatalf("StreamText failed: %v", err)
		}
		defer stream.Close()
		
		// Should fall back to simulated streaming
		eventCount := 0
		for event := range stream.Events() {
			if event.Type == core.EventTextDelta || event.Type == core.EventFinish {
				eventCount++
			}
		}
		
		if eventCount == 0 {
			t.Error("No events received from simulated stream")
		}
		
		t.Logf("Simulated stream events: %d", eventCount)
	})
}

func TestTogetherIntegration(t *testing.T) {
	skipIfNoAPIKey(t, "TOGETHER_API_KEY")
	
	ctx := context.Background()
	provider, err := Together()
	if err != nil {
		t.Fatalf("Failed to create Together provider: %v", err)
	}
	
	t.Run("ModelVariety", func(t *testing.T) {
		models := []string{
			"meta-llama/Llama-3.3-70B-Instruct-Turbo",
			"mistralai/Mixtral-8x7B-Instruct-v0.1",
		}
		
		for _, model := range models {
			t.Run(model, func(t *testing.T) {
				provider, err := Together(WithModel(model))
				if err != nil {
					t.Fatalf("Failed to create provider with model %s: %v", model, err)
				}
				
				result, err := provider.GenerateText(ctx, core.Request{
					Messages: []core.Message{
						{
							Role: core.User,
							Parts: []core.Part{
								core.Text{Text: "Say 'Hello from Together' and nothing else."},
							},
						},
					},
					MaxTokens: 20,
				})
				
				if err != nil {
					// Some models might not be available
					if strings.Contains(err.Error(), "model_not_found") {
						t.Skipf("Model %s not available", model)
					}
					t.Fatalf("GenerateText failed for %s: %v", model, err)
				}
				
				t.Logf("Model %s response: %s", model, result.Text)
			})
		}
	})
}

func TestErrorHandlingIntegration(t *testing.T) {
	ctx := context.Background()
	
	t.Run("InvalidAPIKey", func(t *testing.T) {
		provider, err := New(CompatOpts{
			BaseURL:      "https://api.groq.com/openai/v1",
			APIKey:       "invalid-key-12345",
			ProviderName: "groq",
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}
		
		_, err = provider.GenerateText(ctx, core.Request{
			Messages: []core.Message{
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: "Test"},
					},
				},
			},
		})
		
		if err == nil {
			t.Fatal("Expected error for invalid API key")
		}
		
		if !core.IsUnauthorized(err) {
			t.Errorf("Expected unauthorized error, got: %v", err)
		}
	})
	
	t.Run("NonExistentModel", func(t *testing.T) {
		skipIfNoAPIKey(t, "GROQ_API_KEY")
		
		provider, err := Groq(WithModel("non-existent-model-xyz"))
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}
		
		_, err = provider.GenerateText(ctx, core.Request{
			Messages: []core.Message{
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: "Test"},
					},
				},
			},
		})
		
		if err == nil {
			t.Fatal("Expected error for non-existent model")
		}
		
		// Could be NotFound or InvalidRequest depending on provider
		if !core.IsNotFound(err) && !core.IsInvalidRequest(err) {
			t.Errorf("Expected not found or invalid request error, got: %v", err)
		}
	})
}

func TestCapabilityProbing(t *testing.T) {
	skipIfNoAPIKey(t, "GROQ_API_KEY")
	
	provider, err := Groq()
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	
	// Wait for async probing
	time.Sleep(500 * time.Millisecond)
	
	caps := provider.GetCapabilities()
	if caps == nil {
		t.Skip("Capabilities not available")
	}
	
	t.Logf("Capabilities for Groq:")
	t.Logf("  Models: %d available", len(caps.Models))
	t.Logf("  Supports Tools: %v", caps.SupportsTools)
	t.Logf("  Supports Streaming: %v", caps.SupportsStreaming)
	t.Logf("  Supports JSON Mode: %v", caps.SupportsJSONMode)
	t.Logf("  Max Context Window: %d", caps.MaxContextWindow)
	
	if len(caps.Models) > 0 {
		t.Logf("  First model: %s", caps.Models[0].ID)
	}
}