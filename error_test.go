package gai

import (
	"errors"
	"testing"
)

func TestNewLLMError(t *testing.T) {
	baseErr := errors.New("connection failed")
	llmErr := NewLLMError(baseErr, "openai", "gpt-4")
	
	if llmErr.Err != baseErr {
		t.Errorf("Expected base error to be preserved")
	}
	if llmErr.Provider != "openai" {
		t.Errorf("Expected provider to be openai, got %s", llmErr.Provider)
	}
	if llmErr.Model != "gpt-4" {
		t.Errorf("Expected model to be gpt-4, got %s", llmErr.Model)
	}
}

func TestLLMErrorString(t *testing.T) {
	tests := []struct {
		name     string
		err      *LLMError
		expected string
	}{
		{
			name: "With provider and model",
			err: &LLMError{
				Err:      errors.New("rate limit exceeded"),
				Provider: "openai",
				Model:    "gpt-4",
			},
			expected: "llm openai/gpt-4: rate limit exceeded",
		},
		{
			name: "Without provider and model",
			err: &LLMError{
				Err: errors.New("generic error"),
			},
			expected: "llm error: generic error",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Expected error string %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestLLMErrorContext(t *testing.T) {
	err := NewLLMError(errors.New("test"), "openai", "gpt-4")
	
	// Add context
	err.WithContext("attempt", 3).
		WithContext("request_id", "abc123")
	
	if err.Context["attempt"] != 3 {
		t.Errorf("Expected attempt to be 3, got %v", err.Context["attempt"])
	}
	if err.Context["request_id"] != "abc123" {
		t.Errorf("Expected request_id to be abc123, got %v", err.Context["request_id"])
	}
}

func TestLLMErrorUnwrap(t *testing.T) {
	baseErr := errors.New("base error")
	llmErr := NewLLMError(baseErr, "openai", "gpt-4")
	
	unwrapped := llmErr.Unwrap()
	if unwrapped != baseErr {
		t.Errorf("Expected unwrapped error to be base error")
	}
}

func TestWrapError(t *testing.T) {
	baseErr := errors.New("API timeout")
	parts := NewLLMCallParts().
		WithProvider("anthropic").
		WithModel("claude-3")
	
	wrapped := WrapError(baseErr, *parts)
	
	if wrapped.Err != baseErr {
		t.Errorf("Expected base error to be preserved")
	}
	if wrapped.Provider != "anthropic" {
		t.Errorf("Expected provider to be anthropic, got %s", wrapped.Provider)
	}
	if wrapped.Model != "claude-3" {
		t.Errorf("Expected model to be claude-3, got %s", wrapped.Model)
	}
}

func TestLLMErrorIs(t *testing.T) {
	err1 := NewLLMError(errors.New("test"), "openai", "gpt-4")
	err2 := NewLLMError(errors.New("test"), "openai", "gpt-4")
	err3 := NewLLMError(errors.New("test"), "anthropic", "claude")
	
	if !err1.Is(err2) {
		t.Errorf("Expected errors with same provider/model to match")
	}
	if err1.Is(err3) {
		t.Errorf("Expected errors with different provider/model to not match")
	}
}