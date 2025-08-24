package middleware

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/recera/gai/core"
)

// SafetyOpts configures the safety middleware for content filtering and redaction.
type SafetyOpts struct {
	// BlockPatterns are regex patterns that will block requests/responses if matched.
	BlockPatterns []string
	// RedactPatterns are regex patterns for content that should be redacted.
	RedactPatterns []string
	// RedactReplacement is the string to replace redacted content with.
	RedactReplacement string
	// BlockWords are exact words/phrases that will block content if found.
	BlockWords []string
	// MaxContentLength limits the total content length (0 = no limit).
	MaxContentLength int
	// TransformRequest is a custom function to transform/filter request messages.
	TransformRequest func([]core.Message) ([]core.Message, error)
	// TransformResponse is a custom function to transform/filter response text.
	TransformResponse func(string) (string, error)
	// OnBlocked is called when content is blocked (for observability).
	OnBlocked func(reason string, content string)
	// OnRedacted is called when content is redacted (for observability).
	OnRedacted func(pattern string, count int)
	// StopOnSafetyEvent stops streaming when a safety event is received.
	StopOnSafetyEvent bool
}

// DefaultSafetyOpts returns default safety options with common PII patterns.
func DefaultSafetyOpts() SafetyOpts {
	return SafetyOpts{
		RedactPatterns: []string{
			// Common PII patterns
			`\b\d{3}-\d{2}-\d{4}\b`,                          // SSN
			`\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`,     // Email (case insensitive in compilation)
			`\b(?:\+?1[-.]?)?\(?[0-9]{3}\)?[-.]?[0-9]{3}[-.]?[0-9]{4}\b`, // Phone numbers
			`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|6(?:011|5[0-9]{2})[0-9]{12})\b`, // Credit cards
		},
		RedactReplacement: "[REDACTED]",
		BlockWords:        []string{}, // Empty by default
		MaxContentLength:  0,           // No limit by default
	}
}

// safetyMiddleware implements content filtering and redaction.
type safetyMiddleware struct {
	baseMiddleware
	opts            SafetyOpts
	blockRegexps    []*regexp.Regexp
	redactRegexps   []*regexp.Regexp
	blockWordsLower []string // Lowercase version for case-insensitive matching
	mu              sync.RWMutex
}

// WithSafety creates middleware that filters and redacts content for safety.
func WithSafety(opts SafetyOpts) Middleware {
	// Set defaults
	if opts.RedactReplacement == "" {
		opts.RedactReplacement = "[REDACTED]"
	}

	return func(provider core.Provider) core.Provider {
		m := &safetyMiddleware{
			baseMiddleware:  baseMiddleware{provider: provider},
			opts:           opts,
			blockRegexps:   make([]*regexp.Regexp, 0, len(opts.BlockPatterns)),
			redactRegexps:  make([]*regexp.Regexp, 0, len(opts.RedactPatterns)),
			blockWordsLower: make([]string, 0, len(opts.BlockWords)),
		}

		// Compile block patterns
		for _, pattern := range opts.BlockPatterns {
			if re, err := regexp.Compile(pattern); err == nil {
				m.blockRegexps = append(m.blockRegexps, re)
			}
		}

		// Compile redact patterns with case-insensitive flag where appropriate
		for _, pattern := range opts.RedactPatterns {
			// Add case-insensitive flag for email pattern
			if strings.Contains(pattern, "@") {
				pattern = "(?i)" + pattern
			}
			if re, err := regexp.Compile(pattern); err == nil {
				m.redactRegexps = append(m.redactRegexps, re)
			}
		}

		// Convert block words to lowercase for case-insensitive matching
		for _, word := range opts.BlockWords {
			m.blockWordsLower = append(m.blockWordsLower, strings.ToLower(word))
		}

		return m
	}
}

// checkBlocked checks if content should be blocked based on patterns and words.
func (m *safetyMiddleware) checkBlocked(content string) error {
	// Check block patterns
	for _, re := range m.blockRegexps {
		if re.MatchString(content) {
			reason := fmt.Sprintf("blocked pattern: %s", re.String())
			if m.opts.OnBlocked != nil {
				m.opts.OnBlocked(reason, content)
			}
			return core.NewError(
				core.ErrorSafetyBlocked,
				reason,
				core.WithProvider("middleware"),
			)
		}
	}

	// Check block words (case-insensitive)
	contentLower := strings.ToLower(content)
	for i, word := range m.blockWordsLower {
		if strings.Contains(contentLower, word) {
			reason := fmt.Sprintf("blocked word: %s", m.opts.BlockWords[i])
			if m.opts.OnBlocked != nil {
				m.opts.OnBlocked(reason, content)
			}
			return core.NewError(
				core.ErrorSafetyBlocked,
				reason,
				core.WithProvider("middleware"),
			)
		}
	}

	// Check content length
	if m.opts.MaxContentLength > 0 && len(content) > m.opts.MaxContentLength {
		reason := fmt.Sprintf("content length %d exceeds maximum %d", len(content), m.opts.MaxContentLength)
		if m.opts.OnBlocked != nil {
			m.opts.OnBlocked(reason, content)
		}
		return core.NewError(
			core.ErrorSafetyBlocked,
			reason,
			core.WithProvider("middleware"),
		)
	}

	return nil
}

// redactContent applies redaction patterns to content.
func (m *safetyMiddleware) redactContent(content string) string {
	redacted := content
	totalRedactions := 0

	for _, re := range m.redactRegexps {
		matches := re.FindAllString(redacted, -1)
		if len(matches) > 0 {
			redacted = re.ReplaceAllString(redacted, m.opts.RedactReplacement)
			totalRedactions += len(matches)
			if m.opts.OnRedacted != nil {
				m.opts.OnRedacted(re.String(), len(matches))
			}
		}
	}

	return redacted
}

// filterMessages filters request messages for safety.
func (m *safetyMiddleware) filterMessages(messages []core.Message) ([]core.Message, error) {
	// Apply custom transform if provided
	if m.opts.TransformRequest != nil {
		return m.opts.TransformRequest(messages)
	}

	// Check and redact message content
	filtered := make([]core.Message, 0, len(messages))
	for _, msg := range messages {
		// Create a copy of the message
		newMsg := core.Message{
			Role:  msg.Role,
			Name:  msg.Name,
			Parts: make([]core.Part, 0, len(msg.Parts)),
		}

		// Process each part
		for _, part := range msg.Parts {
			switch p := part.(type) {
			case core.Text:
				// Check if blocked
				if err := m.checkBlocked(p.Text); err != nil {
					return nil, err
				}
				// Redact sensitive content
				p.Text = m.redactContent(p.Text)
				newMsg.Parts = append(newMsg.Parts, p)
			default:
				// Pass through non-text parts unchanged
				newMsg.Parts = append(newMsg.Parts, part)
			}
		}

		filtered = append(filtered, newMsg)
	}

	return filtered, nil
}

// filterResponse filters response text for safety.
func (m *safetyMiddleware) filterResponse(text string) (string, error) {
	// Apply custom transform if provided
	if m.opts.TransformResponse != nil {
		return m.opts.TransformResponse(text)
	}

	// Check if blocked
	if err := m.checkBlocked(text); err != nil {
		return "", err
	}

	// Redact sensitive content
	return m.redactContent(text), nil
}

// GenerateText implements the Provider interface with safety filtering.
func (m *safetyMiddleware) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
	// Filter request messages
	filteredMessages, err := m.filterMessages(req.Messages)
	if err != nil {
		return nil, err
	}
	
	// Create a copy of the request with filtered messages
	filteredReq := req
	filteredReq.Messages = filteredMessages

	// Execute the request
	result, err := m.provider.GenerateText(ctx, filteredReq)
	if err != nil {
		return nil, err
	}

	// Filter the response text
	if result != nil {
		filteredText, err := m.filterResponse(result.Text)
		if err != nil {
			return nil, err
		}
		result.Text = filteredText
	}

	return result, nil
}

// safetyStream wraps a TextStream to filter events.
type safetyStream struct {
	stream   core.TextStream
	safety   *safetyMiddleware
	events   chan core.Event
	done     chan struct{}
	closeOnce sync.Once
}

func (s *safetyStream) Events() <-chan core.Event {
	return s.events
}

func (s *safetyStream) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)
	})
	return s.stream.Close()
}

// processStream filters events from the wrapped stream.
func (s *safetyStream) processStream() {
	defer close(s.events)
	
	var textBuffer strings.Builder
	
	for event := range s.stream.Events() {
		select {
		case <-s.done:
			return
		default:
		}

		// Process different event types
		switch event.Type {
		case core.EventTextDelta:
			// Accumulate text for filtering
			textBuffer.WriteString(event.TextDelta)
			// For streaming, we can't block on individual deltas, but we can redact
			filtered := s.safety.redactContent(event.TextDelta)
			event.TextDelta = filtered
			s.events <- event

		case core.EventSafety:
			// Check if we should stop on safety events
			if s.safety.opts.StopOnSafetyEvent && event.Safety != nil {
				// Send error event and stop
				s.events <- core.Event{
					Type: core.EventError,
					Err: core.NewError(
						core.ErrorSafetyBlocked,
						fmt.Sprintf("stopped due to safety event: %s", event.Safety.Category),
						core.WithProvider("middleware"),
					),
				}
				return
			}
			s.events <- event

		case core.EventFinish:
			// Check the complete accumulated text
			fullText := textBuffer.String()
			if err := s.safety.checkBlocked(fullText); err != nil {
				// Send error instead of finish
				s.events <- core.Event{
					Type: core.EventError,
					Err:  err,
				}
				return
			}
			s.events <- event

		default:
			// Pass through other events unchanged
			s.events <- event
		}
	}
}

// StreamText implements the Provider interface with safety filtering.
func (m *safetyMiddleware) StreamText(ctx context.Context, req core.Request) (core.TextStream, error) {
	// Filter request messages
	filteredMessages, err := m.filterMessages(req.Messages)
	if err != nil {
		return nil, err
	}
	
	// Create a copy of the request with filtered messages
	filteredReq := req
	filteredReq.Messages = filteredMessages

	// Get the underlying stream
	stream, err := m.provider.StreamText(ctx, filteredReq)
	if err != nil {
		return nil, err
	}

	// Wrap the stream with safety filtering
	safeStream := &safetyStream{
		stream: stream,
		safety: m,
		events: make(chan core.Event, 100), // Buffer for smooth streaming
		done:   make(chan struct{}),
	}

	// Start processing in background
	go safeStream.processStream()

	return safeStream, nil
}

// GenerateObject implements the Provider interface with safety filtering.
func (m *safetyMiddleware) GenerateObject(ctx context.Context, req core.Request, schema any) (*core.ObjectResult[any], error) {
	// Filter request messages
	filteredMessages, err := m.filterMessages(req.Messages)
	if err != nil {
		return nil, err
	}
	
	// Create a copy of the request with filtered messages
	filteredReq := req
	filteredReq.Messages = filteredMessages

	// Execute the request
	// Note: Object responses are not filtered as they're structured data
	return m.provider.GenerateObject(ctx, filteredReq, schema)
}

// StreamObject implements the Provider interface with safety filtering.
func (m *safetyMiddleware) StreamObject(ctx context.Context, req core.Request, schema any) (core.ObjectStream[any], error) {
	// Filter request messages
	filteredMessages, err := m.filterMessages(req.Messages)
	if err != nil {
		return nil, err
	}
	
	// Create a copy of the request with filtered messages
	filteredReq := req
	filteredReq.Messages = filteredMessages

	// Get the underlying stream
	// Note: Object streams are not filtered as they're structured data
	return m.provider.StreamObject(ctx, filteredReq, schema)
}