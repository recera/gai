// Package stream provides streaming utilities for AI responses.
// This file implements NDJSON (Newline Delimited JSON) streaming.
package stream

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/recera/gai/core"
)

// NDJSONOptions configures NDJSON streaming behavior.
type NDJSONOptions struct {
	// BufferSize for write operations
	BufferSize int
	// FlushInterval for periodic flushing
	FlushInterval time.Duration
	// CompactJSON removes unnecessary whitespace
	CompactJSON bool
	// IncludeTimestamp adds timestamps to each line
	IncludeTimestamp bool
}

// DefaultNDJSONOptions returns sensible defaults for NDJSON streaming.
func DefaultNDJSONOptions() NDJSONOptions {
	return NDJSONOptions{
		BufferSize:       8192,
		FlushInterval:    100 * time.Millisecond,
		CompactJSON:      true,
		IncludeTimestamp: false,
	}
}

// NDJSON writes a TextStream as newline-delimited JSON to an HTTP response.
func NDJSON(w http.ResponseWriter, stream core.TextStream, opts ...NDJSONOptions) error {
	options := DefaultNDJSONOptions()
	if len(opts) > 0 {
		options = opts[0]
	}
	
	writer := &ndjsonWriter{
		w:       w,
		options: options,
	}
	
	return writer.Write(stream)
}

// ndjsonWriter handles NDJSON protocol details.
type ndjsonWriter struct {
	w       http.ResponseWriter
	options NDJSONOptions
	mu      sync.Mutex
	encoder *json.Encoder
	buffer  *bufio.Writer
}

// Write streams events to the HTTP response as NDJSON.
func (n *ndjsonWriter) Write(stream core.TextStream) error {
	// Set NDJSON headers
	n.setHeaders()
	
	// Get flusher for real-time streaming
	flusher, ok := n.w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported: ResponseWriter does not support Flusher")
	}
	
	// Create buffered writer
	n.buffer = bufio.NewWriterSize(n.w, n.options.BufferSize)
	n.encoder = json.NewEncoder(n.buffer)
	
	// Disable HTML escaping for cleaner output
	n.encoder.SetEscapeHTML(false)
	
	// Create context for flush timer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start flush timer if configured
	var flushDone chan struct{}
	if n.options.FlushInterval > 0 {
		flushDone = make(chan struct{})
		go n.periodicFlush(ctx, flusher, flushDone)
	}
	
	// Process events
	for event := range stream.Events() {
		if err := n.writeEvent(event); err != nil {
			return err
		}
		
		// Flush after each event for low latency
		n.mu.Lock()
		n.buffer.Flush()
		flusher.Flush()
		n.mu.Unlock()
	}
	
	// Write final completion event
	if err := n.writeCompletion(); err != nil {
		return err
	}
	
	// Final flush
	n.mu.Lock()
	n.buffer.Flush()
	flusher.Flush()
	n.mu.Unlock()
	
	// Clean up flush timer
	if flushDone != nil {
		cancel()
		<-flushDone
	}
	
	return nil
}

// setHeaders sets the appropriate NDJSON headers.
func (n *ndjsonWriter) setHeaders() {
	h := n.w.Header()
	h.Set("Content-Type", "application/x-ndjson")
	h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no") // Disable Nginx buffering
	h.Set("Transfer-Encoding", "chunked")
	
	// CORS headers for browser compatibility
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// periodicFlush flushes the buffer at regular intervals.
func (n *ndjsonWriter) periodicFlush(ctx context.Context, flusher http.Flusher, done chan struct{}) {
	defer close(done)
	
	ticker := time.NewTicker(n.options.FlushInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n.mu.Lock()
			n.buffer.Flush()
			flusher.Flush()
			n.mu.Unlock()
		}
	}
}

// writeEvent writes a single event as a JSON line.
func (n *ndjsonWriter) writeEvent(event core.Event) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	// Convert event to NDJSON format
	line := n.eventToNDJSON(event)
	
	// Encode and write
	if err := n.encoder.Encode(line); err != nil {
		return fmt.Errorf("failed to encode event: %w", err)
	}
	
	return nil
}

// writeCompletion writes the final completion line.
func (n *ndjsonWriter) writeCompletion() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	completion := map[string]any{
		"type":     "done",
		"finished": true,
	}
	
	if n.options.IncludeTimestamp {
		completion["timestamp"] = time.Now().Unix()
	}
	
	return n.encoder.Encode(completion)
}

// eventToNDJSON converts a core.Event to NDJSON format.
func (n *ndjsonWriter) eventToNDJSON(event core.Event) map[string]any {
	line := make(map[string]any)
	
	// Always include event type
	line["type"] = event.Type.String()
	
	// Add timestamp if configured
	if n.options.IncludeTimestamp {
		line["timestamp"] = event.Timestamp.Unix()
	}
	
	// Add event-specific fields
	switch event.Type {
	case core.EventTextDelta:
		line["text"] = event.TextDelta
		
	case core.EventAudioDelta:
		line["audio"] = map[string]any{
			"chunk":  event.AudioChunk,
			"format": event.AudioFormat,
		}
		
	case core.EventToolCall:
		line["tool_call"] = map[string]any{
			"name":  event.ToolName,
			"id":    event.ToolID,
			"input": event.ToolInput,
		}
		
	case core.EventToolResult:
		line["tool_result"] = map[string]any{
			"name":   event.ToolName,
			"result": event.ToolResult,
		}
		
	case core.EventCitations:
		line["citations"] = event.Citations
		
	case core.EventSafety:
		line["safety"] = event.Safety
		
	case core.EventFinishStep:
		line["step"] = map[string]any{
			"number":   event.StepNumber,
			"finished": true,
		}
		
	case core.EventFinish:
		line["usage"] = event.Usage
		line["finished"] = true
		
	case core.EventError:
		errorMsg := ""
		if event.Err != nil {
			errorMsg = event.Err.Error()
		}
		line["error"] = errorMsg
		
	case core.EventStart:
		line["started"] = true
		
	default:
		if event.Raw != nil {
			line["raw"] = event.Raw
		}
	}
	
	return line
}

// NDJSONHandler creates an HTTP handler that streams AI responses as NDJSON.
func NDJSONHandler(provider core.Provider, prepareRequest func(*http.Request) (core.Request, error)) http.HandlerFunc {
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
		
		// Stream as NDJSON
		if err := NDJSON(w, stream); err != nil {
			// Log error but don't write to response (headers already sent)
			// In production, this should use proper logging
			_ = err
		}
	}
}

// Reader provides NDJSON reading capabilities.
type Reader struct {
	scanner *bufio.Scanner
	decoder *json.Decoder
}

// NewReader creates a new NDJSON reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		scanner: bufio.NewScanner(r),
	}
}

// Read reads the next JSON object from the stream.
func (r *Reader) Read(v any) error {
	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return err
		}
		return io.EOF
	}
	
	return json.Unmarshal(r.scanner.Bytes(), v)
}

// StreamToChannel converts an NDJSON reader to an event channel.
func StreamToChannel(ctx context.Context, r io.Reader) (<-chan core.Event, error) {
	events := make(chan core.Event, 100)
	reader := NewReader(r)
	
	go func() {
		defer close(events)
		
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			
			var line map[string]any
			if err := reader.Read(&line); err != nil {
				if err != io.EOF {
					events <- core.Event{
						Type:      core.EventError,
						Err:       err,
						Timestamp: time.Now(),
					}
				}
				return
			}
			
			// Convert line to event
			event := ndjsonToEvent(line)
			
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		}
	}()
	
	return events, nil
}

// ndjsonToEvent converts an NDJSON line to a core.Event.
func ndjsonToEvent(line map[string]any) core.Event {
	event := core.Event{
		Timestamp: time.Now(),
	}
	
	// Parse event type
	if typeStr, ok := line["type"].(string); ok {
		// Map string to EventType
		switch typeStr {
		case "start":
			event.Type = core.EventStart
		case "text_delta":
			event.Type = core.EventTextDelta
			if text, ok := line["text"].(string); ok {
				event.TextDelta = text
			}
		case "audio_delta":
			event.Type = core.EventAudioDelta
			// Parse audio data if present
		case "tool_call":
			event.Type = core.EventToolCall
			// Parse tool call data
		case "tool_result":
			event.Type = core.EventToolResult
			// Parse tool result data
		case "citations":
			event.Type = core.EventCitations
			// Parse citations
		case "safety":
			event.Type = core.EventSafety
			// Parse safety data
		case "finish_step":
			event.Type = core.EventFinishStep
		case "finish":
			event.Type = core.EventFinish
		case "error":
			event.Type = core.EventError
			if errMsg, ok := line["error"].(string); ok {
				event.Err = fmt.Errorf("%s", errMsg)
			}
		default:
			event.Type = core.EventRaw
			event.Raw = line
		}
	}
	
	// Parse timestamp if present
	if ts, ok := line["timestamp"].(float64); ok {
		event.Timestamp = time.Unix(int64(ts), 0)
	}
	
	return event
}

// Writer provides a low-level NDJSON writer.
type NDJSONWriter struct {
	w       io.Writer
	encoder *json.Encoder
	mu      sync.Mutex
}

// NewNDJSONWriter creates a new NDJSON writer.
func NewNDJSONWriter(w io.Writer) *NDJSONWriter {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return &NDJSONWriter{
		w:       w,
		encoder: encoder,
	}
}

// Write writes a value as a JSON line.
func (w *NDJSONWriter) Write(v any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.encoder.Encode(v)
}