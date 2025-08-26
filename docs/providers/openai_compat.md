# OpenAI-Compatible Provider Guide

This comprehensive guide covers everything you need to know about using the OpenAI-Compatible provider with GAI, including preset configurations for popular providers like Groq, xAI, Cerebras, Together, Fireworks, and others that implement the OpenAI API specification.

## Table of Contents
- [Overview](#overview)
- [Installation & Setup](#installation--setup)
- [Provider Presets](#provider-presets)
- [Custom Configuration](#custom-configuration)
- [Supported Providers](#supported-providers)
- [Basic Usage](#basic-usage)
- [Streaming](#streaming)
- [Tool Calling](#tool-calling)
- [Structured Outputs](#structured-outputs)
- [Provider-Specific Features](#provider-specific-features)
- [Capability Detection](#capability-detection)
- [Advanced Configuration](#advanced-configuration)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- [Performance Comparison](#performance-comparison)
- [Migration Guide](#migration-guide)
- [Troubleshooting](#troubleshooting)

## Overview

The OpenAI-Compatible provider serves as a universal adapter for any API that implements the OpenAI Chat Completions specification. This enables you to use a wide variety of AI providers with a consistent interface while automatically handling provider-specific quirks and limitations.

### Key Features
- ✅ **Universal Compatibility**: Works with any OpenAI-compatible API endpoint
- ✅ **Provider Presets**: Ready-to-use configurations for popular providers
- ✅ **Automatic Quirk Handling**: Manages provider-specific limitations transparently
- ✅ **Capability Detection**: Probes and caches provider capabilities
- ✅ **Flexible Configuration**: Fine-grained control over provider behavior
- ✅ **Complete Tool Support**: Full tool calling with parallel execution
- ✅ **Structured Outputs**: JSON Schema-based generation
- ✅ **Comprehensive Streaming**: Real-time text streaming
- ✅ **Error Mapping**: Consistent error handling across providers
- ✅ **Custom Headers**: Provider-specific authentication and configuration

### OpenAI-Compatible Ecosystem
- **High-Performance Providers**: Groq, Cerebras (ultra-fast inference)
- **Model Variety**: Together, Fireworks, Anyscale (hundreds of models)
- **Specialized Services**: xAI (Grok models), Baseten (custom deployments)
- **Cost Optimization**: Various providers offer competitive pricing
- **Geographic Distribution**: Different regions and latency profiles

## Installation & Setup

### 1. Install the OpenAI-Compatible Provider

```bash
go get github.com/recera/gai/providers/openai_compat@latest
```

### 2. Choose Your Provider

You'll need API keys for the providers you want to use:

```bash
# Groq (very fast)
export GROQ_API_KEY="your-groq-api-key"

# xAI (Grok models)
export XAI_API_KEY="your-xai-api-key"

# Together (wide model selection)
export TOGETHER_API_KEY="your-together-api-key"

# Fireworks (fast open-source models)
export FIREWORKS_API_KEY="your-fireworks-api-key"

# Cerebras (extremely fast)
export CEREBRAS_API_KEY="your-cerebras-api-key"
```

### 3. Quick Start with Presets

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/recera/gai/core"
    "github.com/recera/gai/providers/openai_compat"
)

func main() {
    // Use Groq preset for ultra-fast inference
    provider, err := openai_compat.Groq()
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Explain machine learning in simple terms."},
                },
            },
        },
        MaxTokens: 150,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Response:", response.Text)
    fmt.Printf("Tokens: %d, Time: fast! ⚡\n", response.Usage.TotalTokens)
}
```

## Provider Presets

### Groq - Ultra-Fast Inference

Groq provides extremely fast inference with open-source models.

```go
// Basic Groq usage
provider, err := openai_compat.Groq()
if err != nil {
    log.Fatal(err)
}

// With custom options
provider, err := openai_compat.Groq(
    openai_compat.WithModel("llama-3.1-8b-instant"),  // Fastest model
    openai_compat.WithAPIKey("your-custom-key"),
    openai_compat.WithMaxRetries(5),
)
```

**Groq Models Available:**
- `llama-3.3-70b-versatile` (default, most capable)
- `llama-3.1-70b-versatile`
- `llama-3.1-8b-instant` (fastest)
- `mixtral-8x7b-32768`
- `gemma2-9b-it`

**Groq Characteristics:**
- ⚡ **Blazing Fast**: Sub-second responses for most queries
- ✅ **Full Features**: Tools, streaming, JSON mode
- ⚠️ **Rate Limits**: Aggressive limits, use retry logic
- 💡 **Best For**: Real-time applications, chatbots, development

### xAI - Grok Models

Access to xAI's Grok family of models with unique capabilities.

```go
// Basic xAI usage
provider, err := openai_compat.XAI()
if err != nil {
    log.Fatal(err)
}

// With specific Grok model
provider, err := openai_compat.XAI(
    openai_compat.WithModel("grok-2-1212"),
)
```

**xAI Models Available:**
- `grok-2-latest` (default, most capable)
- `grok-2-1212`
- `grok-beta`

**xAI Characteristics:**
- 🧠 **Advanced Reasoning**: Strong performance on complex tasks
- 🔄 **Regular Updates**: Frequent model improvements
- 💰 **Premium Pricing**: Higher cost for advanced capabilities
- 💡 **Best For**: Complex reasoning, research, analysis

### Cerebras - Extremely Fast

Cerebras provides ultra-fast inference with specialized hardware.

```go
// Basic Cerebras usage
provider, err := openai_compat.Cerebras()
if err != nil {
    log.Fatal(err)
}

// Note: Cerebras has some limitations
// - No JSON streaming
// - No parallel tool calls
// - Strict rate limits
```

**Cerebras Models Available:**
- `llama-3.3-70b` (default)
- `llama-3.1-70b`
- `llama-3.1-8b`

**Cerebras Characteristics:**
- 🚀 **Ultra Fast**: Extremely fast inference speed
- ⚠️ **Limitations**: No JSON streaming, no parallel tools
- 💰 **Rate Limited**: Strict usage limits
- 💡 **Best For**: High-throughput simple tasks

### Together - Model Variety

Together provides access to hundreds of open-source models.

```go
// Basic Together usage
provider, err := openai_compat.Together()
if err != nil {
    log.Fatal(err)
}

// With specific model
provider, err := openai_compat.Together(
    openai_compat.WithModel("meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo"),
)
```

**Popular Together Models:**
- `meta-llama/Llama-3.3-70B-Instruct-Turbo` (default)
- `meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo`
- `mistralai/Mixtral-8x7B-Instruct-v0.1`
- `NousResearch/Nous-Hermes-2-Mixtral-8x7B-DPO`

**Together Characteristics:**
- 📚 **Model Variety**: Hundreds of open-source models
- 🔧 **Flexibility**: Different model sizes and specializations
- 💰 **Cost Effective**: Competitive pricing
- 💡 **Best For**: Experimentation, model comparison, specialized tasks

### Fireworks - Fast Open-Source

Fireworks provides fast inference for popular open-source models.

```go
// Basic Fireworks usage
provider, err := openai_compat.Fireworks()
if err != nil {
    log.Fatal(err)
}

// With specific model
provider, err := openai_compat.Fireworks(
    openai_compat.WithModel("accounts/fireworks/models/llama-v3p1-8b-instruct"),
)
```

**Fireworks Models:**
- `accounts/fireworks/models/llama-v3p3-70b-instruct` (default)
- `accounts/fireworks/models/llama-v3p1-8b-instruct`
- `accounts/fireworks/models/mixtral-8x7b-instruct`

**Fireworks Characteristics:**
- ⚡ **Fast Performance**: Optimized inference speed
- 🔓 **Open Source Focus**: Popular open-source models
- 💰 **Good Value**: Competitive pricing
- 💡 **Best For**: Production applications, cost-conscious deployments

### Anyscale - Scalable Inference

Anyscale provides scalable inference for open-source models.

```go
// Basic Anyscale usage
provider, err := openai_compat.Anyscale()
if err != nil {
    log.Fatal(err)
}
```

**Anyscale Models:**
- `meta-llama/Meta-Llama-3.1-70B-Instruct`
- `mistralai/Mixtral-8x7B-Instruct-v0.1`

**Anyscale Characteristics:**
- 📈 **Scalability**: Good for variable workloads
- 🔓 **Open Source**: Focus on open-source models
- 🎯 **Enterprise**: Good enterprise support
- 💡 **Best For**: Enterprise deployments, variable loads

### Baseten - Custom Deployments

Baseten allows you to deploy and serve your own models.

```go
// Baseten with custom deployment URL
provider, err := openai_compat.Baseten(
    "https://model-abc123.api.baseten.co/v1",
    openai_compat.WithModel("your-custom-model"),
    openai_compat.WithAPIKey("your-baseten-key"),
)
```

**Baseten Characteristics:**
- 🎯 **Custom Models**: Deploy your own fine-tuned models
- 🔧 **Flexibility**: Full control over model and infrastructure
- 💰 **Variable Pricing**: Based on your deployment
- 💡 **Best For**: Custom models, fine-tuned deployments

## Custom Configuration

### Manual Configuration

For providers not covered by presets, use manual configuration:

```go
provider, err := openai_compat.New(openai_compat.CompatOpts{
    BaseURL:      "https://api.yourprovider.com/v1",
    APIKey:       "your-api-key",
    DefaultModel: "your-model-name",
    ProviderName: "yourprovider",
    
    // Provider limitations
    DisableJSONStreaming:     false,
    DisableParallelToolCalls: false,
    DisableStrictJSONSchema:  false,
    DisableToolChoice:        false,
    
    // Custom configuration
    CustomHeaders: map[string]string{
        "X-Custom-Header": "value",
        "User-Agent":      "YourApp/1.0",
    },
    
    // Retry configuration
    MaxRetries: 3,
    RetryDelay: time.Second,
})
```

### Advanced Configuration

```go
// Custom HTTP client
httpClient := &http.Client{
    Transport: &http.Transport{
        Proxy: http.ProxyFromEnvironment,
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: false,
        },
    },
    Timeout: 30 * time.Second,
}

provider, err := openai_compat.New(openai_compat.CompatOpts{
    BaseURL:      "https://api.provider.com/v1",
    APIKey:       apiKey,
    DefaultModel: "model-name",
    HTTPClient:   httpClient,
    
    // Advanced options
    UnsupportedParams:     []string{"frequency_penalty"}, // Strip these params
    ForceResponseFormat:   "json_object",                 // Force specific format
    PreferResponsesAPI:    true,                          // Use /responses endpoint
    
    // Observability
    MetricsCollector: metricsCollector,
})
```

## Supported Providers

### Verified Compatible Providers

| Provider | Speed | Models | Tools | Streaming | JSON | Pricing |
|----------|-------|--------|-------|-----------|------|---------|
| **Groq** | ⚡⚡⚡ | 5+ | ✅ | ✅ | ✅ | $ |
| **xAI** | ⚡⚡ | 3+ | ✅ | ✅ | ✅ | $$$ |
| **Cerebras** | ⚡⚡⚡ | 3+ | ✅ | ❌* | ❌* | $$ |
| **Together** | ⚡⚡ | 100+ | ✅ | ✅ | ⚠️* | $ |
| **Fireworks** | ⚡⚡ | 20+ | ✅ | ✅ | ⚠️* | $ |
| **Anyscale** | ⚡ | 10+ | ✅ | ✅ | ⚠️* | $$ |
| **Baseten** | ⚡ | Custom | ⚠️* | ✅ | ⚠️* | Variable |

*Limitations handled automatically by the provider

### Testing Provider Compatibility

```go
func testProviderCompatibility(baseURL, apiKey string) {
    provider, err := openai_compat.New(openai_compat.CompatOpts{
        BaseURL:      baseURL,
        APIKey:       apiKey,
        DefaultModel: "test-model",
        ProviderName: "test-provider",
    })
    
    if err != nil {
        fmt.Printf("❌ Provider setup failed: %v\n", err)
        return
    }
    
    ctx := context.Background()
    
    // Test basic generation
    fmt.Println("Testing basic generation...")
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Hello! Respond with just 'OK'"},
                },
            },
        },
        MaxTokens: 5,
    })
    
    if err != nil {
        fmt.Printf("❌ Basic generation failed: %v\n", err)
        return
    }
    
    fmt.Printf("✅ Basic generation works: %s\n", response.Text)
    
    // Test streaming
    fmt.Println("Testing streaming...")
    stream, err := provider.StreamText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Count to 3"},
                },
            },
        },
        Stream: true,
    })
    
    if err != nil {
        fmt.Printf("❌ Streaming failed: %v\n", err)
        return
    }
    
    defer stream.Close()
    streamWorks := false
    
    for event := range stream.Events() {
        if event.Type == core.EventTextDelta {
            streamWorks = true
            break
        }
    }
    
    if streamWorks {
        fmt.Println("✅ Streaming works")
    } else {
        fmt.Println("❌ Streaming doesn't work")
    }
    
    fmt.Println("Provider compatibility test complete!")
}
```

## Basic Usage

### Simple Text Generation

```go
func basicTextGeneration() {
    // Use different providers for comparison
    providers := map[string]func() (*openai_compat.Provider, error){
        "Groq":      openai_compat.Groq,
        "Together":  openai_compat.Together,
        "Fireworks": openai_compat.Fireworks,
    }
    
    prompt := "Explain the concept of recursion in programming."
    
    for name, createProvider := range providers {
        fmt.Printf("\n--- %s Provider ---\n", name)
        
        provider, err := createProvider()
        if err != nil {
            fmt.Printf("Failed to create %s provider: %v\n", name, err)
            continue
        }
        
        start := time.Now()
        response, err := provider.GenerateText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: prompt},
                    },
                },
            },
            MaxTokens:   200,
            Temperature: 0.7,
        })
        
        duration := time.Since(start)
        
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }
        
        fmt.Printf("Response (%v): %s\n", duration, response.Text)
        fmt.Printf("Tokens: %d\n", response.Usage.TotalTokens)
    }
}
```

### Conversation Handling

```go
func conversationExample() {
    provider, err := openai_compat.Groq(
        openai_compat.WithModel("llama-3.1-70b-versatile"),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    messages := []core.Message{
        {
            Role: core.System,
            Parts: []core.Part{
                core.Text{Text: "You are a helpful coding tutor. Provide clear, practical explanations."},
            },
        },
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "I'm learning Go. What's the difference between a slice and an array?"},
            },
        },
        {
            Role: core.Assistant,
            Parts: []core.Part{
                core.Text{Text: "Great question! Arrays have a fixed size defined at compile time (e.g., [5]int), while slices are dynamic and built on top of arrays. Slices are more commonly used in Go because of their flexibility."},
            },
        },
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "Can you show me a code example?"},
            },
        },
    }
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages:    messages,
        Temperature: 0.3, // Lower for code examples
        MaxTokens:   500,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Tutor Response:")
    fmt.Println(response.Text)
}
```

### Provider Selection Based on Task

```go
func taskBasedProviderSelection(taskType string, complexity string) (*openai_compat.Provider, error) {
    switch taskType {
    case "chat", "qa":
        if complexity == "simple" {
            return openai_compat.Groq(
                openai_compat.WithModel("llama-3.1-8b-instant"),
            )
        }
        return openai_compat.Groq()
        
    case "creative", "writing":
        return openai_compat.Together(
            openai_compat.WithModel("meta-llama/Llama-3.3-70B-Instruct-Turbo"),
        )
        
    case "reasoning", "analysis":
        return openai_compat.XAI()
        
    case "code", "technical":
        return openai_compat.Fireworks(
            openai_compat.WithModel("accounts/fireworks/models/llama-v3p3-70b-instruct"),
        )
        
    case "high_volume":
        return openai_compat.Cerebras()
        
    default:
        return openai_compat.Groq() // Default to fast option
    }
}
```

## Streaming

### Basic Streaming

```go
func streamingComparison() {
    providers := map[string]func() (*openai_compat.Provider, error){
        "Groq":     openai_compat.Groq,
        "Together": openai_compat.Together,
    }
    
    prompt := "Write a short story about artificial intelligence."
    
    for name, createProvider := range providers {
        fmt.Printf("\n=== %s Streaming ===\n", name)
        
        provider, err := createProvider()
        if err != nil {
            fmt.Printf("Failed to create provider: %v\n", err)
            continue
        }
        
        stream, err := provider.StreamText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: prompt},
                    },
                },
            },
            Stream:    true,
            MaxTokens: 300,
        })
        
        if err != nil {
            fmt.Printf("Streaming error: %v\n", err)
            continue
        }
        
        var charCount int
        startTime := time.Now()
        firstToken := time.Time{}
        
        for event := range stream.Events() {
            switch event.Type {
            case core.EventTextDelta:
                if firstToken.IsZero() {
                    firstToken = time.Now()
                }
                fmt.Print(event.TextDelta)
                charCount += len(event.TextDelta)
                
            case core.EventFinish:
                totalTime := time.Since(startTime)
                ttft := firstToken.Sub(startTime)
                
                fmt.Printf("\n\n--- %s Performance ---\n", name)
                fmt.Printf("Time to first token: %v\n", ttft)
                fmt.Printf("Total time: %v\n", totalTime)
                fmt.Printf("Characters: %d\n", charCount)
                fmt.Printf("Chars/sec: %.1f\n", float64(charCount)/totalTime.Seconds())
                
            case core.EventError:
                fmt.Printf("\nStream error: %v\n", event.Err)
            }
        }
        
        stream.Close()
    }
}
```

### Real-time Chat Interface

```go
func chatInterface() {
    provider, err := openai_compat.Groq()
    if err != nil {
        log.Fatal(err)
    }
    
    scanner := bufio.NewScanner(os.Stdin)
    messages := []core.Message{
        {
            Role: core.System,
            Parts: []core.Part{
                core.Text{Text: "You are a helpful AI assistant. Be concise and friendly."},
            },
        },
    }
    
    fmt.Println("Chat Interface (type 'quit' to exit)")
    fmt.Println("=====================================")
    
    for {
        fmt.Print("\nYou: ")
        if !scanner.Scan() {
            break
        }
        
        userInput := strings.TrimSpace(scanner.Text())
        if userInput == "quit" {
            break
        }
        
        if userInput == "" {
            continue
        }
        
        // Add user message
        messages = append(messages, core.Message{
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: userInput},
            },
        })
        
        fmt.Print("AI: ")
        
        // Stream the response
        stream, err := provider.StreamText(context.Background(), core.Request{
            Messages:  messages,
            Stream:    true,
            MaxTokens: 200,
        })
        
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }
        
        var assistantResponse strings.Builder
        
        for event := range stream.Events() {
            switch event.Type {
            case core.EventTextDelta:
                fmt.Print(event.TextDelta)
                assistantResponse.WriteString(event.TextDelta)
                
            case core.EventError:
                fmt.Printf("\nError: %v\n", event.Err)
            }
        }
        
        stream.Close()
        
        // Add assistant response to conversation
        messages = append(messages, core.Message{
            Role: core.Assistant,
            Parts: []core.Part{
                core.Text{Text: assistantResponse.String()},
            },
        })
        
        fmt.Println() // New line after response
    }
    
    fmt.Println("Goodbye!")
}
```

## Tool Calling

### Multi-Provider Tool Support

```go
// Web search tool
type SearchInput struct {
    Query   string `json:"query" description:"Search query"`
    MaxResults int `json:"max_results,omitempty" description:"Maximum results (default: 5)"`
}

type SearchOutput struct {
    Results []SearchResult `json:"results"`
    Query   string         `json:"query"`
}

type SearchResult struct {
    Title   string `json:"title"`
    URL     string `json:"url"`
    Snippet string `json:"snippet"`
}

func createSearchTool() tools.Handle {
    return tools.New[SearchInput, SearchOutput](
        "web_search",
        "Search the web for information",
        func(ctx context.Context, input SearchInput, meta tools.Meta) (SearchOutput, error) {
            // Simulate web search
            maxResults := input.MaxResults
            if maxResults == 0 {
                maxResults = 5
            }
            
            results := make([]SearchResult, maxResults)
            for i := 0; i < maxResults; i++ {
                results[i] = SearchResult{
                    Title:   fmt.Sprintf("Result %d for '%s'", i+1, input.Query),
                    URL:     fmt.Sprintf("https://example.com/result%d", i+1),
                    Snippet: fmt.Sprintf("This is snippet %d related to %s...", i+1, input.Query),
                }
            }
            
            return SearchOutput{
                Results: results,
                Query:   input.Query,
            }, nil
        },
    )
}

func toolCallingWithMultipleProviders() {
    providers := map[string]func() (*openai_compat.Provider, error){
        "Groq":     openai_compat.Groq,
        "Together": openai_compat.Together,
        "xAI":      openai_compat.XAI,
    }
    
    searchTool := createSearchTool()
    
    for name, createProvider := range providers {
        fmt.Printf("\n=== %s Tool Calling ===\n", name)
        
        provider, err := createProvider()
        if err != nil {
            fmt.Printf("Failed to create provider: %v\n", err)
            continue
        }
        
        response, err := provider.GenerateText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: "Search for information about 'renewable energy benefits' and summarize the findings."},
                    },
                },
            },
            Tools: []tools.Handle{searchTool},
            ToolChoice: core.ToolAuto,
            MaxTokens: 500,
        })
        
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }
        
        fmt.Printf("Response: %s\n", response.Text)
        
        // Show tool execution details
        for i, step := range response.Steps {
            fmt.Printf("Step %d:\n", i+1)
            for _, call := range step.ToolCalls {
                fmt.Printf("  Tool: %s\n", call.Name)
                fmt.Printf("  Input: %s\n", string(call.Input))
            }
            for _, result := range step.ToolResults {
                fmt.Printf("  Result: %s\n", string(result.Result))
            }
        }
    }
}
```

### Provider-Specific Tool Limitations

```go
func handleProviderToolLimitations() {
    // Some providers have tool limitations
    providers := []struct {
        name     string
        provider func() (*openai_compat.Provider, error)
        supports struct {
            tools         bool
            parallelTools bool
            toolChoice    bool
        }
    }{
        {
            name:     "Groq",
            provider: openai_compat.Groq,
            supports: struct {
                tools         bool
                parallelTools bool
                toolChoice    bool
            }{true, true, true},
        },
        {
            name:     "Cerebras",
            provider: openai_compat.Cerebras,
            supports: struct {
                tools         bool
                parallelTools bool
                toolChoice    bool
            }{true, false, true}, // No parallel tools
        },
    }
    
    searchTool := createSearchTool()
    weatherTool := createWeatherTool()
    
    for _, p := range providers {
        fmt.Printf("\n=== %s Tool Support ===\n", p.name)
        
        provider, err := p.provider()
        if err != nil {
            fmt.Printf("Provider creation failed: %v\n", err)
            continue
        }
        
        if !p.supports.tools {
            fmt.Println("❌ Tools not supported")
            continue
        }
        
        tools := []tools.Handle{searchTool}
        if p.supports.parallelTools {
            tools = append(tools, weatherTool)
            fmt.Println("✅ Parallel tools supported")
        } else {
            fmt.Println("⚠️  Sequential tools only")
        }
        
        toolChoice := core.ToolAuto
        if !p.supports.toolChoice {
            toolChoice = core.ToolNone
            fmt.Println("⚠️  Tool choice not supported")
        }
        
        response, err := provider.GenerateText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: "Search for AI news and tell me the weather in San Francisco."},
                    },
                },
            },
            Tools:     tools,
            ToolChoice: toolChoice,
        })
        
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }
        
        fmt.Printf("Result: %s\n", response.Text)
    }
}
```

## Structured Outputs

### JSON Generation Across Providers

```go
type BlogPost struct {
    Title     string   `json:"title"`
    Author    string   `json:"author"`
    Content   string   `json:"content"`
    Tags      []string `json:"tags"`
    Category  string   `json:"category"`
    WordCount int      `json:"word_count"`
    SEOScore  float64  `json:"seo_score"`
}

func structuredOutputComparison() {
    providers := map[string]func() (*openai_compat.Provider, error){
        "Groq":      openai_compat.Groq,
        "Together":  openai_compat.Together,
        "Fireworks": openai_compat.Fireworks,
    }
    
    for name, createProvider := range providers {
        fmt.Printf("\n=== %s Structured Output ===\n", name)
        
        provider, err := createProvider()
        if err != nil {
            fmt.Printf("Provider creation failed: %v\n", err)
            continue
        }
        
        result, err := provider.GenerateObject(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.System,
                    Parts: []core.Part{
                        core.Text{Text: "Generate a comprehensive blog post structure based on the user's topic."},
                    },
                },
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: "Create a blog post about the benefits of remote work."},
                    },
                },
            },
            MaxTokens: 800,
        }, BlogPost{})
        
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }
        
        blogPost := result.Value.(map[string]interface{})
        
        fmt.Printf("Title: %s\n", blogPost["title"])
        fmt.Printf("Category: %s\n", blogPost["category"])
        fmt.Printf("Tags: %v\n", blogPost["tags"])
        fmt.Printf("Word Count: %.0f\n", blogPost["word_count"])
        fmt.Printf("SEO Score: %.2f\n", blogPost["seo_score"])
        fmt.Printf("Content Preview: %.100s...\n", blogPost["content"])
    }
}
```

### Provider-Specific JSON Handling

```go
func handleJSONQuirks() {
    providers := []struct {
        name              string
        provider          func() (*openai_compat.Provider, error)
        supportsStreaming bool
        strictSchema      bool
    }{
        {"Groq", openai_compat.Groq, true, true},
        {"Together", openai_compat.Together, true, false},
        {"Cerebras", openai_compat.Cerebras, false, true},
    }
    
    for _, p := range providers {
        fmt.Printf("\n=== %s JSON Capabilities ===\n", p.name)
        
        provider, err := p.provider()
        if err != nil {
            fmt.Printf("Provider creation failed: %v\n", err)
            continue
        }
        
        if p.supportsStreaming {
            fmt.Println("✅ JSON Streaming supported")
            // Test streaming structured output
            stream, err := provider.StreamObject(context.Background(), core.Request{
                Messages: []core.Message{
                    {
                        Role: core.User,
                        Parts: []core.Part{
                            core.Text{Text: "Generate a product review structure."},
                        },
                    },
                },
            }, BlogPost{})
            
            if err != nil {
                fmt.Printf("❌ Streaming failed: %v\n", err)
                continue
            }
            
            defer stream.Close()
            
            for event := range stream.Events() {
                if event.Type == core.EventTextDelta {
                    fmt.Print(event.TextDelta)
                } else if event.Type == core.EventFinish {
                    fmt.Println("\n✅ Streaming JSON completed")
                    break
                }
            }
        } else {
            fmt.Println("❌ JSON Streaming not supported")
        }
        
        if p.strictSchema {
            fmt.Println("✅ Strict schema validation supported")
        } else {
            fmt.Println("⚠️  Best-effort schema compliance")
        }
    }
}
```

## Provider-Specific Features

### Groq Optimization

```go
func optimizeForGroq() {
    provider, err := openai_compat.Groq(
        openai_compat.WithModel("llama-3.1-8b-instant"), // Fastest model
        openai_compat.WithMaxRetries(5),                  // Handle rate limits
        openai_compat.WithRetryDelay(500*time.Millisecond), // Fast retries
    )
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Groq is extremely fast, perfect for real-time applications
    requests := []string{
        "What's 2+2?",
        "Define AI",
        "Hello world in Python",
        "Explain photosynthesis briefly",
        "What is Go programming?",
    }
    
    fmt.Println("Groq Speed Test:")
    fmt.Println("================")
    
    start := time.Now()
    for i, prompt := range requests {
        reqStart := time.Now()
        response, err := provider.GenerateText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: prompt},
                    },
                },
            },
            MaxTokens: 50,
        })
        
        reqDuration := time.Since(reqStart)
        
        if err != nil {
            fmt.Printf("Request %d failed: %v\n", i+1, err)
            continue
        }
        
        fmt.Printf("Request %d: %v - %s\n", i+1, reqDuration, strings.Split(response.Text, "\n")[0])
    }
    
    totalTime := time.Since(start)
    fmt.Printf("\nTotal time for %d requests: %v\n", len(requests), totalTime)
    fmt.Printf("Average: %v per request\n", totalTime/time.Duration(len(requests)))
}
```

### Cerebras High Throughput

```go
func optimizeForCerebras() {
    provider, err := openai_compat.Cerebras(
        openai_compat.WithMaxRetries(3),
        openai_compat.WithRetryDelay(2*time.Second), // Longer delays for rate limits
    )
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Cerebras is optimized for high throughput simple tasks
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, 2) // Limit concurrent requests
    results := make(chan Result, 10)
    
    tasks := []string{
        "Summarize: Machine learning is...",
        "Translate to Spanish: Hello world",
        "Complete: The benefits of exercise are...",
        "Answer: What is 15 * 23?",
        "Classify sentiment: I love this product!",
    }
    
    fmt.Println("Cerebras High Throughput Test:")
    fmt.Println("==============================")
    
    for i, task := range tasks {
        wg.Add(1)
        go func(id int, prompt string) {
            defer wg.Done()
            
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            start := time.Now()
            response, err := provider.GenerateText(context.Background(), core.Request{
                Messages: []core.Message{
                    {
                        Role: core.User,
                        Parts: []core.Part{
                            core.Text{Text: prompt},
                        },
                    },
                },
                MaxTokens: 100,
            })
            
            duration := time.Since(start)
            
            results <- Result{
                ID:       id,
                Text:     response.Text,
                Error:    err,
                Duration: duration,
            }
        }(i, task)
    }
    
    go func() {
        wg.Wait()
        close(results)
    }()
    
    for result := range results {
        if result.Error != nil {
            fmt.Printf("Task %d failed: %v\n", result.ID+1, result.Error)
        } else {
            fmt.Printf("Task %d (%v): %s\n", result.ID+1, result.Duration, 
                strings.Split(result.Text, "\n")[0])
        }
    }
}

type Result struct {
    ID       int
    Text     string
    Error    error
    Duration time.Duration
}
```

### Together Model Exploration

```go
func exploreTogetherModels() {
    models := []string{
        "meta-llama/Llama-3.3-70B-Instruct-Turbo",
        "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
        "mistralai/Mixtral-8x7B-Instruct-v0.1",
        "NousResearch/Nous-Hermes-2-Mixtral-8x7B-DPO",
    }
    
    prompt := "Explain the concept of machine learning in one paragraph."
    
    for _, model := range models {
        fmt.Printf("\n--- %s ---\n", model)
        
        provider, err := openai_compat.Together(
            openai_compat.WithModel(model),
        )
        
        if err != nil {
            fmt.Printf("Failed to create provider: %v\n", err)
            continue
        }
        
        start := time.Now()
        response, err := provider.GenerateText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: prompt},
                    },
                },
            },
            MaxTokens: 150,
            Temperature: 0.7,
        })
        
        duration := time.Since(start)
        
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }
        
        fmt.Printf("Response (%v): %s\n", duration, response.Text)
        fmt.Printf("Tokens: %d\n", response.Usage.TotalTokens)
    }
}
```

## Capability Detection

### Automatic Provider Probing

```go
type ProviderCapabilities struct {
    Name              string
    SupportsStreaming bool
    SupportsTools     bool
    SupportsJSONMode  bool
    MaxContextLength  int
    AvgLatency        time.Duration
}

func probeProviderCapabilities(name string, createProvider func() (*openai_compat.Provider, error)) ProviderCapabilities {
    caps := ProviderCapabilities{Name: name}
    
    provider, err := createProvider()
    if err != nil {
        fmt.Printf("Failed to create %s provider: %v\n", name, err)
        return caps
    }
    
    ctx := context.Background()
    
    // Test basic generation and measure latency
    fmt.Printf("Probing %s capabilities...\n", name)
    
    start := time.Now()
    _, err = provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Say 'OK'"},
                },
            },
        },
        MaxTokens: 5,
    })
    
    if err != nil {
        fmt.Printf("❌ Basic generation failed: %v\n", err)
        return caps
    }
    
    caps.AvgLatency = time.Since(start)
    fmt.Printf("✅ Basic generation works (latency: %v)\n", caps.AvgLatency)
    
    // Test streaming
    stream, err := provider.StreamText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Count to 3"},
                },
            },
        },
        Stream: true,
    })
    
    if err == nil {
        defer stream.Close()
        hasEvents := false
        for event := range stream.Events() {
            if event.Type == core.EventTextDelta {
                hasEvents = true
                break
            }
        }
        caps.SupportsStreaming = hasEvents
    }
    
    if caps.SupportsStreaming {
        fmt.Println("✅ Streaming supported")
    } else {
        fmt.Println("❌ Streaming not supported")
    }
    
    // Test JSON mode
    _, err = provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Respond with JSON: {\"status\": \"ok\"}"},
                },
            },
        },
        MaxTokens: 20,
        ProviderOptions: map[string]interface{}{
            "response_format": map[string]string{
                "type": "json_object",
            },
        },
    })
    
    caps.SupportsJSONMode = (err == nil)
    if caps.SupportsJSONMode {
        fmt.Println("✅ JSON mode supported")
    } else {
        fmt.Println("❌ JSON mode not supported")
    }
    
    // Test tools
    simpletool := tools.New[struct{Input string `json:"input"`}, struct{Output string `json:"output"`}](
        "echo",
        "Echo the input",
        func(ctx context.Context, input struct{Input string `json:"input"`}, meta tools.Meta) (struct{Output string `json:"output"`}, error) {
            return struct{Output string `json:"output"`}{Output: input.Input}, nil
        },
    )
    
    _, err = provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Use the echo tool to say 'hello'"},
                },
            },
        },
        Tools: []tools.Handle{simpleTools},
        MaxTokens: 50,
    })
    
    caps.SupportsTools = (err == nil)
    if caps.SupportsTools {
        fmt.Println("✅ Tools supported")
    } else {
        fmt.Println("❌ Tools not supported")
    }
    
    return caps
}

func runCapabilityReport() {
    providers := map[string]func() (*openai_compat.Provider, error){
        "Groq":      openai_compat.Groq,
        "xAI":       openai_compat.XAI,
        "Together":  openai_compat.Together,
        "Fireworks": openai_compat.Fireworks,
        "Cerebras":  openai_compat.Cerebras,
    }
    
    fmt.Println("Provider Capability Report")
    fmt.Println("==========================")
    
    var results []ProviderCapabilities
    
    for name, createProvider := range providers {
        caps := probeProviderCapabilities(name, createProvider)
        results = append(results, caps)
        fmt.Println()
    }
    
    // Summary table
    fmt.Printf("%-12s %-10s %-8s %-6s %-5s %-10s\n", 
        "Provider", "Latency", "Stream", "Tools", "JSON", "Status")
    fmt.Println(strings.Repeat("-", 60))
    
    for _, caps := range results {
        streaming := "❌"
        if caps.SupportsStreaming {
            streaming = "✅"
        }
        
        tools := "❌"
        if caps.SupportsTools {
            tools = "✅"
        }
        
        json := "❌"
        if caps.SupportsJSONMode {
            json = "✅"
        }
        
        fmt.Printf("%-12s %-10v %-8s %-6s %-5s\n", 
            caps.Name, caps.AvgLatency, streaming, tools, json)
    }
}
```

## Advanced Configuration

### Load Balancing Across Providers

```go
type LoadBalancer struct {
    providers []ProviderConfig
    current   int
    mu        sync.Mutex
}

type ProviderConfig struct {
    Name     string
    Provider *openai_compat.Provider
    Weight   int
    Latency  time.Duration
}

func createLoadBalancer() *LoadBalancer {
    providers := []ProviderConfig{
        {
            Name:   "Groq",
            Weight: 3, // Higher weight for faster provider
        },
        {
            Name:   "Together",
            Weight: 2,
        },
        {
            Name:   "Fireworks",
            Weight: 2,
        },
    }
    
    // Initialize providers
    for i := range providers {
        switch providers[i].Name {
        case "Groq":
            if p, err := openai_compat.Groq(); err == nil {
                providers[i].Provider = p
            }
        case "Together":
            if p, err := openai_compat.Together(); err == nil {
                providers[i].Provider = p
            }
        case "Fireworks":
            if p, err := openai_compat.Fireworks(); err == nil {
                providers[i].Provider = p
            }
        }
    }
    
    return &LoadBalancer{providers: providers}
}

func (lb *LoadBalancer) GetNextProvider() *ProviderConfig {
    lb.mu.Lock()
    defer lb.mu.Unlock()
    
    // Simple round-robin with weights
    totalWeight := 0
    for _, p := range lb.providers {
        if p.Provider != nil {
            totalWeight += p.Weight
        }
    }
    
    if totalWeight == 0 {
        return nil
    }
    
    target := lb.current % totalWeight
    current := 0
    
    for i, p := range lb.providers {
        if p.Provider != nil {
            current += p.Weight
            if current > target {
                lb.current++
                return &lb.providers[i]
            }
        }
    }
    
    return nil
}

func (lb *LoadBalancer) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
    maxAttempts := len(lb.providers)
    
    for attempt := 0; attempt < maxAttempts; attempt++ {
        providerConfig := lb.GetNextProvider()
        if providerConfig == nil {
            return nil, fmt.Errorf("no available providers")
        }
        
        start := time.Now()
        result, err := providerConfig.Provider.GenerateText(ctx, req)
        duration := time.Since(start)
        
        // Update latency stats
        providerConfig.Latency = duration
        
        if err == nil {
            return result, nil
        }
        
        fmt.Printf("Provider %s failed (attempt %d): %v\n", 
            providerConfig.Name, attempt+1, err)
    }
    
    return nil, fmt.Errorf("all providers failed")
}

func demonstrateLoadBalancing() {
    lb := createLoadBalancer()
    
    requests := []string{
        "What is AI?",
        "Explain machine learning",
        "Write hello world in Go",
        "Define recursion",
        "What are APIs?",
    }
    
    fmt.Println("Load Balancing Demo:")
    fmt.Println("==================")
    
    for i, prompt := range requests {
        result, err := lb.GenerateText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: prompt},
                    },
                },
            },
            MaxTokens: 100,
        })
        
        if err != nil {
            fmt.Printf("Request %d failed: %v\n", i+1, err)
            continue
        }
        
        fmt.Printf("Request %d: %s\n", i+1, 
            strings.Split(result.Text, "\n")[0])
    }
    
    // Show latency stats
    fmt.Println("\nLatency Summary:")
    for _, p := range lb.providers {
        if p.Provider != nil {
            fmt.Printf("%s: %v\n", p.Name, p.Latency)
        }
    }
}
```

### Fallback Chain Configuration

```go
type FallbackChain struct {
    providers []FallbackProvider
}

type FallbackProvider struct {
    Name     string
    Provider *openai_compat.Provider
    Priority int
    MaxRetries int
    Conditions func(error) bool // Whether to try this provider for the error
}

func createFallbackChain() *FallbackChain {
    return &FallbackChain{
        providers: []FallbackProvider{
            {
                Name:     "Groq (Primary)",
                Priority: 1,
                MaxRetries: 2,
                Conditions: func(err error) bool {
                    // Try Groq first for all requests
                    return true
                },
            },
            {
                Name:     "Together (Backup)",
                Priority: 2,
                MaxRetries: 2,
                Conditions: func(err error) bool {
                    // Use Together if Groq fails
                    return strings.Contains(err.Error(), "rate limit") ||
                           strings.Contains(err.Error(), "unavailable")
                },
            },
            {
                Name:     "Fireworks (Final)",
                Priority: 3,
                MaxRetries: 1,
                Conditions: func(err error) bool {
                    // Use Fireworks as last resort
                    return true
                },
            },
        },
    }
}

func (fc *FallbackChain) Initialize() error {
    for i := range fc.providers {
        switch fc.providers[i].Name {
        case "Groq (Primary)":
            if p, err := openai_compat.Groq(); err == nil {
                fc.providers[i].Provider = p
            } else {
                return fmt.Errorf("failed to initialize Groq: %w", err)
            }
        case "Together (Backup)":
            if p, err := openai_compat.Together(); err == nil {
                fc.providers[i].Provider = p
            } else {
                return fmt.Errorf("failed to initialize Together: %w", err)
            }
        case "Fireworks (Final)":
            if p, err := openai_compat.Fireworks(); err == nil {
                fc.providers[i].Provider = p
            } else {
                return fmt.Errorf("failed to initialize Fireworks: %w", err)
            }
        }
    }
    return nil
}

func (fc *FallbackChain) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
    var lastErr error
    
    for _, fp := range fc.providers {
        if fp.Provider == nil {
            continue
        }
        
        // Skip if conditions don't match (except for first provider)
        if fp.Priority > 1 && lastErr != nil && !fp.Conditions(lastErr) {
            continue
        }
        
        for attempt := 0; attempt < fp.MaxRetries; attempt++ {
            result, err := fp.Provider.GenerateText(ctx, req)
            
            if err == nil {
                if fp.Priority > 1 {
                    fmt.Printf("✅ Fallback successful with %s\n", fp.Name)
                }
                return result, nil
            }
            
            lastErr = err
            fmt.Printf("❌ %s failed (attempt %d/%d): %v\n", 
                fp.Name, attempt+1, fp.MaxRetries, err)
            
            if attempt < fp.MaxRetries-1 {
                time.Sleep(time.Duration(attempt+1) * time.Second) // Exponential backoff
            }
        }
    }
    
    return nil, fmt.Errorf("all fallback providers failed, last error: %w", lastErr)
}

func demonstrateFallbacks() {
    fc := createFallbackChain()
    if err := fc.Initialize(); err != nil {
        log.Fatal(err)
    }
    
    // Simulate various scenarios
    testCases := []struct {
        name   string
        prompt string
    }{
        {"Normal Request", "What is Go programming?"},
        {"Complex Request", "Write a detailed explanation of distributed systems architecture with code examples."},
        {"Simple Request", "Hello"},
    }
    
    fmt.Println("Fallback Chain Demo:")
    fmt.Println("===================")
    
    for _, tc := range testCases {
        fmt.Printf("\n--- %s ---\n", tc.name)
        
        result, err := fc.GenerateText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{
                        core.Text{Text: tc.prompt},
                    },
                },
            },
            MaxTokens: 200,
        })
        
        if err != nil {
            fmt.Printf("❌ All providers failed: %v\n", err)
            continue
        }
        
        fmt.Printf("✅ Success: %s\n", 
            strings.Split(result.Text, "\n")[0])
    }
}
```

## Error Handling

### Comprehensive Error Mapping

```go
func handleProviderSpecificErrors(provider *openai_compat.Provider, providerName string) {
    ctx := context.Background()
    
    // Test various error scenarios
    errorTests := []struct {
        name        string
        req         core.Request
        expectedErr string
    }{
        {
            "Invalid Model",
            core.Request{
                Model: "nonexistent-model-12345",
                Messages: []core.Message{
                    {
                        Role: core.User,
                        Parts: []core.Part{core.Text{Text: "Hello"}},
                    },
                },
            },
            "model not found",
        },
        {
            "Context Too Long",
            core.Request{
                Messages: []core.Message{
                    {
                        Role: core.User,
                        Parts: []core.Part{
                            core.Text{Text: strings.Repeat("very long context ", 10000)},
                        },
                    },
                },
            },
            "context length",
        },
        {
            "Invalid API Key",
            core.Request{
                Messages: []core.Message{
                    {
                        Role: core.User,
                        Parts: []core.Part{core.Text{Text: "Hello"}},
                    },
                },
                ProviderOptions: map[string]interface{}{
                    "api_key": "invalid-key-12345",
                },
            },
            "unauthorized",
        },
    }
    
    fmt.Printf("Error Handling Test for %s:\n", providerName)
    fmt.Println("=====================================")
    
    for _, test := range errorTests {
        fmt.Printf("\nTesting: %s\n", test.name)
        
        _, err := provider.GenerateText(ctx, test.req)
        
        if err != nil {
            // Check for specific error types
            switch {
            case core.IsAuth(err):
                fmt.Printf("✅ Authentication error detected: %v\n", err)
                
            case core.IsRateLimited(err):
                retryAfter := core.GetRetryAfter(err)
                fmt.Printf("✅ Rate limit detected, retry after: %v\n", retryAfter)
                
            case core.IsContextSizeExceeded(err):
                fmt.Printf("✅ Context size exceeded: %v\n", err)
                
            case core.IsBadRequest(err):
                fmt.Printf("✅ Bad request error: %v\n", err)
                
            case core.IsOverloaded(err):
                fmt.Printf("✅ Provider overloaded: %v\n", err)
                
            case core.IsQuotaExceeded(err):
                fmt.Printf("✅ Quota exceeded: %v\n", err)
                
            case core.IsNetwork(err):
                fmt.Printf("✅ Network error: %v\n", err)
                
            default:
                fmt.Printf("❓ Other error: %v\n", err)
            }
        } else {
            fmt.Printf("❓ Expected error but got success\n")
        }
    }
}
```

### Provider-Specific Error Strategies

```go
func createErrorHandlingStrategy(providerName string) func(error) (bool, time.Duration) {
    switch providerName {
    case "groq":
        return func(err error) (shouldRetry bool, delay time.Duration) {
            if core.IsRateLimited(err) {
                // Groq has aggressive rate limits
                return true, 5 * time.Second
            }
            if core.IsOverloaded(err) {
                // Retry overloaded quickly
                return true, 1 * time.Second
            }
            return false, 0
        }
        
    case "cerebras":
        return func(err error) (shouldRetry bool, delay time.Duration) {
            if core.IsRateLimited(err) {
                // Cerebras has strict rate limits
                return true, 10 * time.Second
            }
            return false, 0
        }
        
    case "together", "fireworks":
        return func(err error) (shouldRetry bool, delay time.Duration) {
            if core.IsRateLimited(err) {
                return true, 2 * time.Second
            }
            if core.IsOverloaded(err) {
                return true, 3 * time.Second
            }
            return false, 0
        }
        
    default:
        return func(err error) (shouldRetry bool, delay time.Duration) {
            if core.IsTransient(err) {
                return true, 2 * time.Second
            }
            return false, 0
        }
    }
}

func retryWithStrategy(provider *openai_compat.Provider, providerName string, req core.Request) (*core.TextResult, error) {
    strategy := createErrorHandlingStrategy(providerName)
    maxAttempts := 3
    
    for attempt := 1; attempt <= maxAttempts; attempt++ {
        result, err := provider.GenerateText(context.Background(), req)
        
        if err == nil {
            if attempt > 1 {
                fmt.Printf("✅ Success on attempt %d with %s\n", attempt, providerName)
            }
            return result, nil
        }
        
        fmt.Printf("❌ Attempt %d failed with %s: %v\n", attempt, providerName, err)
        
        if attempt < maxAttempts {
            shouldRetry, delay := strategy(err)
            if shouldRetry {
                fmt.Printf("⏳ Retrying in %v...\n", delay)
                time.Sleep(delay)
                continue
            }
        }
        
        return nil, err
    }
    
    return nil, fmt.Errorf("max attempts exceeded")
}
```

## Best Practices

### 1. Provider Selection Guidelines

```go
func selectOptimalProvider(requirements Requirements) (*openai_compat.Provider, error) {
    switch {
    case requirements.Speed == "critical" && requirements.Quality == "basic":
        // Ultra-fast for simple tasks
        return openai_compat.Groq(
            openai_compat.WithModel("llama-3.1-8b-instant"),
        )
        
    case requirements.Speed == "fast" && requirements.Quality == "high":
        // Balanced speed and quality
        return openai_compat.Groq()
        
    case requirements.Quality == "premium" && requirements.Budget == "high":
        // Best quality available
        return openai_compat.XAI()
        
    case requirements.ModelVariety && requirements.Budget == "low":
        // Many models, cost-effective
        return openai_compat.Together()
        
    case requirements.Reliability == "enterprise":
        // Enterprise-grade reliability
        return openai_compat.Anyscale()
        
    case requirements.CustomModels:
        // Custom model deployment
        return openai_compat.Baseten(requirements.CustomURL)
        
    default:
        // Safe default
        return openai_compat.Groq()
    }
}

type Requirements struct {
    Speed         string // "critical", "fast", "normal"
    Quality       string // "basic", "high", "premium"
    Budget        string // "low", "medium", "high"
    ModelVariety  bool
    Reliability   string
    CustomModels  bool
    CustomURL     string
}
```

### 2. Performance Monitoring

```go
type ProviderMetrics struct {
    Name         string
    RequestCount int
    SuccessCount int
    TotalLatency time.Duration
    ErrorTypes   map[string]int
    mu           sync.RWMutex
}

func (pm *ProviderMetrics) RecordSuccess(latency time.Duration) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    pm.RequestCount++
    pm.SuccessCount++
    pm.TotalLatency += latency
}

func (pm *ProviderMetrics) RecordError(err error) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    pm.RequestCount++
    
    if pm.ErrorTypes == nil {
        pm.ErrorTypes = make(map[string]int)
    }
    
    errorType := "unknown"
    switch {
    case core.IsRateLimited(err):
        errorType = "rate_limited"
    case core.IsAuth(err):
        errorType = "auth"
    case core.IsOverloaded(err):
        errorType = "overloaded"
    case core.IsNetwork(err):
        errorType = "network"
    }
    
    pm.ErrorTypes[errorType]++
}

func (pm *ProviderMetrics) GetStats() map[string]interface{} {
    pm.mu.RLock()
    defer pm.mu.RUnlock()
    
    stats := map[string]interface{}{
        "name":         pm.Name,
        "requests":     pm.RequestCount,
        "success_rate": float64(pm.SuccessCount) / float64(pm.RequestCount),
        "avg_latency":  pm.TotalLatency / time.Duration(pm.SuccessCount),
        "errors":       pm.ErrorTypes,
    }
    
    return stats
}

func monitorProviderPerformance() {
    providers := map[string]func() (*openai_compat.Provider, error){
        "Groq":      openai_compat.Groq,
        "Together":  openai_compat.Together,
        "Fireworks": openai_compat.Fireworks,
    }
    
    metrics := make(map[string]*ProviderMetrics)
    
    for name := range providers {
        metrics[name] = &ProviderMetrics{Name: name}
    }
    
    // Run test workload
    testPrompts := []string{
        "What is AI?",
        "Explain quantum computing",
        "Write hello world in Go",
        "Define machine learning",
        "What are microservices?",
    }
    
    fmt.Println("Running performance monitoring...")
    
    for name, createProvider := range providers {
        fmt.Printf("Testing %s...\n", name)
        
        provider, err := createProvider()
        if err != nil {
            fmt.Printf("Failed to create %s: %v\n", name, err)
            continue
        }
        
        for _, prompt := range testPrompts {
            start := time.Now()
            _, err := provider.GenerateText(context.Background(), core.Request{
                Messages: []core.Message{
                    {
                        Role: core.User,
                        Parts: []core.Part{
                            core.Text{Text: prompt},
                        },
                    },
                },
                MaxTokens: 100,
            })
            
            latency := time.Since(start)
            
            if err != nil {
                metrics[name].RecordError(err)
            } else {
                metrics[name].RecordSuccess(latency)
            }
        }
    }
    
    // Report results
    fmt.Println("\nPerformance Report:")
    fmt.Println("==================")
    
    for name, metric := range metrics {
        stats := metric.GetStats()
        fmt.Printf("\n%s:\n", name)
        fmt.Printf("  Requests: %d\n", stats["requests"])
        fmt.Printf("  Success Rate: %.2f%%\n", stats["success_rate"].(float64)*100)
        fmt.Printf("  Avg Latency: %v\n", stats["avg_latency"])
        if errors, ok := stats["errors"].(map[string]int); ok && len(errors) > 0 {
            fmt.Printf("  Errors: %v\n", errors)
        }
    }
}
```

### 3. Cost Optimization

```go
type CostOptimizer struct {
    providers map[string]ProviderCost
}

type ProviderCost struct {
    Name           string
    InputTokens    float64 // Cost per 1K input tokens
    OutputTokens   float64 // Cost per 1K output tokens
    RequestOverhead float64 // Fixed cost per request
}

func createCostOptimizer() *CostOptimizer {
    return &CostOptimizer{
        providers: map[string]ProviderCost{
            "groq": {
                Name:           "Groq",
                InputTokens:    0.0005, // $0.0005 per 1K tokens (example)
                OutputTokens:   0.0008,
                RequestOverhead: 0.0001,
            },
            "together": {
                Name:           "Together",
                InputTokens:    0.0006,
                OutputTokens:   0.0010,
                RequestOverhead: 0.0001,
            },
            "fireworks": {
                Name:           "Fireworks",
                InputTokens:    0.0004,
                OutputTokens:   0.0007,
                RequestOverhead: 0.0001,
            },
            "xai": {
                Name:           "xAI",
                InputTokens:    0.0020, // Premium pricing
                OutputTokens:   0.0040,
                RequestOverhead: 0.0005,
            },
        },
    }
}

func (co *CostOptimizer) EstimateCost(providerName string, inputTokens, outputTokens int) float64 {
    provider, exists := co.providers[providerName]
    if !exists {
        return 0
    }
    
    inputCost := float64(inputTokens) / 1000 * provider.InputTokens
    outputCost := float64(outputTokens) / 1000 * provider.OutputTokens
    overhead := provider.RequestOverhead
    
    return inputCost + outputCost + overhead
}

func (co *CostOptimizer) FindMostCostEffective(inputTokens, outputTokens int) (string, float64) {
    minCost := float64(math.Inf(1))
    bestProvider := ""
    
    for name := range co.providers {
        cost := co.EstimateCost(name, inputTokens, outputTokens)
        if cost < minCost {
            minCost = cost
            bestProvider = name
        }
    }
    
    return bestProvider, minCost
}

func demonstrateCostOptimization() {
    optimizer := createCostOptimizer()
    
    scenarios := []struct {
        name         string
        inputTokens  int
        outputTokens int
    }{
        {"Short Response", 100, 50},
        {"Medium Response", 500, 200},
        {"Long Response", 1000, 800},
        {"Very Long Response", 2000, 1500},
    }
    
    fmt.Println("Cost Optimization Analysis:")
    fmt.Println("==========================")
    
    for _, scenario := range scenarios {
        fmt.Printf("\n%s (%d input, %d output tokens):\n", 
            scenario.name, scenario.inputTokens, scenario.outputTokens)
        
        costs := make(map[string]float64)
        for providerName := range optimizer.providers {
            cost := optimizer.EstimateCost(providerName, scenario.inputTokens, scenario.outputTokens)
            costs[providerName] = cost
        }
        
        // Sort by cost
        type providerCostPair struct {
            name string
            cost float64
        }
        
        var sorted []providerCostPair
        for name, cost := range costs {
            sorted = append(sorted, providerCostPair{name, cost})
        }
        
        // Simple sort
        for i := 0; i < len(sorted); i++ {
            for j := i + 1; j < len(sorted); j++ {
                if sorted[i].cost > sorted[j].cost {
                    sorted[i], sorted[j] = sorted[j], sorted[i]
                }
            }
        }
        
        for i, pair := range sorted {
            symbol := ""
            if i == 0 {
                symbol = "🏆 " // Best value
            }
            fmt.Printf("  %s%s: $%.6f\n", symbol, pair.name, pair.cost)
        }
        
        bestProvider, bestCost := optimizer.FindMostCostEffective(scenario.inputTokens, scenario.outputTokens)
        fmt.Printf("  → Best choice: %s ($%.6f)\n", bestProvider, bestCost)
    }
}
```

## Performance Comparison

### Speed Benchmarking

```go
func benchmarkProviders() {
    providers := map[string]func() (*openai_compat.Provider, error){
        "Groq":      openai_compat.Groq,
        "Cerebras":  openai_compat.Cerebras,
        "Together":  openai_compat.Together,
        "Fireworks": openai_compat.Fireworks,
    }
    
    benchmarks := []struct {
        name       string
        prompt     string
        maxTokens  int
        complexity string
    }{
        {"Simple QA", "What is 2+2?", 10, "trivial"},
        {"Definition", "Define machine learning", 100, "simple"},
        {"Explanation", "Explain how neural networks work", 300, "medium"},
        {"Code Generation", "Write a binary search algorithm in Go with comments", 500, "complex"},
    }
    
    fmt.Println("Provider Speed Benchmark:")
    fmt.Println("========================")
    
    results := make(map[string]map[string]BenchmarkResult)
    
    for providerName, createProvider := range providers {
        results[providerName] = make(map[string]BenchmarkResult)
        
        fmt.Printf("\nTesting %s:\n", providerName)
        
        provider, err := createProvider()
        if err != nil {
            fmt.Printf("  ❌ Failed to create provider: %v\n", err)
            continue
        }
        
        for _, benchmark := range benchmarks {
            fmt.Printf("  %s... ", benchmark.name)
            
            // Warm up
            provider.GenerateText(context.Background(), core.Request{
                Messages: []core.Message{
                    {
                        Role: core.User,
                        Parts: []core.Part{core.Text{Text: "hi"}},
                    },
                },
                MaxTokens: 1,
            })
            
            // Actual benchmark
            start := time.Now()
            response, err := provider.GenerateText(context.Background(), core.Request{
                Messages: []core.Message{
                    {
                        Role: core.User,
                        Parts: []core.Part{
                            core.Text{Text: benchmark.prompt},
                        },
                    },
                },
                MaxTokens: benchmark.maxTokens,
            })
            
            duration := time.Since(start)
            
            if err != nil {
                fmt.Printf("❌ Error: %v\n", err)
                results[providerName][benchmark.name] = BenchmarkResult{
                    Duration: duration,
                    Error:    err,
                }
                continue
            }
            
            tokensPerSec := float64(response.Usage.OutputTokens) / duration.Seconds()
            
            fmt.Printf("✅ %v (%.1f tok/s)\n", duration, tokensPerSec)
            
            results[providerName][benchmark.name] = BenchmarkResult{
                Duration:     duration,
                TokensPerSec: tokensPerSec,
                Tokens:       response.Usage.OutputTokens,
                Response:     response.Text,
            }
        }
    }
    
    // Summary table
    fmt.Println("\nSummary Table:")
    fmt.Println("==============")
    
    fmt.Printf("%-12s", "Provider")
    for _, benchmark := range benchmarks {
        fmt.Printf("%-15s", benchmark.name)
    }
    fmt.Println()
    
    fmt.Println(strings.Repeat("-", 12+15*len(benchmarks)))
    
    for providerName := range providers {
        fmt.Printf("%-12s", providerName)
        
        for _, benchmark := range benchmarks {
            if result, ok := results[providerName][benchmark.name]; ok && result.Error == nil {
                fmt.Printf("%-15s", fmt.Sprintf("%.1fs", result.Duration.Seconds()))
            } else {
                fmt.Printf("%-15s", "FAILED")
            }
        }
        fmt.Println()
    }
}

type BenchmarkResult struct {
    Duration     time.Duration
    TokensPerSec float64
    Tokens       int
    Response     string
    Error        error
}
```

## Migration Guide

### From OpenAI to Compatible Providers

```go
func migrateFromOpenAI() {
    // Original OpenAI code
    openaiProvider := openai.New(
        openai.WithAPIKey("openai-api-key"),
        openai.WithModel("gpt-4"),
    )
    
    // Migrate to Groq (drop-in replacement)
    groqProvider, _ := openai_compat.Groq(
        openai_compat.WithModel("llama-3.3-70b-versatile"),
    )
    
    // Same request works with both
    request := core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Explain the benefits of containerization."},
                },
            },
        },
        MaxTokens: 500,
    }
    
    // Compare results
    fmt.Println("OpenAI Response:")
    openaiResult, _ := openaiProvider.GenerateText(context.Background(), request)
    fmt.Println(openaiResult.Text)
    
    fmt.Println("\nGroq Response:")
    groqResult, _ := groqProvider.GenerateText(context.Background(), request)
    fmt.Println(groqResult.Text)
    
    // Migration benefits
    fmt.Println("\nMigration Benefits:")
    fmt.Println("- Faster inference with Groq")
    fmt.Println("- Lower costs")
    fmt.Println("- Same API interface")
    fmt.Println("- Multiple provider options")
}
```

### Provider Switching Strategy

```go
func createProviderSwitcher() {
    // Create a switcher that can dynamically change providers
    switcher := &ProviderSwitcher{
        providers: map[string]*openai_compat.Provider{},
        current:   "groq",
    }
    
    // Initialize providers
    if groq, err := openai_compat.Groq(); err == nil {
        switcher.providers["groq"] = groq
    }
    
    if together, err := openai_compat.Together(); err == nil {
        switcher.providers["together"] = together
    }
    
    if xai, err := openai_compat.XAI(); err == nil {
        switcher.providers["xai"] = xai
    }
    
    // Use the switcher
    request := core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a poem about programming."},
                },
            },
        },
    }
    
    // Try with different providers
    fmt.Println("Trying different providers:")
    
    for _, providerName := range []string{"groq", "together", "xai"} {
        switcher.SwitchTo(providerName)
        
        result, err := switcher.GenerateText(context.Background(), request)
        if err != nil {
            fmt.Printf("%s: ❌ %v\n", providerName, err)
            continue
        }
        
        fmt.Printf("%s: ✅ %s\n", providerName, 
            strings.Split(result.Text, "\n")[0])
    }
}

type ProviderSwitcher struct {
    providers map[string]*openai_compat.Provider
    current   string
}

func (ps *ProviderSwitcher) SwitchTo(name string) {
    ps.current = name
}

func (ps *ProviderSwitcher) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
    provider, ok := ps.providers[ps.current]
    if !ok {
        return nil, fmt.Errorf("provider %s not available", ps.current)
    }
    
    return provider.GenerateText(ctx, req)
}
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Authentication Failures

**Problem**: API key errors or unauthorized responses

**Solution**:
```go
func debugAuthentication(providerName string) {
    // Check environment variables
    var apiKeyEnv string
    switch providerName {
    case "groq":
        apiKeyEnv = "GROQ_API_KEY"
    case "xai":
        apiKeyEnv = "XAI_API_KEY"
    case "together":
        apiKeyEnv = "TOGETHER_API_KEY"
    case "fireworks":
        apiKeyEnv = "FIREWORKS_API_KEY"
    case "cerebras":
        apiKeyEnv = "CEREBRAS_API_KEY"
    }
    
    apiKey := os.Getenv(apiKeyEnv)
    if apiKey == "" {
        fmt.Printf("❌ %s environment variable not set\n", apiKeyEnv)
        return
    }
    
    if len(apiKey) < 10 {
        fmt.Printf("❌ %s appears to be invalid (too short)\n", apiKeyEnv)
        return
    }
    
    fmt.Printf("✅ %s is set (length: %d)\n", apiKeyEnv, len(apiKey))
    
    // Test authentication
    var provider *openai_compat.Provider
    var err error
    
    switch providerName {
    case "groq":
        provider, err = openai_compat.Groq()
    case "xai":
        provider, err = openai_compat.XAI()
    case "together":
        provider, err = openai_compat.Together()
    case "fireworks":
        provider, err = openai_compat.Fireworks()
    case "cerebras":
        provider, err = openai_compat.Cerebras()
    }
    
    if err != nil {
        fmt.Printf("❌ Failed to create provider: %v\n", err)
        return
    }
    
    // Test with simple request
    _, err = provider.GenerateText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{core.Text{Text: "Hi"}},
            },
        },
        MaxTokens: 1,
    })
    
    if err != nil {
        if core.IsAuth(err) {
            fmt.Printf("❌ Authentication failed: %v\n", err)
            fmt.Printf("💡 Check your API key for %s\n", providerName)
        } else {
            fmt.Printf("✅ Authentication works, other error: %v\n", err)
        }
    } else {
        fmt.Printf("✅ Authentication successful for %s\n", providerName)
    }
}
```

#### 2. Model Compatibility Issues

**Problem**: Model not found or not supported

**Solution**:
```go
func debugModelSupport() {
    providers := map[string]func() (*openai_compat.Provider, error){
        "Groq":      openai_compat.Groq,
        "Together":  openai_compat.Together,
        "Fireworks": openai_compat.Fireworks,
    }
    
    testModel := "llama-3.1-70b-versatile"
    
    fmt.Printf("Testing model '%s' across providers:\n", testModel)
    fmt.Println("=========================================")
    
    for name, createProvider := range providers {
        fmt.Printf("\n%s:\n", name)
        
        provider, err := createProvider()
        if err != nil {
            fmt.Printf("  ❌ Provider creation failed: %v\n", err)
            continue
        }
        
        // Test with the specific model
        _, err = provider.GenerateText(context.Background(), core.Request{
            Model: testModel,
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{core.Text{Text: "Hello"}},
                },
            },
            MaxTokens: 5,
        })
        
        if err != nil {
            if strings.Contains(err.Error(), "model") {
                fmt.Printf("  ❌ Model not supported: %v\n", err)
                
                // Try with default model
                _, err2 := provider.GenerateText(context.Background(), core.Request{
                    Messages: []core.Message{
                        {
                            Role: core.User,
                            Parts: []core.Part{core.Text{Text: "Hello"}},
                        },
                    },
                    MaxTokens: 5,
                })
                
                if err2 == nil {
                    fmt.Printf("  💡 Try using the provider's default model\n")
                }
            } else {
                fmt.Printf("  ❓ Other error: %v\n", err)
            }
        } else {
            fmt.Printf("  ✅ Model supported\n")
        }
    }
}
```

#### 3. Rate Limiting Issues

**Problem**: Frequent rate limit errors

**Solution**:
```go
func handleRateLimits() {
    // Create provider with aggressive retry settings
    provider, err := openai_compat.Groq(
        openai_compat.WithMaxRetries(5),
        openai_compat.WithRetryDelay(1*time.Second),
    )
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Implement request queuing
    rateLimiter := time.NewTicker(500 * time.Millisecond) // 2 requests per second
    defer rateLimiter.Stop()
    
    requests := []string{
        "What is AI?",
        "Define machine learning",
        "Explain neural networks",
        "What is deep learning?",
        "How do transformers work?",
    }
    
    fmt.Println("Rate-limited request processing:")
    fmt.Println("===============================")
    
    for i, prompt := range requests {
        <-rateLimiter.C // Wait for rate limiter
        
        fmt.Printf("Request %d: ", i+1)
        
        start := time.Now()
        response, err := provider.GenerateText(context.Background(), core.Request{
            Messages: []core.Message{
                {
                    Role: core.User,
                    Parts: []core.Part{core.Text{Text: prompt}},
                },
            },
            MaxTokens: 50,
        })
        
        duration := time.Since(start)
        
        if err != nil {
            if core.IsRateLimited(err) {
                fmt.Printf("Rate limited (took %v)\n", duration)
            } else {
                fmt.Printf("Error: %v\n", err)
            }
        } else {
            fmt.Printf("Success (took %v)\n", duration)
        }
    }
}
```

## Summary

The OpenAI-Compatible provider in GAI offers:
- **Universal Compatibility**: Works with any OpenAI-compatible API
- **Provider Presets**: Ready-to-use configurations for popular providers
- **Automatic Quirk Handling**: Transparent handling of provider limitations
- **Performance Variety**: From ultra-fast (Groq, Cerebras) to high-quality (xAI)
- **Cost Options**: From budget-friendly (Together) to premium (xAI)
- **Model Diversity**: Access to hundreds of models across providers

Key advantages:
- **Flexibility**: Easy switching between providers without code changes
- **Performance**: Access to specialized hardware (Groq's LPUs, Cerebras' WSE)
- **Cost Optimization**: Choose providers based on your budget requirements
- **Model Variety**: Access to models not available elsewhere
- **Redundancy**: Built-in fallback options for reliability

Best practices:
- Use Groq for speed-critical applications
- Use xAI for complex reasoning tasks
- Use Together/Fireworks for cost-effective deployments  
- Use Cerebras for high-throughput simple tasks
- Implement retry logic for rate-limited providers
- Monitor performance and costs across providers

Next steps:
- Explore [Provider Comparison](../guides/provider-comparison.md)
- Learn about [Cost Optimization](../guides/cost-optimization.md) strategies
- Try [Multi-Provider Deployment](../guides/multi-provider.md) patterns
- Review [Performance Tuning](../guides/performance-tuning.md) techniques