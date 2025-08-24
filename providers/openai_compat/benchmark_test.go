package openai_compat

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/recera/gai/core"
	"github.com/recera/gai/tools"
)

// benchmarkServer creates a minimal mock server for benchmarking.
func benchmarkServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			// Return minimal response quickly
			resp := chatCompletionResponse{
				ID:      "bench",
				Choices: []choice{{Message: chatMessage{Content: "benchmark response"}}},
				Usage:   usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
			}
			json.NewEncoder(w).Encode(resp)
			
		case "/v1/models":
			// Return minimal models list
			resp := modelsResponse{
				Data: []ModelInfo{{ID: "test-model", ContextWindow: 8192}},
			}
			json.NewEncoder(w).Encode(resp)
			
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func BenchmarkProviderCreation(b *testing.B) {
	server := benchmarkServer()
	defer server.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := New(CompatOpts{
			BaseURL:      server.URL,
			APIKey:       "bench-key",
			DefaultModel: "test-model",
			ProviderName: "bench",
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateText(b *testing.B) {
	server := benchmarkServer()
	defer server.Close()
	
	provider, err := New(CompatOpts{
		BaseURL:      server.URL,
		APIKey:       "bench-key",
		DefaultModel: "test-model",
		ProviderName: "bench",
	})
	if err != nil {
		b.Fatal(err)
	}
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Hello"}}},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GenerateText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRequestConversion(b *testing.B) {
	provider, err := New(CompatOpts{
		BaseURL:      "https://api.example.com",
		APIKey:       "bench-key",
		DefaultModel: "test-model",
	})
	if err != nil {
		b.Fatal(err)
	}
	
	req := core.Request{
		Model: "test-model",
		Messages: []core.Message{
			{Role: core.System, Parts: []core.Part{core.Text{Text: "You are helpful."}}},
			{Role: core.User, Parts: []core.Part{
				core.Text{Text: "Hello"},
				core.ImageURL{URL: "https://example.com/image.jpg"},
			}},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.convertRequest(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessageConversion(b *testing.B) {
	provider, err := New(CompatOpts{
		BaseURL: "https://api.example.com",
		APIKey:  "bench-key",
	})
	if err != nil {
		b.Fatal(err)
	}
	
	messages := []core.Message{
		{Role: core.System, Parts: []core.Part{core.Text{Text: "System prompt"}}},
		{Role: core.User, Parts: []core.Part{core.Text{Text: "User message"}}},
		{Role: core.Assistant, Parts: []core.Part{core.Text{Text: "Assistant response"}}},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.convertMessages(messages)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkToolConversion(b *testing.B) {
	provider, err := New(CompatOpts{
		BaseURL: "https://api.example.com",
		APIKey:  "bench-key",
	})
	if err != nil {
		b.Fatal(err)
	}
	
	// Create test tools
	type Input struct {
		Query string `json:"query"`
	}
	type Output struct {
		Result string `json:"result"`
	}
	
	tool := tools.New[Input, Output](
		"search", "Search for information",
		func(ctx context.Context, in Input, meta tools.Meta) (Output, error) {
			return Output{Result: "result"}, nil
		},
	)
	
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Search for AI"}}},
		},
		Tools: []core.ToolHandle{tools.NewCoreAdapter(tool)},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		apiReq, err := provider.convertRequest(req)
		if err != nil {
			b.Fatal(err)
		}
		if len(apiReq.Tools) == 0 {
			b.Fatal("Expected tools in converted request")
		}
	}
}

func BenchmarkErrorMapping(b *testing.B) {
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{"Retry-After": []string{"60"}},
		Body: nil, // Will be set in loop
	}
	
	errorBody := errorResponse{
		Error: struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code,omitempty"`
			Param   string `json:"param,omitempty"`
		}{
			Message: "Rate limit exceeded",
			Type:    "rate_limit_error",
			Code:    "rate_limit_exceeded",
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset body for each iteration
		bodyBytes, _ := json.Marshal(errorBody)
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		
		err := MapError(resp, "bench")
		if err == nil {
			b.Fatal("Expected error from MapError")
		}
	}
}

func BenchmarkParameterStripping(b *testing.B) {
	provider, err := New(CompatOpts{
		BaseURL:           "https://api.example.com",
		APIKey:            "bench-key",
		UnsupportedParams: []string{"seed", "top_p", "logit_bias"},
	})
	if err != nil {
		b.Fatal(err)
	}
	
	temp := float32(0.7)
	maxTok := 100
	seed := 42
	topP := float32(0.9)
	
	req := &chatCompletionRequest{
		Model:       "test",
		Temperature: &temp,
		MaxTokens:   &maxTok,
		Seed:        &seed,
		TopP:        &topP,
		LogitBias:   map[string]float32{"hello": 0.5},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stripped := provider.stripUnsupportedParams(req)
		if stripped.Seed != nil || stripped.TopP != nil || stripped.LogitBias != nil {
			b.Fatal("Failed to strip unsupported params")
		}
	}
}

func BenchmarkJSONSchemaGeneration(b *testing.B) {
	provider, err := New(CompatOpts{
		BaseURL: "https://api.example.com",
		APIKey:  "bench-key",
	})
	if err != nil {
		b.Fatal(err)
	}
	
	type TestStruct struct {
		Name        string   `json:"name"`
		Age         int      `json:"age"`
		Tags        []string `json:"tags"`
		Active      bool     `json:"active"`
		Score       float64  `json:"score,omitempty"`
	}
	
	schema := TestStruct{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.generateJSONSchema(schema)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStreamProcessing(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		
		// Send a few chunks
		chunks := []string{
			`{"choices":[{"delta":{"content":"Hello"}}]}`,
			`{"choices":[{"delta":{"content":" world"}}]}`,
			`{"choices":[{"finish_reason":"stop"}],"usage":{"total_tokens":10}}`,
		}
		
		for _, chunk := range chunks {
			w.Write([]byte("data: " + chunk + "\n\n"))
		}
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()
	
	provider, err := New(CompatOpts{
		BaseURL:      server.URL,
		APIKey:       "bench-key",
		ProviderName: "bench",
	})
	if err != nil {
		b.Fatal(err)
	}
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Hello"}}},
		},
		Stream: true,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := provider.StreamText(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
		
		// Consume stream
		for range stream.Events() {
			// Just consume events
		}
		stream.Close()
	}
}

func BenchmarkPresetCreation(b *testing.B) {
	benchmarks := []struct {
		name   string
		create func() (*Provider, error)
	}{
		{
			name:   "Groq",
			create: func() (*Provider, error) { return Groq() },
		},
		{
			name:   "XAI",
			create: func() (*Provider, error) { return XAI() },
		},
		{
			name:   "Cerebras",
			create: func() (*Provider, error) { return Cerebras() },
		},
		{
			name:   "Together",
			create: func() (*Provider, error) { return Together() },
		},
	}
	
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := bm.create()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParallelRequests(b *testing.B) {
	server := benchmarkServer()
	defer server.Close()
	
	provider, err := New(CompatOpts{
		BaseURL:      server.URL,
		APIKey:       "bench-key",
		DefaultModel: "test-model",
		ProviderName: "bench",
	})
	if err != nil {
		b.Fatal(err)
	}
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Hello"}}},
		},
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := provider.GenerateText(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

