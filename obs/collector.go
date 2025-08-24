package obs

import (
	"context"
	"time"
	
	"github.com/recera/gai/core"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Collector implements core.MetricsCollector with OpenTelemetry integration
type Collector struct {
	ctx             context.Context
	requestSpan     trace.Span
	provider        string
	model           string
	usageCollector  *UsageCollector
}

// NewCollector creates a new metrics collector
func NewCollector(ctx context.Context, provider, model string) *Collector {
	return &Collector{
		ctx:            ctx,
		provider:       provider,
		model:          model,
		usageCollector: GlobalUsageCollector(),
	}
}

// SetRequestSpan sets the parent request span
func (c *Collector) SetRequestSpan(span trace.Span) {
	c.requestSpan = span
}

// RecordStep records metrics for a step execution
func (c *Collector) RecordStep(step core.Step, duration time.Duration) {
	// Create a step span if we have a parent
	if c.requestSpan != nil {
		ctx := ContextWithSpan(c.ctx, c.requestSpan)
		stepCtx, stepSpan := StartStepSpan(ctx, StepSpanOptions{
			StepNumber:   step.StepNumber,
			HasToolCalls: len(step.ToolCalls) > 0,
			ToolCount:    len(step.ToolCalls),
			TextLength:   len(step.Text),
		})
		defer stepSpan.End()
		
		// Record step duration
		stepSpan.SetAttributes(
			attribute.Float64("step.duration_ms", float64(duration.Milliseconds())),
		)
		
		// Update context for metrics
		c.ctx = stepCtx
	}
	
	// Record metrics
	RecordRequest(c.ctx, c.provider, c.model, true, duration)
}

// RecordToolExecution records metrics for a tool execution
func (c *Collector) RecordToolExecution(name string, duration time.Duration, err error) {
	// Create a tool span if we have context
	toolCtx, toolSpan := StartToolSpan(c.ctx, ToolSpanOptions{
		ToolName: name,
		Timeout:  duration, // Using duration as timeout for now
	})
	defer toolSpan.End()
	
	success := err == nil
	RecordToolResult(toolSpan, success, 0, duration)
	
	if err != nil {
		RecordError(toolSpan, err, "Tool execution failed")
	}
	
	// Record metrics
	RecordToolExecution(toolCtx, name, success, duration)
}

// RecordTotalExecution records metrics for the total execution
func (c *Collector) RecordTotalExecution(steps int, duration time.Duration) {
	// Record total request metrics
	RecordRequest(c.ctx, c.provider, c.model, true, duration)
	
	// Set final attributes on request span if available
	if c.requestSpan != nil {
		c.requestSpan.SetAttributes(
			attribute.Int("request.total_steps", steps),
			attribute.Float64("request.total_duration_ms", float64(duration.Milliseconds())),
		)
	}
}

// RecordUsageMetrics records token usage metrics
func (c *Collector) RecordUsageMetrics(inputTokens, outputTokens int) {
	RecordUsageData(c.ctx, c.provider, c.model, inputTokens, outputTokens)
	RecordTokens(c.ctx, c.provider, c.model, inputTokens, outputTokens)
	
	// Add to span if available
	if c.requestSpan != nil {
		RecordUsage(c.requestSpan, inputTokens, outputTokens, inputTokens+outputTokens)
	}
}

// RecordStreamEvent records a streaming event
func (c *Collector) RecordStreamEvent(eventType string) {
	RecordStreamEvent(c.ctx, eventType, c.provider)
}

// StartRequest marks the beginning of a request
func (c *Collector) StartRequest() {
	IncrementActiveRequests(c.ctx, c.provider)
}

// EndRequest marks the end of a request
func (c *Collector) EndRequest() {
	DecrementActiveRequests(c.ctx, c.provider)
}

// GetUsage returns the current usage data for the provider
func (c *Collector) GetUsage() *ProviderUsage {
	if c.usageCollector != nil {
		return c.usageCollector.GetProviderUsage(c.provider)
	}
	return &ProviderUsage{
		Provider: c.provider,
	}
}


// IntegratedCollector provides a complete metrics collection solution
// that implements core.MetricsCollector and integrates with OpenTelemetry
type IntegratedCollector struct {
	*Collector
	startTime time.Time
	provider  string
	model     string
}

// NewIntegratedCollector creates a new integrated metrics collector
func NewIntegratedCollector(ctx context.Context, req core.Request) *IntegratedCollector {
	// Extract provider and model from request (these would normally come from provider config)
	provider := "unknown"
	model := req.Model
	if model == "" {
		model = "unknown"
	}
	
	// Start request span
	ctx, span := StartRequestSpan(ctx, RequestSpanOptions{
		Provider:     provider,
		Model:        model,
		Temperature:  req.Temperature,
		MaxTokens:    req.MaxTokens,
		Stream:       req.Stream,
		ToolCount:    len(req.Tools),
		MessageCount: len(req.Messages),
		SystemPrompt: hasSystemPrompt(req.Messages),
		ProviderOptions: req.ProviderOptions,
		Metadata:     req.Metadata,
	})
	
	collector := NewCollector(ctx, provider, model)
	collector.SetRequestSpan(span)
	collector.StartRequest()
	
	return &IntegratedCollector{
		Collector: collector,
		startTime: time.Now(),
		provider:  provider,
		model:     model,
	}
}

// Complete finalizes the metrics collection
func (ic *IntegratedCollector) Complete(success bool, usage *core.Usage, err error) {
	defer ic.EndRequest()
	
	// Record usage if available
	if usage != nil {
		ic.RecordUsageMetrics(usage.InputTokens, usage.OutputTokens)
	}
	
	// Record error if present
	if err != nil && ic.requestSpan != nil {
		RecordError(ic.requestSpan, err, "Request failed")
		RecordErrorMetric(ic.ctx, getErrorType(err), ic.provider, ic.model)
	}
	
	// Finalize request span
	if ic.requestSpan != nil {
		ic.requestSpan.End()
	}
	
	// Record final metrics
	metrics := &RequestMetrics{
		StartTime:    ic.startTime,
		Provider:     ic.provider,
		Model:        ic.model,
		Success:      success,
	}
	
	if usage != nil {
		metrics.InputTokens = usage.InputTokens
		metrics.OutputTokens = usage.OutputTokens
	}
	
	if err != nil {
		metrics.ErrorType = getErrorType(err)
	}
	
	metrics.Record(ic.ctx)
}

// hasSystemPrompt checks if messages contain a system prompt
func hasSystemPrompt(messages []core.Message) bool {
	for _, msg := range messages {
		if msg.Role == core.System {
			return true
		}
	}
	return false
}

// getErrorType determines the error type from an error
func getErrorType(err error) string {
	if err == nil {
		return ""
	}
	
	// Check if it's an AIError
	if aiErr, ok := err.(*core.AIError); ok {
		return string(aiErr.Code)
	}
	
	// Default to generic error
	return "unknown"
}