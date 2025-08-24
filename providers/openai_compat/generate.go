package openai_compat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/recera/gai/core"
)

// GenerateText implements the core.Provider interface for text generation.
// It supports multi-step tool execution when tools are provided.
func (p *Provider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
	// Use metrics collector if available
	if p.config.MetricsCollector != nil {
		defer func(start int64) {
			// Record metrics
		}(0) // Placeholder for timing
	}
	
	// If tools are provided and multi-step execution is needed, use runner
	if len(req.Tools) > 0 && req.StopWhen != nil {
		return p.generateWithTools(ctx, req)
	}
	
	// Simple single-shot generation
	apiReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("converting request: %w", err)
	}
	
	// Strip unsupported parameters
	apiReq = p.stripUnsupportedParams(apiReq)
	
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", apiReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, MapError(resp, p.config.ProviderName)
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
				Text:      result.Text,
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
		// Only include tools in the first request; after tools are executed, 
		// we send messages without tools to get the final response
		var toolsToSend []core.ToolHandle
		var toolChoiceToSend core.ToolChoice
		if stepCount == 0 {
			toolsToSend = req.Tools
			toolChoiceToSend = req.ToolChoice
		}
		
		apiReq, err := p.convertRequest(core.Request{
			Model:           req.Model,
			Messages:        messages,
			Temperature:     req.Temperature,
			MaxTokens:       req.MaxTokens,
			Tools:           toolsToSend,
			ToolChoice:      toolChoiceToSend,
			ProviderOptions: req.ProviderOptions,
		})
		if err != nil {
			return nil, fmt.Errorf("converting request for step %d: %w", stepCount, err)
		}
		
		// Strip unsupported parameters
		apiReq = p.stripUnsupportedParams(apiReq)
		
		resp, err := p.doRequest(ctx, "POST", "/chat/completions", apiReq)
		if err != nil {
			return nil, fmt.Errorf("API request for step %d: %w", stepCount, err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			return nil, MapError(resp, p.config.ProviderName)
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
		
		// Check for tool calls
		if len(choice.Message.ToolCalls) == 0 {
			// No tools called, this is the final response
			steps = append(steps, core.Step{Text: text})
			break
		}
		
		// Convert tool calls
		toolCalls := p.convertToolCallsFromAPI(choice.Message.ToolCalls)
		
		// Execute tools
		toolResults := make([]core.ToolExecution, len(toolCalls))
		for i, tc := range toolCalls {
			// Find the tool
			var tool core.ToolHandle
			for _, t := range req.Tools {
				if t.Name() == tc.Name {
					tool = t
					break
				}
			}
			
			if tool == nil {
				toolResults[i] = core.ToolExecution{
					Name:   tc.Name,
					Result: map[string]string{"error": "tool not found"},
				}
				continue
			}
			
			// Execute the tool
			result, err := tool.Exec(ctx, tc.Input, map[string]interface{}{
				"call_id": tc.ID,
				"messages": messages,
			})
			if err != nil {
				toolResults[i] = core.ToolExecution{
					Name:   tc.Name,
					Result: map[string]string{"error": err.Error()},
				}
			} else {
				toolResults[i] = core.ToolExecution{
					Name:   tc.Name,
					Result: result,
				}
			}
		}
		
		// Add step
		step := core.Step{
			Text:        text,
			ToolCalls:   toolCalls,
			ToolResults: toolResults,
		}
		steps = append(steps, step)
		
		// Add assistant message with tool calls
		// When there are tool calls, the assistant message should indicate that
		if len(toolCalls) > 0 {
			// Add a message indicating tool calls were made
			messages = append(messages, core.Message{
				Role: core.Assistant,
				Parts: []core.Part{
					core.Text{Text: ""}, // Empty text when calling tools
				},
			})
		} else if text != "" {
			// Add assistant message with text if there is any
			messages = append(messages, core.Message{
				Role: core.Assistant,
				Parts: []core.Part{
					core.Text{Text: text},
				},
			})
		}
		
		// Add tool results as messages
		for i, tr := range toolResults {
			resultJSON, _ := json.Marshal(tr.Result)
			messages = append(messages, core.Message{
				Role: core.Tool,
				Parts: []core.Part{
					core.Text{Text: string(resultJSON)},
				},
				Name: toolCalls[i].ID, // Use tool call ID as message name
			})
		}
		
		// Check stop condition
		if req.StopWhen != nil {
			if req.StopWhen.ShouldStop(len(steps), step) {
				break
			}
		}
		
		stepCount++
	}
	
	// Build final result
	// Collect all text from steps
	var finalText strings.Builder
	for _, step := range steps {
		if step.Text != "" {
			finalText.WriteString(step.Text)
			finalText.WriteString(" ")
		}
	}
	
	return &core.TextResult{
		Text:  strings.TrimSpace(finalText.String()),
		Steps: steps,
		Usage: totalUsage,
	}, nil
}

// GenerateObject generates a structured object output.
func (p *Provider) GenerateObject(ctx context.Context, req core.Request, schema interface{}) (*core.ObjectResult[any], error) {
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
	
	// Strip unsupported parameters
	apiReq = p.stripUnsupportedParams(apiReq)
	
	// Make API request
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", apiReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, MapError(resp, p.config.ProviderName)
	}
	
	var apiResp chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	
	// Extract JSON content
	var jsonContent string
	if len(apiResp.Choices) > 0 {
		switch content := apiResp.Choices[0].Message.Content.(type) {
		case string:
			jsonContent = content
		}
	}
	
	if jsonContent == "" {
		return nil, fmt.Errorf("no JSON content in response")
	}
	
	// Parse and validate JSON
	result := reflect.New(reflect.TypeOf(schema)).Interface()
	if err := json.Unmarshal([]byte(jsonContent), result); err != nil {
		return nil, fmt.Errorf("parsing JSON response: %w", err)
	}
	
	return &core.ObjectResult[any]{
		Value: result,
		Usage: core.Usage{
			InputTokens:  apiResp.Usage.PromptTokens,
			OutputTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:  apiResp.Usage.TotalTokens,
		},
		Raw: apiResp,
	}, nil
}

// convertRequest converts a core.Request to an API request.
func (p *Provider) convertRequest(req core.Request) (*chatCompletionRequest, error) {
	apiReq := &chatCompletionRequest{
		Model: p.getModel(req),
		N:     1,
	}
	
	// Handle optional fields
	if req.Temperature > 0 {
		apiReq.Temperature = &req.Temperature
	}
	if req.MaxTokens > 0 {
		apiReq.MaxTokens = &req.MaxTokens
	}
	
	// Convert messages
	messages, err := p.convertMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("converting messages: %w", err)
	}
	apiReq.Messages = messages
	
	// Handle tools
	if len(req.Tools) > 0 && !p.config.DisableToolChoice {
		tools := make([]chatTool, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = chatTool{
				Type: "function",
				Function: toolFunction{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters:  t.InSchemaJSON(),
				},
			}
		}
		apiReq.Tools = tools
		
		// Set tool choice
		switch req.ToolChoice {
		case core.ToolNone:
			apiReq.ToolChoice = "none"
		case core.ToolAuto:
			apiReq.ToolChoice = "auto"
		case core.ToolRequired:
			apiReq.ToolChoice = "required"
		default:
			apiReq.ToolChoice = "auto"
		}
	}
	
	// Handle parallel tool calls
	if !p.config.DisableParallelToolCalls && len(req.Tools) > 0 {
		parallel := true
		apiReq.ParallelToolCalls = &parallel
	}
	
	// Apply provider options
	if req.ProviderOptions != nil {
		if v, ok := req.ProviderOptions["top_p"].(float32); ok {
			apiReq.TopP = &v
		}
		if v, ok := req.ProviderOptions["presence_penalty"].(float32); ok {
			apiReq.PresencePenalty = &v
		}
		if v, ok := req.ProviderOptions["frequency_penalty"].(float32); ok {
			apiReq.FrequencyPenalty = &v
		}
		if v, ok := req.ProviderOptions["seed"].(int); ok {
			apiReq.Seed = &v
		}
		if v, ok := req.ProviderOptions["stop"].([]string); ok {
			apiReq.Stop = v
		}
		if v, ok := req.ProviderOptions["user"].(string); ok {
			apiReq.User = v
		}
	}
	
	return apiReq, nil
}

// convertMessages converts core messages to API messages.
func (p *Provider) convertMessages(messages []core.Message) ([]chatMessage, error) {
	result := make([]chatMessage, 0, len(messages))
	
	for _, msg := range messages {
		apiMsg := chatMessage{
			Role: string(msg.Role),
			Name: msg.Name,
		}
		
		// Convert parts to content
		if len(msg.Parts) == 1 {
			// Single part - use simple format
			if text, ok := msg.Parts[0].(core.Text); ok {
				apiMsg.Content = text.Text
			} else {
				// Convert to content parts
				parts, err := p.convertParts(msg.Parts)
				if err != nil {
					return nil, fmt.Errorf("converting parts: %w", err)
				}
				apiMsg.Content = parts
			}
		} else if len(msg.Parts) > 1 {
			// Multiple parts - use array format
			parts, err := p.convertParts(msg.Parts)
			if err != nil {
				return nil, fmt.Errorf("converting parts: %w", err)
			}
			apiMsg.Content = parts
		}
		
		result = append(result, apiMsg)
	}
	
	return result, nil
}

// convertParts converts message parts to content parts.
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
					URL: p.URL,
				},
			})
		default:
			// Skip unsupported part types for now
			// Could log a warning here
		}
	}
	
	return result, nil
}

// convertToolCallsFromAPI converts API tool calls to core tool calls.
func (p *Provider) convertToolCallsFromAPI(apiCalls []toolCall) []core.ToolCall {
	result := make([]core.ToolCall, len(apiCalls))
	for i, call := range apiCalls {
		result[i] = core.ToolCall{
			ID:    call.ID,
			Name:  call.Function.Name,
			Input: json.RawMessage(call.Function.Arguments),
		}
	}
	return result
}

// stripUnsupportedParams removes parameters that the provider doesn't support.
func (p *Provider) stripUnsupportedParams(req *chatCompletionRequest) *chatCompletionRequest {
	if p.config.UnsupportedParams == nil {
		return req
	}
	
	// Create a copy to avoid modifying the original
	stripped := *req
	
	for _, param := range p.config.UnsupportedParams {
		switch param {
		case "tool_choice":
			stripped.ToolChoice = nil
		case "tools":
			stripped.Tools = nil
		case "parallel_tool_calls":
			stripped.ParallelToolCalls = nil
		case "response_format":
			stripped.ResponseFormat = nil
		case "stream_options":
			stripped.StreamOptions = nil
		case "logit_bias":
			stripped.LogitBias = nil
		case "seed":
			stripped.Seed = nil
		case "top_p":
			stripped.TopP = nil
		case "presence_penalty":
			stripped.PresencePenalty = nil
		case "frequency_penalty":
			stripped.FrequencyPenalty = nil
		}
	}
	
	return &stripped
}

// generateJSONSchema generates a JSON schema for a Go type.
func (p *Provider) generateJSONSchema(v interface{}) (json.RawMessage, error) {
	// This is a simplified implementation
	// In production, you'd want to use a proper schema generation library
	schema := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
	
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct type, got %v", t.Kind())
	}
	
	props := schema["properties"].(map[string]interface{})
	required := []string{}
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		
		// Parse JSON tag
		fieldName := jsonTag
		if idx := len(jsonTag); idx > 0 {
			if comma := findComma(jsonTag); comma != -1 {
				fieldName = jsonTag[:comma]
				if !hasOmitempty(jsonTag[comma+1:]) {
					required = append(required, fieldName)
				}
			} else {
				required = append(required, fieldName)
			}
		}
		
		// Determine field type
		props[fieldName] = getJSONType(field.Type)
	}
	
	schema["required"] = required
	return json.Marshal(schema)
}

// Helper functions for JSON schema generation
func findComma(s string) int {
	for i, r := range s {
		if r == ',' {
			return i
		}
	}
	return -1
}

func hasOmitempty(s string) bool {
	return len(s) >= 9 && s[:9] == "omitempty"
}

func getJSONType(t reflect.Type) map[string]interface{} {
	switch t.Kind() {
	case reflect.String:
		return map[string]interface{}{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]interface{}{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number"}
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.Slice:
		return map[string]interface{}{
			"type":  "array",
			"items": getJSONType(t.Elem()),
		}
	default:
		return map[string]interface{}{"type": "object"}
	}
}