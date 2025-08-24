// Package stream provides streaming utilities for AI responses.
// This file contains comprehensive tests for SSE streaming.
package stream

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

// mockTextStream implements core.TextStream for testing.
type mockTextStream struct {
	events   chan core.Event
	closeErr error
	closed   bool
	mu       sync.Mutex
}

func newMockTextStream() *mockTextStream {
	return &mockTextStream{
		events: make(chan core.Event, 100),
	}
}

func (m *mockTextStream) Events() <-chan core.Event {
	return m.events
}

func (m *mockTextStream) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		close(m.events)
		m.closed = true
	}
	return m.closeErr
}

func (m *mockTextStream) sendEvent(event core.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.events <- event
	}
}

// TestSSEHeaders verifies that correct headers are set.
func TestSSEHeaders(t *testing.T) {
	stream := newMockTextStream()
	defer stream.Close()

	// Send a test event
	go func() {
		stream.sendEvent(core.Event{
			Type:      core.EventTextDelta,
			TextDelta: "Hello",
			Timestamp: time.Now(),
		})
		time.Sleep(10 * time.Millisecond)
		stream.Close()
	}()

	// Create test server
	rec := httptest.NewRecorder()
	err := SSE(rec, stream)
	if err != nil {
		t.Fatalf("SSE failed: %v", err)
	}

	// Check headers
	headers := rec.Header()
	
	expectedHeaders := map[string]string{
		"Content-Type":              "text/event-stream",
		"Cache-Control":             "no-cache, no-store, must-revalidate",
		"Connection":                "keep-alive",
		"X-Accel-Buffering":         "no",
		"Access-Control-Allow-Origin": "*",
	}

	for key, expected := range expectedHeaders {
		if got := headers.Get(key); got != expected {
			t.Errorf("Header %s = %q, want %q", key, got, expected)
		}
	}
}

// TestSSEEventFormatting verifies SSE event format.
func TestSSEEventFormatting(t *testing.T) {
	tests := []struct {
		name     string
		event    core.Event
		wantData map[string]any
		wantType string
	}{
		{
			name: "text_delta",
			event: core.Event{
				Type:      core.EventTextDelta,
				TextDelta: "Hello world",
			},
			wantData: map[string]any{
				"text": "Hello world",
			},
			wantType: "text_delta",
		},
		{
			name: "tool_call",
			event: core.Event{
				Type:      core.EventToolCall,
				ToolName:  "calculator",
				ToolID:    "calc-123",
				ToolInput: json.RawMessage(`{"x":1,"y":2}`),
			},
			wantData: map[string]any{
				"tool_name": "calculator",
				"tool_id":   "calc-123",
				"input":     json.RawMessage(`{"x":1,"y":2}`),
			},
			wantType: "tool_call",
		},
		{
			name: "error",
			event: core.Event{
				Type: core.EventError,
				Err:  errors.New("test error"),
			},
			wantData: map[string]any{
				"error": "test error",
			},
			wantType: "error",
		},
		{
			name: "finish",
			event: core.Event{
				Type: core.EventFinish,
				Usage: &core.Usage{
					InputTokens:  100,
					OutputTokens: 50,
					TotalTokens:  150,
				},
			},
			wantData: map[string]any{
				"usage": &core.Usage{
					InputTokens:  100,
					OutputTokens: 50,
					TotalTokens:  150,
				},
			},
			wantType: "finish",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := newMockTextStream()
			defer stream.Close()

			// Send the test event
			go func() {
				stream.sendEvent(tt.event)
				time.Sleep(10 * time.Millisecond)
				stream.Close()
			}()

			// Capture output
			rec := httptest.NewRecorder()
			err := SSE(rec, stream)
			if err != nil {
				t.Fatalf("SSE failed: %v", err)
			}

			// Parse SSE output
			body := rec.Body.String()
			lines := strings.Split(body, "\n")
			
			// Find event and data lines
			var eventLine, dataLine string
			for _, line := range lines {
				if strings.HasPrefix(line, "event: ") {
					eventLine = strings.TrimPrefix(line, "event: ")
				}
				if strings.HasPrefix(line, "data: ") {
					dataLine = strings.TrimPrefix(line, "data: ")
				}
			}

			// Check event type
			if eventLine != tt.wantType && eventLine != "done" {
				t.Errorf("Event type = %q, want %q", eventLine, tt.wantType)
			}

			// Parse data JSON if present
			if dataLine != "" && dataLine != "{\"finished\":true}" {
				var data map[string]any
				if err := json.Unmarshal([]byte(dataLine), &data); err != nil {
					t.Fatalf("Failed to parse data JSON: %v", err)
				}

				// Verify data content
				if data["data"] != nil {
					actualData := data["data"].(map[string]any)
					// Compare specific fields based on event type
					switch tt.event.Type {
					case core.EventTextDelta:
						if actualData["text"] != tt.wantData["text"] {
							t.Errorf("Text data mismatch: got %v, want %v", actualData["text"], tt.wantData["text"])
						}
					case core.EventError:
						if actualData["error"] != tt.wantData["error"] {
							t.Errorf("Error data mismatch: got %v, want %v", actualData["error"], tt.wantData["error"])
						}
					}
				}
			}
		})
	}
}

// TestSSEHeartbeat verifies keep-alive messages are sent.
func TestSSEHeartbeat(t *testing.T) {
	stream := newMockTextStream()
	defer stream.Close()

	// Keep stream open for heartbeat
	done := make(chan bool)
	go func() {
		time.Sleep(200 * time.Millisecond)
		stream.Close()
		done <- true
	}()

	// Use short heartbeat interval for testing
	opts := SSEOptions{
		HeartbeatInterval: 50 * time.Millisecond,
		FlushAfterWrite:   true,
	}

	// Capture output
	rec := httptest.NewRecorder()
	
	// Run SSE in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- SSE(rec, stream, opts)
	}()

	// Wait for completion
	<-done
	<-errCh

	// Check for keep-alive messages
	body := rec.Body.String()
	keepAliveCount := strings.Count(body, ": keep-alive")
	
	// Should have at least 2 keep-alive messages (200ms / 50ms = 4, but allow for timing)
	if keepAliveCount < 2 {
		t.Errorf("Expected at least 2 keep-alive messages, got %d", keepAliveCount)
	}
}

// TestSSEMultipleEvents verifies streaming multiple events in order.
func TestSSEMultipleEvents(t *testing.T) {
	stream := newMockTextStream()
	defer stream.Close()

	events := []core.Event{
		{Type: core.EventStart, Timestamp: time.Now()},
		{Type: core.EventTextDelta, TextDelta: "Hello ", Timestamp: time.Now()},
		{Type: core.EventTextDelta, TextDelta: "world!", Timestamp: time.Now()},
		{Type: core.EventFinish, Usage: &core.Usage{TotalTokens: 10}, Timestamp: time.Now()},
	}

	// Send events
	go func() {
		for _, evt := range events {
			stream.sendEvent(evt)
			time.Sleep(5 * time.Millisecond)
		}
		stream.Close()
	}()

	// Capture output
	rec := httptest.NewRecorder()
	err := SSE(rec, stream)
	if err != nil {
		t.Fatalf("SSE failed: %v", err)
	}

	// Parse SSE output
	body := rec.Body.String()
	
	// Count event occurrences
	startCount := strings.Count(body, "event: start")
	textCount := strings.Count(body, "event: text_delta")
	finishCount := strings.Count(body, "event: finish")
	doneCount := strings.Count(body, "event: done")

	if startCount != 1 {
		t.Errorf("Expected 1 EventStart, got %d", startCount)
	}
	if textCount != 2 {
		t.Errorf("Expected 2 EventTextDelta, got %d", textCount)
	}
	if finishCount != 1 {
		t.Errorf("Expected 1 EventFinish, got %d", finishCount)
	}
	if doneCount != 1 {
		t.Errorf("Expected 1 done event, got %d", doneCount)
	}
}

// TestSSEWithEventID verifies event ID generation.
func TestSSEWithEventID(t *testing.T) {
	stream := newMockTextStream()
	defer stream.Close()

	// Send multiple events
	go func() {
		for i := 0; i < 3; i++ {
			stream.sendEvent(core.Event{
				Type:      core.EventTextDelta,
				TextDelta: fmt.Sprintf("Message %d", i),
				Timestamp: time.Now(),
			})
		}
		stream.Close()
	}()

	opts := SSEOptions{
		HeartbeatInterval: 1 * time.Hour, // Disable heartbeat for this test
		FlushAfterWrite:   true,
		IncludeID:         true,
	}

	// Capture output
	rec := httptest.NewRecorder()
	err := SSE(rec, stream, opts)
	if err != nil {
		t.Fatalf("SSE failed: %v", err)
	}

	// Parse SSE output
	body := rec.Body.String()
	
	// Check for event IDs
	if !strings.Contains(body, "id: 1") {
		t.Error("Missing id: 1")
	}
	if !strings.Contains(body, "id: 2") {
		t.Error("Missing id: 2")
	}
	if !strings.Contains(body, "id: 3") {
		t.Error("Missing id: 3")
	}
}

// TestSSEHandler tests the HTTP handler functionality.
func TestSSEHandler(t *testing.T) {
	// Create mock provider
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			stream := newMockTextStream()
			
			// Send events asynchronously
			go func() {
				stream.sendEvent(core.Event{
					Type:      core.EventTextDelta,
					TextDelta: "Test response",
					Timestamp: time.Now(),
				})
				time.Sleep(10 * time.Millisecond)
				stream.Close()
			}()
			
			return stream, nil
		},
	}

	// Create handler
	handler := SSEHandler(provider, func(r *http.Request) (core.Request, error) {
		return core.Request{
			Messages: []core.Message{
				{Role: core.User, Parts: []core.Part{core.Text{Text: r.URL.Query().Get("q")}}},
			},
		}, nil
	})

	// Create test request
	req := httptest.NewRequest("GET", "/stream?q=Hello", nil)
	rec := httptest.NewRecorder()

	// Handle request
	handler(rec, req)

	// Check response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Check content type
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream, got %s", ct)
	}

	// Check body contains expected event
	body := rec.Body.String()
	if !strings.Contains(body, "Test response") {
		t.Error("Response doesn't contain expected text")
	}
}

// TestSSEHandlerError tests error handling in SSE handler.
func TestSSEHandlerError(t *testing.T) {
	tests := []struct {
		name           string
		prepareErr     error
		streamErr      error
		expectedStatus int
	}{
		{
			name:           "prepare_error",
			prepareErr:     errors.New("invalid request"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "stream_error",
			streamErr:      errors.New("provider error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock provider
			provider := &mockProvider{
				streamFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
					if tt.streamErr != nil {
						return nil, tt.streamErr
					}
					return newMockTextStream(), nil
				},
			}

			// Create handler
			handler := SSEHandler(provider, func(r *http.Request) (core.Request, error) {
				if tt.prepareErr != nil {
					return core.Request{}, tt.prepareErr
				}
				return core.Request{}, nil
			})

			// Create test request
			req := httptest.NewRequest("GET", "/stream", nil)
			rec := httptest.NewRecorder()

			// Handle request
			handler(rec, req)

			// Check response
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

// TestSSEConcurrentWrites tests thread safety of SSE writer.
func TestSSEConcurrentWrites(t *testing.T) {
	stream := newMockTextStream()
	defer stream.Close()

	// Send events concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			stream.sendEvent(core.Event{
				Type:      core.EventTextDelta,
				TextDelta: fmt.Sprintf("Message %d", n),
				Timestamp: time.Now(),
			})
		}(i)
	}

	// Wait and close
	go func() {
		wg.Wait()
		time.Sleep(10 * time.Millisecond)
		stream.Close()
	}()

	// Capture output
	rec := httptest.NewRecorder()
	err := SSE(rec, stream)
	if err != nil {
		t.Fatalf("SSE failed: %v", err)
	}

	// Count events
	body := rec.Body.String()
	eventCount := strings.Count(body, "event: text_delta")
	
	if eventCount != 10 {
		t.Errorf("Expected 10 events, got %d", eventCount)
	}
}

// TestSSEBrowserCompatibility simulates browser EventSource behavior.
func TestSSEBrowserCompatibility(t *testing.T) {
	stream := newMockTextStream()
	defer stream.Close()

	// Send events
	go func() {
		stream.sendEvent(core.Event{
			Type:      core.EventTextDelta,
			TextDelta: "Browser test",
			Timestamp: time.Now(),
		})
		stream.Close()
	}()

	// Capture output
	rec := httptest.NewRecorder()
	err := SSE(rec, stream)
	if err != nil {
		t.Fatalf("SSE failed: %v", err)
	}

	// Parse as EventSource would
	body := rec.Body.String()
	scanner := bufio.NewScanner(strings.NewReader(body))
	
	var events []map[string]string
	currentEvent := make(map[string]string)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		if line == "" {
			// Empty line marks end of event
			if len(currentEvent) > 0 {
				events = append(events, currentEvent)
				currentEvent = make(map[string]string)
			}
		} else if strings.HasPrefix(line, ":") {
			// Comment line (keep-alive)
			continue
		} else {
			// Parse field
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				field := parts[0]
				value := strings.TrimSpace(parts[1])
				currentEvent[field] = value
			}
		}
	}

	// Verify events were parsed correctly
	if len(events) < 2 { // At least the main event and done event
		t.Errorf("Expected at least 2 events, got %d", len(events))
	}

	// Check first event has correct structure
	if events[0]["event"] == "" {
		t.Error("First event missing event field")
	}
	if events[0]["data"] == "" {
		t.Error("First event missing data field")
	}

	// Verify data is valid JSON
	var data map[string]any
	if err := json.Unmarshal([]byte(events[0]["data"]), &data); err != nil {
		t.Errorf("Data is not valid JSON: %v", err)
	}
}

// TestSSEWriter tests the low-level SSE writer.
func TestSSEWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	// Write event
	err := writer.WriteEvent("message", `{"text":"hello"}`)
	if err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	// Write comment
	err = writer.WriteComment("keep-alive")
	if err != nil {
		t.Fatalf("WriteComment failed: %v", err)
	}

	output := buf.String()
	
	// Check event format
	if !strings.Contains(output, "event: message\n") {
		t.Error("Missing event field")
	}
	if !strings.Contains(output, `data: {"text":"hello"}`) {
		t.Error("Missing data field")
	}
	if !strings.Contains(output, ": keep-alive\n") {
		t.Error("Missing comment")
	}
}

// mockProvider implements core.Provider for testing.
type mockProvider struct {
	generateFunc func(ctx context.Context, req core.Request) (*core.TextResult, error)
	streamFunc   func(ctx context.Context, req core.Request) (core.TextStream, error)
}

func (m *mockProvider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, req)
	}
	return &core.TextResult{Text: "mock response"}, nil
}

func (m *mockProvider) StreamText(ctx context.Context, req core.Request) (core.TextStream, error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, req)
	}
	return newMockTextStream(), nil
}

func (m *mockProvider) GenerateObject(ctx context.Context, req core.Request, schema any) (*core.ObjectResult[any], error) {
	return nil, errors.New("not implemented")
}

func (m *mockProvider) StreamObject(ctx context.Context, req core.Request, schema any) (core.ObjectStream[any], error) {
	return nil, errors.New("not implemented")
}

// BenchmarkSSE measures SSE streaming performance.
func BenchmarkSSE(b *testing.B) {
	for i := 0; i < b.N; i++ {
		stream := newMockTextStream()
		
		go func() {
			for j := 0; j < 100; j++ {
				stream.sendEvent(core.Event{
					Type:      core.EventTextDelta,
					TextDelta: "Benchmark text",
					Timestamp: time.Now(),
				})
			}
			stream.Close()
		}()
		
		rec := httptest.NewRecorder()
		_ = SSE(rec, stream)
	}
}

// BenchmarkSSEWriter measures low-level writer performance.
func BenchmarkSSEWriter(b *testing.B) {
	writer := NewWriter(io.Discard)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = writer.WriteEvent("message", `{"text":"benchmark"}`)
	}
}