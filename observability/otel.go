package observability

// Lightweight placeholders to avoid importing OTel in core library.
// Apps can wrap these with real OpenTelemetry if desired.

import (
	"context"
	"time"
)

type noopSpan struct{}

func (noopSpan) End() {}

type StreamMetrics struct {
	Provider string
	Model    string
	Start    time.Time
	FirstAt  time.Time
}

func StartStream(ctx context.Context, provider, model string) (context.Context, noopSpan, *StreamMetrics) {
	m := &StreamMetrics{Provider: provider, Model: model, Start: time.Now()}
	return ctx, noopSpan{}, m
}

func MarkFirstToken(m *StreamMetrics) {
	if m.FirstAt.IsZero() {
		m.FirstAt = time.Now()
	}
}

func CloseStream(span noopSpan, m *StreamMetrics, _ string) { span.End() }

func ToolCall(ctx context.Context, _, _, _ string) (context.Context, noopSpan) {
	return ctx, noopSpan{}
}
