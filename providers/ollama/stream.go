package ollama

import (
	"bufio"
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

// textStream implements core.TextStream for Ollama streaming responses.
type textStream struct {
	events  chan core.Event
	cancel  context.CancelFunc
	resp    *http.Response
	done    chan struct{}
	err     error
	mu      sync.Mutex
	closed  bool

	// For accumulating tool calls across chunks
	toolCallAccumulator map[string]*toolCallBuilder
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
	chatReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
	}

	// Enable streaming
	chatReq = chatReq.WithStream(true)

	// Create stream context
	streamCtx, cancel := context.WithCancel(ctx)

	// Make streaming request
	resp, err := p.doRequest(streamCtx, "POST", "/api/chat", chatReq)
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
		toolCallAccumulator: make(map[string]*toolCallBuilder),
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

// process handles the streaming response processing.
func (s *textStream) process(ctx context.Context, collector core.MetricsCollector) {
	defer close(s.done)
	defer close(s.events) // Close events channel when done

	// Send start event
	s.sendEvent(core.Event{
		Type:      core.EventStart,
		Timestamp: time.Now(),
	})

	scanner := bufio.NewScanner(s.resp.Body)
	var totalUsage core.Usage

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			s.err = ctx.Err()
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Parse streaming JSON response
		var chunk chatResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			// Skip malformed chunks
			continue
		}

		// Process chunk
		s.processChunk(chunk, &totalUsage)

		// Break if this is the final chunk
		if chunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		s.sendEvent(core.Event{
			Type:      core.EventError,
			Err:       err,
			Timestamp: time.Now(),
		})
		s.err = err
		return
	}

	// Send finish event with usage
	s.sendEvent(core.Event{
		Type:      core.EventFinish,
		Usage:     &totalUsage,
		Timestamp: time.Now(),
	})
}

// processChunk processes a single streaming chunk.
func (s *textStream) processChunk(chunk chatResponse, totalUsage *core.Usage) {
	// Update usage if this is the final chunk
	if chunk.Done {
		promptTokens, completionTokens, total := chunk.GetUsage()
		totalUsage.InputTokens = promptTokens
		totalUsage.OutputTokens = completionTokens
		totalUsage.TotalTokens = total
	}

	// Process message content
	if chunk.Message != nil {
		// Handle text delta
		if chunk.Message.Content != "" {
			s.sendEvent(core.Event{
				Type:      core.EventTextDelta,
				TextDelta: chunk.Message.Content,
				Timestamp: time.Now(),
			})
		}

		// Handle tool calls
		if len(chunk.Message.ToolCalls) > 0 {
			s.processToolCalls(chunk.Message.ToolCalls)
		}
	}
}

// processToolCalls handles streaming tool call chunks.
func (s *textStream) processToolCalls(toolCalls []toolCall) {
	for _, tc := range toolCalls {
		// Ollama may send complete tool calls or partial ones
		// For now, emit tool call events immediately
		// In a more sophisticated implementation, you might accumulate until complete
		
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
	var ollamaErr errorResponse
	if err := json.Unmarshal(body, &ollamaErr); err == nil && ollamaErr.Error != "" {
		return mapOllamaError(statusCode, ollamaErr.Error, body)
	}

	return mapStatusCodeError(statusCode, string(body))
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

	// Prepare request with format
	chatReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
	}

	// Set format for structured output and enable streaming
	chatReq = chatReq.WithFormat(string(schemaBytes)).WithStream(true)

	// Create stream context
	streamCtx, cancel := context.WithCancel(ctx)

	// Make streaming request
	resp, err := p.doRequest(streamCtx, "POST", "/api/chat", chatReq)
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
			toolCallAccumulator: make(map[string]*toolCallBuilder),
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

	scanner := bufio.NewScanner(s.resp.Body)
	var totalUsage core.Usage

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			s.err = ctx.Err()
			s.finalErr = ctx.Err()
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Parse streaming JSON response
		var chunk chatResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}

		// Process chunk for object accumulation
		s.processObjectChunk(chunk, &totalUsage)

		// Break if this is the final chunk
		if chunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		s.sendEvent(core.Event{
			Type:      core.EventError,
			Err:       err,
			Timestamp: time.Now(),
		})
		s.err = err
		s.finalErr = err
		return
	}

	// Send finish event with usage
	s.sendEvent(core.Event{
		Type:      core.EventFinish,
		Usage:     &totalUsage,
		Timestamp: time.Now(),
	})
}

// processObjectChunk processes a chunk for object streaming.
func (s *objectStream) processObjectChunk(chunk chatResponse, totalUsage *core.Usage) {
	// Update usage if this is the final chunk
	if chunk.Done {
		promptTokens, completionTokens, total := chunk.GetUsage()
		totalUsage.InputTokens = promptTokens
		totalUsage.OutputTokens = completionTokens
		totalUsage.TotalTokens = total
	}

	// Process message content
	if chunk.Message != nil && chunk.Message.Content != "" {
		// Accumulate content for JSON parsing
		s.accumulated.WriteString(chunk.Message.Content)

		// Also emit as text delta for real-time display
		s.sendEvent(core.Event{
			Type:      core.EventTextDelta,
			TextDelta: chunk.Message.Content,
			Timestamp: time.Now(),
		})
	}
}

// StreamTextUsingGenerateAPI streams text using the /api/generate endpoint.
// This is useful for models that work better with the generate API.
func (p *Provider) StreamTextUsingGenerateAPI(ctx context.Context, req core.Request) (core.TextStream, error) {
	// Build prompt from messages
	prompt := p.buildPromptFromMessages(req.Messages)
	
	genReq := NewGenerateRequest(p.getModel(req), prompt)
	
	// Set options
	if req.Temperature > 0 {
		if genReq.Options == nil {
			genReq.Options = &modelOptions{}
		}
		genReq.Options.Temperature = &req.Temperature
	}
	
	if req.MaxTokens > 0 {
		if genReq.Options == nil {
			genReq.Options = &modelOptions{}
		}
		genReq.Options.NumPredict = &req.MaxTokens
	}
	
	// Handle system message
	if len(req.Messages) > 0 && req.Messages[0].Role == core.System {
		if len(req.Messages[0].Parts) > 0 {
			if text, ok := req.Messages[0].Parts[0].(core.Text); ok {
				genReq.System = text.Text
			}
		}
	}
	
	// Apply provider options
	if opts, ok := req.ProviderOptions["ollama"].(map[string]interface{}); ok {
		p.applyGenerateOptions(genReq, opts)
	}
	
	// Enable streaming
	stream := true
	genReq.Stream = &stream

	// Create stream context
	streamCtx, cancel := context.WithCancel(ctx)

	// Make streaming request
	resp, err := p.doRequest(streamCtx, "POST", "/api/generate", genReq)
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
	textStream := &generateTextStream{
		events: make(chan core.Event, 100),
		cancel: cancel,
		resp:   resp,
		done:   make(chan struct{}),
	}

	// Start processing in background
	go textStream.processGenerate(streamCtx, p.collector)

	return textStream, nil
}

// generateTextStream implements core.TextStream for /api/generate responses.
type generateTextStream struct {
	events chan core.Event
	cancel context.CancelFunc
	resp   *http.Response
	done   chan struct{}
	err    error
	mu     sync.Mutex
	closed bool
}

// Events returns the event channel.
func (s *generateTextStream) Events() <-chan core.Event {
	return s.events
}

// Close terminates the stream.
func (s *generateTextStream) Close() error {
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

	return s.err
}

// processGenerate handles the generate API streaming response processing.
func (s *generateTextStream) processGenerate(ctx context.Context, collector core.MetricsCollector) {
	defer close(s.done)
	defer close(s.events) // Close events channel when done

	// Send start event
	s.sendEvent(core.Event{
		Type:      core.EventStart,
		Timestamp: time.Now(),
	})

	scanner := bufio.NewScanner(s.resp.Body)
	var totalUsage core.Usage

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			s.err = ctx.Err()
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Parse streaming JSON response
		var chunk generateResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			// Skip malformed chunks
			continue
		}

		// Process chunk
		if chunk.Response != "" {
			s.sendEvent(core.Event{
				Type:      core.EventTextDelta,
				TextDelta: chunk.Response,
				Timestamp: time.Now(),
			})
		}

		// Update usage on final chunk
		if chunk.Done {
			totalUsage.InputTokens = chunk.PromptEvalCount
			totalUsage.OutputTokens = chunk.EvalCount
			totalUsage.TotalTokens = totalUsage.InputTokens + totalUsage.OutputTokens
			break
		}
	}

	if err := scanner.Err(); err != nil {
		s.sendEvent(core.Event{
			Type:      core.EventError,
			Err:       err,
			Timestamp: time.Now(),
		})
		s.err = err
		return
	}

	// Send finish event with usage
	s.sendEvent(core.Event{
		Type:      core.EventFinish,
		Usage:     &totalUsage,
		Timestamp: time.Now(),
	})
}

// sendEvent sends an event to the channel if not closed.
func (s *generateTextStream) sendEvent(event core.Event) {
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