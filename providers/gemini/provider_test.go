package gemini

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
	"github.com/recera/gai/tools"
)

// mockGeminiServer creates a mock Gemini API server for testing.
func mockGeminiServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check API key
		apiKey := r.URL.Query().Get("key")
		if apiKey != "test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
					Status  string `json:"status"`
					Details []struct {
						Type     string            `json:"@type"`
						Reason   string            `json:"reason"`
						Domain   string            `json:"domain"`
						Metadata map[string]string `json:"metadata"`
					} `json:"details"`
				}{
					Code:    401,
					Message: "API key not valid",
					Status:  "UNAUTHENTICATED",
				},
			})
			return
		}

		// Route based on path
		path := r.URL.Path
		
		// Handle file uploads
		if strings.Contains(path, "/upload/") && strings.Contains(path, "/files") {
			handleFileUpload(w, r)
			return
		}

		// Handle text generation
		if strings.Contains(path, ":generateContent") {
			handleGenerateContent(w, r)
			return
		}

		// Handle streaming
		if strings.Contains(path, ":streamGenerateContent") {
			handleStreamGenerateContent(w, r)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Return mock file response
	response := map[string]any{
		"file": map[string]string{
			"name": "files/mock-file-123",
			"uri":  "https://generativelanguage.googleapis.com/v1beta/files/mock-file-123",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleGenerateContent(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req GenerateContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check for safety trigger
	for _, content := range req.Contents {
		for _, part := range content.Parts {
			if strings.Contains(part.Text, "unsafe content") {
				// Return safety block
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(GenerateContentResponse{
					PromptFeedback: &PromptFeedback{
						BlockReason: "SAFETY",
						SafetyRatings: []SafetyRating{
							{
								Category:    "HARM_CATEGORY_DANGEROUS_CONTENT",
								Probability: "HIGH",
								Blocked:     true,
							},
						},
					},
				})
				return
			}
		}
	}

	// Check for tool calls
	var response GenerateContentResponse
	
	if len(req.Tools) > 0 {
		// Simulate tool call
		response = GenerateContentResponse{
			Candidates: []Candidate{
				{
					Content: Content{
						Role: "model",
						Parts: []Part{
							{
								Text: "I'll help you with that. Let me check the weather.",
							},
							{
								FunctionCall: &FunctionCall{
									Name: "get_weather",
									Args: json.RawMessage(`{"location":"San Francisco"}`),
								},
							},
						},
					},
					FinishReason: "STOP",
					SafetyRatings: []SafetyRating{
						{
							Category:    "HARM_CATEGORY_HARASSMENT",
							Probability: "NEGLIGIBLE",
							Blocked:     false,
						},
					},
				},
			},
			UsageMetadata: &UsageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 15,
				TotalTokenCount:      25,
			},
		}
	} else {
		// Regular text response
		responseText := "This is a test response from Gemini."
		
		// Add citations if requested
		var citationMetadata *CitationMetadata
		for _, content := range req.Contents {
			for _, part := range content.Parts {
				if strings.Contains(part.Text, "with citations") {
					citationMetadata = &CitationMetadata{
						CitationSources: []CitationSource{
							{
								StartIndex: 0,
								EndIndex:   10,
								URI:        "https://example.com/source1",
								Title:      "Example Source",
							},
						},
					}
					responseText = "According to research, this is a cited response."
				}
			}
		}

		response = GenerateContentResponse{
			Candidates: []Candidate{
				{
					Content: Content{
						Role: "model",
						Parts: []Part{
							{Text: responseText},
						},
					},
					FinishReason:     "STOP",
					CitationMetadata: citationMetadata,
					SafetyRatings: []SafetyRating{
						{
							Category:    "HARM_CATEGORY_HARASSMENT",
							Probability: "NEGLIGIBLE",
							Blocked:     false,
						},
						{
							Category:    "HARM_CATEGORY_HATE_SPEECH",
							Probability: "NEGLIGIBLE",
							Blocked:     false,
						},
					},
				},
			},
			UsageMetadata: &UsageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 20,
				TotalTokenCount:      30,
			},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleStreamGenerateContent(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req GenerateContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Send streaming chunks
	chunks := []string{
		"Hello ",
		"from ",
		"streaming ",
		"Gemini!",
	}

	for i, chunk := range chunks {
		response := StreamingResponse{
			Candidates: []Candidate{
				{
					Content: Content{
						Role: "model",
						Parts: []Part{
							{Text: chunk},
						},
					},
					SafetyRatings: []SafetyRating{
						{
							Category:    "HARM_CATEGORY_HARASSMENT",
							Probability: "NEGLIGIBLE",
							Blocked:     false,
						},
					},
				},
			},
		}

		// Add usage on last chunk
		if i == len(chunks)-1 {
			response.UsageMetadata = &UsageMetadata{
				PromptTokenCount:     5,
				CandidatesTokenCount: 10,
				TotalTokenCount:      15,
			}
		}

		data, _ := json.Marshal(response)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		
		time.Sleep(10 * time.Millisecond) // Simulate streaming delay
	}

	// Send done signal
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func TestProviderCreation(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		want struct {
			apiKey     string
			baseURL    string
			model      string
			maxRetries int
		}
	}{
		{
			name: "default provider",
			opts: []Option{
				WithAPIKey("test-key"),
			},
			want: struct {
				apiKey     string
				baseURL    string
				model      string
				maxRetries int
			}{
				apiKey:     "test-key",
				baseURL:    defaultBaseURL,
				model:      "gemini-1.5-flash",
				maxRetries: 3,
			},
		},
		{
			name: "custom configuration",
			opts: []Option{
				WithAPIKey("custom-key"),
				WithBaseURL("https://custom.api.com"),
				WithModel("gemini-1.5-pro"),
				WithMaxRetries(5),
			},
			want: struct {
				apiKey     string
				baseURL    string
				model      string
				maxRetries int
			}{
				apiKey:     "custom-key",
				baseURL:    "https://custom.api.com",
				model:      "gemini-1.5-pro",
				maxRetries: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.opts...)
			
			if p.apiKey != tt.want.apiKey {
				t.Errorf("apiKey = %q, want %q", p.apiKey, tt.want.apiKey)
			}
			if p.baseURL != tt.want.baseURL {
				t.Errorf("baseURL = %q, want %q", p.baseURL, tt.want.baseURL)
			}
			if p.model != tt.want.model {
				t.Errorf("model = %q, want %q", p.model, tt.want.model)
			}
			if p.maxRetries != tt.want.maxRetries {
				t.Errorf("maxRetries = %d, want %d", p.maxRetries, tt.want.maxRetries)
			}
		})
	}
}

func TestGenerateText(t *testing.T) {
	server := mockGeminiServer()
	defer server.Close()

	provider := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
		WithModel("gemini-1.5-flash"),
	)

	tests := []struct {
		name    string
		request core.Request
		wantErr bool
		check   func(*testing.T, *core.TextResult)
	}{
		{
			name: "simple text generation",
			request: core.Request{
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "Hello, Gemini!"},
						},
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result *core.TextResult) {
				if result.Text != "This is a test response from Gemini." {
					t.Errorf("unexpected response: %q", result.Text)
				}
				if result.Usage.TotalTokens != 30 {
					t.Errorf("unexpected total tokens: %d", result.Usage.TotalTokens)
				}
			},
		},
		{
			name: "with system message",
			request: core.Request{
				Messages: []core.Message{
					{
						Role: core.System,
						Parts: []core.Part{
							core.Text{Text: "You are a helpful assistant."},
						},
					},
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "Hello!"},
						},
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result *core.TextResult) {
				if result.Text == "" {
					t.Error("expected non-empty response")
				}
			},
		},
		{
			name: "with citations",
			request: core.Request{
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "Tell me something with citations"},
						},
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result *core.TextResult) {
				if !strings.Contains(result.Text, "cited response") {
					t.Errorf("expected cited response, got: %q", result.Text)
				}
			},
		},
		{
			name: "safety blocking",
			request: core.Request{
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "Generate unsafe content"},
						},
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result *core.TextResult) {
				if result.Text != "" {
					t.Errorf("expected empty response for blocked content, got: %q", result.Text)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := provider.GenerateText(ctx, tt.request)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.check != nil && result != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestStreamText(t *testing.T) {
	server := mockGeminiServer()
	defer server.Close()

	provider := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
		WithModel("gemini-1.5-flash"),
	)

	ctx := context.Background()
	stream, err := provider.StreamText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Stream a response"},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("StreamText() error = %v", err)
	}
	defer stream.Close()

	var events []core.Event
	for event := range stream.Events() {
		events = append(events, event)
	}

	// Check we got expected events
	hasStart := false
	hasText := false
	hasFinish := false
	totalText := ""

	for _, event := range events {
		switch event.Type {
		case core.EventStart:
			hasStart = true
		case core.EventTextDelta:
			hasText = true
			totalText += event.TextDelta
		case core.EventFinish:
			hasFinish = true
			if event.Usage == nil {
				t.Error("expected usage in finish event")
			}
		}
	}

	if !hasStart {
		t.Error("missing start event")
	}
	if !hasText {
		t.Error("missing text events")
	}
	if !hasFinish {
		t.Error("missing finish event")
	}
	if totalText != "Hello from streaming Gemini!" {
		t.Errorf("unexpected streamed text: %q", totalText)
	}
}

func TestToolCalling(t *testing.T) {
	server := mockGeminiServer()
	defer server.Close()

	provider := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
		WithModel("gemini-1.5-flash"),
	)

	// Create a test tool
	weatherTool := tools.New[struct{ Location string }, struct{ Temperature float64 }](
		"get_weather",
		"Get weather for a location",
		func(ctx context.Context, input struct{ Location string }, meta tools.Meta) (struct{ Temperature float64 }, error) {
			return struct{ Temperature float64 }{Temperature: 72.5}, nil
		},
	)

	ctx := context.Background()
	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What's the weather?"},
				},
			},
		},
		Tools: []core.ToolHandle{
			tools.NewCoreAdapter(weatherTool),
		},
		ToolChoice: core.ToolAuto,
	})

	if err != nil {
		t.Fatalf("GenerateText() with tools error = %v", err)
	}

	if len(result.Steps) == 0 {
		t.Error("expected at least one step")
	}

	// Check for tool call in steps
	hasToolCall := false
	for _, step := range result.Steps {
		if len(step.ToolCalls) > 0 {
			hasToolCall = true
			if step.ToolCalls[0].Name != "get_weather" {
				t.Errorf("unexpected tool name: %q", step.ToolCalls[0].Name)
			}
		}
	}

	if !hasToolCall {
		t.Error("expected tool call in steps")
	}
}

func TestErrorHandling(t *testing.T) {
	server := mockGeminiServer()
	defer server.Close()

	tests := []struct {
		name      string
		apiKey    string
		wantError core.ErrorCode
	}{
		{
			name:      "invalid API key",
			apiKey:    "invalid-key",
			wantError: core.ErrorUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := New(
				WithAPIKey(tt.apiKey),
				WithBaseURL(server.URL),
			)

			ctx := context.Background()
			_, err := provider.GenerateText(ctx, core.Request{
				Messages: []core.Message{
					{
						Role: core.User,
						Parts: []core.Part{
							core.Text{Text: "Hello"},
						},
					},
				},
			})

			if err == nil {
				t.Fatal("expected error but got none")
			}

			aiErr, ok := err.(*core.AIError)
			if !ok {
				t.Fatalf("expected AIError, got %T", err)
			}

			if aiErr.Code != tt.wantError {
				t.Errorf("error code = %v, want %v", aiErr.Code, tt.wantError)
			}
		})
	}
}

func TestSafetyConfiguration(t *testing.T) {
	server := mockGeminiServer()
	defer server.Close()

	provider := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
		WithDefaultSafety(&core.SafetyConfig{
			Harassment: core.SafetyBlockMost,
			Hate:       core.SafetyBlockMost,
			Sexual:     core.SafetyBlockMost,
			Dangerous:  core.SafetyBlockFew,
		}),
	)

	ctx := context.Background()
	result, err := provider.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Hello"},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("GenerateText() error = %v", err)
	}

	if result.Text == "" {
		t.Error("expected non-empty response")
	}
}

func TestStructuredOutput(t *testing.T) {
	server := mockGeminiServer()
	defer server.Close()

	provider := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
	)

	type Recipe struct {
		Name        string   `json:"name"`
		Ingredients []string `json:"ingredients"`
	}

	ctx := context.Background()
	
	// Mock server returns regular text, so we'll get a parsing error
	// In a real test, the mock would return proper JSON
	_, err := provider.GenerateObject(ctx, core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Generate a recipe"},
				},
			},
		},
	}, Recipe{})

	// We expect an error since the mock doesn't return JSON
	if err == nil {
		t.Error("expected error parsing non-JSON response")
	}
}