package gai

// Message constructor functions for cleaner API usage

// NewUserMessage creates a new user message with the given text content
func NewUserMessage(text string) Message {
	m := Message{Role: "user", Contents: []Content{}}
	m.AddTextContent(text)
	return m
}

// NewAssistantMessage creates a new assistant message with the given text content
func NewAssistantMessage(text string) Message {
	m := Message{Role: "assistant", Contents: []Content{}}
	m.AddTextContent(text)
	return m
}

// NewSystemMessage creates a new system message with the given text content
func NewSystemMessage(text string) Message {
	m := Message{Role: "system", Contents: []Content{}}
	m.AddTextContent(text)
	return m
}

// NewToolResponseMessage creates a tool response message suitable for providers
// that accept role:"tool" messages. Set ToolCallID on the returned message when
// replying to specific tool calls (e.g., OpenAI tool_call_id).
func NewToolResponseMessage(output string) Message {
	m := Message{Role: "tool", Contents: []Content{}}
	m.AddContent(TextContent{Text: output})
	return m
}

// MessageBuilder provides a fluent interface for building complex messages
type MessageBuilder struct {
	message Message
}

// NewMessageBuilder creates a new message builder with the specified role
func NewMessageBuilder(role string) *MessageBuilder {
	return &MessageBuilder{
		message: Message{Role: role, Contents: []Content{}},
	}
}

// WithText adds text content to the message
func (mb *MessageBuilder) WithText(text string) *MessageBuilder {
	mb.message.AddTextContent(text)
	return mb
}

// WithImage adds image content to the message
func (mb *MessageBuilder) WithImage(mimeType string, data []byte) *MessageBuilder {
	mb.message.AddImageContent(mimeType, data)
	return mb
}

// WithImageURL adds image content from a URL to the message
func (mb *MessageBuilder) WithImageURL(mimeType, url string) *MessageBuilder {
	mb.message.AddImageContentFromURL(mimeType, url)
	return mb
}

// Build returns the constructed message
func (mb *MessageBuilder) Build() Message {
	return mb.message
}

// Convenience functions for common patterns

// NewUserMessageWithImage creates a user message with both text and image content
func NewUserMessageWithImage(text, imageMimeType string, imageData []byte) Message {
	return NewMessageBuilder("user").
		WithText(text).
		WithImage(imageMimeType, imageData).
		Build()
}

// NewUserMessageWithImageURL creates a user message with text and image from URL
func NewUserMessageWithImageURL(text, imageMimeType, imageURL string) Message {
	return NewMessageBuilder("user").
		WithText(text).
		WithImageURL(imageMimeType, imageURL).
		Build()
}
