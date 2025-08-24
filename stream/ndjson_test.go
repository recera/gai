// Package stream provides streaming utilities for AI responses.
// This file contains comprehensive tests for NDJSON streaming.
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

// TestNDJSONHeaders verifies that correct headers are set.
func TestNDJSONHeaders(t *testing.T) {
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
	err := NDJSON(rec, stream)
	if err != nil {
		t.Fatalf("NDJSON failed: %v", err)
	}

	// Check headers
	headers := rec.Header()
	
	expectedHeaders := map[string]string{
		"Content-Type":              "application/x-ndjson",
		"Cache-Control":             "no-cache, no-store, must-revalidate",
		"Connection":                "keep-alive",
		"X-Accel-Buffering":         "no",
		"Transfer-Encoding":         "chunked",
		"Access-Control-Allow-Origin": "*",
	}

	for key, expected := range expectedHeaders {
		if got := headers.Get(key); got != expected {
			t.Errorf("Header %s = %q, want %q", key, got, expected)
		}
	}
}

// TestNDJSONLineFormat verifies NDJSON line formatting.
func TestNDJSONLineFormat(t *testing.T) {
	tests := []struct {
		name      string
		event     core.Event
		wantFields map[string]any
	}{
		{
			name: "text_delta",
			event: core.Event{
				Type:      core.EventTextDelta,
				TextDelta: "Hello world",
				Timestamp: time.Now(),
			},
			wantFields: map[string]any{
				"type": "text_delta",
				"text": "Hello world",
			},
		},
		{
			name: "tool_call",
			event: core.Event{
				Type:      core.EventToolCall,
				ToolName:  "calculator",
				ToolID:    "calc-123",
				ToolInput: json.RawMessage(`{"x":1,"y":2}`),
				Timestamp: time.Now(),
			},
			wantFields: map[string]any{
				"type": "tool_call",
				"tool_call": map[string]any{
					"name":  "calculator",
					"id":    "calc-123",
					"input": json.RawMessage(`{"x":1,"y":2}`),
				},
			},
		},
		{
			name: "error",
			event: core.Event{
				Type:      core.EventError,
				Err:       errors.New("test error"),
				Timestamp: time.Now(),
			},
			wantFields: map[string]any{
				"type":  "error",
				"error": "test error",
			},
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
				Timestamp: time.Now(),
			},
			wantFields: map[string]any{
				"type":     "finish",
				"finished": true,
			},
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
			err := NDJSON(rec, stream)
			if err != nil {
				t.Fatalf("NDJSON failed: %v", err)
			}

			// Parse NDJSON output
			body := rec.Body.String()
			lines := strings.Split(strings.TrimSpace(body), "\n")
			
			// Should have at least 2 lines (event + done)
			if len(lines) < 2 {
				t.Fatalf("Expected at least 2 lines, got %d", len(lines))
			}

			// Parse first line
			var data map[string]any
			if err := json.Unmarshal([]byte(lines[0]), &data); err != nil {
				t.Fatalf("Failed to parse NDJSON line: %v", err)
			}

			// Check type field
			if data["type"] != tt.wantFields["type"] {
				t.Errorf("Type = %v, want %v", data["type"], tt.wantFields["type"])
			}

			// Check event-specific fields
			switch tt.event.Type {
			case core.EventTextDelta:
				if data["text"] != tt.wantFields["text"] {
					t.Errorf("Text = %v, want %v", data["text"], tt.wantFields["text"])
				}
			case core.EventError:
				if data["error"] != tt.wantFields["error"] {
					t.Errorf("Error = %v, want %v", data["error"], tt.wantFields["error"])
				}
			}
		})
	}
}

// TestNDJSONMultipleEvents verifies streaming multiple events as separate lines.
func TestNDJSONMultipleEvents(t *testing.T) {
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
	err := NDJSON(rec, stream)
	if err != nil {
		t.Fatalf("NDJSON failed: %v", err)
	}

	// Parse NDJSON output
	body := rec.Body.String()
	lines := strings.Split(strings.TrimSpace(body), "\n")
	
	// Should have 5 lines (4 events + done)
	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var data map[string]any
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
		
		// Check type field exists
		if data["type"] == nil {
			t.Errorf("Line %d missing type field", i)
		}
	}

	// Check last line is done event
	var lastLine map[string]any
	json.Unmarshal([]byte(lines[len(lines)-1]), &lastLine)
	if lastLine["type"] != "done" || lastLine["finished"] != true {
		t.Error("Last line should be done event")
	}
}

// TestNDJSONWithTimestamp verifies timestamp inclusion.
func TestNDJSONWithTimestamp(t *testing.T) {
	stream := newMockTextStream()
	defer stream.Close()

	// Send event
	go func() {
		stream.sendEvent(core.Event{
			Type:      core.EventTextDelta,
			TextDelta: "Timestamp test",
			Timestamp: time.Now(),
		})
		stream.Close()
	}()

	opts := NDJSONOptions{
		BufferSize:       8192,
		FlushInterval:    100 * time.Millisecond,
		CompactJSON:      true,
		IncludeTimestamp: true,
	}

	// Capture output
	rec := httptest.NewRecorder()
	err := NDJSON(rec, stream, opts)
	if err != nil {
		t.Fatalf("NDJSON failed: %v", err)
	}

	// Parse first line
	body := rec.Body.String()
	lines := strings.Split(strings.TrimSpace(body), "\n")
	
	var data map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &data); err != nil {
		t.Fatalf("Failed to parse NDJSON line: %v", err)
	}

	// Check timestamp field
	if data["timestamp"] == nil {
		t.Error("Missing timestamp field")
	}
	
	// Verify timestamp is a number
	if _, ok := data["timestamp"].(float64); !ok {
		t.Error("Timestamp should be a number")
	}
}

// TestNDJSONHandler tests the HTTP handler functionality.
func TestNDJSONHandler(t *testing.T) {
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
	handler := NDJSONHandler(provider, func(r *http.Request) (core.Request, error) {
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
	if ct := rec.Header().Get("Content-Type"); ct != "application/x-ndjson" {
		t.Errorf("Expected Content-Type application/x-ndjson, got %s", ct)
	}

	// Check body contains expected text
	body := rec.Body.String()
	if !strings.Contains(body, "Test response") {
		t.Error("Response doesn't contain expected text")
	}

	// Verify each line is valid JSON
	lines := strings.Split(strings.TrimSpace(body), "\n")
	for _, line := range lines {
		var data map[string]any
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			t.Errorf("Invalid JSON line: %v", err)
		}
	}
}

// TestNDJSONLineIntegrity verifies each line is complete and valid.
func TestNDJSONLineIntegrity(t *testing.T) {
	stream := newMockTextStream()
	defer stream.Close()

	// Send various events with different content
	testEvents := []core.Event{
		{
			Type:      core.EventTextDelta,
			TextDelta: "Line with\nnewline character",
			Timestamp: time.Now(),
		},
		{
			Type:      core.EventTextDelta,
			TextDelta: `{"nested": "json"}`,
			Timestamp: time.Now(),
		},
		{
			Type:      core.EventTextDelta,
			TextDelta: "Special chars: émoji 😀 ümlaut",
			Timestamp: time.Now(),
		},
	}

	go func() {
		for _, evt := range testEvents {
			stream.sendEvent(evt)
			time.Sleep(5 * time.Millisecond)
		}
		stream.Close()
	}()

	// Capture output
	rec := httptest.NewRecorder()
	err := NDJSON(rec, stream)
	if err != nil {
		t.Fatalf("NDJSON failed: %v", err)
	}

	// Parse each line
	body := rec.Body.String()
	scanner := bufio.NewScanner(strings.NewReader(body))
	
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		// Verify line is valid JSON
		var data map[string]any
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			t.Errorf("Line %d is not valid JSON: %v\nLine: %s", lineCount, err, line)
		}
		
		lineCount++
	}

	// Should have correct number of lines (events + done)
	expectedLines := len(testEvents) + 1
	if lineCount != expectedLines {
		t.Errorf("Expected %d lines, got %d", expectedLines, lineCount)
	}
}

// TestNDJSONConcurrentWrites tests thread safety of NDJSON writer.
func TestNDJSONConcurrentWrites(t *testing.T) {
	stream := newMockTextStream()
	defer stream.Close()

	// Send events concurrently
	var wg sync.WaitGroup
	eventCount := 20
	
	for i := 0; i < eventCount; i++ {
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
	err := NDJSON(rec, stream)
	if err != nil {
		t.Fatalf("NDJSON failed: %v", err)
	}

	// Count valid JSON lines
	body := rec.Body.String()
	lines := strings.Split(strings.TrimSpace(body), "\n")
	
	validLines := 0
	for _, line := range lines {
		var data map[string]any
		if err := json.Unmarshal([]byte(line), &data); err == nil {
			validLines++
		}
	}

	// Should have eventCount + 1 (done event) valid lines
	expectedLines := eventCount + 1
	if validLines != expectedLines {
		t.Errorf("Expected %d valid lines, got %d", expectedLines, validLines)
	}
}

// TestNDJSONReader tests the NDJSON reader functionality.
func TestNDJSONReader(t *testing.T) {
	// Create NDJSON input
	lines := []string{
		`{"type":"start","started":true}`,
		`{"type":"text_delta","text":"Hello"}`,
		`{"type":"text_delta","text":" world"}`,
		`{"type":"finish","finished":true}`,
	}
	input := strings.Join(lines, "\n")
	
	// Create reader
	reader := NewReader(strings.NewReader(input))
	
	// Read each line
	for i := range lines {
		var data map[string]any
		err := reader.Read(&data)
		if err != nil {
			t.Fatalf("Failed to read line %d: %v", i, err)
		}
		
		// Verify data was parsed
		if data["type"] == nil {
			t.Errorf("Line %d missing type field", i)
		}
	}
	
	// Next read should return EOF
	var data map[string]any
	err := reader.Read(&data)
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}

// TestStreamToChannel tests converting NDJSON to event channel.
func TestStreamToChannel(t *testing.T) {
	// Create NDJSON input
	lines := []string{
		`{"type":"start","started":true}`,
		`{"type":"text_delta","text":"Hello"}`,
		`{"type":"finish","finished":true}`,
	}
	input := strings.Join(lines, "\n")
	
	// Convert to channel
	ctx := context.Background()
	events, err := StreamToChannel(ctx, strings.NewReader(input))
	if err != nil {
		t.Fatalf("StreamToChannel failed: %v", err)
	}
	
	// Collect events
	var receivedEvents []core.Event
	for event := range events {
		receivedEvents = append(receivedEvents, event)
	}
	
	// Should have 3 events
	if len(receivedEvents) != 3 {
		t.Errorf("Expected 3 events, got %d", len(receivedEvents))
	}
	
	// Verify event types
	expectedTypes := []core.EventType{
		core.EventStart,
		core.EventTextDelta,
		core.EventFinish,
	}
	
	for i, event := range receivedEvents {
		if event.Type != expectedTypes[i] {
			t.Errorf("Event %d: type = %v, want %v", i, event.Type, expectedTypes[i])
		}
	}
}

// TestNDJSONWriter tests the low-level NDJSON writer.
func TestNDJSONWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := NewNDJSONWriter(&buf)
	
	// Write multiple objects
	objects := []map[string]any{
		{"type": "start", "started": true},
		{"type": "text", "content": "Hello"},
		{"type": "end", "finished": true},
	}
	
	for _, obj := range objects {
		if err := writer.Write(obj); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}
	
	// Verify output
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	if len(lines) != len(objects) {
		t.Errorf("Expected %d lines, got %d", len(objects), len(lines))
	}
	
	// Each line should be valid JSON
	for i, line := range lines {
		var data map[string]any
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}
}

// TestNDJSONPeriodicFlush tests periodic flushing behavior.
func TestNDJSONPeriodicFlush(t *testing.T) {
	stream := newMockTextStream()
	defer stream.Close()

	// Keep stream open for flush testing
	done := make(chan bool)
	go func() {
		// Send an event
		stream.sendEvent(core.Event{
			Type:      core.EventTextDelta,
			TextDelta: "Test",
			Timestamp: time.Now(),
		})
		
		// Wait for periodic flush
		time.Sleep(200 * time.Millisecond)
		stream.Close()
		done <- true
	}()

	opts := NDJSONOptions{
		BufferSize:    8192,
		FlushInterval: 50 * time.Millisecond,
		CompactJSON:   true,
	}

	// Capture output
	rec := httptest.NewRecorder()
	
	// Run NDJSON in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- NDJSON(rec, stream, opts)
	}()

	// Wait for completion
	<-done
	<-errCh

	// Verify output was flushed
	body := rec.Body.String()
	if !strings.Contains(body, "Test") {
		t.Error("Event was not flushed")
	}
}

// BenchmarkNDJSON measures NDJSON streaming performance.
func BenchmarkNDJSON(b *testing.B) {
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
		_ = NDJSON(rec, stream)
	}
}

// BenchmarkNDJSONWriter measures low-level writer performance.
func BenchmarkNDJSONWriter(b *testing.B) {
	writer := NewNDJSONWriter(io.Discard)
	data := map[string]any{"type": "text", "content": "benchmark"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = writer.Write(data)
	}
}

// BenchmarkNDJSONReader measures reader performance.
func BenchmarkNDJSONReader(b *testing.B) {
	// Create large NDJSON input
	var lines []string
	for i := 0; i < 1000; i++ {
		lines = append(lines, fmt.Sprintf(`{"type":"text","content":"Line %d"}`, i))
	}
	input := strings.Join(lines, "\n")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := NewReader(strings.NewReader(input))
		for {
			var data map[string]any
			if err := reader.Read(&data); err == io.EOF {
				break
			}
		}
	}
}