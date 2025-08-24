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

// Cartesia implements SpeechProvider for Cartesia TTS API.
type Cartesia struct {
	config     ProviderConfig
	httpClient *http.Client
}

// NewCartesia creates a new Cartesia TTS provider.
func NewCartesia(opts ...CartesiaOption) *Cartesia {
	c := &Cartesia{
		config: ProviderConfig{
			BaseURL:       "https://api.cartesia.ai",
			DefaultVoice:  "narrator-professional", // Professional narrator voice
			DefaultModel:  "sonic-english",
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
		opt(c)
	}

	if c.config.Timeout > 0 {
		c.httpClient.Timeout = c.config.Timeout
	}

	return c
}

// CartesiaOption configures the Cartesia provider.
type CartesiaOption func(*Cartesia)

// WithCartesiaAPIKey sets the API key.
func WithCartesiaAPIKey(key string) CartesiaOption {
	return func(c *Cartesia) {
		c.config.APIKey = key
	}
}

// WithCartesiaVoice sets the default voice.
func WithCartesiaVoice(voice string) CartesiaOption {
	return func(c *Cartesia) {
		c.config.DefaultVoice = voice
	}
}

// WithCartesiaModel sets the default model.
func WithCartesiaModel(model string) CartesiaOption {
	return func(c *Cartesia) {
		c.config.DefaultModel = model
	}
}

// WithCartesiaBaseURL sets a custom base URL.
func WithCartesiaBaseURL(url string) CartesiaOption {
	return func(c *Cartesia) {
		c.config.BaseURL = strings.TrimSuffix(url, "/")
	}
}

// Synthesize converts text to speech using Cartesia.
func (c *Cartesia) Synthesize(ctx context.Context, req SpeechRequest) (SpeechStream, error) {
	// Use defaults if not specified
	voice := req.Voice
	if voice == "" {
		voice = c.config.DefaultVoice
	}

	model := req.Model
	if model == "" {
		model = c.config.DefaultModel
	}

	format := req.Format
	if format == "" {
		format = c.config.DefaultFormat
	}

	// Build request body
	body := cartesiaRequest{
		Text:      req.Text,
		VoiceID:   voice,
		ModelID:   model,
		OutputFormat: cartesiaOutputFormat{
			Container:   c.getContainer(format),
			Encoding:    c.getEncoding(format),
			SampleRate:  c.getSampleRate(format),
		},
		Speed:     c.getSpeed(req.Speed),
		Emotion:   c.getEmotion(req.Options),
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/tts/bytes", c.config.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.config.APIKey)
	httpReq.Header.Set("Cartesia-Version", "2024-06-10")

	// Add custom headers
	for k, v := range c.config.Headers {
		httpReq.Header.Set(k, v)
	}

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, c.mapError(resp.StatusCode, body)
	}

	// Create streaming response
	stream := &cartesiaStream{
		reader: resp.Body,
		chunks: make(chan []byte, 100),
		format: c.getAudioFormat(format),
		done:   make(chan struct{}),
	}

	// Start streaming goroutine
	go stream.stream()

	return stream, nil
}

// ListVoices returns available Cartesia voices.
func (c *Cartesia) ListVoices(ctx context.Context) ([]Voice, error) {
	url := fmt.Sprintf("%s/voices", c.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.config.APIKey)
	req.Header.Set("Cartesia-Version", "2024-06-10")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, c.mapError(resp.StatusCode, body)
	}

	var result cartesiaVoicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	voices := make([]Voice, len(result.Voices))
	for i, v := range result.Voices {
		voices[i] = Voice{
			ID:          v.ID,
			Name:        v.Name,
			Description: v.Description,
			Languages:   v.Languages,
			Gender:      v.Gender,
			Age:         v.AgeGroup,
			Tags:        v.Tags,
		}
	}

	return voices, nil
}

// cartesiaStream implements SpeechStream for Cartesia.
type cartesiaStream struct {
	reader io.ReadCloser
	chunks chan []byte
	format AudioFormat
	err    error
	done   chan struct{}
}

func (s *cartesiaStream) stream() {
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

func (s *cartesiaStream) Chunks() <-chan []byte {
	return s.chunks
}

func (s *cartesiaStream) Format() AudioFormat {
	return s.format
}

func (s *cartesiaStream) Close() error {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	return s.reader.Close()
}

func (s *cartesiaStream) Error() error {
	return s.err
}

// Helper methods

func (c *Cartesia) getSpeed(speed float32) float32 {
	if speed == 0 {
		return 1.0 // default normal speed
	}
	if speed < 0.5 {
		return 0.5
	}
	if speed > 2.0 {
		return 2.0
	}
	return speed
}

func (c *Cartesia) getEmotion(options map[string]any) string {
	if options != nil {
		if emotion, ok := options["emotion"].(string); ok {
			return emotion
		}
	}
	return "" // neutral by default
}

func (c *Cartesia) getContainer(format string) string {
	switch format {
	case FormatMP3:
		return "mp3"
	case FormatWAV:
		return "wav"
	case FormatOGG:
		return "ogg"
	case FormatFLAC:
		return "flac"
	default:
		return "mp3"
	}
}

func (c *Cartesia) getEncoding(format string) string {
	switch format {
	case FormatPCM:
		return "pcm_s16le"
	case FormatMuLaw:
		return "pcm_mulaw"
	default:
		return "" // Use container default
	}
}

func (c *Cartesia) getSampleRate(format string) int {
	switch format {
	case FormatULaw, FormatMuLaw:
		return 8000
	case FormatPCM:
		return 44100
	default:
		return 0 // Use default
	}
}

func (c *Cartesia) getAudioFormat(format string) AudioFormat {
	switch format {
	case FormatMP3:
		return AudioFormat{
			MIME:     MimeMP3,
			Encoding: FormatMP3,
			Bitrate:  128000,
		}
	case FormatWAV:
		return AudioFormat{
			MIME:       MimeWAV,
			Encoding:   FormatPCM,
			SampleRate: 44100,
			Channels:   1,
			BitDepth:   16,
		}
	case FormatOGG:
		return AudioFormat{
			MIME:     MimeOGG,
			Encoding: FormatOGG,
			Bitrate:  128000,
		}
	case FormatFLAC:
		return AudioFormat{
			MIME:       MimeFLAC,
			Encoding:   FormatFLAC,
			SampleRate: 44100,
			Channels:   1,
			BitDepth:   16,
		}
	case FormatPCM:
		return AudioFormat{
			MIME:       "audio/pcm",
			Encoding:   FormatPCM,
			SampleRate: 44100,
			Channels:   1,
			BitDepth:   16,
		}
	default:
		return AudioFormat{
			MIME:     MimeMP3,
			Encoding: FormatMP3,
			Bitrate:  128000,
		}
	}
}

func (c *Cartesia) mapError(statusCode int, body []byte) error {
	var apiErr cartesiaError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error != "" {
		return core.NewError(c.mapErrorCode(statusCode), apiErr.Error,
			core.WithProvider("cartesia"))
	}

	// Generic error mapping
	switch statusCode {
	case http.StatusUnauthorized:
		return core.NewError(core.ErrorUnauthorized, "invalid API key",
			core.WithProvider("cartesia"))
	case http.StatusForbidden:
		return core.NewError(core.ErrorForbidden, "access denied",
			core.WithProvider("cartesia"))
	case http.StatusNotFound:
		return core.NewError(core.ErrorNotFound, "voice not found",
			core.WithProvider("cartesia"))
	case http.StatusTooManyRequests:
		return core.NewError(core.ErrorRateLimited, "rate limited",
			core.WithProvider("cartesia"),
			core.WithTemporary(true))
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return core.NewError(core.ErrorProviderUnavailable, "service unavailable",
			core.WithProvider("cartesia"),
			core.WithTemporary(true))
	default:
		return core.NewError(core.ErrorInternal, fmt.Sprintf("HTTP %d: %s", statusCode, string(body)),
			core.WithProvider("cartesia"))
	}
}

func (c *Cartesia) mapErrorCode(statusCode int) core.ErrorCode {
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

type cartesiaRequest struct {
	Text         string               `json:"text"`
	VoiceID      string               `json:"voice_id"`
	ModelID      string               `json:"model_id"`
	OutputFormat cartesiaOutputFormat `json:"output_format"`
	Speed        float32              `json:"speed,omitempty"`
	Emotion      string               `json:"emotion,omitempty"`
}

type cartesiaOutputFormat struct {
	Container  string `json:"container"`
	Encoding   string `json:"encoding,omitempty"`
	SampleRate int    `json:"sample_rate,omitempty"`
}

type cartesiaVoicesResponse struct {
	Voices []cartesiaVoice `json:"voices"`
}

type cartesiaVoice struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Languages   []string `json:"languages"`
	Gender      string   `json:"gender"`
	AgeGroup    string   `json:"age_group"`
	Tags        []string `json:"tags"`
}

type cartesiaError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}