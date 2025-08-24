package anthropic

import (
	"encoding/json"
)

// messagesRequest represents the request structure for Anthropic's Messages API.
type messagesRequest struct {
	Model         string      `json:"model"`
	MaxTokens     int         `json:"max_tokens"`
	Messages      []message   `json:"messages"`
	System        string      `json:"system,omitempty"`
	Temperature   *float32    `json:"temperature,omitempty"`
	TopP          *float32    `json:"top_p,omitempty"`
	TopK          *int        `json:"top_k,omitempty"`
	StopSequences []string    `json:"stop_sequences,omitempty"`
	Tools         []tool      `json:"tools,omitempty"`
	Stream        bool        `json:"stream,omitempty"`
}

// message represents a message in the conversation.
type message struct {
	Role    string      `json:"role"` // "user" or "assistant"
	Content interface{} `json:"content"` // string or []contentBlock
}

// contentBlock represents a block of content within a message.
type contentBlock struct {
	Type string `json:"type"` // "text", "image", "tool_use", "tool_result"

	// Text content
	Text string `json:"text,omitempty"`

	// Image content
	Source *imageSource `json:"source,omitempty"`

	// Tool use content
	ID    string                 `json:"id,omitempty"`    // For tool_use and tool_result
	Name  string                 `json:"name,omitempty"`  // For tool_use
	Input map[string]interface{} `json:"input,omitempty"` // For tool_use

	// Tool result content
	Content interface{} `json:"content,omitempty"` // For tool_result - can be string or content blocks
	IsError bool        `json:"is_error,omitempty"` // For tool_result
}

// imageSource represents an image source in Anthropic format.
type imageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/jpeg", "image/png", etc.
	Data      string `json:"data"`       // Base64 encoded image data
}

// tool represents a tool definition in Anthropic format.
type tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// messagesResponse represents the response structure from Anthropic's Messages API.
type messagesResponse struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"` // "message"
	Role         string        `json:"role"` // "assistant"
	Content      []contentBlock `json:"content"`
	Model        string        `json:"model"`
	StopReason   string        `json:"stop_reason"` // "end_turn", "max_tokens", "stop_sequence", "tool_use"
	StopSequence string        `json:"stop_sequence,omitempty"`
	Usage        usage         `json:"usage"`
}

// usage represents token usage information in Anthropic format.
type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Streaming types

// streamEvent represents a streaming event from Anthropic.
type streamEvent struct {
	Type string `json:"type"`
	// Event-specific data is embedded based on type
	*messageStartEvent
	*messageStopEvent
	*messageDeltaEvent
	*contentBlockStartEvent
	*contentBlockStopEvent
	*contentBlockDeltaEvent
	*pingEvent
	*errorEvent
}

// messageStartEvent represents the start of a message stream.
type messageStartEvent struct {
	Message *messagesResponse `json:"message,omitempty"`
}

// messageStopEvent represents the end of a message stream.
type messageStopEvent struct {
	// No additional fields
}

// messageDeltaEvent represents changes to top-level message properties.
type messageDeltaEvent struct {
	Delta *messageDelta `json:"delta,omitempty"`
	Usage *usage        `json:"usage,omitempty"`
}

// messageDelta represents incremental changes to message properties.
type messageDelta struct {
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}

// contentBlockStartEvent represents the start of a content block.
type contentBlockStartEvent struct {
	Index        int           `json:"index"`
	ContentBlock *contentBlock `json:"content_block,omitempty"`
}

// contentBlockStopEvent represents the end of a content block.
type contentBlockStopEvent struct {
	Index int `json:"index"`
}

// contentBlockDeltaEvent represents incremental changes to a content block.
type contentBlockDeltaEvent struct {
	Index int                   `json:"index"`
	Delta *contentBlockDelta    `json:"delta,omitempty"`
}

// contentBlockDelta represents incremental changes to content block data.
type contentBlockDelta struct {
	Type        string                 `json:"type,omitempty"`
	Text        string                 `json:"text,omitempty"`        // For text deltas
	Input       map[string]interface{} `json:"input,omitempty"`       // For tool_use input deltas
	PartialJSON string                 `json:"partial_json,omitempty"` // For partial JSON in tool inputs
}

// pingEvent represents a ping event to keep the connection alive.
type pingEvent struct {
	// No additional fields
}

// errorEvent represents an error during streaming.
type errorEvent struct {
	Error *apiError `json:"error,omitempty"`
}

// apiError represents an error response from the Anthropic API.
type apiError struct {
	Type    string `json:"type"`    // Error type
	Message string `json:"message"` // Human-readable error message
}

// Custom unmarshal for streamEvent to handle different event types
func (e *streamEvent) UnmarshalJSON(data []byte) error {
	// First, parse just the type field
	var typeOnly struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typeOnly); err != nil {
		return err
	}
	
	e.Type = typeOnly.Type
	
	// Then parse the specific event data based on type
	switch e.Type {
	case "message_start":
		e.messageStartEvent = &messageStartEvent{}
		return json.Unmarshal(data, e.messageStartEvent)
	case "message_stop":
		e.messageStopEvent = &messageStopEvent{}
		return json.Unmarshal(data, e.messageStopEvent)
	case "message_delta":
		e.messageDeltaEvent = &messageDeltaEvent{}
		return json.Unmarshal(data, e.messageDeltaEvent)
	case "content_block_start":
		e.contentBlockStartEvent = &contentBlockStartEvent{}
		return json.Unmarshal(data, e.contentBlockStartEvent)
	case "content_block_stop":
		e.contentBlockStopEvent = &contentBlockStopEvent{}
		return json.Unmarshal(data, e.contentBlockStopEvent)
	case "content_block_delta":
		e.contentBlockDeltaEvent = &contentBlockDeltaEvent{}
		return json.Unmarshal(data, e.contentBlockDeltaEvent)
	case "ping":
		e.pingEvent = &pingEvent{}
		return json.Unmarshal(data, e.pingEvent)
	case "error":
		e.errorEvent = &errorEvent{}
		return json.Unmarshal(data, e.errorEvent)
	default:
		// Unknown event type - just store the raw data
		return nil
	}
}

// toolUseContentBlock is a specialized content block for tool usage.
type toolUseContentBlock struct {
	Type  string                 `json:"type"`  // "tool_use"
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// toolResultContentBlock is a specialized content block for tool results.
type toolResultContentBlock struct {
	Type      string      `json:"type"`       // "tool_result"
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content"`   // Can be string or content blocks
	IsError   bool        `json:"is_error,omitempty"`
}

// textContentBlock is a specialized content block for text.
type textContentBlock struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// Helper functions for creating content blocks

// NewTextContent creates a text content block.
func NewTextContent(text string) contentBlock {
	return contentBlock{
		Type: "text",
		Text: text,
	}
}

// NewToolUseContent creates a tool use content block.
func NewToolUseContent(id, name string, input map[string]interface{}) contentBlock {
	return contentBlock{
		Type:  "tool_use",
		ID:    id,
		Name:  name,
		Input: input,
	}
}

// NewToolResultContent creates a tool result content block.
func NewToolResultContent(toolUseID string, content interface{}, isError bool) contentBlock {
	return contentBlock{
		Type:    "tool_result",
		ID:      toolUseID,
		Content: content,
		IsError: isError,
	}
}

// NewImageContent creates an image content block.
func NewImageContent(mediaType, data string) contentBlock {
	return contentBlock{
		Type: "image",
		Source: &imageSource{
			Type:      "base64",
			MediaType: mediaType,
			Data:      data,
		},
	}
}

// IsToolUse returns true if this content block is a tool use.
func (cb *contentBlock) IsToolUse() bool {
	return cb.Type == "tool_use"
}

// IsToolResult returns true if this content block is a tool result.
func (cb *contentBlock) IsToolResult() bool {
	return cb.Type == "tool_result"
}

// IsText returns true if this content block is text.
func (cb *contentBlock) IsText() bool {
	return cb.Type == "text"
}

// IsImage returns true if this content block is an image.
func (cb *contentBlock) IsImage() bool {
	return cb.Type == "image"
}