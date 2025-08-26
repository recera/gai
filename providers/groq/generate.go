// Package groq - Text generation implementation
package groq

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/obs"
)

// GenerateText generates text with optional multi-step tool execution.
func (p *Provider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	model := p.getModel(req)

	// Use comprehensive GenAI observability wrapper
	return obs.WithGenAIObservability(ctx, "groq", model, obs.GenAIOpChatCompletion, req, func(ctx context.Context) (*core.TextResult, error) {
		return p.executeGenerateText(ctx, req, model)
	})
}

// executeGenerateText handles the actual text generation logic (extracted for observability)
func (p *Provider) executeGenerateText(ctx context.Context, req core.Request, model string) (*core.TextResult, error) {
	modelInfo := p.getModelInfo(model)

	// Validate model capabilities
	if len(req.Tools) > 0 && !modelInfo.SupportsTools {
		return nil, fmt.Errorf("model %s does not support tool calling", model)
	}

	// For multi-step execution with tools
	if len(req.Tools) > 0 && req.StopWhen != nil {
		return p.executeMultiStep(ctx, req, modelInfo)
	}

	// Single-step execution
	groqReq, err := p.convertRequest(req, modelInfo)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
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

	return p.convertTextResponse(groqResp, req.Messages), nil
}

// executeMultiStep handles multi-step tool execution with stopWhen conditions.
func (p *Provider) executeMultiStep(ctx context.Context, req core.Request, modelInfo ModelInfo) (*core.TextResult, error) {
	messages := make([]core.Message, len(req.Messages))
	copy(messages, req.Messages)
	
	var steps []core.Step
	stepNumber := 0
	
	for {
		stepNumber++
		
		// Create request for this step
		stepReq := req
		stepReq.Messages = messages
		
		// Convert and execute
		groqReq, err := p.convertRequest(stepReq, modelInfo)
		if err != nil {
			return nil, fmt.Errorf("converting request for step %d: %w", stepNumber, err)
		}

		resp, err := p.doRequest(ctx, "POST", "/chat/completions", groqReq)
		if err != nil {
			return nil, fmt.Errorf("request failed for step %d: %w", stepNumber, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, p.parseError(resp)
		}

		var groqResp chatCompletionResponse
		if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
			return nil, fmt.Errorf("decoding response for step %d: %w", stepNumber, err)
		}

		// Process the response
		step, newMessages, err := p.processStepResponse(groqResp, messages, req.Tools, stepNumber)
		if err != nil {
			return nil, fmt.Errorf("processing step %d: %w", stepNumber, err)
		}
		
		steps = append(steps, step)
		messages = newMessages
		
		// Check stop condition
		if req.StopWhen != nil && req.StopWhen.ShouldStop(stepNumber, step) {
			break
		}
		
		// Safety check - prevent infinite loops
		if stepNumber >= 20 {
			break
		}
		
		// If no tool calls were made, we're done
		if len(step.ToolCalls) == 0 {
			break
		}
	}
	
	// Build final response
	finalText := ""
	if len(steps) > 0 && len(steps[len(steps)-1].ToolResults) == 0 {
		// Last step has text output
		finalText = steps[len(steps)-1].Text
	}
	
	// Calculate total usage
	totalUsage := core.Usage{}
	for range steps {
		// Usage would be accumulated from each step - simplified for now
		totalUsage.TotalTokens += 100 // Placeholder
	}
	
	return &core.TextResult{
		Text:  finalText,
		Steps: steps,
		Usage: totalUsage,
	}, nil
}

// processStepResponse processes a single step response, handling tool calls.
func (p *Provider) processStepResponse(groqResp chatCompletionResponse, messages []core.Message, tools []core.ToolHandle, stepNumber int) (core.Step, []core.Message, error) {
	if len(groqResp.Choices) == 0 {
		return core.Step{}, nil, fmt.Errorf("no choices in response")
	}

	choice := groqResp.Choices[0]
	step := core.Step{
		StepNumber: stepNumber,
		Timestamp:  time.Now(),
	}
	
	newMessages := make([]core.Message, len(messages))
	copy(newMessages, messages)

	// Handle tool calls if present
	if len(choice.Message.ToolCalls) > 0 {
		// First, add the assistant message with tool calls to the conversation
		content := ""
		if choice.Message.Content != nil {
			if s, ok := choice.Message.Content.(string); ok {
				content = s
			}
		}
		assistantMsg := core.Message{
			Role: core.Assistant,
			Parts: []core.Part{
				core.Text{Text: content},
			},
		}
		newMessages = append(newMessages, assistantMsg)
		
		// Convert tool calls
		for _, tc := range choice.Message.ToolCalls {
			step.ToolCalls = append(step.ToolCalls, core.ToolCall{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: json.RawMessage(tc.Function.Arguments),
			})
		}
		
		// Execute tools and add their results
		for _, toolCall := range step.ToolCalls {
			// Find the tool
			var tool core.ToolHandle
			for _, t := range tools {
				if t.Name() == toolCall.Name {
					tool = t
					break
				}
			}
			
			if tool == nil {
				return step, nil, fmt.Errorf("unknown tool: %s", toolCall.Name)
			}
			
			// Execute the tool
			meta := map[string]interface{}{
				"call_id":     toolCall.ID,
				"step_number": stepNumber,
				"provider":    "groq",
			}
			
			result, err := tool.Exec(context.Background(), toolCall.Input, meta)
			if err != nil {
				step.ToolResults = append(step.ToolResults, core.ToolExecution{
					ID:    toolCall.ID,
					Name:  toolCall.Name,
					Error: err.Error(),
				})
				
				// Add error result to messages
				newMessages = append(newMessages, core.Message{
					Role: core.Tool,
					Parts: []core.Part{
						core.Text{Text: fmt.Sprintf("Error: %s", err.Error())},
					},
					// Store tool call ID for Groq API compatibility
					Name: fmt.Sprintf("tool_call_id:%s", toolCall.ID),
				})
			} else {
				step.ToolResults = append(step.ToolResults, core.ToolExecution{
					ID:     toolCall.ID,
					Name:   toolCall.Name,
					Result: result,
				})
				
				// Add successful result to messages with proper tool_call_id tracking
				resultJSON, _ := json.Marshal(result)
				newMessages = append(newMessages, core.Message{
					Role: core.Tool,
					Parts: []core.Part{
						core.Text{Text: string(resultJSON)},
					},
					// Store the tool call ID that this message responds to
					Name: fmt.Sprintf("tool_call_id:%s", toolCall.ID),
				})
			}
		}
	} else {
		// No tool calls, this is text output
		content := ""
		if choice.Message.Content != nil {
			if s, ok := choice.Message.Content.(string); ok {
				content = s
			}
		}
		step.Text = content
		
		// Add the final assistant message 
		newMessages = append(newMessages, core.Message{
			Role: core.Assistant,
			Parts: []core.Part{
				core.Text{Text: content},
			},
		})
	}
	
	return step, newMessages, nil
}

// convertRequest converts a core.Request to a Groq chat completion request.
func (p *Provider) convertRequest(req core.Request, modelInfo ModelInfo) (*chatCompletionRequest, error) {
	groqReq := &chatCompletionRequest{
		Model: p.getModel(req),
		N:     1, // Only n=1 is supported by Groq
	}

	// Handle temperature - some models may have constraints
	if req.Temperature > 0 {
		groqReq.Temperature = &req.Temperature
	}

	// Handle token limits with model-specific logic
	if req.MaxTokens > 0 {
		if req.MaxTokens > modelInfo.MaxCompletionTokens {
			// Cap at model maximum
			maxTokens := modelInfo.MaxCompletionTokens
			groqReq.MaxTokens = &maxTokens
		} else {
			groqReq.MaxTokens = &req.MaxTokens
		}
	}

	// Convert messages with special handling for tool responses
	messages, err := p.convertMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("converting messages: %w", err)
	}
	groqReq.Messages = messages

	// Convert tools if present and supported
	if len(req.Tools) > 0 && modelInfo.SupportsTools {
		groqReq.Tools = p.convertTools(req.Tools)
		groqReq.ToolChoice = p.convertToolChoice(req.ToolChoice)
		
		// Enable parallel tool calls if supported
		if modelInfo.PerformanceClass == "ultra-fast" || modelInfo.PerformanceClass == "fast" {
			parallelCalls := true
			groqReq.ParallelToolCalls = &parallelCalls
		}
	}

	// Set service tier
	if p.serviceTier != "" {
		groqReq.ServiceTier = &p.serviceTier
	}

	// Note: Groq doesn't support structured_outputs parameter yet
	// JSON mode is handled via response_format in GenerateObject instead

	// Handle provider-specific options
	if opts, ok := req.ProviderOptions["groq"].(map[string]interface{}); ok {
		p.applyProviderOptions(groqReq, opts)
	}

	return groqReq, nil
}

// convertMessages converts core messages to Groq format with proper tool call ID handling.
func (p *Provider) convertMessages(messages []core.Message) ([]chatMessage, error) {
	result := make([]chatMessage, 0, len(messages))
	
	for _, msg := range messages {
		cm := chatMessage{
			Role: string(msg.Role),
			Name: msg.Name,
		}

		// Handle different message types
		switch msg.Role {
		case core.Assistant:
			// Assistant messages might contain tool calls
			if len(msg.Parts) == 1 {
				if text, ok := msg.Parts[0].(core.Text); ok {
					cm.Content = text.Text
				}
			}
			
		case core.Tool:
			// Tool messages need tool_call_id - CRITICAL for Groq compatibility
			if len(msg.Parts) == 1 {
				if text, ok := msg.Parts[0].(core.Text); ok {
					cm.Content = text.Text
					
					// Extract tool_call_id from the message name if stored there
					if strings.HasPrefix(msg.Name, "tool_call_id:") {
						cm.ToolCallID = strings.TrimPrefix(msg.Name, "tool_call_id:")
					} else {
						// Fallback: generate a reasonable tool call ID
						// In production, this should be properly tracked
						cm.ToolCallID = fmt.Sprintf("call_%d", len(result))
					}
				}
			}
			
		default:
			// Handle regular messages (user, system)
			if len(msg.Parts) == 1 {
				if text, ok := msg.Parts[0].(core.Text); ok {
					cm.Content = text.Text
				} else {
					// Convert to content parts for multimodal
					parts, err := p.convertParts(msg.Parts)
					if err != nil {
						return nil, err
					}
					cm.Content = parts
				}
			} else if len(msg.Parts) > 1 {
				// Multiple parts - use content array
				parts, err := p.convertParts(msg.Parts)
				if err != nil {
					return nil, err
				}
				cm.Content = parts
			}
		}

		result = append(result, cm)
	}

	return result, nil
}

// convertParts converts message parts to Groq content parts.
func (p *Provider) convertParts(parts []core.Part) ([]contentPart, error) {
	result := make([]contentPart, 0, len(parts))
	
	for _, part := range parts {
		switch p := part.(type) {
		case core.Text:
			result = append(result, contentPart{
				Type: "text",
				Text: p.Text,
			})
		case core.ImageURL:
			result = append(result, contentPart{
				Type: "image_url",
				ImageURL: &imageURLPart{
					URL:    p.URL,
					Detail: p.Detail,
				},
			})
		case core.Audio, core.Video, core.File:
			// Groq supports some multimodal content but not all
			return nil, fmt.Errorf("unsupported part type for Groq: %T", p)
		default:
			return nil, fmt.Errorf("unknown part type: %T", p)
		}
	}
	
	return result, nil
}

// convertTools converts core tools to Groq format.
func (p *Provider) convertTools(tools []core.ToolHandle) []chatTool {
	result := make([]chatTool, 0, len(tools))
	
	for _, tool := range tools {
		strict := true // Groq supports strict schemas
		result = append(result, chatTool{
			Type: "function",
			Function: function{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.InSchemaJSON(),
				Strict:      &strict,
			},
		})
	}
	
	return result
}

// convertToolChoice converts core tool choice to Groq format.
func (p *Provider) convertToolChoice(choice core.ToolChoice) interface{} {
	switch choice {
	case core.ToolAuto:
		return "auto"
	case core.ToolNone:
		return "none"
	case core.ToolRequired:
		return "required"
	default:
		return "auto"
	}
}

// applyProviderOptions applies Groq-specific options.
func (p *Provider) applyProviderOptions(req *chatCompletionRequest, opts map[string]interface{}) {
	if v, ok := opts["presence_penalty"].(float32); ok {
		req.PresencePenalty = &v
	}
	if v, ok := opts["frequency_penalty"].(float32); ok {
		req.FrequencyPenalty = &v
	}
	if v, ok := opts["top_p"].(float32); ok {
		req.TopP = &v
	}
	if v, ok := opts["top_k"].(int); ok {
		req.TopK = &v
	}
	if v, ok := opts["stop"].([]string); ok {
		req.Stop = v
	}
	if v, ok := opts["seed"].(int); ok {
		req.Seed = &v
	}
	if v, ok := opts["user"].(string); ok {
		req.User = v
	}
	if v, ok := opts["service_tier"].(string); ok {
		req.ServiceTier = &v
	}
}

// Response types and conversion
type chatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type choice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
	LogProbs     interface{} `json:"logprobs"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// convertTextResponse converts Groq response to core format.
func (p *Provider) convertTextResponse(groqResp chatCompletionResponse, originalMessages []core.Message) *core.TextResult {
	if len(groqResp.Choices) == 0 {
		return &core.TextResult{
			Text: "",
			Usage: core.Usage{
				InputTokens:  groqResp.Usage.PromptTokens,
				OutputTokens: groqResp.Usage.CompletionTokens,
				TotalTokens:  groqResp.Usage.TotalTokens,
			},
		}
	}

	choice := groqResp.Choices[0]
	
	content := ""
	if choice.Message.Content != nil {
		if s, ok := choice.Message.Content.(string); ok {
			content = s
		}
	}
	
	return &core.TextResult{
		Text: content,
		Usage: core.Usage{
			InputTokens:  groqResp.Usage.PromptTokens,
			OutputTokens: groqResp.Usage.CompletionTokens,
			TotalTokens:  groqResp.Usage.TotalTokens,
		},
	}
}