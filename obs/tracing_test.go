package obs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// setupTestTracer creates an in-memory exporter for testing
func setupTestTracer() (*tracetest.InMemoryExporter, func()) {
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
		trace.WithResource(nil),
	)
	
	// Set as global provider
	otel.SetTracerProvider(tp)
	SetGlobalTracerProvider(tp)
	
	cleanup := func() {
		// Reset to noop
		otel.SetTracerProvider(otel.GetTracerProvider())
		tracerOnce = sync.Once{}
	}
	
	return exporter, cleanup
}

func TestTracer(t *testing.T) {
	t.Run("returns noop tracer when not configured", func(t *testing.T) {
		// Reset tracer
		tracerOnce = sync.Once{}
		otel.SetTracerProvider(nil)
		
		tracer := Tracer()
		if tracer == nil {
			t.Fatal("expected non-nil tracer")
		}
		
		// Should be noop - create a span and verify it doesn't record
		_, span := tracer.Start(context.Background(), "test")
		if span.IsRecording() {
			t.Error("expected noop span to not be recording")
		}
	})
	
	t.Run("returns configured tracer when provider is set", func(t *testing.T) {
		exporter, cleanup := setupTestTracer()
		defer cleanup()
		
		tracer := Tracer()
		if tracer == nil {
			t.Fatal("expected non-nil tracer")
		}
		
		// Create a span and verify it records
		ctx := context.Background()
		_, span := tracer.Start(ctx, "test")
		span.End()
		
		// Check exported spans
		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}
		
		if spans[0].Name != "test" {
			t.Errorf("expected span name 'test', got %s", spans[0].Name)
		}
	})
}

func TestStartRequestSpan(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()
	
	ctx := context.Background()
	opts := RequestSpanOptions{
		Provider:     "openai",
		Model:        "gpt-4",
		Temperature:  0.7,
		MaxTokens:    1000,
		Stream:       true,
		ToolCount:    3,
		MessageCount: 5,
		SystemPrompt: true,
		ProviderOptions: map[string]any{
			"top_p": 0.9,
		},
		Metadata: map[string]any{
			"user_id": "test123",
		},
	}
	
	ctx, span := StartRequestSpan(ctx, opts)
	span.End()
	
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	
	s := spans[0]
	if s.Name != "ai.request" {
		t.Errorf("expected span name 'ai.request', got %s", s.Name)
	}
	
	// Check attributes
	attrs := s.Attributes
	checkAttribute(t, attrs, "llm.provider", "openai")
	checkAttribute(t, attrs, "llm.model", "gpt-4")
	checkAttribute(t, attrs, "llm.temperature", 0.7)
	checkAttribute(t, attrs, "llm.max_tokens", int64(1000))
	checkAttribute(t, attrs, "llm.stream", true)
	checkAttribute(t, attrs, "llm.tools.count", int64(3))
	checkAttribute(t, attrs, "llm.messages.count", int64(5))
	checkAttribute(t, attrs, "llm.system_prompt", true)
	checkAttribute(t, attrs, "llm.provider.top_p", "0.9")
	checkAttribute(t, attrs, "metadata.user_id", "test123")
}

func TestStartStepSpan(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()
	
	ctx := context.Background()
	opts := StepSpanOptions{
		StepNumber:   2,
		HasToolCalls: true,
		ToolCount:    2,
		TextLength:   150,
	}
	
	ctx, span := StartStepSpan(ctx, opts)
	span.End()
	
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	
	s := spans[0]
	if s.Name != "ai.step.2" {
		t.Errorf("expected span name 'ai.step.2', got %s", s.Name)
	}
	
	attrs := s.Attributes
	checkAttribute(t, attrs, "step.number", int64(2))
	checkAttribute(t, attrs, "step.has_tool_calls", true)
	checkAttribute(t, attrs, "step.tool_count", int64(2))
	checkAttribute(t, attrs, "step.text_length", int64(150))
}

func TestStartToolSpan(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()
	
	ctx := context.Background()
	opts := ToolSpanOptions{
		ToolName:   "get_weather",
		ToolID:     "tool_123",
		InputSize:  256,
		StepNumber: 1,
		Parallel:   true,
		RetryCount: 0,
		Timeout:    30 * time.Second,
	}
	
	ctx, span := StartToolSpan(ctx, opts)
	span.End()
	
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	
	s := spans[0]
	if s.Name != "ai.tool.get_weather" {
		t.Errorf("expected span name 'ai.tool.get_weather', got %s", s.Name)
	}
	
	attrs := s.Attributes
	checkAttribute(t, attrs, "tool.name", "get_weather")
	checkAttribute(t, attrs, "tool.id", "tool_123")
	checkAttribute(t, attrs, "tool.input_size", int64(256))
	checkAttribute(t, attrs, "tool.step_number", int64(1))
	checkAttribute(t, attrs, "tool.parallel", true)
	checkAttribute(t, attrs, "tool.retry_count", int64(0))
	checkAttribute(t, attrs, "tool.timeout_seconds", 30.0)
}

func TestStartPromptSpan(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()
	
	ctx := context.Background()
	opts := PromptSpanOptions{
		Name:        "assistant",
		Version:     "1.0.0",
		Fingerprint: "abc123",
		DataKeys:    []string{"name", "role"},
		Override:    false,
		CacheHit:    true,
	}
	
	ctx, span := StartPromptSpan(ctx, opts)
	span.End()
	
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	
	s := spans[0]
	if s.Name != "ai.prompt.assistant" {
		t.Errorf("expected span name 'ai.prompt.assistant', got %s", s.Name)
	}
	
	attrs := s.Attributes
	checkAttribute(t, attrs, "prompt.name", "assistant")
	checkAttribute(t, attrs, "prompt.version", "1.0.0")
	checkAttribute(t, attrs, "prompt.fingerprint", "abc123")
	checkAttribute(t, attrs, "prompt.override", false)
	checkAttribute(t, attrs, "prompt.cache_hit", true)
}

func TestRecordUsage(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()
	
	ctx := context.Background()
	ctx, span := Tracer().Start(ctx, "test")
	
	RecordUsage(span, 100, 200, 300)
	span.End()
	
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	
	attrs := spans[0].Attributes
	checkAttribute(t, attrs, "usage.input_tokens", int64(100))
	checkAttribute(t, attrs, "usage.output_tokens", int64(200))
	checkAttribute(t, attrs, "usage.total_tokens", int64(300))
}

func TestRecordError(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()
	
	ctx := context.Background()
	ctx, span := Tracer().Start(ctx, "test")
	
	err := errors.New("test error")
	RecordError(span, err, "operation failed")
	span.End()
	
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	
	s := spans[0]
	if s.Status.Code != codes.Error {
		t.Errorf("expected error status, got %v", s.Status.Code)
	}
	
	if s.Status.Description != "operation failed" {
		t.Errorf("expected status description 'operation failed', got %s", s.Status.Description)
	}
	
	// Check that error was recorded
	if len(s.Events) != 1 {
		t.Fatalf("expected 1 event (error), got %d", len(s.Events))
	}
	
	if s.Events[0].Name != "exception" {
		t.Errorf("expected event name 'exception', got %s", s.Events[0].Name)
	}
}

func TestRecordToolResult(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()
	
	ctx := context.Background()
	ctx, span := Tracer().Start(ctx, "test")
	
	RecordToolResult(span, true, 512, 100*time.Millisecond)
	span.End()
	
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	
	attrs := spans[0].Attributes
	checkAttribute(t, attrs, "tool.success", true)
	checkAttribute(t, attrs, "tool.output_size", int64(512))
	checkAttribute(t, attrs, "tool.duration_ms", 100.0)
	
	// Check status
	if spans[0].Status.Code != codes.Ok {
		t.Errorf("expected OK status, got %v", spans[0].Status.Code)
	}
}

func TestRecordProviderLatency(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()
	
	ctx := context.Background()
	ctx, span := Tracer().Start(ctx, "test")
	
	latency := 500 * time.Millisecond
	firstTokenLatency := 50 * time.Millisecond
	RecordProviderLatency(span, latency, &firstTokenLatency)
	span.End()
	
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	
	attrs := spans[0].Attributes
	checkAttribute(t, attrs, "provider.latency_ms", 500.0)
	checkAttribute(t, attrs, "provider.first_token_latency_ms", 50.0)
}

func TestWithSpan(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()
	
	ctx := context.Background()
	
	t.Run("successful execution", func(t *testing.T) {
		executed := false
		err := WithSpan(ctx, "test.operation", func(ctx context.Context, span oteltrace.Span) error {
			executed = true
			// Verify we have a valid span
			if !span.IsRecording() {
				t.Error("expected recording span")
			}
			return nil
		})
		
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		
		if !executed {
			t.Error("function was not executed")
		}
		
		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}
		
		if spans[0].Name != "test.operation" {
			t.Errorf("expected span name 'test.operation', got %s", spans[0].Name)
		}
	})
	
	// Clear spans
	exporter.Reset()
	
	t.Run("error execution", func(t *testing.T) {
		testErr := errors.New("test error")
		err := WithSpan(ctx, "test.error", func(ctx context.Context, span oteltrace.Span) error {
			return testErr
		})
		
		if err != testErr {
			t.Errorf("expected error %v, got %v", testErr, err)
		}
		
		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}
		
		// Check error was recorded
		if spans[0].Status.Code != codes.Error {
			t.Errorf("expected error status, got %v", spans[0].Status.Code)
		}
	})
}

func TestIsEnabled(t *testing.T) {
	t.Run("disabled when no provider", func(t *testing.T) {
		// Reset to no provider
		otel.SetTracerProvider(nil)
		tracerOnce = sync.Once{}
		
		if IsEnabled() {
			t.Error("expected IsEnabled to return false when no provider")
		}
	})
	
	t.Run("enabled when provider is set", func(t *testing.T) {
		_, cleanup := setupTestTracer()
		defer cleanup()
		
		if !IsEnabled() {
			t.Error("expected IsEnabled to return true when provider is set")
		}
	})
}

func TestNestedSpans(t *testing.T) {
	exporter, cleanup := setupTestTracer()
	defer cleanup()
	
	ctx := context.Background()
	
	// Start request span
	ctx, requestSpan := StartRequestSpan(ctx, RequestSpanOptions{
		Provider: "test",
		Model:    "test-model",
	})
	
	// Start step span within request
	stepCtx, stepSpan := StartStepSpan(ctx, StepSpanOptions{
		StepNumber: 1,
	})
	
	// Start tool span within step
	_, toolSpan := StartToolSpan(stepCtx, ToolSpanOptions{
		ToolName: "test_tool",
	})
	
	// End spans in reverse order
	toolSpan.End()
	stepSpan.End()
	requestSpan.End()
	
	spans := exporter.GetSpans()
	if len(spans) != 3 {
		t.Fatalf("expected 3 spans, got %d", len(spans))
	}
	
	// Verify span names
	names := []string{}
	for _, s := range spans {
		names = append(names, s.Name)
	}
	
	expectedNames := []string{"ai.tool.test_tool", "ai.step.1", "ai.request"}
	for _, expected := range expectedNames {
		found := false
		for _, name := range names {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find span named %s", expected)
		}
	}
}

// Helper function to check attribute values
func checkAttribute(t *testing.T, attrs []attribute.KeyValue, key string, expectedValue interface{}) {
	t.Helper()
	
	for _, attr := range attrs {
		if string(attr.Key) == key {
			switch expected := expectedValue.(type) {
			case string:
				if attr.Value.AsString() != expected {
					t.Errorf("attribute %s: expected %v, got %v", key, expected, attr.Value.AsString())
				}
			case int64:
				if attr.Value.AsInt64() != expected {
					t.Errorf("attribute %s: expected %v, got %v", key, expected, attr.Value.AsInt64())
				}
			case float64:
				val := attr.Value.AsFloat64()
				// Use tolerance for float comparison to handle float32 conversion
				const tolerance = 0.0000001
				diff := val - expected
				if diff < 0 {
					diff = -diff
				}
				if diff > tolerance {
					t.Errorf("attribute %s: expected %v, got %v", key, expected, val)
				}
			case bool:
				if attr.Value.AsBool() != expected {
					t.Errorf("attribute %s: expected %v, got %v", key, expected, attr.Value.AsBool())
				}
			case []string:
				slice := attr.Value.AsStringSlice()
				if len(slice) != len(expected) {
					t.Errorf("attribute %s: expected slice length %d, got %d", key, len(expected), len(slice))
					return
				}
				for i, v := range expected {
					if slice[i] != v {
						t.Errorf("attribute %s[%d]: expected %v, got %v", key, i, v, slice[i])
					}
				}
			default:
				t.Errorf("unsupported type for attribute %s: %T", key, expectedValue)
			}
			return
		}
	}
	
	t.Errorf("attribute %s not found", key)
}

func TestZeroOverheadWhenDisabled(t *testing.T) {
	// Reset to no provider
	otel.SetTracerProvider(nil)
	tracerOnce = sync.Once{}
	
	ctx := context.Background()
	
	// All these operations should be no-ops and very fast
	start := time.Now()
	
	for i := 0; i < 10000; i++ {
		_, span := StartRequestSpan(ctx, RequestSpanOptions{
			Provider: "test",
			Model:    "test",
		})
		RecordUsage(span, 100, 200, 300)
		RecordError(span, errors.New("test"), "test")
		span.End()
	}
	
	elapsed := time.Since(start)
	
	// Should complete very quickly when disabled (< 10ms for 10000 iterations)
	if elapsed > 10*time.Millisecond {
		t.Errorf("operations took too long when disabled: %v", elapsed)
	}
}