package obs

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewUsageCollector(t *testing.T) {
	collector := NewUsageCollector(time.Hour)
	
	if collector == nil {
		t.Fatal("expected non-nil collector")
	}
	
	if collector.window != time.Hour {
		t.Errorf("expected window of 1 hour, got %v", collector.window)
	}
	
	if len(collector.usage) != 0 {
		t.Errorf("expected empty usage map, got %d entries", len(collector.usage))
	}
}

func TestUsageCollectorRecord(t *testing.T) {
	collector := NewUsageCollector(time.Hour)
	ctx := context.Background()
	
	// Record some usage
	usage1 := Usage{
		InputTokens:         100,
		OutputTokens:        200,
		TotalTokens:         300,
		EstimatedCostMicrocents: 50,
	}
	
	collector.Record(ctx, "openai", "gpt-4", usage1)
	
	// Verify provider usage
	pu := collector.GetProviderUsage("openai")
	if pu == nil {
		t.Fatal("expected provider usage for openai")
	}
	
	if pu.TotalRequests != 1 {
		t.Errorf("expected 1 request, got %d", pu.TotalRequests)
	}
	
	if pu.TotalInputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", pu.TotalInputTokens)
	}
	
	if pu.TotalOutputTokens != 200 {
		t.Errorf("expected 200 output tokens, got %d", pu.TotalOutputTokens)
	}
	
	if pu.TotalCostMicrocents != 50 {
		t.Errorf("expected 50 microcents, got %d", pu.TotalCostMicrocents)
	}
	
	// Check model usage
	mu, exists := pu.ModelUsage["gpt-4"]
	if !exists {
		t.Fatal("expected model usage for gpt-4")
	}
	
	if mu.Requests != 1 {
		t.Errorf("expected 1 model request, got %d", mu.Requests)
	}
	
	if mu.InputTokens != 100 {
		t.Errorf("expected 100 model input tokens, got %d", mu.InputTokens)
	}
	
	// Record more usage for the same provider/model
	usage2 := Usage{
		InputTokens:         50,
		OutputTokens:        150,
		TotalTokens:         200,
		EstimatedCostMicrocents: 30,
	}
	
	collector.Record(ctx, "openai", "gpt-4", usage2)
	
	// Verify accumulated usage
	pu = collector.GetProviderUsage("openai")
	if pu.TotalRequests != 2 {
		t.Errorf("expected 2 requests, got %d", pu.TotalRequests)
	}
	
	if pu.TotalInputTokens != 150 {
		t.Errorf("expected 150 total input tokens, got %d", pu.TotalInputTokens)
	}
	
	if pu.TotalOutputTokens != 350 {
		t.Errorf("expected 350 total output tokens, got %d", pu.TotalOutputTokens)
	}
	
	if pu.TotalCostMicrocents != 80 {
		t.Errorf("expected 80 total microcents, got %d", pu.TotalCostMicrocents)
	}
}

func TestUsageCollectorMultipleProviders(t *testing.T) {
	collector := NewUsageCollector(time.Hour)
	ctx := context.Background()
	
	// Record usage for multiple providers
	collector.Record(ctx, "openai", "gpt-4", Usage{
		InputTokens:  100,
		OutputTokens: 200,
	})
	
	collector.Record(ctx, "anthropic", "claude-3", Usage{
		InputTokens:  150,
		OutputTokens: 250,
	})
	
	collector.Record(ctx, "openai", "gpt-3.5-turbo", Usage{
		InputTokens:  50,
		OutputTokens: 100,
	})
	
	// Get all usage
	allUsage := collector.GetAllUsage()
	
	if len(allUsage) != 2 {
		t.Errorf("expected 2 providers, got %d", len(allUsage))
	}
	
	// Check OpenAI
	openai, exists := allUsage["openai"]
	if !exists {
		t.Fatal("expected openai in usage")
	}
	
	if openai.TotalRequests != 2 {
		t.Errorf("expected 2 openai requests, got %d", openai.TotalRequests)
	}
	
	if len(openai.ModelUsage) != 2 {
		t.Errorf("expected 2 openai models, got %d", len(openai.ModelUsage))
	}
	
	// Check Anthropic
	anthropic, exists := allUsage["anthropic"]
	if !exists {
		t.Fatal("expected anthropic in usage")
	}
	
	if anthropic.TotalRequests != 1 {
		t.Errorf("expected 1 anthropic request, got %d", anthropic.TotalRequests)
	}
}

func TestUsageCollectorReset(t *testing.T) {
	collector := NewUsageCollector(time.Hour)
	ctx := context.Background()
	
	// Record some usage
	collector.Record(ctx, "openai", "gpt-4", Usage{
		InputTokens:  100,
		OutputTokens: 200,
	})
	
	// Verify usage exists
	allUsage := collector.GetAllUsage()
	if len(allUsage) != 1 {
		t.Errorf("expected 1 provider before reset, got %d", len(allUsage))
	}
	
	// Reset
	collector.Reset()
	
	// Verify usage is cleared
	allUsage = collector.GetAllUsage()
	if len(allUsage) != 0 {
		t.Errorf("expected 0 providers after reset, got %d", len(allUsage))
	}
}

func TestUsageCollectorWindowReset(t *testing.T) {
	// Use a very short window for testing
	collector := NewUsageCollector(10 * time.Millisecond)
	ctx := context.Background()
	
	// Record usage
	collector.Record(ctx, "openai", "gpt-4", Usage{
		InputTokens:  100,
		OutputTokens: 200,
	})
	
	// Verify usage exists
	pu := collector.GetProviderUsage("openai")
	if pu == nil || pu.TotalRequests != 1 {
		t.Error("expected usage before window expiry")
	}
	
	// Wait for window to expire
	time.Sleep(15 * time.Millisecond)
	
	// Record new usage (should trigger reset)
	collector.Record(ctx, "openai", "gpt-4", Usage{
		InputTokens:  50,
		OutputTokens: 100,
	})
	
	// Verify only new usage exists
	pu = collector.GetProviderUsage("openai")
	if pu == nil {
		t.Fatal("expected provider usage after window reset")
	}
	
	if pu.TotalRequests != 1 {
		t.Errorf("expected 1 request after reset, got %d", pu.TotalRequests)
	}
	
	if pu.TotalInputTokens != 50 {
		t.Errorf("expected 50 input tokens after reset, got %d", pu.TotalInputTokens)
	}
}

func TestEstimateCost(t *testing.T) {
	tests := []struct {
		model        string
		inputTokens  int
		outputTokens int
		expectedCost int64
	}{
		// OpenAI models
		{"gpt-4", 1000, 1000, 3000 + 6000},         // $0.03 + $0.06 = $0.09 = 90000 microcents
		{"gpt-4o", 1000, 1000, 500 + 1500},         // $0.005 + $0.015 = $0.02 = 20000 microcents
		{"gpt-4o-mini", 1000, 1000, 15 + 60},       // $0.00015 + $0.0006 = $0.00075 = 750 microcents
		{"gpt-3.5-turbo", 1000, 1000, 50 + 150},    // $0.0005 + $0.0015 = $0.002 = 2000 microcents
		
		// Anthropic models
		{"claude-3-opus", 1000, 1000, 1500 + 7500}, // $0.015 + $0.075 = $0.09 = 90000 microcents
		{"claude-3-sonnet", 1000, 1000, 300 + 1500}, // $0.003 + $0.015 = $0.018 = 18000 microcents
		{"claude-3-haiku", 1000, 1000, 25 + 125},    // $0.00025 + $0.00125 = $0.0015 = 1500 microcents
		
		// Gemini models
		{"gemini-1.5-pro", 1000, 1000, 350 + 1050},  // $0.0035 + $0.0105 = $0.014 = 14000 microcents
		{"gemini-1.5-flash", 1000, 1000, 7 + 21},    // $0.00007 + $0.00021 = $0.00028 = 280 microcents
		
		// Unknown model (should use default)
		{"unknown-model", 1000, 1000, 100 + 200},    // $0.001 + $0.002 = $0.003 = 3000 microcents
		
		// Partial tokens
		{"gpt-4", 500, 750, 1500 + 4500},            // Half and 3/4 of the per-1K rates
	}
	
	for _, test := range tests {
		t.Run(test.model, func(t *testing.T) {
			cost := EstimateCost(test.model, test.inputTokens, test.outputTokens)
			if cost != test.expectedCost {
				t.Errorf("expected cost %d microcents for %s, got %d", test.expectedCost, test.model, cost)
			}
		})
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		microcents int64
		expected   string
	}{
		{0, "<$0.01"},
		{5000, "<$0.01"},      // $0.005
		{10000, "$0.01"},      // $0.01
		{100000, "$0.10"},     // $0.10
		{1000000, "$1.00"},    // $1.00
		{12340000, "$12.34"},  // $12.34
		{99990000, "$99.99"},  // $99.99
	}
	
	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			formatted := FormatCost(test.microcents)
			if formatted != test.expected {
				t.Errorf("expected %s for %d microcents, got %s", test.expected, test.microcents, formatted)
			}
		})
	}
}

func TestGlobalUsageCollector(t *testing.T) {
	// Reset global collector
	collectorOnce = sync.Once{}
	globalCollector = nil
	
	collector1 := GlobalUsageCollector()
	collector2 := GlobalUsageCollector()
	
	if collector1 != collector2 {
		t.Error("expected same global collector instance")
	}
	
	if collector1 == nil {
		t.Fatal("expected non-nil global collector")
	}
}

func TestRecordUsageDataConvenience(t *testing.T) {
	// Reset global collector
	collectorOnce = sync.Once{}
	globalCollector = nil
	
	ctx := context.Background()
	
	// Record usage using convenience function
	RecordUsageData(ctx, "openai", "gpt-4", 100, 200)
	
	// Verify it was recorded
	collector := GlobalUsageCollector()
	pu := collector.GetProviderUsage("openai")
	
	if pu == nil {
		t.Fatal("expected provider usage")
	}
	
	if pu.TotalInputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", pu.TotalInputTokens)
	}
	
	if pu.TotalOutputTokens != 200 {
		t.Errorf("expected 200 output tokens, got %d", pu.TotalOutputTokens)
	}
	
	// Cost should be estimated
	expectedCost := EstimateCost("gpt-4", 100, 200)
	if pu.TotalCostMicrocents != expectedCost {
		t.Errorf("expected cost %d, got %d", expectedCost, pu.TotalCostMicrocents)
	}
}

func TestGenerateReport(t *testing.T) {
	// Reset and setup global collector
	collectorOnce = sync.Once{}
	globalCollector = NewUsageCollector(time.Hour)
	
	ctx := context.Background()
	
	// Record various usage
	RecordUsageData(ctx, "openai", "gpt-4", 1000, 2000)
	RecordUsageData(ctx, "openai", "gpt-3.5-turbo", 500, 1000)
	RecordUsageData(ctx, "anthropic", "claude-3-opus", 1500, 2500)
	RecordUsageData(ctx, "gemini", "gemini-1.5-pro", 800, 1200)
	
	// Generate report
	report := GenerateReport()
	
	if report == nil {
		t.Fatal("expected non-nil report")
	}
	
	if report.Period != time.Hour {
		t.Errorf("expected 1 hour period, got %v", report.Period)
	}
	
	if report.TotalRequests != 4 {
		t.Errorf("expected 4 total requests, got %d", report.TotalRequests)
	}
	
	// Total tokens: 1000+2000 + 500+1000 + 1500+2500 + 800+1200 = 10500
	if report.TotalTokens != 10500 {
		t.Errorf("expected 10500 total tokens, got %d", report.TotalTokens)
	}
	
	if len(report.Providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(report.Providers))
	}
	
	// Find OpenAI provider report
	var openaiReport *ProviderReport
	for i := range report.Providers {
		if report.Providers[i].Provider == "openai" {
			openaiReport = &report.Providers[i]
			break
		}
	}
	
	if openaiReport == nil {
		t.Fatal("expected openai in report")
	}
	
	if openaiReport.Requests != 2 {
		t.Errorf("expected 2 openai requests, got %d", openaiReport.Requests)
	}
	
	if len(openaiReport.Models) != 2 {
		t.Errorf("expected 2 openai models, got %d", len(openaiReport.Models))
	}
	
	// Check cost formatting
	if report.TotalCost == "" {
		t.Error("expected non-empty total cost")
	}
	
	if report.TotalCost[0] != '$' && report.TotalCost[0] != '<' {
		t.Errorf("expected cost to start with $ or <, got %s", report.TotalCost)
	}
}

func TestCopyProviderUsage(t *testing.T) {
	collector := NewUsageCollector(time.Hour)
	ctx := context.Background()
	
	// Record usage
	collector.Record(ctx, "openai", "gpt-4", Usage{
		InputTokens:  100,
		OutputTokens: 200,
	})
	
	// Get provider usage (should be a copy)
	pu1 := collector.GetProviderUsage("openai")
	pu2 := collector.GetProviderUsage("openai")
	
	// Verify they are different instances
	if pu1 == pu2 {
		t.Error("expected different instances (copies)")
	}
	
	// But have same values
	if pu1.TotalRequests != pu2.TotalRequests {
		t.Error("expected same values in copies")
	}
	
	// Modifying one shouldn't affect the other
	pu1.TotalRequests = 999
	if pu2.TotalRequests == 999 {
		t.Error("modifying copy affected other copy")
	}
	
	// Original should be unchanged
	pu3 := collector.GetProviderUsage("openai")
	if pu3.TotalRequests != 1 {
		t.Error("original data was modified")
	}
}

func TestConcurrentUsageRecording(t *testing.T) {
	collector := NewUsageCollector(time.Hour)
	ctx := context.Background()
	
	// Record usage concurrently
	const goroutines = 10
	const recordsPerGoroutine = 100
	
	done := make(chan bool, goroutines)
	
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < recordsPerGoroutine; j++ {
				provider := "provider" + string(rune('0'+id%3))
				model := "model" + string(rune('0'+j%2))
				
				collector.Record(ctx, provider, model, Usage{
					InputTokens:  10,
					OutputTokens: 20,
				})
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}
	
	// Verify total requests
	allUsage := collector.GetAllUsage()
	totalRequests := int64(0)
	for _, pu := range allUsage {
		totalRequests += pu.TotalRequests
	}
	
	expectedRequests := int64(goroutines * recordsPerGoroutine)
	if totalRequests != expectedRequests {
		t.Errorf("expected %d total requests, got %d", expectedRequests, totalRequests)
	}
}