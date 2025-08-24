package media

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/tools"
)

// Benchmarks for ElevenLabs provider

func BenchmarkElevenLabsSynthesize(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio data"))
	}))
	defer server.Close()

	provider := NewElevenLabs(
		WithElevenLabsAPIKey("test-key"),
		WithElevenLabsBaseURL(server.URL),
	)

	ctx := context.Background()
	req := SpeechRequest{
		Text:   "Benchmark test",
		Voice:  "test-voice",
		Format: FormatMP3,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		stream, err := provider.Synthesize(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
		stream.Close()
	}
}

func BenchmarkElevenLabsStreamProcessing(b *testing.B) {
	// Benchmark streaming chunk processing
	audioData := bytes.Repeat([]byte("audio"), 1000) // 5KB of data

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := io.NopCloser(bytes.NewReader(audioData))
		stream := &elevenLabsStream{
			reader: reader,
			chunks: make(chan []byte, 100),
			format: AudioFormat{MIME: MimeMP3, Encoding: FormatMP3},
			done:   make(chan struct{}),
		}

		go stream.stream()

		// Consume chunks
		for range stream.Chunks() {
			// Just drain the channel
		}
	}
}

// Benchmarks for Cartesia provider

func BenchmarkCartesiaSynthesize(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio data"))
	}))
	defer server.Close()

	provider := NewCartesia(
		WithCartesiaAPIKey("test-key"),
		WithCartesiaBaseURL(server.URL),
	)

	ctx := context.Background()
	req := SpeechRequest{
		Text:   "Benchmark test",
		Voice:  "narrator",
		Format: FormatMP3,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		stream, err := provider.Synthesize(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
		stream.Close()
	}
}

// Benchmarks for Whisper provider

func BenchmarkWhisperTranscribe(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := whisperResponse{
			Text:     "Transcribed text",
			Language: "en",
			Duration: 5.0,
			Words: []whisperWord{
				{Word: "Transcribed", Start: 0.0, End: 0.5},
				{Word: "text", Start: 0.5, End: 1.0},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewWhisper(
		WithWhisperAPIKey("test-key"),
		WithWhisperBaseURL(server.URL),
	)

	ctx := context.Background()
	req := TranscriptionRequest{
		Audio: core.BlobRef{
			Kind:  core.BlobBytes,
			Bytes: []byte("fake audio"),
			MIME:  "audio/wav",
		},
		Language: "en",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := provider.Transcribe(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmarks for Deepgram provider

func BenchmarkDeepgramTranscribe(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := deepgramResponse{
			Metadata: deepgramMetadata{
				Duration: 5.0,
			},
			Results: deepgramResults{
				Channels: []deepgramChannel{
					{
						DetectedLanguage: "en",
						Alternatives: []deepgramAlternative{
							{
								Transcript: "Transcribed text",
								Confidence: 0.95,
								Words: []deepgramWord{
									{Word: "Transcribed", Start: 0.0, End: 0.5, Confidence: 0.98},
									{Word: "text", Start: 0.5, End: 1.0, Confidence: 0.92},
								},
							},
						},
					},
				},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewDeepgram(
		WithDeepgramAPIKey("test-key"),
		WithDeepgramBaseURL(server.URL),
	)

	ctx := context.Background()
	req := TranscriptionRequest{
		Audio: core.BlobRef{
			Kind:  core.BlobBytes,
			Bytes: []byte("fake audio"),
			MIME:  "audio/wav",
		},
		Language:  "en",
		Punctuate: true,
		Diarize:   true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := provider.Transcribe(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeepgramConvertResponse(b *testing.B) {
	provider := NewDeepgram()

	resp := &deepgramResponse{
		Metadata: deepgramMetadata{
			Duration: 10.5,
		},
		Results: deepgramResults{
			Channels: []deepgramChannel{
				{
					DetectedLanguage: "en",
					Alternatives: []deepgramAlternative{
						{
							Transcript: "This is a longer transcription with multiple words",
							Confidence: 0.95,
							Words: []deepgramWord{
								{Word: "This", Start: 0.0, End: 0.3, Confidence: 0.98},
								{Word: "is", Start: 0.3, End: 0.5, Confidence: 0.97},
								{Word: "a", Start: 0.5, End: 0.6, Confidence: 0.96},
								{Word: "longer", Start: 0.6, End: 1.0, Confidence: 0.95},
								{Word: "transcription", Start: 1.0, End: 1.5, Confidence: 0.94},
								{Word: "with", Start: 1.5, End: 1.7, Confidence: 0.93},
								{Word: "multiple", Start: 1.7, End: 2.2, Confidence: 0.92},
								{Word: "words", Start: 2.2, End: 2.5, Confidence: 0.91},
							},
						},
					},
				},
			},
			Utterances: []deepgramUtterance{
				{
					Start:      0.0,
					End:        2.5,
					Transcript: "This is a longer transcription with multiple words",
					Speaker:    0,
				},
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = provider.convertResponse(resp)
	}
}

// Benchmarks for Speak tool

func BenchmarkSpeakTool(b *testing.B) {
	mockProvider := &mockSpeechProvider{
		synthesizeFunc: func(ctx context.Context, req SpeechRequest) (SpeechStream, error) {
			return &mockSpeechStream{
				chunks: [][]byte{[]byte("audio data")},
				format: AudioFormat{MIME: MimeMP3, Encoding: FormatMP3},
			}, nil
		},
	}

	tool := NewSpeakTool(mockProvider)
	ctx := context.Background()
	meta := tools.Meta{CallID: "benchmark", RequestID: "bench-req"}

	input := SpeakInput{
		Text:          "Benchmark text",
		Format:        FormatMP3,
		ReturnDataURL: true,
	}

	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := tool.Exec(ctx, inputJSON, meta)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSpeakToolWithFile(b *testing.B) {
	tempDir := b.TempDir()

	mockProvider := &mockSpeechProvider{
		synthesizeFunc: func(ctx context.Context, req SpeechRequest) (SpeechStream, error) {
			return &mockSpeechStream{
				chunks: [][]byte{[]byte("audio data for file")},
				format: AudioFormat{MIME: MimeMP3, Encoding: FormatMP3},
			}, nil
		},
	}

	tool := NewSpeakTool(
		mockProvider,
		WithSpeakToolTempDir(tempDir),
	)

	ctx := context.Background()
	meta := tools.Meta{CallID: "benchmark", RequestID: "bench-req"}

	input := SpeakInput{
		Text:       "Benchmark text for file",
		Format:     FormatMP3,
		SaveToFile: true,
	}

	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := tool.Exec(ctx, inputJSON, meta)
		if err != nil {
			b.Fatal(err)
		}
		// Clean up file
		if output, ok := result.(SpeakOutput); ok && output.FilePath != "" {
			os.Remove(output.FilePath)
		}
	}
}

// Benchmarks for error mapping

func BenchmarkErrorMapping(b *testing.B) {
	providers := []struct {
		name string
		fn   func(int, []byte) error
	}{
		{
			name: "ElevenLabs",
			fn:   NewElevenLabs().mapError,
		},
		{
			name: "Cartesia",
			fn:   NewCartesia().mapError,
		},
		{
			name: "Whisper",
			fn:   NewWhisper().mapError,
		},
		{
			name: "Deepgram",
			fn:   NewDeepgram().mapError,
		},
	}

	testCases := []struct {
		statusCode int
		body       []byte
	}{
		{http.StatusUnauthorized, []byte(`{"error":"Invalid API key"}`)},
		{http.StatusTooManyRequests, []byte(`{"error":"Rate limited"}`)},
		{http.StatusInternalServerError, []byte(`{"error":"Internal error"}`)},
	}

	for _, provider := range providers {
		for _, tc := range testCases {
			b.Run(provider.name+"-"+http.StatusText(tc.statusCode), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_ = provider.fn(tc.statusCode, tc.body)
				}
			})
		}
	}
}

// Benchmark parallel processing

func BenchmarkParallelSynthesis(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio data"))
	}))
	defer server.Close()

	provider := NewElevenLabs(
		WithElevenLabsAPIKey("test-key"),
		WithElevenLabsBaseURL(server.URL),
	)

	ctx := context.Background()
	req := SpeechRequest{
		Text:   "Parallel benchmark test",
		Voice:  "test-voice",
		Format: FormatMP3,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			stream, err := provider.Synthesize(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
			// Consume stream
			for range stream.Chunks() {
			}
			stream.Close()
		}
	})
}

func BenchmarkParallelTranscription(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate processing
		time.Sleep(10 * time.Millisecond)
		response := whisperResponse{
			Text:     "Transcribed text",
			Language: "en",
			Duration: 5.0,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewWhisper(
		WithWhisperAPIKey("test-key"),
		WithWhisperBaseURL(server.URL),
	)

	ctx := context.Background()
	req := TranscriptionRequest{
		Audio: core.BlobRef{
			Kind:  core.BlobBytes,
			Bytes: []byte("fake audio"),
			MIME:  "audio/wav",
		},
		Language: "en",
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := provider.Transcribe(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}