// Braintrust Integration Demo - Enhanced with Automatic GenAI Observability
// This example demonstrates the GAI framework's automatic GenAI observability integration.
// All GenAI semantic conventions (gen_ai.*) are handled automatically by the framework!
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/obs"
	"github.com/recera/gai/providers/openai"
	"github.com/recera/gai/tools"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	// Load environment variables from .env file
	if err := loadEnv(".env"); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}
	
	// Verify required environment variables
	if err := checkEnvVars(); err != nil {
		log.Fatal(err)
	}

	// Initialize Braintrust observability
	shutdown := initBraintrustObservability()
	defer shutdown()

	// Create OpenAI provider
	provider := createOpenAIProvider()

	// Run comprehensive demo scenarios
	ctx := context.Background()
	
	fmt.Println("🧠 === Braintrust + GAI Framework Demo (Automatic GenAI Observability) ===\n")
	
	// Scenario 1: Simple chat with automatic observability
	fmt.Println("1. 💬 Simple Chat (Automatic Observability)")
	if err := simpleChatDemo(ctx, provider); err != nil {
		log.Printf("Simple chat demo failed: %v", err)
	}
	
	// Scenario 2: Multi-step tool execution  
	fmt.Println("\n2. 🔧 Multi-Step Tool Execution (Automatic Observability)")
	if err := multiStepToolDemo(ctx, provider); err != nil {
		log.Printf("Multi-step tool demo failed: %v", err)
	}
	
	// Scenario 3: Complex reasoning workflow
	fmt.Println("\n3. 🤖 Complex AI Agent Workflow (Automatic Observability)")
	if err := complexAgentDemo(ctx, provider); err != nil {
		log.Printf("Complex agent demo failed: %v", err)
	}
	
	// Scenario 4: Error handling
	fmt.Println("\n4. ⚠️ Error Handling (Automatic Observability)")
	if err := errorHandlingDemo(ctx, provider); err != nil {
		log.Printf("Error handling demo failed: %v", err)
	}
	
	// Scenario 5: Performance monitoring
	fmt.Println("\n5. 📊 Performance Monitoring (Automatic Observability)")
	performanceMonitoringDemo(ctx, provider)
	
	fmt.Println("\n✅ Demo completed! All traces automatically include:")
	fmt.Println("   • gen_ai.system = 'openai' (automatic provider mapping)")
	fmt.Println("   • gen_ai.operation.name = 'chat_completion' (automatic operation detection)")
	fmt.Println("   • gen_ai.request.model = 'gpt-4o-mini'") 
	fmt.Println("   • gen_ai.prompt.*.role and gen_ai.prompt.*.content (automatic content capture)")
	fmt.Println("   • gen_ai.completion and gen_ai.usage.* (automatic completion capture)")
	fmt.Println("   • Proper span names: 'chat_completion gpt-4o-mini'")
	fmt.Println("\n🔗 Check your Braintrust dashboard: https://www.braintrust.dev/app")
	
	// Wait for traces to be sent
	time.Sleep(2 * time.Second)
}

// simpleChatDemo demonstrates basic chat with automatic Braintrust tracing
func simpleChatDemo(ctx context.Context, provider *openai.Provider) error {
	// Create request - framework automatically handles ALL GenAI observability!
	request := core.Request{
		Model:       "gpt-4o-mini",
		Temperature: 0.7,
		MaxTokens:   150,
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a helpful AI assistant. Be concise and friendly."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Hello! Can you tell me a fun fact about AI?"},
				},
			},
		},
	}
	
	// Execute request - GenAI observability happens automatically!
	// Automatically creates: gen_ai.system="openai", gen_ai.prompt.*, gen_ai.completion, etc.
	result, err := provider.GenerateText(ctx, request)
	if err != nil {
		return fmt.Errorf("generating text: %w", err)
	}
	
	fmt.Printf("   🤖 Response: %s\n", result.Text)
	fmt.Printf("   📊 Tokens: %d input, %d output (%d total)\n", 
		result.Usage.InputTokens, result.Usage.OutputTokens, result.Usage.TotalTokens)
	
	return nil
}

// Tool definitions for multi-step demo
type WeatherInput struct {
	Location string `json:"location" description:"The city and state/country to get weather for"`
}

type WeatherOutput struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`  
	Conditions  string  `json:"conditions"`
	Summary     string  `json:"summary"`
}

type CalculatorInput struct {
	Operation string  `json:"operation" description:"Mathematical operation: add, subtract, multiply, divide"`
	A         float64 `json:"a" description:"First number"`
	B         float64 `json:"b" description:"Second number"`
}

type CalculatorOutput struct {
	Result     float64 `json:"result"`
	Expression string  `json:"expression"`
}

// multiStepToolDemo demonstrates multi-step execution with automatic observability
func multiStepToolDemo(ctx context.Context, provider *openai.Provider) error {
	// Create weather tool
	weatherTool := tools.New[WeatherInput, WeatherOutput](
		"get_weather",
		"Get current weather information for a specific location",
		func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
			time.Sleep(200 * time.Millisecond) // Simulate API call
			return WeatherOutput{
				Location:    input.Location,
				Temperature: 72.5,
				Conditions:  "Partly cloudy",
				Summary:     fmt.Sprintf("Pleasant weather in %s with mild temperatures", input.Location),
			}, nil
		},
	)
	
	// Create calculator tool with enhanced operation mapping
	calculatorTool := tools.New[CalculatorInput, CalculatorOutput](
		"calculator",
		"Perform basic mathematical calculations. Supports: add, subtract, multiply, divide",
		func(ctx context.Context, input CalculatorInput, meta tools.Meta) (CalculatorOutput, error) {
			var result float64
			var expression string
			
			// Normalize operation to handle AI variations
			operation := strings.ToLower(strings.TrimSpace(input.Operation))
			switch operation {
			case "add", "addition":
				result = input.A + input.B
				expression = fmt.Sprintf("%.2f + %.2f = %.2f", input.A, input.B, result)
			case "multiply", "multiplication": // Critical: handle both variants
				result = input.A * input.B
				expression = fmt.Sprintf("%.2f × %.2f = %.2f", input.A, input.B, result)
			case "subtract", "subtraction":
				result = input.A - input.B
				expression = fmt.Sprintf("%.2f - %.2f = %.2f", input.A, input.B, result)
			case "divide", "division":
				if input.B == 0 {
					return CalculatorOutput{}, fmt.Errorf("division by zero")
				}
				result = input.A / input.B
				expression = fmt.Sprintf("%.2f ÷ %.2f = %.2f", input.A, input.B, result)
			default:
				return CalculatorOutput{}, fmt.Errorf("unsupported operation: %s (supported: add, subtract, multiply, divide)", input.Operation)
			}
			
			return CalculatorOutput{
				Result:     result,
				Expression: expression,
			}, nil
		},
	)
	
	// Create multi-step request - framework handles ALL observability automatically!
	request := core.Request{
		Model:       "gpt-4o-mini",
		Temperature: 0.3,
		MaxTokens:   500,
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a helpful assistant with access to weather information and a calculator. Use tools when needed and provide comprehensive responses."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "What's the weather like in San Francisco? Also, can you calculate what 25.5 multiplied by 3.2 equals?"},
				},
			},
		},
		Tools: []core.ToolHandle{NewToolAdapter(weatherTool), NewToolAdapter(calculatorTool)},
		StopWhen: core.MaxSteps(5),
	}
	
	// Execute multi-step request - GenAI observability happens automatically!
	// Automatically creates: gen_ai.tools=["get_weather","calculator"], step tracking, etc.
	result, err := provider.GenerateText(ctx, request)
	if err != nil {
		return fmt.Errorf("multi-step execution: %w", err)
	}
	
	fmt.Printf("   📋 Executed %d steps\n", len(result.Steps))
	for i, step := range result.Steps {
		fmt.Printf("   Step %d: %d tool calls, %d results\n", i+1, len(step.ToolCalls), len(step.ToolResults))
	}
	fmt.Printf("   🤖 Final Response: %s\n", result.Text)
	fmt.Printf("   📊 Total Tokens: %d\n", result.Usage.TotalTokens)
	
	return nil
}

// Research tool for complex agent demo
type ResearchInput struct {
	Query  string `json:"query" description:"The research query or topic"`
	Domain string `json:"domain" description:"The domain to research (science, tech, business, etc.)"`
}

type ResearchOutput struct {
	Query     string   `json:"query"`
	Results   []string `json:"results"`
	Sources   []string `json:"sources"`
	Summary   string   `json:"summary"`
	Confidence float64 `json:"confidence"`
}

// complexAgentDemo demonstrates complex AI agent workflow with automatic observability
func complexAgentDemo(ctx context.Context, provider *openai.Provider) error {
	// Create research tool
	researchTool := tools.New[ResearchInput, ResearchOutput](
		"research",
		"Conduct research on a specific topic and provide detailed findings",
		func(ctx context.Context, input ResearchInput, meta tools.Meta) (ResearchOutput, error) {
			time.Sleep(500 * time.Millisecond) // Simulate research delay
			
			results := []string{
				"Recent advances in " + input.Domain + " show significant progress",
				"Key findings indicate multiple breakthrough areas",
				"Industry experts predict continued innovation",
			}
			
			return ResearchOutput{
				Query:     input.Query,
				Results:   results,
				Sources:   []string{"Academic Journal A", "Industry Report B", "Expert Analysis C"},
				Summary:   fmt.Sprintf("Research on '%s' in %s domain reveals promising developments", input.Query, input.Domain),
				Confidence: 0.87,
			}, nil
		},
	)
	
	request := core.Request{
		Model:       "gpt-4o-mini",
		Temperature: 0.4,
		MaxTokens:   800,
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are an expert research assistant. Use the research tool to gather information and provide comprehensive analysis."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Please research the latest developments in artificial intelligence and provide a detailed analysis."},
				},
			},
		},
		Tools: []core.ToolHandle{NewToolAdapter(researchTool)},
		StopWhen: core.MaxSteps(3),
	}
	
	// Execute complex agent workflow - GenAI observability happens automatically!
	result, err := provider.GenerateText(ctx, request)
	if err != nil {
		return fmt.Errorf("complex agent execution: %w", err)
	}
	
	fmt.Printf("   🔬 Research Steps: %d\n", len(result.Steps))
	fmt.Printf("   🤖 Analysis: %s\n", result.Text[:min(200, len(result.Text))]+"...")
	fmt.Printf("   📊 Total Tokens: %d\n", result.Usage.TotalTokens)
	
	return nil
}

// errorHandlingDemo demonstrates error handling with automatic observability
func errorHandlingDemo(ctx context.Context, provider *openai.Provider) error {
	// Create a tool that intentionally fails sometimes
	flakyTool := tools.New[map[string]string, map[string]string](
		"flaky_service",
		"A service that sometimes fails to test error handling",
		func(ctx context.Context, input map[string]string, meta tools.Meta) (map[string]string, error) {
			// Simulate a service failure
			return nil, fmt.Errorf("service temporarily unavailable")
		},
	)
	
	request := core.Request{
		Model:       "gpt-4o-mini",
		Temperature: 0.2,
		MaxTokens:   300,
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You have access to a flaky service. If it fails, provide a helpful response explaining the situation."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Can you use the flaky service to get some data?"},
				},
			},
		},
		Tools: []core.ToolHandle{NewToolAdapter(flakyTool)},
		StopWhen: core.MaxSteps(3),
	}
	
	// Execute request with error handling - GenAI observability happens automatically!
	// Errors are automatically recorded in spans with proper error attributes
	result, err := provider.GenerateText(ctx, request)
	if err != nil {
		return fmt.Errorf("error handling demo: %w", err)
	}
	
	fmt.Printf("   ⚠️ Handled %d steps with potential errors\n", len(result.Steps))
	fmt.Printf("   🤖 Response: %s\n", result.Text)
	
	return nil
}

// performanceMonitoringDemo demonstrates performance monitoring with automatic observability
func performanceMonitoringDemo(ctx context.Context, provider *openai.Provider) {
	fmt.Println("   📊 Running 3 parallel requests to demonstrate performance monitoring...")
	
	// Create multiple concurrent requests
	requests := []core.Request{
		{
			Model:       "gpt-4o-mini",
			Temperature: 0.5,
			MaxTokens:   100,
			Messages: []core.Message{
				{Role: core.User, Parts: []core.Part{core.Text{Text: "What is machine learning?"}}},
			},
		},
		{
			Model:       "gpt-4o-mini",
			Temperature: 0.5,
			MaxTokens:   100,
			Messages: []core.Message{
				{Role: core.User, Parts: []core.Part{core.Text{Text: "Explain neural networks."}}},
			},
		},
		{
			Model:       "gpt-4o-mini",
			Temperature: 0.5,
			MaxTokens:   100,
			Messages: []core.Message{
				{Role: core.User, Parts: []core.Part{core.Text{Text: "What is deep learning?"}}},
			},
		},
	}
	
	// Execute all requests concurrently - each gets automatic GenAI observability!
	results := make(chan *core.TextResult, len(requests))
	errors := make(chan error, len(requests))
	
	for i, req := range requests {
		go func(idx int, request core.Request) {
			start := time.Now()
			result, err := provider.GenerateText(ctx, request)
			duration := time.Since(start)
			
			if err != nil {
				errors <- err
				return
			}
			
			fmt.Printf("   Request %d completed in %v (%d tokens)\n", idx+1, duration, result.Usage.TotalTokens)
			results <- result
		}(i, req)
	}
	
	// Collect results
	var totalTokens int
	for i := 0; i < len(requests); i++ {
		select {
		case result := <-results:
			totalTokens += result.Usage.TotalTokens
		case err := <-errors:
			fmt.Printf("   Request failed: %v\n", err)
		case <-time.After(10 * time.Second):
			fmt.Printf("   Request %d timed out\n", i+1)
		}
	}
	
	fmt.Printf("   🏁 Completed performance test with %d total tokens\n", totalTokens)
}

// Utility functions (same as before)
func loadEnv(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			   (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}
		
		os.Setenv(key, value)
	}
	
	return scanner.Err()
}

func checkEnvVars() error {
	required := []string{"OPENAI_API_KEY", "BRAINTRUST_API_KEY", "BRAINTRUST_PROJECT_NAME"}
	missing := []string{}
	
	for _, key := range required {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}
	
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	
	return nil
}

func initBraintrustObservability() func() {
	// Clear any existing OTEL environment variables that might interfere
	os.Unsetenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT") 
	os.Unsetenv("OTEL_EXPORTER_OTLP_HEADERS")
	
	// Create OTLP HTTP exporter for Braintrust
	// Based on official Braintrust documentation: https://www.braintrust.dev/docs/cookbook/recipes/OTEL-logging
	braintrustProjectID := os.Getenv("BRAINTRUST_PROJECT_ID")
	braintrustProjectName := os.Getenv("BRAINTRUST_PROJECT_NAME")
	
	// Use project_id if available, otherwise project_name
	parentHeader := fmt.Sprintf("project_name:%s", braintrustProjectName)
	if braintrustProjectID != "" {
		parentHeader = fmt.Sprintf("project_id:%s", braintrustProjectID)
	}
	
	// Try using environment variables approach as recommended by Braintrust docs
	authHeader := fmt.Sprintf("Authorization=Bearer %s", os.Getenv("BRAINTRUST_API_KEY"))
	btParentHeader := fmt.Sprintf("x-bt-parent=%s", parentHeader)
	combinedHeaders := fmt.Sprintf("%s, %s", authHeader, btParentHeader)
	
	os.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "https://api.braintrust.dev/otel/v1/traces")
	os.Setenv("OTEL_EXPORTER_OTLP_HEADERS", combinedHeaders)
	
	log.Printf("Setting environment variables:")
	log.Printf("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=https://api.braintrust.dev/otel/v1/traces")
	log.Printf("OTEL_EXPORTER_OTLP_HEADERS=%s", combinedHeaders)
	
	// Create exporter without explicit configuration, letting it use environment variables
	exporter, err := otlptracehttp.New(context.Background())
	if err != nil {
		log.Fatalf("Failed to create OTLP exporter: %v", err)
	}

	// Create resource with proper attributes
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("braintrust.project.name", os.Getenv("BRAINTRUST_PROJECT_NAME")),
			attribute.String("service.name", "gai-braintrust-demo"),
			attribute.String("service.version", "2.0.0-automatic-observability"),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create resource: %v", err)
	}

	// Create tracer provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	// Set global tracer provider for the obs package
	otel.SetTracerProvider(tp)
	obs.SetGlobalTracerProvider(tp)

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}
}

func createOpenAIProvider() *openai.Provider {
	return openai.New(
		openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		openai.WithMaxRetries(3),
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Tool adapter to bridge tools.Handle and core.ToolHandle (same as before)
type ToolAdapter struct {
	handle tools.Handle
}

func NewToolAdapter(handle tools.Handle) *ToolAdapter {
	return &ToolAdapter{handle: handle}
}

func (ta *ToolAdapter) Name() string {
	return ta.handle.Name()
}

func (ta *ToolAdapter) Description() string {
	return ta.handle.Description()
}

func (ta *ToolAdapter) InSchemaJSON() []byte {
	return ta.handle.InSchemaJSON()
}

func (ta *ToolAdapter) OutSchemaJSON() []byte {
	return ta.handle.OutSchemaJSON()
}

func (ta *ToolAdapter) Exec(ctx context.Context, input json.RawMessage, meta interface{}) (any, error) {
	// Convert meta to tools.Meta if needed
	var toolMeta tools.Meta
	if meta != nil {
		if tm, ok := meta.(tools.Meta); ok {
			toolMeta = tm
		} else {
			// Try to convert from map
			if metaMap, ok := meta.(map[string]interface{}); ok {
				if callID, ok := metaMap["call_id"].(string); ok {
					toolMeta.CallID = callID
				}
			}
		}
	}
	return ta.handle.Exec(ctx, input, toolMeta)
}