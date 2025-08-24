package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/recera/gai/core"
)

// Deepgram implements TranscriptionProvider for Deepgram API.
type Deepgram struct {
	config     ProviderConfig
	httpClient *http.Client
	dialer     *websocket.Dialer
}

// NewDeepgram creates a new Deepgram STT provider.
func NewDeepgram(opts ...DeepgramOption) *Deepgram {
	d := &Deepgram{
		config: ProviderConfig{
			BaseURL:       "https://api.deepgram.com",
			DefaultModel:  "nova-2",
			Timeout:       60 * time.Second,
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
		dialer: &websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(d)
	}

	if d.config.Timeout > 0 {
		d.httpClient.Timeout = d.config.Timeout
	}

	return d
}

// DeepgramOption configures the Deepgram provider.
type DeepgramOption func(*Deepgram)

// WithDeepgramAPIKey sets the API key.
func WithDeepgramAPIKey(key string) DeepgramOption {
	return func(d *Deepgram) {
		d.config.APIKey = key
	}
}

// WithDeepgramBaseURL sets a custom base URL.
func WithDeepgramBaseURL(url string) DeepgramOption {
	return func(d *Deepgram) {
		d.config.BaseURL = strings.TrimSuffix(url, "/")
	}
}

// WithDeepgramModel sets the default model.
func WithDeepgramModel(model string) DeepgramOption {
	return func(d *Deepgram) {
		d.config.DefaultModel = model
	}
}

// Transcribe converts audio to text using Deepgram.
func (d *Deepgram) Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResult, error) {
	// Get audio data
	audioData, err := d.getAudioData(ctx, req.Audio)
	if err != nil {
		return nil, fmt.Errorf("get audio data: %w", err)
	}

	// Build query parameters
	params := url.Values{}
	
	model := req.Model
	if model == "" {
		model = d.config.DefaultModel
	}
	params.Set("model", model)
	
	if req.Language != "" {
		params.Set("language", req.Language)
	} else {
		params.Set("detect_language", "true")
	}
	
	params.Set("punctuate", fmt.Sprintf("%t", req.Punctuate))
	params.Set("diarize", fmt.Sprintf("%t", req.Diarize))
	params.Set("profanity_filter", fmt.Sprintf("%t", req.FilterProfanity))
	
	if req.MaxAlternatives > 0 {
		params.Set("alternatives", fmt.Sprintf("%d", req.MaxAlternatives))
	}
	
	// Add keywords as search terms
	if len(req.Keywords) > 0 {
		params.Set("search", strings.Join(req.Keywords, ","))
	}
	
	// Request detailed output
	params.Set("utterances", "true")
	params.Set("word_timestamps", "true")

	// Create HTTP request
	url := fmt.Sprintf("%s/v1/listen?%s", d.config.BaseURL, params.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(audioData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", d.getContentType(req.Audio))
	httpReq.Header.Set("Authorization", fmt.Sprintf("Token %s", d.config.APIKey))
	
	// Add custom headers
	for k, v := range d.config.Headers {
		httpReq.Header.Set(k, v)
	}

	// Execute request
	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, d.mapError(resp.StatusCode, body)
	}

	// Parse response
	var dgResp deepgramResponse
	if err := json.NewDecoder(resp.Body).Decode(&dgResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to TranscriptionResult
	result := d.convertResponse(&dgResp)
	return result, nil
}

// TranscribeStream processes streaming audio input using WebSocket.
func (d *Deepgram) TranscribeStream(ctx context.Context, audio io.Reader) (TranscriptionStream, error) {
	// Build WebSocket URL
	wsURL := strings.Replace(d.config.BaseURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	
	params := url.Values{}
	params.Set("model", d.config.DefaultModel)
	params.Set("punctuate", "true")
	params.Set("interim_results", "true")
	params.Set("utterance_end_ms", "1000")
	params.Set("vad_events", "true")
	
	fullURL := fmt.Sprintf("%s/v1/listen?%s", wsURL, params.Encode())
	
	// Set headers
	headers := http.Header{}
	headers.Set("Authorization", fmt.Sprintf("Token %s", d.config.APIKey))
	
	// Connect WebSocket
	conn, _, err := d.dialer.DialContext(ctx, fullURL, headers)
	if err != nil {
		return nil, fmt.Errorf("connect websocket: %w", err)
	}
	
	// Create stream
	stream := &deepgramStream{
		conn:   conn,
		audio:  audio,
		events: make(chan TranscriptionEvent, 100),
		done:   make(chan struct{}),
	}
	
	// Start streaming goroutines
	go stream.sendAudio()
	go stream.receiveEvents()
	
	return stream, nil
}

// deepgramStream implements TranscriptionStream for Deepgram.
type deepgramStream struct {
	conn   *websocket.Conn
	audio  io.Reader
	events chan TranscriptionEvent
	done   chan struct{}
	err    error
}

func (s *deepgramStream) sendAudio() {
	defer s.conn.Close()
	
	buffer := make([]byte, 8192)
	for {
		select {
		case <-s.done:
			return
		default:
			n, err := s.audio.Read(buffer)
			if n > 0 {
				if err := s.conn.WriteMessage(websocket.BinaryMessage, buffer[:n]); err != nil {
					s.err = err
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					s.err = err
				}
				// Send close message
				s.conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"CloseStream"}`))
				return
			}
		}
	}
}

func (s *deepgramStream) receiveEvents() {
	defer close(s.events)
	
	for {
		select {
		case <-s.done:
			return
		default:
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					s.events <- TranscriptionEvent{
						Type:  TranscriptionError,
						Error: err,
					}
				}
				s.events <- TranscriptionEvent{Type: TranscriptionEnd}
				return
			}
			
			var result deepgramStreamResult
			if err := json.Unmarshal(message, &result); err != nil {
				continue // Skip malformed messages
			}
			
			// Convert to TranscriptionEvent
			if result.Type == "Results" && len(result.Channel.Alternatives) > 0 {
				alt := result.Channel.Alternatives[0]
				event := TranscriptionEvent{
					Text:    alt.Transcript,
					IsFinal: result.IsFinal,
				}
				
				if result.IsFinal {
					event.Type = TranscriptionFinal
				} else {
					event.Type = TranscriptionPartial
				}
				
				// Add word timings if available
				if len(alt.Words) > 0 {
					event.Words = make([]WordTiming, len(alt.Words))
					for i, w := range alt.Words {
						event.Words[i] = WordTiming{
							Word:       w.Word,
							Start:      time.Duration(w.Start * float64(time.Second)),
							End:        time.Duration(w.End * float64(time.Second)),
							Confidence: w.Confidence,
							Speaker:    w.Speaker,
						}
					}
				}
				
				s.events <- event
			}
		}
	}
}

func (s *deepgramStream) Events() <-chan TranscriptionEvent {
	return s.events
}

func (s *deepgramStream) Close() error {
	close(s.done)
	return s.conn.Close()
}

// Helper methods

func (d *Deepgram) getAudioData(ctx context.Context, blob core.BlobRef) ([]byte, error) {
	switch blob.Kind {
	case core.BlobBytes:
		return blob.Bytes, nil
	case core.BlobURL:
		// Download audio from URL
		req, err := http.NewRequestWithContext(ctx, "GET", blob.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("create download request: %w", err)
		}
		resp, err := d.httpClient.Do(req)
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

func (d *Deepgram) getContentType(blob core.BlobRef) string {
	if blob.MIME != "" {
		return blob.MIME
	}
	// Default to WAV
	return "audio/wav"
}

func (d *Deepgram) convertResponse(resp *deepgramResponse) *TranscriptionResult {
	result := &TranscriptionResult{
		Language: resp.Results.Channels[0].DetectedLanguage,
		Duration: time.Duration(resp.Metadata.Duration * float64(time.Second)),
	}
	
	// Get primary transcript
	if len(resp.Results.Channels) > 0 && len(resp.Results.Channels[0].Alternatives) > 0 {
		primary := resp.Results.Channels[0].Alternatives[0]
		result.Text = primary.Transcript
		result.Confidence = primary.Confidence
		
		// Add alternatives
		if len(resp.Results.Channels[0].Alternatives) > 1 {
			result.Alternatives = make([]TranscriptionAlternative, len(resp.Results.Channels[0].Alternatives)-1)
			for i, alt := range resp.Results.Channels[0].Alternatives[1:] {
				result.Alternatives[i] = TranscriptionAlternative{
					Text:       alt.Transcript,
					Confidence: alt.Confidence,
				}
			}
		}
		
		// Add word timings
		if len(primary.Words) > 0 {
			result.Words = make([]WordTiming, len(primary.Words))
			for i, w := range primary.Words {
				result.Words[i] = WordTiming{
					Word:       w.Word,
					Start:      time.Duration(w.Start * float64(time.Second)),
					End:        time.Duration(w.End * float64(time.Second)),
					Confidence: w.Confidence,
					Speaker:    w.Speaker,
				}
			}
		}
	}
	
	// Add speaker segments if diarization was enabled
	if len(resp.Results.Utterances) > 0 {
		result.Speakers = make([]SpeakerSegment, len(resp.Results.Utterances))
		for i, u := range resp.Results.Utterances {
			result.Speakers[i] = SpeakerSegment{
				Speaker: u.Speaker,
				Start:   time.Duration(u.Start * float64(time.Second)),
				End:     time.Duration(u.End * float64(time.Second)),
				Text:    u.Transcript,
			}
		}
	}
	
	return result
}

func (d *Deepgram) mapError(statusCode int, body []byte) error {
	var apiErr deepgramError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		return core.NewError(d.mapErrorCode(statusCode), apiErr.Message,
			core.WithProvider("deepgram"))
	}

	// Generic error mapping
	switch statusCode {
	case http.StatusUnauthorized:
		return core.NewError(core.ErrorUnauthorized, "invalid API key",
			core.WithProvider("deepgram"))
	case http.StatusForbidden:
		return core.NewError(core.ErrorForbidden, "access denied",
			core.WithProvider("deepgram"))
	case http.StatusNotFound:
		return core.NewError(core.ErrorNotFound, "endpoint not found",
			core.WithProvider("deepgram"))
	case http.StatusRequestEntityTooLarge:
		return core.NewError(core.ErrorInvalidRequest, "audio file too large",
			core.WithProvider("deepgram"))
	case http.StatusTooManyRequests:
		return core.NewError(core.ErrorRateLimited, "rate limited",
			core.WithProvider("deepgram"),
			core.WithTemporary(true))
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return core.NewError(core.ErrorProviderUnavailable, "service unavailable",
			core.WithProvider("deepgram"),
			core.WithTemporary(true))
	default:
		return core.NewError(core.ErrorInternal, fmt.Sprintf("HTTP %d: %s", statusCode, string(body)),
			core.WithProvider("deepgram"))
	}
}

func (d *Deepgram) mapErrorCode(statusCode int) core.ErrorCode {
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

type deepgramResponse struct {
	Metadata deepgramMetadata `json:"metadata"`
	Results  deepgramResults  `json:"results"`
}

type deepgramMetadata struct {
	TransactionKey string  `json:"transaction_key"`
	RequestID      string  `json:"request_id"`
	Sha256         string  `json:"sha256"`
	Created        string  `json:"created"`
	Duration       float64 `json:"duration"`
	Channels       int     `json:"channels"`
}

type deepgramResults struct {
	Channels   []deepgramChannel   `json:"channels"`
	Utterances []deepgramUtterance `json:"utterances"`
}

type deepgramChannel struct {
	Alternatives     []deepgramAlternative `json:"alternatives"`
	DetectedLanguage string                `json:"detected_language"`
}

type deepgramAlternative struct {
	Transcript string          `json:"transcript"`
	Confidence float32         `json:"confidence"`
	Words      []deepgramWord  `json:"words"`
}

type deepgramWord struct {
	Word       string  `json:"word"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Confidence float32 `json:"confidence"`
	Speaker    int     `json:"speaker"`
}

type deepgramUtterance struct {
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Confidence float32 `json:"confidence"`
	Channel    int     `json:"channel"`
	Transcript string  `json:"transcript"`
	Speaker    int     `json:"speaker"`
}

type deepgramStreamResult struct {
	Type      string `json:"type"`
	IsFinal   bool   `json:"is_final"`
	Channel   deepgramChannel `json:"channel"`
}

type deepgramError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}