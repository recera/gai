# gai - Go AI SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/collinshill/gai.svg)](https://pkg.go.dev/github.com/collinshill/gai)
[![Go Report Card](https://goreportcard.com/badge/github.com/collinshill/gai)](https://goreportcard.com/report/github.com/collinshill/gai)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

gai (Go AI) is a powerful, type-safe Go SDK for building AI applications with Large Language Models (LLMs). It provides a unified interface for multiple providers while offering advanced features like structured outputs, robust JSON parsing, and comprehensive error handling.

## ✨ Key Features

- **🔌 Multi-Provider Support**: Seamless integration with OpenAI, Anthropic, Google Gemini, Groq, and Cerebras
- **🛡️ Type-Safe Actions**: Generic `Action[T]` pattern for compile-time type safety
- **🏗️ Fluent Builder API**: Chainable methods for elegant request construction
- **🧩 Robust JSON Parsing**: Handles malformed LLM outputs with intelligent recovery
- **⚡ Zero Configuration**: Works out of the box with environment variables
- **🎯 Structured Errors**: Rich error context with provider, model, and HTTP details
- **📝 Template Support**: Built-in prompt templating with Go's text/template
- **🔍 Conversation Management**: Tools for history manipulation and token management

## 📦 Installation

```bash
go get github.com/collinshill/gai
```

## 🚀 Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/collinshill/gai"
)

func main() {
    // Create client (reads API keys from environment)
    client, err := gai.NewClient()
    if err != nil {
        log.Fatal(err)
    }
    
    // Simple completion
    parts := gai.NewLLMCallParts().
        WithProvider("openai").
        WithModel("gpt-4o").
        WithUserMessage("What's the capital of France?")
    
    response, err := client.GetCompletion(context.Background(), parts)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response.Content)
}
```

### Type-Safe Structured Outputs

```go
// Define your response structure
type CityInfo struct {
    Name       string   `json:"name" desc:"City name"`
    Country    string   `json:"country" desc:"Country name"`
    Population int      `json:"population" desc:"Approximate population"`
    Languages  []string `json:"languages" desc:"Main languages spoken"`
}

// Use Action[T] for type-safe responses
action := gai.NewAction[CityInfo]().
    WithProvider("openai").
    WithModel("gpt-4o").
    WithSystem("You are a helpful geography assistant.").
    WithUserMessage("Tell me about Tokyo")

cityInfo, err := action.Run(context.Background(), client)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("City: %s, Population: %d\n", cityInfo.Name, cityInfo.Population)
```

## 🔧 Configuration

### Environment Variables

Create a `.env` file or set environment variables:

```env
OPENAI_API_KEY=your_openai_key
ANTHROPIC_API_KEY=your_anthropic_key
GEMINI_API_KEY=your_gemini_key
GROQ_API_KEY=your_groq_key
CEREBRAS_API_KEY=your_cerebras_key
```

### Client Options

```go
// Custom configuration
client, err := gai.NewClient(
    gai.WithOpenAIKey("your-api-key"),        // Provide key directly
    gai.WithHTTPTimeout(60*time.Second),      // Custom timeout
    gai.WithoutEnvFile(),                     // Disable .env loading
    gai.WithDefaultProvider("anthropic"),     // Set default provider
)

// Load specific .env file
client, err := gai.NewClient(
    gai.WithEnvFile("/path/to/.env"),
)
```

## 🎯 Advanced Features

### Conversation Management

```go
// Build a conversation
conv := gai.NewLLMCallParts().
    WithSystem("You are a helpful assistant.").
    WithUserMessage("What's Python?").
    WithAssistantMessage("Python is a programming language...").
    WithUserMessage("What are its main uses?")

// Manage conversation history
conv.KeepLastMessages(10)                    // Keep only last 10 messages
lastUser, index := conv.FindLastMessage("user")  // Find messages
filtered := conv.FilterMessages(func(m gai.Message) bool {
    return m.Role == "user"
})
```

### Prompt Templates

```go
tmpl, err := gai.NewPromptTemplate(`
Analyze the {{.Language}} code in {{.Filename}}:
- Check for bugs
- Suggest improvements
- Rate code quality (1-10)
`)

parts := gai.NewLLMCallParts()
err = gai.RenderSystemTemplate(parts, tmpl, map[string]interface{}{
    "Language": "Go",
    "Filename": "main.go",
})
```

### Error Handling

```go
response, err := client.GetCompletion(ctx, parts)
if err != nil {
    // Structured error information
    if llmErr, ok := err.(*gai.LLMError); ok {
        fmt.Printf("Provider: %s\n", llmErr.Provider)
        fmt.Printf("Model: %s\n", llmErr.Model)
        fmt.Printf("Status Code: %d\n", llmErr.StatusCode)
        fmt.Printf("Raw Response: %s\n", llmErr.LastRaw)
    }
}
```

### Token Management

```go
// Estimate tokens
tokenizer := gai.NewSimpleTokenizer()
tokens := parts.EstimateTokens(tokenizer)

// Prune to fit context window
removed, err := parts.PruneToTokens(4000, tokenizer)

// Keep recent messages while pruning
removed, err := parts.PruneKeepingRecent(5, 4000, tokenizer)
```

## 📚 Providers

### Supported Models

| Provider | Example Models | Configuration |
|----------|---------------|---------------|
| OpenAI | gpt-4o, gpt-4o-mini, gpt-3.5-turbo | `WithProvider("openai")` |
| Anthropic | claude-3-sonnet, claude-3-haiku | `WithProvider("anthropic")` |
| Google | gemini-2.0-flash-exp, gemini-pro | `WithProvider("gemini")` |
| Groq | llama-3.3-70b, mixtral-8x7b | `WithProvider("groq")` |
| Cerebras | llama-3.3-70b | `WithProvider("cerebras")` |

## 🏗️ Architecture

### Core Components

- **LLMClient**: Main interface for all LLM operations
- **LLMCallParts**: Request configuration with fluent builder methods
- **Action[T]**: Generic wrapper for type-safe structured outputs
- **Response Parser**: Three-stage pipeline for robust JSON parsing

### Package Structure

```
github.com/collinshill/gai/
├── core_types.go          # Core types (Message, LLMResponse, etc.)
├── llm_client.go          # Client implementation
├── action.go              # Generic Action[T] pattern
├── providers/             # Provider implementations
│   ├── openai.go
│   ├── anthropic.go
│   └── ...
└── responseParser/        # Robust JSON parsing
    ├── cleanup/           # Markdown/text preprocessing
    ├── parser/            # JSON parsing with error recovery
    └── coercer/          # Type coercion and mapping
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Response parsing strategies inspired by [BAML](https://github.com/baml-ai/llmjson)
- Built with love for the Go community

## 📖 More Examples

Check out the [examples](examples/) directory for more detailed examples:

- [Basic completions](examples/01_simple_completion.go)
- [Structured outputs](examples/02_structured_output.go)
- [Conversation management](examples/03_conversation.go)
- [Error handling](examples/04_error_handling.go)
- [Template usage](examples/05_templates.go)