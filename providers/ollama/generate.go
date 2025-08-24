package ollama

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	chatReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
	}

	// Disable streaming for single-shot
	chatReq = chatReq.WithStream(false)

	resp, err := p.doRequest(ctx, "POST", "/api/chat", chatReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Convert to core.TextResult
	promptTokens, completionTokens, totalTokens := chatResp.GetUsage()
	result := &core.TextResult{
		Usage: core.Usage{
			InputTokens:  promptTokens,
			OutputTokens: completionTokens,
			TotalTokens:  totalTokens,
		},
		Raw: chatResp,
	}

	if chatResp.Message != nil {
		result.Text = chatResp.Message.Content

		// Handle tool calls if present
		if len(chatResp.Message.ToolCalls) > 0 {
			step := core.Step{
				Text:      result.Text,
				ToolCalls: p.convertToolCallsFromAPI(chatResp.Message.ToolCalls),
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
		chatReq, err := p.convertRequest(core.Request{
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

		// Disable streaming for multi-step
		chatReq = chatReq.WithStream(false)

		resp, err := p.doRequest(ctx, "POST", "/api/chat", chatReq)
		if err != nil {
			return nil, fmt.Errorf("API request for step %d: %w", stepCount, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, p.parseError(resp)
		}

		var chatResp chatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			return nil, fmt.Errorf("decoding response for step %d: %w", stepCount, err)
		}

		// Update usage
		promptTokens, completionTokens, _ := chatResp.GetUsage()
		totalUsage.InputTokens += promptTokens
		totalUsage.OutputTokens += completionTokens
		totalUsage.TotalTokens += promptTokens + completionTokens

		if chatResp.Message == nil {
			break
		}

		text := chatResp.Message.Content

		// Create step
		step := core.Step{
			Text:       text,
			StepNumber: stepCount,
		}

		// Add assistant message to conversation
		messages = append(messages, core.Message{
			Role:  core.Assistant,
			Parts: []core.Part{core.Text{Text: text}},
		})

		// Handle tool calls
		if len(chatResp.Message.ToolCalls) > 0 {
			step.ToolCalls = p.convertToolCallsFromAPI(chatResp.Message.ToolCalls)

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

// convertToolCallsFromAPI converts Ollama tool calls to core format.
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

	// Prepare request with format
	chatReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
	}

	// Set format for structured output
	chatReq = chatReq.WithFormat(string(schemaBytes)).WithStream(false)

	// Make API request
	resp, err := p.doRequest(ctx, "POST", "/api/chat", chatReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Extract and parse the JSON response
	if chatResp.Message == nil {
		return nil, fmt.Errorf("no message in response")
	}

	content := chatResp.Message.Content
	
	// Parse the JSON content
	var value any
	if err := json.Unmarshal([]byte(content), &value); err != nil {
		return nil, fmt.Errorf("parsing JSON response: %w", err)
	}

	promptTokens, completionTokens, totalTokens := chatResp.GetUsage()
	result := &core.ObjectResult[any]{
		Value: value,
		Usage: core.Usage{
			InputTokens:  promptTokens,
			OutputTokens: completionTokens,
			TotalTokens:  totalTokens,
		},
		Raw: chatResp,
	}

	return result, nil
}

// generateUsingGenerateAPI uses Ollama's /api/generate endpoint for simple completions.
// This is useful for models that don't support the chat format well.
func (p *Provider) generateUsingGenerateAPI(ctx context.Context, req core.Request) (*core.TextResult, error) {
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
	
	// Disable streaming for simple generation
	stream := false
	genReq.Stream = &stream

	resp, err := p.doRequest(ctx, "POST", "/api/generate", genReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	// For non-streaming, read all chunks until done=true
	var fullResponse strings.Builder
	var totalUsage core.Usage
	
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Bytes()
		
		var genResp generateResponse
		if err := json.Unmarshal(line, &genResp); err != nil {
			continue // Skip malformed lines
		}
		
		fullResponse.WriteString(genResp.Response)
		
		// Update usage on final response
		if genResp.Done {
			totalUsage.InputTokens = genResp.PromptEvalCount
			totalUsage.OutputTokens = genResp.EvalCount
			totalUsage.TotalTokens = totalUsage.InputTokens + totalUsage.OutputTokens
			break
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return &core.TextResult{
		Text:  fullResponse.String(),
		Usage: totalUsage,
		Raw:   nil, // Could store the last generateResponse if needed
	}, nil
}

// buildPromptFromMessages builds a single prompt string from messages.
func (p *Provider) buildPromptFromMessages(messages []core.Message) string {
	var prompt strings.Builder
	
	for _, msg := range messages {
		switch msg.Role {
		case core.System:
			// System messages are handled separately in generate API
			continue
		case core.User:
			prompt.WriteString("User: ")
		case core.Assistant:
			prompt.WriteString("Assistant: ")
		case core.Tool:
			prompt.WriteString("Tool: ")
		}
		
		for _, part := range msg.Parts {
			if text, ok := part.(core.Text); ok {
				prompt.WriteString(text.Text)
			}
		}
		prompt.WriteString("\n\n")
	}
	
	return prompt.String()
}

// applyGenerateOptions applies Ollama-specific options to generate requests.
func (p *Provider) applyGenerateOptions(req *generateRequest, opts map[string]interface{}) {
	if req.Options == nil {
		req.Options = &modelOptions{}
	}

	// Same as chat options
	if v, ok := opts["top_k"].(int); ok {
		req.Options.TopK = &v
	}
	if v, ok := opts["top_p"].(float32); ok {
		req.Options.TopP = &v
	}
	if v, ok := opts["repeat_penalty"].(float32); ok {
		req.Options.RepeatPenalty = &v
	}
	if v, ok := opts["seed"].(int); ok {
		req.Options.Seed = &v
	}
	if v, ok := opts["num_ctx"].(int); ok {
		req.Options.NumCtx = &v
	}
	if v, ok := opts["template"].(string); ok {
		req.Template = v
	}
	if v, ok := opts["raw"].(bool); ok {
		req.Raw = &v
	}
}