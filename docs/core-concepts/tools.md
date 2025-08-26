# Tools and Function Calling

This guide provides a comprehensive understanding of GAI's type-safe tools system, enabling AI models to call functions, interact with external services, and perform complex multi-step workflows.

## Table of Contents
- [Overview](#overview)
- [Tool Architecture](#tool-architecture)
- [Creating Tools](#creating-tools)
- [Type Safety](#type-safety)
- [Tool Execution](#tool-execution)
- [Multi-Step Workflows](#multi-step-workflows)
- [Stop Conditions](#stop-conditions)
- [Parallel Execution](#parallel-execution)
- [Error Handling](#error-handling)
- [Tool Patterns](#tool-patterns)
- [Best Practices](#best-practices)
- [Advanced Features](#advanced-features)

## Overview

GAI's tools system enables AI models to execute functions, interact with external APIs, access databases, and perform complex operations. The system is designed around several key principles:

- **Type Safety**: Input and output types are defined at compile-time using Go generics
- **JSON Schema Generation**: Automatic schema generation from Go types
- **Multi-Step Execution**: Tools can trigger additional AI reasoning and tool calls
- **Parallel Execution**: Multiple tools can run simultaneously
- **Error Resilience**: Comprehensive error handling and recovery

### Key Benefits

```go
// Type-safe tool definition
weatherTool := tools.New[WeatherInput, WeatherOutput](
    "get_weather",
    "Get current weather for a location",
    func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
        // Implementation with full type safety
        return WeatherOutput{
            Temperature: 72,
            Condition:   "sunny",
            Humidity:    65,
        }, nil
    },
)

// AI can call this tool with type validation
response, err := provider.GenerateText(ctx, core.Request{
    Messages: messages,
    Tools:    []tools.Handle{weatherTool},
    ToolChoice: core.ToolAuto,
})
```

## Tool Architecture

### Tool Interface

All tools implement the `Handle` interface:

```go
type Handle interface {
    Name() string
    Description() string
    Schema() *JSONSchema
    Execute(ctx context.Context, input []byte, meta Meta) ([]byte, error)
}
```

### Generic Tool Creation

GAI provides `tools.New[Input, Output]` for type-safe tool creation:

```go
func New[Input, Output any](
    name string,
    description string,
    fn func(ctx context.Context, input Input, meta Meta) (Output, error),
) Handle
```

This generic function:
1. Generates JSON schema from Input type
2. Provides type-safe marshaling/unmarshaling
3. Handles execution context and metadata
4. Manages error handling and timeouts

### Tool Metadata

The `Meta` type provides execution context:

```go
type Meta struct {
    CallID    string            // Unique call identifier
    MessageID string            // Message that triggered the call
    Provider  string            // Provider making the call
    Model     string            // Model name
    Headers   map[string]string // Custom headers
    Timeout   time.Duration     // Execution timeout
}
```

## Creating Tools

### Basic Tool Creation

```go
// Define input/output types
type CalculatorInput struct {
    Expression string `json:"expression" jsonschema:"required,description=Mathematical expression to evaluate"`
    Precision  int    `json:"precision,omitempty" jsonschema:"default=2,minimum=0,maximum=10"`
}

type CalculatorOutput struct {
    Result      float64 `json:"result"`
    Expression  string  `json:"original_expression"`
    Steps       []string `json:"calculation_steps,omitempty"`
}

// Create the tool
func createCalculatorTool() tools.Handle {
    return tools.New[CalculatorInput, CalculatorOutput](
        "calculator",
        "Perform mathematical calculations with step-by-step breakdown",
        func(ctx context.Context, input CalculatorInput, meta tools.Meta) (CalculatorOutput, error) {
            // Parse and evaluate expression
            result, steps, err := evaluateExpression(input.Expression)
            if err != nil {
                return CalculatorOutput{}, fmt.Errorf("calculation error: %w", err)
            }
            
            // Apply precision
            rounded := math.Round(result*math.Pow(10, float64(input.Precision))) / 
                      math.Pow(10, float64(input.Precision))
            
            return CalculatorOutput{
                Result:     rounded,
                Expression: input.Expression,
                Steps:      steps,
            }, nil
        },
    )
}
```

### Database Query Tool

```go
type DatabaseQueryInput struct {
    Query      string            `json:"query" jsonschema:"required,description=SQL query to execute"`
    Parameters map[string]any    `json:"parameters,omitempty" jsonschema:"description=Query parameters"`
    MaxRows    int               `json:"max_rows,omitempty" jsonschema:"default=100,minimum=1,maximum=1000"`
}

type DatabaseQueryOutput struct {
    Rows       []map[string]any `json:"rows"`
    RowCount   int              `json:"row_count"`
    Columns    []string         `json:"columns"`
    ExecTime   string           `json:"execution_time"`
}

func createDatabaseTool(db *sql.DB) tools.Handle {
    return tools.New[DatabaseQueryInput, DatabaseQueryOutput](
        "query_database",
        "Execute SQL queries against the database",
        func(ctx context.Context, input DatabaseQueryInput, meta tools.Meta) (DatabaseQueryOutput, error) {
            start := time.Now()
            
            // Validate query (implement safety checks)
            if err := validateQuery(input.Query); err != nil {
                return DatabaseQueryOutput{}, fmt.Errorf("query validation failed: %w", err)
            }
            
            // Execute query with timeout
            ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
            defer cancel()
            
            rows, err := db.QueryContext(ctx, input.Query)
            if err != nil {
                return DatabaseQueryOutput{}, fmt.Errorf("query execution failed: %w", err)
            }
            defer rows.Close()
            
            // Process results
            columns, _ := rows.Columns()
            var results []map[string]any
            
            for rows.Next() && len(results) < input.MaxRows {
                values := make([]any, len(columns))
                valuePtrs := make([]any, len(columns))
                
                for i := range values {
                    valuePtrs[i] = &values[i]
                }
                
                if err := rows.Scan(valuePtrs...); err != nil {
                    continue // Skip problematic rows
                }
                
                row := make(map[string]any)
                for i, col := range columns {
                    row[col] = values[i]
                }
                results = append(results, row)
            }
            
            execTime := time.Since(start)
            
            return DatabaseQueryOutput{
                Rows:     results,
                RowCount: len(results),
                Columns:  columns,
                ExecTime: execTime.String(),
            }, nil
        },
    )
}
```

### HTTP API Tool

```go
type HTTPRequestInput struct {
    URL     string            `json:"url" jsonschema:"required,format=uri"`
    Method  string            `json:"method,omitempty" jsonschema:"enum=GET,enum=POST,enum=PUT,enum=DELETE,default=GET"`
    Headers map[string]string `json:"headers,omitempty"`
    Body    string            `json:"body,omitempty"`
    Timeout int               `json:"timeout_seconds,omitempty" jsonschema:"default=30,minimum=1,maximum=300"`
}

type HTTPRequestOutput struct {
    StatusCode int               `json:"status_code"`
    Headers    map[string]string `json:"response_headers"`
    Body       string            `json:"response_body"`
    Size       int               `json:"response_size_bytes"`
    Duration   string            `json:"duration"`
}

func createHTTPTool() tools.Handle {
    client := &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns:    10,
            IdleConnTimeout: 30 * time.Second,
        },
    }
    
    return tools.New[HTTPRequestInput, HTTPRequestOutput](
        "http_request",
        "Make HTTP requests to web APIs",
        func(ctx context.Context, input HTTPRequestInput, meta tools.Meta) (HTTPRequestOutput, error) {
            start := time.Now()
            
            // Create request
            var bodyReader io.Reader
            if input.Body != "" {
                bodyReader = strings.NewReader(input.Body)
            }
            
            req, err := http.NewRequestWithContext(ctx, input.Method, input.URL, bodyReader)
            if err != nil {
                return HTTPRequestOutput{}, fmt.Errorf("failed to create request: %w", err)
            }
            
            // Set headers
            for key, value := range input.Headers {
                req.Header.Set(key, value)
            }
            
            // Execute request
            resp, err := client.Do(req)
            if err != nil {
                return HTTPRequestOutput{}, fmt.Errorf("request failed: %w", err)
            }
            defer resp.Body.Close()
            
            // Read response
            bodyBytes, err := io.ReadAll(resp.Body)
            if err != nil {
                return HTTPRequestOutput{}, fmt.Errorf("failed to read response: %w", err)
            }
            
            // Extract headers
            respHeaders := make(map[string]string)
            for key, values := range resp.Header {
                respHeaders[key] = strings.Join(values, ", ")
            }
            
            duration := time.Since(start)
            
            return HTTPRequestOutput{
                StatusCode: resp.StatusCode,
                Headers:    respHeaders,
                Body:       string(bodyBytes),
                Size:       len(bodyBytes),
                Duration:   duration.String(),
            }, nil
        },
    )
}
```

## Type Safety

### JSON Schema Generation

GAI automatically generates JSON schemas from Go types using struct tags:

```go
type WeatherInput struct {
    Location string  `json:"location" jsonschema:"required,description=City and country (e.g. 'London, UK')"`
    Units    string  `json:"units,omitempty" jsonschema:"enum=metric,enum=imperial,enum=kelvin,default=metric"`
    Language string  `json:"language,omitempty" jsonschema:"pattern=^[a-z]{2}$,default=en"`
    Details  bool    `json:"include_details,omitempty" jsonschema:"default=false"`
}
```

### Validation

Input validation happens automatically:

```go
// This will generate schema validation
type UserInput struct {
    Email    string `json:"email" jsonschema:"required,format=email"`
    Age      int    `json:"age" jsonschema:"minimum=0,maximum=150"`
    Username string `json:"username" jsonschema:"required,minLength=3,maxLength=50,pattern=^[a-zA-Z0-9_]+$"`
    Tags     []string `json:"tags,omitempty" jsonschema:"maxItems=10"`
}
```

### Custom Validation

Add custom validation in the tool function:

```go
func validateAndProcessUser(ctx context.Context, input UserInput, meta tools.Meta) (UserOutput, error) {
    // Custom validation beyond JSON schema
    if strings.Contains(input.Username, "admin") {
        return UserOutput{}, fmt.Errorf("username cannot contain 'admin'")
    }
    
    // Business logic validation
    if exists, err := checkUserExists(input.Username); err != nil {
        return UserOutput{}, fmt.Errorf("validation error: %w", err)
    } else if exists {
        return UserOutput{}, fmt.Errorf("username already exists")
    }
    
    // Process valid input
    return processUser(input)
}
```

## Tool Execution

### Single Tool Usage

```go
func singleToolExample() {
    provider := openai.New(openai.WithAPIKey(apiKey))
    weatherTool := createWeatherTool()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "What's the weather like in Tokyo?"},
                },
            },
        },
        Tools: []tools.Handle{weatherTool},
        ToolChoice: core.ToolAuto, // Let AI decide when to use tools
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Response:", response.Text)
    
    // Show tool execution steps
    for i, step := range response.Steps {
        fmt.Printf("Step %d:\n", i+1)
        for _, call := range step.ToolCalls {
            fmt.Printf("  Tool: %s\n", call.Name)
            fmt.Printf("  Input: %s\n", string(call.Input))
            fmt.Printf("  Output: %s\n", string(call.Result))
        }
    }
}
```

### Multiple Tool Usage

```go
func multipleToolsExample() {
    provider := anthropic.New(anthropic.WithAPIKey(apiKey))
    
    tools := []tools.Handle{
        createWeatherTool(),
        createNewsSearchTool(),
        createCalculatorTool(),
        createDatabaseTool(db),
    }
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are a helpful assistant with access to weather, news, calculator, and database tools. Use them as needed."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Give me a morning briefing: weather in SF, top 3 tech news, calculate my portfolio value (100 AAPL shares at current price), and query our user growth from the database."},
                },
            },
        },
        Tools: tools,
        ToolChoice: core.ToolAuto,
        MaxTokens: 2000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Morning Briefing:", response.Text)
}
```

## Multi-Step Workflows

### Stop Conditions

Control multi-step execution with stop conditions:

```go
// Stop after maximum steps
response, err := provider.GenerateText(ctx, core.Request{
    Messages: messages,
    Tools:    tools,
    StopWhen: core.MaxSteps(10), // Stop after 10 steps
})

// Stop when no more tools are needed
response, err := provider.GenerateText(ctx, core.Request{
    Messages: messages,
    Tools:    tools,
    StopWhen: core.NoMoreTools(), // Stop when AI doesn't call any tools
})

// Stop when specific tool is used
response, err := provider.GenerateText(ctx, core.Request{
    Messages: messages,
    Tools:    tools,
    StopWhen: core.UntilToolSeen("send_email"), // Stop after email is sent
})

// Combine multiple conditions
response, err := provider.GenerateText(ctx, core.Request{
    Messages: messages,
    Tools:    tools,
    StopWhen: core.CombineConditions(
        core.MaxSteps(20),
        core.NoMoreTools(),
        core.UntilToolSeen("final_report"),
    ),
})
```

### Complex Workflow Example

```go
func complexWorkflowExample() {
    tools := []tools.Handle{
        createWebSearchTool(),
        createDatabaseQueryTool(),
        createDataAnalysisTool(),
        createChartGeneratorTool(),
        createReportWriterTool(),
        createEmailTool(),
    }
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are a data analyst. Execute complex workflows step by step. Always provide detailed analysis and professional reports."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Research AI market trends, analyze our company's position using internal data, create visualizations, write a strategic report, and email it to leadership@company.com"},
                },
            },
        },
        Tools: tools,
        ToolChoice: core.ToolAuto,
        
        // Control execution flow
        StopWhen: core.CombineConditions(
            core.MaxSteps(25),
            core.UntilToolSeen("send_email"),
        ),
        
        MaxTokens: 4000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Display workflow execution
    fmt.Println("Workflow Result:", response.Text)
    
    fmt.Println("\nExecution Timeline:")
    for i, step := range response.Steps {
        fmt.Printf("Step %d (%d tools called):\n", i+1, len(step.ToolCalls))
        
        for _, call := range step.ToolCalls {
            fmt.Printf("  🔧 %s\n", call.Name)
        }
        
        if step.Text != "" {
            fmt.Printf("  💭 AI Reasoning: %s\n", truncate(step.Text, 100))
        }
        
        fmt.Println()
    }
}
```

## Stop Conditions

### Built-in Stop Conditions

GAI provides several built-in stop conditions:

```go
// MaxSteps - Limit number of execution steps
func MaxSteps(n int) StopCondition

// NoMoreTools - Stop when AI doesn't call any tools
func NoMoreTools() StopCondition

// UntilToolSeen - Stop after a specific tool is called
func UntilToolSeen(toolName string) StopCondition

// CombineConditions - Logical OR of multiple conditions
func CombineConditions(conditions ...StopCondition) StopCondition
```

### Custom Stop Conditions

Create custom stop conditions:

```go
// Stop when specific pattern is achieved
func StopWhenPatternMatch(pattern string) core.StopCondition {
    return func(stepNum int, step core.Step) bool {
        // Check if any tool result matches pattern
        for _, result := range step.ToolResults {
            if strings.Contains(string(result.Result), pattern) {
                return true // Stop execution
            }
        }
        return false // Continue
    }
}

// Stop based on cost/time limits
func StopWhenLimitsExceeded(maxCost float64, maxTime time.Duration) core.StopCondition {
    startTime := time.Now()
    totalCost := 0.0
    
    return func(stepNum int, step core.Step) bool {
        // Calculate estimated cost (implement your cost calculation)
        stepCost := estimateStepCost(step)
        totalCost += stepCost
        
        elapsed := time.Since(startTime)
        
        return totalCost >= maxCost || elapsed >= maxTime
    }
}

// Usage
response, err := provider.GenerateText(ctx, core.Request{
    Messages: messages,
    Tools:    tools,
    StopWhen: core.CombineConditions(
        StopWhenPatternMatch("WORKFLOW_COMPLETE"),
        StopWhenLimitsExceeded(10.0, 5*time.Minute),
        core.MaxSteps(30),
    ),
})
```

## Parallel Execution

### Automatic Parallel Tool Execution

GAI automatically executes independent tools in parallel:

```go
// These tools will run in parallel if called simultaneously
tools := []tools.Handle{
    createWeatherTool(),     // Can run independently
    createNewsSearchTool(),  // Can run independently  
    createStockPriceTool(),  // Can run independently
}

response, err := provider.GenerateText(ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Get weather for NYC, latest tech news, and AAPL stock price"},
            },
        },
    },
    Tools: tools,
    ToolChoice: core.ToolAuto,
})

// All three tools may execute in parallel
```

### Controlling Parallelism

Configure parallel execution limits:

```go
// Configure provider with parallelism control
provider := openai.New(
    openai.WithAPIKey(apiKey),
    openai.WithMaxParallelTools(5), // Max 5 tools in parallel
    openai.WithToolTimeout(30*time.Second),
)
```

## Error Handling

### Tool-Level Error Handling

```go
func createRobustTool() tools.Handle {
    return tools.New[ToolInput, ToolOutput](
        "robust_tool",
        "A tool with comprehensive error handling",
        func(ctx context.Context, input ToolInput, meta tools.Meta) (ToolOutput, error) {
            // Input validation
            if err := validateInput(input); err != nil {
                return ToolOutput{}, fmt.Errorf("invalid input: %w", err)
            }
            
            // Timeout handling
            ctx, cancel := context.WithTimeout(ctx, meta.Timeout)
            defer cancel()
            
            // Retry logic for transient errors
            var result ToolOutput
            err := retry.Do(
                func() error {
                    var err error
                    result, err = performOperation(ctx, input)
                    return err
                },
                retry.Attempts(3),
                retry.Delay(time.Second),
                retry.DelayType(retry.BackOffDelay),
                retry.RetryIf(func(err error) bool {
                    // Retry only transient errors
                    return isTransientError(err)
                }),
            )
            
            if err != nil {
                return ToolOutput{}, fmt.Errorf("operation failed after retries: %w", err)
            }
            
            return result, nil
        },
    )
}
```

### Graceful Degradation

```go
func createFallbackTool() tools.Handle {
    return tools.New[SearchInput, SearchOutput](
        "web_search_with_fallback",
        "Search with multiple fallback options",
        func(ctx context.Context, input SearchInput, meta tools.Meta) (SearchOutput, error) {
            // Try primary search API
            if result, err := primarySearch(ctx, input); err == nil {
                return result, nil
            }
            
            // Fallback to secondary API
            if result, err := secondarySearch(ctx, input); err == nil {
                result.Source = "fallback"
                return result, nil
            }
            
            // Last resort: cached/static results
            if result, err := cachedSearch(input); err == nil {
                result.Source = "cache"
                result.Warning = "Results from cache due to API unavailability"
                return result, nil
            }
            
            return SearchOutput{}, fmt.Errorf("all search options failed")
        },
    )
}
```

## Tool Patterns

### Command Pattern

```go
type Command interface {
    Execute(ctx context.Context) (any, error)
    Undo(ctx context.Context) error
    Description() string
}

type CommandInput struct {
    Operation string            `json:"operation" jsonschema:"required"`
    Arguments map[string]any   `json:"arguments,omitempty"`
    Metadata  map[string]string `json:"metadata,omitempty"`
}

func createCommandTool(registry map[string]Command) tools.Handle {
    return tools.New[CommandInput, any](
        "execute_command",
        "Execute system commands",
        func(ctx context.Context, input CommandInput, meta tools.Meta) (any, error) {
            command, exists := registry[input.Operation]
            if !exists {
                return nil, fmt.Errorf("unknown command: %s", input.Operation)
            }
            
            return command.Execute(ctx)
        },
    )
}
```

### Factory Pattern

```go
type ToolFactory struct {
    tools map[string]func() tools.Handle
}

func NewToolFactory() *ToolFactory {
    return &ToolFactory{
        tools: make(map[string]func() tools.Handle),
    }
}

func (f *ToolFactory) Register(name string, creator func() tools.Handle) {
    f.tools[name] = creator
}

func (f *ToolFactory) Create(name string) (tools.Handle, error) {
    creator, exists := f.tools[name]
    if !exists {
        return nil, fmt.Errorf("unknown tool: %s", name)
    }
    return creator(), nil
}

// Usage
factory := NewToolFactory()
factory.Register("weather", createWeatherTool)
factory.Register("database", func() tools.Handle { return createDatabaseTool(db) })

weatherTool, err := factory.Create("weather")
```

### Plugin Pattern

```go
type ToolPlugin interface {
    Name() string
    Version() string
    Tools() []tools.Handle
    Initialize(config map[string]any) error
    Cleanup() error
}

type PluginManager struct {
    plugins map[string]ToolPlugin
    tools   map[string]tools.Handle
}

func (pm *PluginManager) LoadPlugin(plugin ToolPlugin, config map[string]any) error {
    if err := plugin.Initialize(config); err != nil {
        return fmt.Errorf("plugin initialization failed: %w", err)
    }
    
    pm.plugins[plugin.Name()] = plugin
    
    // Register all plugin tools
    for _, tool := range plugin.Tools() {
        pm.tools[tool.Name()] = tool
    }
    
    return nil
}
```

## Best Practices

### 1. Type-Safe Design

Always use strong typing for tool inputs and outputs:

```go
// Good: Specific, typed input/output
type OrderInput struct {
    CustomerID string    `json:"customer_id" jsonschema:"required"`
    Items      []Item    `json:"items" jsonschema:"required,minItems=1"`
    ShippingAddr Address `json:"shipping_address" jsonschema:"required"`
}

type OrderOutput struct {
    OrderID      string    `json:"order_id"`
    TotalAmount  float64   `json:"total_amount"`
    EstimatedDelivery string `json:"estimated_delivery"`
    TrackingNumber string  `json:"tracking_number"`
}

// Bad: Generic, untyped interface
type GenericInput struct {
    Data map[string]any `json:"data"`
}
```

### 2. Comprehensive Error Handling

```go
func createProductionTool() tools.Handle {
    return tools.New[ToolInput, ToolOutput](
        "production_tool",
        "Production-ready tool with comprehensive error handling",
        func(ctx context.Context, input ToolInput, meta tools.Meta) (ToolOutput, error) {
            // Validate inputs
            if err := input.Validate(); err != nil {
                return ToolOutput{}, &ToolError{
                    Type:    "validation_error",
                    Message: "Input validation failed",
                    Details: err.Error(),
                }
            }
            
            // Set up monitoring
            defer func(start time.Time) {
                duration := time.Since(start)
                metrics.RecordToolExecution(meta.CallID, duration, err != nil)
            }(time.Now())
            
            // Execute with timeout
            ctx, cancel := context.WithTimeout(ctx, meta.Timeout)
            defer cancel()
            
            result, err := executeBusinessLogic(ctx, input)
            if err != nil {
                return ToolOutput{}, fmt.Errorf("business logic failed: %w", err)
            }
            
            return result, nil
        },
    )
}
```

### 3. Tool Documentation

```go
// Document your tools thoroughly
func createDocumentedTool() tools.Handle {
    return tools.New[DocumentedInput, DocumentedOutput](
        "documented_tool",
        `Comprehensive tool description:
        
        This tool performs X operation by:
        1. Validating input parameters
        2. Connecting to external service Y
        3. Processing data according to business rules
        4. Returning structured results
        
        Use cases:
        - Scenario A: When you need to...
        - Scenario B: For processing...
        
        Limitations:
        - Maximum 1000 items per request
        - Requires valid API credentials
        - May be rate-limited during peak hours`,
        func(ctx context.Context, input DocumentedInput, meta tools.Meta) (DocumentedOutput, error) {
            // Implementation
        },
    )
}
```

### 4. Resource Management

```go
func createResourceAwareTool(pool *ConnectionPool) tools.Handle {
    return tools.New[ToolInput, ToolOutput](
        "resource_aware_tool",
        "Tool that properly manages resources",
        func(ctx context.Context, input ToolInput, meta tools.Meta) (ToolOutput, error) {
            // Acquire resources
            conn, err := pool.Get()
            if err != nil {
                return ToolOutput{}, fmt.Errorf("failed to acquire connection: %w", err)
            }
            defer pool.Put(conn) // Always return resources
            
            // Use context for cancellation
            select {
            case <-ctx.Done():
                return ToolOutput{}, ctx.Err()
            default:
                // Continue with operation
            }
            
            return performOperation(conn, input)
        },
    )
}
```

### 5. Testing Tools

```go
func TestCalculatorTool(t *testing.T) {
    tool := createCalculatorTool()
    
    testCases := []struct {
        name     string
        input    CalculatorInput
        expected CalculatorOutput
        wantErr  bool
    }{
        {
            name:  "simple addition",
            input: CalculatorInput{Expression: "2 + 2", Precision: 0},
            expected: CalculatorOutput{
                Result:     4,
                Expression: "2 + 2",
            },
            wantErr: false,
        },
        {
            name:    "invalid expression",
            input:   CalculatorInput{Expression: "invalid"},
            wantErr: true,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            ctx := context.Background()
            meta := tools.Meta{
                CallID:  "test-call",
                Timeout: 5 * time.Second,
            }
            
            inputBytes, _ := json.Marshal(tc.input)
            outputBytes, err := tool.Execute(ctx, inputBytes, meta)
            
            if tc.wantErr {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            
            var output CalculatorOutput
            err = json.Unmarshal(outputBytes, &output)
            assert.NoError(t, err)
            assert.Equal(t, tc.expected.Result, output.Result)
        })
    }
}
```

## Advanced Features

### Tool Composition

```go
type CompositeToolInput struct {
    Pipeline []PipelineStep `json:"pipeline" jsonschema:"required,minItems=1"`
}

type PipelineStep struct {
    Tool   string         `json:"tool" jsonschema:"required"`
    Input  map[string]any `json:"input" jsonschema:"required"`
}

func createCompositiveTool(registry map[string]tools.Handle) tools.Handle {
    return tools.New[CompositeToolInput, map[string]any](
        "composite_tool",
        "Execute multiple tools in sequence, passing outputs as inputs",
        func(ctx context.Context, input CompositeToolInput, meta tools.Meta) (map[string]any, error) {
            var result map[string]any
            
            for i, step := range input.Pipeline {
                tool, exists := registry[step.Tool]
                if !exists {
                    return nil, fmt.Errorf("unknown tool in step %d: %s", i+1, step.Tool)
                }
                
                // Use previous result as input if available
                stepInput := step.Input
                if result != nil && i > 0 {
                    stepInput = mergeInputs(stepInput, result)
                }
                
                inputBytes, _ := json.Marshal(stepInput)
                outputBytes, err := tool.Execute(ctx, inputBytes, meta)
                if err != nil {
                    return nil, fmt.Errorf("step %d failed: %w", i+1, err)
                }
                
                if err := json.Unmarshal(outputBytes, &result); err != nil {
                    return nil, fmt.Errorf("failed to parse step %d output: %w", i+1, err)
                }
            }
            
            return result, nil
        },
    )
}
```

### Tool Registry with Dynamic Loading

```go
type DynamicToolRegistry struct {
    tools     map[string]tools.Handle
    factories map[string]func(config map[string]any) (tools.Handle, error)
    configs   map[string]map[string]any
    mu        sync.RWMutex
}

func (r *DynamicToolRegistry) Register(name string, factory func(map[string]any) (tools.Handle, error)) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.factories[name] = factory
}

func (r *DynamicToolRegistry) Get(name string) (tools.Handle, error) {
    r.mu.RLock()
    if tool, exists := r.tools[name]; exists {
        r.mu.RUnlock()
        return tool, nil
    }
    r.mu.RUnlock()
    
    // Create tool dynamically
    r.mu.Lock()
    defer r.mu.Unlock()
    
    factory, exists := r.factories[name]
    if !exists {
        return nil, fmt.Errorf("unknown tool: %s", name)
    }
    
    config := r.configs[name]
    tool, err := factory(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create tool %s: %w", name, err)
    }
    
    r.tools[name] = tool
    return tool, nil
}
```

## Summary

GAI's tools system provides:

1. **Type Safety**: Compile-time checking with Go generics
2. **Schema Generation**: Automatic JSON schema from types
3. **Multi-Step Execution**: Complex workflows with stop conditions
4. **Parallel Execution**: Automatic parallelization of independent tools
5. **Error Resilience**: Comprehensive error handling and recovery
6. **Extensibility**: Plugin patterns and dynamic loading

Key benefits:
- **Developer Experience**: Type-safe tool development
- **AI Integration**: Seamless function calling across all providers
- **Production Ready**: Built-in timeouts, retries, and monitoring
- **Flexible**: Support for simple functions to complex workflows

Next steps:
- [Streaming](./streaming.md) - Real-time response handling
- [Multi-Step Execution](./multi-step.md) - Advanced workflow patterns
- [Providers](./providers.md) - Understanding provider capabilities