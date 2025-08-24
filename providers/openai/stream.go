package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/recera/gai/core"
)

// textStream implements core.TextStream for OpenAI streaming responses.
type textStream struct {
	events  chan core.Event
	cancel  context.CancelFunc
	resp    *http.Response
	done    chan struct{}
	err     error
	mu      sync.Mutex
	closed  bool
	
	// For accumulating tool calls across chunks
	toolCallAccumulator map[int]*toolCallBuilder
}

// toolCallBuilder accumulates tool call data across streaming chunks.
type toolCallBuilder struct {
	id       string
	name     string
	args     strings.Builder
	complete bool
}

// StreamText implements streaming text generation.
func (p *Provider) StreamText(ctx context.Context, req core.Request) (core.TextStream, error) {

	// Convert request
	apiReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
	}

	// Enable streaming
	apiReq.Stream = true
	apiReq.StreamOptions = &streamOptions{
		IncludeUsage: true,
	}

	// Create stream context
	streamCtx, cancel := context.WithCancel(ctx)

	// Make streaming request
	resp, err := p.doRequest(streamCtx, "POST", "/chat/completions", apiReq)
	if err != nil {
		cancel()
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		return nil, p.parseErrorFromBody(resp.StatusCode, body)
	}

	// Create stream
	stream := &textStream{
		events:              make(chan core.Event, 100),
		cancel:              cancel,
		resp:                resp,
		done:                make(chan struct{}),
		toolCallAccumulator: make(map[int]*toolCallBuilder),
	}

	// Start processing in background
	go stream.process(streamCtx, p.collector)

	return stream, nil
}

// Events returns the event channel.
func (s *textStream) Events() <-chan core.Event {
	return s.events
}

// Close terminates the stream.
func (s *textStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	// Cancel context
	s.cancel()

	// Close response body
	if s.resp != nil && s.resp.Body != nil {
		s.resp.Body.Close()
	}

	// Wait for processing to complete
	select {
	case <-s.done:
	case <-time.After(5 * time.Second):
		// Timeout waiting for processing
	}

	// Event channel is closed by process() method

	return s.err
}

// process handles the SSE stream processing.
func (s *textStream) process(ctx context.Context, collector core.MetricsCollector) {
	defer close(s.done)
	defer close(s.events) // Close events channel when done

	// Send start event
	s.sendEvent(core.Event{
		Type:      core.EventStart,
		Timestamp: time.Now(),
	})

	reader := bufio.NewReader(s.resp.Body)
	var totalUsage core.Usage

	for {
		select {
		case <-ctx.Done():
			s.err = ctx.Err()
			return
		default:
		}

		// Read line
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				s.sendEvent(core.Event{
					Type:      core.EventError,
					Err:       err,
					Timestamp: time.Now(),
				})
				s.err = err
			}
			break
		}

		// Trim whitespace
		line = bytes.TrimSpace(line)

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Parse SSE format
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}

		// Extract data
		data := bytes.TrimPrefix(line, []byte("data: "))

		// Check for end of stream
		if bytes.Equal(data, []byte("[DONE]")) {
			break
		}

		// Parse chunk
		var chunk streamChunk
		if err := json.Unmarshal(data, &chunk); err != nil {
			// Skip malformed chunks
			continue
		}

		// Process chunk
		s.processChunk(chunk, &totalUsage)
	}

	// Send finish event with usage
	s.sendEvent(core.Event{
		Type:      core.EventFinish,
		Usage:     &totalUsage,
		Timestamp: time.Now(),
	})

}

// processChunk processes a single streaming chunk.
func (s *textStream) processChunk(chunk streamChunk, totalUsage *core.Usage) {
	// Update usage if present
	if chunk.Usage != nil {
		totalUsage.InputTokens = chunk.Usage.PromptTokens
		totalUsage.OutputTokens = chunk.Usage.CompletionTokens
		totalUsage.TotalTokens = chunk.Usage.TotalTokens
	}

	// Process choices
	for _, choice := range chunk.Choices {
		// Handle text delta
		if choice.Delta.Content != nil && *choice.Delta.Content != "" {
			s.sendEvent(core.Event{
				Type:      core.EventTextDelta,
				TextDelta: *choice.Delta.Content,
				Timestamp: time.Now(),
			})
		}

		// Handle tool calls
		if len(choice.Delta.ToolCalls) > 0 {
			s.processToolCalls(choice.Delta.ToolCalls)
		}

		// Handle finish reason
		if choice.FinishReason != nil {
			// Could emit a step finish event if needed
		}
	}
}

// processToolCalls handles streaming tool call chunks.
func (s *textStream) processToolCalls(toolCalls []toolCall) {
	for _, tc := range toolCalls {
		// Note: In streaming, tool calls come in pieces
		// We need to accumulate them
		
		// For now, emit tool call events immediately
		// In production, you'd accumulate until complete
		s.sendEvent(core.Event{
			Type:      core.EventToolCall,
			ToolName:  tc.Function.Name,
			ToolID:    tc.ID,
			ToolInput: json.RawMessage(tc.Function.Arguments),
			Timestamp: time.Now(),
		})
	}
}

// sendEvent sends an event to the channel if not closed.
func (s *textStream) sendEvent(event core.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	select {
	case s.events <- event:
	default:
		// Channel full, drop event (or could block)
	}
}

// parseErrorFromBody parses an error from response body.
func (p *Provider) parseErrorFromBody(statusCode int, body []byte) error {
	var apiErr struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &apiErr); err != nil {
		return fmt.Errorf("HTTP %d: %s", statusCode, body)
	}

	// Create a mock response to use the error mapper
	resp := &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}
	return MapError(resp)
}

// objectStream implements core.ObjectStream for structured output streaming.
type objectStream struct {
	textStream
	schema      any
	accumulated strings.Builder
	finalValue  any
	finalErr    error
	finalOnce   sync.Once
}

// StreamObject implements streaming generation of structured objects.
func (p *Provider) StreamObject(ctx context.Context, req core.Request, schema any) (core.ObjectStream[any], error) {
	// Convert schema to JSON Schema format
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("marshaling schema: %w", err)
	}

	// Prepare request with response format
	apiReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
	}

	// Set response format for structured output
	apiReq.ResponseFormat = &responseFormat{
		Type: "json_schema",
		JSONSchema: &jsonSchemaFormat{
			Name:   "response",
			Schema: schemaBytes,
			Strict: true,
		},
	}

	// Enable streaming
	apiReq.Stream = true
	apiReq.StreamOptions = &streamOptions{
		IncludeUsage: true,
	}

	// Create stream context
	streamCtx, cancel := context.WithCancel(ctx)

	// Make streaming request
	resp, err := p.doRequest(streamCtx, "POST", "/chat/completions", apiReq)
	if err != nil {
		cancel()
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		return nil, p.parseErrorFromBody(resp.StatusCode, body)
	}

	// Create object stream
	stream := &objectStream{
		textStream: textStream{
			events:              make(chan core.Event, 100),
			cancel:              cancel,
			resp:                resp,
			done:                make(chan struct{}),
			toolCallAccumulator: make(map[int]*toolCallBuilder),
		},
		schema: schema,
	}

	// Start processing in background
	go stream.processObject(streamCtx, p.collector)

	return stream, nil
}

// Final returns the final validated object.
func (s *objectStream) Final() (*any, error) {
	// Wait for stream to complete
	<-s.done

	s.finalOnce.Do(func() {
		if s.finalErr != nil {
			return
		}

		// Parse accumulated JSON
		jsonStr := s.accumulated.String()
		if jsonStr == "" {
			s.finalErr = fmt.Errorf("no content received")
			return
		}

		var value any
		if err := json.Unmarshal([]byte(jsonStr), &value); err != nil {
			s.finalErr = fmt.Errorf("parsing JSON: %w", err)
			return
		}

		// TODO: Validate against schema if needed
		s.finalValue = value
	})

	if s.finalErr != nil {
		return nil, s.finalErr
	}

	return &s.finalValue, nil
}

// processObject handles object streaming processing.
func (s *objectStream) processObject(ctx context.Context, collector core.MetricsCollector) {
	defer close(s.done)
	defer close(s.events) // Close events channel when done

	// Send start event
	s.sendEvent(core.Event{
		Type:      core.EventStart,
		Timestamp: time.Now(),
	})

	reader := bufio.NewReader(s.resp.Body)
	var totalUsage core.Usage

	for {
		select {
		case <-ctx.Done():
			s.err = ctx.Err()
			s.finalErr = ctx.Err()
			return
		default:
		}

		// Read line
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				s.sendEvent(core.Event{
					Type:      core.EventError,
					Err:       err,
					Timestamp: time.Now(),
				})
				s.err = err
				s.finalErr = err
			}
			break
		}

		// Trim whitespace
		line = bytes.TrimSpace(line)

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Parse SSE format
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}

		// Extract data
		data := bytes.TrimPrefix(line, []byte("data: "))

		// Check for end of stream
		if bytes.Equal(data, []byte("[DONE]")) {
			break
		}

		// Parse chunk
		var chunk streamChunk
		if err := json.Unmarshal(data, &chunk); err != nil {
			continue
		}

		// Process chunk for object accumulation
		s.processObjectChunk(chunk, &totalUsage)
	}

	// Send finish event with usage
	s.sendEvent(core.Event{
		Type:      core.EventFinish,
		Usage:     &totalUsage,
		Timestamp: time.Now(),
	})

}

// processObjectChunk processes a chunk for object streaming.
func (s *objectStream) processObjectChunk(chunk streamChunk, totalUsage *core.Usage) {
	// Update usage if present
	if chunk.Usage != nil {
		totalUsage.InputTokens = chunk.Usage.PromptTokens
		totalUsage.OutputTokens = chunk.Usage.CompletionTokens
		totalUsage.TotalTokens = chunk.Usage.TotalTokens
	}

	// Process choices
	for _, choice := range chunk.Choices {
		// Accumulate content for JSON parsing
		if choice.Delta.Content != nil && *choice.Delta.Content != "" {
			s.accumulated.WriteString(*choice.Delta.Content)
			
			// Also emit as text delta for real-time display
			s.sendEvent(core.Event{
				Type:      core.EventTextDelta,
				TextDelta: *choice.Delta.Content,
				Timestamp: time.Now(),
			})
		}
	}
}