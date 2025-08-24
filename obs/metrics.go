package obs

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

var (
	// meter is the global meter instance
	meter metric.Meter
	// meterOnce ensures meter is initialized only once
	meterOnce sync.Once
	// noopMeter is used when metrics are disabled (initialized lazily)
	
	// Pre-created instruments for performance
	requestCounter       metric.Int64Counter
	requestDuration      metric.Float64Histogram
	tokenCounter         metric.Int64Counter
	toolExecutionCounter metric.Int64Counter
	toolDuration         metric.Float64Histogram
	errorCounter         metric.Int64Counter
	streamEventCounter   metric.Int64Counter
	activeRequests       metric.Int64UpDownCounter
	cacheHitRatio        metric.Float64Histogram
	promptRenderDuration metric.Float64Histogram
)

// Meter returns the configured meter or a noop meter if not configured.
func Meter() metric.Meter {
	meterOnce.Do(func() {
		provider := otel.GetMeterProvider()
		if provider == nil {
			// Create a noop meter when not configured
			noopProvider := noop.NewMeterProvider()
			meter = noopProvider.Meter("")
		} else {
			meter = provider.Meter(
				"github.com/recera/gai",
				metric.WithInstrumentationVersion("1.0.0"),
			)
			initializeInstruments()
		}
	})
	return meter
}

// initializeInstruments creates all metric instruments
func initializeInstruments() {
	var err error
	
	// Request metrics
	requestCounter, err = meter.Int64Counter(
		"ai.requests.total",
		metric.WithDescription("Total number of AI requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		// Log error but continue - metrics are non-critical
	}
	
	requestDuration, err = meter.Float64Histogram(
		"ai.request.duration",
		metric.WithDescription("Duration of AI requests in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		// Log error but continue
	}
	
	// Token metrics
	tokenCounter, err = meter.Int64Counter(
		"ai.tokens.total",
		metric.WithDescription("Total number of tokens processed"),
		metric.WithUnit("1"),
	)
	if err != nil {
		// Log error but continue
	}
	
	// Tool metrics
	toolExecutionCounter, err = meter.Int64Counter(
		"ai.tools.executions",
		metric.WithDescription("Total number of tool executions"),
		metric.WithUnit("1"),
	)
	if err != nil {
		// Log error but continue
	}
	
	toolDuration, err = meter.Float64Histogram(
		"ai.tool.duration",
		metric.WithDescription("Duration of tool executions in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		// Log error but continue
	}
	
	// Error metrics
	errorCounter, err = meter.Int64Counter(
		"ai.errors.total",
		metric.WithDescription("Total number of errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		// Log error but continue
	}
	
	// Streaming metrics
	streamEventCounter, err = meter.Int64Counter(
		"ai.stream.events",
		metric.WithDescription("Total number of stream events"),
		metric.WithUnit("1"),
	)
	if err != nil {
		// Log error but continue
	}
	
	// Active requests gauge
	activeRequests, err = meter.Int64UpDownCounter(
		"ai.requests.active",
		metric.WithDescription("Number of active AI requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		// Log error but continue
	}
	
	// Cache metrics
	cacheHitRatio, err = meter.Float64Histogram(
		"ai.cache.hit_ratio",
		metric.WithDescription("Cache hit ratio"),
		metric.WithUnit("1"),
	)
	if err != nil {
		// Log error but continue
	}
	
	// Prompt metrics
	promptRenderDuration, err = meter.Float64Histogram(
		"ai.prompt.render_duration",
		metric.WithDescription("Duration of prompt rendering in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		// Log error but continue
	}
}

// RecordRequest records metrics for an AI request
func RecordRequest(ctx context.Context, provider, model string, success bool, duration time.Duration) {
	if requestCounter == nil || requestDuration == nil {
		return // Metrics not initialized
	}
	
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.Bool("success", success),
	}
	
	requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	requestDuration.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attrs...))
}

// RecordTokens records token usage metrics
func RecordTokens(ctx context.Context, provider, model string, inputTokens, outputTokens int) {
	if tokenCounter == nil {
		return
	}
	
	if inputTokens > 0 {
		tokenCounter.Add(ctx, int64(inputTokens), metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
			attribute.String("type", "input"),
		))
	}
	
	if outputTokens > 0 {
		tokenCounter.Add(ctx, int64(outputTokens), metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
			attribute.String("type", "output"),
		))
	}
}

// RecordToolExecution records metrics for a tool execution
func RecordToolExecution(ctx context.Context, toolName string, success bool, duration time.Duration) {
	if toolExecutionCounter == nil || toolDuration == nil {
		return
	}
	
	attrs := []attribute.KeyValue{
		attribute.String("tool", toolName),
		attribute.Bool("success", success),
	}
	
	toolExecutionCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	toolDuration.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attrs...))
}

// RecordError records an error metric
func RecordErrorMetric(ctx context.Context, errorType, provider, model string) {
	if errorCounter == nil {
		return
	}
	
	errorCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("type", errorType),
		attribute.String("provider", provider),
		attribute.String("model", model),
	))
}

// RecordStreamEvent records a streaming event metric
func RecordStreamEvent(ctx context.Context, eventType, provider string) {
	if streamEventCounter == nil {
		return
	}
	
	streamEventCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("type", eventType),
		attribute.String("provider", provider),
	))
}

// IncrementActiveRequests increments the active requests counter
func IncrementActiveRequests(ctx context.Context, provider string) {
	if activeRequests == nil {
		return
	}
	
	activeRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("provider", provider),
	))
}

// DecrementActiveRequests decrements the active requests counter
func DecrementActiveRequests(ctx context.Context, provider string) {
	if activeRequests == nil {
		return
	}
	
	activeRequests.Add(ctx, -1, metric.WithAttributes(
		attribute.String("provider", provider),
	))
}

// RecordCacheHit records cache hit/miss metrics
func RecordCacheHit(ctx context.Context, cacheType string, hit bool) {
	if cacheHitRatio == nil {
		return
	}
	
	ratio := 0.0
	if hit {
		ratio = 1.0
	}
	
	cacheHitRatio.Record(ctx, ratio, metric.WithAttributes(
		attribute.String("type", cacheType),
	))
}

// RecordPromptRender records prompt rendering metrics
func RecordPromptRender(ctx context.Context, name, version string, cacheHit bool, duration time.Duration) {
	if promptRenderDuration == nil {
		return
	}
	
	promptRenderDuration.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(
		attribute.String("name", name),
		attribute.String("version", version),
		attribute.Bool("cache_hit", cacheHit),
	))
}

// RequestMetrics provides a convenient way to record all metrics for a request
type RequestMetrics struct {
	StartTime    time.Time
	Provider     string
	Model        string
	InputTokens  int
	OutputTokens int
	Success      bool
	ErrorType    string
}

// Record records all metrics for a request
func (m *RequestMetrics) Record(ctx context.Context) {
	duration := time.Since(m.StartTime)
	
	// Record request metrics
	RecordRequest(ctx, m.Provider, m.Model, m.Success, duration)
	
	// Record token metrics
	RecordTokens(ctx, m.Provider, m.Model, m.InputTokens, m.OutputTokens)
	
	// Record error if not successful
	if !m.Success && m.ErrorType != "" {
		RecordErrorMetric(ctx, m.ErrorType, m.Provider, m.Model)
	}
}

// ToolMetrics provides a convenient way to record all metrics for a tool
type ToolMetrics struct {
	StartTime time.Time
	ToolName  string
	Success   bool
}

// Record records all metrics for a tool execution
func (m *ToolMetrics) Record(ctx context.Context) {
	duration := time.Since(m.StartTime)
	RecordToolExecution(ctx, m.ToolName, m.Success, duration)
}

// SetGlobalMeterProvider sets the global meter provider
// This should be called once at application startup
func SetGlobalMeterProvider(provider metric.MeterProvider) {
	otel.SetMeterProvider(provider)
	// Reset the meter to pick up the new provider
	meterOnce = sync.Once{}
}