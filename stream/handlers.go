// Package stream provides streaming utilities for AI responses.
// This file implements normalized and passthrough streaming handlers.
package stream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/recera/gai/core"
)

// StreamMode defines how events are formatted for transmission.
type StreamMode int

const (
	// ModeNormalized uses the stable gai.events.v1 format (default)
	ModeNormalized StreamMode = iota
	// ModePassthrough preserves provider-specific format
	ModePassthrough
)

// StreamConfig configures streaming behavior.
type StreamConfig struct {
	// Mode determines event formatting
	Mode StreamMode
	// RequestID for the stream
	RequestID string
	// TraceID for distributed tracing
	TraceID string
	// Provider name for metadata
	Provider string
	// Model name for metadata
	Model string
	// Options for SSE or NDJSON
	SSEOptions    *SSEOptions
	NDJSONOptions *NDJSONOptions
}

// SSENormalized streams events in normalized gai.events.v1 format via SSE.
func SSENormalized(w http.ResponseWriter, stream core.TextStream, config StreamConfig) error {
	// Create normalizer
	normalizer := NewNormalizer(config.RequestID, config.TraceID).
		WithProvider(config.Provider).
		WithModel(config.Model)

	// Create normalized stream
	normalizedStream := NewNormalizedStream(stream, normalizer)
	defer normalizedStream.Close()

	// Set SSE headers
	setSSEHeaders(w)

	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported: ResponseWriter does not support Flusher")
	}

	// Stream normalized events
	for event := range normalizedStream.Events() {
		// Convert to SSE format
		if err := writeNormalizedSSEEvent(w, event, flusher); err != nil {
			return err
		}
	}

	// Send completion
	fmt.Fprint(w, "event: done\n")
	fmt.Fprint(w, "data: {\"type\":\"done\",\"finished\":true}\n\n")
	flusher.Flush()

	return nil
}

// SSEPassthroughOpenAI streams events in OpenAI-compatible format via SSE.
func SSEPassthroughOpenAI(w http.ResponseWriter, stream core.TextStream, config StreamConfig) error {
	// Set SSE headers
	setSSEHeaders(w)

	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported: ResponseWriter does not support Flusher")
	}

	// Stream in OpenAI format
	for event := range stream.Events() {
		if err := writeOpenAISSEEvent(w, event, flusher); err != nil {
			return err
		}
	}

	// Send OpenAI-style done marker
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()

	return nil
}

// NDJSONNormalized streams events in normalized gai.events.v1 format via NDJSON.
func NDJSONNormalized(w http.ResponseWriter, stream core.TextStream, config StreamConfig) error {
	// Create normalizer
	normalizer := NewNormalizer(config.RequestID, config.TraceID).
		WithProvider(config.Provider).
		WithModel(config.Model)

	// Create normalized stream
	normalizedStream := NewNormalizedStream(stream, normalizer)
	defer normalizedStream.Close()

	// Set NDJSON headers
	setNDJSONHeaders(w)

	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported: ResponseWriter does not support Flusher")
	}

	// Create encoder
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	// Stream normalized events
	for event := range normalizedStream.Events() {
		if err := encoder.Encode(event); err != nil {
			return err
		}
		flusher.Flush()
	}

	// Send completion
	completion := map[string]any{
		"type":     "done",
		"finished": true,
	}
	encoder.Encode(completion)
	flusher.Flush()

	return nil
}

// NDJSONPassthroughOpenAI streams events in OpenAI-compatible format via NDJSON.
func NDJSONPassthroughOpenAI(w http.ResponseWriter, stream core.TextStream, config StreamConfig) error {
	// Set NDJSON headers
	setNDJSONHeaders(w)

	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported: ResponseWriter does not support Flusher")
	}

	// Create encoder
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	// Stream in OpenAI format
	for event := range stream.Events() {
		openAIEvent := convertToOpenAIFormat(event)
		if openAIEvent != nil {
			if err := encoder.Encode(openAIEvent); err != nil {
				return err
			}
			flusher.Flush()
		}
	}

	// Send OpenAI-style done marker
	encoder.Encode(map[string]string{"object": "done"})
	flusher.Flush()

	return nil
}

// UniversalHandler creates an HTTP handler that supports both normalized and passthrough modes.
func UniversalHandler(provider core.Provider, prepareRequest func(*http.Request) (core.Request, StreamConfig, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Prepare request and config
		req, config, err := prepareRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Ensure streaming is enabled
		req.Stream = true

		// Auto-generate RequestID if not provided
		if req.RequestID == "" {
			gen := &DefaultRequestIDGenerator{}
			req.RequestID = gen.Generate()
		}

		// Get stream from provider
		stream, err := provider.StreamText(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer stream.Close()

		// Update config with request metadata
		config.RequestID = req.RequestID
		if config.Model == "" && req.Model != "" {
			config.Model = req.Model
		}

		// Determine format from Accept header or path
		format := detectFormat(r)

		// Stream based on mode and format
		switch {
		case format == "sse" && config.Mode == ModeNormalized:
			err = SSENormalized(w, stream, config)
		case format == "sse" && config.Mode == ModePassthrough:
			err = SSEPassthroughOpenAI(w, stream, config)
		case format == "ndjson" && config.Mode == ModeNormalized:
			err = NDJSONNormalized(w, stream, config)
		case format == "ndjson" && config.Mode == ModePassthrough:
			err = NDJSONPassthroughOpenAI(w, stream, config)
		default:
			// Default to normalized SSE
			err = SSENormalized(w, stream, config)
		}

		if err != nil {
			// Log error but don't write (headers already sent)
			_ = err
		}
	}
}

// Helper functions

func setSSEHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Idempotency-Key")
}

func setNDJSONHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "application/x-ndjson")
	h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	h.Set("Transfer-Encoding", "chunked")
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Idempotency-Key")
}

func detectFormat(r *http.Request) string {
	// Check Accept header
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/event-stream") {
		return "sse"
	}
	if strings.Contains(accept, "application/x-ndjson") || strings.Contains(accept, "application/json") {
		return "ndjson"
	}

	// Check path for hints
	if strings.Contains(r.URL.Path, "/events") || strings.Contains(r.URL.Path, "/sse") {
		return "sse"
	}

	// Default to SSE for browser compatibility
	return "sse"
}

func writeNormalizedSSEEvent(w http.ResponseWriter, event NormalizedEvent, flusher http.Flusher) error {
	// Use compact JSON for efficiency
	data := event.CompactJSON()

	// Write event type and data
	fmt.Fprintf(w, "event: %s\n", event.Type)
	fmt.Fprintf(w, "data: %s\n\n", data)

	flusher.Flush()
	return nil
}

func writeOpenAISSEEvent(w http.ResponseWriter, event core.Event, flusher http.Flusher) error {
	// Convert to OpenAI format
	openAIEvent := convertToOpenAIFormat(event)
	if openAIEvent == nil {
		return nil // Skip events that don't map
	}

	// Marshal to JSON
	data, err := json.Marshal(openAIEvent)
	if err != nil {
		return err
	}

	// Write in OpenAI SSE format
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	return nil
}

func convertToOpenAIFormat(event core.Event) map[string]any {
	// Map core events to OpenAI streaming format
	switch event.Type {
	case core.EventTextDelta:
		return map[string]any{
			"object": "chat.completion.chunk",
			"choices": []map[string]any{
				{
					"index": 0,
					"delta": map[string]any{
						"content": event.TextDelta,
					},
				},
			},
		}

	case core.EventToolCall:
		return map[string]any{
			"object": "chat.completion.chunk",
			"choices": []map[string]any{
				{
					"index": 0,
					"delta": map[string]any{
						"tool_calls": []map[string]any{
							{
								"id":   event.ToolID,
								"type": "function",
								"function": map[string]any{
									"name":      event.ToolName,
									"arguments": string(event.ToolInput),
								},
							},
						},
					},
				},
			},
		}

	case core.EventFinish:
		usage := map[string]any{}
		if event.Usage != nil {
			usage = map[string]any{
				"prompt_tokens":     event.Usage.InputTokens,
				"completion_tokens": event.Usage.OutputTokens,
				"total_tokens":      event.Usage.TotalTokens,
			}
		}
		return map[string]any{
			"object": "chat.completion.chunk",
			"choices": []map[string]any{
				{
					"index":         0,
					"finish_reason": "stop",
				},
			},
			"usage": usage,
		}

	default:
		// Events that don't map to OpenAI format
		return nil
	}
}

// OpenAICompatHandler creates a handler for the /v1/chat/completions endpoint.
func OpenAICompatHandler(provider core.Provider) http.HandlerFunc {
	return UniversalHandler(provider, func(r *http.Request) (core.Request, StreamConfig, error) {
		// Parse OpenAI-format request body
		var body struct {
			Model            string         `json:"model"`
			Messages         []core.Message `json:"messages"`
			Temperature      float32        `json:"temperature,omitempty"`
			MaxTokens        int            `json:"max_tokens,omitempty"`
			Stream           bool           `json:"stream,omitempty"`
			Tools            []any          `json:"tools,omitempty"`
			ToolChoice       any            `json:"tool_choice,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return core.Request{}, StreamConfig{}, err
		}

		// Check for idempotency key
		idempotencyKey := r.Header.Get("X-Idempotency-Key")
		if idempotencyKey == "" {
			idempotencyKey = r.Header.Get("Idempotency-Key")
		}

		// Build core request
		req := core.Request{
			IdempotencyKey: idempotencyKey,
			Model:          body.Model,
			Messages:       body.Messages,
			Temperature:    body.Temperature,
			MaxTokens:      body.MaxTokens,
			Stream:         body.Stream,
		}

		// Configure for passthrough mode
		config := StreamConfig{
			Mode:     ModePassthrough,
			Provider: "openai", // Or detect from model name
			Model:    body.Model,
			TraceID:  r.Header.Get("X-Trace-Id"),
		}

		return req, config, nil
	})
}