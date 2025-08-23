// Package stream provides streaming utilities for AI responses.
// This file implements Server-Sent Events (SSE) streaming for browser compatibility.
package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/recera/gai/core"
)

// SSEOptions configures SSE streaming behavior.
type SSEOptions struct {
	// HeartbeatInterval for keep-alive messages (default: 15s)
	HeartbeatInterval time.Duration
	// FlushAfterWrite forces flush after each write (default: true)
	FlushAfterWrite bool
	// MaxRetries for client reconnection hints
	MaxRetries int
	// BufferSize for the write buffer
	BufferSize int
	// IncludeID adds event IDs for client-side replay
	IncludeID bool
}

// DefaultSSEOptions returns sensible defaults for SSE streaming.
func DefaultSSEOptions() SSEOptions {
	return SSEOptions{
		HeartbeatInterval: 15 * time.Second,
		FlushAfterWrite:   true,
		MaxRetries:        3,
		BufferSize:        4096,
		IncludeID:         false,
	}
}

// SSE writes a TextStream as Server-Sent Events to an HTTP response.
func SSE(w http.ResponseWriter, stream core.TextStream, opts ...SSEOptions) error {
	options := DefaultSSEOptions()
	if len(opts) > 0 {
		options = opts[0]
	}
	
	writer := &sseWriter{
		w:       w,
		options: options,
		eventID: 0,
	}
	
	return writer.Write(stream)
}

// sseWriter handles SSE protocol details.
type sseWriter struct {
	w       http.ResponseWriter
	options SSEOptions
	eventID int64
	mu      sync.Mutex
}

// Write streams events to the HTTP response.
func (s *sseWriter) Write(stream core.TextStream) error {
	// Set SSE headers
	s.setHeaders()
	
	// Get flusher for real-time streaming
	flusher, ok := s.w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported: ResponseWriter does not support Flusher")
	}
	
	// Create context for managing goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start heartbeat goroutine
	heartbeatDone := make(chan struct{})
	go s.sendHeartbeats(ctx, flusher, heartbeatDone)
	
	// Channel for errors from event processing
	errChan := make(chan error, 1)
	
	// Process events
	go func() {
		defer close(heartbeatDone)
		
		for event := range stream.Events() {
			if err := s.writeEvent(event, flusher); err != nil {
				select {
				case errChan <- err:
				default:
				}
				return
			}
		}
		
		// Send completion event
		if err := s.writeCompletion(flusher); err != nil {
			select {
			case errChan <- err:
			default:
			}
		}
	}()
	
	// Wait for completion or error
	select {
	case err := <-errChan:
		return err
	case <-heartbeatDone:
		// Normal completion
		return nil
	}
}

// setHeaders sets the appropriate SSE headers.
func (s *sseWriter) setHeaders() {
	h := s.w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no") // Disable Nginx buffering
	
	// CORS headers for browser compatibility
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// sendHeartbeats sends periodic keep-alive messages.
func (s *sseWriter) sendHeartbeats(ctx context.Context, flusher http.Flusher, done chan struct{}) {
	ticker := time.NewTicker(s.options.HeartbeatInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			s.mu.Lock()
			fmt.Fprint(s.w, ": keep-alive\n\n")
			flusher.Flush()
			s.mu.Unlock()
		}
	}
}

// writeEvent writes a single event to the stream.
func (s *sseWriter) writeEvent(event core.Event, flusher http.Flusher) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Increment event ID
	s.eventID++
	
	// Convert event to SSE format
	sseEvent := s.eventToSSE(event)
	
	// Write event ID if configured
	if s.options.IncludeID {
		if _, err := fmt.Fprintf(s.w, "id: %d\n", s.eventID); err != nil {
			return err
		}
	}
	
	// Write event type
	if _, err := fmt.Fprintf(s.w, "event: %s\n", sseEvent.EventType); err != nil {
		return err
	}
	
	// Write retry hint for errors
	if event.Type == core.EventError && s.options.MaxRetries > 0 {
		if _, err := fmt.Fprintf(s.w, "retry: %d\n", 5000); err != nil {
			return err
		}
	}
	
	// Marshal and write data
	data, err := json.Marshal(sseEvent.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}
	
	// Flush if configured
	if s.options.FlushAfterWrite {
		flusher.Flush()
	}
	
	return nil
}

// writeCompletion writes the final completion event.
func (s *sseWriter) writeCompletion(flusher http.Flusher) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.eventID++
	
	if s.options.IncludeID {
		fmt.Fprintf(s.w, "id: %d\n", s.eventID)
	}
	
	fmt.Fprint(s.w, "event: done\n")
	fmt.Fprint(s.w, "data: {\"finished\":true}\n\n")
	
	flusher.Flush()
	return nil
}

// sseEventData represents the data structure sent in SSE events.
type sseEventData struct {
	EventType string `json:"-"`
	Data      any    `json:"data"`
}

// eventToSSE converts a core.Event to SSE format.
func (s *sseWriter) eventToSSE(event core.Event) sseEventData {
	// Map event types to SSE event names
	eventType := event.Type.String()
	
	// Build data payload based on event type
	var data any
	
	switch event.Type {
	case core.EventTextDelta:
		data = map[string]any{
			"text": event.TextDelta,
		}
	case core.EventAudioDelta:
		data = map[string]any{
			"audio":  event.AudioChunk,
			"format": event.AudioFormat,
		}
	case core.EventToolCall:
		data = map[string]any{
			"tool_name": event.ToolName,
			"tool_id":   event.ToolID,
			"input":     event.ToolInput,
		}
	case core.EventToolResult:
		data = map[string]any{
			"tool_name": event.ToolName,
			"result":    event.ToolResult,
		}
	case core.EventCitations:
		data = map[string]any{
			"citations": event.Citations,
		}
	case core.EventSafety:
		data = map[string]any{
			"safety": event.Safety,
		}
	case core.EventFinishStep:
		data = map[string]any{
			"step_number": event.StepNumber,
		}
	case core.EventFinish:
		data = map[string]any{
			"usage": event.Usage,
		}
	case core.EventError:
		errorMsg := ""
		if event.Err != nil {
			errorMsg = event.Err.Error()
		}
		data = map[string]any{
			"error": errorMsg,
		}
		eventType = "error"
	case core.EventStart:
		data = map[string]any{
			"started": true,
		}
	default:
		data = map[string]any{
			"raw": event.Raw,
		}
	}
	
	return sseEventData{
		EventType: eventType,
		Data:      data,
	}
}

// SSEHandler creates an HTTP handler that streams AI responses as SSE.
func SSEHandler(provider core.Provider, prepareRequest func(*http.Request) (core.Request, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Prepare the AI request
		req, err := prepareRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		// Ensure streaming is enabled
		req.Stream = true
		
		// Get stream from provider
		stream, err := provider.StreamText(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer stream.Close()
		
		// Stream as SSE
		if err := SSE(w, stream); err != nil {
			// Log error but don't write to response (headers already sent)
			// In production, this should use proper logging
			_ = err
		}
	}
}

// Writer provides a low-level SSE writer interface.
type Writer struct {
	w       io.Writer
	flusher http.Flusher
	mu      sync.Mutex
}

// NewWriter creates a new SSE writer.
func NewWriter(w io.Writer) *Writer {
	flusher, _ := w.(http.Flusher)
	return &Writer{
		w:       w,
		flusher: flusher,
	}
}

// WriteEvent writes a raw SSE event.
func (w *Writer) WriteEvent(event, data string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if event != "" {
		if _, err := fmt.Fprintf(w.w, "event: %s\n", event); err != nil {
			return err
		}
	}
	
	if _, err := fmt.Fprintf(w.w, "data: %s\n\n", data); err != nil {
		return err
	}
	
	if w.flusher != nil {
		w.flusher.Flush()
	}
	
	return nil
}

// WriteComment writes an SSE comment (for keep-alive).
func (w *Writer) WriteComment(comment string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if _, err := fmt.Fprintf(w.w, ": %s\n\n", comment); err != nil {
		return err
	}
	
	if w.flusher != nil {
		w.flusher.Flush()
	}
	
	return nil
}