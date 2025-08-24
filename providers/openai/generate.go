package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/recera/gai/core"
)

// GenerateText implements the core.Provider interface for text generation.
// It supports multi-step tool execution when tools are provided.
func (p *Provider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {

	// If tools are provided and multi-step execution is needed, use runner
	if len(req.Tools) > 0 && req.StopWhen != nil {
		return p.generateWithTools(ctx, req)
	}

	// Simple single-shot generation
	apiReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
	}

	resp, err := p.doRequest(ctx, "POST", "/chat/completions", apiReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var apiResp chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Convert to core.TextResult
	result := &core.TextResult{
		Usage: core.Usage{
			InputTokens:  apiResp.Usage.PromptTokens,
			OutputTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:  apiResp.Usage.TotalTokens,
		},
		Raw: apiResp,
	}

	if len(apiResp.Choices) > 0 {
		choice := apiResp.Choices[0]
		
		// Extract text content
		switch content := choice.Message.Content.(type) {
		case string:
			result.Text = content
		case []interface{}:
			// Handle multipart content
			text := ""
			for _, part := range content {
				if p, ok := part.(map[string]interface{}); ok {
					if p["type"] == "text" {
						if t, ok := p["text"].(string); ok {
							text += t
						}
					}
				}
			}
			result.Text = text
		}

		// Handle tool calls if present
		if len(choice.Message.ToolCalls) > 0 {
			step := core.Step{
				Text: result.Text,
				ToolCalls: p.convertToolCallsFromAPI(choice.Message.ToolCalls),
			}
			result.Steps = append(result.Steps, step)
		}
	}


	return result, nil
}

// generateWithTools handles multi-step execution with tools.
func (p *Provider) generateWithTools(ctx context.Context, req core.Request) (*core.TextResult, error) {
	messages := make([]core.Message, len(req.Messages))
	copy(messages, req.Messages)
	
	var steps []core.Step
	var totalUsage core.Usage
	stepCount := 0
	maxSteps := 10 // Safety limit

	for stepCount < maxSteps {
		// Make API request
		apiReq, err := p.convertRequest(core.Request{
			Model:       req.Model,
			Messages:    messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
			Tools:       req.Tools,
			ToolChoice:  req.ToolChoice,
		})
		if err != nil {
			return nil, fmt.Errorf("converting request for step %d: %w", stepCount, err)
		}

		resp, err := p.doRequest(ctx, "POST", "/chat/completions", apiReq)
		if err != nil {
			return nil, fmt.Errorf("API request for step %d: %w", stepCount, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, p.parseError(resp)
		}

		var apiResp chatCompletionResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return nil, fmt.Errorf("decoding response for step %d: %w", stepCount, err)
		}

		// Update usage
		totalUsage.InputTokens += apiResp.Usage.PromptTokens
		totalUsage.OutputTokens += apiResp.Usage.CompletionTokens
		totalUsage.TotalTokens += apiResp.Usage.TotalTokens

		if len(apiResp.Choices) == 0 {
			break
		}

		choice := apiResp.Choices[0]
		
		// Extract text
		var text string
		switch content := choice.Message.Content.(type) {
		case string:
			text = content
		case []interface{}:
			for _, part := range content {
				if p, ok := part.(map[string]interface{}); ok {
					if p["type"] == "text" {
						if t, ok := p["text"].(string); ok {
							text += t
						}
					}
				}
			}
		}

		// Create step
		step := core.Step{
			Text:       text,
			StepNumber: stepCount,
		}

		// Add assistant message to conversation
		messages = append(messages, core.Message{
			Role: core.Assistant,
			Parts: []core.Part{core.Text{Text: text}},
		})

		// Handle tool calls
		if len(choice.Message.ToolCalls) > 0 {
			step.ToolCalls = p.convertToolCallsFromAPI(choice.Message.ToolCalls)
			
			// Execute tools
			toolResults, err := p.executeTools(ctx, req.Tools, step.ToolCalls, messages)
			if err != nil {
				return nil, fmt.Errorf("executing tools for step %d: %w", stepCount, err)
			}
			step.ToolResults = toolResults

			// Add tool results to messages
			for _, result := range toolResults {
				toolMsg := core.Message{
					Role: core.Tool,
					Parts: []core.Part{
						core.Text{Text: p.formatToolResult(result)},
					},
				}
				messages = append(messages, toolMsg)
			}
		}

		steps = append(steps, step)
		stepCount++

		// Check stop condition
		if req.StopWhen != nil && req.StopWhen.ShouldStop(stepCount, step) {
			break
		}

		// If no tool calls were made, we're done
		if len(step.ToolCalls) == 0 {
			break
		}
	}

	// Build final result
	finalText := ""
	if len(steps) > 0 {
		finalText = steps[len(steps)-1].Text
	}

	return &core.TextResult{
		Text:  finalText,
		Steps: steps,
		Usage: totalUsage,
	}, nil
}

// convertToolCallsFromAPI converts OpenAI tool calls to core format.
func (p *Provider) convertToolCallsFromAPI(toolCalls []toolCall) []core.ToolCall {
	result := make([]core.ToolCall, 0, len(toolCalls))
	for _, tc := range toolCalls {
		result = append(result, core.ToolCall{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: json.RawMessage(tc.Function.Arguments),
		})
	}
	return result
}

// executeTools executes tool calls and returns results.
func (p *Provider) executeTools(ctx context.Context, tools []core.ToolHandle, calls []core.ToolCall, messages []core.Message) ([]core.ToolExecution, error) {
	results := make([]core.ToolExecution, len(calls))
	
	// Execute tools sequentially for now (can be parallelized)
	for i, call := range calls {
		tool := p.findTool(tools, call.Name)
		if tool == nil {
			results[i] = core.ToolExecution{
				ID:    call.ID,
				Name:  call.Name,
				Error: fmt.Sprintf("tool not found: %s", call.Name),
			}
			continue
		}

		// Execute tool
		result, err := tool.Exec(ctx, call.Input, map[string]interface{}{
			"messages": messages,
			"call_id":  call.ID,
		})
		
		if err != nil {
			results[i] = core.ToolExecution{
				ID:    call.ID,
				Name:  call.Name,
				Error: err.Error(),
			}
		} else {
			results[i] = core.ToolExecution{
				ID:     call.ID,
				Name:   call.Name,
				Result: result,
			}
		}
	}
	
	return results, nil
}

// findTool finds a tool by name.
func (p *Provider) findTool(tools []core.ToolHandle, name string) core.ToolHandle {
	for _, tool := range tools {
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}

// formatToolResult formats a tool execution result for the API.
func (p *Provider) formatToolResult(result core.ToolExecution) string {
	if result.Error != "" {
		return fmt.Sprintf(`{"error": "%s"}`, result.Error)
	}
	
	data, err := json.Marshal(result.Result)
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal result: %v"}`, err)
	}
	
	return string(data)
}

// GenerateObject generates a structured object conforming to the provided schema.
func (p *Provider) GenerateObject(ctx context.Context, req core.Request, schema any) (*core.ObjectResult[any], error) {

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

	// Make API request
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", apiReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var apiResp chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Extract and parse the JSON response
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := apiResp.Choices[0]
	var content string
	
	switch c := choice.Message.Content.(type) {
	case string:
		content = c
	default:
		return nil, fmt.Errorf("unexpected content type: %T", c)
	}

	// Parse the JSON content
	var value any
	if err := json.Unmarshal([]byte(content), &value); err != nil {
		return nil, fmt.Errorf("parsing JSON response: %w", err)
	}

	result := &core.ObjectResult[any]{
		Value: value,
		Usage: core.Usage{
			InputTokens:  apiResp.Usage.PromptTokens,
			OutputTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:  apiResp.Usage.TotalTokens,
		},
		Raw: apiResp,
	}


	return result, nil
}