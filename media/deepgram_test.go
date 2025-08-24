package media

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

func TestDeepgramTranscribe(t *testing.T) {
	tests := []struct {
		name        string
		req         TranscriptionRequest
		serverFunc  func(w http.ResponseWriter, r *http.Request)
		expectError bool
		errorCode   core.ErrorCode
		checkResult func(t *testing.T, result *TranscriptionResult)
	}{
		{
			name: "successful transcription with diarization",
			req: TranscriptionRequest{
				Audio: core.BlobRef{
					Kind:  core.BlobBytes,
					Bytes: []byte("fake audio data"),
					MIME:  "audio/wav",
				},
				Language:        "en",
				Punctuate:       true,
				Diarize:         true,
				FilterProfanity: false,
				Keywords:        []string{"deepgram", "test"},
				MaxAlternatives: 2,
			},
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if !strings.HasPrefix(r.URL.Path, "/v1/listen") {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if !strings.HasPrefix(r.Header.Get("Authorization"), "Token ") {
					t.Errorf("missing or incorrect authorization header")
				}

				// Check query parameters
				query := r.URL.Query()
				if query.Get("punctuate") != "true" {
					t.Errorf("expected punctuate=true")
				}
				if query.Get("diarize") != "true" {
					t.Errorf("expected diarize=true")
				}
				if query.Get("language") != "en" {
					t.Errorf("expected language=en")
				}
				if !strings.Contains(query.Get("search"), "deepgram") {
					t.Errorf("expected keywords in search")
				}

				// Return transcription
				response := deepgramResponse{
					Metadata: deepgramMetadata{
						TransactionKey: "txn123",
						RequestID:      "req123",
						Duration:       10.5,
					},
					Results: deepgramResults{
						Channels: []deepgramChannel{
							{
								DetectedLanguage: "en",
								Alternatives: []deepgramAlternative{
									{
										Transcript: "Hello, this is speaker one. And this is speaker two.",
										Confidence: 0.95,
										Words: []deepgramWord{
											{Word: "Hello", Start: 0.0, End: 0.5, Confidence: 0.98, Speaker: 0},
											{Word: "this", Start: 0.6, End: 0.8, Confidence: 0.97, Speaker: 0},
											{Word: "is", Start: 0.9, End: 1.0, Confidence: 0.96, Speaker: 0},
											{Word: "speaker", Start: 1.1, End: 1.5, Confidence: 0.95, Speaker: 0},
											{Word: "one", Start: 1.6, End: 1.8, Confidence: 0.94, Speaker: 0},
											{Word: "And", Start: 2.0, End: 2.2, Confidence: 0.93, Speaker: 1},
											{Word: "this", Start: 2.3, End: 2.5, Confidence: 0.92, Speaker: 1},
											{Word: "is", Start: 2.6, End: 2.7, Confidence: 0.91, Speaker: 1},
											{Word: "speaker", Start: 2.8, End: 3.2, Confidence: 0.90, Speaker: 1},
											{Word: "two", Start: 3.3, End: 3.5, Confidence: 0.89, Speaker: 1},
										},
									},
									{
										Transcript: "Hello, this is speaker 1. And this is speaker 2.",
										Confidence: 0.90,
									},
								},
							},
						},
						Utterances: []deepgramUtterance{
							{
								Start:      0.0,
								End:        1.8,
								Transcript: "Hello, this is speaker one.",
								Speaker:    0,
								Confidence: 0.95,
							},
							{
								Start:      2.0,
								End:        3.5,
								Transcript: "And this is speaker two.",
								Speaker:    1,
								Confidence: 0.91,
							},
						},
					},
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			},
			expectError: false,
			checkResult: func(t *testing.T, result *TranscriptionResult) {
				if result.Text != "Hello, this is speaker one. And this is speaker two." {
					t.Errorf("unexpected text: %s", result.Text)
				}
				if result.Language != "en" {
					t.Errorf("expected en, got %s", result.Language)
				}
				if result.Duration != 10500*time.Millisecond {
					t.Errorf("unexpected duration: %v", result.Duration)
				}
				if result.Confidence != 0.95 {
					t.Errorf("expected confidence 0.95, got %f", result.Confidence)
				}
				if len(result.Words) != 10 {
					t.Errorf("expected 10 words, got %d", len(result.Words))
				}
				if len(result.Alternatives) != 1 {
					t.Errorf("expected 1 alternative, got %d", len(result.Alternatives))
				}
				if len(result.Speakers) != 2 {
					t.Errorf("expected 2 speaker segments, got %d", len(result.Speakers))
				}
				// Check speaker assignment
				if result.Words[0].Speaker != 0 {
					t.Errorf("expected first word from speaker 0")
				}
				if result.Words[9].Speaker != 1 {
					t.Errorf("expected last word from speaker 1")
				}
			},
		},
		{
			name: "unauthorized error",
			req: TranscriptionRequest{
				Audio: core.BlobRef{
					Kind:  core.BlobBytes,
					Bytes: []byte("audio"),
				},
			},
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(deepgramError{
					Message: "Invalid API key",
					Code:    "AUTH_ERROR",
				})
			},
			expectError: true,
			errorCode:   core.ErrorUnauthorized,
		},
		{
			name: "rate limited",
			req: TranscriptionRequest{
				Audio: core.BlobRef{
					Kind:  core.BlobBytes,
					Bytes: []byte("audio"),
				},
			},
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(deepgramError{
					Message: "Rate limit exceeded",
					Code:    "RATE_LIMIT",
				})
			},
			expectError: true,
			errorCode:   core.ErrorRateLimited,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(tt.serverFunc))
			defer server.Close()

			// Create provider
			d := NewDeepgram(
				WithDeepgramAPIKey("test-key"),
				WithDeepgramBaseURL(server.URL),
			)

			// Execute transcription
			ctx := context.Background()
			result, err := d.Transcribe(ctx, tt.req)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				aiErr, ok := err.(*core.AIError)
				if !ok {
					t.Errorf("expected AIError, got %T", err)
					return
				}
				if aiErr.Code != tt.errorCode {
					t.Errorf("expected error code %v, got %v", tt.errorCode, aiErr.Code)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestDeepgramGetContentType(t *testing.T) {
	d := NewDeepgram()

	tests := []struct {
		blob     core.BlobRef
		expected string
	}{
		{
			blob:     core.BlobRef{MIME: "audio/mpeg"},
			expected: "audio/mpeg",
		},
		{
			blob:     core.BlobRef{MIME: "audio/wav"},
			expected: "audio/wav",
		},
		{
			blob:     core.BlobRef{MIME: ""},
			expected: "audio/wav", // default
		},
	}

	for _, tt := range tests {
		result := d.getContentType(tt.blob)
		if result != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, result)
		}
	}
}

func TestDeepgramConvertResponse(t *testing.T) {
	d := NewDeepgram()

	resp := &deepgramResponse{
		Metadata: deepgramMetadata{
			Duration: 5.5,
		},
		Results: deepgramResults{
			Channels: []deepgramChannel{
				{
					DetectedLanguage: "en",
					Alternatives: []deepgramAlternative{
						{
							Transcript: "Primary transcript",
							Confidence: 0.98,
							Words: []deepgramWord{
								{Word: "Primary", Start: 0.0, End: 0.5, Confidence: 0.99},
								{Word: "transcript", Start: 0.6, End: 1.0, Confidence: 0.97},
							},
						},
						{
							Transcript: "Alternative transcript",
							Confidence: 0.85,
						},
					},
				},
			},
			Utterances: []deepgramUtterance{
				{
					Start:      0.0,
					End:        1.0,
					Transcript: "Primary transcript",
					Speaker:    0,
				},
			},
		},
	}

	result := d.convertResponse(resp)

	if result.Text != "Primary transcript" {
		t.Errorf("unexpected text: %s", result.Text)
	}
	if result.Language != "en" {
		t.Errorf("expected en, got %s", result.Language)
	}
	if result.Duration != 5500*time.Millisecond {
		t.Errorf("unexpected duration: %v", result.Duration)
	}
	if result.Confidence != 0.98 {
		t.Errorf("expected confidence 0.98, got %f", result.Confidence)
	}
	if len(result.Words) != 2 {
		t.Errorf("expected 2 words, got %d", len(result.Words))
	}
	if len(result.Alternatives) != 1 {
		t.Errorf("expected 1 alternative, got %d", len(result.Alternatives))
	}
	if len(result.Speakers) != 1 {
		t.Errorf("expected 1 speaker segment, got %d", len(result.Speakers))
	}
}

func TestDeepgramErrorMapping(t *testing.T) {
	d := NewDeepgram()

	tests := []struct {
		statusCode int
		body       []byte
		expectCode core.ErrorCode
		expectTemp bool
	}{
		{
			statusCode: http.StatusUnauthorized,
			body:       []byte(`{"message":"Invalid API key","code":"AUTH_ERROR"}`),
			expectCode: core.ErrorUnauthorized,
			expectTemp: false,
		},
		{
			statusCode: http.StatusRequestEntityTooLarge,
			body:       []byte("File too large"),
			expectCode: core.ErrorInvalidRequest,
			expectTemp: false,
		},
		{
			statusCode: http.StatusTooManyRequests,
			body:       []byte(`{"message":"Rate limited","code":"RATE_LIMIT"}`),
			expectCode: core.ErrorRateLimited,
			expectTemp: true,
		},
		{
			statusCode: http.StatusBadGateway,
			body:       []byte("Bad gateway"),
			expectCode: core.ErrorProviderUnavailable,
			expectTemp: true,
		},
	}

	for _, tt := range tests {
		err := d.mapError(tt.statusCode, tt.body)
		if err == nil {
			t.Error("expected error, got nil")
			continue
		}

		aiErr, ok := err.(*core.AIError)
		if !ok {
			t.Errorf("expected AIError, got %T", err)
			continue
		}

		if aiErr.Code != tt.expectCode {
			t.Errorf("expected code %v, got %v", tt.expectCode, aiErr.Code)
		}

		if aiErr.Temporary != tt.expectTemp {
			t.Errorf("expected temporary %v, got %v", tt.expectTemp, aiErr.Temporary)
		}

		if aiErr.Provider != "deepgram" {
			t.Errorf("expected provider deepgram, got %s", aiErr.Provider)
		}
	}
}