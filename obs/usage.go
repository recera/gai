package obs

import (
	"context"
	"sync"
	"time"
)

// Usage represents token usage for an AI request
type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	// Additional fields for future extensions
	InputCharacters  int
	OutputCharacters int
	// Cost estimation (in microcents to avoid floating point)
	EstimatedCostMicrocents int64
}

// UsageCollector collects and aggregates usage metrics
type UsageCollector struct {
	mu       sync.RWMutex
	usage    map[string]*ProviderUsage // keyed by provider
	window   time.Duration
	lastReset time.Time
}

// ProviderUsage tracks usage for a specific provider
type ProviderUsage struct {
	Provider         string
	TotalRequests    int64
	TotalInputTokens int64
	TotalOutputTokens int64
	TotalCostMicrocents int64
	LastUpdated      time.Time
	
	// Per-model breakdown
	ModelUsage map[string]*ModelUsage
}

// ModelUsage tracks usage for a specific model
type ModelUsage struct {
	Model            string
	Requests         int64
	InputTokens      int64
	OutputTokens     int64
	CostMicrocents   int64
	LastUpdated      time.Time
}

// NewUsageCollector creates a new usage collector with the specified window
func NewUsageCollector(window time.Duration) *UsageCollector {
	return &UsageCollector{
		usage:     make(map[string]*ProviderUsage),
		window:    window,
		lastReset: time.Now(),
	}
}

// Record records usage for a request
func (c *UsageCollector) Record(ctx context.Context, provider, model string, usage Usage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check if we need to reset the window
	if time.Since(c.lastReset) > c.window {
		c.resetLocked()
	}
	
	// Get or create provider usage
	pu, exists := c.usage[provider]
	if !exists {
		pu = &ProviderUsage{
			Provider:   provider,
			ModelUsage: make(map[string]*ModelUsage),
		}
		c.usage[provider] = pu
	}
	
	// Update provider totals
	pu.TotalRequests++
	pu.TotalInputTokens += int64(usage.InputTokens)
	pu.TotalOutputTokens += int64(usage.OutputTokens)
	pu.TotalCostMicrocents += usage.EstimatedCostMicrocents
	pu.LastUpdated = time.Now()
	
	// Get or create model usage
	mu, exists := pu.ModelUsage[model]
	if !exists {
		mu = &ModelUsage{
			Model: model,
		}
		pu.ModelUsage[model] = mu
	}
	
	// Update model totals
	mu.Requests++
	mu.InputTokens += int64(usage.InputTokens)
	mu.OutputTokens += int64(usage.OutputTokens)
	mu.CostMicrocents += usage.EstimatedCostMicrocents
	mu.LastUpdated = time.Now()
}

// GetProviderUsage returns usage for a specific provider
func (c *UsageCollector) GetProviderUsage(provider string) *ProviderUsage {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if pu, exists := c.usage[provider]; exists {
		// Return a copy to avoid concurrent modification
		return c.copyProviderUsage(pu)
	}
	return nil
}

// GetAllUsage returns usage for all providers
func (c *UsageCollector) GetAllUsage() map[string]*ProviderUsage {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	result := make(map[string]*ProviderUsage, len(c.usage))
	for k, v := range c.usage {
		result[k] = c.copyProviderUsage(v)
	}
	return result
}

// Reset resets all usage counters
func (c *UsageCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resetLocked()
}

// resetLocked resets usage counters (must be called with lock held)
func (c *UsageCollector) resetLocked() {
	c.usage = make(map[string]*ProviderUsage)
	c.lastReset = time.Now()
}

// copyProviderUsage creates a deep copy of provider usage
func (c *UsageCollector) copyProviderUsage(pu *ProviderUsage) *ProviderUsage {
	result := &ProviderUsage{
		Provider:            pu.Provider,
		TotalRequests:       pu.TotalRequests,
		TotalInputTokens:    pu.TotalInputTokens,
		TotalOutputTokens:   pu.TotalOutputTokens,
		TotalCostMicrocents: pu.TotalCostMicrocents,
		LastUpdated:         pu.LastUpdated,
		ModelUsage:          make(map[string]*ModelUsage, len(pu.ModelUsage)),
	}
	
	for k, v := range pu.ModelUsage {
		result.ModelUsage[k] = &ModelUsage{
			Model:          v.Model,
			Requests:       v.Requests,
			InputTokens:    v.InputTokens,
			OutputTokens:    v.OutputTokens,
			CostMicrocents: v.CostMicrocents,
			LastUpdated:    v.LastUpdated,
		}
	}
	
	return result
}

// Cost estimation for common models (in microcents per 1K tokens)
var modelCosts = map[string]struct {
	InputCost  int64 // microcents per 1K input tokens
	OutputCost int64 // microcents per 1K output tokens
}{
	// OpenAI models
	"gpt-4":            {3000, 6000},   // $0.03/$0.06
	"gpt-4-turbo":      {1000, 3000},   // $0.01/$0.03
	"gpt-4o":           {500, 1500},    // $0.005/$0.015
	"gpt-4o-mini":      {15, 60},       // $0.00015/$0.0006
	"gpt-3.5-turbo":    {50, 150},      // $0.0005/$0.0015
	
	// Anthropic models
	"claude-3-opus":    {1500, 7500},   // $0.015/$0.075
	"claude-3-sonnet":  {300, 1500},    // $0.003/$0.015
	"claude-3-haiku":   {25, 125},      // $0.00025/$0.00125
	
	// Gemini models
	"gemini-1.5-pro":   {350, 1050},    // $0.0035/$0.0105
	"gemini-1.5-flash": {7, 21},        // $0.00007/$0.00021
	
	// Default for unknown models
	"default":          {100, 200},     // $0.001/$0.002
}

// EstimateCost estimates the cost of a request in microcents
func EstimateCost(model string, inputTokens, outputTokens int) int64 {
	costs, exists := modelCosts[model]
	if !exists {
		// Try to find a partial match
		for key, val := range modelCosts {
			if len(key) > 0 && len(model) >= len(key) && model[:len(key)] == key {
				costs = val
				break
			}
		}
		// If still not found, use default
		if !exists {
			costs = modelCosts["default"]
		}
	}
	
	// Calculate cost in microcents
	inputCost := (int64(inputTokens) * costs.InputCost) / 1000
	outputCost := (int64(outputTokens) * costs.OutputCost) / 1000
	
	return inputCost + outputCost
}

// FormatCost formats microcents as a dollar string
func FormatCost(microcents int64) string {
	dollars := float64(microcents) / 1000000.0
	if dollars < 0.01 {
		return "<$0.01"
	}
	return "$" + formatFloat(dollars, 2)
}

// formatFloat formats a float with the specified decimal places
func formatFloat(f float64, decimals int) string {
	format := "%." + string('0'+byte(decimals)) + "f"
	return sprintf(format, f)
}

// sprintf is a helper to avoid importing fmt
func sprintf(format string, args ...interface{}) string {
	// Simple implementation for our specific use case
	if format == "%.2f" && len(args) == 1 {
		if v, ok := args[0].(float64); ok {
			whole := int(v)
			frac := int((v - float64(whole)) * 100)
			if frac < 0 {
				frac = -frac
			}
			if frac < 10 {
				return string('0'+byte(whole/10)) + string('0'+byte(whole%10)) + ".0" + string('0'+byte(frac))
			}
			return string('0'+byte(whole/10)) + string('0'+byte(whole%10)) + "." + string('0'+byte(frac/10)) + string('0'+byte(frac%10))
		}
	}
	return ""
}

// Global usage collector instance
var (
	globalCollector *UsageCollector
	collectorOnce   sync.Once
)

// GlobalUsageCollector returns the global usage collector
func GlobalUsageCollector() *UsageCollector {
	collectorOnce.Do(func() {
		// Default to 1 hour window
		globalCollector = NewUsageCollector(time.Hour)
	})
	return globalCollector
}

// RecordUsageData is a convenience function to record usage to the global collector
func RecordUsageData(ctx context.Context, provider, model string, inputTokens, outputTokens int) {
	usage := Usage{
		InputTokens:         inputTokens,
		OutputTokens:        outputTokens,
		TotalTokens:         inputTokens + outputTokens,
		EstimatedCostMicrocents: EstimateCost(model, inputTokens, outputTokens),
	}
	GlobalUsageCollector().Record(ctx, provider, model, usage)
}

// UsageReport generates a summary report of usage
type UsageReport struct {
	Period       time.Duration
	TotalRequests int64
	TotalTokens  int64
	TotalCost    string
	Providers    []ProviderReport
}

// ProviderReport contains usage report for a provider
type ProviderReport struct {
	Provider     string
	Requests     int64
	InputTokens  int64
	OutputTokens int64
	Cost         string
	Models       []ModelReport
}

// ModelReport contains usage report for a model
type ModelReport struct {
	Model        string
	Requests     int64
	InputTokens  int64
	OutputTokens int64
	Cost         string
}

// GenerateReport generates a usage report
func GenerateReport() *UsageReport {
	collector := GlobalUsageCollector()
	allUsage := collector.GetAllUsage()
	
	report := &UsageReport{
		Period:    collector.window,
		Providers: make([]ProviderReport, 0, len(allUsage)),
	}
	
	for _, pu := range allUsage {
		providerReport := ProviderReport{
			Provider:     pu.Provider,
			Requests:     pu.TotalRequests,
			InputTokens:  pu.TotalInputTokens,
			OutputTokens: pu.TotalOutputTokens,
			Cost:         FormatCost(pu.TotalCostMicrocents),
			Models:       make([]ModelReport, 0, len(pu.ModelUsage)),
		}
		
		for _, mu := range pu.ModelUsage {
			modelReport := ModelReport{
				Model:        mu.Model,
				Requests:     mu.Requests,
				InputTokens:  mu.InputTokens,
				OutputTokens: mu.OutputTokens,
				Cost:         FormatCost(mu.CostMicrocents),
			}
			providerReport.Models = append(providerReport.Models, modelReport)
		}
		
		report.TotalRequests += pu.TotalRequests
		report.TotalTokens += pu.TotalInputTokens + pu.TotalOutputTokens
		report.Providers = append(report.Providers, providerReport)
	}
	
	// Calculate total cost
	var totalCost int64
	for _, pu := range allUsage {
		totalCost += pu.TotalCostMicrocents
	}
	report.TotalCost = FormatCost(totalCost)
	
	return report
}