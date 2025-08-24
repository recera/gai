package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/recera/gai/core"
)

// runRequest executes a single or multi-step request.
func (p *Provider) runRequest(ctx context.Context, req core.Request) (*core.TextResult, error) {
	// If no tools, just generate once
	if len(req.Tools) == 0 {
		return p.generateOnce(ctx, req)
	}

	// Multi-step execution with tools
	messages := make([]core.Message, len(req.Messages))
	copy(messages, req.Messages)

	steps := []core.Step{}
	totalUsage := core.Usage{}
	
	for stepNum := 0; stepNum < 10; stepNum++ { // Max 10 steps to prevent infinite loops
		// Generate with current messages
		stepReq := req
		stepReq.Messages = messages
		
		resp, err := p.generateOnce(ctx, stepReq)
		if err != nil {
			return nil, err
		}

		// Accumulate usage
		totalUsage.InputTokens += resp.Usage.InputTokens
		totalUsage.OutputTokens += resp.Usage.OutputTokens
		totalUsage.TotalTokens += resp.Usage.TotalTokens

		// Check for tool calls
		toolCalls := extractToolCalls(resp.Raw)
		if len(toolCalls) == 0 {
			// No more tools, we're done
			steps = append(steps, core.Step{
				Text: resp.Text,
			})
			return &core.TextResult{
				Text:  resp.Text,
				Steps: steps,
				Usage: totalUsage,
				Raw:   resp.Raw,
			}, nil
		}

		// Execute tools
		toolResults := p.executeTools(ctx, toolCalls, req.Tools, messages)
		
		// Add step
		steps = append(steps, core.Step{
			Text:        resp.Text,
			ToolCalls:   toolCalls,
			ToolResults: toolResults,
		})

		// Add assistant message with tool calls
		assistantMsg := core.Message{
			Role: core.Assistant,
			Parts: []core.Part{
				core.Text{Text: resp.Text},
			},
		}
		messages = append(messages, assistantMsg)

		// Add tool results as messages
		for _, result := range toolResults {
			toolMsg := core.Message{
				Role: core.Tool,
				Parts: []core.Part{
					core.Text{Text: formatToolResult(result)},
				},
			}
			messages = append(messages, toolMsg)
		}

		// Check stop condition
		if req.StopWhen != nil && req.StopWhen.ShouldStop(stepNum+1, steps[len(steps)-1]) {
			break
		}
	}

	// Return accumulated result
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

// generateOnce performs a single generation request.
func (p *Provider) generateOnce(ctx context.Context, req core.Request) (*core.TextResult, error) {
	// Convert request to Gemini format
	geminiReq := p.convertRequest(req)

	// Marshal request
	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, err
	}

	// Make API request with retries
	var resp *GenerateContentResponse
	var lastErr error

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := p.retryDelay * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, lastErr = p.doRequest(ctx, body)
		if lastErr == nil {
			break
		}

		// Check if error is retryable
		if !isRetryable(lastErr) {
			break
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	// Convert response
	return p.convertResponse(resp), nil
}

// doRequest performs the actual HTTP request.
func (p *Provider) doRequest(ctx context.Context, body []byte) (*GenerateContentResponse, error) {
	model := p.model
	if model == "" {
		model = "gemini-1.5-flash"
	}

	url := fmt.Sprintf("%s/%s/models/%s:generateContent?key=%s",
		p.baseURL, apiVersion, model, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
		}
		return nil, mapError(&errResp, resp.StatusCode)
	}

	var geminiResp GenerateContentResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, err
	}

	return &geminiResp, nil
}

// convertRequest converts a GAI request to Gemini format.
func (p *Provider) convertRequest(req core.Request) *GenerateContentRequest {
	geminiReq := &GenerateContentRequest{
		Contents: []Content{},
	}

	// Handle system instruction separately
	var systemParts []Part
	
	// Convert messages
	for _, msg := range req.Messages {
		if msg.Role == core.System {
			// Add to system instruction
			for _, part := range msg.Parts {
				systemParts = append(systemParts, p.convertPart(part))
			}
		} else {
			content := Content{
				Role:  convertRole(msg.Role),
				Parts: []Part{},
			}
			
			for _, part := range msg.Parts {
				content.Parts = append(content.Parts, p.convertPart(part))
			}
			
			geminiReq.Contents = append(geminiReq.Contents, content)
		}
	}

	// Set system instruction if present
	if len(systemParts) > 0 {
		geminiReq.SystemInstruction = &Content{
			Role:  "user", // System instructions use "user" role
			Parts: systemParts,
		}
	}

	// Add generation config
	geminiReq.GenerationConfig = &GenerationConfig{}
	if req.Temperature > 0 {
		temp := float32(req.Temperature)
		geminiReq.GenerationConfig.Temperature = &temp
	}
	if req.MaxTokens > 0 {
		maxTokens := int32(req.MaxTokens)
		geminiReq.GenerationConfig.MaxOutputTokens = &maxTokens
	}

	// Add safety settings
	if req.Safety != nil || p.defaultSafety != nil {
		safety := req.Safety
		if safety == nil {
			safety = p.defaultSafety
		}
		
		geminiReq.SafetySettings = []SafetySetting{}
		
		if safety.Harassment != "" {
			geminiReq.SafetySettings = append(geminiReq.SafetySettings, SafetySetting{
				Category:  "HARM_CATEGORY_HARASSMENT",
				Threshold: convertSafetyLevel(safety.Harassment),
			})
		}
		if safety.Hate != "" {
			geminiReq.SafetySettings = append(geminiReq.SafetySettings, SafetySetting{
				Category:  "HARM_CATEGORY_HATE_SPEECH",
				Threshold: convertSafetyLevel(safety.Hate),
			})
		}
		if safety.Sexual != "" {
			geminiReq.SafetySettings = append(geminiReq.SafetySettings, SafetySetting{
				Category:  "HARM_CATEGORY_SEXUALLY_EXPLICIT",
				Threshold: convertSafetyLevel(safety.Sexual),
			})
		}
		if safety.Dangerous != "" {
			geminiReq.SafetySettings = append(geminiReq.SafetySettings, SafetySetting{
				Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
				Threshold: convertSafetyLevel(safety.Dangerous),
			})
		}
	}

	// Add tools
	if len(req.Tools) > 0 {
		geminiReq.Tools = []Tool{
			{
				FunctionDeclarations: p.convertTools(req.Tools),
			},
		}
		
		if req.ToolChoice != core.ToolNone {
			geminiReq.ToolConfig = &ToolConfig{
				FunctionCallingConfig: &FunctionCallingConfig{
					Mode: convertToolChoice(req.ToolChoice),
				},
			}
		}
	}

	// Handle response schema for structured outputs
	if schema, ok := req.ProviderOptions["response_schema"]; ok {
		schemaJSON, _ := json.Marshal(schema)
		geminiReq.GenerationConfig.ResponseSchema = schemaJSON
		geminiReq.GenerationConfig.ResponseMIMEType = "application/json"
	}

	return geminiReq
}

// convertPart converts a GAI part to Gemini format.
func (prov *Provider) convertPart(part core.Part) Part {
	switch p := part.(type) {
	case core.Text:
		return Part{Text: p.Text}
	case core.ImageURL:
		// Download and inline the image
		if resp, err := http.Get(p.URL); err == nil {
			defer resp.Body.Close()
			if data, err := io.ReadAll(resp.Body); err == nil {
				return Part{
					InlineData: &InlineData{
						MIMEType: resp.Header.Get("Content-Type"),
						Data:     base64.StdEncoding.EncodeToString(data),
					},
				}
			}
		}
		return Part{Text: fmt.Sprintf("[Image: %s]", p.URL)}
	case core.Audio:
		if p.Source.Kind == core.BlobProviderFile {
			if info, ok := prov.fileStore.Get(p.Source.FileID); ok {
				return Part{
					FileData: &FileData{
						MIMEType: info.MIMEType,
						FileURI:  info.URI,
					},
				}
			}
		}
		return Part{Text: "[Audio content]"}
	case core.Video:
		if p.Source.Kind == core.BlobProviderFile {
			if info, ok := prov.fileStore.Get(p.Source.FileID); ok {
				return Part{
					FileData: &FileData{
						MIMEType: info.MIMEType,
						FileURI:  info.URI,
					},
				}
			}
		}
		return Part{Text: "[Video content]"}
	case core.File:
		if p.Source.Kind == core.BlobProviderFile {
			if info, ok := prov.fileStore.Get(p.Source.FileID); ok {
				return Part{
					FileData: &FileData{
						MIMEType: info.MIMEType,
						FileURI:  info.URI,
					},
				}
			}
		}
		return Part{Text: fmt.Sprintf("[File: %s]", p.Name)}
	default:
		return Part{Text: "[Unknown content]"}
	}
}

// convertTools converts GAI tools to Gemini function declarations.
func (p *Provider) convertTools(tools []core.ToolHandle) []FunctionDeclaration {
	funcs := make([]FunctionDeclaration, 0, len(tools))
	
	for _, tool := range tools {
		funcs = append(funcs, FunctionDeclaration{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.InSchemaJSON(),
		})
	}
	
	return funcs
}

// convertResponse converts a Gemini response to GAI format.
func (p *Provider) convertResponse(resp *GenerateContentResponse) *core.TextResult {
	if len(resp.Candidates) == 0 {
		return &core.TextResult{
			Text:  "",
			Usage: core.Usage{},
		}
	}

	candidate := resp.Candidates[0]
	
	// Extract text
	var text strings.Builder
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			text.WriteString(part.Text)
		}
	}

	// Convert usage
	usage := core.Usage{}
	if resp.UsageMetadata != nil {
		usage.InputTokens = resp.UsageMetadata.PromptTokenCount
		usage.OutputTokens = resp.UsageMetadata.CandidatesTokenCount
		usage.TotalTokens = resp.UsageMetadata.TotalTokenCount
	}

	return &core.TextResult{
		Text:  text.String(),
		Usage: usage,
		Raw:   resp,
	}
}

// extractToolCalls extracts tool calls from a response.
func extractToolCalls(raw any) []core.ToolCall {
	resp, ok := raw.(*GenerateContentResponse)
	if !ok || len(resp.Candidates) == 0 {
		return nil
	}

	var calls []core.ToolCall
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.FunctionCall != nil {
			calls = append(calls, core.ToolCall{
				Name:  part.FunctionCall.Name,
				Input: part.FunctionCall.Args,
			})
		}
	}
	
	return calls
}

// executeTools runs tools in parallel.
func (p *Provider) executeTools(ctx context.Context, calls []core.ToolCall, handles []core.ToolHandle, messages []core.Message) []core.ToolExecution {
	results := make([]core.ToolExecution, len(calls))
	
	// Find and execute each tool
	for i, call := range calls {
		var handle core.ToolHandle
		for _, h := range handles {
			if h.Name() == call.Name {
				handle = h
				break
			}
		}
		
		if handle == nil {
			results[i] = core.ToolExecution{
				Name:   call.Name,
				Result: map[string]string{"error": "tool not found"},
			}
			continue
		}

		// Execute tool
		result, err := handle.Exec(ctx, call.Input, nil)
		if err != nil {
			results[i] = core.ToolExecution{
				Name:   call.Name,
				Result: map[string]string{"error": err.Error()},
			}
		} else {
			results[i] = core.ToolExecution{
				Name:   call.Name,
				Result: result,
			}
		}
	}
	
	return results
}

// formatToolResult formats a tool result as text.
func formatToolResult(exec core.ToolExecution) string {
	data, _ := json.Marshal(exec.Result)
	return string(data)
}

// isRetryable checks if an error should be retried.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for specific error types
	if aiErr, ok := err.(*core.AIError); ok {
		return aiErr.Temporary
	}
	
	// Check for network errors
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "EOF")
}