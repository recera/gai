# Ollama Provider Guide

This comprehensive guide covers everything you need to know about using Ollama with GAI for local AI inference, including model management, performance optimization, and advanced configuration for running large language models on your own hardware.

## Table of Contents
- [Overview](#overview)
- [Installation & Setup](#installation--setup)
- [Configuration](#configuration)
- [Supported Models](#supported-models)
- [Basic Usage](#basic-usage)
- [Streaming](#streaming)
- [Tool Calling](#tool-calling)
- [Structured Outputs](#structured-outputs)
- [Multimodal Support](#multimodal-support)
- [Model Management](#model-management)
- [Performance Optimization](#performance-optimization)
- [API Selection](#api-selection)
- [Advanced Configuration](#advanced-configuration)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- [Hardware Optimization](#hardware-optimization)
- [Troubleshooting](#troubleshooting)

## Overview

The Ollama provider enables seamless integration with locally hosted language models through [Ollama](https://ollama.ai), providing high-performance local AI capabilities with complete privacy and control over your data.

### Key Features
- ✅ **Local Operation**: Complete privacy with no data leaving your machine
- ✅ **No API Keys**: No authentication or billing required
- ✅ **Model Management**: List, pull, and manage models programmatically
- ✅ **Dual API Support**: Chat API and Generate API for different use cases
- ✅ **Hardware Optimization**: GPU acceleration with memory management
- ✅ **Tool Calling**: Function calling with compatible models
- ✅ **Streaming**: Real-time text streaming for responsive applications
- ✅ **Structured Outputs**: JSON generation with schema validation
- ✅ **Multimodal**: Support for text and images with vision models
- ✅ **High Performance**: Optimized for low latency local inference

### Ollama's Unique Strengths
- **Complete Privacy**: All processing happens locally
- **No Internet Required**: Works entirely offline once models are downloaded
- **Hardware Control**: Direct GPU/CPU utilization optimization
- **Model Variety**: Access to hundreds of open-source models
- **Cost Effective**: No per-token costs, only hardware investment
- **Customizable**: Custom prompts templates and model parameters
- **Fast Iteration**: Instant model switching and testing

## Installation & Setup

### 1. Install Ollama

First, install Ollama on your system:

```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.ai/install.sh | sh

# Windows
# Download installer from https://ollama.ai/download/windows
```

### 2. Start Ollama Service

```bash
# Start the Ollama service
ollama serve

# The service will run on http://localhost:11434 by default
```

### 3. Pull a Model

```bash
# Pull a popular model
ollama pull llama3.2

# Or pull a specific variant
ollama pull llama3.2:3b
```

### 4. Install the Ollama Provider

```bash
go get github.com/recera/gai/providers/ollama@latest
```

### 5. Verify Setup

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/recera/gai/core"
    "github.com/recera/gai/providers/ollama"
)

func main() {
    // Create provider (no API key required!)
    provider := ollama.New(
        ollama.WithModel("llama3.2"),
        ollama.WithBaseURL("http://localhost:11434"), // Default
    )
    
    // Test with a simple request
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Say 'Hello from Ollama!'"},
                },
            },
        },
        MaxTokens: 10,
    })
    
    if err != nil {
        log.Fatalf("Setup verification failed: %v", err)
    }
    
    fmt.Println("✅ Ollama provider is working!")
    fmt.Println("Response:", response.Text)
}
```

## Configuration

### Basic Configuration

```go
provider := ollama.New(
    ollama.WithBaseURL("http://localhost:11434"),    // Ollama server URL
    ollama.WithModel("llama3.2"),                    // Default model
    ollama.WithKeepAlive("10m"),                     // Model memory duration
    ollama.WithMaxRetries(3),                        // Retry attempts
    ollama.WithRetryDelay(100*time.Millisecond),     // Retry delay
)
```

### Advanced Configuration

```go
// Custom HTTP client for specific requirements
httpClient := &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
    Timeout: 300 * time.Second, // Longer timeout for large models
}

// Advanced configuration
provider := ollama.New(
    ollama.WithBaseURL("http://localhost:11434"),
    ollama.WithHTTPClient(httpClient),
    ollama.WithGenerateAPI(false),                   // Use Chat API (default)
    ollama.WithTemplate("{{.System}}\n{{.Prompt}}"), // Custom template
    ollama.WithKeepAlive("30m"),                     // Keep model loaded longer
    ollama.WithMetricsCollector(metricsCollector),
)
```

### Remote Ollama Configuration

```go
// Connect to remote Ollama instance
provider := ollama.New(
    ollama.WithBaseURL("http://192.168.1.100:11434"),
    ollama.WithModel("llama3.2:70b"), // Larger model on powerful remote machine
    ollama.WithMaxRetries(5),          // More retries for network issues
    ollama.WithRetryDelay(time.Second),
)
```

## Supported Models

### Llama Family

```go
// Llama 3.2 (Latest)
provider := ollama.New(ollama.WithModel("llama3.2"))        // 3B default
provider := ollama.New(ollama.WithModel("llama3.2:1b"))     // Ultra fast
provider := ollama.New(ollama.WithModel("llama3.2:3b"))     // Balanced

// Llama 3.1
provider := ollama.New(ollama.WithModel("llama3.1:8b"))     // Great performance
provider := ollama.New(ollama.WithModel("llama3.1:70b"))    // High capability
provider := ollama.New(ollama.WithModel("llama3.1:405b"))   // Maximum capability
```

### Other Popular Models

```go
// Mistral family
provider := ollama.New(ollama.WithModel("mistral"))         // General purpose
provider := ollama.New(ollama.WithModel("mistral-nemo"))    // Latest Mistral

// Code-specialized models
provider := ollama.New(ollama.WithModel("codellama"))       // Code generation
provider := ollama.New(ollama.WithModel("deepseek-coder")) // Advanced coding

// Specialized models
provider := ollama.New(ollama.WithModel("phi3"))           // Microsoft's efficient model
provider := ollama.New(ollama.WithModel("gemma2"))         // Google's Gemma
provider := ollama.New(ollama.WithModel("qwen2.5"))        // Alibaba's Qwen
```

### Model Comparison

| Model | Size | RAM Needed | Speed | Best For | Tool Calling |
|-------|------|-----------|--------|----------|--------------|
| llama3.2:1b | 1.3GB | 4GB | Fastest | Chat, simple tasks | ❌ |
| llama3.2:3b | 2GB | 6GB | Fast | General use | ❌ |
| llama3.1:8b | 4.7GB | 8GB | Medium | Complex tasks | ✅ |
| llama3.1:70b | 40GB | 64GB | Slower | Advanced reasoning | ✅ |
| codellama | 4GB | 8GB | Medium | Code generation | ❌ |
| mistral-nemo | 7GB | 12GB | Medium | Tool calling | ✅ |

## Basic Usage

### Simple Text Generation

```go
func generateText(provider *ollama.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are a helpful AI assistant. Be concise and accurate."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Explain the benefits of renewable energy sources."},
                },
            },
        },
        Temperature: 0.7,
        MaxTokens:   500,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Response:", response.Text)
    fmt.Printf("Tokens: Input=%d, Output=%d, Total=%d\n", 
        response.Usage.InputTokens,
        response.Usage.OutputTokens, 
        response.Usage.TotalTokens)
}
```

### Conversation with Context

```go
func conversationExample(provider *ollama.Provider) {
    messages := []core.Message{
        {
            Role: core.System,
            Parts: []core.Part{
                core.Text{Text: "You are a knowledgeable programming mentor. Provide practical advice."},
            },
        },
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "I'm learning Go. What's the difference between channels and goroutines?"},
            },
        },
        {
            Role: core.Assistant,
            Parts: []core.Part{
                core.Text{Text: "Goroutines are lightweight threads that run concurrently, while channels are the communication mechanism between goroutines. Think of goroutines as workers and channels as the pipes they use to send messages."},
            },
        },
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Can you show me a simple example?"},
            },
        },
    }
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages:    messages,
        Temperature: 0.5,
        MaxTokens:   800,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Assistant:", response.Text)
}
```

### Working with Different Models

```go
func modelComparison(provider *ollama.Provider) {
    models := []string{"llama3.2:1b", "llama3.2:3b", "llama3.1:8b"}
    prompt := "Explain quantum computing in simple terms."
    
    for _, model := range models {
        fmt.Printf("\n--- Using %s ---\n", model)
        
        start := time.Now()
        response, err := provider.GenerateText(context.Background(), core.Request{
            Model: model, // Override default model
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{core.Text{Text: prompt}},
                },
            },
            MaxTokens: 200,
        })
        
        duration := time.Since(start)
        
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }
        
        fmt.Printf("Response: %s\n", response.Text)
        fmt.Printf("Time: %v, Tokens: %d\n", duration, response.Usage.OutputTokens)
    }
}
```

## Streaming

### Basic Streaming

```go
func streamExample(provider *ollama.Provider) {
    ctx := context.Background()
    
    stream, err := provider.StreamText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a detailed explanation of machine learning concepts."},
                },
            },
        },
        Stream:    true,
        MaxTokens: 1500,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    fmt.Print("Streaming response: ")
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            fmt.Print(event.TextDelta)
        case core.EventError:
            log.Printf("Stream error: %v", event.Err)
        case core.EventFinish:
            fmt.Printf("\n\nStream complete! Usage: %+v\n", event.Usage)
        }
    }
}
```

### Advanced Streaming with Performance Monitoring

```go
func streamWithMonitoring(provider *ollama.Provider) {
    ctx := context.Background()
    
    stream, err := provider.StreamText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Create a comprehensive guide to Go concurrency."},
                },
            },
        },
        Stream: true,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    var (
        totalChars   int
        totalTokens  int
        startTime    = time.Now()
        firstToken   time.Time
        hasFirstToken bool
    )
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventStart:
            fmt.Println("Starting stream...")
            
        case core.EventTextDelta:
            if !hasFirstToken {
                firstToken = time.Now()
                hasFirstToken = true
            }
            
            fmt.Print(event.TextDelta)
            totalChars += len(event.TextDelta)
            
        case core.EventFinish:
            totalTokens = event.Usage.OutputTokens
            elapsed := time.Since(startTime)
            ttft := firstToken.Sub(startTime) // Time to first token
            
            fmt.Printf("\n\n--- Performance Metrics ---\n")
            fmt.Printf("Total time: %v\n", elapsed)
            fmt.Printf("Time to first token: %v\n", ttft)
            fmt.Printf("Characters: %d\n", totalChars)
            fmt.Printf("Tokens: %d\n", totalTokens)
            fmt.Printf("Chars/sec: %.1f\n", float64(totalChars)/elapsed.Seconds())
            fmt.Printf("Tokens/sec: %.1f\n", float64(totalTokens)/elapsed.Seconds())
        }
    }
}
```

## Tool Calling

### Creating Tools for Ollama

```go
// File system tool
type FileInput struct {
    Path      string `json:"path" description:"File path to read"`
    Operation string `json:"operation" description:"Operation: read, write, or list"`
    Content   string `json:"content,omitempty" description:"Content to write (for write operations)"`
}

type FileOutput struct {
    Content string `json:"content"`
    Success bool   `json:"success"`
    Error   string `json:"error,omitempty"`
}

func createFileSystemTool() tools.Handle {
    return tools.New[FileInput, FileOutput](
        "file_system",
        "Read, write, or list files and directories",
        func(ctx context.Context, input FileInput, meta tools.Meta) (FileOutput, error) {
            switch input.Operation {
            case "read":
                content, err := os.ReadFile(input.Path)
                if err != nil {
                    return FileOutput{Success: false, Error: err.Error()}, nil
                }
                return FileOutput{Content: string(content), Success: true}, nil
                
            case "list":
                entries, err := os.ReadDir(input.Path)
                if err != nil {
                    return FileOutput{Success: false, Error: err.Error()}, nil
                }
                
                var files []string
                for _, entry := range entries {
                    files = append(files, entry.Name())
                }
                return FileOutput{Content: strings.Join(files, "\n"), Success: true}, nil
                
            case "write":
                err := os.WriteFile(input.Path, []byte(input.Content), 0644)
                if err != nil {
                    return FileOutput{Success: false, Error: err.Error()}, nil
                }
                return FileOutput{Content: "File written successfully", Success: true}, nil
                
            default:
                return FileOutput{Success: false, Error: "Unknown operation"}, nil
            }
        },
    )
}

// System command tool
type CommandInput struct {
    Command string `json:"command" description:"Shell command to execute"`
}

type CommandOutput struct {
    Output   string `json:"output"`
    ExitCode int    `json:"exit_code"`
    Error    string `json:"error,omitempty"`
}

func createCommandTool() tools.Handle {
    return tools.New[CommandInput, CommandOutput](
        "shell_command",
        "Execute shell commands safely",
        func(ctx context.Context, input CommandInput, meta tools.Meta) (CommandOutput, error) {
            // Be careful with command execution in production!
            cmd := exec.CommandContext(ctx, "sh", "-c", input.Command)
            output, err := cmd.CombinedOutput()
            
            exitCode := 0
            if err != nil {
                if exitError, ok := err.(*exec.ExitError); ok {
                    exitCode = exitError.ExitCode()
                }
            }
            
            return CommandOutput{
                Output:   string(output),
                ExitCode: exitCode,
                Error:    "",
            }, nil
        },
    )
}
```

### Using Tools with Ollama

```go
func toolCallingExample(provider *ollama.Provider) {
    // Note: Only some models support tool calling (Llama 3.1+, Mistral Nemo, etc.)
    fileTool := createFileSystemTool()
    cmdTool := createCommandTool()
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "llama3.1:8b", // Ensure we use a tool-capable model
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "List the files in the current directory, then read the README.md file if it exists."},
                },
            },
        },
        Tools:     []tools.Handle{fileTool, cmdTool},
        ToolChoice: core.ToolAuto,
        MaxTokens:  1000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Response:", response.Text)
    
    // Show execution steps
    for i, step := range response.Steps {
        fmt.Printf("\nStep %d:\n", i+1)
        for _, call := range step.ToolCalls {
            fmt.Printf("  Called: %s\n", call.Name)
            fmt.Printf("  Input: %s\n", string(call.Input))
        }
        for _, result := range step.ToolResults {
            fmt.Printf("  Result: %s\n", string(result.Result))
        }
        if step.Text != "" {
            fmt.Printf("  Response: %s\n", step.Text)
        }
    }
}
```

### Streaming Tool Calls

```go
func streamingToolCalls(provider *ollama.Provider) {
    tools := []tools.Handle{
        createFileSystemTool(),
        createCommandTool(),
    }
    
    ctx := context.Background()
    stream, err := provider.StreamText(ctx, core.Request{
        Model: "llama3.1:8b",
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Check the system disk usage and create a report file with the findings."},
                },
            },
        },
        Tools:     tools,
        ToolChoice: core.ToolAuto,
        Stream:    true,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            fmt.Print(event.TextDelta)
            
        case core.EventToolCall:
            fmt.Printf("\n[Calling tool: %s]\n", event.ToolName)
            
        case core.EventToolResult:
            fmt.Printf("[Tool %s completed]\n", event.ToolName)
            
        case core.EventError:
            fmt.Printf("\nError: %v\n", event.Err)
        }
    }
}
```

## Structured Outputs

### Simple JSON Generation

```go
type TaskList struct {
    Title       string `json:"title"`
    Priority    string `json:"priority"`
    Tasks       []Task `json:"tasks"`
    DueDate     string `json:"due_date"`
    Estimated   int    `json:"estimated_hours"`
}

type Task struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Status      string `json:"status"`
    Assignee    string `json:"assignee"`
}

func structuredOutputExample(provider *ollama.Provider) {
    ctx := context.Background()
    
    result, err := provider.GenerateObject(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "Generate a structured project task list based on the user's request."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Create a task list for building a web application with user authentication."},
                },
            },
        },
        MaxTokens: 800,
    }, TaskList{})
    
    if err != nil {
        log.Fatal(err)
    }
    
    taskList := result.Value.(map[string]interface{})
    
    // Pretty print the result
    jsonBytes, _ := json.MarshalIndent(taskList, "", "  ")
    fmt.Println("Generated Task List:")
    fmt.Println(string(jsonBytes))
}
```

### Streaming Structured Output

```go
func streamingStructuredOutput(provider *ollama.Provider) {
    ctx := context.Background()
    
    stream, err := provider.StreamObject(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Create a detailed project plan for a mobile app development."},
                },
            },
        },
    }, TaskList{})
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    // Stream the JSON as it's being generated
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            fmt.Print(event.TextDelta)
        case core.EventFinish:
            fmt.Println("\n\nParsing final object...")
            
            // Get the final parsed object
            finalObj, err := stream.Final()
            if err != nil {
                fmt.Printf("Parse error: %v\n", err)
                return
            }
            
            fmt.Printf("Final structured result: %+v\n", *finalObj)
        }
    }
}
```

## Multimodal Support

### Image Analysis

```go
func imageAnalysis(provider *ollama.Provider) {
    // Use a vision-capable model
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "llava", // Vision-capable model
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "What do you see in this image? Describe it in detail."},
                    core.ImageURL{
                        URL:    "data:image/jpeg;base64,/9j/4AAQSkZJRgABA...", // Base64 encoded image
                        Detail: "high",
                    },
                },
            },
        },
        MaxTokens: 500,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Image Analysis:")
    fmt.Println(response.Text)
}
```

### Local Image Processing

```go
func processLocalImage(provider *ollama.Provider, imagePath string) {
    // Read and encode local image
    imageData, err := os.ReadFile(imagePath)
    if err != nil {
        log.Fatal(err)
    }
    
    base64Image := base64.StdEncoding.EncodeToString(imageData)
    dataURL := fmt.Sprintf("data:image/jpeg;base64,%s", base64Image)
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "llava:7b",
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Analyze this image and extract any text you see, then summarize the content."},
                    core.ImageURL{URL: dataURL, Detail: "high"},
                },
            },
        },
        MaxTokens: 1000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Local Image Analysis:")
    fmt.Println(response.Text)
}
```

## Model Management

### Listing Available Models

```go
func listModels(provider *ollama.Provider) {
    ctx := context.Background()
    
    models, err := provider.ListModels(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Available Models:")
    fmt.Println("================")
    for _, model := range models {
        fmt.Printf("Name: %s\n", model.Name)
        fmt.Printf("Size: %.1f GB\n", float64(model.Size)/(1024*1024*1024))
        fmt.Printf("Modified: %s\n", model.ModifiedAt.Format("2006-01-02 15:04:05"))
        fmt.Printf("Family: %s\n", model.Details.Family)
        fmt.Printf("Format: %s\n", model.Details.Format)
        fmt.Printf("Parameters: %s\n", model.Details.ParameterSize)
        fmt.Printf("Quantization: %s\n", model.Details.QuantizationLevel)
        fmt.Println("---")
    }
}
```

### Dynamic Model Management

```go
func dynamicModelManagement(provider *ollama.Provider) {
    ctx := context.Background()
    desiredModel := "llama3.2:3b"
    
    // Check if model is available
    available, err := provider.IsModelAvailable(ctx, desiredModel)
    if err != nil {
        log.Fatal(err)
    }
    
    if !available {
        fmt.Printf("Model %s not found locally. Pulling...\n", desiredModel)
        
        // Pull the model
        err = provider.PullModel(ctx, desiredModel)
        if err != nil {
            log.Fatalf("Failed to pull model: %v", err)
        }
        
        fmt.Printf("Successfully pulled %s\n", desiredModel)
    }
    
    // Use the model
    response, err := provider.GenerateText(ctx, core.Request{
        Model: desiredModel,
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Hello! What model are you?"},
                },
            },
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Response from %s: %s\n", desiredModel, response.Text)
}
```

### Model Warming and Preloading

```go
func warmupModels(provider *ollama.Provider) {
    models := []string{"llama3.2:1b", "llama3.2:3b", "codellama"}
    
    // Warm up multiple models by sending a simple request
    for _, model := range models {
        fmt.Printf("Warming up %s...\n", model)
        
        start := time.Now()
        _, err := provider.GenerateText(context.Background(), core.Request{
            Model: model,
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: "Hi"}, // Simple warmup message
                    },
                },
            },
            MaxTokens: 1,
        })
        
        if err != nil {
            fmt.Printf("Failed to warm up %s: %v\n", model, err)
            continue
        }
        
        loadTime := time.Since(start)
        fmt.Printf("Model %s warmed up in %v\n", model, loadTime)
    }
}
```

## Performance Optimization

### Memory Management

```go
func optimizeMemoryUsage(provider *ollama.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Explain machine learning concepts."},
                },
            },
        },
        ProviderOptions: map[string]interface{}{
            "ollama": map[string]interface{}{
                // Memory management options
                "num_ctx":    2048,  // Reduce context size for memory
                "num_gpu":    1,     // Use GPU layers
                "low_vram":   true,  // Enable low VRAM mode
                "num_thread": 8,     // CPU thread count
                
                // Performance options
                "repeat_penalty": 1.1,
                "top_k":         40,
                "top_p":         0.9,
                "temperature":   0.7,
            },
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Optimized Response:", response.Text)
}
```

### Concurrent Processing

```go
func processConcurrently(provider *ollama.Provider, inputs []string) {
    var wg sync.WaitGroup
    results := make(chan Result, len(inputs))
    
    // Semaphore to limit concurrent requests
    semaphore := make(chan struct{}, 3) // Max 3 concurrent requests
    
    for i, input := range inputs {
        wg.Add(1)
        go func(id int, text string) {
            defer wg.Done()
            
            // Acquire semaphore
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            start := time.Now()
            response, err := provider.GenerateText(context.Background(), core.Request{
                Messages: []core.Message{
                    {
                        Role: core.User,
                        Parts: []core.Part{
                            core.Text{Text: "Summarize: " + text},
                        },
                    },
                },
                MaxTokens: 200,
            })
            
            duration := time.Since(start)
            
            if err != nil {
                results <- Result{ID: id, Error: err, Duration: duration}
                return
            }
            
            results <- Result{
                ID:       id,
                Text:     response.Text,
                Usage:    response.Usage,
                Duration: duration,
            }
        }(i, input)
    }
    
    // Close results channel when all goroutines complete
    go func() {
        wg.Wait()
        close(results)
    }()
    
    // Collect results
    for result := range results {
        if result.Error != nil {
            fmt.Printf("Request %d failed: %v (took %v)\n", result.ID, result.Error, result.Duration)
        } else {
            fmt.Printf("Request %d completed in %v: %s\n", result.ID, result.Duration, result.Text)
        }
    }
}

type Result struct {
    ID       int
    Text     string
    Usage    core.Usage
    Error    error
    Duration time.Duration
}
```

### Model Switching Optimization

```go
func optimizeModelSwitching(provider *ollama.Provider) {
    // Use different models for different task types
    tasks := []struct {
        task  string
        model string
        text  string
    }{
        {"quick", "llama3.2:1b", "What's 2+2?"},
        {"coding", "codellama", "Write a Python function to reverse a string."},
        {"analysis", "llama3.1:8b", "Analyze the economic impact of renewable energy."},
        {"creative", "llama3.2:3b", "Write a short story about time travel."},
    }
    
    for _, task := range tasks {
        fmt.Printf("\n--- %s task with %s ---\n", task.task, task.model)
        
        start := time.Now()
        response, err := provider.GenerateText(context.Background(), core.Request{
            Model: task.model,
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: task.text},
                    },
                },
            },
            MaxTokens: 300,
        })
        
        duration := time.Since(start)
        
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }
        
        fmt.Printf("Response: %s\n", response.Text)
        fmt.Printf("Time: %v, Tokens: %d\n", duration, response.Usage.OutputTokens)
    }
}
```

## API Selection

### Chat API vs Generate API

```go
func demonstrateAPISelection() {
    // Chat API (default) - supports conversation history and tools
    chatProvider := ollama.New(
        ollama.WithModel("llama3.2"),
        ollama.WithGenerateAPI(false), // Use Chat API
    )
    
    // Generate API - simple text completion
    generateProvider := ollama.New(
        ollama.WithModel("llama3.2"),
        ollama.WithGenerateAPI(true), // Use Generate API
    )
    
    messages := []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Complete this sentence: The future of AI is"},
            },
        },
    }
    
    // Chat API response
    fmt.Println("--- Chat API ---")
    chatResponse, err := chatProvider.GenerateText(context.Background(), core.Request{
        Messages: messages,
        MaxTokens: 100,
    })
    if err == nil {
        fmt.Println("Response:", chatResponse.Text)
    }
    
    // Generate API response  
    fmt.Println("\n--- Generate API ---")
    genResponse, err := generateProvider.GenerateText(context.Background(), core.Request{
        Messages: messages,
        MaxTokens: 100,
    })
    if err == nil {
        fmt.Println("Response:", genResponse.Text)
    }
}
```

### Custom Template Usage

```go
func customTemplateExample() {
    // Custom template for specific formatting
    provider := ollama.New(
        ollama.WithModel("llama3.2"),
        ollama.WithTemplate(`<|system|>
{{.System}}
<|user|>
{{.Prompt}}
<|assistant|>
`),
    )
    
    response, err := provider.GenerateText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are a helpful coding assistant."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a simple HTTP server in Go."},
                },
            },
        },
        MaxTokens: 500,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Custom Template Response:")
    fmt.Println(response.Text)
}
```

## Advanced Configuration

### Provider-Specific Parameters

```go
func advancedParameterTuning(provider *ollama.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a creative story about space exploration."},
                },
            },
        },
        Temperature: 0.8, // High creativity
        MaxTokens:   1000,
        ProviderOptions: map[string]interface{}{
            "ollama": map[string]interface{}{
                // Sampling parameters
                "top_k":           40,    // Top-k sampling
                "top_p":           0.9,   // Nucleus sampling
                "repeat_penalty":  1.1,   // Reduce repetition
                "seed":           42,     // Reproducible output
                
                // Context and memory
                "num_ctx":        4096,   // Context window size
                "num_gpu":        -1,     // Use all available GPU layers
                "low_vram":       false,  // Disable low VRAM mode for better performance
                
                // Advanced sampling
                "frequency_penalty": 0.1,  // Penalize frequent tokens
                "presence_penalty":  0.1,  // Encourage new topics
                "mirostat":         2,     // Enable Mirostat sampling
                "mirostat_eta":     0.1,   // Mirostat learning rate
                "mirostat_tau":     5.0,   // Target entropy
                
                // Performance tuning
                "num_thread":       0,     // Auto-detect CPU threads
                "num_predict":      1000,  // Maximum tokens to generate
                
                // Stop sequences
                "stop": []string{"THE END", "CONCLUSION", "\n\n---"},
            },
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Advanced Tuned Response:")
    fmt.Println(response.Text)
}
```

### Hardware-Specific Optimization

```go
func hardwareOptimization() {
    // Configuration for different hardware setups
    
    // High-end GPU setup
    gpuProvider := ollama.New(
        ollama.WithModel("llama3.1:70b"),
        ollama.WithKeepAlive("60m"), // Keep large models loaded longer
    )
    
    // CPU-only setup
    cpuProvider := ollama.New(
        ollama.WithModel("llama3.2:3b"),
        ollama.WithKeepAlive("10m"),
    )
    
    // Low-memory setup
    lowMemProvider := ollama.New(
        ollama.WithModel("llama3.2:1b"),
        ollama.WithKeepAlive("5m"),
    )
    
    request := core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Explain quantum computing."},
                },
            },
        },
        MaxTokens: 300,
        ProviderOptions: map[string]interface{}{
            "ollama": map[string]interface{}{
                // GPU setup
                "num_gpu":    -1,    // Use all GPU layers
                "low_vram":   false, // Don't limit VRAM usage
                "num_thread": 0,     // Auto-detect CPU threads
                
                // OR for CPU-only setup
                // "num_gpu":    0,     // No GPU layers
                // "num_thread": 8,     // Specific CPU thread count
                
                // OR for low-memory setup  
                // "num_gpu":    4,     // Limited GPU layers
                // "low_vram":   true,  // Enable VRAM optimization
                // "num_ctx":    1024,  // Smaller context window
            },
        },
    }
    
    // Use appropriate provider based on hardware
    ctx := context.Background()
    response, err := gpuProvider.GenerateText(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Hardware-Optimized Response:")
    fmt.Println(response.Text)
}
```

## Error Handling

### Comprehensive Error Handling

```go
func handleOllamaErrors(provider *ollama.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "nonexistent-model", // This will cause an error
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Hello"},
                },
            },
        },
    })
    
    if err != nil {
        switch {
        case strings.Contains(err.Error(), "model not found"):
            fmt.Println("Model not found - pulling model...")
            if pullErr := provider.PullModel(ctx, "llama3.2"); pullErr != nil {
                fmt.Printf("Failed to pull model: %v\n", pullErr)
            } else {
                fmt.Println("Model pulled successfully")
                // Retry the request
            }
            
        case strings.Contains(err.Error(), "connection refused"):
            fmt.Println("Ollama server not running - please start with 'ollama serve'")
            
        case strings.Contains(err.Error(), "out of memory"):
            fmt.Println("Insufficient memory - try a smaller model or enable low_vram mode")
            
        case strings.Contains(err.Error(), "context length exceeded"):
            fmt.Println("Input too long - reduce message size or increase num_ctx")
            
        case strings.Contains(err.Error(), "timeout"):
            fmt.Println("Request timeout - increase timeout or use a smaller model")
            
        default:
            fmt.Printf("Unexpected error: %v\n", err)
        }
        return
    }
    
    fmt.Println("Success:", response.Text)
}
```

### Connection and Service Checks

```go
func checkOllamaHealth(provider *ollama.Provider) {
    ctx := context.Background()
    
    // Test basic connectivity
    models, err := provider.ListModels(ctx)
    if err != nil {
        if strings.Contains(err.Error(), "connection refused") {
            fmt.Println("❌ Ollama service is not running")
            fmt.Println("💡 Start Ollama with: ollama serve")
            return
        }
        fmt.Printf("❌ Connection error: %v\n", err)
        return
    }
    
    fmt.Printf("✅ Ollama service is running\n")
    fmt.Printf("📋 %d models available\n", len(models))
    
    // Test model availability
    testModel := "llama3.2"
    available, err := provider.IsModelAvailable(ctx, testModel)
    if err != nil {
        fmt.Printf("❌ Error checking model: %v\n", err)
        return
    }
    
    if !available {
        fmt.Printf("⚠️  Model %s not available\n", testModel)
        fmt.Printf("💡 Pull model with: ollama pull %s\n", testModel)
        return
    }
    
    fmt.Printf("✅ Model %s is available\n", testModel)
    
    // Test basic generation
    start := time.Now()
    _, err = provider.GenerateText(ctx, core.Request{
        Model: testModel,
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Hi"},
                },
            },
        },
        MaxTokens: 1,
    })
    
    duration := time.Since(start)
    
    if err != nil {
        fmt.Printf("❌ Generation test failed: %v\n", err)
        return
    }
    
    fmt.Printf("✅ Generation test passed (took %v)\n", duration)
}
```

## Best Practices

### 1. Model Selection Strategy

```go
func selectOptimalModel(taskType string, hardwareType string) string {
    // Define model selection matrix
    modelMatrix := map[string]map[string]string{
        "quick_questions": {
            "high_end_gpu": "llama3.2:3b",
            "mid_range_gpu": "llama3.2:1b",
            "cpu_only":     "llama3.2:1b",
            "low_memory":   "llama3.2:1b",
        },
        "complex_reasoning": {
            "high_end_gpu": "llama3.1:70b",
            "mid_range_gpu": "llama3.1:8b",
            "cpu_only":     "llama3.2:3b",
            "low_memory":   "llama3.2:3b",
        },
        "code_generation": {
            "high_end_gpu": "codellama:34b",
            "mid_range_gpu": "codellama:13b",
            "cpu_only":     "codellama:7b",
            "low_memory":   "codellama:7b",
        },
        "creative_writing": {
            "high_end_gpu": "llama3.1:70b",
            "mid_range_gpu": "llama3.2:3b",
            "cpu_only":     "llama3.2:3b",
            "low_memory":   "llama3.2:1b",
        },
    }
    
    if models, ok := modelMatrix[taskType]; ok {
        if model, ok := models[hardwareType]; ok {
            return model
        }
    }
    
    return "llama3.2:3b" // Safe default
}
```

### 2. Performance Monitoring

```go
func monitorPerformance(provider *ollama.Provider) {
    var totalRequests int
    var totalTime time.Duration
    var totalTokens int
    
    testCases := []string{
        "What is machine learning?",
        "Explain photosynthesis.",
        "Write a haiku about coding.",
        "Solve this math problem: 2x + 5 = 13",
        "Describe the water cycle.",
    }
    
    fmt.Println("Performance Test Results:")
    fmt.Println("========================")
    
    for i, test := range testCases {
        start := time.Now()
        response, err := provider.GenerateText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: test},
                    },
                },
            },
            MaxTokens: 100,
        })
        
        duration := time.Since(start)
        
        if err != nil {
            fmt.Printf("Test %d failed: %v\n", i+1, err)
            continue
        }
        
        totalRequests++
        totalTime += duration
        totalTokens += response.Usage.OutputTokens
        
        fmt.Printf("Test %d: %v (%.1f tokens/sec)\n", 
            i+1, duration, 
            float64(response.Usage.OutputTokens)/duration.Seconds())
    }
    
    if totalRequests > 0 {
        avgTime := totalTime / time.Duration(totalRequests)
        avgTokensPerSec := float64(totalTokens) / totalTime.Seconds()
        
        fmt.Printf("\nAverage Results:\n")
        fmt.Printf("- Time per request: %v\n", avgTime)
        fmt.Printf("- Tokens per second: %.1f\n", avgTokensPerSec)
        fmt.Printf("- Total tokens: %d\n", totalTokens)
    }
}
```

### 3. Resource Management

```go
func manageResources(provider *ollama.Provider) {
    // Set shorter keep-alive for resource-constrained environments
    resourceConstrainedProvider := ollama.New(
        ollama.WithModel("llama3.2:1b"),
        ollama.WithKeepAlive("2m"), // Unload model after 2 minutes
    )
    
    // For production environments with consistent load
    productionProvider := ollama.New(
        ollama.WithModel("llama3.1:8b"),
        ollama.WithKeepAlive("60m"), // Keep model loaded for 1 hour
    )
    
    // Dynamic keep-alive based on usage patterns
    lastUsed := time.Now()
    var provider_to_use *ollama.Provider
    
    // Check if we've been idle
    if time.Since(lastUsed) > 10*time.Minute {
        provider_to_use = resourceConstrainedProvider
    } else {
        provider_to_use = productionProvider
    }
    
    // Use the selected provider
    response, err := provider_to_use.GenerateText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Hello!"},
                },
            },
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Response:", response.Text)
    lastUsed = time.Now()
}
```

## Hardware Optimization

### GPU Acceleration

```go
func optimizeForGPU() {
    // Check available GPU memory and optimize accordingly
    provider := ollama.New(
        ollama.WithModel("llama3.1:8b"),
    )
    
    // Test different GPU layer configurations
    gpuConfigs := []map[string]interface{}{
        {"num_gpu": -1, "low_vram": false}, // Use all GPU layers
        {"num_gpu": 32, "low_vram": false}, // Partial GPU offloading  
        {"num_gpu": 16, "low_vram": true},  // Conservative GPU usage
    }
    
    for i, config := range gpuConfigs {
        fmt.Printf("\n--- GPU Config %d ---\n", i+1)
        
        start := time.Now()
        response, err := provider.GenerateText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: "Explain neural networks briefly."},
                    },
                },
            },
            MaxTokens: 150,
            ProviderOptions: map[string]interface{}{
                "ollama": config,
            },
        })
        
        duration := time.Since(start)
        
        if err != nil {
            fmt.Printf("Config failed: %v\n", err)
            continue
        }
        
        tokensPerSec := float64(response.Usage.OutputTokens) / duration.Seconds()
        fmt.Printf("Performance: %.1f tokens/sec, Time: %v\n", tokensPerSec, duration)
        fmt.Printf("GPU Layers: %v, Low VRAM: %v\n", 
            config["num_gpu"], config["low_vram"])
    }
}
```

### Memory Optimization

```go
func optimizeMemoryUsage() {
    // Different configurations for different memory constraints
    
    // 8GB RAM setup
    lightProvider := ollama.New(
        ollama.WithModel("llama3.2:1b"),
        ollama.WithKeepAlive("5m"),
    )
    
    // 16GB RAM setup
    mediumProvider := ollama.New(
        ollama.WithModel("llama3.2:3b"),
        ollama.WithKeepAlive("15m"),
    )
    
    // 32GB+ RAM setup
    heavyProvider := ollama.New(
        ollama.WithModel("llama3.1:8b"),
        ollama.WithKeepAlive("60m"),
    )
    
    // Memory-optimized request options
    memoryOptimizedOptions := map[string]interface{}{
        "ollama": map[string]interface{}{
            "num_ctx":    2048, // Smaller context window
            "num_batch":  512,  // Smaller batch size
            "low_vram":   true, // Enable VRAM optimization
            "num_gpu":    8,    // Limited GPU layers
        },
    }
    
    // Test with different setups
    providers := []*ollama.Provider{lightProvider, mediumProvider, heavyProvider}
    names := []string{"Light (1B)", "Medium (3B)", "Heavy (8B)"}
    
    for i, provider := range providers {
        fmt.Printf("\n--- Testing %s ---\n", names[i])
        
        start := time.Now()
        response, err := provider.GenerateText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: "Explain renewable energy benefits."},
                    },
                },
            },
            MaxTokens: 200,
            ProviderOptions: memoryOptimizedOptions,
        })
        
        if err != nil {
            fmt.Printf("Failed: %v\n", err)
            continue
        }
        
        duration := time.Since(start)
        fmt.Printf("Time: %v, Quality: %d chars\n", duration, len(response.Text))
    }
}
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Ollama Service Not Running

**Problem**: `connection refused` errors

**Solution**:
```bash
# Check if Ollama is running
ps aux | grep ollama

# Start Ollama service
ollama serve

# Or as background service on Linux
systemctl start ollama

# On macOS with homebrew
brew services start ollama
```

#### 2. Model Not Found

**Problem**: `model not found` errors

**Solution**:
```go
func handleModelNotFound(provider *ollama.Provider, modelName string) {
    ctx := context.Background()
    
    // Check available models
    models, err := provider.ListModels(ctx)
    if err != nil {
        fmt.Printf("Cannot list models: %v\n", err)
        return
    }
    
    fmt.Println("Available models:")
    for _, model := range models {
        fmt.Printf("- %s\n", model.Name)
    }
    
    // Pull the model if needed
    fmt.Printf("Pulling %s...\n", modelName)
    err = provider.PullModel(ctx, modelName)
    if err != nil {
        fmt.Printf("Failed to pull model: %v\n", err)
        return
    }
    
    fmt.Printf("Successfully pulled %s\n", modelName)
}
```

#### 3. Out of Memory Errors

**Problem**: System runs out of memory when loading large models

**Solution**:
```go
func handleMemoryIssues() {
    // Try progressively smaller models
    fallbackModels := []string{
        "llama3.1:8b",   // Try medium model first
        "llama3.2:3b",   // Fall back to smaller
        "llama3.2:1b",   // Last resort
    }
    
    for _, model := range fallbackModels {
        provider := ollama.New(
            ollama.WithModel(model),
            ollama.WithKeepAlive("2m"), // Shorter keep-alive
        )
        
        fmt.Printf("Trying model: %s\n", model)
        
        response, err := provider.GenerateText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: "Hello!"},
                    },
                },
            },
            MaxTokens: 50,
            ProviderOptions: map[string]interface{}{
                "ollama": map[string]interface{}{
                    "low_vram":  true,
                    "num_gpu":   4,    // Limited GPU layers
                    "num_ctx":   1024, // Smaller context
                },
            },
        })
        
        if err == nil {
            fmt.Printf("Success with %s: %s\n", model, response.Text)
            return
        }
        
        fmt.Printf("Failed with %s: %v\n", model, err)
    }
    
    fmt.Println("All models failed - insufficient memory")
}
```

#### 4. Slow Performance

**Problem**: Very slow response times

**Solution**:
```go
func diagnosePerfomance(provider *ollama.Provider) {
    fmt.Println("Performance Diagnostics:")
    fmt.Println("======================")
    
    // Test basic connectivity
    start := time.Now()
    models, err := provider.ListModels(context.Background())
    apiLatency := time.Since(start)
    
    fmt.Printf("API Latency: %v\n", apiLatency)
    
    if err != nil {
        fmt.Printf("API Error: %v\n", err)
        return
    }
    
    // Test different configurations
    configs := []struct {
        name string
        opts map[string]interface{}
    }{
        {
            "GPU Optimized",
            map[string]interface{}{
                "num_gpu":   -1,
                "low_vram":  false,
                "num_ctx":   4096,
            },
        },
        {
            "Balanced",
            map[string]interface{}{
                "num_gpu":   16,
                "low_vram":  false,
                "num_ctx":   2048,
            },
        },
        {
            "Memory Conserved",
            map[string]interface{}{
                "num_gpu":   4,
                "low_vram":  true,
                "num_ctx":   1024,
            },
        },
    }
    
    for _, config := range configs {
        fmt.Printf("\n--- %s ---\n", config.name)
        
        start := time.Now()
        response, err := provider.GenerateText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: "Count to 10."},
                    },
                },
            },
            MaxTokens: 50,
            ProviderOptions: map[string]interface{}{
                "ollama": config.opts,
            },
        })
        
        duration := time.Since(start)
        
        if err != nil {
            fmt.Printf("Failed: %v\n", err)
            continue
        }
        
        tokensPerSec := float64(response.Usage.OutputTokens) / duration.Seconds()
        fmt.Printf("Duration: %v\n", duration)
        fmt.Printf("Tokens/sec: %.1f\n", tokensPerSec)
        fmt.Printf("Response: %s\n", response.Text)
    }
}
```

#### 5. Model Loading Issues

**Problem**: Models fail to load or take very long to load

**Solution**:
```bash
# Check Ollama logs
ollama logs

# Check disk space
df -h

# Check model file integrity
ollama show llama3.2

# Re-pull corrupted models
ollama rm llama3.2
ollama pull llama3.2

# Check for file permissions issues
ls -la ~/.ollama/models/
```

## Summary

The Ollama provider in GAI offers:
- **Complete Privacy**: All processing happens locally with no data leaving your machine
- **No API Costs**: No per-token charges, only hardware investment required
- **Model Flexibility**: Easy access to hundreds of open-source models
- **Hardware Control**: Direct optimization for your specific hardware setup
- **High Performance**: Optimized local inference with GPU acceleration
- **Model Management**: Built-in tools for pulling, listing, and managing models

Key advantages over cloud providers:
- **Privacy**: Complete data privacy and control
- **Cost**: No ongoing API costs after initial hardware investment
- **Latency**: Potentially faster responses with optimized local setup
- **Offline**: Works completely offline once models are downloaded
- **Customization**: Direct control over model parameters and optimization

Best practices:
- Choose models appropriate for your hardware capabilities
- Use GPU acceleration when available
- Optimize keep-alive settings based on usage patterns
- Monitor performance and adjust configurations accordingly
- Use streaming for better user experience
- Implement proper error handling for model availability

Next steps:
- Explore [Local Model Management](../guides/local-models.md)
- Learn about [Hardware Optimization](../guides/hardware-optimization.md)
- Try [Tool Calling](../features/tool-calling.md) with compatible models
- Review [Performance Tuning](../guides/performance-tuning.md) strategies