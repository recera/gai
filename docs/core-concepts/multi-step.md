# Multi-Step Execution and Workflows

This guide provides a comprehensive understanding of GAI's multi-step execution system, enabling AI models to perform complex workflows, chain tool calls, and execute sophisticated reasoning patterns.

## Table of Contents
- [Overview](#overview)
- [Execution Model](#execution-model)
- [Stop Conditions](#stop-conditions)
- [Workflow Patterns](#workflow-patterns)
- [State Management](#state-management)
- [Error Recovery](#error-recovery)
- [Performance Optimization](#performance-optimization)
- [Complex Workflows](#complex-workflows)
- [Monitoring and Observability](#monitoring-and-observability)
- [Best Practices](#best-practices)
- [Advanced Patterns](#advanced-patterns)

## Overview

Multi-step execution allows AI models to perform complex tasks that require multiple reasoning steps, tool calls, and decision points. Instead of single request-response interactions, GAI enables sophisticated workflows where AI can plan, execute, and adapt its approach dynamically.

### Key Capabilities

```go
// Simple single-step execution
response, err := provider.GenerateText(ctx, core.Request{
    Messages: messages,
    Tools:    tools,
})

// Multi-step execution with control
response, err := provider.GenerateText(ctx, core.Request{
    Messages: messages,
    Tools:    tools,
    StopWhen: core.CombineConditions(
        core.MaxSteps(20),              // Limit iterations
        core.NoMoreTools(),             // Stop when AI is done
        core.UntilToolSeen("send_email"), // Stop after specific action
    ),
})
```

### Benefits

- **Complex Problem Solving**: Break down large problems into manageable steps
- **Dynamic Planning**: AI can adjust its approach based on intermediate results
- **Tool Orchestration**: Coordinate multiple tools in sophisticated workflows
- **Error Recovery**: Handle failures and retry with alternative approaches
- **Resource Management**: Control execution time, cost, and complexity

## Execution Model

### Step-by-Step Process

```
1. Initial AI Response
   ├── Generate text and/or tool calls
   └── Check stop conditions
       ├── Stop → Return result
       └── Continue → Step 2

2. Tool Execution
   ├── Execute all tool calls in parallel
   ├── Collect results
   └── Add results to message history

3. Next AI Response
   ├── Process tool results
   ├── Generate next response
   └── Check stop conditions
       ├── Stop → Return result
       └── Continue → Repeat from Step 2
```

### Execution Context

Each step maintains rich context:

```go
type Step struct {
    StepNumber  int           `json:"step_number"`
    ToolCalls   []ToolCall    `json:"tool_calls"`
    ToolResults []ToolResult  `json:"tool_results"`
    Text        string        `json:"text"`
    Usage       Usage         `json:"usage"`
    Duration    time.Duration `json:"duration"`
    Error       error         `json:"error,omitempty"`
}

type TextResult struct {
    Text  string  `json:"text"`
    Steps []Step  `json:"steps"`  // Complete execution history
    Usage Usage   `json:"usage"`  // Cumulative usage
    Raw   any     `json:"raw"`    // Provider-specific data
}
```

### Parallel Tool Execution

Tools within a step execute in parallel:

```go
func executeToolsInParallel(ctx context.Context, toolCalls []ToolCall) []ToolResult {
    results := make([]ToolResult, len(toolCalls))
    var wg sync.WaitGroup
    
    for i, call := range toolCalls {
        wg.Add(1)
        go func(idx int, tc ToolCall) {
            defer wg.Done()
            
            result, err := executeTool(ctx, tc)
            results[idx] = ToolResult{
                CallID: tc.CallID,
                Name:   tc.Name,
                Result: result,
                Error:  err,
            }
        }(i, call)
    }
    
    wg.Wait()
    return results
}
```

## Stop Conditions

### Built-in Stop Conditions

GAI provides several built-in stop conditions:

```go
// MaxSteps - Limit number of execution steps
stepLimit := core.MaxSteps(10)

// NoMoreTools - Stop when AI doesn't call any tools
naturalStop := core.NoMoreTools()

// UntilToolSeen - Stop after a specific tool is called
targetReached := core.UntilToolSeen("send_notification")

// CombineConditions - Logical OR of multiple conditions
combined := core.CombineConditions(
    core.MaxSteps(20),
    core.NoMoreTools(),
    core.UntilToolSeen("final_report"),
)
```

### Custom Stop Conditions

Create domain-specific stop conditions:

```go
// Stop when specific goal is achieved
func StopWhenGoalReached(goalPattern string) core.StopCondition {
    return func(stepNum int, step core.Step) bool {
        // Check if AI's text response indicates goal completion
        return strings.Contains(step.Text, goalPattern)
    }
}

// Stop based on cost limits
func StopWhenCostExceeded(maxCost float64) core.StopCondition {
    totalCost := 0.0
    
    return func(stepNum int, step core.Step) bool {
        stepCost := estimateCost(step.Usage)
        totalCost += stepCost
        
        if totalCost >= maxCost {
            log.Printf("Stopping execution: cost limit reached (%.2f)", totalCost)
            return true
        }
        
        return false
    }
}

// Stop on specific tool result pattern
func StopWhenToolResultMatches(toolName, pattern string) core.StopCondition {
    return func(stepNum int, step core.Step) bool {
        for _, result := range step.ToolResults {
            if result.Name == toolName {
                resultStr := string(result.Result)
                if matched, _ := regexp.MatchString(pattern, resultStr); matched {
                    return true
                }
            }
        }
        return false
    }
}

// Time-based stop condition
func StopWhenTimeExceeded(maxDuration time.Duration) core.StopCondition {
    start := time.Now()
    
    return func(stepNum int, step core.Step) bool {
        elapsed := time.Since(start)
        return elapsed >= maxDuration
    }
}

// Usage example
response, err := provider.GenerateText(ctx, core.Request{
    Messages: messages,
    Tools:    tools,
    StopWhen: core.CombineConditions(
        StopWhenGoalReached("TASK_COMPLETED"),
        StopWhenCostExceeded(5.0),
        core.MaxSteps(25),
    ),
})
```

### Conditional Stop Conditions

Create conditions that depend on execution state:

```go
func StopWhenConditionMet(condition func([]core.Step) bool) core.StopCondition {
    var allSteps []core.Step
    
    return func(stepNum int, step core.Step) bool {
        allSteps = append(allSteps, step)
        return condition(allSteps)
    }
}

// Stop when specific sequence of tools is executed
func StopWhenToolSequence(sequence []string) core.StopCondition {
    return StopWhenConditionMet(func(steps []core.Step) bool {
        executedTools := []string{}
        
        for _, step := range steps {
            for _, call := range step.ToolCalls {
                executedTools = append(executedTools, call.Name)
            }
        }
        
        // Check if sequence appears in executed tools
        return containsSequence(executedTools, sequence)
    })
}

// Stop when error rate exceeds threshold
func StopWhenErrorRateHigh(threshold float64) core.StopCondition {
    return StopWhenConditionMet(func(steps []core.Step) bool {
        totalCalls := 0
        errorCalls := 0
        
        for _, step := range steps {
            for _, result := range step.ToolResults {
                totalCalls++
                if result.Error != nil {
                    errorCalls++
                }
            }
        }
        
        if totalCalls > 5 { // Minimum sample size
            errorRate := float64(errorCalls) / float64(totalCalls)
            return errorRate >= threshold
        }
        
        return false
    })
}
```

## Workflow Patterns

### Sequential Workflow

```go
func sequentialWorkflowExample() {
    tools := []tools.Handle{
        createDataRetrievalTool(),
        createDataProcessingTool(),
        createAnalysisTool(),
        createReportGeneratorTool(),
    }
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "Execute tasks sequentially. Complete each step before moving to the next."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Generate monthly sales report: 1) Get sales data 2) Process the data 3) Analyze trends 4) Create report"},
                },
            },
        },
        Tools: tools,
        StopWhen: core.CombineConditions(
            core.MaxSteps(8),
            core.UntilToolSeen("generate_report"),
        ),
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Display execution timeline
    fmt.Println("Sequential Workflow Results:")
    for i, step := range response.Steps {
        fmt.Printf("Step %d: ", i+1)
        for _, call := range step.ToolCalls {
            fmt.Printf("%s ", call.Name)
        }
        fmt.Printf("(Duration: %v)\n", step.Duration)
    }
}
```

### Parallel Workflow

```go
func parallelWorkflowExample() {
    tools := []tools.Handle{
        createWeatherTool(),
        createNewsSearchTool(),
        createStockTool(),
        createCalendarTool(),
        createEmailTool(),
    }
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You can call multiple tools in parallel when tasks are independent."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Create my morning briefing: weather, top news, stock prices, today's calendar, then email the summary."},
                },
            },
        },
        Tools: tools,
        StopWhen: core.UntilToolSeen("send_email"),
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Analyze parallel execution
    fmt.Println("Parallel Workflow Analysis:")
    for i, step := range response.Steps {
        if len(step.ToolCalls) > 1 {
            fmt.Printf("Step %d: %d tools executed in parallel\n", i+1, len(step.ToolCalls))
            for _, call := range step.ToolCalls {
                fmt.Printf("  - %s\n", call.Name)
            }
        }
    }
}
```

### Conditional Workflow

```go
func conditionalWorkflowExample() {
    tools := []tools.Handle{
        createHealthCheckTool(),
        createFailoverTool(),
        createNotificationTool(),
        createLogAnalysisTool(),
        createRecoveryTool(),
    }
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are a system administrator. Check system health and take appropriate actions based on the results."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Monitor system health. If issues found, perform failover and send alerts. If critical, analyze logs and attempt recovery."},
                },
            },
        },
        Tools: tools,
        StopWhen: core.CombineConditions(
            core.MaxSteps(15),
            StopWhenGoalReached("SYSTEM_STABLE"),
        ),
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Conditional Workflow Execution:")
    fmt.Println(response.Text)
}
```

### Iterative Workflow

```go
func iterativeWorkflowExample() {
    tools := []tools.Handle{
        createDataSamplingTool(),
        createModelTrainingTool(),
        createValidationTool(),
        createMetricsCalculationTool(),
    }
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are a ML engineer. Iteratively improve model performance until target accuracy is reached."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Train a model with target accuracy >95%. Iterate: sample data, train, validate, check metrics. Stop when target reached or after 10 iterations."},
                },
            },
        },
        Tools: tools,
        StopWhen: core.CombineConditions(
            core.MaxSteps(40), // 4 tools × 10 iterations
            StopWhenToolResultMatches("calculate_metrics", "accuracy.*9[5-9]\\.[0-9]%"), // >95%
        ),
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Analyze iterations
    fmt.Println("Iterative Workflow Results:")
    iterations := len(response.Steps) / 4 // Assuming 4 tools per iteration
    fmt.Printf("Completed %d iterations\n", iterations)
    
    // Find final accuracy
    for _, step := range response.Steps {
        for _, result := range step.ToolResults {
            if result.Name == "calculate_metrics" {
                fmt.Printf("Final metrics: %s\n", string(result.Result))
            }
        }
    }
}
```

## State Management

### Workflow State Tracking

```go
type WorkflowState struct {
    Phase       string                 `json:"phase"`
    Progress    float64               `json:"progress"`
    Data        map[string]any        `json:"data"`
    Metrics     map[string]float64    `json:"metrics"`
    Errors      []string              `json:"errors,omitempty"`
    StartTime   time.Time             `json:"start_time"`
    LastUpdate  time.Time             `json:"last_update"`
}

func createStatefulWorkflow() {
    // Initialize state
    state := &WorkflowState{
        Phase:     "initialization",
        Progress:  0.0,
        Data:      make(map[string]any),
        Metrics:   make(map[string]float64),
        StartTime: time.Now(),
    }
    
    // Create state-aware tools
    tools := []tools.Handle{
        createStateAwareTool(state),
        createProgressTrackingTool(state),
        createMetricsCollectionTool(state),
    }
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "Track workflow state and progress. Update phase and metrics as you proceed."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Execute data processing pipeline with progress tracking."},
                },
            },
        },
        Tools: tools,
        StopWhen: StopWhenConditionMet(func(steps []core.Step) bool {
            return state.Progress >= 1.0 // 100% complete
        }),
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Final state
    fmt.Printf("Workflow completed in phase: %s\n", state.Phase)
    fmt.Printf("Final progress: %.1f%%\n", state.Progress*100)
    fmt.Printf("Duration: %v\n", time.Since(state.StartTime))
}

func createStateAwareTool(state *WorkflowState) tools.Handle {
    type StateInput struct {
        Action string         `json:"action"`
        Phase  string         `json:"phase,omitempty"`
        Data   map[string]any `json:"data,omitempty"`
    }
    
    return tools.New[StateInput, WorkflowState](
        "update_state",
        "Update workflow state and progress",
        func(ctx context.Context, input StateInput, meta tools.Meta) (WorkflowState, error) {
            switch input.Action {
            case "set_phase":
                state.Phase = input.Phase
                state.LastUpdate = time.Now()
                
            case "update_data":
                for k, v := range input.Data {
                    state.Data[k] = v
                }
                state.LastUpdate = time.Now()
                
            case "increment_progress":
                if progress, ok := input.Data["increment"].(float64); ok {
                    state.Progress = math.Min(1.0, state.Progress+progress)
                    state.LastUpdate = time.Now()
                }
            }
            
            return *state, nil
        },
    )
}
```

### Persistent State

```go
type PersistentWorkflowManager struct {
    storage  StateStorage
    workflow *WorkflowState
}

type StateStorage interface {
    Save(key string, state *WorkflowState) error
    Load(key string) (*WorkflowState, error)
    Delete(key string) error
}

func (pwm *PersistentWorkflowManager) ExecuteWithPersistence(
    workflowID string,
    provider core.Provider,
    request core.Request,
) (*core.TextResult, error) {
    
    // Load existing state or create new
    state, err := pwm.storage.Load(workflowID)
    if err != nil {
        state = &WorkflowState{
            Phase:     "starting",
            StartTime: time.Now(),
            Data:      make(map[string]any),
        }
    }
    
    // Add state persistence to stop conditions
    originalStopWhen := request.StopWhen
    request.StopWhen = func(stepNum int, step core.Step) bool {
        // Save state after each step
        if saveErr := pwm.storage.Save(workflowID, state); saveErr != nil {
            log.Printf("Failed to save state: %v", saveErr)
        }
        
        // Check original condition
        if originalStopWhen != nil {
            return originalStopWhen(stepNum, step)
        }
        
        return false
    }
    
    // Execute workflow
    result, err := provider.GenerateText(ctx, request)
    
    // Clean up completed workflows
    if err == nil {
        pwm.storage.Delete(workflowID)
    }
    
    return result, err
}
```

## Error Recovery

### Automatic Retry with Backoff

```go
func executeWithRetry(provider core.Provider, request core.Request) (*core.TextResult, error) {
    maxRetries := 3
    baseDelay := time.Second
    
    for attempt := 1; attempt <= maxRetries; attempt++ {
        result, err := provider.GenerateText(ctx, request)
        
        if err == nil {
            return result, nil
        }
        
        // Check if error is retryable
        if !isRetryableError(err) {
            return nil, fmt.Errorf("non-retryable error: %w", err)
        }
        
        // Check if we have remaining attempts
        if attempt >= maxRetries {
            return nil, fmt.Errorf("max retries exceeded: %w", err)
        }
        
        // Exponential backoff
        delay := baseDelay * time.Duration(1<<uint(attempt-1))
        log.Printf("Attempt %d failed: %v. Retrying in %v...", attempt, err, delay)
        time.Sleep(delay)
    }
    
    return nil, fmt.Errorf("all retry attempts failed")
}

func isRetryableError(err error) bool {
    // Check for temporary network errors
    if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
        return true
    }
    
    // Check for specific error patterns
    errStr := err.Error()
    retryablePatterns := []string{
        "rate limit",
        "timeout",
        "temporary",
        "503 Service Unavailable",
        "502 Bad Gateway",
        "connection reset",
    }
    
    for _, pattern := range retryablePatterns {
        if strings.Contains(strings.ToLower(errStr), pattern) {
            return true
        }
    }
    
    return false
}
```

### Graceful Degradation

```go
func executeWithFallback(provider core.Provider, request core.Request) (*core.TextResult, error) {
    // Try with all tools first
    result, err := provider.GenerateText(ctx, request)
    if err == nil {
        return result, nil
    }
    
    log.Printf("Full execution failed: %v. Trying with reduced tool set...", err)
    
    // Fallback: reduce tool complexity
    essentialTools := filterEssentialTools(request.Tools)
    fallbackRequest := request
    fallbackRequest.Tools = essentialTools
    fallbackRequest.StopWhen = core.MaxSteps(10) // Reduce complexity
    
    result, err = provider.GenerateText(ctx, fallbackRequest)
    if err == nil {
        result.Text += "\n\n⚠️ Note: Executed with reduced functionality due to issues with full tool set."
        return result, nil
    }
    
    log.Printf("Fallback execution failed: %v. Trying without tools...", err)
    
    // Last resort: no tools
    minimalRequest := request
    minimalRequest.Tools = nil
    minimalRequest.StopWhen = nil
    
    result, err = provider.GenerateText(ctx, minimalRequest)
    if err == nil {
        result.Text += "\n\n⚠️ Note: Executed without tools due to technical difficulties."
        return result, nil
    }
    
    return nil, fmt.Errorf("all fallback attempts failed: %w", err)
}

func filterEssentialTools(tools []tools.Handle) []tools.Handle {
    essential := []string{
        "get_current_time",
        "search_knowledge_base",
        "calculate",
    }
    
    var filtered []tools.Handle
    for _, tool := range tools {
        for _, name := range essential {
            if tool.Name() == name {
                filtered = append(filtered, tool)
                break
            }
        }
    }
    
    return filtered
}
```

### Circuit Breaker Pattern

```go
type CircuitBreaker struct {
    failureThreshold int
    resetTimeout     time.Duration
    failures         int
    lastFailureTime  time.Time
    state            CircuitState
    mutex            sync.RWMutex
}

type CircuitState int

const (
    StateClosed CircuitState = iota
    StateOpen
    StateHalfOpen
)

func (cb *CircuitBreaker) Execute(fn func() (*core.TextResult, error)) (*core.TextResult, error) {
    cb.mutex.RLock()
    state := cb.state
    failures := cb.failures
    cb.mutex.RUnlock()
    
    // Check if circuit breaker should reset
    if state == StateOpen && time.Since(cb.lastFailureTime) >= cb.resetTimeout {
        cb.mutex.Lock()
        cb.state = StateHalfOpen
        cb.mutex.Unlock()
        state = StateHalfOpen
    }
    
    switch state {
    case StateOpen:
        return nil, fmt.Errorf("circuit breaker is open")
        
    case StateHalfOpen:
        result, err := fn()
        if err != nil {
            cb.recordFailure()
            return nil, err
        }
        cb.recordSuccess()
        return result, nil
        
    case StateClosed:
        result, err := fn()
        if err != nil {
            cb.recordFailure()
            return nil, err
        }
        return result, nil
    }
    
    return nil, fmt.Errorf("unknown circuit breaker state")
}

func (cb *CircuitBreaker) recordFailure() {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    cb.failures++
    cb.lastFailureTime = time.Now()
    
    if cb.failures >= cb.failureThreshold {
        cb.state = StateOpen
    }
}

func (cb *CircuitBreaker) recordSuccess() {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    cb.failures = 0
    cb.state = StateClosed
}

// Usage
func executeWithCircuitBreaker(provider core.Provider, request core.Request) (*core.TextResult, error) {
    cb := &CircuitBreaker{
        failureThreshold: 5,
        resetTimeout:     30 * time.Second,
        state:            StateClosed,
    }
    
    return cb.Execute(func() (*core.TextResult, error) {
        return provider.GenerateText(ctx, request)
    })
}
```

## Performance Optimization

### Parallel Workflow Optimization

```go
type OptimizedExecutor struct {
    maxConcurrentSteps int
    toolTimeout        time.Duration
    stepTimeout        time.Duration
}

func (oe *OptimizedExecutor) Execute(
    provider core.Provider,
    request core.Request,
) (*core.TextResult, error) {
    
    // Optimize request for parallel execution
    optimizedRequest := oe.optimizeRequest(request)
    
    // Execute with performance monitoring
    start := time.Now()
    result, err := provider.GenerateText(ctx, optimizedRequest)
    duration := time.Since(start)
    
    if result != nil {
        // Add performance metrics
        result.Metadata = map[string]any{
            "execution_time":     duration,
            "steps_per_second":   float64(len(result.Steps)) / duration.Seconds(),
            "avg_step_duration":  duration / time.Duration(len(result.Steps)),
            "parallel_efficiency": oe.calculateParallelEfficiency(result.Steps),
        }
    }
    
    return result, err
}

func (oe *OptimizedExecutor) optimizeRequest(request core.Request) core.Request {
    optimized := request
    
    // Optimize tool selection for parallel execution
    optimized.Tools = oe.optimizeToolSet(request.Tools)
    
    // Add performance-aware stop conditions
    originalStopWhen := request.StopWhen
    optimized.StopWhen = func(stepNum int, step core.Step) bool {
        // Stop if step takes too long
        if step.Duration > oe.stepTimeout {
            log.Printf("Stopping due to slow step: %v", step.Duration)
            return true
        }
        
        // Check original condition
        if originalStopWhen != nil {
            return originalStopWhen(stepNum, step)
        }
        
        return false
    }
    
    return optimized
}

func (oe *OptimizedExecutor) calculateParallelEfficiency(steps []core.Step) float64 {
    if len(steps) == 0 {
        return 0
    }
    
    totalTools := 0
    parallelSteps := 0
    
    for _, step := range steps {
        toolCount := len(step.ToolCalls)
        totalTools += toolCount
        
        if toolCount > 1 {
            parallelSteps++
        }
    }
    
    if totalTools == 0 {
        return 0
    }
    
    return float64(parallelSteps) / float64(len(steps))
}
```

### Memory and Resource Management

```go
type ResourceManager struct {
    maxMemoryMB      int
    maxExecutionTime time.Duration
    activeSteps      map[string]*StepMonitor
    mutex            sync.RWMutex
}

type StepMonitor struct {
    StartTime   time.Time
    MemoryStart runtime.MemStats
    ToolCount   int
}

func (rm *ResourceManager) BeforeStep(stepID string, toolCount int) {
    rm.mutex.Lock()
    defer rm.mutex.Unlock()
    
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    rm.activeSteps[stepID] = &StepMonitor{
        StartTime:   time.Now(),
        MemoryStart: m,
        ToolCount:   toolCount,
    }
}

func (rm *ResourceManager) AfterStep(stepID string) StepMetrics {
    rm.mutex.Lock()
    defer rm.mutex.Unlock()
    
    monitor, exists := rm.activeSteps[stepID]
    if !exists {
        return StepMetrics{}
    }
    
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    metrics := StepMetrics{
        Duration:    time.Since(monitor.StartTime),
        MemoryUsed:  int64(m.Alloc - monitor.MemoryStart.Alloc),
        ToolCount:   monitor.ToolCount,
    }
    
    delete(rm.activeSteps, stepID)
    return metrics
}

func (rm *ResourceManager) CheckLimits() error {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    memoryMB := int(m.Alloc / 1024 / 1024)
    if memoryMB > rm.maxMemoryMB {
        return fmt.Errorf("memory limit exceeded: %dMB > %dMB", memoryMB, rm.maxMemoryMB)
    }
    
    rm.mutex.RLock()
    for stepID, monitor := range rm.activeSteps {
        if time.Since(monitor.StartTime) > rm.maxExecutionTime {
            rm.mutex.RUnlock()
            return fmt.Errorf("execution time limit exceeded for step %s", stepID)
        }
    }
    rm.mutex.RUnlock()
    
    return nil
}

type StepMetrics struct {
    Duration   time.Duration
    MemoryUsed int64
    ToolCount  int
}
```

## Complex Workflows

### Multi-Agent Coordination

```go
func multiAgentWorkflowExample() {
    // Define agents with specific roles
    researchAgent := createAgent("researcher", []tools.Handle{
        createWebSearchTool(),
        createDatabaseQueryTool(),
        createDocumentAnalysisTool(),
    })
    
    analysisAgent := createAgent("analyst", []tools.Handle{
        createDataAnalysisTool(),
        createStatisticalTool(),
        createVisualizationTool(),
    })
    
    reportingAgent := createAgent("reporter", []tools.Handle{
        createReportGeneratorTool(),
        createFormattingTool(),
        createDistributionTool(),
    })
    
    // Coordinate agents in workflow
    workflow := []AgentTask{
        {Agent: researchAgent, Task: "Research market trends in AI industry for Q4 2024"},
        {Agent: analysisAgent, Task: "Analyze research data and identify key patterns"},
        {Agent: reportingAgent, Task: "Create executive summary and distribute to stakeholders"},
    }
    
    results, err := executeMultiAgentWorkflow(ctx, workflow)
    if err != nil {
        log.Fatal(err)
    }
    
    // Combine results
    fmt.Println("Multi-Agent Workflow Results:")
    for i, result := range results {
        fmt.Printf("\nAgent %d (%s):\n", i+1, workflow[i].Agent.Name)
        fmt.Printf("Steps: %d\n", len(result.Steps))
        fmt.Printf("Summary: %s\n", truncate(result.Text, 200))
    }
}

type Agent struct {
    Name     string
    Role     string
    Provider core.Provider
    Tools    []tools.Handle
}

type AgentTask struct {
    Agent *Agent
    Task  string
}

func executeMultiAgentWorkflow(ctx context.Context, workflow []AgentTask) ([]*core.TextResult, error) {
    results := make([]*core.TextResult, len(workflow))
    var conversationHistory []core.Message
    
    for i, task := range workflow {
        // Build messages including previous agent results
        messages := []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: fmt.Sprintf("You are %s, a %s. Focus on your specific expertise.", 
                        task.Agent.Name, task.Agent.Role)},
                },
            },
        }
        
        // Add conversation history
        messages = append(messages, conversationHistory...)
        
        // Add current task
        messages = append(messages, core.Message{
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: task.Task},
            },
        })
        
        // Execute agent task
        result, err := task.Agent.Provider.GenerateText(ctx, core.Request{
            Messages: messages,
            Tools:    task.Agent.Tools,
            StopWhen: core.CombineConditions(
                core.MaxSteps(15),
                core.NoMoreTools(),
            ),
        })
        
        if err != nil {
            return nil, fmt.Errorf("agent %s failed: %w", task.Agent.Name, err)
        }
        
        results[i] = result
        
        // Add result to conversation history for next agent
        conversationHistory = append(conversationHistory, core.Message{
            Role: core.Assistant,
            Name: task.Agent.Name,
            Parts: []core.Part{
                core.Text{Text: result.Text},
            },
        })
    }
    
    return results, nil
}
```

### Hierarchical Task Decomposition

```go
func hierarchicalWorkflowExample() {
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: `You are a project manager AI. Break down complex tasks into subtasks.
                    
Available task management commands:
- CREATE_SUBTASK(name, description, priority)
- ASSIGN_RESOURCES(task, resources)  
- SET_DEPENDENCIES(task, dependencies)
- MARK_COMPLETE(task)
- ESCALATE(task, reason)`},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Plan and execute a complete product launch campaign including market research, product development, marketing strategy, and launch coordination."},
                },
            },
        },
        Tools: []tools.Handle{
            createTaskManagerTool(),
            createResourceAllocationTool(),
            createSchedulingTool(),
            createProgressTrackingTool(),
            createCommunicationTool(),
        },
        StopWhen: core.CombineConditions(
            core.MaxSteps(50),
            StopWhenGoalReached("LAUNCH_COMPLETE"),
        ),
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Analyze task hierarchy
    fmt.Println("Hierarchical Workflow Analysis:")
    analyzeTaskHierarchy(response.Steps)
}

func analyzeTaskHierarchy(steps []core.Step) {
    taskLevels := make(map[string]int)
    
    for _, step := range steps {
        for _, result := range step.ToolResults {
            if result.Name == "task_manager" {
                // Parse task creation and hierarchy
                var taskData map[string]any
                if err := json.Unmarshal(result.Result, &taskData); err == nil {
                    if action, ok := taskData["action"].(string); ok && action == "CREATE_SUBTASK" {
                        taskName := taskData["name"].(string)
                        parentTask := taskData["parent"].(string)
                        
                        parentLevel := taskLevels[parentTask]
                        taskLevels[taskName] = parentLevel + 1
                    }
                }
            }
        }
    }
    
    // Display hierarchy
    for task, level := range taskLevels {
        indent := strings.Repeat("  ", level)
        fmt.Printf("%s- %s (Level %d)\n", indent, task, level)
    }
}
```

## Monitoring and Observability

### Execution Metrics

```go
type ExecutionMetrics struct {
    WorkflowID      string            `json:"workflow_id"`
    StartTime       time.Time         `json:"start_time"`
    EndTime         time.Time         `json:"end_time"`
    TotalDuration   time.Duration     `json:"total_duration"`
    StepCount       int               `json:"step_count"`
    ToolCallCount   int               `json:"tool_call_count"`
    ErrorCount      int               `json:"error_count"`
    TokenUsage      Usage             `json:"token_usage"`
    StepMetrics     []StepMetrics     `json:"step_metrics"`
    PerformanceData map[string]any    `json:"performance_data"`
}

func (em *ExecutionMetrics) AddStep(step core.Step) {
    em.StepCount++
    em.ToolCallCount += len(step.ToolCalls)
    
    errorCount := 0
    for _, result := range step.ToolResults {
        if result.Error != nil {
            errorCount++
        }
    }
    em.ErrorCount += errorCount
    
    stepMetrics := StepMetrics{
        StepNumber:    em.StepCount,
        Duration:      step.Duration,
        ToolCount:     len(step.ToolCalls),
        ErrorCount:    errorCount,
        TokensUsed:    step.Usage.TotalTokens,
        ParallelTools: len(step.ToolCalls) > 1,
    }
    
    em.StepMetrics = append(em.StepMetrics, stepMetrics)
}

func (em *ExecutionMetrics) CalculateEfficiency() float64 {
    if em.StepCount == 0 {
        return 0
    }
    
    successfulSteps := em.StepCount - em.ErrorCount
    return float64(successfulSteps) / float64(em.StepCount)
}

func (em *ExecutionMetrics) GetAverageStepDuration() time.Duration {
    if em.StepCount == 0 {
        return 0
    }
    
    return em.TotalDuration / time.Duration(em.StepCount)
}

type ObservableExecutor struct {
    metrics *ExecutionMetrics
    logger  Logger
}

func (oe *ObservableExecutor) Execute(
    provider core.Provider,
    request core.Request,
) (*core.TextResult, error) {
    
    // Initialize metrics
    oe.metrics = &ExecutionMetrics{
        WorkflowID:      generateWorkflowID(),
        StartTime:       time.Now(),
        PerformanceData: make(map[string]any),
    }
    
    oe.logger.Info("Starting workflow", "workflow_id", oe.metrics.WorkflowID)
    
    // Execute with instrumentation
    result, err := oe.executeWithInstrumentation(provider, request)
    
    // Finalize metrics
    oe.metrics.EndTime = time.Now()
    oe.metrics.TotalDuration = oe.metrics.EndTime.Sub(oe.metrics.StartTime)
    
    if result != nil {
        oe.metrics.TokenUsage = result.Usage
        
        // Process each step
        for _, step := range result.Steps {
            oe.metrics.AddStep(step)
        }
    }
    
    // Log final metrics
    oe.logger.Info("Workflow completed",
        "workflow_id", oe.metrics.WorkflowID,
        "duration", oe.metrics.TotalDuration,
        "steps", oe.metrics.StepCount,
        "tool_calls", oe.metrics.ToolCallCount,
        "efficiency", oe.metrics.CalculateEfficiency(),
        "tokens", oe.metrics.TokenUsage.TotalTokens,
    )
    
    // Send metrics to monitoring system
    oe.sendMetricsToMonitoring()
    
    return result, err
}

func (oe *ObservableExecutor) sendMetricsToMonitoring() {
    // Send to your monitoring system (Prometheus, DataDog, etc.)
    metricsJSON, _ := json.Marshal(oe.metrics)
    
    // Example: Send to metrics endpoint
    http.Post("http://monitoring/metrics", "application/json", 
        bytes.NewBuffer(metricsJSON))
}
```

### Real-time Monitoring

```go
type WorkflowMonitor struct {
    activeWorkflows map[string]*WorkflowStatus
    mutex           sync.RWMutex
    subscribers     []chan WorkflowEvent
}

type WorkflowStatus struct {
    ID           string                 `json:"id"`
    Status       string                 `json:"status"`
    CurrentStep  int                    `json:"current_step"`
    TotalSteps   int                    `json:"total_steps,omitempty"`
    StartTime    time.Time             `json:"start_time"`
    LastActivity time.Time             `json:"last_activity"`
    Metadata     map[string]any        `json:"metadata"`
}

type WorkflowEvent struct {
    Type       string         `json:"type"`
    WorkflowID string         `json:"workflow_id"`
    Data       map[string]any `json:"data"`
    Timestamp  time.Time      `json:"timestamp"`
}

func (wm *WorkflowMonitor) StartWorkflow(workflowID string, metadata map[string]any) {
    wm.mutex.Lock()
    defer wm.mutex.Unlock()
    
    status := &WorkflowStatus{
        ID:           workflowID,
        Status:       "running",
        CurrentStep:  0,
        StartTime:    time.Now(),
        LastActivity: time.Now(),
        Metadata:     metadata,
    }
    
    wm.activeWorkflows[workflowID] = status
    
    wm.broadcast(WorkflowEvent{
        Type:       "workflow_started",
        WorkflowID: workflowID,
        Data:       map[string]any{"metadata": metadata},
        Timestamp:  time.Now(),
    })
}

func (wm *WorkflowMonitor) UpdateWorkflow(workflowID string, step int, data map[string]any) {
    wm.mutex.Lock()
    defer wm.mutex.Unlock()
    
    if status, exists := wm.activeWorkflows[workflowID]; exists {
        status.CurrentStep = step
        status.LastActivity = time.Now()
        
        for k, v := range data {
            status.Metadata[k] = v
        }
        
        wm.broadcast(WorkflowEvent{
            Type:       "workflow_updated",
            WorkflowID: workflowID,
            Data:       data,
            Timestamp:  time.Now(),
        })
    }
}

func (wm *WorkflowMonitor) CompleteWorkflow(workflowID string, result map[string]any) {
    wm.mutex.Lock()
    defer wm.mutex.Unlock()
    
    if status, exists := wm.activeWorkflows[workflowID]; exists {
        status.Status = "completed"
        status.LastActivity = time.Now()
        
        wm.broadcast(WorkflowEvent{
            Type:       "workflow_completed",
            WorkflowID: workflowID,
            Data:       result,
            Timestamp:  time.Now(),
        })
        
        delete(wm.activeWorkflows, workflowID)
    }
}

func (wm *WorkflowMonitor) Subscribe() <-chan WorkflowEvent {
    wm.mutex.Lock()
    defer wm.mutex.Unlock()
    
    ch := make(chan WorkflowEvent, 100)
    wm.subscribers = append(wm.subscribers, ch)
    
    return ch
}

func (wm *WorkflowMonitor) broadcast(event WorkflowEvent) {
    for _, ch := range wm.subscribers {
        select {
        case ch <- event:
        default:
            // Channel full, skip this subscriber
        }
    }
}

// Health check endpoint
func (wm *WorkflowMonitor) GetActiveWorkflows() []WorkflowStatus {
    wm.mutex.RLock()
    defer wm.mutex.RUnlock()
    
    var workflows []WorkflowStatus
    for _, status := range wm.activeWorkflows {
        workflows = append(workflows, *status)
    }
    
    return workflows
}
```

## Best Practices

### 1. Design for Observability

```go
func observableWorkflow(provider core.Provider) {
    // Create workflow with comprehensive logging
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: messages,
        Tools:    tools,
        StopWhen: func(stepNum int, step core.Step) bool {
            // Log each step for debugging
            log.Printf("Step %d: %d tools, %v duration, %d tokens",
                stepNum, len(step.ToolCalls), step.Duration, step.Usage.TotalTokens)
            
            // Log tool execution details
            for _, call := range step.ToolCalls {
                log.Printf("  Tool: %s, Input: %s", call.Name, string(call.Input))
            }
            
            for _, result := range step.ToolResults {
                if result.Error != nil {
                    log.Printf("  Error in %s: %v", result.Name, result.Error)
                } else {
                    log.Printf("  Success: %s -> %s", result.Name, truncate(string(result.Result), 100))
                }
            }
            
            // Apply original stop condition
            return stepNum >= 20 || len(step.ToolCalls) == 0
        },
        Metadata: map[string]any{
            "workflow_type": "data_analysis",
            "user_id":      "user123",
            "session_id":   "session456",
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Log final results
    log.Printf("Workflow completed: %d steps, %d tokens, %v duration",
        len(response.Steps), response.Usage.TotalTokens, time.Since(start))
}
```

### 2. Implement Proper Error Handling

```go
func robustWorkflow(provider core.Provider) (*core.TextResult, error) {
    var lastError error
    maxAttempts := 3
    
    for attempt := 1; attempt <= maxAttempts; attempt++ {
        result, err := executeWithRecovery(provider, core.Request{
            Messages: messages,
            Tools:    tools,
            StopWhen: core.CombineConditions(
                core.MaxSteps(25),
                StopWhenErrorRateHigh(0.5), // Stop if >50% tools fail
                StopWhenTimeExceeded(10*time.Minute),
            ),
        })
        
        if err == nil {
            return result, nil
        }
        
        lastError = err
        log.Printf("Attempt %d failed: %v", attempt, err)
        
        // Don't retry certain errors
        if isFatalError(err) {
            break
        }
        
        // Exponential backoff
        if attempt < maxAttempts {
            backoff := time.Duration(attempt*attempt) * time.Second
            log.Printf("Retrying in %v...", backoff)
            time.Sleep(backoff)
        }
    }
    
    return nil, fmt.Errorf("workflow failed after %d attempts: %w", maxAttempts, lastError)
}

func executeWithRecovery(provider core.Provider, request core.Request) (*core.TextResult, error) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Recovered from panic in workflow: %v", r)
        }
    }()
    
    // Add timeout context
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
    defer cancel()
    
    return provider.GenerateText(ctx, request)
}
```

### 3. Optimize for Performance

```go
func optimizedWorkflow(provider core.Provider) {
    // Pre-warm any expensive resources
    prepareResources()
    
    // Use connection pooling for external services
    httpClient := &http.Client{
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     30 * time.Second,
        },
        Timeout: 30 * time.Second,
    }
    
    // Create optimized tools
    tools := []tools.Handle{
        createOptimizedTool(httpClient),
        createCachedTool(cache),
        createPooledDatabaseTool(dbPool),
    }
    
    // Execute with performance monitoring
    start := time.Now()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: messages,
        Tools:    tools,
        StopWhen: func(stepNum int, step core.Step) bool {
            // Performance-based stop conditions
            if step.Duration > 30*time.Second {
                log.Printf("Stopping due to slow step: %v", step.Duration)
                return true
            }
            
            elapsed := time.Since(start)
            if elapsed > 5*time.Minute {
                log.Printf("Stopping due to total time limit")
                return true
            }
            
            return stepNum >= 15 || len(step.ToolCalls) == 0
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Performance analysis
    totalDuration := time.Since(start)
    avgStepDuration := totalDuration / time.Duration(len(response.Steps))
    
    log.Printf("Performance metrics:")
    log.Printf("  Total duration: %v", totalDuration)
    log.Printf("  Steps: %d", len(response.Steps))
    log.Printf("  Avg step duration: %v", avgStepDuration)
    log.Printf("  Tokens/second: %.1f", 
        float64(response.Usage.TotalTokens)/totalDuration.Seconds())
}
```

## Advanced Patterns

### Workflow Templates

```go
type WorkflowTemplate struct {
    Name        string                `json:"name"`
    Description string                `json:"description"`
    Steps       []TemplateStep        `json:"steps"`
    DefaultVars map[string]any       `json:"default_vars"`
    StopWhen    func(int, core.Step) bool `json:"-"`
}

type TemplateStep struct {
    Name         string            `json:"name"`
    SystemPrompt string            `json:"system_prompt"`
    UserPrompt   string            `json:"user_prompt"`
    RequiredVars []string          `json:"required_vars"`
    Tools        []string          `json:"tools"`
    MaxSteps     int               `json:"max_steps"`
}

func (wt *WorkflowTemplate) Execute(
    provider core.Provider,
    variables map[string]any,
    toolRegistry map[string]tools.Handle,
) (*WorkflowResult, error) {
    
    // Merge variables with defaults
    vars := make(map[string]any)
    for k, v := range wt.DefaultVars {
        vars[k] = v
    }
    for k, v := range variables {
        vars[k] = v
    }
    
    // Validate required variables
    if err := wt.validateVariables(vars); err != nil {
        return nil, fmt.Errorf("variable validation failed: %w", err)
    }
    
    var stepResults []StepResult
    
    for i, templateStep := range wt.Steps {
        // Resolve tools for this step
        var stepTools []tools.Handle
        for _, toolName := range templateStep.Tools {
            if tool, exists := toolRegistry[toolName]; exists {
                stepTools = append(stepTools, tool)
            }
        }
        
        // Build messages with variable substitution
        messages := []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: substituteVariables(templateStep.SystemPrompt, vars)},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: substituteVariables(templateStep.UserPrompt, vars)},
                },
            },
        }
        
        // Execute step
        result, err := provider.GenerateText(ctx, core.Request{
            Messages: messages,
            Tools:    stepTools,
            StopWhen: core.MaxSteps(templateStep.MaxSteps),
        })
        
        if err != nil {
            return nil, fmt.Errorf("step %d (%s) failed: %w", i+1, templateStep.Name, err)
        }
        
        stepResult := StepResult{
            Name:     templateStep.Name,
            Result:   result,
            Duration: time.Since(time.Now()), // This would be tracked properly
        }
        
        stepResults = append(stepResults, stepResult)
        
        // Update variables with step results for next steps
        vars[templateStep.Name+"_result"] = result.Text
    }
    
    return &WorkflowResult{
        Template:    wt.Name,
        Steps:       stepResults,
        Variables:   vars,
        TotalSteps:  len(stepResults),
        Success:     true,
    }, nil
}

// Example: Data Analysis Template
func createDataAnalysisTemplate() *WorkflowTemplate {
    return &WorkflowTemplate{
        Name:        "data_analysis",
        Description: "Comprehensive data analysis workflow",
        Steps: []TemplateStep{
            {
                Name:         "data_collection",
                SystemPrompt: "You are a data collection specialist. Gather data from specified sources.",
                UserPrompt:   "Collect data from: {{.data_sources}}. Focus on: {{.focus_areas}}",
                RequiredVars: []string{"data_sources", "focus_areas"},
                Tools:        []string{"database_query", "api_fetch", "file_reader"},
                MaxSteps:     10,
            },
            {
                Name:         "data_processing",
                SystemPrompt: "You are a data processing expert. Clean and transform the collected data.",
                UserPrompt:   "Process the collected data: {{.data_collection_result}}. Apply transformations: {{.transformations}}",
                RequiredVars: []string{"transformations"},
                Tools:        []string{"data_cleaner", "transformer", "validator"},
                MaxSteps:     8,
            },
            {
                Name:         "analysis",
                SystemPrompt: "You are a data analyst. Perform statistical analysis and identify insights.",
                UserPrompt:   "Analyze the processed data: {{.data_processing_result}}. Generate insights for: {{.analysis_goals}}",
                RequiredVars: []string{"analysis_goals"},
                Tools:        []string{"statistical_analysis", "pattern_finder", "trend_analyzer"},
                MaxSteps:     12,
            },
            {
                Name:         "reporting",
                SystemPrompt: "You are a report writer. Create comprehensive reports from analysis results.",
                UserPrompt:   "Create a report from analysis: {{.analysis_result}}. Target audience: {{.audience}}",
                RequiredVars: []string{"audience"},
                Tools:        []string{"report_generator", "chart_creator", "formatter"},
                MaxSteps:     6,
            },
        },
        DefaultVars: map[string]any{
            "transformations": []string{"normalize", "deduplicate", "validate"},
            "audience":        "executives",
        },
    }
}
```

### Workflow Composition

```go
func composeWorkflows(templates []*WorkflowTemplate) *CompositeWorkflow {
    return &CompositeWorkflow{
        Templates:    templates,
        Dependencies: make(map[string][]string),
    }
}

type CompositeWorkflow struct {
    Templates    []*WorkflowTemplate
    Dependencies map[string][]string
    Results      map[string]*WorkflowResult
}

func (cw *CompositeWorkflow) AddDependency(workflow, dependency string) {
    cw.Dependencies[workflow] = append(cw.Dependencies[workflow], dependency)
}

func (cw *CompositeWorkflow) Execute(
    provider core.Provider,
    variables map[string]any,
    toolRegistry map[string]tools.Handle,
) error {
    
    cw.Results = make(map[string]*WorkflowResult)
    executed := make(map[string]bool)
    
    // Topological sort for execution order
    executionOrder, err := cw.getExecutionOrder()
    if err != nil {
        return fmt.Errorf("dependency resolution failed: %w", err)
    }
    
    for _, templateName := range executionOrder {
        template := cw.findTemplate(templateName)
        if template == nil {
            return fmt.Errorf("template not found: %s", templateName)
        }
        
        // Merge variables from dependent workflows
        workflowVars := make(map[string]any)
        for k, v := range variables {
            workflowVars[k] = v
        }
        
        // Add results from dependencies
        for _, dep := range cw.Dependencies[templateName] {
            if result, exists := cw.Results[dep]; exists {
                workflowVars[dep+"_result"] = result
            }
        }
        
        // Execute workflow
        result, err := template.Execute(provider, workflowVars, toolRegistry)
        if err != nil {
            return fmt.Errorf("workflow %s failed: %w", templateName, err)
        }
        
        cw.Results[templateName] = result
        executed[templateName] = true
        
        log.Printf("Completed workflow: %s (%d steps)", templateName, result.TotalSteps)
    }
    
    return nil
}
```

## Summary

GAI's multi-step execution system provides:

1. **Complex Problem Solving**: Break down large tasks into manageable steps
2. **Dynamic Control**: Sophisticated stop conditions and flow control
3. **Error Recovery**: Comprehensive error handling and retry mechanisms
4. **Performance Optimization**: Parallel execution and resource management
5. **Observability**: Complete execution tracking and monitoring
6. **Workflow Patterns**: Templates, composition, and reusable patterns

Key benefits:
- **Scalability**: Handle complex, long-running workflows
- **Reliability**: Built-in error recovery and resilience
- **Flexibility**: Customizable execution patterns and stop conditions
- **Efficiency**: Parallel tool execution and resource optimization
- **Maintainability**: Template-based workflows and composition patterns

The multi-step execution system enables sophisticated AI applications that can perform complex reasoning, coordinate multiple tools, and adapt their approach based on intermediate results, making it possible to build production-grade AI systems that can handle real-world complexity.

Next steps:
- [Tools](./tools.md) - Deep dive into the tools system
- [Streaming](./streaming.md) - Real-time response handling
- [Architecture](./architecture.md) - Overall system design