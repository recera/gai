package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/recera/gai/core"
)

// ElevenLabs implements SpeechProvider for ElevenLabs TTS API.
type ElevenLabs struct {
	config     ProviderConfig
	httpClient *http.Client
}

// NewElevenLabs creates a new ElevenLabs TTS provider.
func NewElevenLabs(opts ...ElevenLabsOption) *ElevenLabs {
	el := &ElevenLabs{
		config: ProviderConfig{
			BaseURL:       "https://api.elevenlabs.io",
			DefaultVoice:  "EXAVITQu4vr4xnSDxMaL", // Sarah - default voice
			DefaultModel:  "eleven_multilingual_v2",
			DefaultFormat: FormatMP3,
			Timeout:       30 * time.Second,
			MaxRetries:    3,
			Headers:       make(map[string]string),
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}

	for _, opt := range opts {
		opt(el)
	}

	if el.config.Timeout > 0 {
		el.httpClient.Timeout = el.config.Timeout
	}

	return el
}

// ElevenLabsOption configures the ElevenLabs provider.
type ElevenLabsOption func(*ElevenLabs)

// WithElevenLabsAPIKey sets the API key.
func WithElevenLabsAPIKey(key string) ElevenLabsOption {
	return func(el *ElevenLabs) {
		el.config.APIKey = key
	}
}

// WithElevenLabsVoice sets the default voice.
func WithElevenLabsVoice(voice string) ElevenLabsOption {
	return func(el *ElevenLabs) {
		el.config.DefaultVoice = voice
	}
}

// WithElevenLabsModel sets the default model.
func WithElevenLabsModel(model string) ElevenLabsOption {
	return func(el *ElevenLabs) {
		el.config.DefaultModel = model
	}
}

// WithElevenLabsBaseURL sets a custom base URL.
func WithElevenLabsBaseURL(url string) ElevenLabsOption {
	return func(el *ElevenLabs) {
		el.config.BaseURL = strings.TrimSuffix(url, "/")
	}
}

// Synthesize converts text to speech using ElevenLabs.
func (el *ElevenLabs) Synthesize(ctx context.Context, req SpeechRequest) (SpeechStream, error) {
	// Use defaults if not specified
	voice := req.Voice
	if voice == "" {
		voice = el.config.DefaultVoice
	}

	model := req.Model
	if model == "" {
		model = el.config.DefaultModel
	}

	format := req.Format
	if format == "" {
		format = el.config.DefaultFormat
	}

	// Build request body
	body := elevenLabsRequest{
		Text:    req.Text,
		ModelID: model,
		VoiceSettings: elevenLabsVoiceSettings{
			Stability:       el.getStability(req.Stability),
			SimilarityBoost: el.getSimilarityBoost(req.SimilarityBoost),
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v1/text-to-speech/%s/stream", el.config.BaseURL, voice)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("xi-api-key", el.config.APIKey)
	httpReq.Header.Set("Accept", el.getAcceptHeader(format))

	// Add custom headers
	for k, v := range el.config.Headers {
		httpReq.Header.Set(k, v)
	}

	// Execute request
	resp, err := el.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, el.mapError(resp.StatusCode, body)
	}

	// Create streaming response
	stream := &elevenLabsStream{
		reader: resp.Body,
		chunks: make(chan []byte, 100),
		format: el.getAudioFormat(format),
		done:   make(chan struct{}),
	}

	// Start streaming goroutine
	go stream.stream()

	return stream, nil
}

// ListVoices returns available ElevenLabs voices.
func (el *ElevenLabs) ListVoices(ctx context.Context) ([]Voice, error) {
	url := fmt.Sprintf("%s/v1/voices", el.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("xi-api-key", el.config.APIKey)

	resp, err := el.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, el.mapError(resp.StatusCode, body)
	}

	var result elevenLabsVoicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	voices := make([]Voice, len(result.Voices))
	for i, v := range result.Voices {
		voices[i] = Voice{
			ID:          v.VoiceID,
			Name:        v.Name,
			Description: v.Description,
			Gender:      v.Labels.Gender,
			Age:         v.Labels.Age,
			Tags:        []string{v.Labels.UseCase, v.Labels.Accent},
			PreviewURL:  v.PreviewURL,
			Premium:     v.Category == "professional",
		}
	}

	return voices, nil
}

// elevenLabsStream implements SpeechStream for ElevenLabs.
type elevenLabsStream struct {
	reader io.ReadCloser
	chunks chan []byte
	format AudioFormat
	err    error
	done   chan struct{}
}

func (s *elevenLabsStream) stream() {
	defer close(s.chunks)
	defer close(s.done)
	defer s.reader.Close()

	// Stream audio chunks
	buffer := make([]byte, 4096)
	for {
		n, err := s.reader.Read(buffer)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			select {
			case s.chunks <- chunk:
			case <-s.done:
				return
			}
		}
		if err != nil {
			if err != io.EOF {
				s.err = err
			}
			return
		}
	}
}

func (s *elevenLabsStream) Chunks() <-chan []byte {
	return s.chunks
}

func (s *elevenLabsStream) Format() AudioFormat {
	return s.format
}

func (s *elevenLabsStream) Close() error {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	return s.reader.Close()
}

func (s *elevenLabsStream) Error() error {
	return s.err
}

// Helper methods

func (el *ElevenLabs) getStability(stability float32) float32 {
	if stability == 0 {
		return 0.5 // default
	}
	if stability < 0 {
		return 0
	}
	if stability > 1 {
		return 1
	}
	return stability
}

func (el *ElevenLabs) getSimilarityBoost(boost float32) float32 {
	if boost == 0 {
		return 0.75 // default
	}
	if boost < 0 {
		return 0
	}
	if boost > 1 {
		return 1
	}
	return boost
}

func (el *ElevenLabs) getAcceptHeader(format string) string {
	switch format {
	case FormatMP3:
		return "audio/mpeg"
	case FormatPCM:
		return "audio/pcm"
	case FormatULaw:
		return "audio/basic"
	default:
		return "audio/mpeg"
	}
}

func (el *ElevenLabs) getAudioFormat(format string) AudioFormat {
	switch format {
	case FormatMP3:
		return AudioFormat{
			MIME:     MimeMP3,
			Encoding: FormatMP3,
			Bitrate:  128000,
		}
	case FormatPCM:
		return AudioFormat{
			MIME:       "audio/pcm",
			Encoding:   FormatPCM,
			SampleRate: 44100,
			Channels:   1,
			BitDepth:   16,
		}
	case FormatULaw:
		return AudioFormat{
			MIME:       MimeBasic,
			Encoding:   FormatULaw,
			SampleRate: 8000,
			Channels:   1,
			BitDepth:   8,
		}
	default:
		return AudioFormat{
			MIME:     MimeMP3,
			Encoding: FormatMP3,
			Bitrate:  128000,
		}
	}
}

func (el *ElevenLabs) mapError(statusCode int, body []byte) error {
	var apiErr elevenLabsError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Detail.Message != "" {
		return core.NewError(el.mapErrorCode(statusCode), apiErr.Detail.Message,
			core.WithProvider("elevenlabs"))
	}

	// Generic error mapping
	switch statusCode {
	case http.StatusUnauthorized:
		return core.NewError(core.ErrorUnauthorized, "invalid API key",
			core.WithProvider("elevenlabs"))
	case http.StatusForbidden:
		return core.NewError(core.ErrorForbidden, "access denied",
			core.WithProvider("elevenlabs"))
	case http.StatusNotFound:
		return core.NewError(core.ErrorNotFound, "voice not found",
			core.WithProvider("elevenlabs"))
	case http.StatusTooManyRequests:
		return core.NewError(core.ErrorRateLimited, "rate limited",
			core.WithProvider("elevenlabs"),
			core.WithTemporary(true))
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return core.NewError(core.ErrorProviderUnavailable, "service unavailable",
			core.WithProvider("elevenlabs"),
			core.WithTemporary(true))
	default:
		return core.NewError(core.ErrorInternal, fmt.Sprintf("HTTP %d: %s", statusCode, string(body)),
			core.WithProvider("elevenlabs"))
	}
}

func (el *ElevenLabs) mapErrorCode(statusCode int) core.ErrorCode {
	switch statusCode {
	case http.StatusBadRequest:
		return core.ErrorInvalidRequest
	case http.StatusUnauthorized:
		return core.ErrorUnauthorized
	case http.StatusForbidden:
		return core.ErrorForbidden
	case http.StatusNotFound:
		return core.ErrorNotFound
	case http.StatusTooManyRequests:
		return core.ErrorRateLimited
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return core.ErrorProviderUnavailable
	default:
		return core.ErrorInternal
	}
}

// Request/response types

type elevenLabsRequest struct {
	Text          string                   `json:"text"`
	ModelID       string                   `json:"model_id"`
	VoiceSettings elevenLabsVoiceSettings `json:"voice_settings"`
}

type elevenLabsVoiceSettings struct {
	Stability       float32 `json:"stability"`
	SimilarityBoost float32 `json:"similarity_boost"`
}

type elevenLabsVoicesResponse struct {
	Voices []elevenLabsVoice `json:"voices"`
}

type elevenLabsVoice struct {
	VoiceID     string                `json:"voice_id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	PreviewURL  string                `json:"preview_url"`
	Category    string                `json:"category"`
	Labels      elevenLabsVoiceLabels `json:"labels"`
}

type elevenLabsVoiceLabels struct {
	Accent  string `json:"accent"`
	Age     string `json:"age"`
	Gender  string `json:"gender"`
	UseCase string `json:"use_case"`
}

type elevenLabsError struct {
	Detail struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"detail"`
}