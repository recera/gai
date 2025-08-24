package anthropic

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

// textStream implements core.TextStream for Anthropic streaming responses.
type textStream struct {
	events  chan core.Event
	cancel  context.CancelFunc
	resp    *http.Response
	done    chan struct{}
	err     error
	mu      sync.Mutex
	closed  bool
	
	// For accumulating content across chunks
	contentBlocks       map[int]*contentBlockAccumulator
	currentMessage      *messagesResponse
	totalUsage          core.Usage
}

// contentBlockAccumulator accumulates content block data across streaming chunks.
type contentBlockAccumulator struct {
	block      *contentBlock
	textBuffer strings.Builder
	inputJSON  strings.Builder
	complete   bool
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

	// Create stream context
	streamCtx, cancel := context.WithCancel(ctx)

	// Make streaming request
	resp, err := p.doRequest(streamCtx, "POST", "/v1/messages", apiReq)
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
		events:        make(chan core.Event, 100),
		cancel:        cancel,
		resp:          resp,
		done:          make(chan struct{}),
		contentBlocks: make(map[int]*contentBlockAccumulator),
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

		// Skip ping events (empty data)
		if len(data) == 0 {
			continue
		}

		// Parse event
		var event streamEvent
		if err := json.Unmarshal(data, &event); err != nil {
			// Skip malformed events
			continue
		}

		// Process event
		s.processEvent(event)
	}

	// Send finish event with usage
	s.sendEvent(core.Event{
		Type:      core.EventFinish,
		Usage:     &s.totalUsage,
		Timestamp: time.Now(),
	})
}

// processEvent processes a single streaming event.
func (s *textStream) processEvent(event streamEvent) {
	switch event.Type {
	case "message_start":
		if event.messageStartEvent != nil && event.messageStartEvent.Message != nil {
			s.currentMessage = event.messageStartEvent.Message
		}

	case "content_block_start":
		if event.contentBlockStartEvent != nil {
			index := event.contentBlockStartEvent.Index
			s.contentBlocks[index] = &contentBlockAccumulator{
				block: event.contentBlockStartEvent.ContentBlock,
			}
		}

	case "content_block_delta":
		if event.contentBlockDeltaEvent != nil {
			s.processContentBlockDelta(event.contentBlockDeltaEvent)
		}

	case "content_block_stop":
		if event.contentBlockStopEvent != nil {
			index := event.contentBlockStopEvent.Index
			if acc, exists := s.contentBlocks[index]; exists {
				acc.complete = true
				s.processCompletedContentBlock(index, acc)
			}
		}

	case "message_delta":
		if event.messageDeltaEvent != nil {
			if event.messageDeltaEvent.Usage != nil {
				s.totalUsage.InputTokens = event.messageDeltaEvent.Usage.InputTokens
				s.totalUsage.OutputTokens = event.messageDeltaEvent.Usage.OutputTokens
				s.totalUsage.TotalTokens = s.totalUsage.InputTokens + s.totalUsage.OutputTokens
			}
		}

	case "message_stop":
		// Message is complete

	case "ping":
		// Keep-alive ping, no action needed

	case "error":
		if event.errorEvent != nil && event.errorEvent.Error != nil {
			s.sendEvent(core.Event{
				Type:      core.EventError,
				Err:       fmt.Errorf("API error: %s", event.errorEvent.Error.Message),
				Timestamp: time.Now(),
			})
		}
	}
}

// processContentBlockDelta handles incremental content block updates.
func (s *textStream) processContentBlockDelta(delta *contentBlockDeltaEvent) {
	index := delta.Index
	acc, exists := s.contentBlocks[index]
	if !exists {
		return
	}

	if delta.Delta != nil {
		switch delta.Delta.Type {
		case "text_delta":
			if delta.Delta.Text != "" {
				acc.textBuffer.WriteString(delta.Delta.Text)
				
				// Send text delta event
				s.sendEvent(core.Event{
					Type:      core.EventTextDelta,
					TextDelta: delta.Delta.Text,
					Timestamp: time.Now(),
				})
			}

		case "input_json_delta":
			if delta.Delta.PartialJSON != "" {
				acc.inputJSON.WriteString(delta.Delta.PartialJSON)
			}
		}
	}
}

// processCompletedContentBlock handles a completed content block.
func (s *textStream) processCompletedContentBlock(index int, acc *contentBlockAccumulator) {
	if acc.block == nil {
		return
	}

	switch acc.block.Type {
	case "text":
		// Text content block is complete
		// All text deltas have already been sent

	case "tool_use":
		// Tool use block is complete, emit tool call event
		var input map[string]interface{}
		jsonStr := acc.inputJSON.String()
		if jsonStr != "" {
			if err := json.Unmarshal([]byte(jsonStr), &input); err != nil {
				// If JSON parsing fails, use empty input
				input = make(map[string]interface{})
			}
		}

		inputJSON, _ := json.Marshal(input)
		
		s.sendEvent(core.Event{
			Type:      core.EventToolCall,
			ToolName:  acc.block.Name,
			ToolID:    acc.block.ID,
			ToolInput: json.RawMessage(inputJSON),
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
			Type    string `json:"type"`
			Message string `json:"message"`
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
	// For Anthropic, we need to modify the request to include JSON formatting instructions
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("marshaling schema: %w", err)
	}

	// Add JSON formatting instructions to the request
	modifiedReq := req
	
	// Add instruction to produce JSON output
	jsonInstructions := fmt.Sprintf(`Please respond with a valid JSON object that conforms to this schema:

%s

Respond with only the JSON object, no additional text.`, string(schemaJSON))

	// If there are existing messages, add the instruction to the last user message
	// or create a new user message with the instruction
	if len(modifiedReq.Messages) > 0 {
		// Find the last user message and append the instruction
		lastUserIndex := -1
		for i := len(modifiedReq.Messages) - 1; i >= 0; i-- {
			if modifiedReq.Messages[i].Role == core.User {
				lastUserIndex = i
				break
			}
		}
		
		if lastUserIndex >= 0 {
			// Append to the last user message
			lastMsg := modifiedReq.Messages[lastUserIndex]
			if len(lastMsg.Parts) > 0 {
				if text, ok := lastMsg.Parts[0].(core.Text); ok {
					modifiedReq.Messages[lastUserIndex].Parts[0] = core.Text{
						Text: text.Text + "\n\n" + jsonInstructions,
					}
				}
			}
		} else {
			// Add a new user message
			modifiedReq.Messages = append(modifiedReq.Messages, core.Message{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: jsonInstructions}},
			})
		}
	} else {
		// No messages yet, add the instruction as the first message
		modifiedReq.Messages = []core.Message{
			{
				Role:  core.User,
				Parts: []core.Part{core.Text{Text: jsonInstructions}},
			},
		}
	}

	// Convert request
	apiReq, err := p.convertRequest(modifiedReq)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
	}

	// Enable streaming
	apiReq.Stream = true

	// Create stream context
	streamCtx, cancel := context.WithCancel(ctx)

	// Make streaming request
	resp, err := p.doRequest(streamCtx, "POST", "/v1/messages", apiReq)
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
			events:        make(chan core.Event, 100),
			cancel:        cancel,
			resp:          resp,
			done:          make(chan struct{}),
			contentBlocks: make(map[int]*contentBlockAccumulator),
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

		// Skip ping events (empty data)
		if len(data) == 0 {
			continue
		}

		// Parse event
		var event streamEvent
		if err := json.Unmarshal(data, &event); err != nil {
			continue
		}

		// Process event for object accumulation
		s.processObjectEvent(event)
	}

	// Send finish event with usage
	s.sendEvent(core.Event{
		Type:      core.EventFinish,
		Usage:     &s.totalUsage,
		Timestamp: time.Now(),
	})
}

// processObjectEvent processes an event for object streaming.
func (s *objectStream) processObjectEvent(event streamEvent) {
	switch event.Type {
	case "message_start":
		if event.messageStartEvent != nil && event.messageStartEvent.Message != nil {
			s.currentMessage = event.messageStartEvent.Message
		}

	case "content_block_start":
		if event.contentBlockStartEvent != nil {
			index := event.contentBlockStartEvent.Index
			s.contentBlocks[index] = &contentBlockAccumulator{
				block: event.contentBlockStartEvent.ContentBlock,
			}
		}

	case "content_block_delta":
		if event.contentBlockDeltaEvent != nil {
			index := event.contentBlockDeltaEvent.Index
			if acc, exists := s.contentBlocks[index]; exists && event.contentBlockDeltaEvent.Delta != nil {
				if event.contentBlockDeltaEvent.Delta.Text != "" {
					// Accumulate text for JSON parsing
					s.accumulated.WriteString(event.contentBlockDeltaEvent.Delta.Text)
					acc.textBuffer.WriteString(event.contentBlockDeltaEvent.Delta.Text)
					
					// Also emit as text delta for real-time display
					s.sendEvent(core.Event{
						Type:      core.EventTextDelta,
						TextDelta: event.contentBlockDeltaEvent.Delta.Text,
						Timestamp: time.Now(),
					})
				}
			}
		}

	case "message_delta":
		if event.messageDeltaEvent != nil {
			if event.messageDeltaEvent.Usage != nil {
				s.totalUsage.InputTokens = event.messageDeltaEvent.Usage.InputTokens
				s.totalUsage.OutputTokens = event.messageDeltaEvent.Usage.OutputTokens
				s.totalUsage.TotalTokens = s.totalUsage.InputTokens + s.totalUsage.OutputTokens
			}
		}

	case "error":
		if event.errorEvent != nil && event.errorEvent.Error != nil {
			s.finalErr = fmt.Errorf("API error: %s", event.errorEvent.Error.Message)
			s.sendEvent(core.Event{
				Type:      core.EventError,
				Err:       s.finalErr,
				Timestamp: time.Now(),
			})
		}
	}
}