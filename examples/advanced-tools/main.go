// Package main demonstrates advanced tool usage with stopWhen conditions
// comparing GPT-5-mini with moonshotai/kimi-k2-instruct on Groq.
// This example showcases the GAI framework's capabilities for complex,
// multi-step AI agent workflows with sophisticated stopping conditions.
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/middleware"
	"github.com/recera/gai/providers/openai"
	"github.com/recera/gai/providers/openai_compat"
	"github.com/recera/gai/tools"
)

// WeatherData represents comprehensive weather information
type WeatherData struct {
	Location      string  `json:"location" jsonschema:"required,description=City name or coordinates"`
	Temperature   float64 `json:"temperature_celsius" jsonschema:"description=Temperature in Celsius"`
	Humidity      int     `json:"humidity_percent" jsonschema:"description=Humidity percentage"`
	WindSpeed     float64 `json:"wind_speed_kmh" jsonschema:"description=Wind speed in km/h"`
	Condition     string  `json:"condition" jsonschema:"description=Weather condition (sunny, cloudy, rainy, etc.)"`
	Pressure      float64 `json:"pressure_hpa" jsonschema:"description=Atmospheric pressure in hPa"`
	UV_Index      int     `json:"uv_index" jsonschema:"description=UV index (0-11)"`
	Visibility    float64 `json:"visibility_km" jsonschema:"description=Visibility in kilometers"`
	DewPoint      float64 `json:"dew_point_celsius" jsonschema:"description=Dew point in Celsius"`
	FeelsLike     float64 `json:"feels_like_celsius" jsonschema:"description=Feels like temperature in Celsius"`
	LastUpdated   string  `json:"last_updated" jsonschema:"description=Last update timestamp"`
}

// WeatherAnalysisInput for the weather analyzer tool
type WeatherAnalysisInput struct {
	Locations         []string `json:"locations" jsonschema:"required,description=List of cities to analyze"`
	AnalysisType      string   `json:"analysis_type" jsonschema:"enum=current,enum=comparison,enum=trend,description=Type of analysis to perform"`
	IncludeRecommendations bool `json:"include_recommendations,omitempty" jsonschema:"description=Whether to include travel/activity recommendations"`
}

// WeatherAnalysisOutput comprehensive weather analysis result
type WeatherAnalysisOutput struct {
	Analysis          string                 `json:"analysis" jsonschema:"description=Detailed weather analysis"`
	LocationData      map[string]WeatherData `json:"location_data" jsonschema:"description=Weather data for each location"`
	ComparisonSummary string                 `json:"comparison_summary,omitempty" jsonschema:"description=Summary comparing locations"`
	Recommendations   []string               `json:"recommendations,omitempty" jsonschema:"description=Activity or travel recommendations"`
	AlertLevel        string                 `json:"alert_level" jsonschema:"enum=low,enum=medium,enum=high,description=Weather alert level"`
	ProcessedAt       string                 `json:"processed_at" jsonschema:"description=Analysis timestamp"`
}

// DataProcessingInput for complex data analysis
type DataProcessingInput struct {
	DataType    string                 `json:"data_type" jsonschema:"enum=numerical,enum=weather,enum=statistical,description=Type of data to process"`
	Data        map[string]interface{} `json:"data" jsonschema:"required,description=Raw data to process"`
	Operations  []string               `json:"operations" jsonschema:"description=List of operations to perform"`
	OutputFormat string                `json:"output_format,omitempty" jsonschema:"enum=summary,enum=detailed,enum=chart,description=Output format preference"`
}

// DataProcessingOutput comprehensive data analysis result
type DataProcessingOutput struct {
	ProcessedData   map[string]interface{} `json:"processed_data" jsonschema:"description=Processed and analyzed data"`
	Summary         string                 `json:"summary" jsonschema:"description=Human-readable summary of results"`
	Statistics      map[string]float64     `json:"statistics,omitempty" jsonschema:"description=Calculated statistical measures"`
	Insights        []string               `json:"insights" jsonschema:"description=Key insights from the analysis"`
	Recommendations []string               `json:"recommendations,omitempty" jsonschema:"description=Actionable recommendations"`
	Confidence      float64                `json:"confidence" jsonschema:"description=Confidence level in results (0-1)"`
	ProcessedAt     string                 `json:"processed_at" jsonschema:"description=Processing timestamp"`
}

func main() {
	fmt.Println("🚀 Advanced AI Agent Examples with GPT-5-mini vs Kimi-K2-Instruct")
	fmt.Println(strings.Repeat("=", 80))
	
	// Validate environment
	if !validateEnvironment() {
		log.Fatal("❌ Environment validation failed. Please check your API keys.")
	}
	
	fmt.Println("✅ Environment validated successfully")
	fmt.Println()

	// Create advanced tools
	weatherTool, dataProcessorTool := createAdvancedTools()
	
	// Create providers
	openAIProvider := createOpenAIProvider()
	groqProvider := createGroqProvider()
	
	if openAIProvider == nil || groqProvider == nil {
		log.Fatal("❌ Failed to create providers")
	}

	// Run comprehensive examples
	ctx := context.Background()
	
	// Example scenarios with different stopWhen conditions
	scenarios := []Scenario{
		{
			Name: "🎯 Research & Analysis Workflow",
			Description: "Multi-step research task with complex data analysis",
			Query: "Analyze the weather in Tokyo, London, and Sydney. Then process the temperature data to identify patterns and provide travel recommendations for someone planning a world tour.",
			StopCondition: core.MaxSteps(4),
			ExpectedSteps: 3,
		},
		{
			Name: "🔄 Iterative Problem Solving",
			Description: "Let the AI decide when to stop based on tool usage",
			Query: "Compare weather conditions between Miami and Dubai for beach vacation planning, then analyze which location offers better conditions for water sports.",
			StopCondition: core.NoMoreTools(),
			ExpectedSteps: 2,
		},
		{
			Name: "🎯 Target-Oriented Execution",
			Description: "Stop after seeing the data processor tool",
			Query: "Get weather data for Paris and Rome, then analyze the data to determine which city has more stable conditions for outdoor photography.",
			StopCondition: core.UntilToolSeen("advanced_data_processor"),
			ExpectedSteps: 2,
		},
		{
			Name: "🔗 Combined Conditions",
			Description: "Multiple stopping conditions combined",
			Query: "Research weather patterns in New York, Los Angeles, and Chicago. Analyze the data for business travel optimization. Provide detailed recommendations.",
			StopCondition: core.CombineConditions(
				core.MaxSteps(5),
				core.UntilToolSeen("advanced_data_processor"),
			),
			ExpectedSteps: 3,
		},
	}

	// Run scenarios for both providers
	for i, scenario := range scenarios {
		fmt.Printf("\n%s\n", scenario.Name)
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("Description: %s\n", scenario.Description)
		fmt.Printf("Expected Steps: ~%d\n\n", scenario.ExpectedSteps)

		// Test with GPT-5-mini
		fmt.Println("🤖 Testing with GPT-5-mini (OpenAI)")
		fmt.Println(strings.Repeat("-", 40))
		gpt5Result := runScenario(ctx, openAIProvider, scenario, weatherTool, dataProcessorTool)
		
		// Test with Kimi-K2-Instruct
		fmt.Println("\n🚀 Testing with Kimi-K2-Instruct (Groq)")
		fmt.Println(strings.Repeat("-", 40))
		kimiResult := runScenario(ctx, groqProvider, scenario, weatherTool, dataProcessorTool)
		
		// Compare results
		fmt.Println("\n📊 Performance Comparison")
		fmt.Println(strings.Repeat("-", 30))
		compareResults(gpt5Result, kimiResult)
		
		if i < len(scenarios)-1 {
			fmt.Printf("\n%s\n", strings.Repeat("•", 80))
		}
	}
	
	fmt.Println("\n🎉 All examples completed successfully!")
	fmt.Println("This demonstrates the GAI framework's advanced capabilities for")
	fmt.Println("complex AI agent workflows with sophisticated stopping conditions.")
}

// Scenario represents a test scenario
type Scenario struct {
	Name          string
	Description   string
	Query         string
	StopCondition core.StopCondition
	ExpectedSteps int
}

// TestResult holds the results of a scenario execution
type TestResult struct {
	Provider      string
	Duration      time.Duration
	StepsExecuted int
	ToolsCalled   int
	TokensUsed    core.Usage
	Success       bool
	FinalText     string
	Error         error
}

func validateEnvironment() bool {
	requiredKeys := []string{"OPENAI_API_KEY", "GROQ_API_KEY"}
	for _, key := range requiredKeys {
		if os.Getenv(key) == "" {
			fmt.Printf("❌ Missing environment variable: %s\n", key)
			return false
		}
	}
	return true
}

func createAdvancedTools() (tools.Handle, tools.Handle) {
	// Advanced Weather Analysis Tool
	weatherTool := tools.New[WeatherAnalysisInput, WeatherAnalysisOutput](
		"advanced_weather_analyzer",
		"Comprehensive weather analysis tool that provides detailed weather data, comparisons, and recommendations for multiple locations",
		func(ctx context.Context, input WeatherAnalysisInput, meta tools.Meta) (WeatherAnalysisOutput, error) {
			fmt.Printf("🌦️  [Tool Execution] Weather Analysis: %s for %v\n", 
				input.AnalysisType, input.Locations)
			
			// Simulate comprehensive weather data collection
			locationData := make(map[string]WeatherData)
			var analysis strings.Builder
			var recommendations []string
			alertLevel := "low"
			
			analysis.WriteString(fmt.Sprintf("Comprehensive %s weather analysis for %d locations:\n\n", 
				input.AnalysisType, len(input.Locations)))
			
			for _, location := range input.Locations {
				// Simulate realistic weather data with some variation
				temp := 15 + (float64(len(location)%20) * 1.5) // Deterministic but varied
				humidity := 40 + (len(location)%40)
				windSpeed := 5 + (float64(len(location)%15) * 0.8)
				pressure := 1013 + (float64(len(location)%20) - 10)
				uvIndex := int(temp/5)
				if uvIndex > 11 { uvIndex = 11 }
				
				conditions := []string{"Sunny", "Partly Cloudy", "Cloudy", "Light Rain", "Clear"}
				condition := conditions[len(location)%len(conditions)]
				
				if temp > 35 || windSpeed > 20 {
					alertLevel = "high"
				} else if temp > 28 || windSpeed > 15 {
					alertLevel = "medium"
				}
				
				weatherData := WeatherData{
					Location:    location,
					Temperature: temp,
					Humidity:    humidity,
					WindSpeed:   windSpeed,
					Condition:   condition,
					Pressure:    pressure,
					UV_Index:    uvIndex,
					Visibility:  15 + (float64(len(location)%10)),
					DewPoint:    temp - 5,
					FeelsLike:   temp + (float64(humidity-50)/10),
					LastUpdated: time.Now().Format("2006-01-02 15:04:05"),
				}
				
				locationData[location] = weatherData
				
				analysis.WriteString(fmt.Sprintf("📍 %s: %s, %.1f°C (feels like %.1f°C)\n", 
					location, condition, temp, weatherData.FeelsLike))
				analysis.WriteString(fmt.Sprintf("   Humidity: %d%%, Wind: %.1f km/h, UV: %d\n\n", 
					humidity, windSpeed, uvIndex))
				
				// Generate recommendations if requested
				if input.IncludeRecommendations {
					if temp > 25 && condition == "Sunny" {
						recommendations = append(recommendations, 
							fmt.Sprintf("%s: Perfect for outdoor activities, don't forget sunscreen (UV: %d)", location, uvIndex))
					} else if condition == "Light Rain" {
						recommendations = append(recommendations, 
							fmt.Sprintf("%s: Indoor activities recommended, or bring an umbrella", location))
					} else {
						recommendations = append(recommendations, 
							fmt.Sprintf("%s: Good for sightseeing, dress for %.1f°C", location, temp))
					}
				}
			}
			
			// Add comparison summary for multiple locations
			var comparisonSummary string
			if len(input.Locations) > 1 {
				warmest := ""
				coldest := ""
				highestTemp := -100.0
				lowestTemp := 100.0
				
				for _, data := range locationData {
					if data.Temperature > highestTemp {
						highestTemp = data.Temperature
						warmest = data.Location
					}
					if data.Temperature < lowestTemp {
						lowestTemp = data.Temperature
						coldest = data.Location
					}
				}
				
				comparisonSummary = fmt.Sprintf("Temperature range: %s (%.1f°C) to %s (%.1f°C). " +
					"Temperature difference: %.1f°C", 
					coldest, lowestTemp, warmest, highestTemp, highestTemp-lowestTemp)
			}
			
			result := WeatherAnalysisOutput{
				Analysis:          analysis.String(),
				LocationData:      locationData,
				ComparisonSummary: comparisonSummary,
				Recommendations:   recommendations,
				AlertLevel:        alertLevel,
				ProcessedAt:       time.Now().Format("2006-01-02 15:04:05"),
			}
			
			// Add small delay to simulate real API calls
			time.Sleep(300 * time.Millisecond)
			
			return result, nil
		},
	)

	// Advanced Data Processing Tool
	dataProcessorTool := tools.New[DataProcessingInput, DataProcessingOutput](
		"advanced_data_processor",
		"Sophisticated data analysis tool that processes various data types, performs statistical analysis, and generates insights and recommendations",
		func(ctx context.Context, input DataProcessingInput, meta tools.Meta) (DataProcessingOutput, error) {
			fmt.Printf("📊 [Tool Execution] Data Processing: %s format, %d operations\n", 
				input.DataType, len(input.Operations))
			
			processedData := make(map[string]interface{})
			statistics := make(map[string]float64)
			var insights []string
			var recommendations []string
			
			// Process based on data type
			switch input.DataType {
			case "weather":
				insights = append(insights, "Weather data processed successfully")
				if temps, ok := input.Data["temperatures"].([]interface{}); ok {
					var tempValues []float64
					for _, t := range temps {
						if tf, ok := t.(float64); ok {
							tempValues = append(tempValues, tf)
						}
					}
					
					if len(tempValues) > 0 {
						mean := calculateMean(tempValues)
						std := calculateStdDev(tempValues, mean)
						min, max := minMax(tempValues)
						
						statistics["mean_temperature"] = mean
						statistics["std_deviation"] = std
						statistics["min_temperature"] = min
						statistics["max_temperature"] = max
						statistics["temperature_range"] = max - min
						
						processedData["temperature_analysis"] = map[string]interface{}{
							"average": mean,
							"range":   max - min,
							"stability": func() string {
								if std < 5 {
									return "stable"
								} else if std < 10 {
									return "moderate"
								}
								return "variable"
							}(),
						}
						
						insights = append(insights, fmt.Sprintf("Average temperature: %.1f°C", mean))
						insights = append(insights, fmt.Sprintf("Temperature variation: %.1f°C (std dev)", std))
						
						if std < 5 {
							insights = append(insights, "Weather patterns are very stable")
							recommendations = append(recommendations, "Excellent for outdoor planning - consistent conditions expected")
						} else if std > 10 {
							insights = append(insights, "High temperature variability detected")
							recommendations = append(recommendations, "Pack clothing for variable weather conditions")
						}
						
						if mean > 25 {
							recommendations = append(recommendations, "Warm weather activities recommended")
						} else if mean < 10 {
							recommendations = append(recommendations, "Cold weather gear advisable")
						}
					}
				}
				
			case "numerical":
				insights = append(insights, "Numerical data analysis completed")
				if values, ok := input.Data["values"].([]interface{}); ok {
					var numValues []float64
					for _, v := range values {
						if nf, ok := v.(float64); ok {
							numValues = append(numValues, nf)
						}
					}
					
					if len(numValues) > 0 {
						mean := calculateMean(numValues)
						statistics["mean"] = mean
						statistics["count"] = float64(len(numValues))
						statistics["sum"] = mean * float64(len(numValues))
						
						insights = append(insights, fmt.Sprintf("Processed %d numerical values", len(numValues)))
					}
				}
				
			case "statistical":
				insights = append(insights, "Statistical analysis performed")
				processedData["analysis_type"] = "comprehensive_statistical"
			}
			
			// Apply operations
			for _, operation := range input.Operations {
				switch operation {
				case "normalize":
					processedData["normalized"] = true
					insights = append(insights, "Data normalization applied")
				case "trend_analysis":
					processedData["trends"] = "analyzed"
					insights = append(insights, "Trend analysis completed")
				case "outlier_detection":
					processedData["outliers"] = "detected"
					insights = append(insights, "Outlier detection performed")
				}
			}
			
			// Calculate confidence based on data quality
			confidence := 0.85
			if len(insights) > 3 {
				confidence = 0.95
			}
			
			summary := fmt.Sprintf("Processed %s data with %d operations. " +
				"Generated %d insights and %d recommendations with %.0f%% confidence.",
				input.DataType, len(input.Operations), len(insights), len(recommendations), confidence*100)
			
			result := DataProcessingOutput{
				ProcessedData:   processedData,
				Summary:         summary,
				Statistics:      statistics,
				Insights:        insights,
				Recommendations: recommendations,
				Confidence:      confidence,
				ProcessedAt:     time.Now().Format("2006-01-02 15:04:05"),
			}
			
			// Add delay to simulate processing time
			time.Sleep(200 * time.Millisecond)
			
			return result, nil
		},
	)
	
	return weatherTool, dataProcessorTool
}

func createOpenAIProvider() core.Provider {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ OPENAI_API_KEY not found")
		return nil
	}

	provider := openai.New(
		openai.WithAPIKey(apiKey),
		openai.WithModel("gpt-5-mini"), // Using GPT-5-mini with fixed API support
	)

	// Add production-ready middleware
	enhancedProvider := middleware.Chain(
		middleware.WithRetry(middleware.RetryOpts{
			MaxAttempts: 3,
			BaseDelay:   time.Second,
			MaxDelay:    10 * time.Second,
			Jitter:      true,
		}),
		middleware.WithRateLimit(middleware.RateLimitOpts{
			RPS:   10, // Conservative rate limiting for GPT-5
			Burst: 20,
		}),
	)(provider)

	return enhancedProvider
}

func createGroqProvider() core.Provider {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ GROQ_API_KEY not found")
		return nil
	}

	provider, err := openai_compat.New(openai_compat.CompatOpts{
		BaseURL:      "https://api.groq.com/openai/v1",
		APIKey:       apiKey,
		DefaultModel: "moonshotai/kimi-k2-instruct",
		ProviderName: "groq",
		MaxRetries:   3,
		RetryDelay:   500 * time.Millisecond,
		
		// Groq/Kimi-K2 specific settings
		DisableJSONStreaming:     false,
		DisableParallelToolCalls: false,
		DisableStrictJSONSchema:  true, // Be conservative
		DisableToolChoice:        false,
		
		CustomHeaders: map[string]string{
			"X-Provider": "groq-kimi-k2",
		},
	})
	
	if err != nil {
		fmt.Printf("❌ Failed to create Groq provider: %v\n", err)
		return nil
	}

	// Add middleware optimized for Groq's high-speed inference
	enhancedProvider := middleware.Chain(
		middleware.WithRetry(middleware.RetryOpts{
			MaxAttempts: 2, // Groq is usually reliable, fewer retries needed
			BaseDelay:   200 * time.Millisecond, // Faster retries due to high speed
			MaxDelay:    2 * time.Second,
			Jitter:      true,
		}),
		middleware.WithRateLimit(middleware.RateLimitOpts{
			RPS:   50, // Groq can handle higher rates
			Burst: 100,
		}),
	)(provider)

	return enhancedProvider
}

func runScenario(ctx context.Context, provider core.Provider, scenario Scenario, weatherTool, dataProcessorTool tools.Handle) *TestResult {
	start := time.Now()
	
	// Convert tools to core handles
	coreTools := tools.ToCoreHandles([]tools.Handle{weatherTool, dataProcessorTool})
	
	// Create request with comprehensive system prompt
	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: `You are an advanced AI research assistant specialized in data analysis and research. 
					You have access to sophisticated tools for weather analysis and data processing. 
					
					Guidelines:
					- Use the weather analyzer tool for comprehensive weather data collection and analysis
					- Use the data processor tool for statistical analysis, pattern recognition, and generating insights
					- Always provide detailed reasoning for your decisions
					- When analyzing data, focus on actionable insights and practical recommendations
					- Be thorough but efficient in your tool usage
					- Combine information from multiple tools to provide comprehensive answers`},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: scenario.Query},
				},
			},
		},
		Tools:       coreTools,
		ToolChoice:  core.ToolAuto,
		StopWhen:    scenario.StopCondition,
		Temperature: 0.7, // Balanced for creativity and consistency
	}

	result, err := provider.GenerateText(ctx, request)
	duration := time.Since(start)
	
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return &TestResult{
			Duration: duration,
			Success:  false,
			Error:    err,
		}
	}

	// Count tool calls
	toolsCalled := 0
	for _, step := range result.Steps {
		toolsCalled += len(step.ToolCalls)
	}

	fmt.Printf("✅ Completed in %.2fs\n", duration.Seconds())
	fmt.Printf("📊 Steps: %d, Tools: %d, Tokens: %d\n", 
		len(result.Steps), toolsCalled, result.Usage.TotalTokens)
	fmt.Printf("💡 Final Response:\n%s\n", truncateText(result.Text, 200))

	return &TestResult{
		Duration:      duration,
		StepsExecuted: len(result.Steps),
		ToolsCalled:   toolsCalled,
		TokensUsed:    result.Usage,
		Success:       true,
		FinalText:     result.Text,
	}
}

func compareResults(gpt5Result, kimiResult *TestResult) {
	if gpt5Result == nil || kimiResult == nil {
		fmt.Println("⚠️  Cannot compare - one or both results are nil")
		return
	}

	fmt.Printf("⏱️  Duration:   GPT-5-mini: %.2fs  |  Kimi-K2: %.2fs", 
		gpt5Result.Duration.Seconds(), kimiResult.Duration.Seconds())
	
	if gpt5Result.Duration < kimiResult.Duration {
		fmt.Printf(" (GPT-5 faster by %.2fs)\n", (kimiResult.Duration - gpt5Result.Duration).Seconds())
	} else {
		fmt.Printf(" (Kimi faster by %.2fs)\n", (gpt5Result.Duration - kimiResult.Duration).Seconds())
	}

	fmt.Printf("🔧 Tool Calls: GPT-5-mini: %d      |  Kimi-K2: %d\n", 
		gpt5Result.ToolsCalled, kimiResult.ToolsCalled)
	fmt.Printf("📝 Steps:      GPT-5-mini: %d      |  Kimi-K2: %d\n", 
		gpt5Result.StepsExecuted, kimiResult.StepsExecuted)
	fmt.Printf("🎯 Tokens:     GPT-5-mini: %d     |  Kimi-K2: %d\n", 
		gpt5Result.TokensUsed.TotalTokens, kimiResult.TokensUsed.TotalTokens)
}

// Utility functions
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	sumSquaredDiff := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	return math.Sqrt(sumSquaredDiff / float64(len(values)-1))
}

func minMax(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	min, max := values[0], values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}