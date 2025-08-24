# OpenAI Provider Guide

This comprehensive guide covers everything you need to know about using OpenAI models with GAI, including GPT-4, GPT-3.5, DALL-E, and advanced features like function calling, structured outputs, and vision capabilities.

## Table of Contents
- [Overview](#overview)
- [Installation & Setup](#installation--setup)
- [Configuration](#configuration)
- [Supported Models](#supported-models)
- [Basic Usage](#basic-usage)
- [Streaming](#streaming)
- [Function Calling](#function-calling)
- [Structured Outputs](#structured-outputs)
- [Vision (GPT-4V)](#vision-gpt-4v)
- [Advanced Features](#advanced-features)
- [Error Handling](#error-handling)
- [Cost Optimization](#cost-optimization)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

The OpenAI provider gives you access to state-of-the-art language models including:
- **GPT-4 Turbo**: Most capable model with 128k context window
- **GPT-4**: Original GPT-4 with strong reasoning
- **GPT-3.5 Turbo**: Fast and cost-effective
- **DALL-E**: Image generation (coming soon)
- **Whisper**: Speech-to-text (via media package)

### Key Features
- ✅ Text generation and chat
- ✅ Streaming responses
- ✅ Function calling (tools)
- ✅ Structured outputs with JSON mode
- ✅ Vision capabilities (GPT-4V)
- ✅ Embeddings
- ✅ Fine-tuned model support
- ✅ Azure OpenAI support

## Installation & Setup

### 1. Install the OpenAI Provider

```bash
go get github.com/yourusername/gai/providers/openai@latest
```

### 2. Obtain an API Key

1. Sign up at [platform.openai.com](https://platform.openai.com)
2. Navigate to [API Keys](https://platform.openai.com/api-keys)
3. Click "Create new secret key"
4. Copy the key (starts with `sk-`)
5. Store it securely

### 3. Set Up Environment

```bash
# Set your API key as an environment variable
export OPENAI_API_KEY="sk-...your-key-here..."

# Optional: Set organization ID
export OPENAI_ORG_ID="org-...your-org-id..."

# Optional: Set default model
export OPENAI_MODEL="gpt-4-turbo-preview"
```

### 4. Verify Setup

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "github.com/yourusername/gai/core"
    "github.com/yourusername/gai/providers/openai"
)

func main() {
    // Create provider
    provider := openai.New(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )
    
    // Test with a simple request
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Say 'Hello from OpenAI!'"},
                },
            },
        },
        MaxTokens: 10,
    })
    
    if err != nil {
        log.Fatalf("Setup verification failed: %v", err)
    }
    
    fmt.Println("✅ OpenAI provider is working!")
    fmt.Println("Response:", response.Text)
}
```

## Configuration

### Basic Configuration

```go
provider := openai.New(
    openai.WithAPIKey("sk-..."),           // Required: Your API key
    openai.WithModel("gpt-4-turbo"),       // Default model to use
    openai.WithOrganization("org-..."),    // Optional: Organization ID
    openai.WithBaseURL("https://..."),     // Optional: Custom endpoint (e.g., Azure)
    openai.WithTimeout(60 * time.Second),  // Request timeout
    openai.WithMaxRetries(3),              // Retry attempts
)
```

### Advanced Configuration

```go
// Custom HTTP client for proxy or special requirements
httpClient := &http.Client{
    Transport: &http.Transport{
        Proxy: http.ProxyFromEnvironment,
        MaxIdleConns: 100,
        MaxIdleConnsPerHost: 10,
    },
    Timeout: 120 * time.Second,
}

provider := openai.New(
    openai.WithAPIKey(apiKey),
    openai.WithHTTPClient(httpClient),
    openai.WithMaxRetries(5),
    openai.WithRetryDelay(time.Second),
    openai.WithMetricsCollector(metricsCollector),
)
```

### Azure OpenAI Configuration

```go
// For Azure OpenAI Service
provider := openai.New(
    openai.WithAPIKey(azureKey),
    openai.WithBaseURL("https://your-resource.openai.azure.com"),
    openai.WithAPIVersion("2024-02-15-preview"),
    openai.WithAzureDeployment("your-deployment-name"),
)
```

## Supported Models

### GPT-4 Family

```go
// GPT-4 Turbo (Recommended - Latest)
provider := openai.New(
    openai.WithModel("gpt-4-turbo"),         // Latest GPT-4 Turbo
    // OR
    openai.WithModel("gpt-4-turbo-2024-04-09"), // Specific version
)

// GPT-4 (Original)
provider := openai.New(
    openai.WithModel("gpt-4"),               // Original GPT-4
    // OR
    openai.WithModel("gpt-4-0613"),          // Specific version
)

// GPT-4 32K Context
provider := openai.New(
    openai.WithModel("gpt-4-32k"),           // 32K context window
)
```

### GPT-3.5 Family

```go
// GPT-3.5 Turbo (Fast & Affordable)
provider := openai.New(
    openai.WithModel("gpt-3.5-turbo"),       // Latest 3.5
    // OR
    openai.WithModel("gpt-3.5-turbo-0125"),  // Latest with improved instruction following
)

// GPT-3.5 16K Context
provider := openai.New(
    openai.WithModel("gpt-3.5-turbo-16k"),   // 16K context window
)
```

### Vision Models

```go
// GPT-4 Vision
provider := openai.New(
    openai.WithModel("gpt-4-vision-preview"), // GPT-4 with vision
    // OR
    openai.WithModel("gpt-4-turbo"),         // Latest turbo includes vision
)
```

### Model Comparison

| Model | Context Window | Strengths | Best For | Relative Cost |
|-------|---------------|-----------|----------|---------------|
| gpt-4-turbo | 128K | Latest capabilities, fast | Production apps | $$$$ |
| gpt-4 | 8K | Strong reasoning | Complex tasks | $$$$ |
| gpt-4-32k | 32K | Long context | Document analysis | $$$$$ |
| gpt-3.5-turbo | 16K | Fast, affordable | Most use cases | $ |
| gpt-3.5-turbo-16k | 16K | Extended context | Longer conversations | $ |

## Basic Usage

### Simple Text Generation

```go
func generateText(provider *openai.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "gpt-4-turbo", // Optional: Override default
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are a helpful assistant."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Explain quantum computing in simple terms."},
                },
            },
        },
        Temperature: 0.7,    // Creativity level (0.0 - 2.0)
        MaxTokens:   500,    // Maximum response length
        TopP:        0.9,    // Nucleus sampling
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
func conversationExample(provider *openai.Provider) {
    messages := []core.Message{
        {
            Role: core.System,
            Parts: []core.Part{
                core.Text{Text: "You are a knowledgeable history teacher."},
            },
        },
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Tell me about the Renaissance."},
            },
        },
        {
            Role: core.Assistant,
            Parts: []core.Part{
                core.Text{Text: "The Renaissance was a period of cultural rebirth in Europe from the 14th to 17th century..."},
            },
        },
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "What were the key inventions?"},
            },
        },
    }
    
    response, err := provider.GenerateText(context.Background(), core.Request{
        Messages: messages,
        Temperature: 0.5, // More focused responses
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Assistant:", response.Text)
}
```

## Streaming

### Basic Streaming

```go
func streamExample(provider *openai.Provider) {
    ctx := context.Background()
    
    stream, err := provider.StreamText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a story about a robot learning to paint."},
                },
            },
        },
        Stream: true,
        MaxTokens: 1000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    fmt.Print("Story: ")
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            fmt.Print(event.TextDelta)
        case core.EventError:
            log.Printf("Stream error: %v", event.Err)
        case core.EventFinish:
            fmt.Println("\n\n[Stream complete]")
        }
    }
}
```

### Advanced Streaming with Progress

```go
func streamWithProgress(provider *openai.Provider) {
    ctx := context.Background()
    
    stream, err := provider.StreamText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "List 10 interesting facts about space."},
                },
            },
        },
        Stream: true,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    var totalChars int
    startTime := time.Now()
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventStart:
            fmt.Println("Starting stream...")
            
        case core.EventTextDelta:
            fmt.Print(event.TextDelta)
            totalChars += len(event.TextDelta)
            
        case core.EventFinish:
            elapsed := time.Since(startTime)
            fmt.Printf("\n\nStats:\n")
            fmt.Printf("- Characters: %d\n", totalChars)
            fmt.Printf("- Time: %v\n", elapsed)
            fmt.Printf("- Speed: %.1f chars/sec\n", float64(totalChars)/elapsed.Seconds())
        }
    }
}
```

## Function Calling

### Defining Tools

```go
// Weather tool
type WeatherInput struct {
    Location string `json:"location" jsonschema:"required,description=City and country"`
    Unit     string `json:"unit,omitempty" jsonschema:"enum=celsius,enum=fahrenheit,default=celsius"`
}

type WeatherOutput struct {
    Temperature float64 `json:"temperature"`
    Condition   string  `json:"condition"`
    Humidity    int     `json:"humidity"`
    WindSpeed   float64 `json:"wind_speed"`
}

func createWeatherTool() tools.Handle {
    return tools.New[WeatherInput, WeatherOutput](
        "get_weather",
        "Get current weather for a location",
        func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
            // In production, call a real weather API
            return WeatherOutput{
                Temperature: 22.5,
                Condition:   "Partly cloudy",
                Humidity:    65,
                WindSpeed:   15.2,
            }, nil
        },
    )
}

// Calculator tool
type CalcInput struct {
    Expression string `json:"expression" jsonschema:"required,description=Mathematical expression to evaluate"`
}

type CalcOutput struct {
    Result float64 `json:"result"`
    Steps  string  `json:"steps,omitempty"`
}

func createCalculatorTool() tools.Handle {
    return tools.New[CalcInput, CalcOutput](
        "calculator",
        "Perform mathematical calculations",
        func(ctx context.Context, input CalcInput, meta tools.Meta) (CalcOutput, error) {
            // Evaluate expression (use a safe math parser in production)
            result := evaluateExpression(input.Expression)
            return CalcOutput{
                Result: result,
                Steps:  "Evaluated: " + input.Expression,
            }, nil
        },
    )
}
```

### Using Tools

```go
func toolCallingExample(provider *openai.Provider) {
    weatherTool := createWeatherTool()
    calcTool := createCalculatorTool()
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "What's the weather in Paris? Also, what's 234 * 567?"},
                },
            },
        },
        Tools: []tools.Handle{weatherTool, calcTool},
        ToolChoice: core.ToolAuto, // Let model decide which tools to use
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
        }
        if step.Text != "" {
            fmt.Printf("  Output: %s\n", step.Text)
        }
    }
}
```

### Parallel Tool Execution

```go
func parallelToolsExample(provider *openai.Provider) {
    // Create multiple tools
    tools := []tools.Handle{
        createWeatherTool(),
        createNewsSearchTool(),
        createStockPriceTool(),
        createCalculatorTool(),
    }
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Give me a morning briefing: weather in NYC, top news, AAPL stock price, and calculate my portfolio value (100 AAPL shares)."},
                },
            },
        },
        Tools: tools,
        ToolChoice: core.ToolAuto,
        // OpenAI can call multiple tools in parallel
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Morning Briefing:")
    fmt.Println(response.Text)
}
```

## Structured Outputs

### JSON Mode

```go
type ProductReview struct {
    ProductName string   `json:"product_name"`
    Rating      int      `json:"rating"`
    Pros        []string `json:"pros"`
    Cons        []string `json:"cons"`
    Summary     string   `json:"summary"`
    Recommended bool     `json:"recommended"`
}

func structuredOutputExample(provider *openai.Provider) {
    ctx := context.Background()
    
    reviewText := `
    The new iPhone 15 Pro is impressive. The titanium build feels premium,
    the camera is exceptional especially in low light, and the A17 Pro chip
    is blazingly fast. However, the price is steep at $999+, battery life
    could be better, and the changes from iPhone 14 Pro are incremental.
    Overall, it's excellent but maybe wait for a sale unless you need the
    latest tech.
    `
    
    result, err := provider.GenerateObject[ProductReview](ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "Extract product review information into structured JSON."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: reviewText},
                },
            },
        },
        // OpenAI will use JSON mode automatically
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    review := result.Value
    fmt.Printf("Product: %s\n", review.ProductName)
    fmt.Printf("Rating: %d/5\n", review.Rating)
    fmt.Printf("Pros: %v\n", review.Pros)
    fmt.Printf("Cons: %v\n", review.Cons)
    fmt.Printf("Recommended: %v\n", review.Recommended)
}
```

### Complex Structured Output

```go
type CompanyAnalysis struct {
    Company     string      `json:"company"`
    Industry    string      `json:"industry"`
    Financials  Financials  `json:"financials"`
    Competitors []string    `json:"competitors"`
    SWOT        SWOTAnalysis `json:"swot"`
    Outlook     string      `json:"outlook"`
}

type Financials struct {
    Revenue     string `json:"revenue"`
    Profit      string `json:"profit"`
    GrowthRate  string `json:"growth_rate"`
    MarketCap   string `json:"market_cap"`
}

type SWOTAnalysis struct {
    Strengths     []string `json:"strengths"`
    Weaknesses    []string `json:"weaknesses"`
    Opportunities []string `json:"opportunities"`
    Threats       []string `json:"threats"`
}

func complexStructuredOutput(provider *openai.Provider) {
    ctx := context.Background()
    
    result, err := provider.GenerateObject[CompanyAnalysis](ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Analyze Apple Inc. and provide a comprehensive business analysis."},
                },
            },
        },
        Temperature: 0.3, // Lower temperature for more consistent structure
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    analysis := result.Value
    
    // Pretty print the analysis
    jsonBytes, _ := json.MarshalIndent(analysis, "", "  ")
    fmt.Println(string(jsonBytes))
}
```

## Vision (GPT-4V)

### Image Analysis

```go
func imageAnalysisExample(provider *openai.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "gpt-4-vision-preview", // or gpt-4-turbo
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "What's in this image? Describe in detail."},
                    core.ImageURL{
                        URL: "https://example.com/architecture-diagram.png",
                        Detail: "high", // Request high detail analysis
                    },
                },
            },
        },
        MaxTokens: 1000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Image Analysis:")
    fmt.Println(response.Text)
}
```

### Multiple Image Comparison

```go
func compareImagesExample(provider *openai.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "gpt-4-turbo",
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Compare these two UI designs:"},
                    core.Text{Text: "Design A:"},
                    core.ImageURL{URL: "https://example.com/design-a.png", Detail: "high"},
                    core.Text{Text: "Design B:"},
                    core.ImageURL{URL: "https://example.com/design-b.png", Detail: "high"},
                    core.Text{Text: "Which has better UX and why?"},
                },
            },
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Design Comparison:")
    fmt.Println(response.Text)
}
```

### Local Image Analysis

```go
func analyzeLocalImage(provider *openai.Provider, imagePath string) {
    // Read and encode image
    imageData, err := os.ReadFile(imagePath)
    if err != nil {
        log.Fatal(err)
    }
    
    base64Image := base64.StdEncoding.EncodeToString(imageData)
    dataURL := fmt.Sprintf("data:image/jpeg;base64,%s", base64Image)
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "gpt-4-vision-preview",
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Analyze this chart and extract the key data points:"},
                    core.ImageURL{URL: dataURL, Detail: "high"},
                },
            },
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Chart Analysis:")
    fmt.Println(response.Text)
}
```

## Advanced Features

### System Fingerprint

```go
// Track model versions for reproducibility
func trackSystemFingerprint(provider *openai.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
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
        log.Fatal(err)
    }
    
    // Access raw response for OpenAI-specific fields
    if raw, ok := response.Raw.(*openai.ChatCompletionResponse); ok {
        fmt.Printf("System Fingerprint: %s\n", raw.SystemFingerprint)
        fmt.Printf("Model: %s\n", raw.Model)
        fmt.Printf("Created: %d\n", raw.Created)
    }
}
```

### Logprobs

```go
func logprobsExample(provider *openai.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Complete: The capital of France is"},
                },
            },
        },
        ProviderOptions: map[string]any{
            "logprobs": true,
            "top_logprobs": 3,
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Response:", response.Text)
    
    // Access logprobs from raw response
    if raw, ok := response.Raw.(*openai.ChatCompletionResponse); ok {
        for _, choice := range raw.Choices {
            if choice.Logprobs != nil {
                fmt.Println("Top token probabilities:")
                for _, logprob := range choice.Logprobs.Content {
                    fmt.Printf("  Token: %s, Logprob: %f\n", 
                        logprob.Token, logprob.Logprob)
                }
            }
        }
    }
}
```

### Seed for Reproducibility

```go
func reproducibleGeneration(provider *openai.Provider) {
    ctx := context.Background()
    seed := 12345
    
    // First generation
    response1, _ := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a haiku about coding"},
                },
            },
        },
        ProviderOptions: map[string]any{
            "seed": seed,
        },
        Temperature: 0.7,
    })
    
    // Second generation with same seed
    response2, _ := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a haiku about coding"},
                },
            },
        },
        ProviderOptions: map[string]any{
            "seed": seed,
        },
        Temperature: 0.7,
    })
    
    fmt.Println("First:", response1.Text)
    fmt.Println("Second:", response2.Text)
    fmt.Println("Identical:", response1.Text == response2.Text)
}
```

## Error Handling

### Common Error Types

```go
func handleOpenAIErrors(provider *openai.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Hello"},
                },
            },
        },
        MaxTokens: 1000000, // Intentionally too high
    })
    
    if err != nil {
        switch {
        case core.IsRateLimited(err):
            // Handle rate limiting
            fmt.Println("Rate limited. Waiting before retry...")
            time.Sleep(core.GetRetryAfter(err))
            // Retry request
            
        case core.IsContextLengthExceeded(err):
            // Handle context length errors
            fmt.Println("Context too long. Trimming messages...")
            // Trim messages and retry
            
        case core.IsInvalidRequest(err):
            // Handle validation errors
            fmt.Println("Invalid request:", err)
            // Fix request parameters
            
        case core.IsUnauthorized(err):
            // Handle auth errors
            fmt.Println("Authentication failed. Check API key.")
            
        case core.IsProviderUnavailable(err):
            // Handle OpenAI service issues
            fmt.Println("OpenAI service unavailable. Trying fallback...")
            // Use fallback provider
            
        default:
            // Unknown error
            fmt.Printf("Unexpected error: %v\n", err)
        }
        return
    }
    
    fmt.Println("Success:", response.Text)
}
```

### Retry Strategy

```go
func robustRequest(provider *openai.Provider) {
    ctx := context.Background()
    maxRetries := 3
    
    var lastErr error
    for attempt := 0; attempt < maxRetries; attempt++ {
        response, err := provider.GenerateText(ctx, core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: "Generate a summary"},
                    },
                },
            },
        })
        
        if err == nil {
            fmt.Println("Success:", response.Text)
            return
        }
        
        lastErr = err
        
        // Determine if we should retry
        if !core.IsTransient(err) {
            fmt.Printf("Non-retryable error: %v\n", err)
            return
        }
        
        // Calculate backoff
        backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
        if core.IsRateLimited(err) {
            backoff = core.GetRetryAfter(err)
        }
        
        fmt.Printf("Attempt %d failed, retrying in %v...\n", attempt+1, backoff)
        time.Sleep(backoff)
    }
    
    fmt.Printf("All retries exhausted: %v\n", lastErr)
}
```

## Cost Optimization

### Token Management

```go
func optimizeTokenUsage(provider *openai.Provider) {
    // Use shorter model for simple tasks
    simpleProvider := openai.New(
        openai.WithModel("gpt-3.5-turbo"),
    )
    
    // Use powerful model for complex tasks
    complexProvider := openai.New(
        openai.WithModel("gpt-4-turbo"),
    )
    
    // Route based on complexity
    func processRequest(text string) {
        if isSimpleQuery(text) {
            // Use cheaper model
            response, _ := simpleProvider.GenerateText(...)
        } else {
            // Use more capable model
            response, _ := complexProvider.GenerateText(...)
        }
    }
}
```

### Context Window Management

```go
func manageContextWindow(messages []core.Message, maxTokens int) []core.Message {
    // Estimate tokens (rough: 1 token ≈ 4 characters)
    totalChars := 0
    for _, msg := range messages {
        for _, part := range msg.Parts {
            if text, ok := part.(core.Text); ok {
                totalChars += len(text.Text)
            }
        }
    }
    
    estimatedTokens := totalChars / 4
    
    if estimatedTokens > maxTokens {
        // Keep system message and recent messages
        systemMsg := messages[0]
        
        // Calculate how many recent messages we can keep
        recentMessages := []core.Message{}
        recentChars := 0
        maxChars := maxTokens * 4
        
        for i := len(messages) - 1; i > 0; i-- {
            msgChars := getMessageChars(messages[i])
            if recentChars + msgChars > maxChars {
                break
            }
            recentMessages = append([]core.Message{messages[i]}, recentMessages...)
            recentChars += msgChars
        }
        
        return append([]core.Message{systemMsg}, recentMessages...)
    }
    
    return messages
}
```

### Caching Strategies

```go
type CachedProvider struct {
    provider *openai.Provider
    cache    map[string]*core.TextResult
    mu       sync.RWMutex
}

func (c *CachedProvider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
    // Create cache key from request
    key := createCacheKey(req)
    
    // Check cache
    c.mu.RLock()
    if cached, ok := c.cache[key]; ok {
        c.mu.RUnlock()
        return cached, nil
    }
    c.mu.RUnlock()
    
    // Generate new response
    response, err := c.provider.GenerateText(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // Cache successful responses
    c.mu.Lock()
    c.cache[key] = response
    c.mu.Unlock()
    
    return response, nil
}
```

## Best Practices

### 1. Model Selection

```go
// Choose the right model for the task
func selectModel(taskType string) string {
    switch taskType {
    case "simple_qa":
        return "gpt-3.5-turbo"      // Fast and cheap
    case "code_generation":
        return "gpt-4-turbo"         // Better at code
    case "complex_reasoning":
        return "gpt-4"               // Strongest reasoning
    case "vision":
        return "gpt-4-vision-preview" // Vision capabilities
    case "long_context":
        return "gpt-4-turbo"         // 128k context
    default:
        return "gpt-3.5-turbo"      // Default to efficient
    }
}
```

### 2. Prompt Engineering

```go
// Effective system prompts
func createEffectiveSystemPrompt(role string) string {
    prompts := map[string]string{
        "coder": `You are an expert programmer with deep knowledge of software engineering best practices.
- Write clean, efficient, well-documented code
- Follow SOLID principles and design patterns
- Include error handling and edge cases
- Explain complex concepts clearly`,
        
        "analyst": `You are a data analyst specializing in business intelligence.
- Provide data-driven insights
- Use clear visualizations and metrics
- Identify trends and patterns
- Make actionable recommendations`,
        
        "teacher": `You are an engaging teacher who makes complex topics accessible.
- Break down concepts into simple steps
- Use analogies and examples
- Check understanding with questions
- Adapt explanations to the student's level`,
    }
    
    return prompts[role]
}
```

### 3. Temperature Settings

```go
// Temperature guidelines
func getTemperature(useCase string) float32 {
    temperatures := map[string]float32{
        "factual_qa":      0.0,  // Deterministic
        "code_generation": 0.2,  // Mostly deterministic
        "analysis":        0.3,  // Slight variation
        "summarization":   0.5,  // Balanced
        "conversation":    0.7,  // Natural variation
        "creative":        0.9,  // Creative writing
        "brainstorming":   1.2,  // Maximum creativity
    }
    
    if temp, ok := temperatures[useCase]; ok {
        return temp
    }
    return 0.7 // Default
}
```

### 4. Streaming Best Practices

```go
// Robust streaming with timeout and cancellation
func robustStreaming(provider *openai.Provider) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    stream, err := provider.StreamText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Tell me a long story"},
                },
            },
        },
        Stream: true,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    timeout := time.NewTimer(25 * time.Second)
    defer timeout.Stop()
    
    for {
        select {
        case event, ok := <-stream.Events():
            if !ok {
                return // Stream closed
            }
            handleEvent(event)
            
        case <-timeout.C:
            fmt.Println("Stream timeout - closing")
            return
            
        case <-ctx.Done():
            fmt.Println("Context cancelled")
            return
        }
    }
}
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Rate Limiting

**Problem**: Getting rate limit errors frequently

**Solution**:
```go
// Implement exponential backoff
provider := openai.New(
    openai.WithAPIKey(apiKey),
    openai.WithMaxRetries(5),
    openai.WithRetryDelay(time.Second),
)

// Or use middleware
provider = middleware.WithRateLimit(
    provider,
    middleware.RateLimitOpts{
        RequestsPerSecond: 10,
        Burst: 20,
    },
)
```

#### 2. Context Length Exceeded

**Problem**: "Context length exceeded" errors

**Solution**:
```go
// Implement sliding window
func maintainContextWindow(messages []core.Message) []core.Message {
    const maxMessages = 20
    
    if len(messages) <= maxMessages {
        return messages
    }
    
    // Keep system message + recent messages
    result := []core.Message{messages[0]} // System
    result = append(result, messages[len(messages)-maxMessages+1:]...)
    return result
}
```

#### 3. Timeout Issues

**Problem**: Requests timing out

**Solution**:
```go
// Increase timeout for long operations
provider := openai.New(
    openai.WithAPIKey(apiKey),
    openai.WithTimeout(120 * time.Second), // 2 minutes
)

// Or use context with custom timeout
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
defer cancel()
```

#### 4. JSON Mode Not Working

**Problem**: Not getting valid JSON in structured outputs

**Solution**:
```go
// Ensure proper system prompt
response, err := provider.GenerateObject[MyStruct](ctx, core.Request{
    Messages: []core.Message{
        {
            Role: core.System,
            Parts: []core.Part{
                core.Text{Text: "You must respond with valid JSON only."},
            },
        },
        // ... rest of messages
    },
    // Provider will automatically enable JSON mode
})
```

## Summary

The OpenAI provider in GAI offers:
- Full access to GPT-4, GPT-3.5, and vision models
- Streaming, function calling, and structured outputs
- Robust error handling and retry logic
- Cost optimization strategies
- Production-ready features

Key takeaways:
- Choose the right model for your use case
- Use streaming for better UX
- Implement proper error handling
- Optimize token usage for cost
- Follow best practices for prompts and temperature

Next steps:
- Explore [Function Calling](../features/tool-calling.md) in depth
- Learn about [Structured Outputs](../features/structured-outputs.md)
- Set up [Observability](../features/observability.md)
- Review [Cost Optimization](../guides/cost-optimization.md) strategies