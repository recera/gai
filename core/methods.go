package core

import (
	"fmt"
)

// Methods for LLMCallParts

// AddMessage adds a message to the conversation
func (l *LLMCallParts) AddMessage(msg Message) {
	l.Messages = append(l.Messages, msg)
}

// AddSystem sets the system message
func (l *LLMCallParts) AddSystem(msg Message) {
	l.System = msg
}

// Fluent builder methods

// WithProvider sets the LLM provider and returns the LLMCallParts for chaining
func (p *LLMCallParts) WithProvider(s string) *LLMCallParts {
	p.Provider = s
	return p
}

// WithModel sets the model name and returns the LLMCallParts for chaining
func (p *LLMCallParts) WithModel(s string) *LLMCallParts {
	p.Model = s
	return p
}

// WithTemp sets the temperature parameter and returns the LLMCallParts for chaining
func (p *LLMCallParts) WithTemp(t float64) *LLMCallParts {
	p.Temperature = t
	return p
}

// WithTemperature is an alias for WithTemp for better readability
func (p *LLMCallParts) WithTemperature(t float64) *LLMCallParts {
	return p.WithTemp(t)
}

// WithMaxTokens sets the maximum tokens and returns the LLMCallParts for chaining
func (p *LLMCallParts) WithMaxTokens(n int) *LLMCallParts {
	p.MaxTokens = n
	return p
}

// WithSystem sets the system message using a text string
func (p *LLMCallParts) WithSystem(text string) *LLMCallParts {
	p.System.Role = "system"
	p.System.AddTextContent(text)
	return p
}

// WithUserMessage adds a user message to the conversation
func (p *LLMCallParts) WithUserMessage(text string) *LLMCallParts {
	msg := Message{Role: "user"}
	msg.AddTextContent(text)
	p.AddMessage(msg)
	return p
}

// WithAssistantMessage adds an assistant message to the conversation
func (p *LLMCallParts) WithAssistantMessage(text string) *LLMCallParts {
	msg := Message{Role: "assistant"}
	msg.AddTextContent(text)
	p.AddMessage(msg)
	return p
}

// WithMessage adds a pre-constructed message to the conversation
func (p *LLMCallParts) WithMessage(msg Message) *LLMCallParts {
	p.AddMessage(msg)
	return p
}

// WithMessages adds multiple messages to the conversation
func (p *LLMCallParts) WithMessages(msgs ...Message) *LLMCallParts {
	for _, msg := range msgs {
		p.AddMessage(msg)
	}
	return p
}

// Clear resets the messages while preserving other settings
func (p *LLMCallParts) Clear() *LLMCallParts {
	p.Messages = []Message{}
	return p
}

// ClearAll resets all messages including system message
func (p *LLMCallParts) ClearAll() *LLMCallParts {
	p.Messages = []Message{}
	p.System = Message{Role: "system", Contents: []Content{}}
	return p
}

// WithTrace sets the trace function for debugging
func (p *LLMCallParts) WithTrace(trace func(TraceInfo)) *LLMCallParts {
	p.Trace = trace
	return p
}

// Methods for Message

// AddContent adds content to a message
func (m *Message) AddContent(content Content) {
	m.Contents = append(m.Contents, content)
}

// AddTextContent adds text content to a message
func (m *Message) AddTextContent(text string) {
	m.Contents = append(m.Contents, TextContent{Text: text})
}

// AddImageContent adds image content to a message
func (m *Message) AddImageContent(mimeType string, data []byte) {
	m.Contents = append(m.Contents, ImageContent{MIMEType: mimeType, Data: data})
}

// AddImageContentFromURL adds image content from a URL to a message
func (m *Message) AddImageContentFromURL(mimeType, url string) {
	m.Contents = append(m.Contents, ImageContent{MIMEType: mimeType, URL: url})
}

// GetTextContent retrieves the text content from a message
func (m *Message) GetTextContent() string {
	for _, content := range m.Contents {
		if textContent, ok := content.(TextContent); ok {
			return textContent.Text
		}
	}
	return ""
}

// CoalesceTextContent merges adjacent text content in a message
func (m *Message) CoalesceTextContent() {
	if len(m.Contents) < 2 {
		return
	}

	newContents := make([]Content, 0, len(m.Contents))
	for _, content := range m.Contents {
		lastContentIndex := len(newContents) - 1
		if lastContentIndex >= 0 {
			if lastText, ok := newContents[lastContentIndex].(TextContent); ok {
				if currentText, ok := content.(TextContent); ok {
					newContents[lastContentIndex] = TextContent{Text: lastText.Text + currentText.Text}
					continue
				}
			}
		}
		newContents = append(newContents, content)
	}

	m.Contents = newContents
}

// Methods for TextContent

// AppendText appends text to TextContent
func (c *TextContent) AppendText(text string) {
	c.Text += text
}

// Methods for LLMError

// Error implements the error interface
func (e *LLMError) Error() string {
	if e.Provider != "" && e.Model != "" {
		return fmt.Sprintf("llm %s/%s: %v", e.Provider, e.Model, e.Err)
	}
	return fmt.Sprintf("llm error: %v", e.Err)
}

// Unwrap returns the underlying error
func (e *LLMError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is support
func (e *LLMError) Is(target error) bool {
	if te, ok := target.(*LLMError); ok {
		return e.Provider == te.Provider && e.Model == te.Model
	}
	return false
}

// WithContext adds context to the error
func (e *LLMError) WithContext(key string, value interface{}) *LLMError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// NewLLMError creates a new LLM error
func NewLLMError(err error, provider, model string) *LLMError {
	return &LLMError{
		Err:      err,
		Provider: provider,
		Model:    model,
	}
}