package obs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// BenchmarkTracingDisabled tests performance when tracing is disabled
func BenchmarkTracingDisabled(b *testing.B) {
	// Ensure tracing is disabled
	otel.SetTracerProvider(nil)
	tracerOnce = sync.Once{}
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, span := StartRequestSpan(ctx, RequestSpanOptions{
			Provider: "test",
			Model:    "test-model",
		})
		
		RecordUsage(span, 100, 200, 300)
		RecordError(span, errors.New("test"), "test error")
		RecordProviderLatency(span, time.Millisecond, nil)
		
		span.End()
	}
}

// BenchmarkTracingEnabled tests performance when tracing is enabled
func BenchmarkTracingEnabled(b *testing.B) {
	// Setup tracing
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	tracerOnce = sync.Once{}
	defer func() {
		otel.SetTracerProvider(nil)
		tracerOnce = sync.Once{}
	}()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, span := StartRequestSpan(ctx, RequestSpanOptions{
			Provider: "test",
			Model:    "test-model",
		})
		
		RecordUsage(span, 100, 200, 300)
		RecordError(span, errors.New("test"), "test error")
		RecordProviderLatency(span, time.Millisecond, nil)
		
		span.End()
	}
}

// BenchmarkMetricsDisabled tests performance when metrics are disabled
func BenchmarkMetricsDisabled(b *testing.B) {
	// Ensure metrics are disabled
	otel.SetMeterProvider(nil)
	meterOnce = sync.Once{}
	resetInstruments()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		RecordRequest(ctx, "test", "test-model", true, time.Millisecond)
		RecordTokens(ctx, "test", "test-model", 100, 200)
		RecordToolExecution(ctx, "test-tool", true, time.Millisecond)
		RecordErrorMetric(ctx, "test-error", "test", "test-model")
	}
}

// BenchmarkMetricsEnabled tests performance when metrics are enabled
func BenchmarkMetricsEnabled(b *testing.B) {
	// Setup metrics
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(
		metric.WithReader(reader),
	)
	otel.SetMeterProvider(provider)
	meterOnce = sync.Once{}
	resetInstruments()
	
	defer func() {
		otel.SetMeterProvider(nil)
		meterOnce = sync.Once{}
		resetInstruments()
	}()
	
	// Initialize instruments
	_ = Meter()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		RecordRequest(ctx, "test", "test-model", true, time.Millisecond)
		RecordTokens(ctx, "test", "test-model", 100, 200)
		RecordToolExecution(ctx, "test-tool", true, time.Millisecond)
		RecordErrorMetric(ctx, "test-error", "test", "test-model")
	}
}

// BenchmarkUsageCollection tests performance of usage collection
func BenchmarkUsageCollection(b *testing.B) {
	collector := NewUsageCollector(time.Hour)
	ctx := context.Background()
	
	usage := Usage{
		InputTokens:         100,
		OutputTokens:        200,
		TotalTokens:         300,
		EstimatedCostMicrocents: 50,
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		collector.Record(ctx, "openai", "gpt-4", usage)
	}
}

// BenchmarkCostEstimation tests performance of cost estimation
func BenchmarkCostEstimation(b *testing.B) {
	models := []string{
		"gpt-4", "gpt-4o", "gpt-3.5-turbo",
		"claude-3-opus", "claude-3-sonnet",
		"gemini-1.5-pro", "unknown-model",
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		model := models[i%len(models)]
		_ = EstimateCost(model, 1000, 1500)
	}
}

// BenchmarkNestedSpans tests performance with nested spans
func BenchmarkNestedSpans(b *testing.B) {
	// Setup tracing
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	tracerOnce = sync.Once{}
	defer func() {
		otel.SetTracerProvider(nil)
		tracerOnce = sync.Once{}
	}()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Request span
		ctx, requestSpan := StartRequestSpan(ctx, RequestSpanOptions{
			Provider: "test",
			Model:    "test-model",
		})
		
		// Step span
		stepCtx, stepSpan := StartStepSpan(ctx, StepSpanOptions{
			StepNumber: 1,
		})
		
		// Tool spans
		for j := 0; j < 3; j++ {
			_, toolSpan := StartToolSpan(stepCtx, ToolSpanOptions{
				ToolName: "test-tool",
			})
			RecordToolResult(toolSpan, true, 256, time.Millisecond)
			toolSpan.End()
		}
		
		stepSpan.End()
		requestSpan.End()
	}
}

// BenchmarkWithSpan tests performance of the WithSpan helper
func BenchmarkWithSpan(b *testing.B) {
	// Setup tracing
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	tracerOnce = sync.Once{}
	defer func() {
		otel.SetTracerProvider(nil)
		tracerOnce = sync.Once{}
	}()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = WithSpan(ctx, "test.operation", func(ctx context.Context, span oteltrace.Span) error {
			// Simulate some work
			RecordUsage(span, 100, 200, 300)
			return nil
		})
	}
}

// BenchmarkConcurrentTracing tests performance with concurrent spans
func BenchmarkConcurrentTracing(b *testing.B) {
	// Setup tracing
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	tracerOnce = sync.Once{}
	defer func() {
		otel.SetTracerProvider(nil)
		tracerOnce = sync.Once{}
	}()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			_, span := StartRequestSpan(ctx, RequestSpanOptions{
				Provider: "test",
				Model:    "test-model",
			})
			RecordUsage(span, 100, 200, 300)
			span.End()
		}
	})
}

// BenchmarkConcurrentMetrics tests performance with concurrent metrics
func BenchmarkConcurrentMetrics(b *testing.B) {
	// Setup metrics
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(
		metric.WithReader(reader),
	)
	otel.SetMeterProvider(provider)
	meterOnce = sync.Once{}
	resetInstruments()
	
	defer func() {
		otel.SetMeterProvider(nil)
		meterOnce = sync.Once{}
		resetInstruments()
	}()
	
	// Initialize instruments
	_ = Meter()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			RecordRequest(ctx, "test", "test-model", true, time.Millisecond)
			RecordTokens(ctx, "test", "test-model", 100, 200)
		}
	})
}

// BenchmarkStreamingMetrics tests performance of streaming metrics
func BenchmarkStreamingMetrics(b *testing.B) {
	// Setup tracing
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	tracerOnce = sync.Once{}
	defer func() {
		otel.SetTracerProvider(nil)
		tracerOnce = sync.Once{}
	}()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, span := StartStreamingSpan(ctx, StreamingSpanOptions{
			Provider:  "test",
			Model:     "test-model",
			EventType: "text",
		})
		
		// Simulate streaming events
		for j := 0; j < 10; j++ {
			RecordStreamingMetrics(span, j+1, (j+1)*100, time.Duration(j)*time.Millisecond)
		}
		
		span.End()
	}
}

// BenchmarkPromptSpans tests performance of prompt spans
func BenchmarkPromptSpans(b *testing.B) {
	// Setup tracing
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	tracerOnce = sync.Once{}
	defer func() {
		otel.SetTracerProvider(nil)
		tracerOnce = sync.Once{}
	}()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, span := StartPromptSpan(ctx, PromptSpanOptions{
			Name:        "assistant",
			Version:     "1.0.0",
			Fingerprint: "abc123def456",
			DataKeys:    []string{"user", "role", "context"},
			Override:    false,
			CacheHit:    i%2 == 0, // Alternate cache hits
		})
		span.End()
	}
}

// BenchmarkGenerateReport tests performance of report generation
func BenchmarkGenerateReport(b *testing.B) {
	// Setup collector with data
	collectorOnce = sync.Once{}
	globalCollector = NewUsageCollector(time.Hour)
	
	ctx := context.Background()
	
	// Populate with some data
	for i := 0; i < 100; i++ {
		provider := []string{"openai", "anthropic", "gemini"}[i%3]
		model := []string{"gpt-4", "claude-3", "gemini-pro"}[i%3]
		RecordUsageData(ctx, provider, model, 100+i, 200+i)
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = GenerateReport()
	}
}

// BenchmarkIsEnabled tests the overhead of checking if tracing is enabled
func BenchmarkIsEnabled(b *testing.B) {
	// Test with disabled tracing
	otel.SetTracerProvider(nil)
	tracerOnce = sync.Once{}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = IsEnabled()
	}
}

// Helper to reset instruments for testing
func resetInstruments() {
	requestCounter = nil
	requestDuration = nil
	tokenCounter = nil
	toolExecutionCounter = nil
	toolDuration = nil
	errorCounter = nil
	streamEventCounter = nil
	activeRequests = nil
	cacheHitRatio = nil
	promptRenderDuration = nil
}