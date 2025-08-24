package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/recera/gai/core"
)

// Whisper implements TranscriptionProvider for OpenAI Whisper API or compatible servers.
type Whisper struct {
	config     ProviderConfig
	httpClient *http.Client
}

// NewWhisper creates a new Whisper STT provider.
func NewWhisper(opts ...WhisperOption) *Whisper {
	w := &Whisper{
		config: ProviderConfig{
			BaseURL:       "https://api.openai.com",
			DefaultModel:  "whisper-1",
			Timeout:       60 * time.Second, // Transcription can take time
			MaxRetries:    3,
			Headers:       make(map[string]string),
		},
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}

	for _, opt := range opts {
		opt(w)
	}

	if w.config.Timeout > 0 {
		w.httpClient.Timeout = w.config.Timeout
	}

	return w
}

// WhisperOption configures the Whisper provider.
type WhisperOption func(*Whisper)

// WithWhisperAPIKey sets the API key.
func WithWhisperAPIKey(key string) WhisperOption {
	return func(w *Whisper) {
		w.config.APIKey = key
	}
}

// WithWhisperBaseURL sets a custom base URL (for self-hosted Whisper).
func WithWhisperBaseURL(url string) WhisperOption {
	return func(w *Whisper) {
		w.config.BaseURL = strings.TrimSuffix(url, "/")
	}
}

// WithWhisperModel sets the default model.
func WithWhisperModel(model string) WhisperOption {
	return func(w *Whisper) {
		w.config.DefaultModel = model
	}
}

// WithWhisperOrganization sets the OpenAI organization ID.
func WithWhisperOrganization(org string) WhisperOption {
	return func(w *Whisper) {
		w.config.Organization = org
	}
}

// Transcribe converts audio to text using Whisper.
func (w *Whisper) Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResult, error) {
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add audio file
	audioData, err := w.getAudioData(ctx, req.Audio)
	if err != nil {
		return nil, fmt.Errorf("get audio data: %w", err)
	}

	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(audioData)); err != nil {
		return nil, fmt.Errorf("write audio data: %w", err)
	}

	// Add model
	model := req.Model
	if model == "" {
		model = w.config.DefaultModel
	}
	if err := writer.WriteField("model", model); err != nil {
		return nil, fmt.Errorf("write model field: %w", err)
	}

	// Add optional parameters
	if req.Language != "" {
		if err := writer.WriteField("language", req.Language); err != nil {
			return nil, fmt.Errorf("write language field: %w", err)
		}
	}

	// Add prompt with keywords if provided
	if len(req.Keywords) > 0 {
		prompt := strings.Join(req.Keywords, ", ")
		if err := writer.WriteField("prompt", prompt); err != nil {
			return nil, fmt.Errorf("write prompt field: %w", err)
		}
	}

	// Request detailed response format
	if err := writer.WriteField("response_format", "verbose_json"); err != nil {
		return nil, fmt.Errorf("write response_format field: %w", err)
	}

	// Request word timestamps
	if err := writer.WriteField("timestamp_granularities[]", "word"); err != nil {
		return nil, fmt.Errorf("write timestamp field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v1/audio/transcriptions", w.config.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", w.config.APIKey))
	if w.config.Organization != "" {
		httpReq.Header.Set("OpenAI-Organization", w.config.Organization)
	}

	// Add custom headers
	for k, v := range w.config.Headers {
		httpReq.Header.Set(k, v)
	}

	// Execute request
	resp, err := w.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, w.mapError(resp.StatusCode, body)
	}

	// Parse response
	var whisperResp whisperResponse
	if err := json.NewDecoder(resp.Body).Decode(&whisperResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to TranscriptionResult
	result := &TranscriptionResult{
		Text:       whisperResp.Text,
		Language:   whisperResp.Language,
		Duration:   time.Duration(whisperResp.Duration * float64(time.Second)),
	}

	// Add word timings if available
	if len(whisperResp.Words) > 0 {
		result.Words = make([]WordTiming, len(whisperResp.Words))
		for i, w := range whisperResp.Words {
			result.Words[i] = WordTiming{
				Word:  w.Word,
				Start: time.Duration(w.Start * float64(time.Second)),
				End:   time.Duration(w.End * float64(time.Second)),
			}
		}
	}

	// Add segments as speaker segments (though Whisper doesn't do diarization)
	if len(whisperResp.Segments) > 0 {
		result.Speakers = make([]SpeakerSegment, len(whisperResp.Segments))
		for i, s := range whisperResp.Segments {
			result.Speakers[i] = SpeakerSegment{
				Speaker: 0, // Whisper doesn't identify speakers
				Start:   time.Duration(s.Start * float64(time.Second)),
				End:     time.Duration(s.End * float64(time.Second)),
				Text:    s.Text,
			}
		}
	}

	return result, nil
}

// TranscribeStream processes streaming audio input (not supported by standard Whisper).
func (w *Whisper) TranscribeStream(ctx context.Context, audio io.Reader) (TranscriptionStream, error) {
	// Standard Whisper API doesn't support streaming
	// This would need a custom implementation or WebSocket-based solution
	return nil, fmt.Errorf("streaming transcription not supported by standard Whisper API")
}

// Helper methods

func (w *Whisper) getAudioData(ctx context.Context, blob core.BlobRef) ([]byte, error) {
	switch blob.Kind {
	case core.BlobBytes:
		return blob.Bytes, nil
	case core.BlobURL:
		// Download audio from URL
		req, err := http.NewRequestWithContext(ctx, "GET", blob.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("create download request: %w", err)
		}
		resp, err := w.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("download audio: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	case core.BlobProviderFile:
		// This would need provider-specific handling
		return nil, fmt.Errorf("provider file references not supported")
	default:
		return nil, fmt.Errorf("unsupported blob kind: %v", blob.Kind)
	}
}

func (w *Whisper) mapError(statusCode int, body []byte) error {
	var apiErr whisperError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		return core.NewError(w.mapErrorCode(statusCode), apiErr.Error.Message,
			core.WithProvider("whisper"))
	}

	// Generic error mapping
	switch statusCode {
	case http.StatusUnauthorized:
		return core.NewError(core.ErrorUnauthorized, "invalid API key",
			core.WithProvider("whisper"))
	case http.StatusForbidden:
		return core.NewError(core.ErrorForbidden, "access denied",
			core.WithProvider("whisper"))
	case http.StatusNotFound:
		return core.NewError(core.ErrorNotFound, "endpoint not found",
			core.WithProvider("whisper"))
	case http.StatusRequestEntityTooLarge:
		return core.NewError(core.ErrorInvalidRequest, "audio file too large",
			core.WithProvider("whisper"))
	case http.StatusTooManyRequests:
		return core.NewError(core.ErrorRateLimited, "rate limited",
			core.WithProvider("whisper"),
			core.WithTemporary(true))
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return core.NewError(core.ErrorProviderUnavailable, "service unavailable",
			core.WithProvider("whisper"),
			core.WithTemporary(true))
	default:
		return core.NewError(core.ErrorInternal, fmt.Sprintf("HTTP %d: %s", statusCode, string(body)),
			core.WithProvider("whisper"))
	}
}

func (w *Whisper) mapErrorCode(statusCode int) core.ErrorCode {
	switch statusCode {
	case http.StatusBadRequest:
		return core.ErrorInvalidRequest
	case http.StatusUnauthorized:
		return core.ErrorUnauthorized
	case http.StatusForbidden:
		return core.ErrorForbidden
	case http.StatusNotFound:
		return core.ErrorNotFound
	case http.StatusRequestEntityTooLarge:
		return core.ErrorInvalidRequest
	case http.StatusTooManyRequests:
		return core.ErrorRateLimited
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return core.ErrorProviderUnavailable
	default:
		return core.ErrorInternal
	}
}

// Request/response types

type whisperResponse struct {
	Text     string            `json:"text"`
	Language string            `json:"language"`
	Duration float64           `json:"duration"`
	Words    []whisperWord     `json:"words"`
	Segments []whisperSegment  `json:"segments"`
}

type whisperWord struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type whisperSegment struct {
	ID               int     `json:"id"`
	Seek             int     `json:"seek"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	Temperature      float64 `json:"temperature"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
}

type whisperError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}