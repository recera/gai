// Package main demonstrates advanced multi-step workflows and coordination with GAI
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/middleware"
	"github.com/recera/gai/providers/anthropic"
	"github.com/recera/gai/providers/groq"
	"github.com/recera/gai/providers/openai"
	"github.com/recera/gai/tools"
)

// Research tool types
type ResearchInput struct {
	Topic    string   `json:"topic" jsonschema:"description=Research topic or question"`
	Sources  []string `json:"sources,omitempty" jsonschema:"description=Specific sources to search"`
	MaxDepth int      `json:"max_depth,omitempty" jsonschema:"description=Research depth level,default=3"`
}

type ResearchOutput struct {
	Topic     string              `json:"topic"`
	Findings  []ResearchFinding   `json:"findings"`
	Sources   []string            `json:"sources"`
	Summary   string              `json:"summary"`
	Keywords  []string            `json:"keywords"`
	Confidence float64            `json:"confidence"`
}

type ResearchFinding struct {
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Source     string  `json:"source"`
	Relevance  float64 `json:"relevance"`
}

// Analysis tool types
type AnalysisInput struct {
	Data        string   `json:"data" jsonschema:"description=Data to analyze"`
	AnalysisType string  `json:"analysis_type" jsonschema:"description=Type of analysis: trend, sentiment, comparative, statistical"`
	Metrics     []string `json:"metrics,omitempty" jsonschema:"description=Specific metrics to calculate"`
}

type AnalysisOutput struct {
	AnalysisType string                 `json:"analysis_type"`
	Results      map[string]interface{} `json:"results"`
	Insights     []string               `json:"insights"`
	Recommendations []string           `json:"recommendations"`
	Confidence   float64               `json:"confidence"`
	Methodology  string                `json:"methodology"`
}

// Report generation tool types
type ReportInput struct {
	Title       string                 `json:"title" jsonschema:"description=Report title"`
	Sections    []string               `json:"sections" jsonschema:"description=Report sections to include"`
	Data        map[string]interface{} `json:"data" jsonschema:"description=Data to include in report"`
	Format      string                 `json:"format,omitempty" jsonschema:"description=Report format: markdown, html, json,default=markdown"`
	Audience    string                 `json:"audience,omitempty" jsonschema:"description=Target audience: technical, executive, general,default=general"`
}

type ReportOutput struct {
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Format      string    `json:"format"`
	Sections    []string  `json:"sections"`
	WordCount   int       `json:"word_count"`
	GeneratedAt time.Time `json:"generated_at"`
}

// Notification tool types
type NotificationInput struct {
	Type     string            `json:"type" jsonschema:"description=Notification type: email, slack, webhook"`
	To       string            `json:"to" jsonschema:"description=Recipient identifier"`
	Subject  string            `json:"subject" jsonschema:"description=Notification subject"`
	Content  string            `json:"content" jsonschema:"description=Notification content"`
	Metadata map[string]string `json:"metadata,omitempty" jsonschema:"description=Additional metadata"`
	Priority string            `json:"priority,omitempty" jsonschema:"description=Priority level: low, medium, high,default=medium"`
}

type NotificationOutput struct {
	Success   bool      `json:"success"`
	MessageID string    `json:"message_id,omitempty"`
	SentAt    time.Time `json:"sent_at"`
	Error     string    `json:"error,omitempty"`
}

func main() {
	fmt.Println("🚀 GAI Advanced Workflows Demo")
	fmt.Println("===============================\n")

	// Check for API keys
	checkAPIKeys()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Example 1: Multi-Provider Comparison
	fmt.Println("=== Example 1: Multi-Provider Comparison ===\n")
	multiProviderExample(ctx)

	// Example 2: Complex Multi-Step Research Workflow
	fmt.Println("\n=== Example 2: Complex Research Workflow ===\n")
	researchWorkflowExample(ctx)

	// Example 3: Advanced Stop Conditions
	fmt.Println("\n=== Example 3: Advanced Stop Conditions ===\n")
	stopConditionsExample(ctx)

	// Example 4: Streaming Multi-Step Workflow
	fmt.Println("\n=== Example 4: Streaming Workflows ===\n")
	streamingWorkflowExample(ctx)
}

func checkAPIKeys() {
	keys := map[string]string{
		"OPENAI_API_KEY":    os.Getenv("OPENAI_API_KEY"),
		"ANTHROPIC_API_KEY": os.Getenv("ANTHROPIC_API_KEY"),
		"GROQ_API_KEY":      os.Getenv("GROQ_API_KEY"),
	}

	available := []string{}
	for name, key := range keys {
		if key != "" {
			available = append(available, strings.Replace(name, "_API_KEY", "", 1))
		}
	}

	if len(available) == 0 {
		log.Fatal("At least one API key must be set (OPENAI_API_KEY, ANTHROPIC_API_KEY, or GROQ_API_KEY)")
	}

	fmt.Printf("Available providers: %v\n\n", available)
}

func multiProviderExample(ctx context.Context) {
	// Create different providers for comparison
	providers := make(map[string]core.Provider)

	// OpenAI GPT-4o Mini (fast and efficient)
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		providers["OpenAI"] = middleware.Chain(
			middleware.WithRetry(middleware.RetryOpts{MaxAttempts: 3}),
		)(openai.New(
			openai.WithAPIKey(apiKey),
			openai.WithModel(openai.GPT4oMini),
		))
	}

	// Anthropic Claude Sonnet 4 (reasoning and analysis)
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		providers["Anthropic"] = middleware.Chain(
			middleware.WithRetry(middleware.RetryOpts{MaxAttempts: 3}),
		)(anthropic.New(
			anthropic.WithAPIKey(apiKey),
			anthropic.WithModel(anthropic.ClaudeSonnet4),
		))
	}

	// Groq (ultra-fast inference)
	if apiKey := os.Getenv("GROQ_API_KEY"); apiKey != "" {
		providers["Groq"] = middleware.Chain(
			middleware.WithRetry(middleware.RetryOpts{MaxAttempts: 3}),
		)(groq.New(
			groq.WithAPIKey(apiKey),
			groq.WithModel(groq.Llama318BInstant), // Ultra-fast LPU inference
		))
	}

	if len(providers) == 0 {
		fmt.Println("No providers available for comparison")
		return
	}

	task := "Write a haiku about artificial intelligence and explain the literary devices used."

	fmt.Printf("Task: %s\n\n", task)

	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: task},
				},
			},
		},
		Temperature: 0.7,
		MaxTokens:   300,
	}

	// Execute task across all available providers
	for name, provider := range providers {
		fmt.Printf("--- %s Response ---\n", name)
		
		start := time.Now()
		result, err := provider.GenerateText(ctx, request)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("Response: %s\n", result.Text)
		fmt.Printf("Performance: %v, %d tokens\n", duration, result.Usage.TotalTokens)
		fmt.Println()
	}
}

func researchWorkflowExample(ctx context.Context) {
	// Create provider (prefer Claude for complex reasoning)
	var provider core.Provider
	
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		provider = anthropic.New(
			anthropic.WithAPIKey(apiKey),
			anthropic.WithModel(anthropic.ClaudeSonnet4),
		)
	} else if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		provider = openai.New(
			openai.WithAPIKey(apiKey),
			openai.WithModel(openai.GPT4o),
		)
	} else if apiKey := os.Getenv("GROQ_API_KEY"); apiKey != "" {
		provider = groq.New(
			groq.WithAPIKey(apiKey),
			groq.WithModel(groq.Llama318BInstant),
		)
	} else {
		fmt.Println("No provider available")
		return
	}

	// Create comprehensive tool suite
	researchTool := createResearchTool()
	analysisTool := createAnalysisTool()
	reportTool := createReportTool()
	notificationTool := createNotificationTool()

	coreTools := tools.ToCoreHandles([]tools.Handle{
		researchTool, analysisTool, reportTool, notificationTool,
	})

	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: `You are a senior research analyst and report writer. Execute comprehensive research workflows:

1. Research topics thoroughly using multiple approaches
2. Analyze findings to identify patterns and insights
3. Generate professional reports with clear recommendations
4. Send notifications upon completion

Work methodically and explain your reasoning at each step.`},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Research the current state of AI safety research, analyze the key challenges and progress, create a comprehensive report, and notify stakeholders@company.com when complete."},
				},
			},
		},
		Tools:      coreTools,
		ToolChoice: core.ToolAuto,
		Temperature: 0.4, // Balanced creativity and accuracy
		MaxTokens:   4000,
		// Advanced stop conditions
		StopWhen: core.CombineConditions(
			core.MaxSteps(20),                    // Limit total steps
			core.NoMoreTools(),                   // Stop when AI indicates completion
			core.UntilToolSeen("send_notification"), // Stop after notification sent
		),
	}

	fmt.Println("Executing comprehensive research workflow...")
	fmt.Println("This will demonstrate:")
	fmt.Println("• Multi-step research coordination")
	fmt.Println("• Cross-tool data flow")
	fmt.Println("• Advanced stop conditions")
	fmt.Println("• Professional reporting")
	fmt.Println()

	start := time.Now()
	result, err := provider.GenerateText(ctx, request)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Workflow error: %v", err)
		return
	}

	fmt.Printf("Workflow completed in %v\n", duration)
	fmt.Printf("Steps executed: %d\n", len(result.Steps))
	fmt.Printf("Total tokens: %d\n", result.Usage.TotalTokens)
	fmt.Println("\nFinal Response:")
	fmt.Println(result.Text)

	// Show workflow execution details
	fmt.Println("\n--- Workflow Execution Details ---")
	for i, step := range result.Steps {
		fmt.Printf("\nStep %d (%d tools called):\n", i+1, len(step.ToolCalls))
		
		// Show reasoning
		if step.Text != "" {
			fmt.Printf("  Reasoning: %s\n", truncateText(step.Text, 100))
		}
		
		// Show tools called
		for _, call := range step.ToolCalls {
			fmt.Printf("  🔧 Called: %s\n", call.Name)
		}
	}
}

func stopConditionsExample(ctx context.Context) {
	// Create provider
	var provider core.Provider
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		provider = openai.New(
			openai.WithAPIKey(apiKey),
			openai.WithModel(openai.GPT4oMini),
		)
	} else if apiKey := os.Getenv("GROQ_API_KEY"); apiKey != "" {
		provider = groq.New(
			groq.WithAPIKey(apiKey),
			groq.WithModel(groq.Llama318BInstant),
		)
	} else {
		fmt.Println("No provider available")
		return
	}

	// Create simple tools for demonstration
	tools := tools.ToCoreHandles([]tools.Handle{
		createSimpleCalculatorTool(),
		createSimpleSearchTool(),
	})

	// Demonstrate different stop conditions
	examples := []struct {
		name        string
		description string
		condition   core.StopCondition
		task        string
	}{
		{
			name:        "MaxSteps",
			description: "Stop after maximum number of steps",
			condition:   core.MaxSteps(3),
			task:        "Calculate 15 * 24, then search for information about that result, then provide analysis.",
		},
		{
			name:        "NoMoreTools", 
			description: "Stop when AI doesn't need more tools",
			condition:   core.NoMoreTools(),
			task:        "Calculate 10 + 5 and tell me if it's prime.",
		},
		{
			name:        "UntilToolSeen",
			description: "Stop after specific tool is used",
			condition:   core.UntilToolSeen("search"),
			task:        "Calculate something interesting and then search for more information about it.",
		},
		{
			name:        "Combined Conditions",
			description: "Multiple conditions with logical OR",
			condition: core.CombineConditions(
				core.MaxSteps(5),
				core.UntilToolSeen("search"),
			),
			task: "Perform calculations and research until you find something interesting or reach 5 steps.",
		},
	}

	for _, example := range examples {
		fmt.Printf("--- %s ---\n", example.name)
		fmt.Printf("Description: %s\n", example.description)
		fmt.Printf("Task: %s\n", example.task)

		request := core.Request{
			Messages: []core.Message{
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: example.task},
					},
				},
			},
			Tools:      tools,
			ToolChoice: core.ToolAuto,
			StopWhen:   example.condition,
		}

		result, err := provider.GenerateText(ctx, request)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("Result: %s\n", result.Text)
		fmt.Printf("Steps executed: %d\n", len(result.Steps))
		fmt.Printf("Tools called: %d\n", countToolCalls(result.Steps))
		fmt.Println()
	}
}

func streamingWorkflowExample(ctx context.Context) {
	// Create provider (Groq for ultra-fast streaming)
	var provider core.Provider
	if apiKey := os.Getenv("GROQ_API_KEY"); apiKey != "" {
		provider = groq.New(
			groq.WithAPIKey(apiKey),
			groq.WithModel(groq.Llama318BInstant),
		)
	} else if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		provider = openai.New(
			openai.WithAPIKey(apiKey),
			openai.WithModel(openai.GPT4oMini),
		)
	} else {
		fmt.Println("No provider available")
		return
	}

	tools := tools.ToCoreHandles([]tools.Handle{
		createSimpleCalculatorTool(),
		createSimpleSearchTool(),
	})

	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Calculate the result of 144 + 256, search for information about that number, and explain its mathematical significance."},
				},
			},
		},
		Tools:      tools,
		ToolChoice: core.ToolAuto,
		StopWhen:   core.MaxSteps(5),
		Stream:     true,
	}

	stream, err := provider.StreamText(ctx, request)
	if err != nil {
		log.Printf("Error creating stream: %v", err)
		return
	}
	defer stream.Close()

	fmt.Println("Streaming multi-step workflow:")
	fmt.Println(strings.Repeat("=", 60))

	stepCount := 0
	toolCount := 0

	for event := range stream.Events() {
		switch event.Type {
		case core.EventTextDelta:
			fmt.Print(event.TextDelta)

		case core.EventToolCall:
			fmt.Printf("\n\n🔧 Tool Call: %s", event.ToolCall.Name)
			toolCount++

		case core.EventToolResult:
			fmt.Printf("\n✅ Tool Result Received")
			stepCount++

		case core.EventFinish:
			fmt.Printf("\n\n%s\n", strings.Repeat("=", 60))
			fmt.Printf("Streaming workflow completed!\n")
			fmt.Printf("Steps: %d, Tools: %d", stepCount, toolCount)
			if event.Usage != nil {
				fmt.Printf(", Tokens: %d", event.Usage.TotalTokens)
			}
			fmt.Println()

		case core.EventError:
			fmt.Printf("\nStream error: %v\n", event.Error)
		}
	}
}

// Tool implementations
func createResearchTool() tools.Handle {
	return tools.New[ResearchInput, ResearchOutput](
		"research_topic",
		"Conduct comprehensive research on a given topic",
		func(ctx context.Context, input ResearchInput, meta tools.Meta) (ResearchOutput, error) {
			fmt.Printf("🔍 Researching: %s\n", input.Topic)
			
			// Simulate research delay
			time.Sleep(500 * time.Millisecond)

			// Simulate comprehensive research results
			findings := []ResearchFinding{
				{
					Title:     "Current State Analysis",
					Content:   fmt.Sprintf("Recent developments in %s show significant progress...", input.Topic),
					Source:    "Academic Database",
					Relevance: 0.95,
				},
				{
					Title:     "Industry Trends", 
					Content:   fmt.Sprintf("Key trends in %s include automation and enhanced safety measures...", input.Topic),
					Source:    "Industry Reports",
					Relevance: 0.87,
				},
				{
					Title:     "Future Projections",
					Content:   fmt.Sprintf("Experts predict %s will evolve significantly over the next decade...", input.Topic),
					Source:    "Expert Interviews",
					Relevance: 0.92,
				},
			}

			return ResearchOutput{
				Topic:      input.Topic,
				Findings:   findings,
				Sources:    []string{"Academic Database", "Industry Reports", "Expert Interviews"},
				Summary:    fmt.Sprintf("Comprehensive research on %s reveals ongoing development and future opportunities.", input.Topic),
				Keywords:   []string{"development", "trends", "future", "analysis"},
				Confidence: 0.91,
			}, nil
		},
	)
}

func createAnalysisTool() tools.Handle {
	return tools.New[AnalysisInput, AnalysisOutput](
		"analyze_data",
		"Perform detailed analysis on research data",
		func(ctx context.Context, input AnalysisInput, meta tools.Meta) (AnalysisOutput, error) {
			fmt.Printf("📊 Analyzing: %s\n", input.AnalysisType)
			
			time.Sleep(300 * time.Millisecond)

			results := map[string]interface{}{
				"data_points":     42,
				"trends_detected": 3,
				"correlation":     0.78,
				"statistical_significance": 0.05,
			}

			insights := []string{
				"Strong positive correlation identified in key metrics",
				"Emerging trends suggest accelerating development",
				"Risk factors are manageable with proper oversight",
			}

			recommendations := []string{
				"Continue monitoring key performance indicators",
				"Increase investment in promising areas",
				"Establish regular review cycles",
			}

			return AnalysisOutput{
				AnalysisType:    input.AnalysisType,
				Results:         results,
				Insights:        insights,
				Recommendations: recommendations,
				Confidence:      0.89,
				Methodology:     "Statistical analysis with trend identification and correlation mapping",
			}, nil
		},
	)
}

func createReportTool() tools.Handle {
	return tools.New[ReportInput, ReportOutput](
		"generate_report",
		"Generate comprehensive reports from analyzed data",
		func(ctx context.Context, input ReportInput, meta tools.Meta) (ReportOutput, error) {
			fmt.Printf("📝 Generating report: %s\n", input.Title)
			
			time.Sleep(400 * time.Millisecond)

			// Generate professional report content
			content := fmt.Sprintf(`# %s

## Executive Summary
This comprehensive analysis provides detailed insights into the current state and future prospects of the research topic.

## Key Findings
- Significant progress has been observed across multiple dimensions
- Emerging trends indicate positive trajectory
- Risk assessment shows manageable challenges

## Analysis Results  
Based on our comprehensive data analysis, we have identified several key patterns and correlations that inform our understanding.

## Recommendations
1. Continue systematic monitoring of key indicators
2. Strategic investment in high-potential areas
3. Regular assessment and adjustment of approaches

## Methodology
Our analysis employed multiple research methodologies to ensure comprehensive coverage and reliable results.

## Conclusion
The findings support optimistic projections while highlighting the importance of continued vigilance and strategic planning.

---
*Report generated on %s*
`, input.Title, time.Now().Format("January 2, 2006"))

			return ReportOutput{
				Title:       input.Title,
				Content:     content,
				Format:      input.Format,
				Sections:    input.Sections,
				WordCount:   len(strings.Fields(content)),
				GeneratedAt: time.Now(),
			}, nil
		},
	)
}

func createNotificationTool() tools.Handle {
	return tools.New[NotificationInput, NotificationOutput](
		"send_notification",
		"Send notifications to stakeholders",
		func(ctx context.Context, input NotificationInput, meta tools.Meta) (NotificationOutput, error) {
			fmt.Printf("📧 Sending %s notification to: %s\n", input.Type, input.To)
			fmt.Printf("   Subject: %s\n", input.Subject)
			
			time.Sleep(200 * time.Millisecond)

			return NotificationOutput{
				Success:   true,
				MessageID: fmt.Sprintf("msg_%d", time.Now().Unix()),
				SentAt:    time.Now(),
			}, nil
		},
	)
}

// Simple tools for stop conditions example
func createSimpleCalculatorTool() tools.Handle {
	return tools.New[map[string]interface{}, map[string]interface{}](
		"calculator",
		"Perform basic mathematical calculations",
		func(ctx context.Context, input map[string]interface{}, meta tools.Meta) (map[string]interface{}, error) {
			expression, ok := input["expression"].(string)
			if !ok {
				return map[string]interface{}{"error": "expression required"}, nil
			}

			fmt.Printf("🔢 Calculating: %s\n", expression)

			// Simple calculation examples
			results := map[string]float64{
				"15 * 24":  360,
				"10 + 5":   15,
				"144 + 256": 400,
			}

			if result, found := results[expression]; found {
				return map[string]interface{}{
					"expression": expression,
					"result":     result,
				}, nil
			}

			return map[string]interface{}{
				"expression": expression,
				"result":     42, // Default result
			}, nil
		},
	)
}

func createSimpleSearchTool() tools.Handle {
	return tools.New[map[string]interface{}, map[string]interface{}](
		"search",
		"Search for information on the internet",
		func(ctx context.Context, input map[string]interface{}, meta tools.Meta) (map[string]interface{}, error) {
			query, ok := input["query"].(string)
			if !ok {
				return map[string]interface{}{"error": "query required"}, nil
			}

			fmt.Printf("🔍 Searching: %s\n", query)

			return map[string]interface{}{
				"query": query,
				"results": []map[string]interface{}{
					{
						"title":   fmt.Sprintf("Information about %s", query),
						"snippet": fmt.Sprintf("Comprehensive overview of %s with detailed analysis...", query),
						"url":     "https://example.com/result1",
					},
					{
						"title":   fmt.Sprintf("Latest research on %s", query),
						"snippet": fmt.Sprintf("Recent findings and developments in %s research...", query),
						"url":     "https://example.com/result2",
					},
				},
			}, nil
		},
	)
}

// Helper functions
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

func countToolCalls(steps []core.Step) int {
	count := 0
	for _, step := range steps {
		count += len(step.ToolCalls)
	}
	return count
}