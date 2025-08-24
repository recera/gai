package obs

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// setupTestMeter creates an in-memory metric reader for testing
func setupTestMeter() (*metric.ManualReader, func()) {
	reader := metric.NewManualReader()
	
	provider := metric.NewMeterProvider(
		metric.WithReader(reader),
	)
	
	// Set as global provider
	otel.SetMeterProvider(provider)
	SetGlobalMeterProvider(provider)
	
	cleanup := func() {
		// Reset to noop
		otel.SetMeterProvider(noop.NewMeterProvider())
		meterOnce = sync.Once{}
		// Reset instruments
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
	
	return reader, cleanup
}

// collectMetrics collects metrics from the reader
func collectMetrics(t *testing.T, reader *metric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	
	var rm metricdata.ResourceMetrics
	err := reader.Collect(context.Background(), &rm)
	if err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}
	return rm
}

func TestMeter(t *testing.T) {
	t.Run("returns noop meter when not configured", func(t *testing.T) {
		// Reset meter
		meterOnce = sync.Once{}
		otel.SetMeterProvider(nil)
		
		meter := Meter()
		if meter == nil {
			t.Fatal("expected non-nil meter")
		}
	})
	
	t.Run("returns configured meter when provider is set", func(t *testing.T) {
		reader, cleanup := setupTestMeter()
		defer cleanup()
		
		meter := Meter()
		if meter == nil {
			t.Fatal("expected non-nil meter")
		}
		
		// Create a counter to verify it works
		counter, err := meter.Int64Counter("test.counter")
		if err != nil {
			t.Fatalf("failed to create counter: %v", err)
		}
		
		counter.Add(context.Background(), 1)
		
		// Collect metrics
		rm := collectMetrics(t, reader)
		if len(rm.ScopeMetrics) == 0 {
			t.Error("expected metrics to be recorded")
		}
	})
}

func TestRecordRequest(t *testing.T) {
	reader, cleanup := setupTestMeter()
	defer cleanup()
	
	// Initialize instruments
	_ = Meter()
	
	ctx := context.Background()
	RecordRequest(ctx, "openai", "gpt-4", true, 500*time.Millisecond)
	RecordRequest(ctx, "anthropic", "claude-3", false, 200*time.Millisecond)
	
	rm := collectMetrics(t, reader)
	
	// Find request metrics
	requestCountFound := false
	requestDurationFound := false
	
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch m.Name {
			case "ai.requests.total":
				requestCountFound = true
				// Should have 2 data points
				data, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Errorf("expected Sum[int64] for requests.total, got %T", m.Data)
					continue
				}
				
				totalCount := int64(0)
				for _, dp := range data.DataPoints {
					totalCount += dp.Value
				}
				
				if totalCount != 2 {
					t.Errorf("expected 2 total requests, got %d", totalCount)
				}
				
			case "ai.request.duration":
				requestDurationFound = true
				// Should have histogram data
				_, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					t.Errorf("expected Histogram[float64] for request.duration, got %T", m.Data)
				}
			}
		}
	}
	
	if !requestCountFound {
		t.Error("request count metric not found")
	}
	if !requestDurationFound {
		t.Error("request duration metric not found")
	}
}

func TestRecordTokens(t *testing.T) {
	reader, cleanup := setupTestMeter()
	defer cleanup()
	
	// Initialize instruments
	_ = Meter()
	
	ctx := context.Background()
	RecordTokens(ctx, "openai", "gpt-4", 100, 200)
	RecordTokens(ctx, "anthropic", "claude-3", 150, 250)
	
	rm := collectMetrics(t, reader)
	
	tokenCountFound := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "ai.tokens.total" {
				tokenCountFound = true
				
				data, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Errorf("expected Sum[int64] for tokens.total, got %T", m.Data)
					continue
				}
				
				// Calculate total tokens
				totalTokens := int64(0)
				for _, dp := range data.DataPoints {
					totalTokens += dp.Value
				}
				
				// Should be 100+200+150+250 = 700
				if totalTokens != 700 {
					t.Errorf("expected 700 total tokens, got %d", totalTokens)
				}
			}
		}
	}
	
	if !tokenCountFound {
		t.Error("token count metric not found")
	}
}

func TestRecordToolExecution(t *testing.T) {
	reader, cleanup := setupTestMeter()
	defer cleanup()
	
	// Initialize instruments
	_ = Meter()
	
	ctx := context.Background()
	RecordToolExecution(ctx, "get_weather", true, 100*time.Millisecond)
	RecordToolExecution(ctx, "search", false, 50*time.Millisecond)
	
	rm := collectMetrics(t, reader)
	
	toolCountFound := false
	toolDurationFound := false
	
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch m.Name {
			case "ai.tools.executions":
				toolCountFound = true
				data, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Errorf("expected Sum[int64] for tools.executions, got %T", m.Data)
					continue
				}
				
				totalCount := int64(0)
				for _, dp := range data.DataPoints {
					totalCount += dp.Value
				}
				
				if totalCount != 2 {
					t.Errorf("expected 2 tool executions, got %d", totalCount)
				}
				
			case "ai.tool.duration":
				toolDurationFound = true
				_, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					t.Errorf("expected Histogram[float64] for tool.duration, got %T", m.Data)
				}
			}
		}
	}
	
	if !toolCountFound {
		t.Error("tool execution count metric not found")
	}
	if !toolDurationFound {
		t.Error("tool duration metric not found")
	}
}

func TestRecordErrorMetric(t *testing.T) {
	reader, cleanup := setupTestMeter()
	defer cleanup()
	
	// Initialize instruments
	_ = Meter()
	
	ctx := context.Background()
	RecordErrorMetric(ctx, "rate_limited", "openai", "gpt-4")
	RecordErrorMetric(ctx, "timeout", "anthropic", "claude-3")
	
	rm := collectMetrics(t, reader)
	
	errorCountFound := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "ai.errors.total" {
				errorCountFound = true
				
				data, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Errorf("expected Sum[int64] for errors.total, got %T", m.Data)
					continue
				}
				
				totalErrors := int64(0)
				for _, dp := range data.DataPoints {
					totalErrors += dp.Value
				}
				
				if totalErrors != 2 {
					t.Errorf("expected 2 errors, got %d", totalErrors)
				}
			}
		}
	}
	
	if !errorCountFound {
		t.Error("error count metric not found")
	}
}

func TestActiveRequests(t *testing.T) {
	reader, cleanup := setupTestMeter()
	defer cleanup()
	
	// Initialize instruments
	_ = Meter()
	
	ctx := context.Background()
	
	// Increment active requests
	IncrementActiveRequests(ctx, "openai")
	IncrementActiveRequests(ctx, "openai")
	IncrementActiveRequests(ctx, "anthropic")
	
	// Decrement one
	DecrementActiveRequests(ctx, "openai")
	
	rm := collectMetrics(t, reader)
	
	activeFound := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "ai.requests.active" {
				activeFound = true
				
				data, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Errorf("expected Sum[int64] for requests.active, got %T", m.Data)
					continue
				}
				
				// Should have net 2 active (3 increments - 1 decrement)
				totalActive := int64(0)
				for _, dp := range data.DataPoints {
					totalActive += dp.Value
				}
				
				if totalActive != 2 {
					t.Errorf("expected 2 active requests, got %d", totalActive)
				}
			}
		}
	}
	
	if !activeFound {
		t.Error("active requests metric not found")
	}
}

func TestRecordCacheHit(t *testing.T) {
	reader, cleanup := setupTestMeter()
	defer cleanup()
	
	// Initialize instruments
	_ = Meter()
	
	ctx := context.Background()
	RecordCacheHit(ctx, "prompt", true)
	RecordCacheHit(ctx, "prompt", false)
	RecordCacheHit(ctx, "schema", true)
	
	rm := collectMetrics(t, reader)
	
	cacheFound := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "ai.cache.hit_ratio" {
				cacheFound = true
				
				data, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					t.Errorf("expected Histogram[float64] for cache.hit_ratio, got %T", m.Data)
					continue
				}
				
				// Should have 3 data points
				totalCount := 0
				for _, dp := range data.DataPoints {
					totalCount += int(dp.Count)
				}
				
				if totalCount != 3 {
					t.Errorf("expected 3 cache accesses, got %d", totalCount)
				}
			}
		}
	}
	
	if !cacheFound {
		t.Error("cache hit ratio metric not found")
	}
}

func TestRecordPromptRender(t *testing.T) {
	reader, cleanup := setupTestMeter()
	defer cleanup()
	
	// Initialize instruments
	_ = Meter()
	
	ctx := context.Background()
	RecordPromptRender(ctx, "assistant", "1.0.0", true, 5*time.Millisecond)
	RecordPromptRender(ctx, "assistant", "1.0.0", false, 15*time.Millisecond)
	
	rm := collectMetrics(t, reader)
	
	promptFound := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "ai.prompt.render_duration" {
				promptFound = true
				
				data, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					t.Errorf("expected Histogram[float64] for prompt.render_duration, got %T", m.Data)
					continue
				}
				
				// Should have 2 data points
				totalCount := 0
				for _, dp := range data.DataPoints {
					totalCount += int(dp.Count)
				}
				
				if totalCount != 2 {
					t.Errorf("expected 2 prompt renders, got %d", totalCount)
				}
			}
		}
	}
	
	if !promptFound {
		t.Error("prompt render duration metric not found")
	}
}

func TestRequestMetrics(t *testing.T) {
	reader, cleanup := setupTestMeter()
	defer cleanup()
	
	// Initialize instruments
	_ = Meter()
	
	ctx := context.Background()
	
	metrics := &RequestMetrics{
		StartTime:    time.Now().Add(-100 * time.Millisecond),
		Provider:     "openai",
		Model:        "gpt-4",
		InputTokens:  100,
		OutputTokens: 200,
		Success:      true,
	}
	
	metrics.Record(ctx)
	
	rm := collectMetrics(t, reader)
	
	// Should have recorded request, tokens
	foundRequest := false
	foundTokens := false
	
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch m.Name {
			case "ai.requests.total":
				foundRequest = true
			case "ai.tokens.total":
				foundTokens = true
			}
		}
	}
	
	if !foundRequest {
		t.Error("request metric not found")
	}
	if !foundTokens {
		t.Error("token metrics not found")
	}
}

func TestToolMetrics(t *testing.T) {
	reader, cleanup := setupTestMeter()
	defer cleanup()
	
	// Initialize instruments
	_ = Meter()
	
	ctx := context.Background()
	
	metrics := &ToolMetrics{
		StartTime: time.Now().Add(-50 * time.Millisecond),
		ToolName:  "get_weather",
		Success:   true,
	}
	
	metrics.Record(ctx)
	
	rm := collectMetrics(t, reader)
	
	foundTool := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "ai.tools.executions" {
				foundTool = true
			}
		}
	}
	
	if !foundTool {
		t.Error("tool execution metric not found")
	}
}

func TestZeroOverheadMetricsWhenDisabled(t *testing.T) {
	// Reset to no provider
	otel.SetMeterProvider(nil)
	meterOnce = sync.Once{}
	
	// Reset all instruments
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
	
	ctx := context.Background()
	
	// All these operations should be no-ops and very fast
	start := time.Now()
	
	for i := 0; i < 10000; i++ {
		RecordRequest(ctx, "test", "test", true, time.Millisecond)
		RecordTokens(ctx, "test", "test", 100, 200)
		RecordToolExecution(ctx, "test", true, time.Millisecond)
		RecordErrorMetric(ctx, "test", "test", "test")
		IncrementActiveRequests(ctx, "test")
		DecrementActiveRequests(ctx, "test")
		RecordCacheHit(ctx, "test", true)
		RecordPromptRender(ctx, "test", "1.0", true, time.Millisecond)
	}
	
	elapsed := time.Since(start)
	
	// Should complete very quickly when disabled (< 10ms for 10000 iterations)
	if elapsed > 10*time.Millisecond {
		t.Errorf("operations took too long when disabled: %v", elapsed)
	}
}