// Package genai_observability provides GenAI semantic conventions support
// for OpenTelemetry tracing, specifically optimized for Braintrust integration.
//
// This package complements the existing obs package by adding standardized
// GenAI semantic conventions while maintaining compatibility with the GAI
// framework's observability patterns.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/recera/gai/core"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// GenAISpanConfig configures GenAI semantic conventions for spans
type GenAISpanConfig struct {
	// Core GenAI attributes
	System    string // "openai", "groq", "anthropic", etc.
	Operation string // "chat_completion", "text_completion", etc.
	Model     string // "gpt-4", "llama-3.3-70b-versatile", etc.
	
	// Request parameters
	Temperature   *float32
	MaxTokens     *int
	TopP          *float32
	FrequencyPenalty *float32
	PresencePenalty  *float32
	
	// Content capture options
	CapturePrompts     bool // Whether to capture prompt content
	CaptureCompletions bool // Whether to capture completion content
	UseEvents         bool // Use OpenTelemetry events vs attributes
	
	// Metadata
	Metadata map[string]any
}

// SetGenAISpanAttributes sets GenAI semantic convention attributes on a span
// This function bridges the GAI framework's custom observability with standard GenAI conventions
func SetGenAISpanAttributes(span trace.Span, config GenAISpanConfig) {
	if span == nil {
		return
	}
	
	// Core GenAI semantic convention attributes
	attrs := []attribute.KeyValue{
		attribute.String("gen_ai.system", config.System),
		attribute.String("gen_ai.operation.name", config.Operation),
		attribute.String("gen_ai.request.model", config.Model),
	}
	
	// Optional request parameters
	if config.Temperature != nil {
		attrs = append(attrs, attribute.Float64("gen_ai.request.temperature", float64(*config.Temperature)))
	}
	if config.MaxTokens != nil {
		attrs = append(attrs, attribute.Int("gen_ai.request.max_tokens", *config.MaxTokens))
	}
	if config.TopP != nil {
		attrs = append(attrs, attribute.Float64("gen_ai.request.top_p", float64(*config.TopP)))
	}
	if config.FrequencyPenalty != nil {
		attrs = append(attrs, attribute.Float64("gen_ai.request.frequency_penalty", float64(*config.FrequencyPenalty)))
	}
	if config.PresencePenalty != nil {
		attrs = append(attrs, attribute.Float64("gen_ai.request.presence_penalty", float64(*config.PresencePenalty)))
	}
	
	// Add metadata as custom attributes
	for k, v := range config.Metadata {
		attrs = append(attrs, attribute.String(fmt.Sprintf("gen_ai.metadata.%s", k), fmt.Sprint(v)))
	}
	
	span.SetAttributes(attrs...)
}

// SetGenAIPromptAttributes sets prompt-related attributes following GenAI semantic conventions
// Supports both individual attributes and JSON serialization approaches
func SetGenAIPromptAttributes(span trace.Span, messages []core.Message, useJSON bool) error {
	if span == nil || len(messages) == 0 {
		return nil
	}
	
	if useJSON {
		return setGenAIPromptJSON(span, messages)
	}
	return setGenAIPromptIndividual(span, messages)
}

// setGenAIPromptJSON sets prompts as JSON-serialized attributes
func setGenAIPromptJSON(span trace.Span, messages []core.Message) error {
	type promptMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	
	var prompts []promptMessage
	for _, msg := range messages {
		content := extractTextContent(msg.Parts)
		if content != "" {
			prompts = append(prompts, promptMessage{
				Role:    string(msg.Role),
				Content: content,
			})
		}
	}
	
	if len(prompts) > 0 {
		promptJSON, err := json.Marshal(prompts)
		if err != nil {
			return fmt.Errorf("failed to marshal prompt JSON: %w", err)
		}
		span.SetAttributes(attribute.String("gen_ai.prompt_json", string(promptJSON)))
	}
	
	return nil
}

// setGenAIPromptIndividual sets prompts as individual indexed attributes
func setGenAIPromptIndividual(span trace.Span, messages []core.Message) error {
	var attrs []attribute.KeyValue
	
	for i, msg := range messages {
		content := extractTextContent(msg.Parts)
		if content != "" {
			attrs = append(attrs,
				attribute.String(fmt.Sprintf("gen_ai.prompt.%d.role", i), string(msg.Role)),
				attribute.String(fmt.Sprintf("gen_ai.prompt.%d.content", i), content),
			)
		}
	}
	
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	
	return nil
}

// AddGenAIPromptEvents adds prompt messages as OpenTelemetry events
// This approach provides timing information and is more robust for content capture
func AddGenAIPromptEvents(span trace.Span, messages []core.Message, system string) {
	if span == nil {
		return
	}
	
	for _, msg := range messages {
		content := extractTextContent(msg.Parts)
		if content == "" {
			continue
		}
		
		var eventName string
		switch msg.Role {
		case core.System:
			eventName = "gen_ai.system.message"
		case core.User:
			eventName = "gen_ai.user.message"  
		case core.Assistant:
			eventName = "gen_ai.assistant.message"
		case core.Tool:
			eventName = "gen_ai.tool.message"
		default:
			continue
		}
		
		span.AddEvent(eventName, trace.WithAttributes(
			attribute.String("gen_ai.system", system),
			attribute.String("gen_ai.message.role", string(msg.Role)),
			attribute.String("gen_ai.message.content", content),
		))
	}
}

// SetGenAICompletionAttributes sets completion-related attributes
func SetGenAICompletionAttributes(span trace.Span, result *core.TextResult, useJSON bool) error {
	if span == nil || result == nil {
		return nil
	}
	
	// Set completion content
	if result.Text != "" {
		if useJSON {
			completion := []map[string]string{
				{
					"role":    "assistant",
					"content": result.Text,
				},
			}
			completionJSON, err := json.Marshal(completion)
			if err != nil {
				return fmt.Errorf("failed to marshal completion JSON: %w", err)
			}
			span.SetAttributes(attribute.String("gen_ai.completion_json", string(completionJSON)))
		} else {
			span.SetAttributes(attribute.String("gen_ai.completion", result.Text))
		}
	}
	
	// Set usage attributes following GenAI semantic conventions
	if result.Usage.TotalTokens > 0 {
		span.SetAttributes(
			attribute.Int("gen_ai.usage.prompt_tokens", result.Usage.InputTokens),
			attribute.Int("gen_ai.usage.completion_tokens", result.Usage.OutputTokens),
			attribute.Int("gen_ai.usage.total_tokens", result.Usage.TotalTokens),
		)
	}
	
	// Set finish reason if available from steps
	if len(result.Steps) > 0 {
		lastStep := result.Steps[len(result.Steps)-1]
		if len(lastStep.ToolCalls) == 0 && lastStep.Text != "" {
			span.SetAttributes(attribute.String("gen_ai.completion.finish_reason", "stop"))
		} else if len(lastStep.ToolCalls) > 0 {
			span.SetAttributes(attribute.String("gen_ai.completion.finish_reason", "tool_calls"))
		}
	}
	
	return nil
}

// AddGenAICompletionEvent adds a completion event following GenAI semantic conventions
func AddGenAICompletionEvent(span trace.Span, result *core.TextResult, system string) {
	if span == nil || result == nil || result.Text == "" {
		return
	}
	
	span.AddEvent("gen_ai.choice", trace.WithAttributes(
		attribute.String("gen_ai.system", system),
		attribute.Int("gen_ai.choice.index", 0),
		attribute.String("gen_ai.choice.finish_reason", "stop"),
		attribute.String("gen_ai.choice.message.role", "assistant"),
		attribute.String("gen_ai.choice.message.content", result.Text),
	))
}

// SetGenAISpanName sets span name according to GenAI semantic conventions
// Format: "{operation} {model}" e.g., "chat_completion gpt-4"
func SetGenAISpanName(span trace.Span, operation, model string) {
	if span == nil {
		return
	}
	
	spanName := fmt.Sprintf("%s %s", operation, model)
	span.SetName(spanName)
}

// extractTextContent extracts text content from message parts
func extractTextContent(parts []core.Part) string {
	var textParts []string
	
	for _, part := range parts {
		if text, ok := part.(core.Text); ok {
			textParts = append(textParts, text.Text)
		}
	}
	
	return strings.Join(textParts, " ")
}

// ConfigureGenAISpan configures a span with comprehensive GenAI semantic conventions
// This is the main function that orchestrates all GenAI attributes and events
func ConfigureGenAISpan(ctx context.Context, span trace.Span, request core.Request, config GenAISpanConfig) error {
	if span == nil {
		return fmt.Errorf("span cannot be nil")
	}
	
	// 1. Set span name according to GenAI conventions
	SetGenAISpanName(span, config.Operation, config.Model)
	
	// 2. Set core GenAI attributes
	SetGenAISpanAttributes(span, config)
	
	// 3. Set prompt attributes/events
	if config.CapturePrompts && len(request.Messages) > 0 {
		if config.UseEvents {
			AddGenAIPromptEvents(span, request.Messages, config.System)
		} else {
			if err := SetGenAIPromptAttributes(span, request.Messages, false); err != nil {
				return fmt.Errorf("failed to set prompt attributes: %w", err)
			}
		}
	}
	
	// 4. Set tool information if available
	if len(request.Tools) > 0 {
		toolNames := make([]string, len(request.Tools))
		for i, tool := range request.Tools {
			toolNames[i] = tool.Name()
		}
		span.SetAttributes(
			attribute.StringSlice("gen_ai.tools", toolNames),
			attribute.Int("gen_ai.tools.count", len(toolNames)),
		)
	}
	
	// 5. Set request-specific attributes from core.Request
	if request.Temperature > 0 && config.Temperature == nil {
		temp := request.Temperature
		config.Temperature = &temp
		span.SetAttributes(attribute.Float64("gen_ai.request.temperature", float64(temp)))
	}
	
	if request.MaxTokens > 0 && config.MaxTokens == nil {
		maxTokens := request.MaxTokens
		config.MaxTokens = &maxTokens
		span.SetAttributes(attribute.Int("gen_ai.request.max_tokens", maxTokens))
	}
	
	return nil
}

// ConfigureGenAICompletion configures completion-related GenAI attributes and events
func ConfigureGenAICompletion(span trace.Span, result *core.TextResult, config GenAISpanConfig) error {
	if span == nil || result == nil {
		return nil
	}
	
	// Set completion attributes
	if config.CaptureCompletions {
		if config.UseEvents {
			AddGenAICompletionEvent(span, result, config.System)
		} else {
			if err := SetGenAICompletionAttributes(span, result, false); err != nil {
				return fmt.Errorf("failed to set completion attributes: %w", err)
			}
		}
	}
	
	return nil
}

// BraintrustOptimizedConfig returns a GenAI configuration optimized for Braintrust
func BraintrustOptimizedConfig(system, model string) GenAISpanConfig {
	return GenAISpanConfig{
		System:             system,
		Operation:          "chat_completion", // Default operation
		Model:              model,
		CapturePrompts:     true, // Critical for Braintrust content display
		CaptureCompletions: true, // Critical for Braintrust content display
		UseEvents:          false, // Attributes are more reliable than events for Braintrust
		Metadata:           make(map[string]any),
	}
}