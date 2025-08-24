// Package gemini implements the Google Gemini provider for the GAI framework.
// It supports Gemini models with unique features including file uploads, safety
// configuration, citations, and multimodal content processing.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/recera/gai/core"
)

const (
	defaultBaseURL = "https://generativelanguage.googleapis.com"
	defaultTimeout = 60 * time.Second
	apiVersion     = "v1beta"
)

// Provider implements the core.Provider interface for Google Gemini.
type Provider struct {
	apiKey         string
	baseURL        string
	model          string
	client         *http.Client
	maxRetries     int
	retryDelay     time.Duration
	collector      core.MetricsCollector
	fileStore      *FileStore // For managing uploaded files
	defaultSafety  *core.SafetyConfig
	mu             sync.RWMutex
}

// Option configures the Gemini provider.
type Option func(*Provider)

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(p *Provider) {
		p.apiKey = key
	}
}

// WithBaseURL sets a custom base URL (useful for proxies or regional endpoints).
func WithBaseURL(url string) Option {
	return func(p *Provider) {
		p.baseURL = url
	}
}

// WithModel sets the default model to use.
func WithModel(model string) Option {
	return func(p *Provider) {
		p.model = model
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(p *Provider) {
		p.client = client
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(n int) Option {
	return func(p *Provider) {
		p.maxRetries = n
	}
}

// WithRetryDelay sets the base delay between retries.
func WithRetryDelay(d time.Duration) Option {
	return func(p *Provider) {
		p.retryDelay = d
	}
}

// WithMetricsCollector sets the metrics collector for observability.
func WithMetricsCollector(collector core.MetricsCollector) Option {
	return func(p *Provider) {
		p.collector = collector
	}
}

// WithDefaultSafety sets default safety settings for all requests.
func WithDefaultSafety(safety *core.SafetyConfig) Option {
	return func(p *Provider) {
		p.defaultSafety = safety
	}
}

// New creates a new Gemini provider with the given options.
func New(opts ...Option) *Provider {
	p := &Provider{
		baseURL:    defaultBaseURL,
		model:      "gemini-1.5-flash",
		maxRetries: 3,
		retryDelay: time.Second,
		fileStore:  NewFileStore(),
	}

	// Apply options
	for _, opt := range opts {
		opt(p)
	}

	// Create HTTP client if not provided
	if p.client == nil {
		p.client = &http.Client{
			Timeout: defaultTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
		}
	}

	return p
}

// GenerateText generates text with optional multi-step tool execution.
func (p *Provider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
	// Handle file uploads if needed
	req, err := p.processFiles(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to process files: %w", err)
	}

	// Run the request (potentially with tools)
	result, err := p.runRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// StreamText streams text generation with events.
func (p *Provider) StreamText(ctx context.Context, req core.Request) (core.TextStream, error) {
	// Handle file uploads if needed
	req, err := p.processFiles(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to process files: %w", err)
	}

	// Create the stream
	stream, err := p.createStream(ctx, req)
	if err != nil {
		return nil, err
	}

	return stream, nil
}

// GenerateObject generates a structured object conforming to the provided schema.
func (p *Provider) GenerateObject(ctx context.Context, req core.Request, schema any) (*core.ObjectResult[any], error) {
	// Add response schema to request
	reqWithSchema := req
	if reqWithSchema.ProviderOptions == nil {
		reqWithSchema.ProviderOptions = make(map[string]any)
	}
	reqWithSchema.ProviderOptions["response_schema"] = schema

	// Generate text with schema constraint
	result, err := p.GenerateText(ctx, reqWithSchema)
	if err != nil {
		return nil, err
	}

	// Parse the JSON response
	var obj any
	if err := json.Unmarshal([]byte(result.Text), &obj); err != nil {
		// Try to repair JSON if needed
		repaired := repairJSON(result.Text)
		if err := json.Unmarshal([]byte(repaired), &obj); err != nil {
			return nil, fmt.Errorf("failed to parse object: %w", err)
		}
	}

	return &core.ObjectResult[any]{
		Value: obj,
		Steps: result.Steps,
		Usage: result.Usage,
		Raw:   result.Raw,
	}, nil
}

// StreamObject streams a structured object generation.
func (p *Provider) StreamObject(ctx context.Context, req core.Request, schema any) (core.ObjectStream[any], error) {
	// Add response schema to request
	reqWithSchema := req
	if reqWithSchema.ProviderOptions == nil {
		reqWithSchema.ProviderOptions = make(map[string]any)
	}
	reqWithSchema.ProviderOptions["response_schema"] = schema
	reqWithSchema.ProviderOptions["stream_object"] = true

	// Create text stream
	textStream, err := p.StreamText(ctx, reqWithSchema)
	if err != nil {
		return nil, err
	}

	// Wrap as object stream
	return &objectStream[any]{
		TextStream: textStream,
		schema:     schema,
	}, nil
}

// processFiles handles file uploads for BlobRef entries.
func (prov *Provider) processFiles(ctx context.Context, req core.Request) (core.Request, error) {
	// Clone request to avoid mutation
	processed := req
	processed.Messages = make([]core.Message, len(req.Messages))
	copy(processed.Messages, req.Messages)

	for i, msg := range processed.Messages {
		newParts := make([]core.Part, 0, len(msg.Parts))
		for _, part := range msg.Parts {
			switch p := part.(type) {
			case core.Audio:
				if p.Source.Kind == core.BlobBytes || (p.Source.Kind == core.BlobURL && len(p.Source.URL) > 0) {
					// Upload audio file
					fileID, err := prov.uploadFile(ctx, p.Source, "audio")
					if err != nil {
						return req, fmt.Errorf("failed to upload audio: %w", err)
					}
					// Update source to use file ID
					p.Source.Kind = core.BlobProviderFile
					p.Source.FileID = fileID
				}
				newParts = append(newParts, p)
			case core.Video:
				if p.Source.Kind == core.BlobBytes || (p.Source.Kind == core.BlobURL && len(p.Source.URL) > 0) {
					// Upload video file
					fileID, err := prov.uploadFile(ctx, p.Source, "video")
					if err != nil {
						return req, fmt.Errorf("failed to upload video: %w", err)
					}
					// Update source to use file ID
					p.Source.Kind = core.BlobProviderFile
					p.Source.FileID = fileID
				}
				newParts = append(newParts, p)
			case core.File:
				if p.Source.Kind == core.BlobBytes || (p.Source.Kind == core.BlobURL && len(p.Source.URL) > 0) {
					// Upload file
					fileID, err := prov.uploadFile(ctx, p.Source, "document")
					if err != nil {
						return req, fmt.Errorf("failed to upload file: %w", err)
					}
					// Update source to use file ID
					p.Source.Kind = core.BlobProviderFile
					p.Source.FileID = fileID
				}
				newParts = append(newParts, p)
			default:
				newParts = append(newParts, part)
			}
		}
		processed.Messages[i].Parts = newParts
	}

	return processed, nil
}

// uploadFile uploads a file to Gemini's file API.
func (p *Provider) uploadFile(ctx context.Context, source core.BlobRef, purpose string) (string, error) {
	// Check if already uploaded
	if source.Kind == core.BlobProviderFile && source.FileID != "" {
		return source.FileID, nil
	}

	// Get file content
	var content []byte
	var mimeType string

	switch source.Kind {
	case core.BlobBytes:
		content = source.Bytes
		mimeType = source.MIME
	case core.BlobURL:
		// Download file from URL
		resp, err := p.client.Get(source.URL)
		if err != nil {
			return "", fmt.Errorf("failed to download file: %w", err)
		}
		defer resp.Body.Close()

		content, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		
		if mimeType == "" {
			mimeType = resp.Header.Get("Content-Type")
		}
	default:
		return "", fmt.Errorf("unsupported blob kind: %v", source.Kind)
	}

	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Create multipart upload
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file field
	part, err := writer.CreateFormFile("file", "upload")
	if err != nil {
		return "", err
	}
	if _, err := part.Write(content); err != nil {
		return "", err
	}

	// Add metadata
	if err := writer.WriteField("mime_type", mimeType); err != nil {
		return "", err
	}
	if err := writer.WriteField("purpose", purpose); err != nil {
		return "", err
	}

	if err := writer.Close(); err != nil {
		return "", err
	}

	// Upload to Gemini
	url := fmt.Sprintf("%s/upload/%s/files?uploadType=multipart", p.baseURL, apiVersion)
	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Goog-Api-Key", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, body)
	}

	// Parse response
	var uploadResp struct {
		File struct {
			Name string `json:"name"`
			URI  string `json:"uri"`
		} `json:"file"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", err
	}

	// Store file info
	fileID := uploadResp.File.Name
	p.fileStore.Store(fileID, &FileInfo{
		ID:        fileID,
		URI:       uploadResp.File.URI,
		MIMEType:  mimeType,
		Size:      int64(len(content)),
		ExpiresAt: time.Now().Add(48 * time.Hour), // Gemini files expire after 48 hours
	})

	return fileID, nil
}

// repairJSON attempts to fix common JSON formatting issues.
func repairJSON(text string) string {
	// Trim whitespace and markdown code blocks
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	
	// Ensure it starts and ends with object/array delimiters
	if !strings.HasPrefix(text, "{") && !strings.HasPrefix(text, "[") {
		// Try to find JSON object in the text
		start := strings.Index(text, "{")
		if start == -1 {
			start = strings.Index(text, "[")
		}
		if start != -1 {
			text = text[start:]
		}
	}
	
	return text
}


// objectStream wraps a TextStream to produce structured objects.
type objectStream[T any] struct {
	core.TextStream
	schema any
	mu     sync.Mutex
	final  *T
	err    error
	done   bool
}

func (s *objectStream[T]) Final() (*T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done {
		return s.final, s.err
	}

	// Accumulate all text
	var text strings.Builder
	for event := range s.Events() {
		if event.Type == core.EventTextDelta {
			text.WriteString(event.TextDelta)
		}
		if event.Type == core.EventError {
			s.err = event.Err
			s.done = true
			return nil, s.err
		}
	}

	// Parse JSON
	var obj T
	if err := json.Unmarshal([]byte(text.String()), &obj); err != nil {
		// Try to repair
		repaired := repairJSON(text.String())
		if err := json.Unmarshal([]byte(repaired), &obj); err != nil {
			s.err = fmt.Errorf("failed to parse object: %w", err)
			s.done = true
			return nil, s.err
		}
	}

	s.final = &obj
	s.done = true
	return s.final, nil
}