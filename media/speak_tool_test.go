package media

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/recera/gai/tools"
)

// Mock speech provider for testing
type mockSpeechProvider struct {
	synthesizeFunc func(ctx context.Context, req SpeechRequest) (SpeechStream, error)
	listVoicesFunc func(ctx context.Context) ([]Voice, error)
}

func (m *mockSpeechProvider) Synthesize(ctx context.Context, req SpeechRequest) (SpeechStream, error) {
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(ctx, req)
	}
	// Default implementation
	return &mockSpeechStream{
		chunks: [][]byte{[]byte("mock audio data")},
		format: AudioFormat{
			MIME:     MimeMP3,
			Encoding: FormatMP3,
		},
	}, nil
}

func (m *mockSpeechProvider) ListVoices(ctx context.Context) ([]Voice, error) {
	if m.listVoicesFunc != nil {
		return m.listVoicesFunc(ctx)
	}
	return []Voice{}, nil
}

// Mock speech stream for testing
type mockSpeechStream struct {
	chunks   [][]byte
	format   AudioFormat
	err      error
	chunkIdx int
	done     chan struct{}
}

func (s *mockSpeechStream) Chunks() <-chan []byte {
	ch := make(chan []byte)
	go func() {
		defer close(ch)
		for _, chunk := range s.chunks {
			ch <- chunk
		}
	}()
	return ch
}

func (s *mockSpeechStream) Format() AudioFormat {
	return s.format
}

func (s *mockSpeechStream) Close() error {
	if s.done != nil {
		close(s.done)
	}
	return nil
}

func (s *mockSpeechStream) Error() error {
	return s.err
}

func TestSpeakTool(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		input          SpeakInput
		mockProvider   *mockSpeechProvider
		opts           []SpeakToolOption
		expectSuccess  bool
		checkOutput    func(t *testing.T, output SpeakOutput)
	}{
		{
			name: "successful synthesis with file save",
			input: SpeakInput{
				Text:       "Hello, world!",
				Voice:      "test-voice",
				Format:     FormatMP3,
				Speed:      1.0,
				SaveToFile: true,
			},
			mockProvider: &mockSpeechProvider{},
			opts: []SpeakToolOption{
				WithSpeakToolTempDir(tempDir),
			},
			expectSuccess: true,
			checkOutput: func(t *testing.T, output SpeakOutput) {
				if !output.Success {
					t.Error("expected success")
				}
				if output.FilePath == "" {
					t.Error("expected file path")
				}
				if !strings.HasPrefix(output.FilePath, tempDir) {
					t.Errorf("file not in temp dir: %s", output.FilePath)
				}
				if !strings.HasSuffix(output.FilePath, ".mp3") {
					t.Errorf("expected .mp3 extension: %s", output.FilePath)
				}
				// Check file exists
				if _, err := os.Stat(output.FilePath); err != nil {
					t.Errorf("file doesn't exist: %v", err)
				}
				// Clean up
				os.Remove(output.FilePath)
			},
		},
		{
			name: "successful synthesis with data URL",
			input: SpeakInput{
				Text:          "Test audio",
				Format:        FormatWAV,
				ReturnDataURL: true,
			},
			mockProvider: &mockSpeechProvider{
				synthesizeFunc: func(ctx context.Context, req SpeechRequest) (SpeechStream, error) {
					return &mockSpeechStream{
						chunks: [][]byte{[]byte("test wav data")},
						format: AudioFormat{
							MIME:     MimeWAV,
							Encoding: FormatWAV,
						},
					}, nil
				},
			},
			expectSuccess: true,
			checkOutput: func(t *testing.T, output SpeakOutput) {
				if !output.Success {
					t.Error("expected success")
				}
				if output.DataURL == "" {
					t.Error("expected data URL")
				}
				if !strings.HasPrefix(output.DataURL, "data:audio/wav;base64,") {
					t.Errorf("unexpected data URL format: %s", output.DataURL)
				}
				// Decode and verify data
				parts := strings.Split(output.DataURL, ",")
				if len(parts) != 2 {
					t.Error("invalid data URL format")
					return
				}
				decoded, err := base64.StdEncoding.DecodeString(parts[1])
				if err != nil {
					t.Errorf("failed to decode base64: %v", err)
				}
				if string(decoded) != "test wav data" {
					t.Errorf("unexpected decoded data: %s", string(decoded))
				}
			},
		},
		{
			name: "empty text error",
			input: SpeakInput{
				Text: "",
			},
			mockProvider:  &mockSpeechProvider{},
			expectSuccess: false,
			checkOutput: func(t *testing.T, output SpeakOutput) {
				if output.Success {
					t.Error("expected failure")
				}
				if !strings.Contains(output.Error, "text is required") {
					t.Errorf("unexpected error: %s", output.Error)
				}
			},
		},
		{
			name: "text too long error",
			input: SpeakInput{
				Text: strings.Repeat("a", 1000),
			},
			mockProvider: &mockSpeechProvider{},
			opts: []SpeakToolOption{
				WithSpeakToolMaxTextLength(100),
			},
			expectSuccess: false,
			checkOutput: func(t *testing.T, output SpeakOutput) {
				if output.Success {
					t.Error("expected failure")
				}
				if !strings.Contains(output.Error, "text too long") {
					t.Errorf("unexpected error: %s", output.Error)
				}
			},
		},
		{
			name: "default data URL when no output specified",
			input: SpeakInput{
				Text: "Default output",
			},
			mockProvider:  &mockSpeechProvider{},
			expectSuccess: true,
			checkOutput: func(t *testing.T, output SpeakOutput) {
				if !output.Success {
					t.Error("expected success")
				}
				if output.DataURL == "" {
					t.Error("expected data URL by default")
				}
				if output.FilePath != "" {
					t.Error("unexpected file path")
				}
			},
		},
		{
			name: "duration estimation",
			input: SpeakInput{
				Text:  "This is a test sentence with about ten words total.",
				Speed: 1.5,
			},
			mockProvider:  &mockSpeechProvider{},
			expectSuccess: true,
			checkOutput: func(t *testing.T, output SpeakOutput) {
				if !output.Success {
					t.Error("expected success")
				}
				// 10 words at 1.5x speed (225 wpm) = ~2.67 seconds
				if output.DurationSeconds < 2 || output.DurationSeconds > 4 {
					t.Errorf("unexpected duration estimate: %f", output.DurationSeconds)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create speak tool
			tool := NewSpeakTool(tt.mockProvider, tt.opts...)

			// Execute tool
			ctx := context.Background()
			meta := tools.Meta{
				CallID:    "test-call",
				RequestID: "test-request",
			}

			output, err := executeSpeak(tool, ctx, tt.input, meta)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check output
			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestSpeakToolWithCleanup(t *testing.T) {
	tempDir := t.TempDir()
	
	mockProvider := &mockSpeechProvider{}
	tool := NewSpeakTool(
		mockProvider,
		WithSpeakToolTempDir(tempDir),
		WithSpeakToolCleanup(100*time.Millisecond),
	)

	ctx := context.Background()
	meta := tools.Meta{}

	input := SpeakInput{
		Text:       "Cleanup test",
		SaveToFile: true,
	}

	output, err := executeSpeak(tool, ctx, input, meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !output.Success {
		t.Fatalf("expected success, got error: %s", output.Error)
	}

	filePath := output.FilePath
	if filePath == "" {
		t.Fatal("expected file path")
	}

	// File should exist initially
	if _, err := os.Stat(filePath); err != nil {
		t.Errorf("file doesn't exist initially: %v", err)
	}

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// File should be deleted after cleanup
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("file should have been cleaned up")
	}
}

func TestSpeakToolHelpers(t *testing.T) {
	// Test getMimeType
	tests := []struct {
		format   string
		expected string
	}{
		{FormatMP3, MimeMP3},
		{FormatWAV, MimeWAV},
		{FormatOGG, MimeOGG},
		{FormatOpus, MimeOpus},
		{FormatFLAC, MimeFLAC},
		{FormatWebM, MimeWebM},
		{"unknown", "audio/mpeg"},
	}

	for _, tt := range tests {
		result := getMimeType(tt.format)
		if result != tt.expected {
			t.Errorf("getMimeType(%s): expected %s, got %s", tt.format, tt.expected, result)
		}
	}

	// Test generateID
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("expected non-empty ID")
	}
	if id1 == id2 {
		t.Error("expected unique IDs")
	}
	if !strings.Contains(id1, "_") && len(id1) < 10 {
		t.Errorf("unexpected ID format: %s", id1)
	}
}

// Helper function to execute the speak tool
func executeSpeak(tool tools.Handle, ctx context.Context, input SpeakInput, meta tools.Meta) (SpeakOutput, error) {
	// The tool is already typed, but we need to call it through the Handle interface
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return SpeakOutput{}, err
	}

	result, err := tool.Exec(ctx, inputJSON, meta)
	if err != nil {
		return SpeakOutput{}, err
	}

	output, ok := result.(SpeakOutput)
	if !ok {
		return SpeakOutput{}, fmt.Errorf("unexpected output type: %T", result)
	}

	return output, nil
}