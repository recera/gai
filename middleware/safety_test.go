package middleware

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

func TestSafetyMiddleware_RedactPatterns(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			// Echo back the user message
			if len(req.Messages) > 0 && len(req.Messages[0].Parts) > 0 {
				if text, ok := req.Messages[0].Parts[0].(core.Text); ok {
					return &core.TextResult{Text: text.Text}, nil
				}
			}
			return &core.TextResult{Text: "no message"}, nil
		},
	}

	opts := SafetyOpts{
		RedactPatterns: []string{
			`\b\d{3}-\d{2}-\d{4}\b`, // SSN
			`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email
		},
		RedactReplacement: "[REDACTED]",
	}
	
	provider := WithSafety(opts)(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "My SSN is 123-45-6789 and email is test@example.com"},
				},
			},
		},
	}
	
	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	expected := "My SSN is [REDACTED] and email is [REDACTED]"
	if result.Text != expected {
		t.Errorf("expected '%s', got '%s'", expected, result.Text)
	}
}

func TestSafetyMiddleware_BlockPatterns(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "response"}, nil
		},
	}

	opts := SafetyOpts{
		BlockPatterns: []string{
			`\bFORBIDDEN\b`,
		},
	}
	
	provider := WithSafety(opts)(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "This contains FORBIDDEN content"},
				},
			},
		},
	}
	
	_, err := provider.GenerateText(ctx, req)
	if err == nil {
		t.Fatal("expected error for blocked content")
	}
	if !core.IsContentFiltered(err) {
		t.Errorf("expected content filter error, got %v", err)
	}
}

func TestSafetyMiddleware_BlockWords(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "response"}, nil
		},
	}

	blockedReason := ""
	opts := SafetyOpts{
		BlockWords: []string{"badword", "inappropriate"},
		OnBlocked: func(reason, content string) {
			blockedReason = reason
		},
	}
	
	provider := WithSafety(opts)(mock)
	
	ctx := context.Background()
	
	testCases := []struct {
		name        string
		text        string
		shouldBlock bool
		word        string
	}{
		{"exact match", "this has badword in it", true, "badword"},
		{"case insensitive", "this has BADWORD in it", true, "badword"},
		{"partial match", "this has badwords in it", true, "badword"},
		{"no match", "this is clean text", false, ""},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			blockedReason = ""
			req := core.Request{
				Messages: []core.Message{
					{Role: core.User, Parts: []core.Part{core.Text{Text: tc.text}}},
				},
			}
			
			_, err := provider.GenerateText(ctx, req)
			
			if tc.shouldBlock {
				if err == nil {
					t.Error("expected error for blocked word")
				} else if !core.IsContentFiltered(err) {
					t.Errorf("expected content filter error, got %v", err)
				}
				if !strings.Contains(blockedReason, tc.word) {
					t.Errorf("expected block reason to contain '%s', got '%s'", tc.word, blockedReason)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSafetyMiddleware_MaxContentLength(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: strings.Repeat("a", 200)}, nil
		},
	}

	opts := SafetyOpts{
		MaxContentLength: 100,
	}
	
	provider := WithSafety(opts)(mock)
	
	ctx := context.Background()
	
	// Test request content length
	req := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: strings.Repeat("x", 150)},
				},
			},
		},
	}
	
	_, err := provider.GenerateText(ctx, req)
	if err == nil {
		t.Fatal("expected error for content exceeding max length")
	}
	if !core.IsContentFiltered(err) {
		t.Errorf("expected content filter error, got %v", err)
	}
	
	// Test response content length
	req.Messages[0].Parts[0] = core.Text{Text: "short"}
	_, err = provider.GenerateText(ctx, req)
	if err == nil {
		t.Fatal("expected error for response exceeding max length")
	}
	if !core.IsContentFiltered(err) {
		t.Errorf("expected content filter error for response, got %v", err)
	}
}

func TestSafetyMiddleware_CustomTransforms(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			if len(req.Messages) > 0 && len(req.Messages[0].Parts) > 0 {
				if text, ok := req.Messages[0].Parts[0].(core.Text); ok {
					return &core.TextResult{Text: text.Text}, nil
				}
			}
			return &core.TextResult{Text: "no message"}, nil
		},
	}

	opts := SafetyOpts{
		TransformRequest: func(messages []core.Message) ([]core.Message, error) {
			// Add a prefix to all user messages
			transformed := make([]core.Message, len(messages))
			for i, msg := range messages {
				transformed[i] = msg
				if msg.Role == core.User && len(msg.Parts) > 0 {
					if text, ok := msg.Parts[0].(core.Text); ok {
						text.Text = "[FILTERED] " + text.Text
						transformed[i].Parts[0] = text
					}
				}
			}
			return transformed, nil
		},
		TransformResponse: func(text string) (string, error) {
			// Add a suffix to responses
			return text + " [PROCESSED]", nil
		},
	}
	
	provider := WithSafety(opts)(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "hello"}}},
		},
	}
	
	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	expected := "[FILTERED] hello [PROCESSED]"
	if result.Text != expected {
		t.Errorf("expected '%s', got '%s'", expected, result.Text)
	}
}

func TestSafetyMiddleware_StreamFiltering(t *testing.T) {
	eventsChan := make(chan core.Event, 10)
	mock := &mockProvider{
		streamTextFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			// Send some events with content to filter
			go func() {
				eventsChan <- core.Event{Type: core.EventStart}
				eventsChan <- core.Event{Type: core.EventTextDelta, TextDelta: "My email is "}
				eventsChan <- core.Event{Type: core.EventTextDelta, TextDelta: "test@example.com"}
				eventsChan <- core.Event{Type: core.EventTextDelta, TextDelta: " and done"}
				eventsChan <- core.Event{Type: core.EventFinish}
				close(eventsChan)
			}()
			return &mockTextStream{events: eventsChan}, nil
		},
	}

	opts := SafetyOpts{
		RedactPatterns: []string{
			`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
		},
		RedactReplacement: "[EMAIL]",
	}
	
	provider := WithSafety(opts)(mock)
	
	ctx := context.Background()
	stream, err := provider.StreamText(ctx, core.Request{
		Messages: []core.Message{{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}}},
	})
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	var textDeltas []string
	for event := range stream.Events() {
		if event.Type == core.EventTextDelta {
			textDeltas = append(textDeltas, event.TextDelta)
		}
	}
	
	// Check that email was redacted in the stream
	combined := strings.Join(textDeltas, "")
	if !strings.Contains(combined, "[EMAIL]") {
		t.Errorf("expected email to be redacted, got: %s", combined)
	}
	if strings.Contains(combined, "@example.com") {
		t.Errorf("email not fully redacted: %s", combined)
	}
}

func TestSafetyMiddleware_StreamSafetyEvent(t *testing.T) {
	eventsChan := make(chan core.Event, 10)
	mock := &mockProvider{
		streamTextFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			go func() {
				eventsChan <- core.Event{Type: core.EventStart}
				eventsChan <- core.Event{Type: core.EventTextDelta, TextDelta: "Some text"}
				eventsChan <- core.Event{
					Type: core.EventSafety,
					Safety: &core.SafetyEvent{
						Category: "violence",
						Action:   "blocked",
						Score:    0.95,
					},
				}
				eventsChan <- core.Event{Type: core.EventFinish}
				close(eventsChan)
			}()
			return &mockTextStream{events: eventsChan}, nil
		},
	}

	opts := SafetyOpts{
		StopOnSafetyEvent: true,
	}
	
	provider := WithSafety(opts)(mock)
	
	ctx := context.Background()
	stream, err := provider.StreamText(ctx, core.Request{
		Messages: []core.Message{{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}}},
	})
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	var errorReceived bool
	var safetyReceived bool
	
	for event := range stream.Events() {
		switch event.Type {
		case core.EventSafety:
			safetyReceived = true
		case core.EventError:
			errorReceived = true
			if !core.IsContentFiltered(event.Err) {
				t.Errorf("expected content filter error, got %v", event.Err)
			}
		}
	}
	
	if !safetyReceived {
		t.Error("safety event not received")
	}
	if !errorReceived {
		t.Error("error event not received after safety event")
	}
}

func TestSafetyMiddleware_NonTextParts(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			// Count the parts
			partCount := 0
			for _, msg := range req.Messages {
				partCount += len(msg.Parts)
			}
			return &core.TextResult{Text: string(rune(partCount))}, nil
		},
	}

	opts := SafetyOpts{
		RedactPatterns: []string{`\d+`},
		RedactReplacement: "[NUM]",
	}
	
	provider := WithSafety(opts)(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Check this image with number 123"},
					core.ImageURL{URL: "https://example.com/image.png"},
					core.Audio{Source: core.BlobRef{Kind: core.BlobURL, URL: "https://example.com/audio.mp3"}},
				},
			},
		},
	}
	
	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Should have processed all 3 parts (text was redacted but other parts passed through)
	if result.Text != "3" {
		t.Errorf("expected all parts to be preserved, got result: %s", result.Text)
	}
}

func TestSafetyMiddleware_OnCallbacks(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "My SSN is 123-45-6789"}, nil
		},
	}

	var redactedPattern string
	var redactedCount int
	var blockedReason string
	
	opts := SafetyOpts{
		RedactPatterns: []string{`\d{3}-\d{2}-\d{4}`},
		RedactReplacement: "[SSN]",
		BlockWords: []string{"forbidden"},
		OnRedacted: func(pattern string, count int) {
			redactedPattern = pattern
			redactedCount = count
		},
		OnBlocked: func(reason, content string) {
			blockedReason = reason
		},
	}
	
	provider := WithSafety(opts)(mock)
	ctx := context.Background()
	
	// Test redaction callback
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}},
		},
	}
	
	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if !strings.Contains(result.Text, "[SSN]") {
		t.Errorf("SSN not redacted: %s", result.Text)
	}
	if redactedCount != 1 {
		t.Errorf("expected 1 redaction, got %d", redactedCount)
	}
	if !strings.Contains(redactedPattern, "\\d{3}") {
		t.Errorf("unexpected pattern in callback: %s", redactedPattern)
	}
	
	// Test blocked callback
	req.Messages[0].Parts[0] = core.Text{Text: "this is forbidden"}
	_, err = provider.GenerateText(ctx, req)
	
	if err == nil {
		t.Fatal("expected error for blocked word")
	}
	if !strings.Contains(blockedReason, "forbidden") {
		t.Errorf("expected block reason to mention 'forbidden', got: %s", blockedReason)
	}
}

func TestSafetyMiddleware_ConcurrentSafety(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "response"}, nil
		},
	}

	opts := SafetyOpts{
		RedactPatterns: []string{`\d+`},
		RedactReplacement: "[NUM]",
	}
	
	provider := WithSafety(opts)(mock)
	
	ctx := context.Background()
	var wg sync.WaitGroup
	
	// Run multiple concurrent requests
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			req := core.Request{
				Messages: []core.Message{
					{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}},
				},
			}
			_, err := provider.GenerateText(ctx, req)
			if err != nil {
				t.Errorf("request %d failed: %v", n, err)
			}
		}(i)
	}
	
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent requests timed out")
	}
}

func TestSafetyMiddleware_TransformError(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "response"}, nil
		},
	}

	customErr := errors.New("transform failed")
	opts := SafetyOpts{
		TransformRequest: func(messages []core.Message) ([]core.Message, error) {
			return nil, customErr
		},
	}
	
	provider := WithSafety(opts)(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: "test"}}},
		},
	}
	
	_, err := provider.GenerateText(ctx, req)
	if err == nil {
		t.Fatal("expected error from transform")
	}
	if !errors.Is(err, customErr) {
		t.Errorf("expected custom error, got %v", err)
	}
}