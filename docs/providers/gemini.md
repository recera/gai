# Gemini Provider Guide

This comprehensive guide covers everything you need to know about using Google's Gemini models with GAI, including Gemini 1.5 Flash, Pro, and experimental models, along with unique features like file uploads, safety configuration, citations, and comprehensive multimodal support.

## Table of Contents
- [Overview](#overview)
- [Installation & Setup](#installation--setup)
- [Configuration](#configuration)
- [Supported Models](#supported-models)
- [Basic Usage](#basic-usage)
- [Streaming](#streaming)
- [File Upload Support](#file-upload-support)
- [Safety Configuration](#safety-configuration)
- [Citations Support](#citations-support)
- [Multimodal Capabilities](#multimodal-capabilities)
- [Structured Outputs](#structured-outputs)
- [Tool Calling](#tool-calling)
- [System Instructions](#system-instructions)
- [Advanced Features](#advanced-features)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- [Performance Optimization](#performance-optimization)
- [Troubleshooting](#troubleshooting)

## Overview

The Gemini provider gives you access to Google's Gemini family of models, offering unique capabilities:
- **Gemini 1.5 Flash**: Fast, efficient model for most tasks
- **Gemini 1.5 Pro**: Advanced reasoning and complex task handling
- **Gemini 2.0 Flash Experimental**: Next-generation capabilities
- **Large Context Windows**: Up to 2 million tokens
- **Native File Upload**: Automatic handling of large media files

### Key Features
- ✅ **Automatic File Upload**: Native handling of images, videos, audio, and documents
- ✅ **Safety Configuration**: Fine-grained content safety controls with real-time events
- ✅ **Citations Support**: Automatic source attribution and streaming citation events
- ✅ **Multimodal Processing**: Support for text, images, audio, video, and documents
- ✅ **Structured Outputs**: JSON Schema-based response generation
- ✅ **Tool Calling**: Function calling with parallel execution
- ✅ **Large Context**: Up to 2M token context windows
- ✅ **Streaming**: Real-time responses with safety and citation events
- ✅ **System Instructions**: Separate handling of system prompts

### Gemini's Unique Strengths
- **Multimodal Native**: Built from the ground up for multimodal understanding
- **Massive Context**: 2 million token context window for large document processing
- **File API Integration**: Seamless upload and processing of large media files
- **Safety First**: Advanced safety filtering with configurable thresholds
- **Grounded Responses**: Built-in citation support for factual accuracy
- **Cost Effective**: Competitive pricing with generous free tier

## Installation & Setup

### 1. Install the Gemini Provider

```bash
go get github.com/recera/gai/providers/gemini@latest
```

### 2. Obtain an API Key

1. Visit [Google AI Studio](https://aistudio.google.com/app/apikey)
2. Create a new project or select existing one
3. Generate an API key
4. Copy the key for use in your application

### 3. Set Up Environment

```bash
# Set your API key
export GOOGLE_API_KEY="your-api-key-here"

# Optional: Set default model
export GEMINI_MODEL="gemini-1.5-flash"
```

### 4. Verify Setup

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "github.com/recera/gai/core"
    "github.com/recera/gai/providers/gemini"
)

func main() {
    // Create provider
    provider := gemini.New(
        gemini.WithAPIKey(os.Getenv("GOOGLE_API_KEY")),
    )
    
    // Test with a simple request
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Say 'Hello from Gemini!'"},
                },
            },
        },
        MaxTokens: 10,
    })
    
    if err != nil {
        log.Fatalf("Setup verification failed: %v", err)
    }
    
    fmt.Println("✅ Gemini provider is working!")
    fmt.Println("Response:", response.Text)
}
```

## Configuration

### Basic Configuration

```go
provider := gemini.New(
    gemini.WithAPIKey("your-api-key"),           // Required
    gemini.WithModel("gemini-1.5-flash"),       // Default model
    gemini.WithBaseURL("https://generativelanguage.googleapis.com"), // Custom endpoint
    gemini.WithMaxRetries(3),                   // Retry attempts
    gemini.WithRetryDelay(time.Second),         // Retry delay
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

// Advanced configuration
provider := gemini.New(
    gemini.WithAPIKey(apiKey),
    gemini.WithHTTPClient(httpClient),
    gemini.WithMetricsCollector(metricsCollector),
    // Default safety settings
    gemini.WithDefaultSafety(&core.SafetyConfig{
        Harassment: core.SafetyBlockMost,
        Hate:       core.SafetyBlockMost,
        Sexual:     core.SafetyBlockMost,
        Dangerous:  core.SafetyBlockFew,
    }),
)
```

### Environment-Based Configuration

```go
// Load configuration from environment
func createProviderFromEnv() *gemini.Provider {
    return gemini.New(
        gemini.WithAPIKey(os.Getenv("GOOGLE_API_KEY")),
        gemini.WithModel(getEnvOrDefault("GEMINI_MODEL", "gemini-1.5-flash")),
        gemini.WithBaseURL(getEnvOrDefault("GEMINI_BASE_URL", 
            "https://generativelanguage.googleapis.com")),
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

### Gemini 1.5 Family (Latest)

```go
// Gemini 1.5 Flash - Fast and efficient (default)
provider := gemini.New(
    gemini.WithModel("gemini-1.5-flash"),
)

// Gemini 1.5 Flash 8B - Smaller, faster variant
provider := gemini.New(
    gemini.WithModel("gemini-1.5-flash-8b"),
)

// Gemini 1.5 Pro - Advanced reasoning
provider := gemini.New(
    gemini.WithModel("gemini-1.5-pro"),
)
```

### Experimental Models

```go
// Gemini 2.0 Flash Experimental - Next generation
provider := gemini.New(
    gemini.WithModel("gemini-2.0-flash-exp"),
)
```

### Model Comparison

| Model | Context Window | Strengths | Best For | Speed | Cost |
|-------|---------------|-----------|----------|-------|------|
| Gemini 1.5 Flash | 1M tokens | Fast, efficient | Most use cases | Fast | $ |
| Gemini 1.5 Flash 8B | 1M tokens | Ultra fast, small | High volume | Fastest | $ |
| Gemini 1.5 Pro | 2M tokens | Advanced reasoning | Complex analysis | Medium | $$$ |
| Gemini 2.0 Flash Exp | 1M tokens | Latest features | Experimental | Medium | $$ |

## Basic Usage

### Simple Text Generation

```go
func generateText(provider *gemini.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Explain the benefits of renewable energy in 3 paragraphs."},
                },
            },
        },
        MaxTokens:   500,
        Temperature: 0.7,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Gemini's Response:")
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
func conversationExample(provider *gemini.Provider) {
    ctx := context.Background()
    
    messages := []core.Message{
        {
            Role: core.System,
            Parts: []core.Part{
                core.Text{Text: "You are a helpful AI assistant specializing in science education. Be engaging and informative."},
            },
        },
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "What is photosynthesis?"},
            },
        },
        {
            Role: core.Assistant,
            Parts: []core.Part{
                core.Text{Text: "Photosynthesis is the process by which plants convert light energy into chemical energy, producing glucose and oxygen from carbon dioxide and water."},
            },
        },
        {
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: "How does this process benefit ecosystems?"},
            },
        },
    }
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages:    messages,
        MaxTokens:   800,
        Temperature: 0.6,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Gemini:", response.Text)
}
```

### Large Document Processing

```go
func processLargeDocument(provider *gemini.Provider, document string) {
    ctx := context.Background()
    
    // Gemini can handle up to 2M tokens in Pro model
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "gemini-1.5-pro", // Use Pro for large documents
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are an expert document analyzer. Provide comprehensive analysis with specific examples."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: fmt.Sprintf(`Analyze this document and provide:
1. Executive summary (5 sentences)
2. Key themes and concepts
3. Critical findings or insights
4. Potential concerns or issues
5. Strategic recommendations

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
func streamingExample(provider *gemini.Provider) {
    ctx := context.Background()
    
    stream, err := provider.StreamText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a detailed explanation of machine learning fundamentals."},
                },
            },
        },
        MaxTokens:   1500,
        Temperature: 0.7,
        Stream:      true,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    fmt.Println("Streaming Gemini's response:")
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

### Advanced Streaming with Safety and Citations

```go
func advancedStreaming(provider *gemini.Provider) {
    ctx := context.Background()
    
    stream, err := provider.StreamText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Research and explain the latest developments in quantum computing."},
                },
            },
        },
        Safety: &core.SafetyConfig{
            Harassment: core.SafetyBlockFew,
            Hate:       core.SafetyBlockSome,
            Sexual:     core.SafetyBlockMost,
            Dangerous:  core.SafetyBlockFew,
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
            fmt.Println("Gemini is researching...")
            
        case core.EventTextDelta:
            text := event.TextDelta
            fmt.Print(text)
            fullText.WriteString(text)
            tokenCount++ // Approximate
            
        case core.EventSafety:
            fmt.Printf("\n[Safety Check: %s - %s (score: %.2f)]\n",
                event.Safety.Category,
                event.Safety.Action,
                event.Safety.Score)
                
        case core.EventCitations:
            fmt.Printf("\n[Citations Found: %d sources]\n", len(event.Citations))
            for i, citation := range event.Citations {
                fmt.Printf("  %d. %s (%s)\n", i+1, citation.Title, citation.URI)
            }
            
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

## File Upload Support

### Automatic File Upload

The Gemini provider automatically handles large files by uploading them to Gemini's File API:

```go
func videoAnalysis(provider *gemini.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Analyze this video and provide insights on the content, visual elements, and key moments."},
                    core.Video{
                        Source: core.BlobRef{
                            Kind: core.BlobURL,
                            URL:  "https://example.com/presentation.mp4",
                            MIME: "video/mp4",
                        },
                    },
                },
            },
        },
        MaxTokens: 1000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Video Analysis:")
    fmt.Println(response.Text)
}
```

### Audio Processing

```go
func audioTranscription(provider *gemini.Provider) {
    // Read local audio file
    audioData, err := os.ReadFile("meeting-recording.mp3")
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Transcribe this audio and provide a summary of key discussion points."},
                    core.Audio{
                        Source: core.BlobRef{
                            Kind:  core.BlobBytes,
                            Bytes: audioData,
                            MIME:  "audio/mp3",
                        },
                    },
                },
            },
        },
        MaxTokens: 1500,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Audio Transcription and Summary:")
    fmt.Println(response.Text)
}
```

### Document Processing

```go
func documentAnalysis(provider *gemini.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Extract key information from this document and create a structured summary."},
                    core.File{
                        Source: core.BlobRef{
                            Kind: core.BlobURL,
                            URL:  "https://example.com/report.pdf",
                            MIME: "application/pdf",
                        },
                    },
                },
            },
        },
        MaxTokens: 2000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Document Summary:")
    fmt.Println(response.Text)
}
```

### Supported File Types

- **Images**: JPEG, PNG, GIF, WebP
- **Videos**: MP4, AVI, MOV, WebM, MPEG
- **Audio**: MP3, WAV, FLAC, AAC
- **Documents**: PDF, TXT, HTML, CSS, JavaScript, Markdown, CSV, XML

## Safety Configuration

### Per-Request Safety Settings

```go
func safetyConfigExample(provider *gemini.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Discuss the challenges facing modern healthcare systems."},
                },
            },
        },
        Safety: &core.SafetyConfig{
            Harassment: core.SafetyBlockFew,    // Block only high probability
            Hate:       core.SafetyBlockSome,   // Block medium and above
            Sexual:     core.SafetyBlockMost,   // Block low and above
            Dangerous:  core.SafetyBlockNone,   // Don't block
        },
        MaxTokens: 800,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Healthcare Discussion:")
    fmt.Println(response.Text)
}
```

### Safety Events in Streaming

```go
func streamingWithSafetyEvents(provider *gemini.Provider) {
    ctx := context.Background()
    
    stream, err := provider.StreamText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a story about overcoming adversity."},
                },
            },
        },
        Safety: &core.SafetyConfig{
            Harassment: core.SafetyBlockSome,
            Hate:       core.SafetyBlockMost,
            Sexual:     core.SafetyBlockMost,
            Dangerous:  core.SafetyBlockFew,
        },
        Stream: true,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventSafety:
            safety := event.Safety
            fmt.Printf("\n[Safety Check: %s - %s]\n", 
                safety.Category, safety.Action)
            if safety.Blocked {
                fmt.Printf("Content blocked due to %s (score: %.2f)\n", 
                    safety.Category, safety.Score)
            }
            
        case core.EventTextDelta:
            fmt.Print(event.TextDelta)
            
        case core.EventError:
            fmt.Printf("\nError: %v\n", event.Err)
        }
    }
}
```

### Safety Levels

- `SafetyBlockNone`: Don't block any content
- `SafetyBlockFew`: Block only high probability harmful content
- `SafetyBlockSome`: Block medium and high probability harmful content
- `SafetyBlockMost`: Block low, medium, and high probability harmful content

## Citations Support

### Enabling Citations

```go
func citationsExample(provider *gemini.Provider) {
    ctx := context.Background()
    
    stream, err := provider.StreamText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "What are the latest scientific findings about climate change impacts on ocean ecosystems?"},
                },
            },
        },
        Stream: true, // Citations work best with streaming
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    var citations []core.Citation
    
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            fmt.Print(event.TextDelta)
            
        case core.EventCitations:
            citations = append(citations, event.Citations...)
            fmt.Printf("\n--- New Citations Found ---\n")
            for _, citation := range event.Citations {
                fmt.Printf("• %s\n  Source: %s\n  Range: %d-%d\n\n",
                    citation.Title,
                    citation.URI,
                    citation.Start,
                    citation.End)
            }
            
        case core.EventFinish:
            fmt.Printf("\n\n--- All Citations ---\n")
            for i, citation := range citations {
                fmt.Printf("%d. %s\n   %s\n", 
                    i+1, citation.Title, citation.URI)
            }
        }
    }
}
```

## Multimodal Capabilities

### Image and Text Analysis

```go
func multimodalAnalysis(provider *gemini.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "gemini-1.5-pro", // Pro model for complex analysis
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Analyze this chart and provide detailed insights:"},
                    core.ImageURL{
                        URL:    "https://example.com/financial-chart.png",
                        Detail: "high",
                    },
                    core.Text{Text: "Focus on trends, patterns, and potential implications for investment decisions."},
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

### Multiple Media Types

```go
func mixedMediaAnalysis(provider *gemini.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "gemini-1.5-pro",
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Compare and analyze these different data sources:"},
                    core.Text{Text: "1. Financial data (image):"},
                    core.ImageURL{URL: "https://example.com/financial-data.png"},
                    core.Text{Text: "2. Market analysis (document):"},
                    core.File{
                        Source: core.BlobRef{
                            Kind: core.BlobURL,
                            URL:  "https://example.com/market-analysis.pdf",
                            MIME: "application/pdf",
                        },
                    },
                    core.Text{Text: "3. Expert interview (audio):"},
                    core.Audio{
                        Source: core.BlobRef{
                            Kind: core.BlobURL,
                            URL:  "https://example.com/expert-interview.mp3",
                            MIME: "audio/mp3",
                        },
                    },
                    core.Text{Text: "Provide a comprehensive analysis combining insights from all sources."},
                },
            },
        },
        MaxTokens: 2000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Multi-Source Analysis:")
    fmt.Println(response.Text)
}
```

## Structured Outputs

### Simple Structured Generation

```go
type ProductReview struct {
    ProductName string   `json:"product_name"`
    Rating      int      `json:"rating"`
    Pros        []string `json:"pros"`
    Cons        []string `json:"cons"`
    Summary     string   `json:"summary"`
    Recommended bool     `json:"recommended"`
    Price       float64  `json:"estimated_price"`
}

func structuredOutputExample(provider *gemini.Provider) {
    ctx := context.Background()
    
    reviewText := `
    I've been using the new MacBook Pro M3 for three months now. The performance 
    is incredible - compilation times are blazingly fast, and the battery life 
    easily gets me through a full day of development work. The display is gorgeous 
    and the build quality feels premium. However, the price point is quite steep 
    at $2000+, and some software still has compatibility issues with Apple Silicon. 
    The port selection could be better too. Overall, it's an excellent machine 
    but expensive.
    `
    
    result, err := provider.GenerateObject(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "Extract and structure product review information from the text."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: reviewText},
                },
            },
        },
        MaxTokens: 500,
    }, ProductReview{})
    
    if err != nil {
        log.Fatal(err)
    }
    
    review := result.Value.(map[string]interface{})
    fmt.Printf("Product: %s\n", review["product_name"])
    fmt.Printf("Rating: %.0f/5\n", review["rating"])
    fmt.Printf("Pros: %v\n", review["pros"])
    fmt.Printf("Cons: %v\n", review["cons"])
    fmt.Printf("Recommended: %v\n", review["recommended"])
}
```

### Complex Nested Structures

```go
type CompanyAnalysis struct {
    Company      string         `json:"company"`
    Industry     string         `json:"industry"`
    MarketCap    string         `json:"market_cap"`
    Financials   Financials     `json:"financials"`
    Competitors  []string       `json:"competitors"`
    SWOT         SWOTAnalysis   `json:"swot_analysis"`
    Outlook      OutlookSection `json:"outlook"`
    Risk         RiskAssessment `json:"risk_assessment"`
}

type Financials struct {
    Revenue       string `json:"annual_revenue"`
    Profit        string `json:"net_profit"`
    GrowthRate    string `json:"growth_rate"`
    DebtToEquity  string `json:"debt_to_equity"`
}

type SWOTAnalysis struct {
    Strengths     []string `json:"strengths"`
    Weaknesses    []string `json:"weaknesses"`
    Opportunities []string `json:"opportunities"`
    Threats       []string `json:"threats"`
}

type OutlookSection struct {
    ShortTerm string `json:"short_term"`
    LongTerm  string `json:"long_term"`
    Rating    string `json:"overall_rating"`
}

type RiskAssessment struct {
    Level   string   `json:"risk_level"`
    Factors []string `json:"risk_factors"`
}

func complexStructuredOutput(provider *gemini.Provider) {
    ctx := context.Background()
    
    result, err := provider.GenerateObject(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Analyze Tesla Inc. and provide a comprehensive business analysis covering all financial and strategic aspects."},
                },
            },
        },
        MaxTokens:   2000,
        Temperature: 0.3, // Lower temperature for structured data
    }, CompanyAnalysis{})
    
    if err != nil {
        log.Fatal(err)
    }
    
    analysis := result.Value.(map[string]interface{})
    
    // Pretty print the analysis
    jsonBytes, _ := json.MarshalIndent(analysis, "", "  ")
    fmt.Println("Tesla Analysis:")
    fmt.Println(string(jsonBytes))
}
```

## Tool Calling

### Creating Tools for Gemini

```go
// Weather lookup tool
type WeatherInput struct {
    Location string `json:"location" jsonschema:"required,description=City and country"`
    Unit     string `json:"unit,omitempty" jsonschema:"enum=celsius,enum=fahrenheit,default=celsius"`
}

type WeatherOutput struct {
    Temperature float64 `json:"temperature"`
    Condition   string  `json:"condition"`
    Humidity    int     `json:"humidity"`
    WindSpeed   float64 `json:"wind_speed"`
    Forecast    string  `json:"forecast"`
}

func createWeatherTool() tools.Handle {
    return tools.New[WeatherInput, WeatherOutput](
        "get_weather",
        "Get current weather and forecast for a location",
        func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
            // In production, call a real weather API
            return WeatherOutput{
                Temperature: 22.5,
                Condition:   "Partly cloudy with light breeze",
                Humidity:    65,
                WindSpeed:   15.2,
                Forecast:    "Sunny tomorrow, rain expected weekend",
            }, nil
        },
    )
}

// Stock information tool
type StockInput struct {
    Symbol string `json:"symbol" jsonschema:"required,description=Stock symbol (e.g., AAPL, TSLA)"`
}

type StockOutput struct {
    Symbol      string  `json:"symbol"`
    Price       float64 `json:"current_price"`
    Change      float64 `json:"price_change"`
    ChangePercent float64 `json:"change_percent"`
    Volume      int64   `json:"volume"`
    MarketCap   string  `json:"market_cap"`
}

func createStockTool() tools.Handle {
    return tools.New[StockInput, StockOutput](
        "get_stock_info",
        "Get current stock price and market information",
        func(ctx context.Context, input StockInput, meta tools.Meta) (StockOutput, error) {
            // Simulate stock data
            return StockOutput{
                Symbol:        input.Symbol,
                Price:         178.50,
                Change:        2.30,
                ChangePercent: 1.31,
                Volume:        45678900,
                MarketCap:     "2.8T",
            }, nil
        },
    )
}
```

### Using Tools with Gemini

```go
func toolCallingExample(provider *gemini.Provider) {
    weatherTool := createWeatherTool()
    stockTool := createStockTool()
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "What's the weather like in Tokyo today? Also, how is Apple stock performing?"},
                },
            },
        },
        Tools: []tools.Handle{weatherTool, stockTool},
        ToolChoice: core.ToolAuto, // Let Gemini decide which tools to use
        MaxTokens: 800,
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

### Parallel Tool Execution

```go
func parallelToolsExample(provider *gemini.Provider) {
    // Create multiple tools
    tools := []tools.Handle{
        createWeatherTool(),
        createStockTool(),
        createNewsSearchTool(),
        createCalculatorTool(),
    }
    
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Give me a morning briefing: weather in San Francisco, TSLA stock price, latest tech news, and calculate the percentage change if TSLA goes from $180 to $200."},
                },
            },
        },
        Tools: tools,
        ToolChoice: core.ToolAuto,
        MaxTokens: 1200,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Morning Briefing from Gemini:")
    fmt.Println(response.Text)
}
```

## System Instructions

### Using System Instructions

```go
func systemInstructionsExample(provider *gemini.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: `You are Claude, a helpful AI assistant created by Anthropic.

Your personality and approach:
- Be concise but thorough in your explanations
- Use examples to illustrate complex concepts
- Ask clarifying questions when requests are ambiguous
- Acknowledge the limits of your knowledge
- Be encouraging and supportive in your responses

For technical questions:
- Provide practical, actionable advice
- Include code examples when relevant
- Explain the reasoning behind your recommendations
- Consider edge cases and potential issues

Always prioritize accuracy and helpfulness.`},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "I'm learning Go programming. Can you help me understand channels?"},
                },
            },
        },
        MaxTokens: 1000,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("System-Guided Response:")
    fmt.Println(response.Text)
}
```

## Advanced Features

### Context Window Optimization

```go
func handleLargeContext(provider *gemini.Provider, documents []string) {
    // Gemini 1.5 Pro supports up to 2M tokens
    
    ctx := context.Background()
    
    // Combine multiple documents
    var combinedText strings.Builder
    for i, doc := range documents {
        combinedText.WriteString(fmt.Sprintf("Document %d:\n%s\n\n", i+1, doc))
    }
    
    response, err := provider.GenerateText(ctx, core.Request{
        Model: "gemini-1.5-pro", // Use Pro for large contexts
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are an expert document analyst. Analyze the provided documents and extract key insights, relationships, and patterns."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: fmt.Sprintf(`Analyze these documents and provide:

1. Cross-document themes and patterns
2. Contradictions or inconsistencies 
3. Key insights and findings
4. Synthesis and conclusions
5. Recommendations for action

Documents:
%s`, combinedText.String())},
                },
            },
        },
        MaxTokens: 3000,
        Temperature: 0.3,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Multi-Document Analysis:")
    fmt.Println(response.Text)
}
```

### Provider-Specific Options

```go
func geminiProviderOptions(provider *gemini.Provider) {
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
        MaxTokens: 1000,
        Temperature: 0.8,
        ProviderOptions: map[string]interface{}{
            "gemini": map[string]interface{}{
                "top_p":           0.9,
                "top_k":           40,
                "candidate_count": 1,
                "stop_sequences":  []string{"THE END", "CONCLUSION"},
            },
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Creative Story:")
    fmt.Println(response.Text)
}
```

## Error Handling

### Comprehensive Error Handling

```go
func handleGeminiErrors(provider *gemini.Provider) {
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
            fmt.Println("Rate limited by Google")
            waitTime := core.GetRetryAfter(err)
            fmt.Printf("Waiting %v before retry...\n", waitTime)
            time.Sleep(waitTime)
            // Retry logic here
            
        case core.IsContextSizeExceeded(err):
            fmt.Println("Message too long for Gemini model")
            // Implement context reduction strategy
            
        case core.IsBadRequest(err):
            fmt.Println("Invalid request format:", err)
            // Fix request parameters
            
        case core.IsAuth(err):
            fmt.Println("API key issue:", err)
            // Check Google API key configuration
            
        case core.IsSafetyBlocked(err):
            fmt.Println("Content blocked by safety filters")
            // Adjust content or safety settings
            
        case core.IsOverloaded(err):
            fmt.Println("Gemini service is overloaded")
            // Use fallback or queue for later
            
        case core.IsQuotaExceeded(err):
            fmt.Println("Quota exceeded - check billing and limits")
            
        default:
            fmt.Printf("Unexpected error: %v\n", err)
        }
        return
    }
    
    fmt.Println("Success:", response.Text)
}
```

### File Upload Error Handling

```go
func handleFileUploadErrors(provider *gemini.Provider) {
    ctx := context.Background()
    
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Analyze this large video file"},
                    core.Video{
                        Source: core.BlobRef{
                            Kind: core.BlobURL,
                            URL:  "https://example.com/very-large-video.mp4",
                            MIME: "video/mp4",
                        },
                    },
                },
            },
        },
    })
    
    if err != nil {
        if strings.Contains(err.Error(), "file too large") {
            fmt.Println("File is too large for upload")
            // Handle large file differently
        } else if strings.Contains(err.Error(), "unsupported file type") {
            fmt.Println("File type not supported")
            // Convert or use different approach
        } else if strings.Contains(err.Error(), "upload failed") {
            fmt.Println("File upload failed")
            // Retry or use different method
        } else {
            fmt.Printf("File processing error: %v\n", err)
        }
        return
    }
    
    fmt.Println("Video Analysis:", response.Text)
}
```

## Best Practices

### 1. Model Selection

```go
func selectGeminiModel(taskType string) string {
    modelSelection := map[string]string{
        "quick_questions":    "gemini-1.5-flash-8b",    // Ultra fast
        "general_tasks":      "gemini-1.5-flash",       // Fast & efficient
        "complex_analysis":   "gemini-1.5-pro",         // Most capable
        "multimodal":         "gemini-1.5-pro",         // Best for files
        "large_documents":    "gemini-1.5-pro",         // 2M token context
        "creative_writing":   "gemini-1.5-pro",         // More creative
        "experimental":       "gemini-2.0-flash-exp",   // Latest features
    }
    
    if model, ok := modelSelection[taskType]; ok {
        return model
    }
    return "gemini-1.5-flash" // Default
}
```

### 2. Safety Configuration Guidelines

```go
func configureSafety(contentType string) *core.SafetyConfig {
    switch contentType {
    case "educational":
        return &core.SafetyConfig{
            Harassment: core.SafetyBlockSome,
            Hate:       core.SafetyBlockSome,
            Sexual:     core.SafetyBlockMost,
            Dangerous:  core.SafetyBlockFew,
        }
    case "medical":
        return &core.SafetyConfig{
            Harassment: core.SafetyBlockMost,
            Hate:       core.SafetyBlockMost,
            Sexual:     core.SafetyBlockMost,
            Dangerous:  core.SafetyBlockSome,
        }
    case "creative":
        return &core.SafetyConfig{
            Harassment: core.SafetyBlockSome,
            Hate:       core.SafetyBlockSome,
            Sexual:     core.SafetyBlockSome,
            Dangerous:  core.SafetyBlockFew,
        }
    default:
        return &core.SafetyConfig{
            Harassment: core.SafetyBlockSome,
            Hate:       core.SafetyBlockSome,
            Sexual:     core.SafetyBlockMost,
            Dangerous:  core.SafetyBlockSome,
        }
    }
}
```

### 3. File Upload Best Practices

```go
func optimizeFileUpload(fileSize int64, mimeType string) core.BlobRef {
    // Use appropriate upload method based on file size
    if fileSize < 20*1024*1024 { // 20MB
        // Small files - use BlobBytes for faster processing
        data, _ := os.ReadFile("small-file.jpg")
        return core.BlobRef{
            Kind:  core.BlobBytes,
            Bytes: data,
            MIME:  mimeType,
        }
    } else {
        // Large files - use BlobURL to avoid memory issues
        return core.BlobRef{
            Kind: core.BlobURL,
            URL:  "https://example.com/large-file.mp4",
            MIME: mimeType,
        }
    }
}
```

### 4. Context Window Management

```go
func manageGeminiContext(messages []core.Message, modelType string) []core.Message {
    var maxTokens int
    switch modelType {
    case "gemini-1.5-pro":
        maxTokens = 2000000 // 2M tokens
    case "gemini-1.5-flash", "gemini-1.5-flash-8b":
        maxTokens = 1000000 // 1M tokens
    case "gemini-2.0-flash-exp":
        maxTokens = 1000000 // 1M tokens
    default:
        maxTokens = 1000000
    }
    
    // Estimate current token usage (rough: 1 token ≈ 4 characters)
    currentTokens := estimateTokens(messages)
    
    if currentTokens <= maxTokens {
        return messages
    }
    
    // For Gemini, prioritize keeping system instructions and recent context
    return truncateMessagesForGemini(messages, maxTokens)
}
```

## Performance Optimization

### 1. Streaming for Better UX

```go
func optimizedStreaming(provider *gemini.Provider) {
    // Always use streaming for long-form content
    stream, err := provider.StreamText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a detailed analysis..."},
                },
            },
        },
        Stream: true,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    
    // Process events efficiently
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            // Stream text immediately to user
            fmt.Print(event.TextDelta)
        }
    }
}
```

### 2. Concurrent Processing

```go
func processConcurrently(provider *gemini.Provider, documents []string) {
    var wg sync.WaitGroup
    results := make(chan string, len(documents))
    
    // Process multiple documents concurrently
    for i, doc := range documents {
        wg.Add(1)
        go func(id int, content string) {
            defer wg.Done()
            
            response, err := provider.GenerateText(context.Background(), core.Request{
                Messages: []core.Message{
                    {
                        Role: core.User,
                        Parts: []core.Part{
                            core.Text{Text: "Summarize: " + content},
                        },
                    },
                },
            })
            
            if err != nil {
                results <- fmt.Sprintf("Document %d: Error - %v", id, err)
                return
            }
            
            results <- fmt.Sprintf("Document %d: %s", id, response.Text)
        }(i, doc)
    }
    
    // Wait for all goroutines to complete
    go func() {
        wg.Wait()
        close(results)
    }()
    
    // Collect results
    for result := range results {
        fmt.Println(result)
    }
}
```

### 3. Caching Strategies

```go
type GeminiCache struct {
    cache sync.Map
    ttl   time.Duration
}

type cacheEntry struct {
    result    *core.TextResult
    timestamp time.Time
}

func (c *GeminiCache) GetOrGenerate(
    ctx context.Context,
    provider *gemini.Provider,
    request core.Request,
) (*core.TextResult, error) {
    // Create cache key from request content
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

## Troubleshooting

### Common Issues and Solutions

#### 1. File Upload Failures

**Problem**: Files failing to upload or process

**Solution**:
```go
// Check file size and format
func validateFile(blobRef core.BlobRef) error {
    // Max file size for Gemini: 20MB for images, 2GB for videos
    maxSizes := map[string]int64{
        "image/*": 20 * 1024 * 1024,    // 20MB
        "video/*": 2 * 1024 * 1024 * 1024, // 2GB
        "audio/*": 200 * 1024 * 1024,   // 200MB
        "text/*":  20 * 1024 * 1024,    // 20MB
    }
    
    if blobRef.Kind == core.BlobBytes && len(blobRef.Bytes) > 20*1024*1024 {
        return fmt.Errorf("file too large for direct upload, use BlobURL instead")
    }
    
    return nil
}
```

#### 2. Safety Blocks

**Problem**: Content getting blocked by safety filters

**Solution**:
```go
// Adjust safety settings and retry
func handleSafetyBlocks(provider *gemini.Provider, request core.Request) {
    // Try with more permissive settings
    request.Safety = &core.SafetyConfig{
        Harassment: core.SafetyBlockFew,
        Hate:       core.SafetyBlockFew,
        Sexual:     core.SafetyBlockSome,
        Dangerous:  core.SafetyBlockFew,
    }
    
    response, err := provider.GenerateText(context.Background(), request)
    if err != nil && core.IsSafetyBlocked(err) {
        fmt.Println("Content still blocked - consider rephrasing input")
    }
}
```

#### 3. Context Length Issues

**Problem**: "Maximum context length exceeded"

**Solution**:
```go
// Implement sliding window for very large contexts
func handleLargeContext(messages []core.Message) []core.Message {
    const maxTokens = 1000000 // For Flash models
    
    estimated := estimateTokens(messages)
    if estimated <= maxTokens {
        return messages
    }
    
    // Keep system message and truncate conversation
    if len(messages) > 0 && messages[0].Role == core.System {
        systemMsg := messages[0]
        recentMessages := messages[1:]
        
        // Keep most recent messages that fit
        truncated := truncateToTokenLimit(recentMessages, maxTokens-1000)
        
        return append([]core.Message{systemMsg}, truncated...)
    }
    
    return truncateToTokenLimit(messages, maxTokens)
}
```

#### 4. Quota and Rate Limits

**Problem**: Hitting usage quotas or rate limits

**Solution**:
```go
// Implement exponential backoff with quota awareness
func handleQuotaLimits(provider *gemini.Provider) {
    provider := gemini.New(
        gemini.WithAPIKey(apiKey),
        gemini.WithMaxRetries(5),
        gemini.WithRetryDelay(2 * time.Second), // Longer delays for Gemini
    )
    
    // Use rate limiting middleware
    rateLimited := middleware.WithRateLimit(
        provider,
        middleware.RateLimitOpts{
            RequestsPerMinute: 15, // Conservative limit
            Burst:            5,
        },
    )
}
```

## Summary

The Gemini provider in GAI offers:
- **Unique File Handling**: Automatic upload and processing of large media files
- **Advanced Safety**: Fine-grained safety controls with real-time events
- **Citation Support**: Built-in source attribution for factual accuracy
- **Massive Context**: Up to 2M tokens for large document processing
- **Multimodal Native**: Purpose-built for mixed media understanding

Key advantages over other providers:
- **File API Integration**: Seamless handling of videos, audio, and large documents
- **Safety Events**: Real-time safety assessment with configurable thresholds  
- **Citation Streaming**: Live source attribution during generation
- **Cost Effectiveness**: Competitive pricing with generous free tier
- **Massive Context**: 2M token context window vs 200K for Claude, 128K for GPT-4

Best practices:
- Use Pro model for complex analysis and large contexts
- Configure safety settings based on your use case
- Leverage file uploads for multimodal applications
- Use streaming for better user experience
- Take advantage of the large context window for document analysis

Next steps:
- Explore [Tool Calling](../features/tool-calling.md) with Gemini
- Learn about [Safety Configuration](../guides/safety-configuration.md)
- Try [File Upload Capabilities](../features/file-uploads.md)
- Review [Multimodal Processing](../guides/multimodal.md) strategies