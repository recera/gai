# Anthropic Provider Guide

This comprehensive guide covers everything you need to know about using Anthropic's Claude models with GAI, including Claude 3 Opus, Sonnet, and Haiku, along with advanced features like vision, function calling, and streaming.

## Table of Contents
- [Overview](#overview)
- [Installation & Setup](#installation--setup)
- [Configuration](#configuration)
- [Supported Models](#supported-models)
- [Basic Usage](#basic-usage)
- [Streaming](#streaming)
- [Function Calling](#function-calling)
- [Vision Capabilities](#vision-capabilities)
- [System Prompts](#system-prompts)
- [Advanced Features](#advanced-features)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- [Cost Optimization](#cost-optimization)
- [Migration from OpenAI](#migration-from-openai)
- [Troubleshooting](#troubleshooting)

## Overview

The Anthropic provider gives you access to the Claude family of models, known for:
- **Claude 3 Opus**: Most capable model for complex tasks
- **Claude 3 Sonnet**: Balanced performance and cost
- **Claude 3 Haiku**: Fast and affordable
- **Claude 2.1**: Previous generation with 200K context
- **Claude Instant**: Fastest response times

### Key Features
- ✅ 200K+ token context windows
- ✅ Advanced reasoning and analysis
- ✅ Vision capabilities (Claude 3)
- ✅ Function calling
- ✅ Streaming responses
- ✅ Constitutional AI for safety
- ✅ Strong coding capabilities
- ✅ Multi-language support

### Anthropic's Unique Strengths
- **Helpful, Harmless, and Honest**: Built with constitutional AI
- **Long Context**: Up to 200K tokens (500+ pages)
- **Nuanced Understanding**: Excellent at complex instructions
- **Safety-First**: Reduced hallucinations and harmful outputs
- **Research Quality**: Strong analytical and writing capabilities

## Installation & Setup

### 1. Install the Anthropic Provider

```bash
go get github.com/yourusername/gai/providers/anthropic@latest
```

### 2. Obtain an API Key

1. Visit [console.anthropic.com](https://console.anthropic.com)
2. Create an account (may require approval)
3. Navigate to API Keys
4. Create a new API key
5. Copy the key (starts with `sk-ant-`)

### 3. Set Up Environment

```bash
# Set your API key
export ANTHROPIC_API_KEY="sk-ant-...your-key-here..."

# Optional: Set default model
export ANTHROPIC_MODEL="claude-3-opus-20240229"

# Optional: Set API version
export ANTHROPIC_VERSION="2024-02-15"
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
    "github.com/yourusername/gai/providers/anthropic"
)

func main() {
    // Create provider
    provider := anthropic.New(
        anthropic.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
    )
    
    // Test with a simple request
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Say 'Hello from Claude!'"},
                },
            },
        },
        MaxTokens: 10,
    })
    
    if err != nil {
        log.Fatalf("Setup verification failed: %v", err)
    }
    
    fmt.Println("✅ Anthropic provider is working!")
    fmt.Println("Response:", response.Text)
}
```

## Configuration

### Basic Configuration

```go
provider := anthropic.New(
    anthropic.WithAPIKey("sk-ant-..."),              // Required
    anthropic.WithModel("claude-3-opus-20240229"),   // Default model
    anthropic.WithVersion("2024-02-15"),             // API version
    anthropic.WithBaseURL("https://api.anthropic.com"), // Custom endpoint
    anthropic.WithTimeout(60 * time.Second),         // Request timeout
    anthropic.WithMaxRetries(3),                     // Retry attempts
)
```

### Advanced Configuration

```go
// Custom HTTP client configuration
httpClient := &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
    Timeout: 120 * time.Second,
}

// Beta features configuration
provider := anthropic.New(
    anthropic.WithAPIKey(apiKey),
    anthropic.WithHTTPClient(httpClient),
    anthropic.WithBetaFeatures([]string{
        "tools-2024-04-04",        // Enhanced tool calling
        "max-tokens-3-5-sonnet",   // Extended token limits
    }),
    anthropic.WithMetricsCollector(metricsCollector),
)
```

### Environment-Based Configuration

```go
// Load configuration from environment
func createProviderFromEnv() *anthropic.Provider {
    return anthropic.New(
        anthropic.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
        anthropic.WithModel(getEnvOrDefault("ANTHROPIC_MODEL", "claude-3-sonnet-20240229")),
        anthropic.WithVersion(getEnvOrDefault("ANTHROPIC_VERSION", "2024-02-15")),
    )
}

func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

## Supported Models

### Claude 3 Family (Latest)

```go
// Claude 3 Opus - Most capable
provider := anthropic.New(
    anthropic.WithModel("claude-3-opus-20240229"),
)

// Claude 3 Sonnet - Balanced
provider := anthropic.New(
    anthropic.WithModel("claude-3-sonnet-20240229"),
)

// Claude 3 Haiku - Fast & affordable
provider := anthropic.New(
    anthropic.WithModel("claude-3-haiku-20240307"),
)
```

### Previous Generation

```go
// Claude 2.1 - 200K context
provider := anthropic.New(
    anthropic.WithModel("claude-2.1"),
)

// Claude Instant - Fastest
provider := anthropic.New(
    anthropic.WithModel("claude-instant-1.2"),
)
```

### Model Comparison

| Model | Context Window | Strengths | Best For | Speed | Cost |
|-------|---------------|-----------|----------|-------|------|
| Claude 3 Opus | 200K | Most capable, best reasoning | Complex analysis, research | Slower | $$$$$ |
| Claude 3 Sonnet | 200K | Balanced performance | General use, coding | Medium | $$$ |
| Claude 3 Haiku | 200K | Fast, efficient | Quick tasks, high volume | Fast | $ |
| Claude 2.1 | 200K | Reliable, proven | Production systems | Medium | $$ |
| Claude Instant | 100K | Very fast | Real-time apps | Fastest | $ |

## Basic Usage

### Simple Text Generation

```go
func generateText(provider *anthropic.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Explain the concept of recursion in programming with an example."},
                },
            },
        },
        MaxTokens:   500,
        Temperature: 0.7,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Claude's Response:")
    fmt.Println(response.Text)
    
    // Print usage statistics
    fmt.Printf("\nTokens: Input=%d, Output=%d, Total=%d\n",
        response.Usage.InputTokens,
        response.Usage.OutputTokens,
        response.Usage.TotalTokens)
}
```

### Conversation with Context

```go
func conversationExample(provider *anthropic.Provider) {
    ctx := context.Background()
    
    messages := []core.Message{
        {
            Role: core.System,
            Parts: []core.Part{
                core.Text{Text: "You are a helpful AI assistant specializing in technical explanations. Be concise but thorough."},
            },
        },
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "What is a monad in functional programming?"},
            },
        },
        {
            Role: core.Assistant,
            Parts: []core.Part{
                core.Text{Text: "A monad is a design pattern in functional programming that provides a way to wrap values and chain operations while handling complexity like null values, errors, or asynchronous operations behind the scenes."},
            },
        },
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Can you give me a practical example in JavaScript?"},
            },
        },
    }
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages:    messages,
        MaxTokens:   800,
        Temperature: 0.5,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Claude:", response.Text)
}
```

### Long Context Processing

```go
func processLongDocument(provider *anthropic.Provider, document string) {
    ctx := context.Background()
    
    // Claude can handle very long contexts (up to 200K tokens)
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are an expert document analyzer. Provide comprehensive analysis."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: fmt.Sprintf(`Please analyze this document and provide:
1. Executive summary (3-5 sentences)
2. Key themes and topics
3. Important findings or conclusions
4. Any potential issues or concerns
5. Recommendations for next steps

Document:
%s`, document)},
                },
            },
        },
        MaxTokens:   2000,
        Temperature: 0.3, // Lower temperature for analytical tasks
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Document Analysis:")
    fmt.Println(response.Text)
}
```

## Streaming

### Basic Streaming Example

```go
func streamingExample(provider *anthropic.Provider) {
    ctx := context.Background()
    
    stream, err := provider.StreamText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a detailed guide on building REST APIs in Go."},
                },
            },
        },
        MaxTokens:   2000,
        Temperature: 0.7,
        Stream:      true,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    fmt.Println("Streaming Claude's response:")
    fmt.Println("=" + strings.Repeat("=", 50))
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            fmt.Print(event.TextDelta)
        case core.EventError:
            log.Printf("Stream error: %v", event.Err)
        case core.EventFinish:
            fmt.Println("\n" + strings.Repeat("=", 50))
            fmt.Println("Stream complete!")
        }
    }
}
```

### Advanced Streaming with Metadata

```go
func advancedStreaming(provider *anthropic.Provider) {
    ctx := context.Background()
    
    stream, err := provider.StreamText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Explain quantum computing step by step."},
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
        fullText    strings.Builder
        tokenCount  int
        startTime   = time.Now()
    )
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventStart:
            fmt.Println("Claude is thinking...")
            
        case core.EventTextDelta:
            text := event.TextDelta
            fmt.Print(text)
            fullText.WriteString(text)
            tokenCount++ // Approximate
            
        case core.EventFinish:
            elapsed := time.Since(startTime)
            fmt.Printf("\n\n--- Statistics ---\n")
            fmt.Printf("Total characters: %d\n", fullText.Len())
            fmt.Printf("Approximate tokens: %d\n", tokenCount)
            fmt.Printf("Time taken: %v\n", elapsed)
            fmt.Printf("Speed: %.2f tokens/sec\n", float64(tokenCount)/elapsed.Seconds())
        }
    }
}
```

## Function Calling

### Defining Tools for Claude

```go
// Search tool
type SearchInput struct {
    Query    string   `json:"query" jsonschema:"required,description=Search query"`
    Filters  []string `json:"filters,omitempty" jsonschema:"description=Optional filters"`
    MaxResults int    `json:"max_results,omitempty" jsonschema:"default=10,minimum=1,maximum=50"`
}

type SearchOutput struct {
    Results []SearchResult `json:"results"`
    Total   int           `json:"total"`
}

type SearchResult struct {
    Title   string `json:"title"`
    URL     string `json:"url"`
    Snippet string `json:"snippet"`
    Score   float64 `json:"relevance_score"`
}

func createSearchTool() tools.Handle {
    return tools.New[SearchInput, SearchOutput](
        "web_search",
        "Search the web for information",
        func(ctx context.Context, input SearchInput, meta tools.Meta) (SearchOutput, error) {
            // Implement actual search logic here
            return SearchOutput{
                Results: []SearchResult{
                    {
                        Title:   "Example Result",
                        URL:     "https://example.com",
                        Snippet: "This is a sample search result...",
                        Score:   0.95,
                    },
                },
                Total: 1,
            }, nil
        },
    )
}

// Calculator tool
type CalculatorInput struct {
    Expression string `json:"expression" jsonschema:"required,description=Mathematical expression"`
    Precision  int    `json:"precision,omitempty" jsonschema:"default=2,minimum=0,maximum=10"`
}

type CalculatorOutput struct {
    Result      float64 `json:"result"`
    Explanation string  `json:"explanation,omitempty"`
}

func createCalculatorTool() tools.Handle {
    return tools.New[CalculatorInput, CalculatorOutput](
        "calculator",
        "Perform mathematical calculations",
        func(ctx context.Context, input CalculatorInput, meta tools.Meta) (CalculatorOutput, error) {
            // Use a safe expression evaluator in production
            result := evaluateMathExpression(input.Expression)
            
            return CalculatorOutput{
                Result:      roundToPrecision(result, input.Precision),
                Explanation: fmt.Sprintf("Calculated: %s = %f", input.Expression, result),
            }, nil
        },
    )
}
```

### Using Tools with Claude

```go
func toolCallingExample(provider *anthropic.Provider) {
    searchTool := createSearchTool()
    calcTool := createCalculatorTool()
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Search for information about the speed of light and calculate how long it takes light to travel from the Sun to Earth (distance is 93 million miles)."},
                },
            },
        },
        Tools:      []tools.Handle{searchTool, calcTool},
        ToolChoice: core.ToolAuto,
        MaxTokens:  1000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Claude's Response with Tools:")
    fmt.Println(response.Text)
    
    // Display tool usage
    for i, step := range response.Steps {
        if len(step.ToolCalls) > 0 {
            fmt.Printf("\nStep %d - Tools Used:\n", i+1)
            for _, call := range step.ToolCalls {
                fmt.Printf("  - %s\n", call.Name)
            }
        }
    }
}
```

### Complex Multi-Tool Workflow

```go
func complexToolWorkflow(provider *anthropic.Provider) {
    // Create multiple specialized tools
    tools := []tools.Handle{
        createDatabaseQueryTool(),
        createDataAnalysisTool(),
        createVisualizationTool(),
        createReportGeneratorTool(),
    }
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are a data analyst assistant. Use the available tools to help with analysis tasks."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Analyze our Q4 sales data: query the database for sales by region, identify trends, create visualizations, and generate an executive report."},
                },
            },
        },
        Tools:      tools,
        ToolChoice: core.ToolAuto,
        MaxTokens:  2000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Analysis Complete:")
    fmt.Println(response.Text)
}
```

## Vision Capabilities

### Image Analysis with Claude 3

```go
func imageAnalysis(provider *anthropic.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "claude-3-opus-20240229", // Vision-capable model
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Analyze this chart and provide insights:"},
                    core.ImageURL{
                        URL:    "https://example.com/sales-chart.png",
                        Detail: "high",
                    },
                    core.Text{Text: "Focus on trends, anomalies, and actionable recommendations."},
                },
            },
        },
        MaxTokens: 1000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Chart Analysis:")
    fmt.Println(response.Text)
}
```

### Multiple Image Comparison

```go
func compareImages(provider *anthropic.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "claude-3-sonnet-20240229",
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Compare these two architectural designs:"},
                    core.Text{Text: "Design A:"},
                    core.ImageURL{URL: "https://example.com/design-a.jpg"},
                    core.Text{Text: "Design B:"},
                    core.ImageURL{URL: "https://example.com/design-b.jpg"},
                    core.Text{Text: "Evaluate: aesthetics, functionality, cost-effectiveness, and sustainability."},
                },
            },
        },
        MaxTokens:   1500,
        Temperature: 0.5,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Design Comparison:")
    fmt.Println(response.Text)
}
```

### OCR and Document Processing

```go
func processScannedDocument(provider *anthropic.Provider, imageData []byte) {
    // Convert image to base64
    base64Image := base64.StdEncoding.EncodeToString(imageData)
    dataURL := fmt.Sprintf("data:image/png;base64,%s", base64Image)
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "claude-3-opus-20240229",
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Extract and structure the text from this document:"},
                    core.ImageURL{URL: dataURL, Detail: "high"},
                    core.Text{Text: "Provide: 1) Full text extraction 2) Document structure 3) Key information summary"},
                },
            },
        },
        MaxTokens: 2000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Document Extraction:")
    fmt.Println(response.Text)
}
```

## System Prompts

### Effective System Prompts for Claude

```go
// Claude responds well to detailed, structured system prompts
func createSystemPrompt(role string) string {
    systemPrompts := map[string]string{
        "analyst": `You are Claude, an AI assistant created by Anthropic to be helpful, harmless, and honest.
        
Your role: Senior Data Analyst
Your expertise:
- Statistical analysis and data interpretation
- Business intelligence and reporting
- Predictive modeling and forecasting
- Data visualization best practices

Guidelines:
- Always base conclusions on data
- Acknowledge uncertainty when appropriate
- Provide confidence levels for predictions
- Suggest additional analyses when relevant
- Use clear, non-technical language for business stakeholders`,

        "coder": `You are Claude, an AI assistant created by Anthropic to be helpful, harmless, and honest.

Your role: Expert Software Engineer
Your approach:
- Write clean, maintainable, production-ready code
- Follow best practices and design patterns
- Include comprehensive error handling
- Add helpful comments and documentation
- Consider edge cases and performance
- Suggest tests for critical functionality

When writing code:
- Use meaningful variable names
- Keep functions small and focused
- Apply SOLID principles
- Ensure code is secure by default`,

        "teacher": `You are Claude, an AI assistant created by Anthropic to be helpful, harmless, and honest.

Your role: Patient and Engaging Teacher
Teaching style:
- Break complex concepts into simple steps
- Use analogies and real-world examples
- Check understanding with questions
- Provide practice problems when appropriate
- Adapt explanations to the student's level
- Encourage curiosity and deeper learning

Remember to:
- Be encouraging and supportive
- Acknowledge when students are struggling
- Celebrate progress and understanding`,
    }
    
    return systemPrompts[role]
}
```

### Using System Prompts Effectively

```go
func effectiveSystemPromptUsage(provider *anthropic.Provider) {
    ctx := context.Background()
    
    // Detailed system prompt for specific behavior
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: `You are a technical documentation expert. 
                    
Your task is to write clear, comprehensive documentation that:
- Uses consistent formatting and structure
- Includes practical examples for every concept
- Provides both beginner and advanced perspectives
- Highlights common pitfalls and best practices
- Uses diagrams and visual aids where helpful

Format your responses with:
- Clear headings and subheadings
- Bulleted lists for key points
- Code blocks with syntax highlighting hints
- Tables for comparisons
- Links to relevant resources`},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Document the process of setting up a CI/CD pipeline with GitHub Actions."},
                },
            },
        },
        MaxTokens:   2000,
        Temperature: 0.6,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response.Text)
}
```

## Advanced Features

### Constitutional AI Principles

```go
// Claude is trained with constitutional AI for safety
func safeGeneration(provider *anthropic.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are a helpful, harmless, and honest AI assistant. Always prioritize user safety and well-being."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Help me understand the ethical implications of AI in healthcare."},
                },
            },
        },
        MaxTokens:   1000,
        Temperature: 0.5,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Ethical Analysis:")
    fmt.Println(response.Text)
}
```

### Structured Output with Claude

```go
type ResearchPaper struct {
    Title       string   `json:"title"`
    Abstract    string   `json:"abstract"`
    Authors     []string `json:"authors"`
    Keywords    []string `json:"keywords"`
    Sections    []Section `json:"sections"`
    Conclusions string   `json:"conclusions"`
    References  []string `json:"references"`
}

type Section struct {
    Heading string `json:"heading"`
    Content string `json:"content"`
}

func structuredResearch(provider *anthropic.Provider) {
    ctx := context.Background()
    
    result, err := provider.GenerateObject[ResearchPaper](ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "Generate a structured research paper outline on the given topic."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Create a research paper structure on: The Impact of Large Language Models on Software Development"},
                },
            },
        },
        MaxTokens:   2000,
        Temperature: 0.7,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    paper := result.Value
    fmt.Printf("Title: %s\n", paper.Title)
    fmt.Printf("Abstract: %s\n", paper.Abstract)
    fmt.Printf("Keywords: %v\n", paper.Keywords)
    
    for _, section := range paper.Sections {
        fmt.Printf("\n## %s\n%s\n", section.Heading, section.Content)
    }
}
```

### Handling Long Contexts

```go
func longContextProcessing(provider *anthropic.Provider) {
    // Claude can handle up to 200K tokens (~500 pages)
    
    // Load a large document
    largeDocument := loadLargeDocument() // Your function to load content
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are an expert at analyzing large documents and extracting key insights."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: fmt.Sprintf(`Analyze this entire document and provide:
                    
1. Executive Summary (200 words)
2. Main Themes (bulleted list)
3. Key Findings (numbered list)
4. Critical Issues Identified
5. Recommendations
6. Areas Requiring Further Investigation

Document:
%s`, largeDocument)},
                },
            },
        },
        MaxTokens:   3000,
        Temperature: 0.3, // Lower for analytical tasks
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Document Analysis:")
    fmt.Println(response.Text)
}
```

## Error Handling

### Comprehensive Error Handling

```go
func handleAnthropicErrors(provider *anthropic.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Test message"},
                },
            },
        },
    })
    
    if err != nil {
        switch {
        case core.IsRateLimited(err):
            fmt.Println("Rate limited by Anthropic")
            waitTime := core.GetRetryAfter(err)
            fmt.Printf("Waiting %v before retry...\n", waitTime)
            time.Sleep(waitTime)
            // Retry logic here
            
        case core.IsContextLengthExceeded(err):
            fmt.Println("Message too long for Claude")
            // Implement context reduction strategy
            
        case core.IsInvalidRequest(err):
            fmt.Println("Invalid request format:", err)
            // Fix request parameters
            
        case core.IsUnauthorized(err):
            fmt.Println("API key issue:", err)
            // Check API key configuration
            
        case core.IsSafetyBlocked(err):
            fmt.Println("Content blocked by safety filters")
            // Adjust content or handle gracefully
            
        case core.IsProviderUnavailable(err):
            fmt.Println("Anthropic service is unavailable")
            // Use fallback or queue for later
            
        default:
            fmt.Printf("Unexpected error: %v\n", err)
        }
        return
    }
    
    fmt.Println("Success:", response.Text)
}
```

### Retry Strategy

```go
func robustAnthropicRequest(provider *anthropic.Provider) {
    maxRetries := 3
    baseDelay := time.Second
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        
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
        
        if !core.IsTransient(err) {
            fmt.Printf("Non-retryable error: %v\n", err)
            return
        }
        
        delay := baseDelay * time.Duration(math.Pow(2, float64(attempt)))
        fmt.Printf("Attempt %d failed, retrying in %v...\n", attempt+1, delay)
        time.Sleep(delay)
    }
    
    fmt.Println("All retries exhausted")
}
```

## Best Practices

### 1. Optimal Model Selection

```go
func selectClaudeModel(task string) string {
    modelSelection := map[string]string{
        "complex_analysis":   "claude-3-opus-20240229",    // Most capable
        "general_tasks":      "claude-3-sonnet-20240229",  // Balanced
        "quick_responses":    "claude-3-haiku-20240307",   // Fast
        "vision_tasks":       "claude-3-opus-20240229",    // Best vision
        "coding":            "claude-3-sonnet-20240229",   // Good for code
        "creative_writing":   "claude-3-opus-20240229",    // Most creative
        "data_extraction":    "claude-3-haiku-20240307",   // Fast & cheap
        "long_context":       "claude-3-opus-20240229",    // Best for long docs
    }
    
    if model, ok := modelSelection[task]; ok {
        return model
    }
    return "claude-3-sonnet-20240229" // Default
}
```

### 2. Prompt Engineering for Claude

```go
func claudeOptimizedPrompt(taskType string) string {
    // Claude responds well to clear structure and explicit instructions
    prompts := map[string]string{
        "analysis": `Please analyze the provided information following this structure:

1. Overview: Brief summary of the main points
2. Detailed Analysis: 
   - Key findings with evidence
   - Patterns and trends identified
   - Potential issues or concerns
3. Recommendations: Actionable next steps
4. Confidence Assessment: Rate your confidence in each conclusion

Use bullet points for clarity and cite specific data when available.`,

        "coding": `Please provide a solution with the following components:

1. Approach: Brief explanation of the solution strategy
2. Implementation: Complete, production-ready code with:
   - Proper error handling
   - Clear comments
   - Edge case consideration
3. Testing: Example test cases
4. Complexity: Time and space complexity analysis
5. Alternatives: Brief mention of other approaches

Ensure the code follows best practices and is maintainable.`,

        "creative": `Create content that is:
- Original and engaging
- Appropriate for the target audience
- Well-structured with clear flow
- Rich in detail and imagery
- Factually accurate where applicable

Feel free to be creative while maintaining coherence and quality.`,
    }
    
    return prompts[taskType]
}
```

### 3. Temperature Guidelines

```go
func getClaudeTemperature(useCase string) float32 {
    temperatures := map[string]float32{
        "factual_qa":        0.0,  // Most deterministic
        "analysis":          0.2,  // Focused analysis
        "coding":            0.3,  // Consistent code
        "summarization":     0.4,  // Balanced summaries
        "general_chat":      0.7,  // Natural conversation
        "creative_writing":  0.9,  // Creative content
        "brainstorming":     1.0,  // Maximum creativity
    }
    
    if temp, ok := temperatures[useCase]; ok {
        return temp
    }
    return 0.7 // Default
}
```

### 4. Context Management

```go
func manageClaudeContext(messages []core.Message, maxTokens int) []core.Message {
    // Claude handles long contexts well, but we still need to manage tokens
    
    const (
        // Claude's context windows
        claudeMaxTokens = 200000
        // Reserve tokens for response
        responseBuffer = 4000
    )
    
    availableTokens := minInt(maxTokens, claudeMaxTokens-responseBuffer)
    
    // Estimate current token usage
    currentTokens := estimateTokens(messages)
    
    if currentTokens <= availableTokens {
        return messages
    }
    
    // Prioritize keeping system message and recent context
    return truncateMessages(messages, availableTokens)
}
```

## Cost Optimization

### Token-Efficient Strategies

```go
func optimizeClaudeCosts() {
    // Use appropriate models for different tasks
    
    // For simple tasks, use Haiku
    haikuProvider := anthropic.New(
        anthropic.WithModel("claude-3-haiku-20240307"),
    )
    
    // For complex tasks, use Opus
    opusProvider := anthropic.New(
        anthropic.WithModel("claude-3-opus-20240229"),
    )
    
    // Route based on task complexity
    func routeRequest(complexity string, request core.Request) {
        switch complexity {
        case "simple":
            haikuProvider.GenerateText(ctx, request)
        case "complex":
            opusProvider.GenerateText(ctx, request)
        default:
            // Use Sonnet for balanced performance/cost
            sonnetProvider.GenerateText(ctx, request)
        }
    }
}
```

### Caching Strategies

```go
type ClaudeCache struct {
    cache sync.Map
    ttl   time.Duration
}

func (c *ClaudeCache) GetOrGenerate(
    ctx context.Context,
    provider *anthropic.Provider,
    request core.Request,
) (*core.TextResult, error) {
    key := generateCacheKey(request)
    
    // Check cache
    if cached, ok := c.cache.Load(key); ok {
        if entry, ok := cached.(*cacheEntry); ok {
            if time.Since(entry.timestamp) < c.ttl {
                return entry.result, nil
            }
        }
    }
    
    // Generate new response
    result, err := provider.GenerateText(ctx, request)
    if err != nil {
        return nil, err
    }
    
    // Cache successful response
    c.cache.Store(key, &cacheEntry{
        result:    result,
        timestamp: time.Now(),
    })
    
    return result, nil
}
```

## Migration from OpenAI

### Code Migration Guide

```go
// OpenAI code
openaiProvider := openai.New(
    openai.WithAPIKey(openaiKey),
    openai.WithModel("gpt-4"),
)

// Equivalent Anthropic code
anthropicProvider := anthropic.New(
    anthropic.WithAPIKey(anthropicKey),
    anthropic.WithModel("claude-3-opus-20240229"),
)

// The request structure remains the same!
request := core.Request{
    Messages: []core.Message{
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Hello!"},
            },
        },
    },
}

// Both work identically
openaiResponse, _ := openaiProvider.GenerateText(ctx, request)
anthropicResponse, _ := anthropicProvider.GenerateText(ctx, request)
```

### Feature Mapping

| OpenAI Feature | Anthropic Equivalent | Notes |
|---------------|---------------------|-------|
| GPT-4 | Claude 3 Opus | Similar capabilities |
| GPT-3.5 Turbo | Claude 3 Haiku | Fast, affordable |
| Function Calling | Tool Use | Same implementation in GAI |
| JSON Mode | Structured Output | Automatic in GAI |
| Vision (GPT-4V) | Claude 3 Vision | All Claude 3 models |
| 128K context | 200K context | Claude has larger context |
| Streaming | Streaming | Identical API |

## Troubleshooting

### Common Issues and Solutions

#### 1. API Key Not Working

**Problem**: Getting authentication errors

**Solution**:
```go
// Verify API key format
if !strings.HasPrefix(apiKey, "sk-ant-") {
    log.Fatal("Invalid Anthropic API key format")
}

// Check API key permissions
provider := anthropic.New(
    anthropic.WithAPIKey(apiKey),
    anthropic.WithDebug(true), // Enable debug logging
)
```

#### 2. Rate Limiting

**Problem**: Frequent rate limit errors

**Solution**:
```go
// Implement exponential backoff
provider := anthropic.New(
    anthropic.WithAPIKey(apiKey),
    anthropic.WithMaxRetries(5),
    anthropic.WithRetryDelay(2 * time.Second),
)

// Or use rate limiting middleware
rateLimited := middleware.WithRateLimit(
    provider,
    middleware.RateLimitOpts{
        RequestsPerMinute: 50,
    },
)
```

#### 3. Context Length Issues

**Problem**: "Maximum context length exceeded"

**Solution**:
```go
// Implement sliding window for conversations
func maintainContextForClaude(messages []core.Message) []core.Message {
    const maxMessages = 50 // Adjust based on content
    
    if len(messages) <= maxMessages {
        return messages
    }
    
    // Keep system message and recent history
    result := []core.Message{messages[0]}
    result = append(result, messages[len(messages)-maxMessages+1:]...)
    return result
}
```

#### 4. Vision Not Working

**Problem**: Images not being processed

**Solution**:
```go
// Ensure using Claude 3 model
provider := anthropic.New(
    anthropic.WithModel("claude-3-opus-20240229"), // or sonnet/haiku
)

// Verify image format
validFormats := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
```

## Summary

The Anthropic provider in GAI offers:
- Access to Claude 3 family with superior reasoning
- 200K token context windows
- Vision capabilities across all Claude 3 models
- Strong safety and constitutional AI
- Excellent performance on complex tasks

Key advantages over other providers:
- Larger context windows (200K vs 128K)
- Better at following complex instructions
- Stronger safety guarantees
- More nuanced understanding
- Better at refusing harmful requests appropriately

Best practices:
- Use Opus for complex tasks, Haiku for simple ones
- Leverage the large context window for document analysis
- Take advantage of Claude's strong reasoning abilities
- Use clear, structured prompts for best results

Next steps:
- Explore [Tool Calling](../features/tool-calling.md) with Claude
- Learn about [Long Context Processing](../guides/long-context.md)
- Review [Cost Optimization](../guides/cost-optimization.md) strategies
- Try [Vision Capabilities](../features/vision.md) with Claude 3