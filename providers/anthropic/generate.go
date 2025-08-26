package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/obs"
)

// GenerateText implements the core.Provider interface for text generation.
// It supports multi-step tool execution when tools are provided.
func (p *Provider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
	model := p.getModel(req)

	// Use comprehensive GenAI observability wrapper
	return obs.WithGenAIObservability(ctx, "anthropic", model, obs.GenAIOpChatCompletion, req, func(ctx context.Context) (*core.TextResult, error) {
		return p.executeGenerateText(ctx, req)
	})
}

// executeGenerateText handles the actual text generation logic (extracted for observability)
func (p *Provider) executeGenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
	// If tools are provided and multi-step execution is needed, use multi-step runner
	if len(req.Tools) > 0 && req.StopWhen != nil {
		return p.generateWithTools(ctx, req)
	}

	// Simple single-shot generation
	apiReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
	}

	resp, err := p.doRequest(ctx, "POST", "/v1/messages", apiReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var apiResp messagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Convert to core.TextResult
	result := &core.TextResult{
		Usage: core.Usage{
			InputTokens:  apiResp.Usage.InputTokens,
			OutputTokens: apiResp.Usage.OutputTokens,
			TotalTokens:  apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		},
		Raw: apiResp,
	}

	// Extract text and tool calls from content blocks
	var textParts []string
	var toolCalls []core.ToolCall

	for _, block := range apiResp.Content {
		switch block.Type {
		case "text":
			textParts = append(textParts, block.Text)
		case "tool_use":
			// Convert tool use to core format
			inputJSON, err := json.Marshal(block.Input)
			if err != nil {
				return nil, fmt.Errorf("marshaling tool input: %w", err)
			}
			
			toolCalls = append(toolCalls, core.ToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: json.RawMessage(inputJSON),
			})
		}
	}

	// Combine text parts
	for i, part := range textParts {
		if i > 0 {
			result.Text += "\n"
		}
		result.Text += part
	}

	// If there were tool calls, create a step
	if len(toolCalls) > 0 {
		step := core.Step{
			Text:       result.Text,
			ToolCalls:  toolCalls,
			StepNumber: 0,
			Timestamp:  time.Now(),
		}
		result.Steps = append(result.Steps, step)
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
		// Convert current conversation to API request
		apiReq, err := p.convertRequest(core.Request{
			Model:       req.Model,
			Messages:    messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
			Tools:       req.Tools,
		})
		if err != nil {
			return nil, fmt.Errorf("converting request for step %d: %w", stepCount, err)
		}

		// Make API request
		resp, err := p.doRequest(ctx, "POST", "/v1/messages", apiReq)
		if err != nil {
			return nil, fmt.Errorf("API request for step %d: %w", stepCount, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, p.parseError(resp)
		}

		var apiResp messagesResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return nil, fmt.Errorf("decoding response for step %d: %w", stepCount, err)
		}

		// Update usage
		totalUsage.InputTokens += apiResp.Usage.InputTokens
		totalUsage.OutputTokens += apiResp.Usage.OutputTokens
		totalUsage.TotalTokens = totalUsage.InputTokens + totalUsage.OutputTokens

		// Process response content
		var textParts []string
		var toolCalls []core.ToolCall

		for _, block := range apiResp.Content {
			switch block.Type {
			case "text":
				textParts = append(textParts, block.Text)
			case "tool_use":
				// Convert tool use to core format
				inputJSON, err := json.Marshal(block.Input)
				if err != nil {
					return nil, fmt.Errorf("marshaling tool input: %w", err)
				}
				
				toolCalls = append(toolCalls, core.ToolCall{
					ID:    block.ID,
					Name:  block.Name,
					Input: json.RawMessage(inputJSON),
				})
			}
		}

		// Combine text parts
		var stepText string
		for i, part := range textParts {
			if i > 0 {
				stepText += "\n"
			}
			stepText += part
		}

		// Create step
		step := core.Step{
			Text:       stepText,
			ToolCalls:  toolCalls,
			StepNumber: stepCount,
			Timestamp:  time.Now(),
		}

		// Add assistant message to conversation
		if stepText != "" || len(toolCalls) > 0 {
			// Build content blocks for the assistant message
			var assistantContent []contentBlock
			
			if stepText != "" {
				assistantContent = append(assistantContent, NewTextContent(stepText))
			}
			
			for _, tc := range toolCalls {
				var input map[string]interface{}
				if err := json.Unmarshal(tc.Input, &input); err != nil {
					input = make(map[string]interface{})
				}
				assistantContent = append(assistantContent, NewToolUseContent(tc.ID, tc.Name, input))
			}

			assistantMessage := core.Message{
				Role: core.Assistant,
				Parts: []core.Part{core.Text{Text: stepText}},
			}
			messages = append(messages, assistantMessage)
		}

		// Execute tools if present
		if len(toolCalls) > 0 {
			toolResults, err := p.executeTools(ctx, req.Tools, toolCalls, messages)
			if err != nil {
				return nil, fmt.Errorf("executing tools for step %d: %w", stepCount, err)
			}
			step.ToolResults = toolResults

			// Add tool results to conversation
			// In Anthropic format, we add a user message with tool result content blocks
			var resultContent []contentBlock
			for _, result := range toolResults {
				var content interface{}
				if result.Error != "" {
					content = result.Error
				} else {
					content = result.Result
				}
				
				resultContent = append(resultContent, NewToolResultContent(result.ID, content, result.Error != ""))
			}

			// Convert to a user message (Anthropic expects tool results in user messages)
			if len(resultContent) > 0 {
				toolResultMessage := core.Message{
					Role:  core.User,
					Parts: []core.Part{core.Text{Text: p.formatToolResults(toolResults)}},
				}
				messages = append(messages, toolResultMessage)
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

// formatToolResults formats tool execution results for inclusion in messages.
func (p *Provider) formatToolResults(results []core.ToolExecution) string {
	var parts []string
	for _, result := range results {
		if result.Error != "" {
			parts = append(parts, fmt.Sprintf("Tool %s error: %s", result.Name, result.Error))
		} else {
			data, err := json.Marshal(result.Result)
			if err != nil {
				parts = append(parts, fmt.Sprintf("Tool %s result: %v", result.Name, result.Result))
			} else {
				parts = append(parts, fmt.Sprintf("Tool %s result: %s", result.Name, string(data)))
			}
		}
	}
	return fmt.Sprintf("Tool results:\n%s", joinParts(parts, "\n"))
}

// joinParts joins string parts with a separator.
func joinParts(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// GenerateObject generates a structured object conforming to the provided schema.
func (p *Provider) GenerateObject(ctx context.Context, req core.Request, schema any) (*core.ObjectResult[any], error) {
	model := p.getModel(req)

	// Convert ObjectResult to TextResult for observability compatibility
	textResult, err := obs.WithGenAIObservability(ctx, "anthropic", model, obs.GenAIOpObjectCompletion, req, func(ctx context.Context) (*core.TextResult, error) {
		objectResult, err := p.executeGenerateObject(ctx, req, schema)
		if err != nil {
			return nil, err
		}
		
		// Convert ObjectResult to TextResult for observability
		jsonBytes, _ := json.Marshal(objectResult.Value)
		return &core.TextResult{
			Text:  string(jsonBytes),
			Usage: objectResult.Usage,
			Raw:   objectResult.Raw,
		}, nil
	})

	if err != nil {
		return nil, err
	}

	// Convert back to ObjectResult
	var result interface{}
	if err := json.Unmarshal([]byte(textResult.Text), &result); err != nil {
		return nil, fmt.Errorf("parsing object result: %w", err)
	}

	return &core.ObjectResult[any]{
		Value: result,
		Usage: textResult.Usage,
		Raw:   textResult.Raw,
	}, nil
}

// executeGenerateObject handles the actual object generation logic (extracted for observability)
func (p *Provider) executeGenerateObject(ctx context.Context, req core.Request, schema any) (*core.ObjectResult[any], error) {
	// For Anthropic, we need to include instructions in the system prompt or user message
	// to produce JSON output, as they don't have a dedicated structured output mode like OpenAI
	
	// Convert schema to a description
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

	// Generate text with the modified request
	textResult, err := p.GenerateText(ctx, modifiedReq)
	if err != nil {
		return nil, err
	}

	// Parse the JSON response
	var value any
	if err := json.Unmarshal([]byte(textResult.Text), &value); err != nil {
		return nil, fmt.Errorf("parsing JSON response: %w", err)
	}

	// TODO: Validate against the provided schema
	// For now, we'll return the parsed value as-is

	result := &core.ObjectResult[any]{
		Value: value,
		Steps: textResult.Steps,
		Usage: textResult.Usage,
		Raw:   textResult.Raw,
	}

	return result, nil
}