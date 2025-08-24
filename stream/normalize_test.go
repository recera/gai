// Package stream provides streaming utilities for AI responses.
// This file contains tests for event normalization and golden wire format validation.
package stream

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

// TestNormalizedEventSchema verifies the wire format structure.
func TestNormalizedEventSchema(t *testing.T) {
	// Create test timestamp
	ts := time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		event    core.Event
		expected NormalizedEvent
	}{
		{
			name: "start_event",
			event: core.Event{
				Type:      core.EventStart,
				Timestamp: ts,
			},
			expected: NormalizedEvent{
				Schema:    SchemaVersion,
				Type:      EventTypeStart,
				Timestamp: ts.UnixMilli(),
				Sequence:  1,
			},
		},
		{
			name: "text_delta",
			event: core.Event{
				Type:      core.EventTextDelta,
				TextDelta: "Hello world",
				Timestamp: ts,
			},
			expected: NormalizedEvent{
				Schema:    SchemaVersion,
				Type:      EventTypeTextDelta,
				Timestamp: ts.UnixMilli(),
				Sequence:  1,
				Text:      "Hello world",
			},
		},
		{
			name: "tool_call",
			event: core.Event{
				Type:      core.EventToolCall,
				ToolName:  "calculator",
				ToolID:    "call_123",
				ToolInput: json.RawMessage(`{"x":5,"y":3}`),
				Timestamp: ts,
			},
			expected: NormalizedEvent{
				Schema:    SchemaVersion,
				Type:      EventTypeToolCall,
				Timestamp: ts.UnixMilli(),
				Sequence:  1,
				CallID:    "call_123",
				ToolCall: &ToolCallData{
					Name:  "calculator",
					Input: json.RawMessage(`{"x":5,"y":3}`),
				},
			},
		},
		{
			name: "error_event",
			event: core.Event{
				Type: core.EventError,
				Err: core.NewError(
					core.ErrorRateLimited,
					"Rate limit exceeded",
					core.WithRetryAfter(60*time.Second),
				),
				Timestamp: ts,
			},
			expected: NormalizedEvent{
				Schema:    SchemaVersion,
				Type:      EventTypeError,
				Timestamp: ts.UnixMilli(),
				Sequence:  1,
				Error: &ErrorData{
					Code:       "rate_limited",
					Message:    "Rate limit exceeded",
					Temporary:  true,
					RetryAfter: 60000,
				},
			},
		},
		{
			name: "finish_event",
			event: core.Event{
				Type: core.EventFinish,
				Usage: &core.Usage{
					InputTokens:  100,
					OutputTokens: 50,
					TotalTokens:  150,
				},
				Timestamp: ts,
			},
			expected: NormalizedEvent{
				Schema:    SchemaVersion,
				Type:      EventTypeFinish,
				Timestamp: ts.UnixMilli(),
				Sequence:  1,
				Usage: &UsageData{
					InputTokens:  100,
					OutputTokens: 50,
					TotalTokens:  150,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizer := NewNormalizer("req_123", "trace_456")
			normalized := normalizer.Normalize(tt.event)

			// Check type
			if normalized.Type != tt.expected.Type {
				t.Errorf("Type = %v, want %v", normalized.Type, tt.expected.Type)
			}

			// Check schema
			if normalized.Schema != tt.expected.Schema {
				t.Errorf("Schema = %v, want %v", normalized.Schema, tt.expected.Schema)
			}

			// Check timestamp
			if normalized.Timestamp != tt.expected.Timestamp {
				t.Errorf("Timestamp = %v, want %v", normalized.Timestamp, tt.expected.Timestamp)
			}

			// Check event-specific fields
			switch tt.expected.Type {
			case EventTypeTextDelta:
				if normalized.Text != tt.expected.Text {
					t.Errorf("Text = %v, want %v", normalized.Text, tt.expected.Text)
				}
			case EventTypeToolCall:
				if normalized.CallID != tt.expected.CallID {
					t.Errorf("CallID = %v, want %v", normalized.CallID, tt.expected.CallID)
				}
				if normalized.ToolCall.Name != tt.expected.ToolCall.Name {
					t.Errorf("ToolCall.Name = %v, want %v", normalized.ToolCall.Name, tt.expected.ToolCall.Name)
				}
			case EventTypeError:
				if normalized.Error.Code != tt.expected.Error.Code {
					t.Errorf("Error.Code = %v, want %v", normalized.Error.Code, tt.expected.Error.Code)
				}
				if normalized.Error.RetryAfter != tt.expected.Error.RetryAfter {
					t.Errorf("Error.RetryAfter = %v, want %v", normalized.Error.RetryAfter, tt.expected.Error.RetryAfter)
				}
			case EventTypeFinish:
				if normalized.Usage.InputTokens != tt.expected.Usage.InputTokens {
					t.Errorf("Usage.InputTokens = %v, want %v", normalized.Usage.InputTokens, tt.expected.Usage.InputTokens)
				}
			}
		})
	}
}

// TestGoldenWireFormat validates the exact JSON wire format.
func TestGoldenWireFormat(t *testing.T) {
	// Fixed timestamp for deterministic output
	ts := time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC)

	// Create normalizer with fixed metadata
	normalizer := NewNormalizer("req_abc123", "trace_xyz789").
		WithProvider("openai").
		WithModel("gpt-4o-mini")

	// Test cases with expected JSON output
	tests := []struct {
		name     string
		event    core.Event
		expected string
	}{
		{
			name: "start_event",
			event: core.Event{
				Type:      core.EventStart,
				Timestamp: ts,
			},
			expected: `{"schema":"gai.events.v1","type":"start","ts":1705314600000,"seq":1,"trace_id":"trace_xyz789","request_id":"req_abc123","provider":"openai","model":"gpt-4o-mini"}`,
		},
		{
			name: "text_delta_minimal",
			event: core.Event{
				Type:      core.EventTextDelta,
				TextDelta: "Hello",
				Timestamp: ts,
			},
			expected: `{"schema":"gai.events.v1","type":"text.delta","ts":1705314600000,"seq":2,"trace_id":"trace_xyz789","request_id":"req_abc123","text":"Hello"}`,
		},
		{
			name: "tool_call_complete",
			event: core.Event{
				Type:      core.EventToolCall,
				ToolName:  "search",
				ToolID:    "call_456",
				ToolInput: json.RawMessage(`{"query":"golang"}`),
				Timestamp: ts,
			},
			expected: `{"schema":"gai.events.v1","type":"tool.call","ts":1705314600000,"seq":3,"trace_id":"trace_xyz789","request_id":"req_abc123","call_id":"call_456","tool_call":{"name":"search","input":{"query":"golang"}}}`,
		},
		{
			name: "error_with_retry",
			event: core.Event{
				Type: core.EventError,
				Err: core.NewError(
					core.ErrorRateLimited,
					"Too many requests",
					core.WithRetryAfter(30*time.Second),
				),
				Timestamp: ts,
			},
			expected: `{"schema":"gai.events.v1","type":"error","ts":1705314600000,"seq":4,"trace_id":"trace_xyz789","request_id":"req_abc123","error":{"code":"rate_limited","message":"Too many requests","temporary":true,"retry_after_ms":30000}}`,
		},
		{
			name: "finish_with_usage",
			event: core.Event{
				Type: core.EventFinish,
				Usage: &core.Usage{
					InputTokens:  150,
					OutputTokens: 75,
					TotalTokens:  225,
				},
				Timestamp: ts,
			},
			expected: `{"schema":"gai.events.v1","type":"finish","ts":1705314600000,"seq":5,"trace_id":"trace_xyz789","request_id":"req_abc123","provider":"openai","model":"gpt-4o-mini","usage":{"input_tokens":150,"output_tokens":75,"total_tokens":225}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizer.Normalize(tt.event)
			
			// Marshal to JSON
			data, err := json.Marshal(normalized)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Compare JSON strings
			got := string(data)
			if got != tt.expected {
				t.Errorf("JSON mismatch:\ngot:  %s\nwant: %s", got, tt.expected)
			}

			// Verify it can be parsed back
			var parsed NormalizedEvent
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Validate schema
			if err := ValidateSchema(parsed); err != nil && normalized.Type == EventTypeStart {
				t.Errorf("Schema validation failed: %v", err)
			}
		})
	}
}

// TestCompactJSON verifies the compact JSON format.
func TestCompactJSON(t *testing.T) {
	tests := []struct {
		name     string
		event    NormalizedEvent
		expected map[string]any
	}{
		{
			name: "text_delta_compact",
			event: NormalizedEvent{
				Type:     EventTypeTextDelta,
				Sequence: 5,
				Text:     "Hello world",
			},
			expected: map[string]any{
				"type": "text.delta",
				"seq":  int64(5),
				"text": "Hello world",
			},
		},
		{
			name: "start_event_compact",
			event: NormalizedEvent{
				Schema:    SchemaVersion,
				Type:      EventTypeStart,
				RequestID: "req_123",
				TraceID:   "trace_456",
				Provider:  "anthropic",
				Model:     "claude-3",
			},
			expected: map[string]any{
				"type":       "start",
				"schema":     SchemaVersion,
				"request_id": "req_123",
				"trace_id":   "trace_456",
				"provider":   "anthropic",
				"model":      "claude-3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compact := tt.event.CompactJSON()
			
			// Parse compact JSON
			var parsed map[string]any
			if err := json.Unmarshal(compact, &parsed); err != nil {
				t.Fatalf("Failed to parse compact JSON: %v", err)
			}

			// Check expected fields
			for key, expected := range tt.expected {
				// Handle float comparison for seq field
				if key == "seq" {
					if gotSeq, ok := parsed[key].(float64); ok {
						if int(gotSeq) != expected {
							t.Errorf("Field %s = %v, want %v", key, int(gotSeq), expected)
						}
					} else {
						t.Errorf("Field %s is not a number", key)
					}
				} else if parsed[key] != expected {
					t.Errorf("Field %s = %v, want %v", key, parsed[key], expected)
				}
			}

			// Ensure no extra fields
			for key := range parsed {
				if _, ok := tt.expected[key]; !ok {
					t.Errorf("Unexpected field in compact JSON: %s", key)
				}
			}
		})
	}
}

// TestNormalizedStream verifies streaming normalization.
func TestNormalizedStream(t *testing.T) {
	// Create mock stream
	mockStream := newMockTextStream()
	
	// Send test events
	go func() {
		events := []core.Event{
			{Type: core.EventStart, Timestamp: time.Now()},
			{Type: core.EventTextDelta, TextDelta: "Hello ", Timestamp: time.Now()},
			{Type: core.EventTextDelta, TextDelta: "world!", Timestamp: time.Now()},
			{Type: core.EventFinish, Usage: &core.Usage{TotalTokens: 10}, Timestamp: time.Now()},
		}
		
		for _, evt := range events {
			mockStream.sendEvent(evt)
			time.Sleep(5 * time.Millisecond)
		}
		mockStream.Close()
	}()

	// Create normalized stream
	normalizer := NewNormalizer("req_test", "trace_test")
	normalizedStream := NewNormalizedStream(mockStream, normalizer)
	defer normalizedStream.Close()

	// Collect normalized events
	var events []NormalizedEvent
	for event := range normalizedStream.Events() {
		events = append(events, event)
	}

	// Verify event count
	if len(events) != 4 {
		t.Errorf("Expected 4 events, got %d", len(events))
	}

	// Verify event types
	expectedTypes := []NormalizedEventType{
		EventTypeStart,
		EventTypeTextDelta,
		EventTypeTextDelta,
		EventTypeFinish,
	}

	for i, evt := range events {
		if evt.Type != expectedTypes[i] {
			t.Errorf("Event %d: type = %v, want %v", i, evt.Type, expectedTypes[i])
		}
		
		// Verify schema is always present
		if evt.Schema != SchemaVersion {
			t.Errorf("Event %d: missing or wrong schema", i)
		}
		
		// Verify sequence is increasing
		if evt.Sequence != int64(i+1) {
			t.Errorf("Event %d: sequence = %d, want %d", i, evt.Sequence, i+1)
		}
		
		// Verify request ID and trace ID
		if evt.RequestID != "req_test" {
			t.Errorf("Event %d: wrong request ID", i)
		}
		if evt.TraceID != "trace_test" {
			t.Errorf("Event %d: wrong trace ID", i)
		}
	}

	// Verify text content
	if events[1].Text != "Hello " || events[2].Text != "world!" {
		t.Error("Text content mismatch")
	}

	// Verify usage
	if events[3].Usage == nil || events[3].Usage.TotalTokens != 10 {
		t.Error("Usage data mismatch")
	}
}

// TestParseNormalizedEvent verifies parsing of normalized events.
func TestParseNormalizedEvent(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name: "valid_text_event",
			json: `{"schema":"gai.events.v1","type":"text.delta","ts":1705314600000,"seq":1,"text":"Hello"}`,
		},
		{
			name: "valid_error_event",
			json: `{"type":"error","error":{"code":"timeout","message":"Request timed out"}}`,
		},
		{
			name:    "invalid_json",
			json:    `{"type":"text.delta","text":`,
			wantErr: true,
		},
		{
			name: "missing_type",
			json: `{"schema":"gai.events.v1","text":"Hello"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseNormalizedEvent([]byte(tt.json))
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if event == nil {
				t.Fatal("Parsed event is nil")
			}
			
			// Verify it can be marshaled back
			if _, err := json.Marshal(event); err != nil {
				t.Errorf("Failed to marshal parsed event: %v", err)
			}
		})
	}
}

// TestSchemaValidation verifies schema version checking.
func TestSchemaValidation(t *testing.T) {
	tests := []struct {
		name    string
		event   NormalizedEvent
		wantErr bool
	}{
		{
			name: "correct_schema",
			event: NormalizedEvent{
				Type:   EventTypeStart,
				Schema: SchemaVersion,
			},
			wantErr: false,
		},
		{
			name: "wrong_schema",
			event: NormalizedEvent{
				Type:   EventTypeStart,
				Schema: "gai.events.v2",
			},
			wantErr: true,
		},
		{
			name: "non_start_event",
			event: NormalizedEvent{
				Type:   EventTypeTextDelta,
				Schema: "anything",
			},
			wantErr: false, // Only start events are validated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSchema(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRequestIDGeneration verifies request ID generation.
func TestRequestIDGeneration(t *testing.T) {
	gen := &DefaultRequestIDGenerator{}
	
	// Generate multiple IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := gen.Generate()
		
		// Check format
		if !strings.HasPrefix(id, "req_") {
			t.Errorf("Invalid ID format: %s", id)
		}
		
		// Check uniqueness
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

// TestWireFormatStability writes golden files for regression testing.
func TestWireFormatStability(t *testing.T) {
	// Skip in CI to avoid golden file changes
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping golden file test in CI")
	}

	// Fixed timestamp for deterministic output
	ts := time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC)
	
	// Create normalizer
	normalizer := NewNormalizer("req_golden", "trace_golden").
		WithProvider("test").
		WithModel("test-model")

	// Generate test events
	events := []core.Event{
		{Type: core.EventStart, Timestamp: ts},
		{Type: core.EventTextDelta, TextDelta: "The quick brown fox", Timestamp: ts},
		{Type: core.EventToolCall, ToolName: "test_tool", ToolID: "call_1", ToolInput: json.RawMessage(`{"test":true}`), Timestamp: ts},
		{Type: core.EventToolResult, ToolName: "test_tool", ToolResult: map[string]any{"result": "success"}, Timestamp: ts},
		{Type: core.EventFinish, Usage: &core.Usage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30}, Timestamp: ts},
	}

	// Normalize events and write to golden file
	goldenDir := "testdata/golden"
	os.MkdirAll(goldenDir, 0755)
	
	goldenFile := filepath.Join(goldenDir, "wire_format.json")
	
	var lines []string
	for _, event := range events {
		normalized := normalizer.Normalize(event)
		data, err := json.Marshal(normalized)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}
		lines = append(lines, string(data))
	}

	// Write or compare golden file
	golden := strings.Join(lines, "\n")
	
	if _, err := os.Stat(goldenFile); os.IsNotExist(err) {
		// Create golden file
		if err := os.WriteFile(goldenFile, []byte(golden), 0644); err != nil {
			t.Fatalf("Failed to write golden file: %v", err)
		}
		t.Log("Created golden file:", goldenFile)
	} else {
		// Compare with existing golden file
		existing, err := os.ReadFile(goldenFile)
		if err != nil {
			t.Fatalf("Failed to read golden file: %v", err)
		}
		
		if string(existing) != golden {
			t.Errorf("Wire format has changed! This is a breaking change.\nExpected:\n%s\nGot:\n%s", existing, golden)
			
			// Write actual output for inspection
			actualFile := filepath.Join(goldenDir, "wire_format_actual.json")
			os.WriteFile(actualFile, []byte(golden), 0644)
			t.Logf("Actual output written to: %s", actualFile)
		}
	}
}

// BenchmarkNormalization measures normalization performance.
func BenchmarkNormalization(b *testing.B) {
	normalizer := NewNormalizer("req_bench", "trace_bench")
	event := core.Event{
		Type:      core.EventTextDelta,
		TextDelta: "Benchmark text content",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = normalizer.Normalize(event)
	}
}

// BenchmarkCompactJSON measures compact JSON generation performance.
func BenchmarkCompactJSON(b *testing.B) {
	event := NormalizedEvent{
		Type:     EventTypeTextDelta,
		Sequence: 100,
		Text:     "Benchmark text content",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = event.CompactJSON()
	}
}

// BenchmarkParseNormalizedEvent measures parsing performance.
func BenchmarkParseNormalizedEvent(b *testing.B) {
	json := []byte(`{"schema":"gai.events.v1","type":"text.delta","ts":1705314600000,"seq":1,"text":"Hello"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseNormalizedEvent(json)
	}
}

// mockEventSource simulates a provider event source for testing.
type mockEventSource struct {
	events []core.Event
	index  int
}

func (m *mockEventSource) NextEvent() (core.Event, error) {
	if m.index >= len(m.events) {
		return core.Event{}, io.EOF
	}
	event := m.events[m.index]
	m.index++
	return event, nil
}

// Example demonstrates using the normalization API.
func ExampleNormalizer() {
	// Create normalizer with metadata
	normalizer := NewNormalizer("req_123", "trace_456").
		WithProvider("openai").
		WithModel("gpt-4")

	// Normalize an event
	event := core.Event{
		Type:      core.EventTextDelta,
		TextDelta: "Hello, world!",
		Timestamp: time.Now(),
	}

	normalized := normalizer.Normalize(event)
	
	// Marshal to JSON for transmission
	data, _ := json.Marshal(normalized)
	fmt.Printf("Normalized event: %s\n", string(data))
}

// Example_compactJSON demonstrates compact JSON format.
func Example_compactJSON() {
	event := NormalizedEvent{
		Type:     EventTypeTextDelta,
		Sequence: 5,
		Text:     "Example text",
	}

	compact := event.CompactJSON()
	fmt.Printf("Compact JSON: %s\n", string(compact))
	// Output: Compact JSON: {"seq":5,"text":"Example text","type":"text.delta"}
}