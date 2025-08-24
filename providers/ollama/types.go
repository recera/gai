package ollama

import (
	"encoding/json"
	"time"
)

// chatRequest represents the request structure for Ollama's /api/chat endpoint.
type chatRequest struct {
	Model     string         `json:"model"`
	Messages  []chatMessage  `json:"messages"`
	Tools     []chatTool     `json:"tools,omitempty"`
	Format    string         `json:"format,omitempty"`
	Options   *modelOptions  `json:"options,omitempty"`
	Stream    *bool          `json:"stream,omitempty"`
	KeepAlive *string        `json:"keep_alive,omitempty"`
	Template  string         `json:"template,omitempty"`
}

// chatMessage represents a message in the chat conversation.
type chatMessage struct {
	Role       string          `json:"role"` // "system", "user", "assistant", "tool"
	Content    string          `json:"content"`
	Images     []string        `json:"images,omitempty"`     // Base64 encoded images
	ToolCalls  []toolCall      `json:"tool_calls,omitempty"` // For assistant messages
}

// chatTool represents a tool available to the model.
type chatTool struct {
	Type     string   `json:"type"`     // "function"
	Function function `json:"function"`
}

// function represents a function tool definition.
type function struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// toolCall represents a tool call made by the model.
type toolCall struct {
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type"`     // "function"
	Function functionCall `json:"function"`
}

// functionCall represents the function call details.
type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// modelOptions contains model-specific parameters.
type modelOptions struct {
	// Sampling parameters
	Temperature      *float32 `json:"temperature,omitempty"`
	TopK             *int     `json:"top_k,omitempty"`
	TopP             *float32 `json:"top_p,omitempty"`
	RepeatPenalty    *float32 `json:"repeat_penalty,omitempty"`
	Seed             *int     `json:"seed,omitempty"`
	
	// Generation parameters
	NumPredict       *int     `json:"num_predict,omitempty"`  // Max tokens to generate
	NumCtx           *int     `json:"num_ctx,omitempty"`      // Context size
	NumBatch         *int     `json:"num_batch,omitempty"`    // Batch size
	NumGQA           *int     `json:"num_gqa,omitempty"`      // Group query attention
	NumGPU           *int     `json:"num_gpu,omitempty"`      // GPU layers
	MainGPU          *int     `json:"main_gpu,omitempty"`     // Main GPU
	LowVRAM          *bool    `json:"low_vram,omitempty"`     // Low VRAM mode
	
	// Stop sequences
	Stop             []string `json:"stop,omitempty"`
	
	// Advanced parameters
	FrequencyPenalty *float32 `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float32 `json:"presence_penalty,omitempty"`
	Mirostat         *int     `json:"mirostat,omitempty"`
	MirostatEta      *float32 `json:"mirostat_eta,omitempty"`
	MirostatTau      *float32 `json:"mirostat_tau,omitempty"`
	PenalizeNewline  *bool    `json:"penalize_newline,omitempty"`
}

// chatResponse represents the response from Ollama's /api/chat endpoint.
type chatResponse struct {
	Model              string         `json:"model"`
	CreatedAt          time.Time      `json:"created_at"`
	Message            *chatMessage   `json:"message,omitempty"`
	Done               bool           `json:"done"`
	TotalDuration      int64          `json:"total_duration,omitempty"`
	LoadDuration       int64          `json:"load_duration,omitempty"`
	PromptEvalCount    int            `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64          `json:"prompt_eval_duration,omitempty"`
	EvalCount          int            `json:"eval_count,omitempty"`
	EvalDuration       int64          `json:"eval_duration,omitempty"`
	Context            []int          `json:"context,omitempty"`
}

// generateRequest represents the request structure for Ollama's /api/generate endpoint.
type generateRequest struct {
	Model     string        `json:"model"`
	Prompt    string        `json:"prompt"`
	Images    []string      `json:"images,omitempty"`  // Base64 encoded images
	Format    string        `json:"format,omitempty"`  // "json" or JSON schema
	Options   *modelOptions `json:"options,omitempty"`
	System    string        `json:"system,omitempty"`
	Template  string        `json:"template,omitempty"`
	Context   []int         `json:"context,omitempty"`
	Stream    *bool         `json:"stream,omitempty"`
	Raw       *bool         `json:"raw,omitempty"`
	KeepAlive *string       `json:"keep_alive,omitempty"`
}

// generateResponse represents the response from Ollama's /api/generate endpoint.
type generateResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	Context            []int     `json:"context,omitempty"`
	TotalDuration      int64     `json:"total_duration,omitempty"`
	LoadDuration       int64     `json:"load_duration,omitempty"`
	PromptEvalCount    int       `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64     `json:"prompt_eval_duration,omitempty"`
	EvalCount          int       `json:"eval_count,omitempty"`
	EvalDuration       int64     `json:"eval_duration,omitempty"`
}

// modelsResponse represents the response from the /api/tags endpoint.
type modelsResponse struct {
	Models []model `json:"models"`
}

// model represents a model available in Ollama.
type model struct {
	Name       string            `json:"name"`
	Size       int64             `json:"size"`
	Digest     string            `json:"digest"`
	ModifiedAt time.Time         `json:"modified_at"`
	Details    map[string]string `json:"details"`
}

// errorResponse represents an error response from Ollama.
type errorResponse struct {
	Error string `json:"error"`
}

// Helper functions for creating request objects

// NewChatRequest creates a new chat request with default settings.
func NewChatRequest(model string, messages []chatMessage) *chatRequest {
	stream := true
	keepAlive := "5m"
	return &chatRequest{
		Model:     model,
		Messages:  messages,
		Stream:    &stream,
		KeepAlive: &keepAlive,
	}
}

// NewGenerateRequest creates a new generate request with default settings.
func NewGenerateRequest(model, prompt string) *generateRequest {
	stream := true
	keepAlive := "5m"
	return &generateRequest{
		Model:     model,
		Prompt:    prompt,
		Stream:    &stream,
		KeepAlive: &keepAlive,
	}
}

// WithTemperature sets the temperature option.
func (r *chatRequest) WithTemperature(temp float32) *chatRequest {
	if r.Options == nil {
		r.Options = &modelOptions{}
	}
	r.Options.Temperature = &temp
	return r
}

// WithMaxTokens sets the max tokens option.
func (r *chatRequest) WithMaxTokens(maxTokens int) *chatRequest {
	if r.Options == nil {
		r.Options = &modelOptions{}
	}
	r.Options.NumPredict = &maxTokens
	return r
}

// WithTools sets the tools available to the model.
func (r *chatRequest) WithTools(tools []chatTool) *chatRequest {
	r.Tools = tools
	return r
}

// WithFormat sets the response format (e.g., "json" or JSON schema).
func (r *chatRequest) WithFormat(format string) *chatRequest {
	r.Format = format
	return r
}

// WithStream controls streaming behavior.
func (r *chatRequest) WithStream(stream bool) *chatRequest {
	r.Stream = &stream
	return r
}

// IsToolUse returns true if this message contains tool calls.
func (m *chatMessage) IsToolUse() bool {
	return len(m.ToolCalls) > 0
}

// IsComplete returns true if this response is complete (done=true).
func (r *chatResponse) IsComplete() bool {
	return r.Done
}

// HasContent returns true if this response has content.
func (r *chatResponse) HasContent() bool {
	return r.Message != nil && r.Message.Content != ""
}

// HasToolCalls returns true if this response contains tool calls.
func (r *chatResponse) HasToolCalls() bool {
	return r.Message != nil && len(r.Message.ToolCalls) > 0
}

// GetUsage calculates usage information from the response.
func (r *chatResponse) GetUsage() (promptTokens, completionTokens, totalTokens int) {
	promptTokens = r.PromptEvalCount
	completionTokens = r.EvalCount
	totalTokens = promptTokens + completionTokens
	return
}

// GetLatency calculates the total latency in milliseconds.
func (r *chatResponse) GetLatency() int64 {
	if r.TotalDuration > 0 {
		return r.TotalDuration / 1000000 // Convert nanoseconds to milliseconds
	}
	return 0
}