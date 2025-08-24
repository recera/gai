package openai_compat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/recera/gai/core"
)

// textStream implements core.TextStream for OpenAI-compatible streaming.
type textStream struct {
	ctx    context.Context
	cancel context.CancelFunc
	events chan core.Event
	resp   *http.Response
	
	// State for accumulating tool calls
	toolCallAccumulator map[int]toolCallBuilder
}

// toolCallBuilder accumulates tool call fragments.
type toolCallBuilder struct {
	id       string
	name     string
	args     strings.Builder
}

// StreamText implements streaming text generation.
func (p *Provider) StreamText(ctx context.Context, req core.Request) (core.TextStream, error) {
	apiReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
	}
	
	// Enable streaming unless disabled
	if !p.config.DisableJSONStreaming {
		apiReq.Stream = true
		apiReq.StreamOptions = &streamOptions{
			IncludeUsage: true,
		}
	} else {
		// Fall back to non-streaming
		return p.simulateStream(ctx, req)
	}
	
	// Strip unsupported parameters
	apiReq = p.stripUnsupportedParams(apiReq)
	
	// Create stream context
	streamCtx, cancel := context.WithCancel(ctx)
	
	// Make streaming request
	resp, err := p.doRequest(streamCtx, "POST", "/chat/completions", apiReq)
	if err != nil {
		cancel()
		return nil, err
	}
	
	if resp.StatusCode != http.StatusOK {
		cancel()
		resp.Body.Close()
		return nil, MapError(resp, p.config.ProviderName)
	}
	
	// Create stream
	stream := &textStream{
		ctx:                 streamCtx,
		cancel:              cancel,
		events:              make(chan core.Event, 100),
		resp:                resp,
		toolCallAccumulator: make(map[int]toolCallBuilder),
	}
	
	// Start processing SSE stream
	go stream.process()
	
	return stream, nil
}

// simulateStream creates a stream from a non-streaming response.
func (p *Provider) simulateStream(ctx context.Context, req core.Request) (core.TextStream, error) {
	// Make non-streaming request
	result, err := p.GenerateText(ctx, req)
	if err != nil {
		return nil, err
	}
	
	// Create simulated stream
	streamCtx, cancel := context.WithCancel(ctx)
	events := make(chan core.Event, 10)
	
	go func() {
		defer close(events)
		
		// Send start event
		select {
		case events <- core.Event{Type: core.EventStart}:
		case <-streamCtx.Done():
			return
		}
		
		// Send text as a single delta
		if result.Text != "" {
			select {
			case events <- core.Event{
				Type:      core.EventTextDelta,
				TextDelta: result.Text,
			}:
			case <-streamCtx.Done():
				return
			}
		}
		
		// Send tool calls if any
		for _, step := range result.Steps {
			for _, tc := range step.ToolCalls {
				select {
				case events <- core.Event{
					Type:      core.EventToolCall,
					ToolName:  tc.Name,
					ToolInput: tc.Input,
				}:
				case <-streamCtx.Done():
					return
				}
			}
		}
		
		// Send finish event with usage
		select {
		case events <- core.Event{
			Type: core.EventFinish,
			Usage: &result.Usage,
		}:
		case <-streamCtx.Done():
			return
		}
	}()
	
	return &textStream{
		ctx:    streamCtx,
		cancel: cancel,
		events: events,
	}, nil
}

// process handles the SSE stream processing.
func (s *textStream) process() {
	defer close(s.events)
	defer s.resp.Body.Close()
	
	// Send start event
	s.sendEvent(core.Event{Type: core.EventStart})
	
	reader := bufio.NewReader(s.resp.Body)
	var usage *core.Usage
	
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				s.sendEvent(core.Event{
					Type: core.EventError,
					Err:  fmt.Errorf("reading stream: %w", err),
				})
			}
			break
		}
		
		// Trim whitespace
		line = bytes.TrimSpace(line)
		
		// Skip empty lines
		if len(line) == 0 {
			continue
		}
		
		// Check for SSE data prefix
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
			// Some providers might send malformed JSON, skip
			continue
		}
		
		// Process chunk
		for _, choice := range chunk.Choices {
			if choice.Delta != nil {
				// Handle text content
				switch content := choice.Delta.Content.(type) {
				case string:
					if content != "" {
						s.sendEvent(core.Event{
							Type:      core.EventTextDelta,
							TextDelta: content,
						})
					}
				}
				
				// Handle tool calls
				for idx, tc := range choice.Delta.ToolCalls {
					s.accumulateToolCall(idx, tc)
				}
			}
			
			// Handle finish reason
			if choice.FinishReason != nil && *choice.FinishReason != "" {
				// Emit accumulated tool calls
				s.emitToolCalls()
			}
		}
		
		// Capture usage if present
		if chunk.Usage != nil {
			usage = &core.Usage{
				InputTokens:  chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
				TotalTokens:  chunk.Usage.TotalTokens,
			}
		}
	}
	
	// Emit any remaining tool calls
	s.emitToolCalls()
	
	// Send finish event
	s.sendEvent(core.Event{
		Type:  core.EventFinish,
		Usage: usage,
	})
}

// accumulateToolCall accumulates tool call fragments.
func (s *textStream) accumulateToolCall(idx int, tc toolCall) {
	builder, exists := s.toolCallAccumulator[idx]
	if !exists {
		builder = toolCallBuilder{}
	}
	
	if tc.ID != "" {
		builder.id = tc.ID
	}
	if tc.Function.Name != "" {
		builder.name = tc.Function.Name
	}
	if tc.Function.Arguments != "" {
		builder.args.WriteString(tc.Function.Arguments)
	}
	
	s.toolCallAccumulator[idx] = builder
}

// emitToolCalls sends accumulated tool calls as events.
func (s *textStream) emitToolCalls() {
	for _, builder := range s.toolCallAccumulator {
		if builder.name != "" && builder.args.Len() > 0 {
			s.sendEvent(core.Event{
				Type:      core.EventToolCall,
				ToolName:  builder.name,
				ToolInput: json.RawMessage(builder.args.String()),
			})
		}
	}
	// Clear accumulator
	s.toolCallAccumulator = make(map[int]toolCallBuilder)
}

// sendEvent sends an event to the channel.
func (s *textStream) sendEvent(event core.Event) {
	select {
	case s.events <- event:
	case <-s.ctx.Done():
	}
}

// Events returns the event channel.
func (s *textStream) Events() <-chan core.Event {
	return s.events
}

// Close closes the stream.
func (s *textStream) Close() error {
	s.cancel()
	if s.resp != nil && s.resp.Body != nil {
		return s.resp.Body.Close()
	}
	return nil
}

// objectStream extends textStream for structured object streaming.
type objectStream struct {
	textStream
	schema      any
	accumulator strings.Builder // Changed from accumulated to match usage
	finalValue  any
	finalErr    error
	finalOnce   sync.Once
}

// StreamObject implements streaming structured output generation.
func (p *Provider) StreamObject(ctx context.Context, req core.Request, schema interface{}) (core.ObjectStream[any], error) {
	// Generate JSON schema from the type
	schemaBytes, err := p.generateJSONSchema(schema)
	if err != nil {
		return nil, fmt.Errorf("generating JSON schema: %w", err)
	}
	
	apiReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
	}
	
	// Set response format for structured output
	if !p.config.DisableStrictJSONSchema {
		apiReq.ResponseFormat = &responseFormat{
			Type: "json_schema",
			JSONSchema: &jsonSchemaFormat{
				Name:   "response",
				Schema: schemaBytes,
				Strict: true,
			},
		}
	} else {
		// Fall back to json_object mode
		apiReq.ResponseFormat = &responseFormat{
			Type: "json_object",
		}
		// Add instruction to follow schema
		if len(apiReq.Messages) > 0 {
			lastMsg := &apiReq.Messages[len(apiReq.Messages)-1]
			switch content := lastMsg.Content.(type) {
			case string:
				lastMsg.Content = content + fmt.Sprintf("\n\nRespond with JSON matching this schema:\n%s", string(schemaBytes))
			}
		}
	}
	
	// Enable streaming unless disabled
	if !p.config.DisableJSONStreaming {
		apiReq.Stream = true
		apiReq.StreamOptions = &streamOptions{
			IncludeUsage: true,
		}
	} else {
		// Fall back to non-streaming
		return p.simulateObjectStream(ctx, req, schema)
	}
	
	// Strip unsupported parameters
	apiReq = p.stripUnsupportedParams(apiReq)
	
	// Create stream context
	streamCtx, cancel := context.WithCancel(ctx)
	
	// Make streaming request
	resp, err := p.doRequest(streamCtx, "POST", "/chat/completions", apiReq)
	if err != nil {
		cancel()
		return nil, err
	}
	
	if resp.StatusCode != http.StatusOK {
		cancel()
		resp.Body.Close()
		return nil, MapError(resp, p.config.ProviderName)
	}
	
	// Create object stream
	stream := &objectStream{
		textStream: textStream{
			ctx:    streamCtx,
			cancel: cancel,
			events: make(chan core.Event, 100),
			resp:   resp,
		},
		schema: schema,
	}
	
	// Start processing SSE stream
	go stream.process()
	
	return stream, nil
}


// Final returns the final validated object.
func (s *objectStream) Final() (*any, error) {
	// Wait for stream to complete by consuming all events
	for range s.Events() {
		// Just consume
	}
	
	s.finalOnce.Do(func() {
		if s.finalErr != nil {
			return
		}
		
		// Parse accumulated JSON
		jsonStr := s.accumulator.String()
		if jsonStr == "" {
			s.finalErr = fmt.Errorf("no JSON content received")
			return
		}
		
		// Create a new instance of the schema type
		result := reflect.New(reflect.TypeOf(s.schema).Elem()).Interface()
		if err := json.Unmarshal([]byte(jsonStr), result); err != nil {
			s.finalErr = fmt.Errorf("failed to unmarshal JSON: %w", err)
			return
		}
		
		s.finalValue = result
	})
	
	if s.finalErr != nil {
		return nil, s.finalErr
	}
	
	return &s.finalValue, nil
}

// process handles object streaming.
func (s *objectStream) process() {
	defer close(s.events)
	defer s.resp.Body.Close()
	
	// Send start event
	s.sendEvent(core.Event{Type: core.EventStart})
	
	reader := bufio.NewReader(s.resp.Body)
	var usage *core.Usage
	
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				s.sendEvent(core.Event{
					Type: core.EventError,
					Err:  fmt.Errorf("reading stream: %w", err),
				})
			}
			break
		}
		
		// Trim whitespace
		line = bytes.TrimSpace(line)
		
		// Skip empty lines
		if len(line) == 0 {
			continue
		}
		
		// Check for SSE data prefix
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
		
		// Process chunk
		for _, choice := range chunk.Choices {
			if choice.Delta != nil {
				// Accumulate JSON content
				switch content := choice.Delta.Content.(type) {
				case string:
					if content != "" {
						s.accumulator.WriteString(content)
						// Send as text delta for now
						s.sendEvent(core.Event{
							Type:      core.EventTextDelta,
							TextDelta: content,
						})
					}
				}
			}
		}
		
		// Capture usage if present
		if chunk.Usage != nil {
			usage = &core.Usage{
				InputTokens:  chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
				TotalTokens:  chunk.Usage.TotalTokens,
			}
		}
	}
	
	// Parse and validate final JSON
	if s.accumulator.Len() > 0 {
		result := reflect.New(reflect.TypeOf(s.schema)).Interface()
		if err := json.Unmarshal([]byte(s.accumulator.String()), result); err != nil {
			s.sendEvent(core.Event{
				Type: core.EventError,
				Err:  fmt.Errorf("parsing JSON response: %w", err),
			})
		} else {
			// Send the parsed object as raw data
			s.sendEvent(core.Event{
				Type: core.EventRaw,
				Raw:  result,
			})
		}
	}
	
	// Send finish event
	s.sendEvent(core.Event{
		Type:  core.EventFinish,
		Usage: usage,
	})
}

// simulateObjectStream creates a stream from a non-streaming object response.
func (p *Provider) simulateObjectStream(ctx context.Context, req core.Request, schema interface{}) (core.ObjectStream[any], error) {
	// Make non-streaming request
	result, err := p.GenerateObject(ctx, req, schema)
	if err != nil {
		return nil, err
	}
	
	// Create simulated stream
	streamCtx, cancel := context.WithCancel(ctx)
	events := make(chan core.Event, 10)
	
	go func() {
		defer close(events)
		
		// Send start event
		select {
		case events <- core.Event{Type: core.EventStart}:
		case <-streamCtx.Done():
			return
		}
		
		// Send object as JSON text
		if jsonBytes, err := json.Marshal(result.Value); err == nil {
			select {
			case events <- core.Event{
				Type:      core.EventTextDelta,
				TextDelta: string(jsonBytes),
			}:
			case <-streamCtx.Done():
				return
			}
		}
		
		// Send object as raw
		select {
		case events <- core.Event{
			Type: core.EventRaw,
			Raw:  result.Value,
		}:
		case <-streamCtx.Done():
			return
		}
		
		// Send finish event with usage
		select {
		case events <- core.Event{
			Type:  core.EventFinish,
			Usage: &result.Usage,
		}:
		case <-streamCtx.Done():
			return
		}
	}()
	
	// Create object stream with the result
	stream := &objectStream{
		textStream: textStream{
			ctx:    streamCtx,
			cancel: cancel,
			events: events,
		},
		schema:     schema,
		finalValue: result.Value,
	}
	
	return stream, nil
}