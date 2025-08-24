package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

// TestIntegrationWithMockServer tests the full integration flow with a mock Ollama server.
func TestIntegrationWithMockServer(t *testing.T) {
	// Create a comprehensive mock server that handles all endpoints
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/chat":
			handleMockChat(t, w, r)
		case "/api/generate":
			handleMockGenerate(t, w, r)
		case "/api/tags":
			handleMockTags(t, w, r)
		case "/api/pull":
			handleMockPull(t, w, r)
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Test suite with the mock server
	t.Run("ChatAPI", func(t *testing.T) {
		testChatAPI(t, server.URL)
	})
	
	t.Run("GenerateAPI", func(t *testing.T) {
		testGenerateAPI(t, server.URL)
	})
	
	t.Run("StreamingAPI", func(t *testing.T) {
		testStreamingAPI(t, server.URL)
	})
	
	t.Run("ToolsAPI", func(t *testing.T) {
		testToolsAPI(t, server.URL)
	})
	
	t.Run("StructuredOutputAPI", func(t *testing.T) {
		testStructuredOutputAPI(t, server.URL)
	})
	
	t.Run("ModelManagement", func(t *testing.T) {
		testModelManagement(t, server.URL)
	})
	
	t.Run("ErrorHandling", func(t *testing.T) {
		testErrorHandling(t, server.URL)
	})
}

func handleMockChat(t *testing.T, w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Handle different scenarios based on request content
	if len(req.Messages) > 0 && strings.Contains(req.Messages[len(req.Messages)-1].Content, "error") {
		http.Error(w, `{"error": "model not found"}`, http.StatusBadRequest)
		return
	}

	// Check if streaming is requested
	isStreaming := req.Stream != nil && *req.Stream

	if isStreaming {
		// Stream response
		w.Header().Set("Content-Type", "application/x-ndjson")
		
		chunks := []chatResponse{
			{
				Model:     req.Model,
				CreatedAt: time.Now(),
				Message:   &chatMessage{Role: "assistant", Content: "Hello"},
				Done:      false,
			},
			{
				Model:     req.Model,
				CreatedAt: time.Now(),
				Message:   &chatMessage{Role: "assistant", Content: " world"},
				Done:      false,
			},
			{
				Model:             req.Model,
				CreatedAt:         time.Now(),
				Message:           &chatMessage{Role: "assistant", Content: "!"},
				Done:              true,
				PromptEvalCount:   10,
				EvalCount:         5,
				TotalDuration:     1000000000,
				LoadDuration:      100000000,
				PromptEvalDuration: 500000000,
				EvalDuration:       400000000,
			},
		}

		for _, chunk := range chunks {
			json.NewEncoder(w).Encode(chunk)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	} else {
		// Single response
		response := chatResponse{
			Model:     req.Model,
			CreatedAt: time.Now(),
			Message: &chatMessage{
				Role:    "assistant",
				Content: generateMockResponse(req),
			},
			Done:               true,
			PromptEvalCount:    10,
			EvalCount:          5,
			TotalDuration:      1000000000,
			LoadDuration:       100000000,
			PromptEvalDuration: 500000000,
			EvalDuration:       400000000,
		}

		// Handle tool calls
		if len(req.Tools) > 0 && strings.Contains(req.Messages[len(req.Messages)-1].Content, "weather") {
			response.Message.ToolCalls = []toolCall{
				{
					ID:   "call_123",
					Type: "function",
					Function: functionCall{
						Name:      "get_weather",
						Arguments: `{"location": "New York"}`,
					},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func handleMockGenerate(t *testing.T, w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if streaming is requested
	isStreaming := req.Stream != nil && *req.Stream

	if isStreaming {
		// Stream response
		w.Header().Set("Content-Type", "application/x-ndjson")
		
		chunks := []generateResponse{
			{
				Model:     req.Model,
				CreatedAt: time.Now(),
				Response:  "Generated ",
				Done:      false,
			},
			{
				Model:     req.Model,
				CreatedAt: time.Now(),
				Response:  "response ",
				Done:      false,
			},
			{
				Model:              req.Model,
				CreatedAt:          time.Now(),
				Response:           "complete.",
				Done:               true,
				PromptEvalCount:    8,
				EvalCount:          6,
				TotalDuration:      1500000000,
				LoadDuration:       150000000,
				PromptEvalDuration: 600000000,
				EvalDuration:       750000000,
			},
		}

		for _, chunk := range chunks {
			json.NewEncoder(w).Encode(chunk)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	} else {
		// Single response
		response := generateResponse{
			Model:              req.Model,
			CreatedAt:          time.Now(),
			Response:           "Generated response complete.",
			Done:               true,
			PromptEvalCount:    8,
			EvalCount:          6,
			TotalDuration:      1500000000,
			LoadDuration:       150000000,
			PromptEvalDuration: 600000000,
			EvalDuration:       750000000,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func handleMockTags(t *testing.T, w http.ResponseWriter, r *http.Request) {
	response := modelsResponse{
		Models: []model{
			{
				Name:       "llama3.2",
				Size:       12345678,
				Digest:     "sha256:abc123",
				ModifiedAt: time.Now(),
				Details:    map[string]string{"format": "gguf", "family": "llama"},
			},
			{
				Name:       "mistral:7b",
				Size:       87654321,
				Digest:     "sha256:def456",
				ModifiedAt: time.Now(),
				Details:    map[string]string{"format": "gguf", "family": "mistral"},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleMockPull(t *testing.T, w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req["name"] == "nonexistent" {
		http.Error(w, `{"error": "model not found"}`, http.StatusNotFound)
		return
	}

	// Simulate successful pull initiation
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "pulling manifest"}`))
}

func generateMockResponse(req chatRequest) string {
	lastMessage := req.Messages[len(req.Messages)-1].Content
	
	// Generate contextual responses based on input
	switch {
	case strings.Contains(lastMessage, "hello"):
		return "Hello! How can I help you today?"
	case strings.Contains(lastMessage, "weather"):
		return "I can help you check the weather. Let me use a tool for that."
	case req.Format != "":
		// For structured output, return JSON
		if strings.Contains(lastMessage, "person") {
			return `{"name": "Alice", "age": 25, "city": "Boston"}`
		}
		return `{"result": "structured output"}`
	default:
		return "I understand your message and I'm here to help."
	}
}

func testChatAPI(t *testing.T, baseURL string) {
	provider := New(WithBaseURL(baseURL), WithModel("llama3.2"))

	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Hello, how are you?"}}},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	result, err := provider.GenerateText(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateText failed: %v", err)
	}

	if result.Text == "" {
		t.Error("Expected non-empty response text")
	}

	if result.Usage.InputTokens == 0 {
		t.Error("Expected non-zero input tokens")
	}

	if result.Usage.OutputTokens == 0 {
		t.Error("Expected non-zero output tokens")
	}

	t.Logf("Response: %s", result.Text)
	t.Logf("Usage: %+v", result.Usage)
}

func testGenerateAPI(t *testing.T, baseURL string) {
	provider := New(WithBaseURL(baseURL), WithModel("llama3.2"), WithGenerateAPI(true))

	req := core.Request{
		Messages: []core.Message{
			{Role: core.System, Parts: []core.Part{core.Text{Text: "You are a helpful assistant."}}},
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Generate a short story."}}},
		},
		Temperature: 0.8,
		MaxTokens:   200,
	}

	result, err := provider.generateUsingGenerateAPI(context.Background(), req)
	if err != nil {
		t.Fatalf("generateUsingGenerateAPI failed: %v", err)
	}

	if result.Text == "" {
		t.Error("Expected non-empty response text")
	}

	t.Logf("Generated text: %s", result.Text)
}

func testStreamingAPI(t *testing.T, baseURL string) {
	provider := New(WithBaseURL(baseURL), WithModel("llama3.2"))

	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Tell me a joke."}}},
		},
	}

	stream, err := provider.StreamText(context.Background(), req)
	if err != nil {
		t.Fatalf("StreamText failed: %v", err)
	}
	defer stream.Close()

	var textParts []string
	var eventCount int

	for event := range stream.Events() {
		eventCount++
		switch event.Type {
		case core.EventStart:
			t.Log("Stream started")
		case core.EventTextDelta:
			textParts = append(textParts, event.TextDelta)
			t.Logf("Text delta: %s", event.TextDelta)
		case core.EventFinish:
			t.Logf("Stream finished with usage: %+v", event.Usage)
		case core.EventError:
			t.Errorf("Stream error: %v", event.Err)
		}
	}

	if eventCount == 0 {
		t.Error("Expected to receive events from stream")
	}

	fullText := strings.Join(textParts, "")
	if fullText == "" {
		t.Error("Expected non-empty streamed text")
	}

	t.Logf("Full streamed text: %s", fullText)
}

func testToolsAPI(t *testing.T, baseURL string) {
	provider := New(WithBaseURL(baseURL), WithModel("llama3.2"))

	// Create a mock weather tool
	weatherTool := &mockWeatherTool{}

	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "What's the weather like?"}}},
		},
		Tools: []core.ToolHandle{weatherTool},
	}

	result, err := provider.GenerateText(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateText with tools failed: %v", err)
	}

	if result.Text == "" {
		t.Error("Expected non-empty response text")
	}

	// Should have steps with tool calls
	if len(result.Steps) == 0 {
		t.Error("Expected steps with tool calls")
	} else {
		step := result.Steps[0]
		if len(step.ToolCalls) == 0 {
			t.Error("Expected tool calls in step")
		}
		t.Logf("Tool calls: %+v", step.ToolCalls)
	}

	t.Logf("Response with tools: %s", result.Text)
}

func testStructuredOutputAPI(t *testing.T, baseURL string) {
	provider := New(WithBaseURL(baseURL), WithModel("llama3.2"))

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

	result, err := provider.GenerateObject(context.Background(), req, schema)
	if err != nil {
		t.Fatalf("GenerateObject failed: %v", err)
	}

	obj, ok := result.Value.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected object result, got %T", result.Value)
	}

	if obj["name"] == nil {
		t.Error("Expected name field in object")
	}

	if obj["age"] == nil {
		t.Error("Expected age field in object")
	}

	t.Logf("Generated object: %+v", obj)
}

func testModelManagement(t *testing.T, baseURL string) {
	provider := New(WithBaseURL(baseURL))

	// List models
	models, err := provider.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) == 0 {
		t.Error("Expected at least one model")
	}

	t.Logf("Available models: %d", len(models))
	for _, model := range models {
		t.Logf("- %s (size: %d bytes)", model.Name, model.Size)
	}

	// Check model availability
	available, err := provider.IsModelAvailable(context.Background(), "llama3.2")
	if err != nil {
		t.Fatalf("IsModelAvailable failed: %v", err)
	}

	if !available {
		t.Error("Expected llama3.2 to be available")
	}

	// Check unavailable model
	available, err = provider.IsModelAvailable(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("IsModelAvailable failed: %v", err)
	}

	if available {
		t.Error("Expected nonexistent model to be unavailable")
	}

	// Test pull model (successful)
	err = provider.PullModel(context.Background(), "llama3.1")
	if err != nil {
		t.Fatalf("PullModel failed: %v", err)
	}

	t.Log("Model pull initiated successfully")
}

func testErrorHandling(t *testing.T, baseURL string) {
	provider := New(WithBaseURL(baseURL), WithModel("llama3.2"))

	// Test error response from server
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "trigger error"}}},
		},
	}

	result, err := provider.GenerateText(context.Background(), req)
	if err == nil {
		t.Error("Expected error from server, got nil")
	}
	if result != nil {
		t.Error("Expected nil result on error")
	}

	// Verify error type
	if !IsModelNotFoundError(err) {
		t.Errorf("Expected model not found error, got: %v", err)
	}

	t.Logf("Error correctly handled: %v", err)
}

// Mock tool for testing tool calling
type mockWeatherTool struct{}

func (m *mockWeatherTool) Name() string {
	return "get_weather"
}

func (m *mockWeatherTool) Description() string {
	return "Get current weather information for a location"
}

func (m *mockWeatherTool) InSchemaJSON() []byte {
	return []byte(`{
		"type": "object",
		"properties": {
			"location": {
				"type": "string",
				"description": "The location to get weather for"
			}
		},
		"required": ["location"]
	}`)
}

func (m *mockWeatherTool) OutSchemaJSON() []byte {
	return []byte(`{
		"type": "object",
		"properties": {
			"temperature": {"type": "number"},
			"condition": {"type": "string"},
			"humidity": {"type": "number"}
		}
	}`)
}

func (m *mockWeatherTool) Exec(ctx context.Context, raw json.RawMessage, meta interface{}) (any, error) {
	var input struct {
		Location string `json:"location"`
	}
	
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"temperature": 22.5,
		"condition":   "partly cloudy",
		"humidity":    0.65,
		"location":    input.Location,
	}, nil
}

// TestIntegrationLive tests against a real Ollama instance (optional, requires Ollama running)
func TestIntegrationLive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live integration test in short mode")
	}

	// Check if OLLAMA_TEST_LIVE environment variable is set
	// This allows developers to opt into live testing
	if testing.Short() {
		t.Skip("Live tests require long mode and OLLAMA_TEST_LIVE=1")
	}

	provider := New()

	// Test basic connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	models, err := provider.ListModels(ctx)
	if err != nil {
		t.Skipf("Ollama not available: %v", err)
		return
	}

	if len(models) == 0 {
		t.Skip("No models available in Ollama")
		return
	}

	t.Logf("Found %d models in live Ollama instance", len(models))

	// Use the first available model for testing
	testModel := models[0].Name
	t.Logf("Using model: %s", testModel)

	// Test simple generation
	req := core.Request{
		Model: testModel,
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "Say hello in exactly 5 words."}}},
		},
		MaxTokens: 50,
	}

	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		t.Fatalf("Live GenerateText failed: %v", err)
	}

	if result.Text == "" {
		t.Error("Expected non-empty response from live Ollama")
	}

	t.Logf("Live response: %s", result.Text)
	t.Logf("Live usage: %+v", result.Usage)
}