package prompts_test

import (
	"context"
	"embed"
	"testing"

	"github.com/recera/gai/core"
	"github.com/recera/gai/prompts"
)

// Test embedded filesystem
//
//go:embed testdata/*.tmpl
var integrationFS embed.FS

// TestCoreIntegration verifies that prompts work seamlessly with core types.
func TestCoreIntegration(t *testing.T) {
	// Create registry
	reg, err := prompts.NewRegistry(integrationFS)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Render a system prompt
	ctx := context.Background()
	data := map[string]any{
		"Audience": "developers",
		"Length":   "detailed",
		"Topics":   []string{"architecture", "performance", "security"},
	}

	systemPrompt, id, err := reg.Render(ctx, "summarize", "1.0.0", data)
	if err != nil {
		t.Fatalf("failed to render prompt: %v", err)
	}

	// Create a core.Request with the rendered prompt
	request := core.Request{
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   1000,
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: systemPrompt},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Analyze the GAI framework architecture"},
				},
			},
		},
		Metadata: map[string]any{
			"prompt.name":        id.Name,
			"prompt.version":     id.Version,
			"prompt.fingerprint": id.Fingerprint,
			"request.source":     "test",
		},
	}

	// Verify the request is properly constructed
	if len(request.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(request.Messages))
	}

	if request.Messages[0].Role != core.System {
		t.Errorf("first message role = %v, want %v", request.Messages[0].Role, core.System)
	}

	// Verify metadata contains prompt information
	if request.Metadata["prompt.name"] != id.Name {
		t.Errorf("metadata prompt.name = %v, want %v", request.Metadata["prompt.name"], id.Name)
	}

	if request.Metadata["prompt.version"] != id.Version {
		t.Errorf("metadata prompt.version = %v, want %v", request.Metadata["prompt.version"], id.Version)
	}

	if request.Metadata["prompt.fingerprint"] != id.Fingerprint {
		t.Errorf("metadata prompt.fingerprint = %v, want %v", request.Metadata["prompt.fingerprint"], id.Fingerprint)
	}
}

// TestMultiModalIntegration tests prompts with multimodal messages.
func TestMultiModalIntegration(t *testing.T) {
	reg, err := prompts.NewRegistry(integrationFS)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Render an analysis prompt
	ctx := context.Background()
	data := map[string]any{
		"Context": "Image analysis for quality control",
		"Parameters": map[string]any{
			"threshold": 0.95,
			"mode":      "strict",
		},
		"Focus": "defects",
	}

	analysisPrompt, id, err := reg.Render(ctx, "analyze", "1.0.0", data)
	if err != nil {
		t.Fatalf("failed to render prompt: %v", err)
	}

	// Create a multimodal request
	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: analysisPrompt},
					core.ImageURL{URL: "https://example.com/product.jpg"},
				},
			},
		},
		Metadata: map[string]any{
			"prompt.id": id,
		},
	}

	// Verify multimodal message construction
	if len(request.Messages[0].Parts) != 2 {
		t.Errorf("expected 2 parts, got %d", len(request.Messages[0].Parts))
	}

	// Check that both text and image parts are present
	hasText := false
	hasImage := false
	for _, part := range request.Messages[0].Parts {
		switch part.(type) {
		case core.Text:
			hasText = true
		case core.ImageURL:
			hasImage = true
		}
	}

	if !hasText {
		t.Error("message missing text part")
	}
	if !hasImage {
		t.Error("message missing image part")
	}
}

// TestStreamingPromptIntegration tests using prompts with streaming responses.
func TestStreamingPromptIntegration(t *testing.T) {
	reg, err := prompts.NewRegistry(integrationFS)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Render a greeting prompt
	ctx := context.Background()
	greetingPrompt, _, err := reg.Render(ctx, "greet", "1.0.0", map[string]any{
		"Name": "Developer",
	})
	if err != nil {
		t.Fatalf("failed to render prompt: %v", err)
	}

	// Create a streaming request
	request := core.Request{
		Stream: true,
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: greetingPrompt},
				},
			},
		},
		StopWhen: core.MaxSteps(3),
	}

	// Verify streaming is enabled
	if !request.Stream {
		t.Error("streaming should be enabled")
	}

	// Verify stop condition is set
	if request.StopWhen == nil {
		t.Error("stop condition should be set")
	}

	// Test stop condition
	testStep := core.Step{
		Text: "Hello, Developer!",
	}
	
	// The stop condition should not trigger on first step
	if request.StopWhen.ShouldStop(1, testStep) {
		t.Error("stop condition should not trigger on first step")
	}

	// Should trigger after max steps
	if !request.StopWhen.ShouldStop(3, testStep) {
		t.Error("stop condition should trigger after max steps")
	}
}

// TestPromptVersionManagement tests version management with core metadata.
func TestPromptVersionManagement(t *testing.T) {
	reg, err := prompts.NewRegistry(integrationFS)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	ctx := context.Background()
	versions := []string{"1.0.0", "1.1.0"}
	
	requests := make([]core.Request, 0, len(versions))
	
	for _, version := range versions {
		data := map[string]any{"Name": "Test"}
		
		// Skip if version doesn't exist
		prompt, id, err := reg.Render(ctx, "greet", version, data)
		if err != nil {
			continue
		}
		
		req := core.Request{
			Messages: []core.Message{
				{
					Role: core.System,
					Parts: []core.Part{
						core.Text{Text: prompt},
					},
				},
			},
			Metadata: map[string]any{
				"prompt.name":        id.Name,
				"prompt.version":     id.Version,
				"prompt.fingerprint": id.Fingerprint,
			},
		}
		
		requests = append(requests, req)
	}
	
	// Verify we have at least one version
	if len(requests) == 0 {
		t.Error("expected at least one version to be rendered")
	}
	
	// Verify different versions have different fingerprints
	if len(requests) >= 2 {
		fp1 := requests[0].Metadata["prompt.fingerprint"]
		fp2 := requests[1].Metadata["prompt.fingerprint"]
		
		if fp1 == fp2 {
			t.Error("different versions should have different fingerprints")
		}
	}
}

// TestErrorPropagation tests that prompt errors are properly handled.
func TestErrorPropagation(t *testing.T) {
	reg, err := prompts.NewRegistry(integrationFS)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	ctx := context.Background()
	
	// Try to render non-existent template
	_, _, err = reg.Render(ctx, "nonexistent", "1.0.0", nil)
	if err == nil {
		t.Error("expected error for non-existent template")
	}

	// The error should be suitable for wrapping in core.AIError
	aiErr := &core.AIError{
		Category:  core.ErrorCategoryBadRequest,
		Code:      "TEMPLATE_NOT_FOUND",
		Message:   "prompt template not found",
		Provider:  "prompts",
		Cause:     err,
		Retryable: false,
	}

	if aiErr.Error() == "" {
		t.Error("AI error should have a message")
	}

	if aiErr.Category != core.ErrorCategoryBadRequest {
		t.Error("AI error should have correct category")
	}
}

// BenchmarkIntegratedRender benchmarks rendering prompts for use in requests.
func BenchmarkIntegratedRender(b *testing.B) {
	reg, err := prompts.NewRegistry(integrationFS)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	data := map[string]any{
		"Name": "Benchmark",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		prompt, id, err := reg.Render(ctx, "greet", "1.0.0", data)
		if err != nil {
			b.Fatal(err)
		}

		request := core.Request{
			Messages: []core.Message{
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: prompt},
					},
				},
			},
			Metadata: map[string]any{
				"prompt.id": id,
			},
		}

		_ = request
	}
}