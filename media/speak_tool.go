package media

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/recera/gai/tools"
)

// SpeakInput defines the input for the Speak tool.
type SpeakInput struct {
	// Text to speak (required).
	Text string `json:"text" jsonschema:"required,description=The text to convert to speech"`

	// Voice to use (optional, uses provider default if not specified).
	Voice string `json:"voice,omitempty" jsonschema:"description=Voice ID or name to use for synthesis"`

	// Format for the output (optional, defaults to mp3).
	Format string `json:"format,omitempty" jsonschema:"enum=mp3,enum=wav,enum=ogg,description=Audio format for the output"`

	// Speed of speech (optional, 0.5 to 2.0, default 1.0).
	Speed float32 `json:"speed,omitempty" jsonschema:"minimum=0.5,maximum=2.0,description=Speaking speed (0.5 to 2.0)"`

	// Save to file (optional, if true saves to temp file and returns path).
	SaveToFile bool `json:"save_to_file,omitempty" jsonschema:"description=Save audio to a temporary file"`

	// Return as data URL (optional, if true returns base64 data URL).
	ReturnDataURL bool `json:"return_data_url,omitempty" jsonschema:"description=Return audio as a base64 data URL"`
}

// SpeakOutput defines the output from the Speak tool.
type SpeakOutput struct {
	// Success indicates whether the speech synthesis succeeded.
	Success bool `json:"success"`

	// FilePath is the path to the saved audio file (if SaveToFile was true).
	FilePath string `json:"file_path,omitempty"`

	// DataURL is the base64-encoded data URL (if ReturnDataURL was true).
	DataURL string `json:"data_url,omitempty"`

	// Format of the audio.
	Format string `json:"format"`

	// Duration in seconds (estimated).
	DurationSeconds float64 `json:"duration_seconds,omitempty"`

	// Size in bytes.
	SizeBytes int `json:"size_bytes"`

	// Error message if synthesis failed.
	Error string `json:"error,omitempty"`
}

// NewSpeakTool creates a tool that allows LLMs to trigger TTS.
func NewSpeakTool(provider SpeechProvider, opts ...SpeakToolOption) tools.Handle {
	cfg := speakToolConfig{
		tempDir:       os.TempDir(),
		maxTextLength: 5000,
		defaultFormat: FormatMP3,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return tools.New[SpeakInput, SpeakOutput](
		"speak",
		"Convert text to speech audio. Use this when you need to generate spoken audio from text.",
		func(ctx context.Context, input SpeakInput, meta tools.Meta) (SpeakOutput, error) {
			// Validate input
			if input.Text == "" {
				return SpeakOutput{
					Success: false,
					Error:   "text is required",
				}, nil
			}

			if len(input.Text) > cfg.maxTextLength {
				return SpeakOutput{
					Success: false,
					Error:   fmt.Sprintf("text too long (max %d characters)", cfg.maxTextLength),
				}, nil
			}

			// Set defaults
			format := input.Format
			if format == "" {
				format = cfg.defaultFormat
			}

			speed := input.Speed
			if speed == 0 {
				speed = 1.0
			}

			// Create speech request
			req := SpeechRequest{
				Text:   input.Text,
				Voice:  input.Voice,
				Format: format,
				Speed:  speed,
			}

			// Synthesize speech
			stream, err := provider.Synthesize(ctx, req)
			if err != nil {
				return SpeakOutput{
					Success: false,
					Error:   fmt.Sprintf("synthesis failed: %v", err),
				}, nil
			}
			defer stream.Close()

			// Collect audio chunks
			var audioBuffer bytes.Buffer
			for chunk := range stream.Chunks() {
				audioBuffer.Write(chunk)
			}

			// Check for stream errors
			if err := stream.Error(); err != nil {
				return SpeakOutput{
					Success: false,
					Error:   fmt.Sprintf("stream error: %v", err),
				}, nil
			}

			audioData := audioBuffer.Bytes()
			output := SpeakOutput{
				Success:   true,
				Format:    format,
				SizeBytes: len(audioData),
			}

			// Estimate duration (rough estimate based on typical speech rate)
			wordCount := float64(len(strings.Fields(input.Text)))
			wordsPerMinute := 150.0 * float64(speed) // Typical speech rate
			output.DurationSeconds = (wordCount / wordsPerMinute) * 60.0

			// Save to file if requested
			if input.SaveToFile {
				filename := fmt.Sprintf("speech_%s.%s", generateID(), format)
				filePath := filepath.Join(cfg.tempDir, filename)

				if err := os.WriteFile(filePath, audioData, 0644); err != nil {
					return SpeakOutput{
						Success: false,
						Error:   fmt.Sprintf("failed to save file: %v", err),
					}, nil
				}

				output.FilePath = filePath

				// Schedule cleanup if configured
				if cfg.cleanupAfter > 0 {
					go func() {
						select {
						case <-time.After(cfg.cleanupAfter):
							os.Remove(filePath)
						case <-ctx.Done():
							// Context cancelled, don't cleanup
						}
					}()
				}
			}

			// Return as data URL if requested
			if input.ReturnDataURL {
				mimeType := getMimeType(format)
				encoded := base64.StdEncoding.EncodeToString(audioData)
				output.DataURL = fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
			}

			// If neither file nor data URL requested, at least return the data URL
			if !input.SaveToFile && !input.ReturnDataURL {
				mimeType := getMimeType(format)
				encoded := base64.StdEncoding.EncodeToString(audioData)
				output.DataURL = fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
			}

			return output, nil
		},
	)
}

// SpeakToolOption configures the Speak tool.
type SpeakToolOption func(*speakToolConfig)

// WithSpeakToolTempDir sets the temporary directory for audio files.
func WithSpeakToolTempDir(dir string) SpeakToolOption {
	return func(cfg *speakToolConfig) {
		cfg.tempDir = dir
	}
}

// WithSpeakToolMaxTextLength sets the maximum text length.
func WithSpeakToolMaxTextLength(length int) SpeakToolOption {
	return func(cfg *speakToolConfig) {
		cfg.maxTextLength = length
	}
}

// WithSpeakToolDefaultFormat sets the default audio format.
func WithSpeakToolDefaultFormat(format string) SpeakToolOption {
	return func(cfg *speakToolConfig) {
		cfg.defaultFormat = format
	}
}

// WithSpeakToolCleanup sets automatic file cleanup duration.
func WithSpeakToolCleanup(duration time.Duration) SpeakToolOption {
	return func(cfg *speakToolConfig) {
		cfg.cleanupAfter = duration
	}
}

type speakToolConfig struct {
	tempDir       string
	maxTextLength int
	defaultFormat string
	cleanupAfter  time.Duration
}

// Helper functions

func getMimeType(format string) string {
	switch format {
	case FormatMP3:
		return MimeMP3
	case FormatWAV:
		return MimeWAV
	case FormatOGG:
		return MimeOGG
	case FormatOpus:
		return MimeOpus
	case FormatFLAC:
		return MimeFLAC
	case FormatWebM:
		return MimeWebM
	default:
		return "audio/mpeg"
	}
}

func generateID() string {
	// Simple ID generation using timestamp and random suffix
	now := time.Now().Unix()
	random := make([]byte, 4)
	if _, err := io.ReadFull(rand.Reader, random); err != nil {
		// Fallback to timestamp only
		return fmt.Sprintf("%d", now)
	}
	return fmt.Sprintf("%d_%x", now, random)
}