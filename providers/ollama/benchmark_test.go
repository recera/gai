package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

// BenchmarkProvider_GenerateText benchmarks text generation performance
func BenchmarkProvider_GenerateText(b *testing.B) {
	server := createBenchmarkServer(b)
	defer server.Close()

	provider := New(WithBaseURL(server.URL), WithModel("llama3.2"))

	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Hello, how are you?"}}},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := provider.GenerateText(context.Background(), req)
		if err != nil {
			b.Fatalf("GenerateText failed: %v", err)
		}
		if result.Text == "" {
			b.Error("Expected non-empty response")
		}
	}
}

// BenchmarkProvider_StreamText benchmarks streaming text generation
func BenchmarkProvider_StreamText(b *testing.B) {
	server := createStreamingBenchmarkServer(b)
	defer server.Close()

	provider := New(WithBaseURL(server.URL), WithModel("llama3.2"))

	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Tell me a story"}}},
		},
		Temperature: 0.7,
		MaxTokens:   200,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		stream, err := provider.StreamText(context.Background(), req)
		if err != nil {
			b.Fatalf("StreamText failed: %v", err)
		}

		eventCount := 0
		for event := range stream.Events() {
			eventCount++
			if event.Type == core.EventError {
				b.Errorf("Stream error: %v", event.Err)
			}
		}

		stream.Close()

		if eventCount == 0 {
			b.Error("Expected events from stream")
		}
	}
}

// BenchmarkProvider_GenerateObject benchmarks structured output generation
func BenchmarkProvider_GenerateObject(b *testing.B) {
	server := createBenchmarkServer(b)
	defer server.Close()

	provider := New(WithBaseURL(server.URL), WithModel("llama3.2"))

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
			"age":  map[string]interface{}{"type": "integer"},
			"city": map[string]interface{}{"type": "string"},
		},
		"required": []string{"name", "age"},
	}

	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Generate a person"}}},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := provider.GenerateObject(context.Background(), req, schema)
		if err != nil {
			b.Fatalf("GenerateObject failed: %v", err)
		}
		if result.Value == nil {
			b.Error("Expected non-nil object value")
		}
	}
}

// BenchmarkProvider_convertRequest benchmarks request conversion
func BenchmarkProvider_convertRequest(b *testing.B) {
	provider := New(WithModel("llama3.2"))

	req := core.Request{
		Messages: []core.Message{
			{Role: core.System, Parts: []core.Part{core.Text{Text: "You are helpful"}}},
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Hello world"}}},
		},
		Temperature: 0.7,
		MaxTokens:   100,
		Tools:       []core.ToolHandle{&mockToolHandle{name: "test", desc: "test tool"}},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		chatReq, err := provider.convertRequest(req)
		if err != nil {
			b.Fatalf("convertRequest failed: %v", err)
		}
		if chatReq.Model == "" {
			b.Error("Expected non-empty model")
		}
	}
}

// BenchmarkProvider_convertMessages benchmarks message conversion
func BenchmarkProvider_convertMessages(b *testing.B) {
	provider := New()

	messages := []core.Message{
		{Role: core.System, Parts: []core.Part{core.Text{Text: "System prompt"}}},
		{Role: core.User, Parts: []core.Part{core.Text{Text: "User message 1"}}},
		{Role: core.Assistant, Parts: []core.Part{core.Text{Text: "Assistant response 1"}}},
		{Role: core.User, Parts: []core.Part{core.Text{Text: "User message 2"}}},
		{Role: core.Assistant, Parts: []core.Part{core.Text{Text: "Assistant response 2"}}},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := provider.convertMessages(messages)
		if err != nil {
			b.Fatalf("convertMessages failed: %v", err)
		}
		if len(result) != len(messages) {
			b.Errorf("Expected %d messages, got %d", len(messages), len(result))
		}
	}
}

// BenchmarkProvider_convertTools benchmarks tool conversion
func BenchmarkProvider_convertTools(b *testing.B) {
	provider := New()

	tools := []core.ToolHandle{
		&mockToolHandle{name: "tool1", desc: "First tool", inSchema: json.RawMessage(`{"type":"object"}`)},
		&mockToolHandle{name: "tool2", desc: "Second tool", inSchema: json.RawMessage(`{"type":"object"}`)},
		&mockToolHandle{name: "tool3", desc: "Third tool", inSchema: json.RawMessage(`{"type":"object"}`)},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := provider.convertTools(tools)
		if len(result) != len(tools) {
			b.Errorf("Expected %d tools, got %d", len(tools), len(result))
		}
	}
}

// BenchmarkConcurrentRequests benchmarks concurrent request handling
func BenchmarkConcurrentRequests(b *testing.B) {
	server := createBenchmarkServer(b)
	defer server.Close()

	provider := New(WithBaseURL(server.URL), WithModel("llama3.2"))

	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Concurrent test"}}},
		},
		MaxTokens: 50,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result, err := provider.GenerateText(context.Background(), req)
			if err != nil {
				b.Errorf("GenerateText failed: %v", err)
				continue
			}
			if result.Text == "" {
				b.Error("Expected non-empty response")
			}
		}
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocations during processing
func BenchmarkMemoryAllocation(b *testing.B) {
	provider := New(WithModel("llama3.2"))

	// Large request to test memory efficiency
	var largeParts []core.Part
	for i := 0; i < 100; i++ {
		largeParts = append(largeParts, core.Text{Text: "This is a large text part for memory testing. "})
	}

	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: largeParts},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := provider.convertRequest(req)
		if err != nil {
			b.Fatalf("convertRequest failed: %v", err)
		}
	}
}

// BenchmarkJSONParsing benchmarks JSON parsing performance
func BenchmarkJSONParsing(b *testing.B) {
	// Sample chat response JSON
	responseJSON := `{
		"model": "llama3.2",
		"created_at": "2025-01-01T00:00:00Z",
		"message": {
			"role": "assistant",
			"content": "This is a test response with multiple words and sentences to test JSON parsing performance."
		},
		"done": true,
		"total_duration": 1000000000,
		"load_duration": 100000000,
		"prompt_eval_count": 15,
		"prompt_eval_duration": 500000000,
		"eval_count": 25,
		"eval_duration": 400000000
	}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var response chatResponse
		err := json.Unmarshal([]byte(responseJSON), &response)
		if err != nil {
			b.Fatalf("JSON unmarshal failed: %v", err)
		}
		if !response.Done {
			b.Error("Expected response to be done")
		}
	}
}

// Helper functions for benchmarks

func createBenchmarkServer(b *testing.B) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/chat":
			var req chatRequest
			json.NewDecoder(r.Body).Decode(&req)

			response := chatResponse{
				Model:     req.Model,
				CreatedAt: time.Now(),
				Message: &chatMessage{
					Role:    "assistant",
					Content: "Benchmark response for performance testing.",
				},
				Done:               true,
				PromptEvalCount:    10,
				EvalCount:          8,
				TotalDuration:      500000000, // 500ms
				LoadDuration:       50000000,  // 50ms
				PromptEvalDuration: 200000000, // 200ms
				EvalDuration:       250000000, // 250ms
			}

			// Handle structured output
			if req.Format != "" {
				response.Message.Content = `{"name": "Benchmark User", "age": 30, "city": "Test City"}`
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)

		case "/api/tags":
			response := modelsResponse{
				Models: []model{
					{Name: "llama3.2", Size: 12345678, ModifiedAt: time.Now()},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)

		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
}

func createStreamingBenchmarkServer(b *testing.B) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/x-ndjson")

		// Simulate streaming with multiple chunks
		chunks := []chatResponse{
			{Model: "llama3.2", CreatedAt: time.Now(), Message: &chatMessage{Role: "assistant", Content: "Streaming "}, Done: false},
			{Model: "llama3.2", CreatedAt: time.Now(), Message: &chatMessage{Role: "assistant", Content: "benchmark "}, Done: false},
			{Model: "llama3.2", CreatedAt: time.Now(), Message: &chatMessage{Role: "assistant", Content: "response."}, Done: false},
			{
				Model:              "llama3.2",
				CreatedAt:          time.Now(),
				Message:            &chatMessage{Role: "assistant", Content: ""},
				Done:               true,
				PromptEvalCount:    12,
				EvalCount:          15,
				TotalDuration:      750000000,
				LoadDuration:       75000000,
				PromptEvalDuration: 300000000,
				EvalDuration:       375000000,
			},
		}

		for _, chunk := range chunks {
			json.NewEncoder(w).Encode(chunk)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
}

// Performance comparison benchmarks

func BenchmarkCompareResponseSizes(b *testing.B) {
	sizes := []struct {
		name     string
		content  string
	}{
		{"Small", "Short response"},
		{"Medium", strings.Repeat("Medium length response with more content. ", 10)},
		{"Large", strings.Repeat("This is a much larger response with significantly more content to test parsing performance with larger payloads. ", 50)},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			responseJSON := createResponseJSON(size.content)
			
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var response chatResponse
				err := json.Unmarshal([]byte(responseJSON), &response)
				if err != nil {
					b.Fatalf("JSON unmarshal failed: %v", err)
				}
			}
		})
	}
}

func createResponseJSON(content string) string {
	response := chatResponse{
		Model:     "llama3.2",
		CreatedAt: time.Now(),
		Message: &chatMessage{
			Role:    "assistant",
			Content: content,
		},
		Done:            true,
		PromptEvalCount: 20,
		EvalCount:       len(content) / 4, // Rough token estimate
	}

	jsonBytes, _ := json.Marshal(response)
	return string(jsonBytes)
}

// Memory pressure benchmarks

func BenchmarkMemoryPressure(b *testing.B) {
	provider := New(WithModel("llama3.2"))

	// Create requests of varying sizes
	sizes := []int{1, 10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Messages_%d", size), func(b *testing.B) {
			var messages []core.Message
			for i := 0; i < size; i++ {
				messages = append(messages, core.Message{
					Role:  core.User,
					Parts: []core.Part{core.Text{Text: fmt.Sprintf("Message %d", i)}},
				})
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := provider.convertMessages(messages)
				if err != nil {
					b.Fatalf("convertMessages failed: %v", err)
				}
			}
		})
	}
}