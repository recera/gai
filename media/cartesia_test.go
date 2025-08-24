package media

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/recera/gai/core"
)

func TestCartesiaSynthesize(t *testing.T) {
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
				Text:   "Hello from Cartesia",
				Voice:  "narrator",
				Format: FormatMP3,
				Speed:  1.0,
			},
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/tts/bytes" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if r.Header.Get("X-API-Key") != "test-key" {
					t.Errorf("missing or incorrect API key")
				}
				if r.Header.Get("Cartesia-Version") != "2024-06-10" {
					t.Errorf("missing version header")
				}

				// Verify request body
				var req cartesiaRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("failed to decode request: %v", err)
				}
				if req.Text != "Hello from Cartesia" {
					t.Errorf("unexpected text: %s", req.Text)
				}
				if req.VoiceID != "narrator" {
					t.Errorf("unexpected voice: %s", req.VoiceID)
				}

				// Return audio data
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("cartesia audio data"))
			},
			expectError: false,
		},
		{
			name: "unauthorized error",
			req: SpeechRequest{
				Text: "Test",
			},
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(cartesiaError{
					Error: "Invalid API key",
					Code:  "AUTH_ERROR",
				})
			},
			expectError: true,
			errorCode:   core.ErrorUnauthorized,
		},
		{
			name: "rate limited",
			req: SpeechRequest{
				Text: "Test",
			},
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(cartesiaError{
					Error: "Rate limit exceeded",
					Code:  "RATE_LIMIT",
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
			c := NewCartesia(
				WithCartesiaAPIKey("test-key"),
				WithCartesiaBaseURL(server.URL),
			)

			// Execute synthesis
			ctx := context.Background()
			stream, err := c.Synthesize(ctx, tt.req)

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

				if buf.String() != "cartesia audio data" {
					t.Errorf("unexpected audio data: %s", buf.String())
				}
			}
		})
	}
}

func TestCartesiaListVoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/voices" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Return voice list
		response := cartesiaVoicesResponse{
			Voices: []cartesiaVoice{
				{
					ID:          "narrator",
					Name:        "Professional Narrator",
					Description: "Clear and engaging narrator voice",
					Languages:   []string{"en", "es", "fr"},
					Gender:      "male",
					AgeGroup:    "adult",
					Tags:        []string{"professional", "clear", "engaging"},
				},
				{
					ID:          "conversational",
					Name:        "Conversational",
					Description: "Natural conversational voice",
					Languages:   []string{"en"},
					Gender:      "female",
					AgeGroup:    "young-adult",
					Tags:        []string{"friendly", "casual"},
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	c := NewCartesia(
		WithCartesiaAPIKey("test-key"),
		WithCartesiaBaseURL(server.URL),
	)

	ctx := context.Background()
	voices, err := c.ListVoices(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(voices) != 2 {
		t.Errorf("expected 2 voices, got %d", len(voices))
	}

	// Check first voice
	if voices[0].ID != "narrator" {
		t.Errorf("expected narrator, got %s", voices[0].ID)
	}
	if len(voices[0].Languages) != 3 {
		t.Errorf("expected 3 languages, got %d", len(voices[0].Languages))
	}

	// Check second voice
	if voices[1].Gender != "female" {
		t.Errorf("expected female, got %s", voices[1].Gender)
	}
	if voices[1].Age != "young-adult" {
		t.Errorf("expected young-adult, got %s", voices[1].Age)
	}
}

func TestCartesiaHelpers(t *testing.T) {
	c := NewCartesia()

	// Test speed
	if s := c.getSpeed(0); s != 1.0 {
		t.Errorf("expected default 1.0, got %f", s)
	}
	if s := c.getSpeed(0.3); s != 0.5 {
		t.Errorf("expected 0.5, got %f", s)
	}
	if s := c.getSpeed(3.0); s != 2.0 {
		t.Errorf("expected 2.0, got %f", s)
	}
	if s := c.getSpeed(1.5); s != 1.5 {
		t.Errorf("expected 1.5, got %f", s)
	}

	// Test emotion
	opts := map[string]any{"emotion": "happy"}
	if e := c.getEmotion(opts); e != "happy" {
		t.Errorf("expected happy, got %s", e)
	}
	if e := c.getEmotion(nil); e != "" {
		t.Errorf("expected empty, got %s", e)
	}

	// Test container
	if cont := c.getContainer(FormatMP3); cont != "mp3" {
		t.Errorf("expected mp3, got %s", cont)
	}
	if cont := c.getContainer(FormatWAV); cont != "wav" {
		t.Errorf("expected wav, got %s", cont)
	}
	if cont := c.getContainer("unknown"); cont != "mp3" {
		t.Errorf("expected default mp3, got %s", cont)
	}

	// Test encoding
	if enc := c.getEncoding(FormatPCM); enc != "pcm_s16le" {
		t.Errorf("expected pcm_s16le, got %s", enc)
	}
	if enc := c.getEncoding(FormatMuLaw); enc != "pcm_mulaw" {
		t.Errorf("expected pcm_mulaw, got %s", enc)
	}
	if enc := c.getEncoding(FormatMP3); enc != "" {
		t.Errorf("expected empty, got %s", enc)
	}

	// Test sample rate
	if sr := c.getSampleRate(FormatULaw); sr != 8000 {
		t.Errorf("expected 8000, got %d", sr)
	}
	if sr := c.getSampleRate(FormatPCM); sr != 44100 {
		t.Errorf("expected 44100, got %d", sr)
	}
	if sr := c.getSampleRate(FormatMP3); sr != 0 {
		t.Errorf("expected 0, got %d", sr)
	}

	// Test audio format
	format := c.getAudioFormat(FormatMP3)
	if format.MIME != MimeMP3 {
		t.Errorf("expected %s, got %s", MimeMP3, format.MIME)
	}

	format = c.getAudioFormat(FormatWAV)
	if format.MIME != MimeWAV {
		t.Errorf("expected %s, got %s", MimeWAV, format.MIME)
	}
	if format.SampleRate != 44100 {
		t.Errorf("expected 44100, got %d", format.SampleRate)
	}

	format = c.getAudioFormat(FormatFLAC)
	if format.BitDepth != 16 {
		t.Errorf("expected 16, got %d", format.BitDepth)
	}
}

func TestCartesiaErrorMapping(t *testing.T) {
	c := NewCartesia()

	tests := []struct {
		statusCode int
		body       []byte
		expectCode core.ErrorCode
		expectTemp bool
	}{
		{
			statusCode: http.StatusUnauthorized,
			body:       []byte(`{"error":"Invalid API key","code":"AUTH_ERROR"}`),
			expectCode: core.ErrorUnauthorized,
			expectTemp: false,
		},
		{
			statusCode: http.StatusTooManyRequests,
			body:       []byte(`{"error":"Rate limited","code":"RATE_LIMIT"}`),
			expectCode: core.ErrorRateLimited,
			expectTemp: true,
		},
		{
			statusCode: http.StatusServiceUnavailable,
			body:       []byte("service unavailable"),
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
		err := c.mapError(tt.statusCode, tt.body)
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

		if aiErr.Provider != "cartesia" {
			t.Errorf("expected provider cartesia, got %s", aiErr.Provider)
		}
	}
}