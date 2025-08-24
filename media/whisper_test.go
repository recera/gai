package media

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

func TestWhisperTranscribe(t *testing.T) {
	tests := []struct {
		name        string
		req         TranscriptionRequest
		serverFunc  func(w http.ResponseWriter, r *http.Request)
		expectError bool
		errorCode   core.ErrorCode
		checkResult func(t *testing.T, result *TranscriptionResult)
	}{
		{
			name: "successful transcription",
			req: TranscriptionRequest{
				Audio: core.BlobRef{
					Kind:  core.BlobBytes,
					Bytes: []byte("fake audio data"),
					MIME:  "audio/wav",
				},
				Language: "en",
				Keywords: []string{"test", "audio"},
			},
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/v1/audio/transcriptions" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
					t.Errorf("missing or incorrect authorization header")
				}

				// Parse multipart form
				err := r.ParseMultipartForm(10 << 20) // 10MB
				if err != nil {
					t.Errorf("failed to parse multipart: %v", err)
				}

				// Check fields
				if model := r.FormValue("model"); model != "whisper-1" {
					t.Errorf("expected whisper-1, got %s", model)
				}
				if lang := r.FormValue("language"); lang != "en" {
					t.Errorf("expected en, got %s", lang)
				}
				if prompt := r.FormValue("prompt"); !strings.Contains(prompt, "test") {
					t.Errorf("expected keywords in prompt, got %s", prompt)
				}

				// Return transcription
				response := whisperResponse{
					Text:     "This is a test transcription",
					Language: "en",
					Duration: 5.2,
					Words: []whisperWord{
						{Word: "This", Start: 0.0, End: 0.5},
						{Word: "is", Start: 0.5, End: 0.7},
						{Word: "a", Start: 0.7, End: 0.8},
						{Word: "test", Start: 0.8, End: 1.2},
						{Word: "transcription", Start: 1.2, End: 2.0},
					},
					Segments: []whisperSegment{
						{
							ID:    0,
							Start: 0.0,
							End:   2.0,
							Text:  "This is a test transcription",
						},
					},
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			},
			expectError: false,
			checkResult: func(t *testing.T, result *TranscriptionResult) {
				if result.Text != "This is a test transcription" {
					t.Errorf("unexpected text: %s", result.Text)
				}
				if result.Language != "en" {
					t.Errorf("expected en, got %s", result.Language)
				}
				if result.Duration != 5200*time.Millisecond {
					t.Errorf("unexpected duration: %v", result.Duration)
				}
				if len(result.Words) != 5 {
					t.Errorf("expected 5 words, got %d", len(result.Words))
				}
				if len(result.Speakers) != 1 {
					t.Errorf("expected 1 segment, got %d", len(result.Speakers))
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
				json.NewEncoder(w).Encode(whisperError{
					Error: struct {
						Message string `json:"message"`
						Type    string `json:"type"`
						Code    string `json:"code"`
					}{
						Message: "Invalid API key",
						Type:    "invalid_request_error",
						Code:    "invalid_api_key",
					},
				})
			},
			expectError: true,
			errorCode:   core.ErrorUnauthorized,
		},
		{
			name: "file too large",
			req: TranscriptionRequest{
				Audio: core.BlobRef{
					Kind:  core.BlobBytes,
					Bytes: []byte("audio"),
				},
			},
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				w.Write([]byte("File too large"))
			},
			expectError: true,
			errorCode:   core.ErrorInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(tt.serverFunc))
			defer server.Close()

			// Create provider
			w := NewWhisper(
				WithWhisperAPIKey("test-key"),
				WithWhisperBaseURL(server.URL),
			)

			// Execute transcription
			ctx := context.Background()
			result, err := w.Transcribe(ctx, tt.req)

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

func TestWhisperTranscribeStream(t *testing.T) {
	w := NewWhisper()

	// Whisper doesn't support streaming
	ctx := context.Background()
	reader := bytes.NewReader([]byte("audio data"))
	_, err := w.TranscribeStream(ctx, reader)

	if err == nil {
		t.Error("expected error for unsupported streaming")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWhisperGetAudioData(t *testing.T) {
	w := NewWhisper()
	ctx := context.Background()

	// Test BlobBytes
	blob := core.BlobRef{
		Kind:  core.BlobBytes,
		Bytes: []byte("test audio"),
	}
	data, err := w.getAudioData(ctx, blob)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(data) != "test audio" {
		t.Errorf("unexpected data: %s", string(data))
	}

	// Test BlobURL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("downloaded audio"))
	}))
	defer server.Close()

	blob = core.BlobRef{
		Kind: core.BlobURL,
		URL:  server.URL,
	}
	data, err = w.getAudioData(ctx, blob)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(data) != "downloaded audio" {
		t.Errorf("unexpected data: %s", string(data))
	}

	// Test unsupported kind
	blob = core.BlobRef{
		Kind:   core.BlobProviderFile,
		FileID: "file123",
	}
	_, err = w.getAudioData(ctx, blob)
	if err == nil {
		t.Error("expected error for provider file")
	}
}

func TestWhisperErrorMapping(t *testing.T) {
	w := NewWhisper()

	tests := []struct {
		statusCode int
		body       []byte
		expectCode core.ErrorCode
		expectTemp bool
	}{
		{
			statusCode: http.StatusUnauthorized,
			body:       []byte(`{"error":{"message":"Invalid API key","type":"auth","code":"invalid_key"}}`),
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
			body:       []byte("Rate limited"),
			expectCode: core.ErrorRateLimited,
			expectTemp: true,
		},
		{
			statusCode: http.StatusServiceUnavailable,
			body:       []byte("Service unavailable"),
			expectCode: core.ErrorProviderUnavailable,
			expectTemp: true,
		},
	}

	for _, tt := range tests {
		err := w.mapError(tt.statusCode, tt.body)
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

		if aiErr.Provider != "whisper" {
			t.Errorf("expected provider whisper, got %s", aiErr.Provider)
		}
	}
}

// Helper function to create multipart form data for testing
func createMultipartForm(t *testing.T, fields map[string]string, files map[string][]byte) (io.Reader, string) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add fields
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("failed to write field %s: %v", key, err)
		}
	}

	// Add files
	for name, data := range files {
		part, err := writer.CreateFormFile(name, "audio.wav")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("failed to write file data: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	return &buf, writer.FormDataContentType()
}