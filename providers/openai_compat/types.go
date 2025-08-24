package openai_compat

import (
	"encoding/json"
)

// API Request Types

// chatCompletionRequest represents the OpenAI-compatible chat completion request.
type chatCompletionRequest struct {
	Model             string                   `json:"model"`
	Messages          []chatMessage            `json:"messages"`
	Temperature       *float32                 `json:"temperature,omitempty"`
	MaxTokens         *int                     `json:"max_tokens,omitempty"`
	Tools             []chatTool               `json:"tools,omitempty"`
	ToolChoice        interface{}              `json:"tool_choice,omitempty"`
	Stream            bool                     `json:"stream,omitempty"`
	ResponseFormat    *responseFormat          `json:"response_format,omitempty"`
	StreamOptions     *streamOptions           `json:"stream_options,omitempty"`
	ParallelToolCalls *bool                    `json:"parallel_tool_calls,omitempty"`
	N                 int                      `json:"n,omitempty"`
	Stop              []string                 `json:"stop,omitempty"`
	PresencePenalty   *float32                 `json:"presence_penalty,omitempty"`
	FrequencyPenalty  *float32                 `json:"frequency_penalty,omitempty"`
	LogitBias         map[string]float32       `json:"logit_bias,omitempty"`
	User              string                   `json:"user,omitempty"`
	Seed              *int                     `json:"seed,omitempty"`
	TopP              *float32                 `json:"top_p,omitempty"`
}

// chatMessage represents a message in the conversation.
type chatMessage struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"` // string or []contentPart
	Name       string      `json:"name,omitempty"`
	ToolCalls  []toolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// contentPart represents a part of multimodal content.
type contentPart struct {
	Type     string       `json:"type"` // "text" or "image_url"
	Text     string       `json:"text,omitempty"`
	ImageURL *imageURLPart `json:"image_url,omitempty"`
}

// imageURLPart represents an image URL in content.
type imageURLPart struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "low", "high", or "auto"
}

// chatTool represents a tool/function definition.
type chatTool struct {
	Type     string       `json:"type"` // "function"
	Function toolFunction `json:"function"`
}

// toolFunction represents a function that can be called.
type toolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// toolCall represents a tool call in a message.
type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function functionCall `json:"function"`
}

// functionCall represents a function call.
type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// responseFormat specifies the format of the response.
type responseFormat struct {
	Type       string            `json:"type"` // "text", "json_object", or "json_schema"
	JSONSchema *jsonSchemaFormat `json:"json_schema,omitempty"`
}

// jsonSchemaFormat represents JSON schema configuration.
type jsonSchemaFormat struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict,omitempty"`
}

// streamOptions configures streaming behavior.
type streamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// API Response Types

// chatCompletionResponse represents the chat completion response.
type chatCompletionResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []choice  `json:"choices"`
	Usage   usage     `json:"usage,omitempty"`
}

// choice represents a completion choice.
type choice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
	LogProbs     interface{} `json:"logprobs"`
}

// usage represents token usage information.
type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Streaming Response Types

// streamChunk represents a streaming response chunk.
type streamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []streamChoice `json:"choices"`
	Usage   *usage         `json:"usage,omitempty"`
}

// streamChoice represents a choice in a streaming response.
type streamChoice struct {
	Index        int          `json:"index"`
	Delta        *deltaMessage `json:"delta,omitempty"`
	FinishReason *string      `json:"finish_reason,omitempty"`
	LogProbs     interface{}  `json:"logprobs"`
}

// deltaMessage represents incremental message content in streaming.
type deltaMessage struct {
	Role      string      `json:"role,omitempty"`
	Content   interface{} `json:"content,omitempty"` // string or null
	ToolCalls []toolCall  `json:"tool_calls,omitempty"`
}

// Error Response Types

// errorResponse represents an error response from the API.
type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code,omitempty"`
		Param   string `json:"param,omitempty"`
	} `json:"error"`
}

// Models Response Types

// modelsResponse represents the response from the models endpoint.
type modelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}