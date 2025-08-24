package media

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

func TestElevenLabsSynthesize(t *testing.T) {
	tests := []struct {
		name        string
		req         SpeechRequest
		serverFunc  func(w http.ResponseWriter, r *http.Request)
		expectError bool
		errorCode   core.ErrorCode
	}{
		{
			name: "successful synthesis",
			req: SpeechRequest{
				Text:   "Hello, world!",
				Voice:  "test-voice",
				Format: FormatMP3,
			},
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/v1/text-to-speech/test-voice/stream" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if r.Header.Get("xi-api-key") != "test-key" {
					t.Errorf("missing or incorrect API key")
				}

				// Return audio data
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("fake audio data"))
			},
			expectError: false,
		},
		{
			name: "unauthorized error",
			req: SpeechRequest{
				Text: "Hello",
			},
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(elevenLabsError{
					Detail: struct {
						Status  string `json:"status"`
						Message string `json:"message"`
					}{
						Status:  "unauthorized",
						Message: "Invalid API key",
					},
				})
			},
			expectError: true,
			errorCode:   core.ErrorUnauthorized,
		},
		{
			name: "rate limited",
			req: SpeechRequest{
				Text: "Hello",
			},
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Rate limited"))
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
			el := NewElevenLabs(
				WithElevenLabsAPIKey("test-key"),
				WithElevenLabsBaseURL(server.URL),
			)

			// Execute synthesis
			ctx := context.Background()
			stream, err := el.Synthesize(ctx, tt.req)

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
				defer stream.Close()

				// Read audio data
				var buf bytes.Buffer
				for chunk := range stream.Chunks() {
					buf.Write(chunk)
				}

				if buf.String() != "fake audio data" {
					t.Errorf("unexpected audio data: %s", buf.String())
				}

				// Check format
				format := stream.Format()
				if format.Encoding != FormatMP3 {
					t.Errorf("expected MP3 format, got %s", format.Encoding)
				}
			}
		})
	}
}

func TestElevenLabsListVoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/voices" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Return voice list
		response := elevenLabsVoicesResponse{
			Voices: []elevenLabsVoice{
				{
					VoiceID:     "voice1",
					Name:        "Voice One",
					Description: "Test voice",
					PreviewURL:  "https://example.com/preview1.mp3",
					Category:    "professional",
					Labels: elevenLabsVoiceLabels{
						Gender:  "female",
						Age:     "young",
						Accent:  "american",
						UseCase: "conversational",
					},
				},
				{
					VoiceID:     "voice2",
					Name:        "Voice Two",
					Description: "Another test voice",
					PreviewURL:  "https://example.com/preview2.mp3",
					Category:    "standard",
					Labels: elevenLabsVoiceLabels{
						Gender:  "male",
						Age:     "middle-aged",
						Accent:  "british",
						UseCase: "narrative",
					},
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	el := NewElevenLabs(
		WithElevenLabsAPIKey("test-key"),
		WithElevenLabsBaseURL(server.URL),
	)

	ctx := context.Background()
	voices, err := el.ListVoices(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(voices) != 2 {
		t.Errorf("expected 2 voices, got %d", len(voices))
	}

	// Check first voice
	if voices[0].ID != "voice1" {
		t.Errorf("expected voice1, got %s", voices[0].ID)
	}
	if voices[0].Gender != "female" {
		t.Errorf("expected female, got %s", voices[0].Gender)
	}
	if !voices[0].Premium {
		t.Error("expected premium voice")
	}

	// Check second voice
	if voices[1].ID != "voice2" {
		t.Errorf("expected voice2, got %s", voices[1].ID)
	}
	if voices[1].Premium {
		t.Error("expected non-premium voice")
	}
}

func TestElevenLabsStream(t *testing.T) {
	// Test the stream implementation
	reader := io.NopCloser(bytes.NewReader([]byte("test audio data")))
	stream := &elevenLabsStream{
		reader: reader,
		chunks: make(chan []byte, 100),
		format: AudioFormat{
			MIME:     MimeMP3,
			Encoding: FormatMP3,
			Bitrate:  128000,
		},
		done: make(chan struct{}),
	}

	// Start streaming
	go stream.stream()

	// Collect chunks
	var buf bytes.Buffer
	timeout := time.After(1 * time.Second)
	for {
		select {
		case chunk, ok := <-stream.Chunks():
			if !ok {
				goto done
			}
			buf.Write(chunk)
		case <-timeout:
			t.Fatal("timeout waiting for chunks")
		}
	}
done:

	if buf.String() != "test audio data" {
		t.Errorf("unexpected data: %s", buf.String())
	}

	// Check format
	format := stream.Format()
	if format.MIME != MimeMP3 {
		t.Errorf("expected audio/mpeg, got %s", format.MIME)
	}

	// Close stream
	if err := stream.Close(); err != nil {
		t.Errorf("close error: %v", err)
	}
}

func TestElevenLabsHelpers(t *testing.T) {
	el := NewElevenLabs()

	// Test stability
	if s := el.getStability(0); s != 0.5 {
		t.Errorf("expected default 0.5, got %f", s)
	}
	if s := el.getStability(-1); s != 0 {
		t.Errorf("expected 0, got %f", s)
	}
	if s := el.getStability(2); s != 1 {
		t.Errorf("expected 1, got %f", s)
	}
	if s := el.getStability(0.7); s != 0.7 {
		t.Errorf("expected 0.7, got %f", s)
	}

	// Test similarity boost
	if s := el.getSimilarityBoost(0); s != 0.75 {
		t.Errorf("expected default 0.75, got %f", s)
	}
	if s := el.getSimilarityBoost(-1); s != 0 {
		t.Errorf("expected 0, got %f", s)
	}
	if s := el.getSimilarityBoost(2); s != 1 {
		t.Errorf("expected 1, got %f", s)
	}

	// Test accept header
	if h := el.getAcceptHeader(FormatMP3); h != "audio/mpeg" {
		t.Errorf("expected audio/mpeg, got %s", h)
	}
	if h := el.getAcceptHeader(FormatPCM); h != "audio/pcm" {
		t.Errorf("expected audio/pcm, got %s", h)
	}
	if h := el.getAcceptHeader(FormatULaw); h != "audio/basic" {
		t.Errorf("expected audio/basic, got %s", h)
	}
	if h := el.getAcceptHeader("unknown"); h != "audio/mpeg" {
		t.Errorf("expected default audio/mpeg, got %s", h)
	}

	// Test audio format
	format := el.getAudioFormat(FormatMP3)
	if format.MIME != MimeMP3 {
		t.Errorf("expected %s, got %s", MimeMP3, format.MIME)
	}

	format = el.getAudioFormat(FormatPCM)
	if format.SampleRate != 44100 {
		t.Errorf("expected 44100, got %d", format.SampleRate)
	}
	if format.BitDepth != 16 {
		t.Errorf("expected 16, got %d", format.BitDepth)
	}
}

func TestElevenLabsErrorMapping(t *testing.T) {
	el := NewElevenLabs()

	tests := []struct {
		statusCode int
		body       []byte
		expectCode core.ErrorCode
		expectTemp bool
	}{
		{
			statusCode: http.StatusUnauthorized,
			body:       []byte(`{"detail":{"status":"unauthorized","message":"Invalid API key"}}`),
			expectCode: core.ErrorUnauthorized,
			expectTemp: false,
		},
		{
			statusCode: http.StatusTooManyRequests,
			body:       []byte("rate limited"),
			expectCode: core.ErrorRateLimited,
			expectTemp: true,
		},
		{
			statusCode: http.StatusInternalServerError,
			body:       []byte("internal error"),
			expectCode: core.ErrorProviderUnavailable,
			expectTemp: true,
		},
		{
			statusCode: http.StatusNotFound,
			body:       []byte("not found"),
			expectCode: core.ErrorNotFound,
			expectTemp: false,
		},
	}

	for _, tt := range tests {
		err := el.mapError(tt.statusCode, tt.body)
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

		if aiErr.Provider != "elevenlabs" {
			t.Errorf("expected provider elevenlabs, got %s", aiErr.Provider)
		}
	}
}