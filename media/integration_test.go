// +build integration

package media

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

// Integration tests that can run against real APIs when credentials are provided
// Run with: go test -tags=integration -v ./media

func TestElevenLabsIntegration(t *testing.T) {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	if apiKey == "" {
		t.Skip("ELEVENLABS_API_KEY not set, skipping integration test")
	}

	provider := NewElevenLabs(
		WithElevenLabsAPIKey(apiKey),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("ListVoices", func(t *testing.T) {
		voices, err := provider.ListVoices(ctx)
		if err != nil {
			t.Fatalf("failed to list voices: %v", err)
		}

		if len(voices) == 0 {
			t.Error("expected at least one voice")
		}

		for _, voice := range voices {
			t.Logf("Voice: %s - %s", voice.Name, voice.Description)
		}
	})

	t.Run("Synthesize", func(t *testing.T) {
		req := SpeechRequest{
			Text:   "Hello from the GAI framework integration test.",
			Format: FormatMP3,
		}

		stream, err := provider.Synthesize(ctx, req)
		if err != nil {
			t.Fatalf("failed to synthesize: %v", err)
		}
		defer stream.Close()

		var totalBytes int
		for chunk := range stream.Chunks() {
			totalBytes += len(chunk)
		}

		if totalBytes == 0 {
			t.Error("expected audio data")
		}

		t.Logf("Received %d bytes of audio data", totalBytes)

		if err := stream.Error(); err != nil {
			t.Errorf("stream error: %v", err)
		}
	})
}

func TestCartesiaIntegration(t *testing.T) {
	apiKey := os.Getenv("CARTESIA_API_KEY")
	if apiKey == "" {
		t.Skip("CARTESIA_API_KEY not set, skipping integration test")
	}

	provider := NewCartesia(
		WithCartesiaAPIKey(apiKey),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("ListVoices", func(t *testing.T) {
		voices, err := provider.ListVoices(ctx)
		if err != nil {
			t.Fatalf("failed to list voices: %v", err)
		}

		if len(voices) == 0 {
			t.Error("expected at least one voice")
		}

		for _, voice := range voices {
			t.Logf("Voice: %s - %s (Languages: %v)", voice.Name, voice.Description, voice.Languages)
		}
	})

	t.Run("Synthesize", func(t *testing.T) {
		req := SpeechRequest{
			Text:   "Testing Cartesia text to speech integration.",
			Format: FormatMP3,
			Speed:  1.0,
		}

		stream, err := provider.Synthesize(ctx, req)
		if err != nil {
			t.Fatalf("failed to synthesize: %v", err)
		}
		defer stream.Close()

		var totalBytes int
		for chunk := range stream.Chunks() {
			totalBytes += len(chunk)
		}

		if totalBytes == 0 {
			t.Error("expected audio data")
		}

		t.Logf("Received %d bytes of audio data", totalBytes)
	})
}

func TestWhisperIntegration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping Whisper integration test")
	}

	provider := NewWhisper(
		WithWhisperAPIKey(apiKey),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("Transcribe", func(t *testing.T) {
		// This would need a real audio file or URL
		audioURL := os.Getenv("TEST_AUDIO_URL")
		if audioURL == "" {
			t.Skip("TEST_AUDIO_URL not set, skipping transcription test")
		}

		req := TranscriptionRequest{
			Audio: core.BlobRef{
				Kind: core.BlobURL,
				URL:  audioURL,
				MIME: "audio/wav",
			},
			Language:  "en",
			Punctuate: true,
		}

		result, err := provider.Transcribe(ctx, req)
		if err != nil {
			t.Fatalf("failed to transcribe: %v", err)
		}

		if result.Text == "" {
			t.Error("expected transcription text")
		}

		t.Logf("Transcription: %s", result.Text)
		t.Logf("Language: %s, Duration: %v", result.Language, result.Duration)

		if len(result.Words) > 0 {
			t.Logf("Word count: %d", len(result.Words))
		}
	})
}

func TestDeepgramIntegration(t *testing.T) {
	apiKey := os.Getenv("DEEPGRAM_API_KEY")
	if apiKey == "" {
		t.Skip("DEEPGRAM_API_KEY not set, skipping integration test")
	}

	provider := NewDeepgram(
		WithDeepgramAPIKey(apiKey),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("Transcribe", func(t *testing.T) {
		// This would need a real audio file or URL
		audioURL := os.Getenv("TEST_AUDIO_URL")
		if audioURL == "" {
			t.Skip("TEST_AUDIO_URL not set, skipping transcription test")
		}

		req := TranscriptionRequest{
			Audio: core.BlobRef{
				Kind: core.BlobURL,
				URL:  audioURL,
				MIME: "audio/wav",
			},
			Language:        "en",
			Punctuate:       true,
			Diarize:         true,
			MaxAlternatives: 2,
		}

		result, err := provider.Transcribe(ctx, req)
		if err != nil {
			t.Fatalf("failed to transcribe: %v", err)
		}

		if result.Text == "" {
			t.Error("expected transcription text")
		}

		t.Logf("Transcription: %s", result.Text)
		t.Logf("Language: %s, Duration: %v, Confidence: %f", 
			result.Language, result.Duration, result.Confidence)

		if len(result.Words) > 0 {
			t.Logf("Word count: %d", len(result.Words))
		}

		if len(result.Speakers) > 0 {
			t.Logf("Speaker segments: %d", len(result.Speakers))
			for i, speaker := range result.Speakers {
				t.Logf("  Speaker %d: %v-%v: %s", 
					speaker.Speaker, speaker.Start, speaker.End, speaker.Text)
				if i >= 2 {
					break // Just show first few
				}
			}
		}

		if len(result.Alternatives) > 0 {
			t.Logf("Alternatives: %d", len(result.Alternatives))
			for i, alt := range result.Alternatives {
				t.Logf("  Alternative %d (confidence %f): %s", 
					i+1, alt.Confidence, alt.Text)
			}
		}
	})
}

func TestEndToEndTTSSTT(t *testing.T) {
	// This test synthesizes speech and then transcribes it
	elevenLabsKey := os.Getenv("ELEVENLABS_API_KEY")
	whisperKey := os.Getenv("OPENAI_API_KEY")

	if elevenLabsKey == "" || whisperKey == "" {
		t.Skip("Both ELEVENLABS_API_KEY and OPENAI_API_KEY required for end-to-end test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Synthesize speech
	tts := NewElevenLabs(WithElevenLabsAPIKey(elevenLabsKey))
	
	testText := "The quick brown fox jumps over the lazy dog."
	
	speechReq := SpeechRequest{
		Text:   testText,
		Format: FormatMP3,
	}

	stream, err := tts.Synthesize(ctx, speechReq)
	if err != nil {
		t.Fatalf("failed to synthesize: %v", err)
	}
	defer stream.Close()

	// Collect audio
	var audioData []byte
	for chunk := range stream.Chunks() {
		audioData = append(audioData, chunk...)
	}

	if len(audioData) == 0 {
		t.Fatal("no audio data generated")
	}

	t.Logf("Generated %d bytes of audio", len(audioData))

	// Transcribe the audio
	stt := NewWhisper(WithWhisperAPIKey(whisperKey))

	transcribeReq := TranscriptionRequest{
		Audio: core.BlobRef{
			Kind:  core.BlobBytes,
			Bytes: audioData,
			MIME:  "audio/mpeg",
		},
		Language: "en",
	}

	result, err := stt.Transcribe(ctx, transcribeReq)
	if err != nil {
		t.Fatalf("failed to transcribe: %v", err)
	}

	t.Logf("Original text: %s", testText)
	t.Logf("Transcribed text: %s", result.Text)

	// Check if transcription is reasonably close
	// (May not be exact due to TTS/STT variations)
	if result.Text == "" {
		t.Error("transcription is empty")
	}
}