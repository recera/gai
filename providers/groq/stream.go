// Package groq - Streaming implementation
package groq

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/recera/gai/core"
)

// StreamText streams text generation with events.
func (p *Provider) StreamText(ctx context.Context, req core.Request) (core.TextStream, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	model := p.getModel(req)
	modelInfo := p.getModelInfo(model)

	// Validate streaming support
	if !modelInfo.SupportsStreaming {
		return nil, fmt.Errorf("model %s does not support streaming", model)
	}

	// Convert request for streaming
	groqReq, err := p.convertRequest(req, modelInfo)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
	}

	// Enable streaming
	groqReq.Stream = true
	groqReq.StreamOptions = &streamOptions{IncludeUsage: true}

	// Make the request
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", groqReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		return nil, p.parseError(resp)
	}

	// Create and return the stream
	stream := &groqTextStream{
		provider: p,
		response: resp,
		tools:    req.Tools,
		events:   make(chan core.Event, 100),
		done:     make(chan struct{}),
	}

	// Start processing the stream
	go stream.processStream(ctx)

	return stream, nil
}

// groqTextStream implements core.TextStream for Groq streaming responses.
type groqTextStream struct {
	provider *Provider
	response *http.Response
	tools    []core.ToolHandle
	events   chan core.Event
	done     chan struct{}
	mu       sync.Mutex
	closed   bool
	err      error
}

// Events returns the channel of streaming events.
func (s *groqTextStream) Events() <-chan core.Event {
	return s.events
}

// Close terminates the stream.
func (s *groqTextStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	close(s.done)
	
	if s.response != nil {
		s.response.Body.Close()
	}

	return nil
}

// processStream processes the SSE stream from Groq.
func (s *groqTextStream) processStream(ctx context.Context) {
	defer func() {
		close(s.events)
		s.response.Body.Close()
	}()

	// Send start event
	s.sendEvent(core.Event{
		Type:      core.EventStart,
		Timestamp: time.Now(),
	})

	scanner := bufio.NewScanner(s.response.Body)
	var currentToolCalls []toolCall
	var fullText strings.Builder

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			s.sendEvent(core.Event{
				Type:      core.EventError,
				Err:       ctx.Err(),
				Timestamp: time.Now(),
			})
			return
		case <-s.done:
			return
		default:
		}

		line := scanner.Text()
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE format
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			
			// Check for stream end
			if data == "[DONE]" {
				// Process any pending tool calls
				if len(currentToolCalls) > 0 {
					err := s.executeToolCalls(ctx, currentToolCalls)
					if err != nil {
						s.sendEvent(core.Event{
							Type:      core.EventError,
							Err:       err,
							Timestamp: time.Now(),
						})
						return
					}
					currentToolCalls = nil
				}
				
				// Send final event
				s.sendEvent(core.Event{
					Type:      core.EventFinish,
					Timestamp: time.Now(),
				})
				return
			}

			// Parse the JSON chunk
			var chunk streamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				// Skip malformed chunks
				continue
			}

			// Process the chunk
			s.processChunk(chunk, &currentToolCalls, &fullText)
		}
	}

	if err := scanner.Err(); err != nil {
		s.sendEvent(core.Event{
			Type:      core.EventError,
			Err:       fmt.Errorf("stream reading error: %w", err),
			Timestamp: time.Now(),
		})
	}
}

// processChunk processes a single streaming chunk.
func (s *groqTextStream) processChunk(chunk streamChunk, currentToolCalls *[]toolCall, fullText *strings.Builder) {
	if len(chunk.Choices) == 0 {
		return
	}

	choice := chunk.Choices[0]
	delta := choice.Delta

	// Handle text deltas
	if delta.Content != nil && *delta.Content != "" {
		fullText.WriteString(*delta.Content)
		s.sendEvent(core.Event{
			Type:      core.EventTextDelta,
			TextDelta: *delta.Content,
			Timestamp: time.Now(),
		})
	}

	// Handle tool calls
	if len(delta.ToolCalls) > 0 {
		for _, tc := range delta.ToolCalls {
			// Groq streams tool calls as deltas
			if tc.ID != "" {
				// New tool call
				*currentToolCalls = append(*currentToolCalls, tc)
				s.sendEvent(core.Event{
					Type:      core.EventToolCall,
					ToolName:  tc.Function.Name,
					ToolID:    tc.ID,
					Timestamp: time.Now(),
				})
			} else {
				// Update existing tool call (arguments delta)
				if len(*currentToolCalls) > 0 {
					lastIndex := len(*currentToolCalls) - 1
					(*currentToolCalls)[lastIndex].Function.Arguments += tc.Function.Arguments
				}
			}
		}
	}

	// Handle finish reason
	if choice.FinishReason != nil && *choice.FinishReason == "tool_calls" {
		// Tool calls are complete, execute them
		if len(*currentToolCalls) > 0 {
			ctx := context.Background() // Use background context for tool execution
			if err := s.executeToolCalls(ctx, *currentToolCalls); err != nil {
				s.sendEvent(core.Event{
					Type:      core.EventError,
					Err:       err,
					Timestamp: time.Now(),
				})
			}
			*currentToolCalls = nil
		}
	}

	// Handle usage information
	if chunk.Usage != nil {
		s.sendEvent(core.Event{
			Type: core.EventFinish,
			Usage: &core.Usage{
				InputTokens:  chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
				TotalTokens:  chunk.Usage.TotalTokens,
			},
			Timestamp: time.Now(),
		})
	}
}

// executeToolCalls executes tool calls during streaming.
func (s *groqTextStream) executeToolCalls(ctx context.Context, toolCalls []toolCall) error {
	for _, tc := range toolCalls {
		// Find the tool
		var tool core.ToolHandle
		for _, t := range s.tools {
			if t.Name() == tc.Function.Name {
				tool = t
				break
			}
		}

		if tool == nil {
			return fmt.Errorf("unknown tool: %s", tc.Function.Name)
		}

		// Parse tool input
		var toolInput json.RawMessage = []byte(tc.Function.Arguments)
		
		s.sendEvent(core.Event{
			Type:      core.EventToolCall,
			ToolName:  tc.Function.Name,
			ToolID:    tc.ID,
			ToolInput: toolInput,
			Timestamp: time.Now(),
		})

		// Execute the tool
		meta := map[string]interface{}{
			"call_id":  tc.ID,
			"provider": "groq",
		}

		result, err := tool.Exec(ctx, toolInput, meta)
		if err != nil {
			s.sendEvent(core.Event{
				Type:      core.EventError,
				Err:       fmt.Errorf("tool %s execution failed: %w", tc.Function.Name, err),
				Timestamp: time.Now(),
			})
			continue
		}

		// Send tool result
		s.sendEvent(core.Event{
			Type:       core.EventToolResult,
			ToolResult: result,
			ToolName:   tc.Function.Name,
			Timestamp:  time.Now(),
		})
	}

	return nil
}

// sendEvent safely sends an event to the channel.
func (s *groqTextStream) sendEvent(event core.Event) {
	select {
	case s.events <- event:
	case <-s.done:
	default:
		// Channel is full, drop the event to prevent blocking
	}
}

// Streaming response types
type streamChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []deltaChoice `json:"choices"`
	Usage   *usage        `json:"usage,omitempty"`
}

type deltaChoice struct {
	Index        int          `json:"index"`
	Delta        messageDelta `json:"delta"`
	FinishReason *string      `json:"finish_reason,omitempty"`
}

type messageDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   *string    `json:"content,omitempty"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
}

// GenerateObject generates a structured object (not yet implemented for Groq).
func (p *Provider) GenerateObject(ctx context.Context, req core.Request, schema any) (*core.ObjectResult[any], error) {
	// For now, we'll implement this using JSON mode
	model := p.getModel(req)
	modelInfo := p.getModelInfo(model)

	if !modelInfo.SupportsJSON {
		return nil, fmt.Errorf("model %s does not support structured outputs", model)
	}

	// Convert schema to JSON schema format
	// This is a simplified implementation
	groqReq, err := p.convertRequest(req, modelInfo)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
	}

	// Enable JSON mode
	groqReq.ResponseFormat = &responseFormat{
		Type: "json_object",
	}

	resp, err := p.doRequest(ctx, "POST", "/chat/completions", groqReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, p.parseError(resp)
	}

	var groqResp chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(groqResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	// Parse the JSON response
	var result interface{}
	content := ""
	if groqResp.Choices[0].Message.Content != nil {
		if s, ok := groqResp.Choices[0].Message.Content.(string); ok {
			content = s
		}
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parsing JSON response: %w", err)
	}

	return &core.ObjectResult[any]{
		Value: result,
		Usage: core.Usage{
			InputTokens:  groqResp.Usage.PromptTokens,
			OutputTokens: groqResp.Usage.CompletionTokens,
			TotalTokens:  groqResp.Usage.TotalTokens,
		},
	}, nil
}

// StreamObject streams generation of a structured object (placeholder implementation).
func (p *Provider) StreamObject(ctx context.Context, req core.Request, schema any) (core.ObjectStream[any], error) {
	return nil, fmt.Errorf("StreamObject not yet implemented for Groq provider")
}