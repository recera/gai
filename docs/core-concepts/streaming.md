# Streaming and Real-Time Responses

This guide provides a comprehensive understanding of GAI's streaming system, enabling real-time AI responses, live tool execution, and efficient handling of long-running operations.

## Table of Contents
- [Overview](#overview)
- [Streaming Architecture](#streaming-architecture)
- [Event System](#event-system)
- [Text Streaming](#text-streaming)
- [Structured Output Streaming](#structured-output-streaming)
- [Tool Execution Streaming](#tool-execution-streaming)
- [Error Handling](#error-handling)
- [Performance Optimization](#performance-optimization)
- [Streaming Patterns](#streaming-patterns)
- [Best Practices](#best-practices)
- [Advanced Features](#advanced-features)

## Overview

GAI's streaming system enables real-time, event-driven communication with AI providers. Instead of waiting for complete responses, applications can process results as they arrive, providing better user experience and enabling interactive applications.

### Key Benefits

```go
// Non-streaming: Wait for complete response
response, err := provider.GenerateText(ctx, request)
fmt.Println(response.Text) // All text at once

// Streaming: Process response as it arrives
stream, err := provider.StreamText(ctx, request)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for event := range stream.Events() {
    switch event.Type {
    case core.EventTextDelta:
        fmt.Print(event.TextDelta) // Text chunk by chunk
    case core.EventFinish:
        fmt.Println("\n[Complete]")
    }
}
```

### Streaming Capabilities

- **Text Generation**: Real-time text streaming with delta updates
- **Structured Outputs**: Incremental JSON object construction
- **Tool Execution**: Live tool call execution and results
- **Multi-Step Workflows**: Stream complex multi-step operations
- **Error Recovery**: Graceful handling of streaming errors

## Streaming Architecture

### Stream Interface

All streams implement the core streaming interface:

```go
type TextStream interface {
    Events() <-chan Event  // Read-only channel for events
    Close() error          // Clean shutdown
}

type ObjectStream[T any] interface {
    Events() <-chan Event  // Read-only channel for events
    Close() error          // Clean shutdown
}
```

### Event-Driven Design

Streaming uses an event-driven architecture:

```
AI Provider → Protocol Adapter → Event Pipeline → Application

[SSE/NDJSON]   [Parse & Convert]   [Channel Events]   [Process Events]
```

### Channel-Based Flow Control

GAI uses Go channels for natural flow control:
- **Backpressure**: Slow consumers automatically slow down producers
- **Cancellation**: Context-based cancellation propagates through streams
- **Multiplexing**: Multiple goroutines can process events concurrently

## Event System

### Core Event Types

```go
type Event struct {
    Type       EventType      // Event classification
    TextDelta  string         // Text chunk (for text events)
    Object     any           // Partial object (for object events)
    ToolCall   *ToolCall     // Tool execution request
    ToolResult *ToolResult   // Tool execution result
    Citations  []Citation    // Source citations
    Safety     *SafetyEvent  // Content safety information
    Usage      *Usage        // Token usage stats
    Error      error         // Error information
    Metadata   map[string]any // Provider-specific data
}

type EventType int
const (
    EventStart        EventType = iota // Stream started
    EventTextDelta                     // Text chunk received
    EventObjectDelta                   // Object chunk received
    EventToolCall                      // AI requested tool execution
    EventToolResult                    // Tool execution completed
    EventCitations                     // Source citations provided
    EventSafety                       // Safety/moderation event
    EventUsage                        // Token usage information
    EventError                        // Error occurred
    EventFinish                       // Stream completed
)
```

### Event Processing Pattern

```go
func processStream(stream core.TextStream) error {
    defer stream.Close()
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventStart:
            fmt.Println("🚀 Stream started")
            
        case core.EventTextDelta:
            // Process text chunk
            fmt.Print(event.TextDelta)
            
        case core.EventToolCall:
            // AI is calling a tool
            fmt.Printf("\n🔧 Calling tool: %s\n", event.ToolCall.Name)
            
        case core.EventToolResult:
            // Tool execution completed
            fmt.Printf("✅ Tool result: %s\n", event.ToolResult.Result)
            
        case core.EventUsage:
            // Token usage information
            fmt.Printf("📊 Tokens: %d\n", event.Usage.TotalTokens)
            
        case core.EventError:
            // Handle streaming errors
            return fmt.Errorf("stream error: %w", event.Error)
            
        case core.EventFinish:
            fmt.Println("\n✨ Stream complete")
            return nil
        }
    }
    
    return nil
}
```

## Text Streaming

### Basic Text Streaming

```go
func basicTextStreaming(provider core.Provider) {
    stream, err := provider.StreamText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a short story about a robot learning to paint."},
                },
            },
        },
        MaxTokens:   1000,
        Temperature: 0.7,
    })
    
    if err != nil {
        log.Fatal("Failed to start stream:", err)
    }
    defer stream.Close()
    
    var fullText strings.Builder
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            // Display text in real-time
            fmt.Print(event.TextDelta)
            fullText.WriteString(event.TextDelta)
            
        case core.EventError:
            log.Printf("Stream error: %v", event.Error)
            
        case core.EventFinish:
            fmt.Println("\n\n[Story Complete]")
            fmt.Printf("Total length: %d characters\n", fullText.Len())
        }
    }
}
```

### Interactive Streaming

```go
func interactiveStreaming(provider core.Provider) {
    reader := bufio.NewReader(os.Stdin)
    conversation := []core.Message{
        {
            Role: core.System,
            Parts: []core.Part{
                core.Text{Text: "You are a helpful assistant. Provide concise, helpful responses."},
            },
        },
    }
    
    for {
        fmt.Print("You: ")
        userInput, _ := reader.ReadString('\n')
        userInput = strings.TrimSpace(userInput)
        
        if userInput == "exit" {
            break
        }
        
        // Add user message
        conversation = append(conversation, core.Message{
            Role: core.User,
            Parts: []core.Part{core.Text{Text: userInput}},
        })
        
        // Stream response
        fmt.Print("Assistant: ")
        
        stream, err := provider.StreamText(context.Background(), core.Request{
            Messages:  conversation,
            MaxTokens: 500,
        })
        
        if err != nil {
            log.Printf("Error: %v\n", err)
            continue
        }
        
        var assistantResponse strings.Builder
        
        for event := range stream.Events() {
            switch event.Type {
            case core.EventTextDelta:
                fmt.Print(event.TextDelta)
                assistantResponse.WriteString(event.TextDelta)
                
            case core.EventFinish:
                fmt.Println("\n")
                
                // Add assistant response to conversation
                conversation = append(conversation, core.Message{
                    Role: core.Assistant,
                    Parts: []core.Part{core.Text{Text: assistantResponse.String()}},
                })
                
            case core.EventError:
                log.Printf("Stream error: %v\n", event.Error)
            }
        }
        
        stream.Close()
    }
}
```

### Streaming with Progress Tracking

```go
func streamingWithProgress(provider core.Provider) {
    stream, err := provider.StreamText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a detailed technical analysis of distributed systems architecture, covering scalability, reliability, and performance considerations."},
                },
            },
        },
        MaxTokens: 2000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    var (
        charCount   int
        wordCount   int
        startTime   = time.Now()
        lastUpdate  = time.Now()
    )
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            fmt.Print(event.TextDelta)
            
            // Update counters
            charCount += len(event.TextDelta)
            wordCount += len(strings.Fields(event.TextDelta))
            
            // Show progress every second
            if time.Since(lastUpdate) > time.Second {
                elapsed := time.Since(startTime)
                charsPerSec := float64(charCount) / elapsed.Seconds()
                
                fmt.Printf("\n[📊 Progress: %d chars, %d words, %.1f chars/sec]\n", 
                    charCount, wordCount, charsPerSec)
                
                lastUpdate = time.Now()
            }
            
        case core.EventUsage:
            if event.Usage != nil {
                fmt.Printf("\n[📈 Tokens: %d input, %d output, %d total]\n", 
                    event.Usage.InputTokens, event.Usage.OutputTokens, event.Usage.TotalTokens)
            }
            
        case core.EventFinish:
            elapsed := time.Since(startTime)
            fmt.Printf("\n\n✨ Complete! %d characters in %v (%.1f chars/sec)\n", 
                charCount, elapsed, float64(charCount)/elapsed.Seconds())
        }
    }
}
```

## Structured Output Streaming

### Streaming JSON Objects

```go
type AnalysisResult struct {
    Summary     string            `json:"summary"`
    KeyPoints   []string          `json:"key_points"`
    Confidence  float64           `json:"confidence"`
    Metadata    map[string]string `json:"metadata"`
}

func streamStructuredOutput(provider core.Provider) {
    stream, err := provider.StreamObject(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Analyze this market data and provide a structured summary with key insights."},
                },
            },
        },
    }, AnalysisResult{})
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    var partialObject AnalysisResult
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventObjectDelta:
            // Update partial object
            if err := json.Unmarshal(event.Object.([]byte), &partialObject); err == nil {
                // Display current state
                fmt.Printf("\r🔄 Building analysis... Confidence: %.1f%%", partialObject.Confidence*100)
            }
            
        case core.EventFinish:
            // Final structured result
            fmt.Printf("\n✅ Analysis Complete!\n")
            fmt.Printf("Summary: %s\n", partialObject.Summary)
            fmt.Printf("Key Points: %v\n", partialObject.KeyPoints)
            fmt.Printf("Confidence: %.1f%%\n", partialObject.Confidence*100)
            
        case core.EventError:
            log.Printf("Streaming error: %v\n", event.Error)
        }
    }
}
```

### Incremental Data Processing

```go
func streamIncrementalData(provider core.Provider) {
    stream, err := provider.StreamObject(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Process this dataset and provide incremental results as they become available."},
                },
            },
        },
    }, map[string]any{})
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    processed := make(map[string]any)
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventObjectDelta:
            // Merge incremental updates
            if delta, ok := event.Object.(map[string]any); ok {
                for key, value := range delta {
                    processed[key] = value
                    
                    // Process each field as it becomes available
                    switch key {
                    case "records_processed":
                        fmt.Printf("📋 Records processed: %v\n", value)
                    case "errors_found":
                        fmt.Printf("⚠️  Errors found: %v\n", value)
                    case "completion_percentage":
                        if pct, ok := value.(float64); ok {
                            fmt.Printf("📊 Progress: %.1f%%\n", pct)
                        }
                    }
                }
            }
            
        case core.EventFinish:
            fmt.Println("✅ Processing complete!")
            prettyPrint(processed)
        }
    }
}
```

## Tool Execution Streaming

### Real-Time Tool Execution

```go
func streamToolExecution(provider core.Provider) {
    tools := []tools.Handle{
        createWeatherTool(),
        createDatabaseQueryTool(),
        createAnalysisTool(),
    }
    
    stream, err := provider.StreamText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Get weather for NYC, query user growth data, and analyze trends."},
                },
            },
        },
        Tools: tools,
        ToolChoice: core.ToolAuto,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    var (
        toolsExecuting = make(map[string]bool)
        executionOrder []string
    )
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            fmt.Print(event.TextDelta)
            
        case core.EventToolCall:
            toolName := event.ToolCall.Name
            fmt.Printf("\n🔧 Executing: %s\n", toolName)
            fmt.Printf("   Input: %s\n", string(event.ToolCall.Input))
            
            toolsExecuting[toolName] = true
            executionOrder = append(executionOrder, toolName)
            
        case core.EventToolResult:
            toolName := event.ToolResult.Name
            fmt.Printf("✅ %s completed\n", toolName)
            fmt.Printf("   Result: %s\n", truncate(string(event.ToolResult.Result), 100))
            
            delete(toolsExecuting, toolName)
            
            // Show currently executing tools
            if len(toolsExecuting) > 0 {
                executing := make([]string, 0, len(toolsExecuting))
                for name := range toolsExecuting {
                    executing = append(executing, name)
                }
                fmt.Printf("🔄 Still running: %v\n", executing)
            }
            
        case core.EventFinish:
            fmt.Printf("\n✨ All tools completed!\n")
            fmt.Printf("Execution order: %v\n", executionOrder)
        }
    }
}
```

### Multi-Step Workflow Streaming

```go
func streamMultiStepWorkflow(provider core.Provider) {
    tools := []tools.Handle{
        createWebSearchTool(),
        createDataAnalysisTool(),
        createReportGeneratorTool(),
        createEmailTool(),
    }
    
    stream, err := provider.StreamText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are a research assistant. Execute tasks step by step and provide updates."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Research AI market trends, analyze the data, create a report, and email it to team@company.com"},
                },
            },
        },
        Tools: tools,
        StopWhen: core.CombineConditions(
            core.MaxSteps(15),
            core.UntilToolSeen("send_email"),
        ),
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    var (
        stepCount      int
        completedTools []string
        workflowStart  = time.Now()
    )
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            fmt.Print(event.TextDelta)
            
        case core.EventToolCall:
            stepCount++
            fmt.Printf("\n📍 Step %d: %s\n", stepCount, event.ToolCall.Name)
            
            elapsed := time.Since(workflowStart)
            fmt.Printf("   ⏱️  Elapsed: %v\n", elapsed)
            
        case core.EventToolResult:
            toolName := event.ToolResult.Name
            completedTools = append(completedTools, toolName)
            
            fmt.Printf("   ✅ %s completed\n", toolName)
            
            // Show progress
            progress := float64(len(completedTools)) / float64(stepCount) * 100
            fmt.Printf("   📊 Progress: %.1f%% (%d/%d tools)\n", 
                progress, len(completedTools), stepCount)
            
        case core.EventFinish:
            totalTime := time.Since(workflowStart)
            fmt.Printf("\n🎉 Workflow completed in %v!\n", totalTime)
            fmt.Printf("Total steps: %d\n", stepCount)
            fmt.Printf("Tools used: %v\n", completedTools)
            
        case core.EventError:
            fmt.Printf("\n❌ Workflow error: %v\n", event.Error)
        }
    }
}
```

## Error Handling

### Graceful Error Recovery

```go
func streamWithErrorRecovery(provider core.Provider) error {
    maxRetries := 3
    retryDelay := time.Second
    
    for attempt := 1; attempt <= maxRetries; attempt++ {
        stream, err := provider.StreamText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: "Generate a comprehensive report on cloud computing trends."},
                    },
                },
            },
        })
        
        if err != nil {
            log.Printf("Attempt %d failed to create stream: %v", attempt, err)
            if attempt < maxRetries {
                time.Sleep(retryDelay * time.Duration(attempt))
                continue
            }
            return fmt.Errorf("all attempts failed: %w", err)
        }
        
        // Process stream with error handling
        err = processStreamSafely(stream)
        if err == nil {
            return nil // Success
        }
        
        log.Printf("Attempt %d stream failed: %v", attempt, err)
        stream.Close()
        
        if attempt < maxRetries {
            time.Sleep(retryDelay * time.Duration(attempt))
        }
    }
    
    return fmt.Errorf("all streaming attempts failed")
}

func processStreamSafely(stream core.TextStream) error {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Stream panic recovered: %v", r)
        }
        stream.Close()
    }()
    
    timeout := time.NewTimer(5 * time.Minute)
    defer timeout.Stop()
    
    for {
        select {
        case event, ok := <-stream.Events():
            if !ok {
                return nil // Stream closed normally
            }
            
            switch event.Type {
            case core.EventError:
                if isRecoverableError(event.Error) {
                    log.Printf("Recoverable error: %v", event.Error)
                    continue
                }
                return fmt.Errorf("non-recoverable error: %w", event.Error)
                
            case core.EventTextDelta:
                // Process text safely
                if err := processTextSafely(event.TextDelta); err != nil {
                    return fmt.Errorf("text processing failed: %w", err)
                }
                
            case core.EventFinish:
                return nil
            }
            
        case <-timeout.C:
            return fmt.Errorf("stream timeout")
        }
    }
}
```

### Error Classification

```go
func isRecoverableError(err error) bool {
    // Check for temporary/recoverable errors
    var netErr net.Error
    if errors.As(err, &netErr) && netErr.Temporary() {
        return true
    }
    
    // Rate limiting errors
    if strings.Contains(err.Error(), "rate limit") ||
       strings.Contains(err.Error(), "quota exceeded") {
        return true
    }
    
    // Temporary service errors
    if strings.Contains(err.Error(), "temporary") ||
       strings.Contains(err.Error(), "try again") {
        return true
    }
    
    return false
}

func handleStreamError(err error, attempt int) (retry bool, delay time.Duration) {
    switch {
    case strings.Contains(err.Error(), "rate limit"):
        // Exponential backoff for rate limits
        return true, time.Second * time.Duration(1<<uint(attempt))
        
    case strings.Contains(err.Error(), "timeout"):
        // Short delay for timeouts
        return true, time.Millisecond * 500
        
    case strings.Contains(err.Error(), "connection"):
        // Network issues
        return true, time.Second * time.Duration(attempt)
        
    default:
        // Non-recoverable error
        return false, 0
    }
}
```

## Performance Optimization

### Efficient Event Processing

```go
func optimizedStreaming(provider core.Provider) {
    stream, err := provider.StreamText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Generate a large document with multiple sections."},
                },
            },
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    // Use buffered channels for processing
    textProcessor := make(chan string, 100)
    
    // Background text processing
    go func() {
        var builder strings.Builder
        builder.Grow(10000) // Pre-allocate capacity
        
        for text := range textProcessor {
            builder.WriteString(text)
            
            // Batch process every 1KB
            if builder.Len() > 1024 {
                processTextBatch(builder.String())
                builder.Reset()
            }
        }
        
        // Process remaining text
        if builder.Len() > 0 {
            processTextBatch(builder.String())
        }
    }()
    
    // Stream processing with batching
    eventBuffer := make([]core.Event, 0, 10)
    
    for event := range stream.Events() {
        eventBuffer = append(eventBuffer, event)
        
        // Process in batches for efficiency
        if len(eventBuffer) >= 10 || event.Type == core.EventFinish {
            processBatchEvents(eventBuffer, textProcessor)
            eventBuffer = eventBuffer[:0] // Reset slice
        }
    }
    
    close(textProcessor)
}

func processBatchEvents(events []core.Event, textCh chan<- string) {
    for _, event := range events {
        switch event.Type {
        case core.EventTextDelta:
            select {
            case textCh <- event.TextDelta:
            default:
                // Channel full, process synchronously
                processTextBatch(event.TextDelta)
            }
        case core.EventFinish:
            fmt.Println("Stream finished")
        }
    }
}
```

### Memory Management

```go
type StreamProcessor struct {
    eventPool  sync.Pool
    bufferPool sync.Pool
}

func NewStreamProcessor() *StreamProcessor {
    return &StreamProcessor{
        eventPool: sync.Pool{
            New: func() any {
                return &core.Event{}
            },
        },
        bufferPool: sync.Pool{
            New: func() any {
                buf := make([]byte, 0, 1024)
                return &buf
            },
        },
    }
}

func (sp *StreamProcessor) ProcessStream(stream core.TextStream) error {
    defer stream.Close()
    
    for event := range stream.Events() {
        // Get buffer from pool
        buf := sp.bufferPool.Get().(*[]byte)
        *buf = (*buf)[:0] // Reset length
        
        switch event.Type {
        case core.EventTextDelta:
            // Use pooled buffer for processing
            *buf = append(*buf, event.TextDelta...)
            sp.processText(*buf)
            
        case core.EventFinish:
            sp.bufferPool.Put(buf)
            return nil
        }
        
        // Return buffer to pool
        sp.bufferPool.Put(buf)
    }
    
    return nil
}
```

## Streaming Patterns

### Producer-Consumer Pattern

```go
func producerConsumerStreaming(provider core.Provider) {
    // Create processing pipeline
    textChan := make(chan string, 50)
    processingChan := make(chan ProcessedText, 20)
    outputChan := make(chan FinalOutput, 10)
    
    // Start consumers
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    var wg sync.WaitGroup
    
    // Text processor
    wg.Add(1)
    go func() {
        defer wg.Done()
        defer close(processingChan)
        
        for text := range textChan {
            processed := processText(text)
            select {
            case processingChan <- processed:
            case <-ctx.Done():
                return
            }
        }
    }()
    
    // Analysis processor
    wg.Add(1)
    go func() {
        defer wg.Done()
        defer close(outputChan)
        
        for processed := range processingChan {
            analyzed := analyzeText(processed)
            select {
            case outputChan <- analyzed:
            case <-ctx.Done():
                return
            }
        }
    }()
    
    // Output handler
    wg.Add(1)
    go func() {
        defer wg.Done()
        
        for output := range outputChan {
            displayOutput(output)
        }
    }()
    
    // Producer (stream reader)
    stream, err := provider.StreamText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Generate detailed analysis of current market conditions."},
                },
            },
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            select {
            case textChan <- event.TextDelta:
            case <-ctx.Done():
                return
            }
            
        case core.EventFinish:
            close(textChan)
            wg.Wait() // Wait for all processors to complete
            return
        }
    }
}
```

### Fan-Out Pattern

```go
func fanOutStreaming(provider core.Provider) {
    stream, err := provider.StreamText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Generate comprehensive report on multiple topics."},
                },
            },
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    // Create multiple processing channels
    channels := []chan string{
        make(chan string, 20), // Sentiment analysis
        make(chan string, 20), // Keyword extraction  
        make(chan string, 20), // Language detection
        make(chan string, 20), // Summary generation
    }
    
    processors := []func(string){
        analyzeSentiment,
        extractKeywords,
        detectLanguage,
        generateSummary,
    }
    
    // Start processors
    var wg sync.WaitGroup
    for i, ch := range channels {
        wg.Add(1)
        go func(ch chan string, processor func(string)) {
            defer wg.Done()
            for text := range ch {
                processor(text)
            }
        }(ch, processors[i])
    }
    
    // Fan out events to all processors
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            for _, ch := range channels {
                select {
                case ch <- event.TextDelta:
                default:
                    // Channel full, skip this processor
                }
            }
            
        case core.EventFinish:
            // Close all channels
            for _, ch := range channels {
                close(ch)
            }
            
            // Wait for all processors
            wg.Wait()
            return
        }
    }
}
```

## Best Practices

### 1. Always Close Streams

```go
// Good: Always close streams
func goodStreamHandling(provider core.Provider) error {
    stream, err := provider.StreamText(ctx, request)
    if err != nil {
        return err
    }
    defer stream.Close() // Always close
    
    for event := range stream.Events() {
        // Process events
    }
    
    return nil
}

// Better: Close with error handling
func betterStreamHandling(provider core.Provider) error {
    stream, err := provider.StreamText(ctx, request)
    if err != nil {
        return err
    }
    
    defer func() {
        if closeErr := stream.Close(); closeErr != nil {
            log.Printf("Error closing stream: %v", closeErr)
        }
    }()
    
    for event := range stream.Events() {
        // Process events
    }
    
    return nil
}
```

### 2. Handle Context Cancellation

```go
func contextAwareStreaming(provider core.Provider) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    stream, err := provider.StreamText(ctx, request)
    if err != nil {
        return err
    }
    defer stream.Close()
    
    for {
        select {
        case event, ok := <-stream.Events():
            if !ok {
                return nil // Stream closed
            }
            
            // Process event
            if err := processEvent(event); err != nil {
                return err
            }
            
        case <-ctx.Done():
            return ctx.Err() // Timeout or cancellation
        }
    }
}
```

### 3. Implement Backpressure

```go
func backpressureAwareStreaming(provider core.Provider) {
    stream, err := provider.StreamText(context.Background(), request)
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    // Buffered channel for processing
    processingQueue := make(chan core.Event, 10)
    
    // Background processor with rate limiting
    go func() {
        limiter := time.NewTicker(100 * time.Millisecond) // Max 10 events/sec
        defer limiter.Stop()
        
        for event := range processingQueue {
            <-limiter.C // Rate limit processing
            processEvent(event)
        }
    }()
    
    for event := range stream.Events() {
        select {
        case processingQueue <- event:
            // Queued successfully
        default:
            // Queue full, apply backpressure
            log.Printf("Processing queue full, slowing down...")
            time.Sleep(50 * time.Millisecond)
            
            // Try again
            processingQueue <- event
        }
        
        if event.Type == core.EventFinish {
            close(processingQueue)
            break
        }
    }
}
```

### 4. Monitor Stream Health

```go
type StreamMonitor struct {
    startTime     time.Time
    eventsCount   int64
    bytesReceived int64
    errors        int64
    lastEvent     time.Time
}

func (sm *StreamMonitor) trackEvent(event core.Event) {
    atomic.AddInt64(&sm.eventsCount, 1)
    sm.lastEvent = time.Now()
    
    switch event.Type {
    case core.EventTextDelta:
        atomic.AddInt64(&sm.bytesReceived, int64(len(event.TextDelta)))
    case core.EventError:
        atomic.AddInt64(&sm.errors, 1)
    }
}

func (sm *StreamMonitor) healthCheck() StreamHealth {
    eventsCount := atomic.LoadInt64(&sm.eventsCount)
    bytesReceived := atomic.LoadInt64(&sm.bytesReceived)
    errors := atomic.LoadInt64(&sm.errors)
    
    elapsed := time.Since(sm.startTime)
    timeSinceLastEvent := time.Since(sm.lastEvent)
    
    return StreamHealth{
        EventsPerSecond:    float64(eventsCount) / elapsed.Seconds(),
        BytesPerSecond:     float64(bytesReceived) / elapsed.Seconds(),
        ErrorRate:          float64(errors) / float64(eventsCount),
        TimeSinceLastEvent: timeSinceLastEvent,
        IsStalled:          timeSinceLastEvent > 30*time.Second,
    }
}

func monitoredStreaming(provider core.Provider) {
    monitor := &StreamMonitor{startTime: time.Now()}
    
    stream, err := provider.StreamText(context.Background(), request)
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    // Health check ticker
    healthTicker := time.NewTicker(5 * time.Second)
    defer healthTicker.Stop()
    
    for {
        select {
        case event, ok := <-stream.Events():
            if !ok {
                return
            }
            
            monitor.trackEvent(event)
            processEvent(event)
            
            if event.Type == core.EventFinish {
                return
            }
            
        case <-healthTicker.C:
            health := monitor.healthCheck()
            if health.IsStalled {
                log.Printf("⚠️  Stream appears stalled: %v since last event", health.TimeSinceLastEvent)
            } else {
                log.Printf("📊 Stream health: %.1f events/sec, %.1f KB/sec, %.1f%% errors",
                    health.EventsPerSecond, health.BytesPerSecond/1024, health.ErrorRate*100)
            }
        }
    }
}
```

## Advanced Features

### Custom Event Filters

```go
type EventFilter func(core.Event) bool

func WithEventFilters(stream core.TextStream, filters ...EventFilter) core.TextStream {
    return &FilteredStream{
        inner:   stream,
        filters: filters,
    }
}

type FilteredStream struct {
    inner   core.TextStream
    filters []EventFilter
    events  chan core.Event
    once    sync.Once
}

func (fs *FilteredStream) Events() <-chan core.Event {
    fs.once.Do(func() {
        fs.events = make(chan core.Event, 10)
        
        go func() {
            defer close(fs.events)
            
            for event := range fs.inner.Events() {
                // Apply filters
                include := true
                for _, filter := range fs.filters {
                    if !filter(event) {
                        include = false
                        break
                    }
                }
                
                if include {
                    fs.events <- event
                }
            }
        }()
    })
    
    return fs.events
}

// Usage
textOnlyFilter := func(event core.Event) bool {
    return event.Type == core.EventTextDelta || event.Type == core.EventFinish
}

errorFilter := func(event core.Event) bool {
    return event.Type != core.EventError
}

filteredStream := WithEventFilters(stream, textOnlyFilter, errorFilter)
```

### Stream Multiplexing

```go
func multiplexStreams(streams ...core.TextStream) <-chan core.Event {
    output := make(chan core.Event, 50)
    
    var wg sync.WaitGroup
    
    for i, stream := range streams {
        wg.Add(1)
        go func(id int, s core.TextStream) {
            defer wg.Done()
            defer s.Close()
            
            for event := range s.Events() {
                // Add stream identifier
                event.Metadata = map[string]any{
                    "stream_id": id,
                }
                
                select {
                case output <- event:
                case <-time.After(5 * time.Second):
                    log.Printf("Stream %d event dropped due to timeout", id)
                }
            }
        }(i, stream)
    }
    
    go func() {
        wg.Wait()
        close(output)
    }()
    
    return output
}
```

## Summary

GAI's streaming system provides:

1. **Real-Time Processing**: Event-driven architecture for immediate response
2. **Type Safety**: Structured events with compile-time checking
3. **Flow Control**: Natural backpressure through Go channels
4. **Error Resilience**: Comprehensive error handling and recovery
5. **Performance**: Efficient memory management and processing
6. **Flexibility**: Multiple streaming patterns and customization options

Key benefits:
- **User Experience**: Real-time feedback and interaction
- **Scalability**: Efficient resource utilization
- **Reliability**: Graceful error handling and recovery
- **Observability**: Built-in monitoring and health checks

Next steps:
- [Multi-Step Execution](./multi-step.md) - Complex workflow patterns
- [Tools](./tools.md) - Function calling with streaming
- [Providers](./providers.md) - Provider-specific streaming features