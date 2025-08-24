// This example demonstrates comprehensive observability features in the GAI framework,
// including distributed tracing, metrics collection, and usage accounting.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/recera/gai/obs"
	"github.com/recera/gai/tools"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func main() {
	// Initialize observability
	shutdown := initObservability()
	defer shutdown()

	// Run example scenarios
	ctx := context.Background()
	
	fmt.Println("=== Observability Example ===\n")
	
	// Example 1: Basic request with tracing
	fmt.Println("1. Basic Request with Tracing")
	basicRequestExample(ctx)
	
	// Example 2: Multi-step execution with tools
	fmt.Println("\n2. Multi-Step Execution with Tools")
	multiStepExample(ctx)
	
	// Example 3: Usage accounting and reporting
	fmt.Println("\n3. Usage Accounting")
	usageAccountingExample(ctx)
	
	// Example 4: Error tracking
	fmt.Println("\n4. Error Tracking")
	errorTrackingExample(ctx)
	
	// Example 5: Performance monitoring
	fmt.Println("\n5. Performance Monitoring")
	performanceExample(ctx)
	
	// Generate final report
	fmt.Println("\n=== Usage Report ===")
	generateUsageReport()
}

// initObservability sets up tracing and metrics
func initObservability() func() {
	// Create resource
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("gai-example"),
		semconv.ServiceVersion("1.0.0"),
	)
	
	// Setup tracing
	traceExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		log.Fatal(err)
	}
	
	tp := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	obs.SetGlobalTracerProvider(tp)
	
	// Setup metrics
	metricExporter, err := stdoutmetric.New()
	if err != nil {
		log.Fatal(err)
	}
	
	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter)),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	obs.SetGlobalMeterProvider(mp)
	
	// Return cleanup function
	return func() {
		ctx := context.Background()
		tp.Shutdown(ctx)
		mp.Shutdown(ctx)
	}
}

// basicRequestExample demonstrates basic request tracing
func basicRequestExample(ctx context.Context) {
	// Start request span
	ctx, span := obs.StartRequestSpan(ctx, obs.RequestSpanOptions{
		Provider:     "openai",
		Model:        "gpt-4",
		Temperature:  0.7,
		MaxTokens:    100,
		Stream:       false,
		MessageCount: 1,
	})
	defer span.End()
	
	// Simulate request processing
	time.Sleep(100 * time.Millisecond)
	
	// Record usage
	obs.RecordUsage(span, 50, 45, 95)
	
	// Record metrics
	obs.RecordRequest(ctx, "openai", "gpt-4", true, 100*time.Millisecond)
	obs.RecordTokens(ctx, "openai", "gpt-4", 50, 45)
	obs.RecordUsageData(ctx, "openai", "gpt-4", 50, 45)
	
	fmt.Println("  ✓ Request traced and metrics recorded")
}

// multiStepExample demonstrates multi-step execution with tools
func multiStepExample(ctx context.Context) {
	// Start request span
	ctx, requestSpan := obs.StartRequestSpan(ctx, obs.RequestSpanOptions{
		Provider:  "anthropic",
		Model:     "claude-3-sonnet",
		ToolCount: 2,
	})
	defer requestSpan.End()
	
	// Step 1: Initial response
	stepCtx, stepSpan := obs.StartStepSpan(ctx, obs.StepSpanOptions{
		StepNumber:   1,
		HasToolCalls: true,
		ToolCount:    2,
		TextLength:   150,
	})
	
	// Simulate tool calls
	tools := []string{"get_weather", "search_web"}
	for _, toolName := range tools {
		toolCtx, toolSpan := obs.StartToolSpan(stepCtx, obs.ToolSpanOptions{
			ToolName:   toolName,
			ToolID:     fmt.Sprintf("call_%s_1", toolName),
			InputSize:  100,
			StepNumber: 1,
			Parallel:   true,
			Timeout:    5 * time.Second,
		})
		
		// Simulate tool execution
		time.Sleep(50 * time.Millisecond)
		
		// Record tool result
		obs.RecordToolResult(toolSpan, true, 200, 50*time.Millisecond)
		obs.RecordToolExecution(toolCtx, toolName, true, 50*time.Millisecond)
		
		toolSpan.End()
	}
	
	stepSpan.End()
	
	// Step 2: Final response
	_, step2Span := obs.StartStepSpan(ctx, obs.StepSpanOptions{
		StepNumber:   2,
		HasToolCalls: false,
		TextLength:   300,
	})
	time.Sleep(75 * time.Millisecond)
	step2Span.End()
	
	// Record total usage
	obs.RecordUsage(requestSpan, 150, 200, 350)
	obs.RecordUsageData(ctx, "anthropic", "claude-3-sonnet", 150, 200)
	
	fmt.Println("  ✓ Multi-step execution with 2 tools traced")
}

// usageAccountingExample demonstrates usage tracking and cost estimation
func usageAccountingExample(ctx context.Context) {
	providers := []struct {
		name   string
		model  string
		input  int
		output int
	}{
		{"openai", "gpt-4", 1000, 1500},
		{"openai", "gpt-3.5-turbo", 500, 750},
		{"anthropic", "claude-3-opus", 2000, 2500},
		{"gemini", "gemini-1.5-pro", 800, 1200},
	}
	
	for _, p := range providers {
		// Record usage
		obs.RecordUsageData(ctx, p.name, p.model, p.input, p.output)
		
		// Estimate cost
		cost := obs.EstimateCost(p.model, p.input, p.output)
		formatted := obs.FormatCost(cost)
		
		fmt.Printf("  %s/%s: %d tokens, cost: %s\n", 
			p.name, p.model, p.input+p.output, formatted)
	}
}

// errorTrackingExample demonstrates error tracking
func errorTrackingExample(ctx context.Context) {
	// Simulate various error types
	errors := []struct {
		errorType string
		provider  string
		model     string
	}{
		{"rate_limited", "openai", "gpt-4"},
		{"timeout", "anthropic", "claude-3"},
		{"content_filtered", "openai", "gpt-4"},
		{"bad_request", "gemini", "gemini-pro"},
	}
	
	for _, e := range errors {
		// Record error metric
		obs.RecordErrorMetric(ctx, e.errorType, e.provider, e.model)
		
		// Also track in a span
		_, span := obs.StartRequestSpan(ctx, obs.RequestSpanOptions{
			Provider: e.provider,
			Model:    e.model,
		})
		
		// Record error on span
		err := fmt.Errorf("%s error", e.errorType)
		obs.RecordError(span, err, fmt.Sprintf("Simulated %s", e.errorType))
		
		span.End()
		
		fmt.Printf("  ✓ Tracked %s error for %s/%s\n", e.errorType, e.provider, e.model)
	}
}

// performanceExample demonstrates performance monitoring
func performanceExample(ctx context.Context) {
	// Monitor cache performance
	cacheAccesses := []bool{true, true, false, true, false, true, true, false}
	hits := 0
	for _, hit := range cacheAccesses {
		obs.RecordCacheHit(ctx, "prompt", hit)
		if hit {
			hits++
		}
	}
	hitRate := float64(hits) / float64(len(cacheAccesses)) * 100
	fmt.Printf("  Cache hit rate: %.1f%% (%d/%d)\n", hitRate, hits, len(cacheAccesses))
	
	// Monitor streaming performance
	streamEvents := []string{"start", "text_delta", "text_delta", "tool_call", "text_delta", "finish"}
	for _, event := range streamEvents {
		obs.RecordStreamEvent(ctx, event, "openai")
	}
	fmt.Printf("  Streamed %d events\n", len(streamEvents))
	
	// Monitor active requests
	obs.IncrementActiveRequests(ctx, "openai")
	obs.IncrementActiveRequests(ctx, "anthropic")
	obs.DecrementActiveRequests(ctx, "openai")
	fmt.Println("  Active requests tracked")
}

// generateUsageReport generates and displays a usage report
func generateUsageReport() {
	report := obs.GenerateReport()
	
	fmt.Printf("Period: %v\n", report.Period)
	fmt.Printf("Total Requests: %d\n", report.TotalRequests)
	fmt.Printf("Total Tokens: %d\n", report.TotalTokens)
	fmt.Printf("Total Cost: %s\n", report.TotalCost)
	
	for _, provider := range report.Providers {
		fmt.Printf("\n%s:\n", provider.Provider)
		fmt.Printf("  Requests: %d\n", provider.Requests)
		fmt.Printf("  Input Tokens: %d\n", provider.InputTokens)
		fmt.Printf("  Output Tokens: %d\n", provider.OutputTokens)
		fmt.Printf("  Cost: %s\n", provider.Cost)
		
		for _, model := range provider.Models {
			fmt.Printf("    %s: %d requests, %s\n",
				model.Model, model.Requests, model.Cost)
		}
	}
}

// Example of using observability with actual tools
func toolExample(ctx context.Context) {
	// Define a weather tool with automatic observability
	type WeatherInput struct {
		Location string `json:"location"`
	}
	
	type WeatherOutput struct {
		Temperature float64 `json:"temperature"`
		Conditions  string  `json:"conditions"`
	}
	
	weatherTool := tools.New[WeatherInput, WeatherOutput](
		"get_weather",
		"Get current weather for a location",
		func(ctx context.Context, in WeatherInput, meta tools.Meta) (WeatherOutput, error) {
			// Tool execution is automatically traced
			time.Sleep(100 * time.Millisecond)
			
			return WeatherOutput{
				Temperature: 72.5,
				Conditions:  "Sunny",
			}, nil
		},
	)
	
	// Execute tool (tracing happens automatically)
	input := `{"location": "San Francisco"}`
	result, err := weatherTool.Exec(ctx, []byte(input), tools.Meta{
		CallID:     "call_123",
		StepNumber: 1,
	})
	
	if err != nil {
		log.Printf("Tool error: %v", err)
	} else {
		fmt.Printf("Tool result: %+v\n", result)
	}
}

// Example of monitoring long-running operations
func longRunningExample(ctx context.Context) {
	// Use WithSpan for convenient span management
	err := obs.WithSpan(ctx, "long_operation", func(ctx context.Context, span oteltrace.Span) error {
		// Simulate phases of work
		phases := []string{"initialization", "processing", "finalization"}
		
		for i, phase := range phases {
			// Create child span for each phase
			phaseCtx, phaseSpan := obs.Tracer().Start(ctx, phase)
			
			// Simulate work
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
			
			// Record phase-specific metrics
			phaseSpan.SetAttributes(
				attribute.Int("phase.index", i),
				attribute.String("phase.name", phase),
			)
			
			phaseSpan.End()
			_ = phaseCtx // Use context if needed
		}
		
		return nil
	})
	
	if err != nil {
		log.Printf("Operation failed: %v", err)
	}
}