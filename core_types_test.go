package gai

import (
	"testing"
)

func TestNewLLMCallParts(t *testing.T) {
	parts := NewLLMCallParts()

	// Check defaults
	if parts.Provider != "" {
		t.Errorf("Expected default provider to be empty, got %s", parts.Provider)
	}
	if parts.Model != "" {
		t.Errorf("Expected default model to be empty, got %s", parts.Model)
	}
	if parts.MaxTokens != 1000 {
		t.Errorf("Expected default max tokens to be 1000, got %d", parts.MaxTokens)
	}
	if parts.Temperature != 0.2 {
		t.Errorf("Expected default temperature to be 0.2, got %f", parts.Temperature)
	}
}

func TestLLMCallPartsBuilder(t *testing.T) {
	parts := NewLLMCallParts().
		WithProvider("openai").
		WithModel("gpt-4").
		WithTemperature(0.7).
		WithMaxTokens(2000).
		WithSystem("You are helpful").
		WithUserMessage("Hello").
		WithAssistantMessage("Hi there!")

	if parts.Provider != "openai" {
		t.Errorf("Expected provider to be openai, got %s", parts.Provider)
	}
	if parts.Model != "gpt-4" {
		t.Errorf("Expected model to be gpt-4, got %s", parts.Model)
	}
	if parts.Temperature != 0.7 {
		t.Errorf("Expected temperature to be 0.7, got %f", parts.Temperature)
	}
	if parts.MaxTokens != 2000 {
		t.Errorf("Expected max tokens to be 2000, got %d", parts.MaxTokens)
	}

	// Check system message
	if parts.System.GetTextContent() != "You are helpful" {
		t.Errorf("Expected system message to be 'You are helpful', got %s", parts.System.GetTextContent())
	}

	// Check messages
	if len(parts.Messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(parts.Messages))
	}
	if parts.Messages[0].Role != "user" || parts.Messages[0].GetTextContent() != "Hello" {
		t.Errorf("Expected first message to be user: Hello")
	}
	if parts.Messages[1].Role != "assistant" || parts.Messages[1].GetTextContent() != "Hi there!" {
		t.Errorf("Expected second message to be assistant: Hi there!")
	}
}

func TestMessageConstructors(t *testing.T) {
	tests := []struct {
		name     string
		message  Message
		wantRole string
		wantText string
	}{
		{
			name:     "User message",
			message:  NewUserMessage("Hello"),
			wantRole: "user",
			wantText: "Hello",
		},
		{
			name:     "Assistant message",
			message:  NewAssistantMessage("Hi there"),
			wantRole: "assistant",
			wantText: "Hi there",
		},
		{
			name:     "System message",
			message:  NewSystemMessage("You are helpful"),
			wantRole: "system",
			wantText: "You are helpful",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.message.Role != tt.wantRole {
				t.Errorf("Expected role %s, got %s", tt.wantRole, tt.message.Role)
			}
			if tt.message.GetTextContent() != tt.wantText {
				t.Errorf("Expected text %s, got %s", tt.wantText, tt.message.GetTextContent())
			}
		})
	}
}

func TestMessageBuilder(t *testing.T) {
	msg := NewMessageBuilder("user").
		WithText("Check this image:").
		WithImage("image/png", []byte("fake-image-data")).
		Build()

	if msg.Role != "user" {
		t.Errorf("Expected role to be user, got %s", msg.Role)
	}

	if len(msg.Contents) != 2 {
		t.Fatalf("Expected 2 contents, got %d", len(msg.Contents))
	}

	// Check text content
	if textContent, ok := msg.Contents[0].(TextContent); !ok {
		t.Errorf("Expected first content to be TextContent")
	} else if textContent.Text != "Check this image:" {
		t.Errorf("Expected text 'Check this image:', got %s", textContent.Text)
	}

	// Check image content
	if imageContent, ok := msg.Contents[1].(ImageContent); !ok {
		t.Errorf("Expected second content to be ImageContent")
	} else {
		if imageContent.MIMEType != "image/png" {
			t.Errorf("Expected MIME type image/png, got %s", imageContent.MIMEType)
		}
		if string(imageContent.Data) != "fake-image-data" {
			t.Errorf("Expected image data to match")
		}
	}
}

func TestCoalesceTextContent(t *testing.T) {
	msg := Message{Role: "user", Contents: []Content{}}
	msg.AddTextContent("Hello ")
	msg.AddTextContent("world")
	msg.AddImageContent("image/png", []byte("data"))
	msg.AddTextContent(" from ")
	msg.AddTextContent("Go!")

	// Before coalescing
	if len(msg.Contents) != 5 {
		t.Errorf("Expected 5 contents before coalescing, got %d", len(msg.Contents))
	}

	msg.CoalesceTextContent()

	// After coalescing
	if len(msg.Contents) != 3 {
		t.Errorf("Expected 3 contents after coalescing, got %d", len(msg.Contents))
	}

	// Check coalesced content
	if text, ok := msg.Contents[0].(TextContent); !ok {
		t.Errorf("Expected first content to be TextContent")
	} else if text.Text != "Hello world" {
		t.Errorf("Expected 'Hello world', got %s", text.Text)
	}

	if _, ok := msg.Contents[1].(ImageContent); !ok {
		t.Errorf("Expected second content to be ImageContent")
	}

	if text, ok := msg.Contents[2].(TextContent); !ok {
		t.Errorf("Expected third content to be TextContent")
	} else if text.Text != " from Go!" {
		t.Errorf("Expected ' from Go!', got %s", text.Text)
	}
}

func TestClearMethods(t *testing.T) {
	parts := NewLLMCallParts().
		WithSystem("System message").
		WithUserMessage("Message 1").
		WithAssistantMessage("Response 1").
		WithUserMessage("Message 2")

	// Test Clear (keeps system message)
	parts.Clear()
	if len(parts.Messages) != 0 {
		t.Errorf("Expected no messages after Clear, got %d", len(parts.Messages))
	}
	if parts.System.GetTextContent() != "System message" {
		t.Errorf("Expected system message to remain after Clear")
	}

	// Add messages again
	parts.WithUserMessage("New message")

	// Test ClearAll
	parts.ClearAll()
	if len(parts.Messages) != 0 {
		t.Errorf("Expected no messages after ClearAll, got %d", len(parts.Messages))
	}
	if parts.System.GetTextContent() != "" {
		t.Errorf("Expected empty system message after ClearAll, got %s", parts.System.GetTextContent())
	}
}
