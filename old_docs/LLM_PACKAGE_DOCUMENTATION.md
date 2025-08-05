# LLM Package Documentation

The `internal/llm` package provides a unified interface for interacting with multiple Large Language Model (LLM) providers, with advanced response parsing capabilities and type-safe structured outputs.

## Table of Contents

1. [Package Overview](#package-overview)
2. [Core Components](#core-components)
3. [Getting Started](#getting-started)
4. [Provider Support](#provider-support)
5. [LLMCallParts - Request Configuration](#llmcallparts---request-configuration)
6. [GetResponseObject - Structured Outputs](#getresponseobject---structured-outputs)
7. [BuildActionPrompt - Template Integration](#buildactionprompt---template-integration)
8. [Response Parsing System](#response-parsing-system)
9. [API Reference](#api-reference)
10. [Examples](#examples)

## Package Overview

The LLM package consists of several key modules:

- **Core Client**: Unified interface for multiple LLM providers
- **Provider Implementations**: Support for OpenAI, Anthropic, Gemini, Groq, and Cerebras
- **Response Parser**: Robust JSON parsing with error recovery
- **Type System**: Structured request/response handling

### Key Features

- **Multi-provider support** with consistent API
- **Automatic API key management** from environment variables
- **Structured response parsing** with type coercion
- **Robust error handling** and retry mechanisms
- **Template-based prompt construction**
- **Multimodal support** (text and images)

## Core Components

### Main Package (`llm/`)

```go
// Core types (re-exported from subpackages)
type LLMCallParts = types.LLMCallParts
type Message = types.Message
type Content = types.Content
type LLMResponse = types.LLMResponse

// Main client interface
type LLMClient interface {
    GetCompletion(ctx context.Context, parts LLMCallParts) (LLMResponse, error)
    GetResponseObject(ctx context.Context, parts LLMCallParts, v any) error
}
```

### Provider System (`llm/providers/`)

All providers implement the `ProviderClient` interface:

```go
type ProviderClient interface {
    GetCompletion(ctx context.Context, parts types.LLMCallParts) (types.LLMResponse, error)
}
```

Supported providers:
- **OpenAI**: GPT-3.5, GPT-4, etc.
- **Anthropic**: Claude models
- **Google Gemini**: Gemini Pro, etc.
- **Groq**: Fast inference models
- **Cerebras**: High-performance models

### Response Parser (`llm/responseParser/`)

The response parser handles the challenging task of converting LLM outputs to structured Go types:

- **Cleanup**: Removes markdown, extracts JSON, fixes incomplete structures
- **Parser**: Handles malformed JSON, relaxed syntax, and autocorrection
- **Coercer**: Performs intelligent type coercion between JSON and Go types

## Getting Started

### 1. Installation

The package is internal to your project. Import it as:

```go
import "goru/internal/llm"
```

### 2. Environment Setup

Create a `.env` file in your project root with your API keys:

```env
OPENAI_API_KEY=your_openai_key
ANTHROPIC_API_KEY=your_anthropic_key
GEMINI_API_KEY=your_gemini_key
GROQ_API_KEY=your_groq_key
CEREBRAS_API_KEY=your_cerebras_key
```

### 3. Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "goru/internal/llm"
)

func main() {
    // Create client
    client, err := llm.NewClient()
    if err != nil {
        panic(err)
    }

    // Configure request
    parts := llm.NewLLMCallParts()
    parts.Provider = "openai"
    parts.Model = "gpt-4"
    
    // Add system message
    parts.System.AddTextContent("You are a helpful assistant.")
    
    // Add user message
    userMsg := llm.Message{Role: "user"}
    userMsg.AddTextContent("What is the capital of France?")
    parts.AddMessage(userMsg)

    // Get response
    response, err := client.GetCompletion(context.Background(), parts)
    if err != nil {
        panic(err)
    }

    fmt.Println(response.Content)
}
```

## Provider Support

### Default Configuration

The default `NewLLMCallParts()` creates a configuration with:
- Provider: "cerebras"
- Model: "llama-3.3-70b"
- MaxTokens: 1000
- Temperature: 0.2

### Provider-Specific Models

Each provider supports different models. Configure them in the `LLMCallParts`:

```go
parts := llm.NewLLMCallParts()

// OpenAI
parts.Provider = "openai"
parts.Model = "gpt-4o"

// Anthropic
parts.Provider = "anthropic" 
parts.Model = "claude-3-sonnet-20240229"

// Gemini
parts.Provider = "gemini"
parts.Model = "gemini-2.0-flash-exp"

// Groq
parts.Provider = "groq"
parts.Model = "mixtral-8x7b-32768"

// Cerebras
parts.Provider = "cerebras"
parts.Model = "llama-3.3-70b"
```

## LLMCallParts - Request Configuration

The `LLMCallParts` struct configures your LLM request:

```go
type LLMCallParts struct {
    Provider    string     // LLM provider ("openai", "anthropic", etc.)
    Model       string     // Model name
    System      Message    // System message
    Messages    []Message  // Conversation messages
    MaxTokens   int        // Maximum tokens to generate
    Temperature float64    // Creativity/randomness (0.0-1.0)
}
```

### Working with Messages

Messages support multimodal content:

```go
parts := llm.NewLLMCallParts()

// System message
parts.System.AddTextContent("You are a helpful assistant.")

// User message with text
userMsg := llm.Message{Role: "user"}
userMsg.AddTextContent("Describe this image:")

// Add image content
imageData, _ := os.ReadFile("image.jpg")
userMsg.AddImageContent("image/jpeg", imageData)

// Or add image from URL
userMsg.AddImageContentFromURL("image/jpeg", "https://example.com/image.jpg")

parts.AddMessage(userMsg)

// Assistant response
assistantMsg := llm.Message{Role: "assistant"}
assistantMsg.AddTextContent("I can see...")
parts.AddMessage(assistantMsg)
```

### File Content Integration

Load content directly from files:

```go
// Add file content as text
fileContent := llm.AddFileAsContent("path/to/file.txt")
userMsg.AddContent(fileContent)
```

## GetResponseObject - Structured Outputs

The `GetResponseObject` method forces LLM outputs into Go structs, providing type-safe responses:

### How It Works

1. **Generate Instructions**: Creates detailed JSON format instructions from your struct
2. **Make LLM Call**: Requests response in the specified format
3. **Parse & Validate**: Robust parsing with error recovery
4. **Retry on Failure**: Automatically retries with correction prompts

### Example Usage

```go
// Define your response structure
type WeatherResponse struct {
    Location    string  `json:"location" desc:"City name"`
    Temperature int     `json:"temperature" desc:"Temperature in Celsius"`
    Conditions  string  `json:"conditions" desc:"Weather conditions"`
    Humidity    float64 `json:"humidity" desc:"Humidity percentage"`
}

// Use GetResponseObject
parts := llm.NewLLMCallParts()
parts.System.AddTextContent("You are a weather API.")

userMsg := llm.Message{Role: "user"}
userMsg.AddTextContent("What's the weather in Paris?")
parts.AddMessage(userMsg)

var weather WeatherResponse
err := client.GetResponseObject(context.Background(), parts, &weather)
if err != nil {
    panic(err)
}

fmt.Printf("Weather in %s: %d°C, %s\n", 
    weather.Location, weather.Temperature, weather.Conditions)
```

### Struct Tags for Better Parsing

Use struct tags to improve parsing:

```go
type TaskList struct {
    Tasks []Task `json:"tasks" desc:"List of tasks to complete"`
    Total int    `json:"total_count" desc:"Total number of tasks"`
}

type Task struct {
    ID       string `json:"id" desc:"Unique task identifier"`
    Title    string `json:"title" desc:"Task title"`
    Done     bool   `json:"completed" desc:"Whether task is finished"`
    Priority int    `json:"priority" desc:"Priority level 1-5"`
}
```

The `desc` tag provides descriptions that help the LLM understand field purposes.

## BuildActionPrompt - Template Integration

`BuildActionPrompt` combines file-based prompts with structured response formats:

```go
func BuildActionPrompt(filePath string, responseStruct any) (string, error)
```

### How It Works

1. **Read Template**: Loads prompt template from file
2. **Generate Instructions**: Creates JSON format instructions from struct
3. **Combine**: Concatenates template + instructions

### Example Usage

Create a prompt file `prompts/analyze_code.txt`:
```
Analyze the following code and identify:
- Potential bugs
- Performance issues  
- Security vulnerabilities
- Code quality improvements

Code to analyze:
{{USER_WILL_PROVIDE_CODE}}
```

Define response structure:
```go
type CodeAnalysis struct {
    Bugs          []Issue `json:"bugs" desc:"Potential bugs found"`
    Performance   []Issue `json:"performance" desc:"Performance issues"`
    Security      []Issue `json:"security" desc:"Security vulnerabilities"`
    Improvements  []Issue `json:"improvements" desc:"Code quality suggestions"`
}

type Issue struct {
    Line        int    `json:"line" desc:"Line number"`
    Description string `json:"description" desc:"Issue description"`
    Severity    string `json:"severity" desc:"low, medium, high, critical"`
    Suggestion  string `json:"suggestion" desc:"How to fix"`
}
```

Use BuildActionPrompt:
```go
prompt, err := llm.BuildActionPrompt("prompts/analyze_code.txt", CodeAnalysis{})
if err != nil {
    panic(err)
}

parts := llm.NewLLMCallParts()
parts.System.AddTextContent(prompt)

// Add user code
userMsg := llm.Message{Role: "user"}
userMsg.AddTextContent(codeToAnalyze)
parts.AddMessage(userMsg)

var analysis CodeAnalysis
err = client.GetResponseObject(context.Background(), parts, &analysis)
```

## Response Parsing System

The response parser is the most sophisticated part of the package, handling the messy reality of LLM outputs.

### Three-Layer Architecture

1. **Cleanup Layer** (`cleanup/`): Preprocesses LLM responses
2. **Parser Layer** (`parser/`): Converts to canonical JSON
3. **Coercer Layer** (`coercer/`): Type coercion and mapping

### Cleanup Features

- **Markdown Removal**: Strips code fences and formatting
- **JSON Extraction**: Finds JSON within mixed text
- **Structure Completion**: Balances brackets and braces

### Parser Features

- **Relaxed Parsing**: Handles comments, trailing commas
- **Quote Normalization**: Fixes smart quotes, single quotes
- **Format Correction**: Fixes `=>` instead of `:`, etc.
- **Autocompletion**: Completes truncated JSON

### Coercer Features

- **Type Coercion**: String→Number, String→Boolean, etc.
- **Case Handling**: snake_case, camelCase, PascalCase
- **Time Parsing**: Multiple time formats
- **Collection Handling**: CSV strings to slices

### Parsing Options

Control parsing behavior with options:

```go
// Default: lenient parsing with all recovery features
opts := responseParser.DefaultOptions()

// Strict: requires valid JSON
opts := responseParser.StrictOptions()

// Custom options
opts := responseParser.ParseOptions{
    Relaxed:       true,  // Allow comments, trailing commas
    Extract:       true,  // Extract JSON from text
    Autocomplete:  true,  // Complete partial JSON
    FixFormat:     true,  // Fix common formatting issues
    AllowCoercion: true,  // Enable type coercion
}

var result MyStruct
err := responseParser.ParseInto(llmOutput, &result, opts)
```

## API Reference

### Core Functions

```go
// Create new client
func NewClient() (LLMClient, error)

// Create default call configuration
func NewLLMCallParts() LLMCallParts

// Generate response format instructions
func ResponseInstructions(s interface{}) (string, error)

// Parse LLM response into struct
func ParseInto(raw string, target interface{}) error

// Build prompt from template + struct
func BuildActionPrompt(filePath string, responseStruct any) (string, error)

// Read file as string
func StringFromPath(filePath string) (string, error)

// Add file content to message
func AddFileAsContent(filePath string) TextContent
```

### LLMClient Interface

```go
type LLMClient interface {
    // Get raw text response
    GetCompletion(ctx context.Context, parts LLMCallParts) (LLMResponse, error)
    
    // Get structured response
    GetResponseObject(ctx context.Context, parts LLMCallParts, v any) error
}
```

### Types

```go
// Request configuration
type LLMCallParts struct {
    Provider    string
    Model       string  
    System      Message
    Messages    []Message
    MaxTokens   int
    Temperature float64
}

// Message in conversation
type Message struct {
    Role     string    // "system", "user", "assistant"
    Contents []Content // Text, images, etc.
}

// Content types
type TextContent struct {
    Text string
}

type ImageContent struct {
    MIMEType string
    Data     []byte  // Raw image data
    URL      string  // Or image URL
}

// Response structure
type LLMResponse struct {
    Content      string      // Generated text
    FinishReason string      // Why generation stopped
    Usage        TokenUsage  // Token consumption
}

type TokenUsage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}
```

## Examples

### Example 1: Simple Q&A

```go
func askQuestion(client llm.LLMClient, question string) (string, error) {
    parts := llm.NewLLMCallParts()
    parts.Provider = "openai"
    parts.Model = "gpt-4"
    
    userMsg := llm.Message{Role: "user"}
    userMsg.AddTextContent(question)
    parts.AddMessage(userMsg)
    
    response, err := client.GetCompletion(context.Background(), parts)
    if err != nil {
        return "", err
    }
    
    return response.Content, nil
}
```

### Example 2: Code Analysis

```go
type CodeReview struct {
    Issues       []CodeIssue `json:"issues" desc:"Code issues found"`
    Score        int         `json:"score" desc:"Code quality score 1-10"`
    Summary      string      `json:"summary" desc:"Overall assessment"`
    Suggestions  []string    `json:"suggestions" desc:"Improvement suggestions"`
}

type CodeIssue struct {
    Type        string `json:"type" desc:"bug, style, performance, security"`
    Line        int    `json:"line" desc:"Line number"`
    Description string `json:"description" desc:"Issue description"`
    Severity    string `json:"severity" desc:"low, medium, high"`
}

func reviewCode(client llm.LLMClient, code string) (*CodeReview, error) {
    parts := llm.NewLLMCallParts()
    parts.System.AddTextContent("You are an expert code reviewer.")
    
    userMsg := llm.Message{Role: "user"}
    userMsg.AddTextContent(fmt.Sprintf("Review this code:\n\n%s", code))
    parts.AddMessage(userMsg)
    
    var review CodeReview
    err := client.GetResponseObject(context.Background(), parts, &review)
    if err != nil {
        return nil, err
    }
    
    return &review, nil
}
```

### Example 3: Multimodal Analysis

```go
func analyzeImage(client llm.LLMClient, imagePath string) (string, error) {
    parts := llm.NewLLMCallParts()
    parts.Provider = "openai"
    parts.Model = "gpt-4-vision-preview"
    
    parts.System.AddTextContent("Analyze images in detail.")
    
    userMsg := llm.Message{Role: "user"}
    userMsg.AddTextContent("What do you see in this image?")
    
    imageData, err := os.ReadFile(imagePath)
    if err != nil {
        return "", err
    }
    
    userMsg.AddImageContent("image/jpeg", imageData)
    parts.AddMessage(userMsg)
    
    response, err := client.GetCompletion(context.Background(), parts)
    if err != nil {
        return "", err
    }
    
    return response.Content, nil
}
```

### Example 4: Template-Based Workflow

```go
// prompts/summarize.txt:
// Summarize the following text, focusing on key points and actionable insights.
// Keep the summary concise but comprehensive.
//
// Text to summarize:

type Summary struct {
    KeyPoints    []string `json:"key_points" desc:"Main points from the text"`
    ActionItems  []string `json:"action_items" desc:"Actionable insights"`
    WordCount    int      `json:"word_count" desc:"Original text word count"`
    SummaryRatio float64  `json:"summary_ratio" desc:"Summary length vs original"`
}

func summarizeText(client llm.LLMClient, text string) (*Summary, error) {
    prompt, err := llm.BuildActionPrompt("prompts/summarize.txt", Summary{})
    if err != nil {
        return nil, err
    }
    
    parts := llm.NewLLMCallParts()
    parts.System.AddTextContent(prompt)
    
    userMsg := llm.Message{Role: "user"}
    userMsg.AddTextContent(text)
    parts.AddMessage(userMsg)
    
    var summary Summary
    err = client.GetResponseObject(context.Background(), parts, &summary)
    if err != nil {
        return nil, err
    }
    
    return &summary, nil
}
```

---

This package provides a robust, production-ready foundation for LLM integration with structured outputs, error recovery, and multi-provider support. The response parsing system handles the complexity of LLM outputs while providing a clean, type-safe interface for your applications.