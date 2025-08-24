// Package core provides the multi-step execution engine for AI interactions.
// This file implements the runner that orchestrates tool calls and manages
// conversation flow across multiple steps.

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Runner orchestrates multi-step execution with tools.
type Runner struct {
	// provider is the underlying AI provider
	provider Provider
	// maxParallel limits concurrent tool executions
	maxParallel int
	// timeout for individual tool executions
	toolTimeout time.Duration
	// metrics collector (optional)
	metrics MetricsCollector
}

// MetricsCollector collects execution metrics.
type MetricsCollector interface {
	RecordStep(step Step, duration time.Duration)
	RecordToolExecution(name string, duration time.Duration, err error)
	RecordTotalExecution(steps int, duration time.Duration)
}

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithMaxParallel sets the maximum number of parallel tool executions.
func WithMaxParallel(n int) RunnerOption {
	return func(r *Runner) {
		if n > 0 {
			r.maxParallel = n
		}
	}
}

// WithToolTimeout sets the timeout for individual tool executions.
func WithToolTimeout(d time.Duration) RunnerOption {
	return func(r *Runner) {
		r.toolTimeout = d
	}
}

// WithMetrics sets a metrics collector.
func WithMetrics(m MetricsCollector) RunnerOption {
	return func(r *Runner) {
		r.metrics = m
	}
}

// NewRunner creates a new Runner with the given provider and options.
func NewRunner(provider Provider, opts ...RunnerOption) *Runner {
	r := &Runner{
		provider:    provider,
		maxParallel: 10, // default
		toolTimeout: 30 * time.Second,
	}
	
	for _, opt := range opts {
		opt(r)
	}
	
	return r
}

// ExecuteRequest runs a potentially multi-step request with tool execution.
func (r *Runner) ExecuteRequest(ctx context.Context, req Request) (*TextResult, error) {
	startTime := time.Now()
	
	// If no tools or stop condition, delegate to single-shot provider
	if len(req.Tools) == 0 || req.StopWhen == nil {
		return r.provider.GenerateText(ctx, req)
	}
	
	// Prepare for multi-step execution
	messages := make([]Message, len(req.Messages))
	copy(messages, req.Messages)
	
	steps := make([]Step, 0, 4) // pre-allocate for common case
	stepNum := 0
	
	// Create a request without StopWhen for provider calls
	providerReq := req
	providerReq.StopWhen = nil
	
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		stepNum++
		stepStart := time.Now()
		
		// Update messages for this iteration
		providerReq.Messages = messages
		
		// Call the provider for one step
		result, err := r.provider.GenerateText(ctx, providerReq)
		if err != nil {
			return nil, fmt.Errorf("step %d failed: %w", stepNum, err)
		}
		
		// Extract tool calls from the result
		toolCalls := r.extractToolCalls(result)
		
		// Create step record
		step := Step{
			Text:       result.Text,
			ToolCalls:  toolCalls,
			StepNumber: stepNum,
			Timestamp:  time.Now(),
		}
		
		// If there are tool calls, execute them
		if len(toolCalls) > 0 {
			toolResults, err := r.executeTools(ctx, req.Tools, toolCalls, messages)
			if err != nil {
				return nil, fmt.Errorf("tool execution failed at step %d: %w", stepNum, err)
			}
			step.ToolResults = toolResults
			
			// Append assistant message with tool calls
			messages = append(messages, Message{
				Role: Assistant,
				Parts: []Part{
					Text{Text: result.Text},
				},
			})
			
			// Append tool results as messages
			for _, result := range toolResults {
				messages = append(messages, r.toolResultToMessage(result))
			}
		} else {
			// No tools called, append assistant response
			if result.Text != "" {
				messages = append(messages, Message{
					Role: Assistant,
					Parts: []Part{
						Text{Text: result.Text},
					},
				})
			}
		}
		
		steps = append(steps, step)
		
		// Record metrics
		if r.metrics != nil {
			r.metrics.RecordStep(step, time.Since(stepStart))
		}
		
		// Check stop condition
		if req.StopWhen != nil && req.StopWhen.ShouldStop(stepNum, step) {
			break
		}
		
		// Safety: prevent infinite loops
		if stepNum > 100 {
			return nil, fmt.Errorf("maximum step limit (100) exceeded")
		}
		
		// If no tools were called and we got a response, we're done
		if len(toolCalls) == 0 {
			break
		}
	}
	
	// Record total execution metrics
	if r.metrics != nil {
		r.metrics.RecordTotalExecution(len(steps), time.Since(startTime))
	}
	
	// Build final result
	finalText := ""
	if len(steps) > 0 {
		finalText = steps[len(steps)-1].Text
	}
	
	// Calculate total usage
	totalUsage := Usage{}
	// Note: Individual step usage would need to be tracked by provider
	// This is a simplified aggregation for Phase 1
	
	return &TextResult{
		Text:  finalText,
		Steps: steps,
		Usage: totalUsage,
	}, nil
}

// extractToolCalls extracts tool calls from a provider result.
// This would need to be adapted based on how providers return tool calls.
func (r *Runner) extractToolCalls(result *TextResult) []ToolCall {
	// In a real implementation, this would parse the provider's response
	// to extract tool call indicators. For now, return empty.
	if len(result.Steps) > 0 && len(result.Steps[0].ToolCalls) > 0 {
		return result.Steps[0].ToolCalls
	}
	return nil
}

// executeTools runs tool calls in parallel with proper error handling.
func (r *Runner) executeTools(ctx context.Context, tools []ToolHandle, calls []ToolCall, messages []Message) ([]ToolExecution, error) {
	if len(calls) == 0 {
		return nil, nil
	}
	
	results := make([]ToolExecution, len(calls))
	
	// Use a semaphore to limit parallelism
	sem := make(chan struct{}, r.maxParallel)
	
	// Use WaitGroup to track completions
	var wg sync.WaitGroup
	
	// Use atomic counter for early exit on context cancellation
	var canceled int32
	
	// Error channel for collecting errors
	errChan := make(chan error, len(calls))
	
	for i, call := range calls {
		wg.Add(1)
		
		go func(idx int, tc ToolCall) {
			defer wg.Done()
			
			// Check if we should exit early
			if atomic.LoadInt32(&canceled) != 0 {
				return
			}
			
			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				atomic.StoreInt32(&canceled, 1)
				errChan <- ctx.Err()
				return
			}
			
			// Find the tool
			tool := r.findTool(tools, tc.Name)
			if tool == nil {
				results[idx] = ToolExecution{
					ID:    tc.ID,
					Name:  tc.Name,
					Error: fmt.Sprintf("unknown tool: %s", tc.Name),
				}
				return
			}
			
			// Create context with timeout
			toolCtx := ctx
			if r.toolTimeout > 0 {
				var cancel context.CancelFunc
				toolCtx, cancel = context.WithTimeout(ctx, r.toolTimeout)
				defer cancel()
			}
			
			// Execute the tool
			startTime := time.Now()
			result, err := r.executeTool(toolCtx, tool, tc, messages)
			duration := time.Since(startTime)
			
			// Record metrics
			if r.metrics != nil {
				r.metrics.RecordToolExecution(tc.Name, duration, err)
			}
			
			if err != nil {
				results[idx] = ToolExecution{
					ID:    tc.ID,
					Name:  tc.Name,
					Error: err.Error(),
				}
			} else {
				results[idx] = ToolExecution{
					ID:     tc.ID,
					Name:   tc.Name,
					Result: result,
				}
			}
		}(i, call)
	}
	
	// Wait for all tools to complete
	wg.Wait()
	close(errChan)
	
	// Check for context cancellation errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}
	
	return results, nil
}

// findTool finds a tool by name in the tools list.
func (r *Runner) findTool(tools []ToolHandle, name string) ToolHandle {
	for _, tool := range tools {
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}

// executeTool executes a single tool with proper error recovery.
func (r *Runner) executeTool(ctx context.Context, tool ToolHandle, call ToolCall, messages []Message) (result any, err error) {
	// Defer panic recovery
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tool %s panicked: %v", call.Name, r)
		}
	}()
	
	// Create meta information for tool execution
	// The meta type is defined in the tools package, but we pass it as interface{}
	// to avoid circular dependencies
	meta := map[string]interface{}{
		"call_id":     call.ID,
		"messages":    messages,
		"step_number": len(messages), // Approximate step number based on message count
	}
	
	// Execute the tool using its Exec method
	result, err = tool.Exec(ctx, call.Input, meta)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}
	
	return result, nil
}

// toolResultToMessage converts a tool execution result to a message.
func (r *Runner) toolResultToMessage(result ToolExecution) Message {
	// Serialize the result
	var content string
	if result.Error != "" {
		content = fmt.Sprintf("Error executing %s: %s", result.Name, result.Error)
	} else {
		// Marshal result to JSON for inclusion in message
		data, err := json.Marshal(result.Result)
		if err != nil {
			content = fmt.Sprintf("Error serializing result for %s: %v", result.Name, err)
		} else {
			content = string(data)
		}
	}
	
	return Message{
		Role: Tool,
		Parts: []Part{
			Text{Text: content},
		},
		Name: result.Name,
	}
}

// StreamExecuteRequest runs a potentially multi-step request with streaming.
func (r *Runner) StreamExecuteRequest(ctx context.Context, req Request) (TextStream, error) {
	// If no tools, delegate to provider streaming
	if len(req.Tools) == 0 || req.StopWhen == nil {
		return r.provider.StreamText(ctx, req)
	}
	
	// For multi-step streaming, we need to create a custom stream
	// that coordinates multiple provider calls
	return r.createMultiStepStream(ctx, req)
}

// multiStepStream implements TextStream for multi-step execution.
type multiStepStream struct {
	events chan Event
	cancel context.CancelFunc
	done   chan struct{}
}

func (m *multiStepStream) Events() <-chan Event {
	return m.events
}

func (m *multiStepStream) Close() error {
	m.cancel()
	<-m.done
	return nil
}

// createMultiStepStream creates a stream for multi-step execution.
func (r *Runner) createMultiStepStream(ctx context.Context, req Request) (TextStream, error) {
	ctx, cancel := context.WithCancel(ctx)
	
	stream := &multiStepStream{
		events: make(chan Event, 100), // buffered for performance
		cancel: cancel,
		done:   make(chan struct{}),
	}
	
	// Run the multi-step execution in a goroutine
	go func() {
		defer close(stream.done)
		defer close(stream.events)
		
		// Send start event
		stream.events <- Event{
			Type:      EventStart,
			Timestamp: time.Now(),
		}
		
		messages := make([]Message, len(req.Messages))
		copy(messages, req.Messages)
		
		stepNum := 0
		providerReq := req
		providerReq.StopWhen = nil
		
		for {
			select {
			case <-ctx.Done():
				stream.events <- Event{
					Type:      EventError,
					Err:       ctx.Err(),
					Timestamp: time.Now(),
				}
				return
			default:
			}
			
			stepNum++
			providerReq.Messages = messages
			
			// Stream from provider
			providerStream, err := r.provider.StreamText(ctx, providerReq)
			if err != nil {
				stream.events <- Event{
					Type:      EventError,
					Err:       err,
					Timestamp: time.Now(),
				}
				return
			}
			
			// Collect events and build step
			var stepText string
			var toolCalls []ToolCall
			
			for event := range providerStream.Events() {
				// Forward most events
				stream.events <- event
				
				// Collect data for step
				switch event.Type {
				case EventTextDelta:
					stepText += event.TextDelta
				case EventToolCall:
					toolCalls = append(toolCalls, ToolCall{
						ID:    event.ToolID,
						Name:  event.ToolName,
						Input: event.ToolInput,
					})
				case EventError:
					return
				}
			}
			
			providerStream.Close()
			
			// Create step
			step := Step{
				Text:       stepText,
				ToolCalls:  toolCalls,
				StepNumber: stepNum,
				Timestamp:  time.Now(),
			}
			
			// Execute tools if any
			if len(toolCalls) > 0 {
				toolResults, err := r.executeTools(ctx, req.Tools, toolCalls, messages)
				if err != nil {
					stream.events <- Event{
						Type:      EventError,
						Err:       err,
						Timestamp: time.Now(),
					}
					return
				}
				
				step.ToolResults = toolResults
				
				// Send tool result events
				for _, result := range toolResults {
					stream.events <- Event{
						Type:       EventToolResult,
						ToolName:   result.Name,
						ToolResult: result.Result,
						Timestamp:  time.Now(),
					}
				}
				
				// Update messages
				messages = append(messages, Message{
					Role:  Assistant,
					Parts: []Part{Text{Text: stepText}},
				})
				
				for _, result := range toolResults {
					messages = append(messages, r.toolResultToMessage(result))
				}
			} else {
				if stepText != "" {
					messages = append(messages, Message{
						Role:  Assistant,
						Parts: []Part{Text{Text: stepText}},
					})
				}
			}
			
			// Send step finish event
			stream.events <- Event{
				Type:       EventFinishStep,
				StepNumber: stepNum,
				Timestamp:  time.Now(),
			}
			
			// Check stop condition
			if req.StopWhen != nil && req.StopWhen.ShouldStop(stepNum, step) {
				break
			}
			
			// Safety limit
			if stepNum > 100 {
				stream.events <- Event{
					Type:      EventError,
					Err:       fmt.Errorf("maximum step limit exceeded"),
					Timestamp: time.Now(),
				}
				return
			}
			
			// If no tools were called, we're done
			if len(toolCalls) == 0 {
				break
			}
		}
		
		// Send finish event
		stream.events <- Event{
			Type:      EventFinish,
			Timestamp: time.Now(),
		}
	}()
	
	return stream, nil
}