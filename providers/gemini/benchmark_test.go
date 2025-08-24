package gemini

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

func BenchmarkProviderCreation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = New(
			WithAPIKey("test-key"),
			WithModel("gemini-1.5-flash"),
			WithMaxRetries(3),
		)
	}
}

func BenchmarkConvertRequest(b *testing.B) {
	p := New(WithAPIKey("test"))
	req := core.Request{
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
					core.Text{Text: "Hello, how can you help me today?"},
				},
			},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.convertRequest(req)
	}
}

func BenchmarkConvertResponse(b *testing.B) {
	p := New(WithAPIKey("test"))
	resp := &GenerateContentResponse{
		Candidates: []Candidate{
			{
				Content: Content{
					Role: "model",
					Parts: []Part{
						{Text: "This is a test response with some content."},
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
			CandidatesTokenCount: 20,
			TotalTokenCount:      30,
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.convertResponse(resp)
	}
}

func BenchmarkSafetyConversion(b *testing.B) {
	safetyConfig := &core.SafetyConfig{
		Harassment: core.SafetyBlockMost,
		Hate:       core.SafetyBlockSome,
		Sexual:     core.SafetyBlockFew,
		Dangerous:  core.SafetyBlockNone,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		settings := []SafetySetting{}
		if safetyConfig.Harassment != "" {
			settings = append(settings, SafetySetting{
				Category:  "HARM_CATEGORY_HARASSMENT",
				Threshold: convertSafetyLevel(safetyConfig.Harassment),
			})
		}
		if safetyConfig.Hate != "" {
			settings = append(settings, SafetySetting{
				Category:  "HARM_CATEGORY_HATE_SPEECH",
				Threshold: convertSafetyLevel(safetyConfig.Hate),
			})
		}
		if safetyConfig.Sexual != "" {
			settings = append(settings, SafetySetting{
				Category:  "HARM_CATEGORY_SEXUALLY_EXPLICIT",
				Threshold: convertSafetyLevel(safetyConfig.Sexual),
			})
		}
		if safetyConfig.Dangerous != "" {
			settings = append(settings, SafetySetting{
				Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
				Threshold: convertSafetyLevel(safetyConfig.Dangerous),
			})
		}
	}
}

func BenchmarkCitationConversion(b *testing.B) {
	metadata := &CitationMetadata{
		CitationSources: []CitationSource{
			{
				StartIndex: 0,
				EndIndex:   10,
				URI:        "https://example.com/source1",
				Title:      "Example Source 1",
			},
			{
				StartIndex: 15,
				EndIndex:   25,
				URI:        "https://example.com/source2",
				Title:      "Example Source 2",
			},
		},
	}
	text := "This is some text with citations in various places."

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = convertCitations(metadata, text)
	}
}

func BenchmarkFileStore(b *testing.B) {
	fs := NewFileStore()
	
	// Pre-populate with some files
	for i := 0; i < 100; i++ {
		fs.Store(string(rune(i)), &FileInfo{
			ID:        string(rune(i)),
			URI:       "https://example.com/file",
			MIMEType:  "image/jpeg",
			Size:      1024,
			ExpiresAt: time.Now().Add(48 * time.Hour),
		})
	}

	b.Run("Store", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			fs.Store("test", &FileInfo{
				ID:        "test",
				URI:       "https://example.com/file",
				MIMEType:  "image/jpeg",
				Size:      1024,
				ExpiresAt: time.Now().Add(48 * time.Hour),
			})
		}
	})

	b.Run("Get", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = fs.Get("50")
		}
	})

	b.Run("Clean", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			fs.Clean()
		}
	})
}

func BenchmarkStreamEventProcessing(b *testing.B) {
	events := make(chan core.Event, 100)
	
	// Create sample streaming response
	response := StreamingResponse{
		Candidates: []Candidate{
			{
				Content: Content{
					Role: "model",
					Parts: []Part{
						{Text: "This is a streaming response chunk."},
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
		UsageMetadata: &UsageMetadata{
			PromptTokenCount:     5,
			CandidatesTokenCount: 10,
			TotalTokenCount:      15,
		},
	}

	data, _ := json.Marshal(response)
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Simulate processing a streaming chunk
		var chunk StreamingResponse
		_ = json.Unmarshal(data, &chunk)
		
		// Process and emit events
		for _, candidate := range chunk.Candidates {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					select {
					case events <- core.Event{
						Type:      core.EventTextDelta,
						TextDelta: part.Text,
						Timestamp: time.Now(),
					}:
					default:
						// Channel full, drop event
					}
				}
			}
		}
	}
}

func BenchmarkErrorMapping(b *testing.B) {
	errResp := &ErrorResponse{
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
			Code:    429,
			Message: "Rate limit exceeded",
			Status:  "RESOURCE_EXHAUSTED",
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mapError(errResp, 429)
	}
}

func BenchmarkJSONRepair(b *testing.B) {
	inputs := []string{
		`{"name": "test", "value": 123}`,
		"```json\n{\"name\": \"test\", \"value\": 123}\n```",
		`Here is the JSON: {"name": "test", "value": 123}`,
		`{"incomplete": "json"`,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repairJSON(inputs[i%len(inputs)])
	}
}

func BenchmarkParallelRequests(b *testing.B) {
	server := mockGeminiServer()
	defer server.Close()

	provider := New(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
	)

	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Hello"},
				},
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = provider.GenerateText(ctx, req)
		}
	})
}