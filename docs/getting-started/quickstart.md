# Quick Start Tutorial

Build your first AI application with GAI in just 5 minutes! This tutorial will guide you through creating a simple but powerful AI assistant that can handle text generation, streaming responses, and tool calling.

## What We'll Build

In this tutorial, we'll create:
1. A simple text generation example
2. A streaming chat application
3. An AI assistant with tool calling
4. A structured data extractor

By the end, you'll understand the core concepts and be ready to build your own AI applications.

## Prerequisites

Before starting, ensure you have:
- GAI installed ([Installation Guide](./installation.md))
- An API key for at least one provider (OpenAI, Anthropic, or Gemini)
- Basic Go knowledge

## Part 1: Your First AI Request

Let's start with the simplest possible example - generating text with AI.

### Step 1: Create a New File

Create a new file called `main.go`:

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
    // Initialize the OpenAI provider with your API key
    provider := openai.New(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-3.5-turbo"), // Using GPT-3.5 for cost efficiency
    )
    
    // Create a context for the request
    ctx := context.Background()
    
    // Create a request with a simple user message
    request := core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a short poem about Go programming"},
                },
            },
        },
        MaxTokens:   100,  // Limit response length
        Temperature: 0.7,  // Add some creativity
    }
    
    // Generate the response
    response, err := provider.GenerateText(ctx, request)
    if err != nil {
        log.Fatalf("Error generating text: %v", err)
    }
    
    // Print the response
    fmt.Println("AI Response:")
    fmt.Println(response.Text)
    
    // Print token usage
    fmt.Printf("\nTokens used: %d input, %d output, %d total\n",
        response.Usage.InputTokens,
        response.Usage.OutputTokens,
        response.Usage.TotalTokens,
    )
}
```

### Step 2: Run the Application

```bash
# Set your API key
export OPENAI_API_KEY="sk-..."

# Run the application
go run main.go
```

**Expected Output:**
```
AI Response:
In Go we write with grace and speed,
Concurrent code that meets each need.
With channels flowing, goroutines dance,
Simple syntax, no inheritance.

Tokens used: 18 input, 35 output, 53 total
```

### Understanding the Code

Let's break down what happened:

1. **Provider Creation**: We created an OpenAI provider with our API key
2. **Request Structure**: We built a request with messages containing user input
3. **Generation**: We called `GenerateText` to get a response
4. **Response**: We received structured output with text and usage information

## Part 2: Streaming Responses

Now let's modify our application to stream responses in real-time, just like ChatGPT!

### Step 1: Create a Streaming Example

Create `streaming.go`:

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
        openai.WithModel("gpt-4"),
    )
    
    // Create a request for streaming
    ctx := context.Background()
    request := core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are a helpful assistant that explains things clearly and concisely."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Explain how neural networks work in simple terms"},
                },
            },
        },
        Stream: true,  // Enable streaming
    }
    
    // Start streaming
    stream, err := provider.StreamText(ctx, request)
    if err != nil {
        log.Fatalf("Error starting stream: %v", err)
    }
    defer stream.Close()
    
    fmt.Println("AI Response (streaming):")
    fmt.Println("------------------------")
    
    // Process stream events
    var fullText string
    for event := range stream.Events() {
        switch event.Type {
        case core.EventTextDelta:
            // Print each text chunk as it arrives
            fmt.Print(event.TextDelta)
            fullText += event.TextDelta
            
        case core.EventFinish:
            // Stream completed
            fmt.Println("\n------------------------")
            fmt.Println("Stream completed!")
            
        case core.EventError:
            // Handle any errors
            log.Printf("Stream error: %v", event.Err)
        }
    }
    
    // Print final statistics
    fmt.Printf("Total characters: %d\n", len(fullText))
}
```

### Step 2: Run the Streaming Example

```bash
go run streaming.go
```

You'll see the response appear word by word, creating a smooth user experience!

### Understanding Streaming

Streaming provides several benefits:
- **Better UX**: Users see responses immediately
- **Lower latency**: First token arrives quickly
- **Memory efficiency**: Process large responses without loading all into memory
- **Cancellation**: Can stop generation mid-stream

## Part 3: Building a Conversational AI

Let's create an interactive chat application that maintains conversation history.

### Step 1: Create an Interactive Chat

Create `chat.go`:

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "log"
    "os"
    "strings"
    
    "github.com/yourusername/gai/core"
    "github.com/yourusername/gai/providers/openai"
)

func main() {
    // Initialize provider
    provider := openai.New(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4"),
    )
    
    // Initialize conversation with system message
    messages := []core.Message{
        {
            Role: core.System,
            Parts: []core.Part{
                core.Text{Text: `You are a helpful AI assistant. You are:
- Friendly and conversational
- Concise but thorough
- Able to remember our conversation history
- Honest about what you don't know`},
            },
        },
    }
    
    // Create a scanner for user input
    scanner := bufio.NewScanner(os.Stdin)
    ctx := context.Background()
    
    fmt.Println("🤖 GAI Chat Assistant")
    fmt.Println("Type 'exit' to quit, 'clear' to reset conversation")
    fmt.Println("----------------------------------------")
    
    for {
        // Get user input
        fmt.Print("\nYou: ")
        if !scanner.Scan() {
            break
        }
        
        input := strings.TrimSpace(scanner.Text())
        
        // Handle special commands
        if input == "exit" {
            fmt.Println("Goodbye! 👋")
            break
        }
        
        if input == "clear" {
            messages = messages[:1] // Keep only system message
            fmt.Println("Conversation cleared.")
            continue
        }
        
        // Add user message to history
        messages = append(messages, core.Message{
            Role: core.User,
            Parts: []core.Part{
                core.Text{Text: input},
            },
        })
        
        // Generate response with streaming
        request := core.Request{
            Messages: messages,
            Stream:   true,
        }
        
        stream, err := provider.StreamText(ctx, request)
        if err != nil {
            log.Printf("Error: %v", err)
            continue
        }
        
        fmt.Print("AI: ")
        var assistantResponse string
        
        // Stream the response
        for event := range stream.Events() {
            switch event.Type {
            case core.EventTextDelta:
                fmt.Print(event.TextDelta)
                assistantResponse += event.TextDelta
            case core.EventError:
                log.Printf("\nError: %v", event.Err)
            }
        }
        stream.Close()
        
        // Add assistant response to history
        messages = append(messages, core.Message{
            Role: core.Assistant,
            Parts: []core.Part{
                core.Text{Text: assistantResponse},
            },
        })
        
        // Show token count
        fmt.Printf("\n[Conversation: %d messages]", len(messages))
    }
}
```

### Step 2: Have a Conversation

```bash
go run chat.go
```

**Example Interaction:**
```
🤖 GAI Chat Assistant
Type 'exit' to quit, 'clear' to reset conversation
----------------------------------------

You: Hello! What can you help me with?
AI: Hello! I can help you with a wide variety of tasks, including:

• Answering questions on various topics
• Writing and editing text
• Explaining complex concepts
• Problem-solving and analysis
• Creative tasks like storytelling or brainstorming
• Programming help and code review
• Math and calculations
• General conversation and advice

What would you like to explore today?
[Conversation: 3 messages]

You: Can you remember what I just asked?
AI: Yes! You just asked me what I can help you with, and I provided a list of various tasks I can assist with, including answering questions, writing, explaining concepts, problem-solving, programming help, and more. I can maintain context throughout our conversation.
[Conversation: 5 messages]
```

## Part 4: Adding Tool Calling

Now let's give our AI the ability to use tools - functions it can call to get real information or perform actions.

### Step 1: Create an AI with Tools

Create `tools.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "math/rand"
    "os"
    "time"
    
    "github.com/yourusername/gai/core"
    "github.com/yourusername/gai/providers/openai"
    "github.com/yourusername/gai/tools"
)

// Define tool input/output structures
type WeatherInput struct {
    Location string `json:"location" jsonschema:"required,description=City name or location"`
    Unit     string `json:"unit,omitempty" jsonschema:"enum=celsius,enum=fahrenheit,description=Temperature unit"`
}

type WeatherOutput struct {
    Temperature float64 `json:"temperature"`
    Condition   string  `json:"condition"`
    Unit        string  `json:"unit"`
    Humidity    int     `json:"humidity"`
}

type TimeInput struct {
    Timezone string `json:"timezone,omitempty" jsonschema:"description=Timezone (e.g., America/New_York)"`
}

type TimeOutput struct {
    CurrentTime string `json:"current_time"`
    Timezone    string `json:"timezone"`
}

func main() {
    // Create provider
    provider := openai.New(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4"),
    )
    
    // Define the weather tool
    weatherTool := tools.New[WeatherInput, WeatherOutput](
        "get_weather",
        "Get current weather for a location",
        func(ctx context.Context, input WeatherInput, meta tools.Meta) (WeatherOutput, error) {
            // In real app, call weather API
            // This is a mock implementation
            fmt.Printf("\n[Tool Called: get_weather(%s)]\n", input.Location)
            
            // Simulate API call
            time.Sleep(500 * time.Millisecond)
            
            // Mock weather data
            conditions := []string{"Sunny", "Cloudy", "Rainy", "Partly Cloudy"}
            unit := input.Unit
            if unit == "" {
                unit = "celsius"
            }
            
            temp := 15.0 + rand.Float64()*20 // 15-35°C
            if unit == "fahrenheit" {
                temp = temp*9/5 + 32
            }
            
            return WeatherOutput{
                Temperature: temp,
                Condition:   conditions[rand.Intn(len(conditions))],
                Unit:        unit,
                Humidity:    40 + rand.Intn(40), // 40-80%
            }, nil
        },
    )
    
    // Define the time tool
    timeTool := tools.New[TimeInput, TimeOutput](
        "get_time",
        "Get current time in a specific timezone",
        func(ctx context.Context, input TimeInput, meta tools.Meta) (TimeOutput, error) {
            fmt.Printf("\n[Tool Called: get_time(%s)]\n", input.Timezone)
            
            tz := input.Timezone
            if tz == "" {
                tz = "UTC"
            }
            
            loc, err := time.LoadLocation(tz)
            if err != nil {
                loc = time.UTC
                tz = "UTC"
            }
            
            currentTime := time.Now().In(loc).Format("2006-01-02 15:04:05 MST")
            
            return TimeOutput{
                CurrentTime: currentTime,
                Timezone:    tz,
            }, nil
        },
    )
    
    // Create request with tools
    ctx := context.Background()
    request := core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "You are a helpful assistant with access to weather and time information."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "What's the weather like in Tokyo and what time is it there?"},
                },
            },
        },
        Tools: []tools.Handle{weatherTool, timeTool},
        ToolChoice: core.ToolAuto, // Let AI decide which tools to use
    }
    
    // Execute request with tool calling
    fmt.Println("User: What's the weather like in Tokyo and what time is it there?")
    fmt.Println("\nProcessing with tools...")
    
    response, err := provider.GenerateText(ctx, request)
    if err != nil {
        log.Fatalf("Error: %v", err)
    }
    
    fmt.Println("\nAI Response:")
    fmt.Println(response.Text)
    
    // Show execution steps
    if len(response.Steps) > 0 {
        fmt.Println("\n--- Execution Steps ---")
        for i, step := range response.Steps {
            fmt.Printf("Step %d:\n", i+1)
            if len(step.ToolCalls) > 0 {
                fmt.Println("  Tool Calls:")
                for _, call := range step.ToolCalls {
                    fmt.Printf("    - %s\n", call.Name)
                }
            }
            if step.Text != "" {
                fmt.Printf("  Response: %s\n", step.Text)
            }
        }
    }
}
```

### Step 2: Run the Tool Example

```bash
go run tools.go
```

**Expected Output:**
```
User: What's the weather like in Tokyo and what time is it there?

Processing with tools...

[Tool Called: get_weather(Tokyo)]

[Tool Called: get_time(Asia/Tokyo)]

AI Response:
Based on the current information:

**Weather in Tokyo:**
- Temperature: 22.5°C
- Condition: Partly Cloudy
- Humidity: 65%

**Current Time in Tokyo:**
- Time: 2024-03-15 14:30:45 JST
- Timezone: Asia/Tokyo

It's a pleasant partly cloudy afternoon in Tokyo with comfortable temperature around 22-23°C.

--- Execution Steps ---
Step 1:
  Tool Calls:
    - get_weather
    - get_time
Step 2:
  Response: Based on the current information...
```

## Part 5: Structured Output

Let's extract structured data from unstructured text - perfect for data processing applications.

### Step 1: Create a Structured Output Example

Create `structured.go`:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    
    "github.com/yourusername/gai/core"
    "github.com/yourusername/gai/providers/openai"
)

// Define the structure we want to extract
type ProductReview struct {
    ProductName string   `json:"product_name"`
    Rating      int      `json:"rating" jsonschema:"minimum=1,maximum=5"`
    Pros        []string `json:"pros"`
    Cons        []string `json:"cons"`
    Summary     string   `json:"summary"`
    Recommended bool     `json:"recommended"`
}

type ContactInfo struct {
    Name    string `json:"name"`
    Email   string `json:"email,omitempty"`
    Phone   string `json:"phone,omitempty"`
    Company string `json:"company,omitempty"`
    Role    string `json:"role,omitempty"`
}

func main() {
    // Create provider
    provider := openai.New(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4"),
    )
    
    ctx := context.Background()
    
    // Example 1: Extract product review
    fmt.Println("Example 1: Extracting Product Review")
    fmt.Println("=====================================")
    
    reviewText := `
    I recently purchased the TechPro X1 Wireless Headphones and I'm mostly satisfied.
    The sound quality is exceptional, especially the bass response. The battery life
    easily lasts 30+ hours as advertised. The comfort is great for long listening sessions.
    However, the price is quite high at $350, and the app could use some improvements.
    The noise cancellation sometimes struggles with wind noise. Overall, I'd give it
    4 out of 5 stars and would recommend it to audiophiles who can afford it.
    `
    
    reviewRequest := core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "Extract structured information from the provided text."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: reviewText},
                },
            },
        },
    }
    
    // Generate structured output
    result, err := provider.GenerateObject[ProductReview](ctx, reviewRequest)
    if err != nil {
        log.Fatalf("Error: %v", err)
    }
    
    // Print the structured data
    fmt.Println("Extracted Review:")
    prettyPrint(result.Value)
    
    // Example 2: Extract contact information
    fmt.Println("\nExample 2: Extracting Contact Information")
    fmt.Println("==========================================")
    
    emailText := `
    Hi there,
    
    My name is John Smith and I'm the CTO at TechCorp. I'm interested in
    discussing your AI solutions for our upcoming project. You can reach
    me at john.smith@techcorp.com or call me at (555) 123-4567.
    
    Best regards,
    John
    `
    
    contactRequest := core.Request{
        Messages: []core.Message{
            {
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "Extract contact information from the email."},
                },
            },
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: emailText},
                },
            },
        },
    }
    
    // Generate structured output
    contactResult, err := provider.GenerateObject[ContactInfo](ctx, contactRequest)
    if err != nil {
        log.Fatalf("Error: %v", err)
    }
    
    fmt.Println("Extracted Contact:")
    prettyPrint(contactResult.Value)
}

func prettyPrint(v interface{}) {
    b, _ := json.MarshalIndent(v, "", "  ")
    fmt.Println(string(b))
}
```

### Step 2: Run the Structured Output Example

```bash
go run structured.go
```

**Expected Output:**
```
Example 1: Extracting Product Review
=====================================
Extracted Review:
{
  "product_name": "TechPro X1 Wireless Headphones",
  "rating": 4,
  "pros": [
    "Exceptional sound quality",
    "Great bass response",
    "30+ hours battery life",
    "Comfortable for long sessions"
  ],
  "cons": [
    "High price at $350",
    "App needs improvements",
    "Noise cancellation struggles with wind"
  ],
  "summary": "High-quality wireless headphones with excellent sound and battery life, but expensive",
  "recommended": true
}

Example 2: Extracting Contact Information
==========================================
Extracted Contact:
{
  "name": "John Smith",
  "email": "john.smith@techcorp.com",
  "phone": "(555) 123-4567",
  "company": "TechCorp",
  "role": "CTO"
}
```

## Part 6: Switching Providers

One of GAI's superpowers is the ability to switch providers without changing your code.

### Step 1: Create a Multi-Provider Example

Create `providers.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"
    
    "github.com/yourusername/gai/core"
    "github.com/yourusername/gai/providers/anthropic"
    "github.com/yourusername/gai/providers/gemini"
    "github.com/yourusername/gai/providers/ollama"
    "github.com/yourusername/gai/providers/openai"
)

func main() {
    ctx := context.Background()
    
    // The same request for all providers
    request := core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a haiku about artificial intelligence"},
                },
            },
        },
        MaxTokens:   100,
        Temperature: 0.7,
    }
    
    // Create different providers
    providers := map[string]core.Provider{
        "OpenAI": openai.New(
            openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
            openai.WithModel("gpt-3.5-turbo"),
        ),
        "Anthropic": anthropic.New(
            anthropic.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
            anthropic.WithModel("claude-3-sonnet-20240229"),
        ),
        "Gemini": gemini.New(
            gemini.WithAPIKey(os.Getenv("GOOGLE_API_KEY")),
            gemini.WithModel("gemini-1.5-flash"),
        ),
        "Ollama (Local)": ollama.New(
            ollama.WithBaseURL("http://localhost:11434"),
            ollama.WithModel("llama3.2"),
        ),
    }
    
    // Test each provider with the same request
    fmt.Println("🎭 Multi-Provider Haiku Generation")
    fmt.Println("===================================\n")
    
    for name, provider := range providers {
        fmt.Printf("Provider: %s\n", name)
        fmt.Println("-------------------")
        
        start := time.Now()
        response, err := provider.GenerateText(ctx, request)
        elapsed := time.Since(start)
        
        if err != nil {
            fmt.Printf("❌ Error: %v\n\n", err)
            continue
        }
        
        fmt.Println(response.Text)
        fmt.Printf("⏱️  Time: %v\n", elapsed.Round(time.Millisecond))
        fmt.Printf("📊 Tokens: %d\n\n", response.Usage.TotalTokens)
    }
}
```

### Step 2: Run the Multi-Provider Example

```bash
# Set all API keys
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export GOOGLE_API_KEY="..."

# Start Ollama if using local models
ollama serve

# Run the example
go run providers.go
```

**Expected Output:**
```
🎭 Multi-Provider Haiku Generation
===================================

Provider: OpenAI
-------------------
Silicon minds wake,
Patterns dance in neural paths,
Future thinks itself.
⏱️  Time: 523ms
📊 Tokens: 28

Provider: Anthropic
-------------------
Circuits awakening
Algorithms dream and learn
Mind without neurons
⏱️  Time: 892ms
📊 Tokens: 25

Provider: Gemini
-------------------
Data flows like streams,
Neural networks learn and grow,
Machine minds awake.
⏱️  Time: 412ms
📊 Tokens: 24

Provider: Ollama (Local)
-------------------
Metal thoughts arise
Silicon dreams computing
Future minds emerge
⏱️  Time: 1.2s
📊 Tokens: 22
```

## Summary and Next Steps

Congratulations! 🎉 You've just built several AI applications with GAI:

1. ✅ **Text Generation** - Simple requests and responses
2. ✅ **Streaming** - Real-time response streaming
3. ✅ **Chat Application** - Interactive conversations with history
4. ✅ **Tool Calling** - AI that can use functions
5. ✅ **Structured Output** - Type-safe data extraction
6. ✅ **Provider Switching** - Same code, different providers

### Key Concepts You've Learned

- **Providers**: Abstraction over different AI services
- **Messages**: How to structure conversations
- **Streaming**: Real-time response handling
- **Tools**: Giving AI the ability to call functions
- **Structured Output**: Getting typed, validated responses
- **Provider Agnostic**: Write once, run with any provider

### Where to Go Next

Now that you understand the basics, explore:

1. **[Core Concepts](../core-concepts/architecture.md)** - Deep dive into GAI's architecture
2. **[Provider Guides](../providers/)** - Learn provider-specific features
3. **[Advanced Features](../features/)** - Middleware, observability, audio
4. **[Tutorials](../tutorials/)** - Build complete applications
5. **[API Reference](../api-reference/)** - Detailed API documentation

### Quick Reference

Here's a handy reference for common patterns:

```go
// Text generation
response, err := provider.GenerateText(ctx, request)

// Streaming
stream, err := provider.StreamText(ctx, request)
for event := range stream.Events() { ... }

// Structured output
result, err := provider.GenerateObject[MyType](ctx, request)

// Tool calling
tool := tools.New[Input, Output](name, description, handler)
request.Tools = []tools.Handle{tool}

// Provider switching
provider := openai.New(...)      // or
provider := anthropic.New(...)   // or
provider := gemini.New(...)      // same API!
```

### Tips for Success

1. **Start Simple**: Begin with text generation before adding complexity
2. **Use Streaming**: For better UX in interactive applications
3. **Handle Errors**: Always check and handle errors appropriately
4. **Monitor Usage**: Track token usage to manage costs
5. **Test Providers**: Different providers have different strengths
6. **Read the Docs**: Our comprehensive documentation has many more examples

### Get Help

- 📖 [Documentation](../)
- 💬 [Discord Community](https://discord.gg/gai)
- 🐛 [GitHub Issues](https://github.com/yourusername/gai/issues)
- 📧 [Email Support](mailto:support@gai.dev)

---

**You're now ready to build amazing AI applications with GAI! 🚀**

Happy coding, and welcome to the GAI community!