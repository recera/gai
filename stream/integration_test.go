// Package stream provides streaming utilities for AI responses.
// This file contains integration tests for SSE and NDJSON streaming with httptest.
package stream

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
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

// TestIntegrationSSEWithHTTPServer tests SSE with a full HTTP server.
func TestIntegrationSSEWithHTTPServer(t *testing.T) {
	// Create a test provider
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			stream := newMockTextStream()
			
			// Simulate AI response generation
			go func() {
				events := []core.Event{
					{Type: core.EventStart, Timestamp: time.Now()},
					{Type: core.EventTextDelta, TextDelta: "The ", Timestamp: time.Now()},
					{Type: core.EventTextDelta, TextDelta: "capital ", Timestamp: time.Now()},
					{Type: core.EventTextDelta, TextDelta: "of ", Timestamp: time.Now()},
					{Type: core.EventTextDelta, TextDelta: "France ", Timestamp: time.Now()},
					{Type: core.EventTextDelta, TextDelta: "is ", Timestamp: time.Now()},
					{Type: core.EventTextDelta, TextDelta: "Paris.", Timestamp: time.Now()},
					{Type: core.EventFinish, Usage: &core.Usage{
						InputTokens:  10,
						OutputTokens: 7,
						TotalTokens:  17,
					}, Timestamp: time.Now()},
				}
				
				for _, evt := range events {
					stream.sendEvent(evt)
					time.Sleep(10 * time.Millisecond) // Simulate processing time
				}
				stream.Close()
			}()
			
			return stream, nil
		},
	}

	// Create SSE handler
	handler := SSEHandler(provider, func(r *http.Request) (core.Request, error) {
		question := r.URL.Query().Get("q")
		if question == "" {
			question = "What is the capital of France?"
		}
		
		return core.Request{
			Messages: []core.Message{
				{Role: core.System, Parts: []core.Part{core.Text{Text: "You are a helpful assistant."}}},
				{Role: core.User, Parts: []core.Part{core.Text{Text: question}}},
			},
			Stream: true,
		}, nil
	})

	// Start test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make request as browser would
	resp, err := http.Get(server.URL + "?q=What%20is%20the%20capital%20of%20France")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Verify headers
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream, got %s", ct)
	}

	// Parse SSE response
	scanner := bufio.NewScanner(resp.Body)
	var events []map[string]string
	currentEvent := make(map[string]string)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		if line == "" {
			if len(currentEvent) > 0 {
				events = append(events, currentEvent)
				currentEvent = make(map[string]string)
			}
		} else if strings.HasPrefix(line, ":") {
			// Comment/keep-alive
			continue
		} else {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				field := parts[0]
				value := strings.TrimSpace(parts[1])
				currentEvent[field] = value
			}
		}
	}

	// Verify we got events
	if len(events) < 8 { // 7 content events + done
		t.Errorf("Expected at least 8 events, got %d", len(events))
	}

	// Reconstruct the full text
	var fullText string
	for _, event := range events {
		if event["event"] == "text_delta" {
			var data map[string]any
			json.Unmarshal([]byte(event["data"]), &data)
			if text, ok := data["text"].(string); ok {
				fullText += text
			}
		}
	}

	expectedText := "The capital of France is Paris."
	if fullText != expectedText {
		t.Errorf("Expected text %q, got %q", expectedText, fullText)
	}
}

// TestIntegrationNDJSONWithHTTPServer tests NDJSON with a full HTTP server.
func TestIntegrationNDJSONWithHTTPServer(t *testing.T) {
	// Create a test provider
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			stream := newMockTextStream()
			
			// Simulate AI response with tool calling
			go func() {
				events := []core.Event{
					{Type: core.EventStart, Timestamp: time.Now()},
					{Type: core.EventTextDelta, TextDelta: "Let me calculate that for you. ", Timestamp: time.Now()},
					{Type: core.EventToolCall, ToolName: "calculator", ToolID: "calc-1", 
						ToolInput: json.RawMessage(`{"operation":"add","x":5,"y":3}`), Timestamp: time.Now()},
					{Type: core.EventToolResult, ToolName: "calculator", 
						ToolResult: map[string]any{"result": 8}, Timestamp: time.Now()},
					{Type: core.EventTextDelta, TextDelta: "The result is 8.", Timestamp: time.Now()},
					{Type: core.EventFinish, Usage: &core.Usage{TotalTokens: 25}, Timestamp: time.Now()},
				}
				
				for _, evt := range events {
					stream.sendEvent(evt)
					time.Sleep(10 * time.Millisecond)
				}
				stream.Close()
			}()
			
			return stream, nil
		},
	}

	// Create NDJSON handler
	handler := NDJSONHandler(provider, func(r *http.Request) (core.Request, error) {
		return core.Request{
			Messages: []core.Message{
				{Role: core.User, Parts: []core.Part{core.Text{Text: "What is 5 + 3?"}}},
			},
			Stream: true,
		}, nil
	})

	// Start test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Verify headers
	if ct := resp.Header.Get("Content-Type"); ct != "application/x-ndjson" {
		t.Errorf("Expected Content-Type application/x-ndjson, got %s", ct)
	}

	// Parse NDJSON response
	scanner := bufio.NewScanner(resp.Body)
	var lines []map[string]any
	
	for scanner.Scan() {
		line := scanner.Text()
		var data map[string]any
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			t.Errorf("Failed to parse NDJSON line: %v", err)
			continue
		}
		lines = append(lines, data)
	}

	// Verify we got expected events
	if len(lines) < 6 {
		t.Errorf("Expected at least 6 lines, got %d", len(lines))
	}

	// Check for tool call and result
	hasToolCall := false
	hasToolResult := false
	
	for _, line := range lines {
		if line["type"] == "tool_call" {
			hasToolCall = true
			toolCall := line["tool_call"].(map[string]any)
			if toolCall["name"] != "calculator" {
				t.Error("Expected calculator tool call")
			}
		}
		if line["type"] == "tool_result" {
			hasToolResult = true
		}
	}

	if !hasToolCall {
		t.Error("Missing tool call event")
	}
	if !hasToolResult {
		t.Error("Missing tool result event")
	}
}

// TestIntegrationConcurrentClients tests multiple concurrent clients.
func TestIntegrationConcurrentClients(t *testing.T) {
	// Create a provider that handles concurrent requests
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			stream := newMockTextStream()
			
			go func() {
				// Send a unique response per client
				stream.sendEvent(core.Event{
					Type:      core.EventTextDelta,
					TextDelta: fmt.Sprintf("Response at %v", time.Now().UnixNano()),
					Timestamp: time.Now(),
				})
				time.Sleep(50 * time.Millisecond)
				stream.Close()
			}()
			
			return stream, nil
		},
	}

	// Create SSE handler
	handler := SSEHandler(provider, func(r *http.Request) (core.Request, error) {
		return core.Request{
			Messages: []core.Message{
				{Role: core.User, Parts: []core.Part{core.Text{Text: "Hello"}}},
			},
			Stream: true,
		}, nil
	})

	// Start test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Launch concurrent clients
	clientCount := 10
	var wg sync.WaitGroup
	responses := make([]string, clientCount)
	errors := make([]error, clientCount)

	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			
			resp, err := http.Get(server.URL)
			if err != nil {
				errors[idx] = err
				return
			}
			defer resp.Body.Close()
			
			// Read full response
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				errors[idx] = err
				return
			}
			
			responses[idx] = string(body)
		}(i)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			t.Errorf("Client %d error: %v", i, err)
		}
	}

	// Verify all clients got responses
	for i, resp := range responses {
		if resp == "" {
			t.Errorf("Client %d got empty response", i)
		}
		if !strings.Contains(resp, "Response at") {
			t.Errorf("Client %d got unexpected response", i)
		}
	}
}

// TestIntegrationSlowClient tests handling of slow clients.
func TestIntegrationSlowClient(t *testing.T) {
	// Create a provider that sends events quickly
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			stream := newMockTextStream()
			
			go func() {
				for i := 0; i < 10; i++ {
					stream.sendEvent(core.Event{
						Type:      core.EventTextDelta,
						TextDelta: fmt.Sprintf("Part %d ", i),
						Timestamp: time.Now(),
					})
					time.Sleep(5 * time.Millisecond)
				}
				stream.Close()
			}()
			
			return stream, nil
		},
	}

	// Create SSE handler
	handler := SSEHandler(provider, func(r *http.Request) (core.Request, error) {
		return core.Request{Stream: true}, nil
	})

	// Start test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make request with slow reading
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read slowly
	reader := bufio.NewReader(resp.Body)
	var fullResponse bytes.Buffer
	
	for i := 0; i < 50; i++ { // Read in small chunks with delays
		buf := make([]byte, 100)
		n, err := reader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil && err != io.EOF {
			t.Fatalf("Read error: %v", err)
		}
		fullResponse.Write(buf[:n])
		time.Sleep(10 * time.Millisecond) // Simulate slow client
	}

	// Verify all parts were received
	response := fullResponse.String()
	for i := 0; i < 10; i++ {
		expected := fmt.Sprintf("Part %d", i)
		if !strings.Contains(response, expected) {
			t.Errorf("Missing %s in response", expected)
		}
	}
}

// TestIntegrationClientDisconnect tests handling of client disconnection.
func TestIntegrationClientDisconnect(t *testing.T) {
	streamClosed := make(chan bool, 1)
	
	// Create a provider that detects stream closure
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			stream := newMockTextStream()
			
			go func() {
				defer func() {
					streamClosed <- true
				}()
				
				// Send events until context is cancelled
				for i := 0; i < 100; i++ {
					select {
					case <-ctx.Done():
						stream.Close()
						return
					default:
					}
					
					stream.sendEvent(core.Event{
						Type:      core.EventTextDelta,
						TextDelta: fmt.Sprintf("Part %d ", i),
						Timestamp: time.Now(),
					})
					time.Sleep(50 * time.Millisecond)
				}
				stream.Close()
			}()
			
			return stream, nil
		},
	}

	// Create SSE handler
	handler := SSEHandler(provider, func(r *http.Request) (core.Request, error) {
		return core.Request{Stream: true}, nil
	})

	// Start test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make request and disconnect early
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
	}

	// Wait for stream to close
	select {
	case <-streamClosed:
		// Good, stream was closed
	case <-time.After(1 * time.Second):
		t.Error("Stream was not closed after client disconnect")
	}
}

// TestIntegrationErrorDuringStream tests error handling during streaming.
func TestIntegrationErrorDuringStream(t *testing.T) {
	// Create a provider that errors mid-stream
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			stream := newMockTextStream()
			
			go func() {
				stream.sendEvent(core.Event{
					Type:      core.EventTextDelta,
					TextDelta: "Starting...",
					Timestamp: time.Now(),
				})
				time.Sleep(10 * time.Millisecond)
				
				stream.sendEvent(core.Event{
					Type:      core.EventError,
					Err:       fmt.Errorf("simulated error"),
					Timestamp: time.Now(),
				})
				stream.Close()
			}()
			
			return stream, nil
		},
	}

	tests := []struct {
		name     string
		format   string
		handler  http.HandlerFunc
		checkErr func(t *testing.T, body string)
	}{
		{
			name:   "SSE_error",
			format: "sse",
			handler: SSEHandler(provider, func(r *http.Request) (core.Request, error) {
				return core.Request{Stream: true}, nil
			}),
			checkErr: func(t *testing.T, body string) {
				if !strings.Contains(body, "event: error") {
					t.Error("Missing error event")
				}
				if !strings.Contains(body, "simulated error") {
					t.Error("Missing error message")
				}
			},
		},
		{
			name:   "NDJSON_error",
			format: "ndjson",
			handler: NDJSONHandler(provider, func(r *http.Request) (core.Request, error) {
				return core.Request{Stream: true}, nil
			}),
			checkErr: func(t *testing.T, body string) {
				lines := strings.Split(strings.TrimSpace(body), "\n")
				hasError := false
				for _, line := range lines {
					if strings.Contains(line, `"type":"error"`) && strings.Contains(line, "simulated error") {
						hasError = true
						break
					}
				}
				if !hasError {
					t.Error("Missing error in NDJSON")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			
			resp, err := http.Get(server.URL)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()
			
			body, _ := io.ReadAll(resp.Body)
			tt.checkErr(t, string(body))
		})
	}
}

// TestIntegrationCrossOriginRequest tests CORS headers.
func TestIntegrationCrossOriginRequest(t *testing.T) {
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			stream := newMockTextStream()
			go func() {
				stream.sendEvent(core.Event{
					Type:      core.EventTextDelta,
					TextDelta: "CORS test",
					Timestamp: time.Now(),
				})
				stream.Close()
			}()
			return stream, nil
		},
	}

	// Test both SSE and NDJSON
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{
			name: "SSE_CORS",
			handler: SSEHandler(provider, func(r *http.Request) (core.Request, error) {
				return core.Request{Stream: true}, nil
			}),
		},
		{
			name: "NDJSON_CORS",
			handler: NDJSONHandler(provider, func(r *http.Request) (core.Request, error) {
				return core.Request{Stream: true}, nil
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			
			// Make request with Origin header
			req, _ := http.NewRequest("GET", server.URL, nil)
			req.Header.Set("Origin", "https://example.com")
			
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()
			
			// Check CORS headers
			if allow := resp.Header.Get("Access-Control-Allow-Origin"); allow != "*" {
				t.Errorf("Expected Access-Control-Allow-Origin *, got %s", allow)
			}
			if methods := resp.Header.Get("Access-Control-Allow-Methods"); !strings.Contains(methods, "GET") {
				t.Errorf("Access-Control-Allow-Methods missing GET: %s", methods)
			}
		})
	}
}

// TestIntegrationMessageOrdering verifies event ordering is preserved.
func TestIntegrationMessageOrdering(t *testing.T) {
	eventCount := 100
	
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			stream := newMockTextStream()
			
			go func() {
				for i := 0; i < eventCount; i++ {
					stream.sendEvent(core.Event{
						Type:      core.EventTextDelta,
						TextDelta: fmt.Sprintf("%d,", i),
						Timestamp: time.Now(),
					})
				}
				stream.Close()
			}()
			
			return stream, nil
		},
	}

	// Test NDJSON ordering
	handler := NDJSONHandler(provider, func(r *http.Request) (core.Request, error) {
		return core.Request{Stream: true}, nil
	})
	
	server := httptest.NewServer(handler)
	defer server.Close()
	
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	// Parse response and extract numbers
	scanner := bufio.NewScanner(resp.Body)
	var numbers []int
	
	for scanner.Scan() {
		line := scanner.Text()
		var data map[string]any
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}
		
		if data["type"] == "text_delta" {
			if text, ok := data["text"].(string); ok {
				// Extract number from format "N,"
				text = strings.TrimSuffix(text, ",")
				var num int
				if _, err := fmt.Sscanf(text, "%d", &num); err == nil {
					numbers = append(numbers, num)
				}
			}
		}
	}
	
	// Verify ordering
	if len(numbers) != eventCount {
		t.Errorf("Expected %d numbers, got %d", eventCount, len(numbers))
	}
	
	for i, num := range numbers {
		if num != i {
			t.Errorf("Order mismatch at position %d: got %d", i, num)
			break
		}
	}
}

// BenchmarkIntegrationSSE measures SSE performance with httptest.
func BenchmarkIntegrationSSE(b *testing.B) {
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			stream := newMockTextStream()
			go func() {
				for i := 0; i < 100; i++ {
					stream.sendEvent(core.Event{
						Type:      core.EventTextDelta,
						TextDelta: "Benchmark text",
						Timestamp: time.Now(),
					})
				}
				stream.Close()
			}()
			return stream, nil
		},
	}
	
	handler := SSEHandler(provider, func(r *http.Request) (core.Request, error) {
		return core.Request{Stream: true}, nil
	})
	
	server := httptest.NewServer(handler)
	defer server.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := http.Get(server.URL)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// BenchmarkIntegrationNDJSON measures NDJSON performance with httptest.
func BenchmarkIntegrationNDJSON(b *testing.B) {
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			stream := newMockTextStream()
			go func() {
				for i := 0; i < 100; i++ {
					stream.sendEvent(core.Event{
						Type:      core.EventTextDelta,
						TextDelta: "Benchmark text",
						Timestamp: time.Now(),
					})
				}
				stream.Close()
			}()
			return stream, nil
		},
	}
	
	handler := NDJSONHandler(provider, func(r *http.Request) (core.Request, error) {
		return core.Request{Stream: true}, nil
	})
	
	server := httptest.NewServer(handler)
	defer server.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := http.Get(server.URL)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}