# Messages and Parts

This guide provides a comprehensive understanding of GAI's message system, which forms the foundation for all AI interactions. You'll learn how to construct conversations, work with multimodal content, and leverage the type-safe message system effectively.

## Table of Contents
- [Overview](#overview)
- [Message Structure](#message-structure)
- [Roles](#roles)
- [Parts System](#parts-system)
- [Text Content](#text-content)
- [Images](#images)
- [Audio Content](#audio-content)
- [Video Content](#video-content)
- [File Attachments](#file-attachments)
- [BlobRef System](#blobref-system)
- [Conversation Patterns](#conversation-patterns)
- [Best Practices](#best-practices)

## Overview

Messages are the fundamental unit of communication in GAI. They represent the conversation between users, AI assistants, systems, and tools. GAI's message system is designed to be:

- **Multimodal**: Support text, images, audio, video, and files in any combination
- **Type-Safe**: Compile-time checking prevents runtime errors
- **Provider-Agnostic**: Same message format works with all providers
- **Extensible**: New content types can be added without breaking existing code

## Message Structure

A message in GAI consists of three main components:

```go
type Message struct {
    Role  Role      // Who is speaking (System, User, Assistant, Tool)
    Parts []Part    // Content parts (can be mixed media)
    Name  string    // Optional: Named participant (for multi-agent scenarios)
}
```

### Basic Message Creation

Here's how to create different types of messages:

```go
// Simple text message from user
userMessage := core.Message{
    Role: core.User,
    Parts: []core.Part{
        core.Text{Text: "Hello, how are you?"},
    },
}

// System message setting context
systemMessage := core.Message{
    Role: core.System,
    Parts: []core.Part{
        core.Text{Text: "You are a helpful assistant specializing in Go programming."},
    },
}

// Assistant response
assistantMessage := core.Message{
    Role: core.Assistant,
    Parts: []core.Part{
        core.Text{Text: "I'm doing well! I'd be happy to help with your Go programming questions."},
    },
}

// Tool result message
toolMessage := core.Message{
    Role: core.Tool,
    Parts: []core.Part{
        core.Text{Text: `{"temperature": 72, "condition": "sunny"}`},
    },
    Name: "weather_tool", // Identifies which tool
}
```

### Multimodal Messages

Messages can contain multiple parts of different types:

```go
// Message with text and image
multimodalMessage := core.Message{
    Role: core.User,
    Parts: []core.Part{
        core.Text{Text: "What's in this image?"},
        core.ImageURL{URL: "https://example.com/photo.jpg"},
    },
}

// Message with multiple images and text
analysisRequest := core.Message{
    Role: core.User,
    Parts: []core.Part{
        core.Text{Text: "Compare these two designs:"},
        core.ImageURL{URL: "https://example.com/design1.png"},
        core.Text{Text: "versus"},
        core.ImageURL{URL: "https://example.com/design2.png"},
        core.Text{Text: "Which one has better user experience?"},
    },
}
```

## Roles

GAI defines four primary roles for messages:

### System Role

The System role provides instructions and context to the AI:

```go
systemMessage := core.Message{
    Role: core.System,
    Parts: []core.Part{
        core.Text{Text: `You are an expert software architect with 20 years of experience.
You specialize in:
- Distributed systems
- Microservices architecture
- Cloud-native applications
- Performance optimization

Always provide practical, production-ready advice with code examples.`},
    },
}
```

**Best Practices for System Messages:**
- Place at the beginning of the conversation
- Be specific about the AI's role and expertise
- Include formatting preferences
- Set behavioral guidelines
- Specify any constraints or rules

### User Role

The User role represents input from the human user:

```go
// Simple question
userQuestion := core.Message{
    Role: core.User,
    Parts: []core.Part{
        core.Text{Text: "How do I implement a circuit breaker pattern in Go?"},
    },
}

// Complex request with context
userRequest := core.Message{
    Role: core.User,
    Parts: []core.Part{
        core.Text{Text: "I'm building a microservices system with the following requirements:"},
        core.Text{Text: "- 100k requests per second\n- 99.99% uptime\n- Global distribution"},
        core.Text{Text: "What architecture would you recommend?"},
    },
}
```

### Assistant Role

The Assistant role represents AI responses:

```go
assistantResponse := core.Message{
    Role: core.Assistant,
    Parts: []core.Part{
        core.Text{Text: "I'll help you implement a circuit breaker pattern. Here's a production-ready implementation:"},
        core.Text{Text: "```go\ntype CircuitBreaker struct {\n    // implementation\n}\n```"},
    },
}
```

### Tool Role

The Tool role represents results from function calls:

```go
// Tool execution result
toolResult := core.Message{
    Role: core.Tool,
    Name: "database_query", // Which tool produced this
    Parts: []core.Part{
        core.Text{Text: `{
            "results": [
                {"id": 1, "name": "Alice", "score": 95},
                {"id": 2, "name": "Bob", "score": 87}
            ],
            "count": 2
        }`},
    },
}
```

## Parts System

Parts are the building blocks of message content. GAI uses a sealed interface pattern for type safety:

```go
type Part interface {
    isPart()      // Sealed interface - can't be implemented outside package
    partType() string // Returns the type identifier
}
```

### Why Parts?

The parts system enables:
1. **Multimodal Content**: Mix different media types in one message
2. **Type Safety**: Compiler ensures only valid part types are used
3. **Provider Flexibility**: Providers can support different part types
4. **Future Extensibility**: New part types can be added

## Text Content

Text is the most common part type:

```go
type Text struct {
    Text string `json:"text"`
}
```

### Basic Text Usage

```go
// Simple text
simple := core.Text{Text: "Hello, world!"}

// Formatted text
formatted := core.Text{Text: `
# Markdown Heading

- Bullet point 1
- Bullet point 2

**Bold text** and *italic text*
`}

// Code blocks
code := core.Text{Text: "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"}
```

### Text Best Practices

```go
// Good: Clear, structured text
goodMessage := core.Message{
    Role: core.User,
    Parts: []core.Part{
        core.Text{Text: "Please analyze this Go code for potential issues:"},
        core.Text{Text: "```go\n" + codeContent + "\n```"},
        core.Text{Text: "Focus on: 1) Race conditions 2) Error handling 3) Performance"},
    },
}

// Bad: Unstructured text
badMessage := core.Message{
    Role: core.User,
    Parts: []core.Part{
        core.Text{Text: "analyze this " + codeContent + " for issues especially race conditions error handling performance"},
    },
}
```

## Images

GAI supports images through URL references or inline data:

```go
type ImageURL struct {
    URL    string `json:"url"`
    Detail string `json:"detail,omitempty"` // "low", "high", or "auto"
}
```

### Image URL Usage

```go
// Basic image URL
image := core.ImageURL{
    URL: "https://example.com/diagram.png",
}

// High-detail image analysis
detailedImage := core.ImageURL{
    URL:    "https://example.com/complex-chart.png",
    Detail: "high", // Request higher resolution processing
}

// Creating a message with images
imageMessage := core.Message{
    Role: core.User,
    Parts: []core.Part{
        core.Text{Text: "Analyze this architecture diagram:"},
        core.ImageURL{URL: "https://example.com/architecture.png", Detail: "high"},
        core.Text{Text: "What are the potential bottlenecks?"},
    },
}
```

### Working with Local Images

For local images, you can:
1. Serve them via HTTP
2. Convert to base64 data URLs
3. Upload to provider's file API

```go
// Base64 data URL approach
import (
    "encoding/base64"
    "fmt"
    "os"
)

func loadLocalImage(path string) (core.ImageURL, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return core.ImageURL{}, err
    }
    
    b64 := base64.StdEncoding.EncodeToString(data)
    dataURL := fmt.Sprintf("data:image/png;base64,%s", b64)
    
    return core.ImageURL{URL: dataURL}, nil
}

// Usage
localImg, err := loadLocalImage("./diagram.png")
if err != nil {
    log.Fatal(err)
}

message := core.Message{
    Role: core.User,
    Parts: []core.Part{
        core.Text{Text: "What's in this image?"},
        localImg,
    },
}
```

### Image Best Practices

```go
// Good: Specific image analysis request
goodImageRequest := core.Message{
    Role: core.User,
    Parts: []core.Part{
        core.Text{Text: "Analyze this UML diagram and identify:"},
        core.ImageURL{URL: imageURL, Detail: "high"},
        core.Text{Text: "1. Design patterns used\n2. Potential coupling issues\n3. Missing components"},
    },
}

// Consider image size and costs
func optimizeImageForAnalysis(imageURL string) core.ImageURL {
    // Use "low" detail for simple queries to save tokens/cost
    if isSimpleQuery {
        return core.ImageURL{URL: imageURL, Detail: "low"}
    }
    // Use "high" for complex analysis
    return core.ImageURL{URL: imageURL, Detail: "high"}
}
```

## Audio Content

Audio support enables speech processing and audio analysis:

```go
type Audio struct {
    Source     BlobRef `json:"source"`
    SampleRate int     `json:"sample_rate,omitempty"`
    Channels   int     `json:"channels,omitempty"`
    Duration   float64 `json:"duration_seconds,omitempty"`
}
```

### Audio Usage Examples

```go
// Audio from URL
audioFromURL := core.Audio{
    Source: core.BlobRef{
        Kind: core.BlobURL,
        URL:  "https://example.com/speech.mp3",
        MIME: "audio/mp3",
    },
}

// Audio from bytes
audioFromBytes := core.Audio{
    Source: core.BlobRef{
        Kind:  core.BlobBytes,
        Bytes: audioData,
        MIME:  "audio/wav",
    },
    SampleRate: 44100,
    Channels:   2,
    Duration:   30.5,
}

// Creating an audio transcription request
transcriptionRequest := core.Message{
    Role: core.User,
    Parts: []core.Part{
        core.Text{Text: "Please transcribe this audio and identify the speakers:"},
        audioFromURL,
    },
}
```

### Audio Processing Patterns

```go
// Pattern 1: Transcription with analysis
func createTranscriptionRequest(audioURL string) core.Message {
    return core.Message{
        Role: core.User,
        Parts: []core.Part{
            core.Text{Text: "Transcribe this audio and provide:"},
            core.Audio{
                Source: core.BlobRef{
                    Kind: core.BlobURL,
                    URL:  audioURL,
                    MIME: "audio/mp3",
                },
            },
            core.Text{Text: "1. Full transcript\n2. Speaker identification\n3. Key topics discussed\n4. Sentiment analysis"},
        },
    }
}

// Pattern 2: Audio comparison
func compareAudioFiles(audio1, audio2 string) core.Message {
    return core.Message{
        Role: core.User,
        Parts: []core.Part{
            core.Text{Text: "Compare these two audio recordings:"},
            core.Audio{Source: core.BlobRef{Kind: core.BlobURL, URL: audio1}},
            core.Text{Text: "versus"},
            core.Audio{Source: core.BlobRef{Kind: core.BlobURL, URL: audio2}},
            core.Text{Text: "Identify differences in: tone, content, speaker emotion"},
        },
    }
}
```

## Video Content

Video support for providers that handle video analysis:

```go
type Video struct {
    Source   BlobRef `json:"source"`
    Duration float64 `json:"duration_seconds,omitempty"`
    Width    int     `json:"width,omitempty"`
    Height   int     `json:"height,omitempty"`
}
```

### Video Usage

```go
// Video from URL
video := core.Video{
    Source: core.BlobRef{
        Kind: core.BlobURL,
        URL:  "https://example.com/demo.mp4",
        MIME: "video/mp4",
    },
    Duration: 120.5,
    Width:    1920,
    Height:   1080,
}

// Video analysis request
videoAnalysis := core.Message{
    Role: core.User,
    Parts: []core.Part{
        core.Text{Text: "Analyze this product demo video:"},
        video,
        core.Text{Text: "Extract: 1) Key features shown 2) UI/UX issues 3) Timestamps of important moments"},
    },
}
```

## File Attachments

Files allow attaching documents for analysis:

```go
type File struct {
    Source  BlobRef `json:"source"`
    Name    string  `json:"name,omitempty"`
    Purpose string  `json:"purpose,omitempty"`
}
```

### File Usage Examples

```go
// PDF document
pdfDoc := core.File{
    Source: core.BlobRef{
        Kind: core.BlobURL,
        URL:  "https://example.com/report.pdf",
        MIME: "application/pdf",
        Size: 1024000, // 1MB
    },
    Name:    "Q4_Report.pdf",
    Purpose: "analysis",
}

// CSV data file
csvFile := core.File{
    Source: core.BlobRef{
        Kind:  core.BlobBytes,
        Bytes: csvData,
        MIME:  "text/csv",
    },
    Name: "sales_data.csv",
}

// Document analysis request
documentRequest := core.Message{
    Role: core.User,
    Parts: []core.Part{
        core.Text{Text: "Analyze this financial report:"},
        pdfDoc,
        core.Text{Text: "Extract: 1) Key metrics 2) YoY growth 3) Risk factors 4) Future projections"},
    },
}
```

### Working with Multiple Files

```go
// Multi-document analysis
func analyzeDocuments(files []core.File) core.Message {
    parts := []core.Part{
        core.Text{Text: "Compare and analyze these documents:"},
    }
    
    for i, file := range files {
        parts = append(parts, file)
        if i < len(files)-1 {
            parts = append(parts, core.Text{Text: fmt.Sprintf("Document %d: %s", i+1, file.Name)})
        }
    }
    
    parts = append(parts, core.Text{Text: "Provide a comparative analysis highlighting similarities and differences."})
    
    return core.Message{
        Role:  core.User,
        Parts: parts,
    }
}
```

## BlobRef System

BlobRef is GAI's universal system for referencing binary content:

```go
type BlobRef struct {
    Kind   BlobKind `json:"kind"`
    URL    string   `json:"url,omitempty"`
    Bytes  []byte   `json:"bytes,omitempty"`
    FileID string   `json:"file_id,omitempty"`
    MIME   string   `json:"mime,omitempty"`
    Size   int64    `json:"size,omitempty"`
}

type BlobKind uint8
const (
    BlobURL          BlobKind = iota // Reference by URL
    BlobBytes                        // Inline bytes
    BlobProviderFile                 // Provider-specific file ID
)
```

### BlobRef Patterns

```go
// Pattern 1: URL reference (most common)
urlBlob := core.BlobRef{
    Kind: core.BlobURL,
    URL:  "https://example.com/data.json",
    MIME: "application/json",
}

// Pattern 2: Inline small files
inlineBlob := core.BlobRef{
    Kind:  core.BlobBytes,
    Bytes: []byte("Hello, World!"),
    MIME:  "text/plain",
    Size:  13,
}

// Pattern 3: Provider file reference (after upload)
providerBlob := core.BlobRef{
    Kind:   core.BlobProviderFile,
    FileID: "file-abc123", // Provider-specific ID
    MIME:   "image/png",
    Size:   45678,
}
```

### BlobRef Best Practices

```go
// Good: Choose appropriate BlobKind based on size and usage
func createBlobRef(data []byte, url string) core.BlobRef {
    // Use URL for large files
    if len(data) > 1024*1024 { // > 1MB
        return core.BlobRef{
            Kind: core.BlobURL,
            URL:  url,
            MIME: detectMIME(data),
            Size: int64(len(data)),
        }
    }
    
    // Use inline for small files
    return core.BlobRef{
        Kind:  core.BlobBytes,
        Bytes: data,
        MIME:  detectMIME(data),
        Size:  int64(len(data)),
    }
}

// Good: Include metadata for better processing
func createRichBlobRef(filepath string) (core.BlobRef, error) {
    info, err := os.Stat(filepath)
    if err != nil {
        return core.BlobRef{}, err
    }
    
    data, err := os.ReadFile(filepath)
    if err != nil {
        return core.BlobRef{}, err
    }
    
    return core.BlobRef{
        Kind:  core.BlobBytes,
        Bytes: data,
        MIME:  detectMIMEFromFile(filepath),
        Size:  info.Size(),
    }, nil
}
```

## Conversation Patterns

### Basic Conversation Flow

```go
// Building a conversation
conversation := []core.Message{
    // 1. System context
    {
        Role: core.System,
        Parts: []core.Part{
            core.Text{Text: "You are a helpful coding assistant."},
        },
    },
    // 2. User question
    {
        Role: core.User,
        Parts: []core.Part{
            core.Text{Text: "How do I handle errors in Go?"},
        },
    },
    // 3. Assistant response
    {
        Role: core.Assistant,
        Parts: []core.Part{
            core.Text{Text: "In Go, error handling is explicit. Here are the key patterns..."},
        },
    },
    // 4. Follow-up question
    {
        Role: core.User,
        Parts: []core.Part{
            core.Text{Text: "Can you show me an example with custom errors?"},
        },
    },
}
```

### Multi-Turn Conversations

```go
type ConversationManager struct {
    messages []core.Message
    maxTurns int
}

func (cm *ConversationManager) AddUserMessage(text string) {
    cm.messages = append(cm.messages, core.Message{
        Role: core.User,
        Parts: []core.Part{core.Text{Text: text}},
    })
    cm.trimHistory()
}

func (cm *ConversationManager) AddAssistantMessage(text string) {
    cm.messages = append(cm.messages, core.Message{
        Role: core.Assistant,
        Parts: []core.Part{core.Text{Text: text}},
    })
    cm.trimHistory()
}

func (cm *ConversationManager) trimHistory() {
    // Keep system message + last N turns
    if len(cm.messages) > cm.maxTurns*2+1 {
        systemMsg := cm.messages[0]
        recentMsgs := cm.messages[len(cm.messages)-cm.maxTurns*2:]
        cm.messages = append([]core.Message{systemMsg}, recentMsgs...)
    }
}

func (cm *ConversationManager) GetRequest() core.Request {
    return core.Request{
        Messages: cm.messages,
    }
}
```

### Tool-Calling Conversations

```go
// Conversation with tool calls
toolConversation := []core.Message{
    {
        Role: core.System,
        Parts: []core.Part{
            core.Text{Text: "You have access to weather and news tools."},
        },
    },
    {
        Role: core.User,
        Parts: []core.Part{
            core.Text{Text: "What's the weather in Tokyo and what's in the news?"},
        },
    },
    {
        Role: core.Assistant,
        Parts: []core.Part{
            core.Text{Text: "I'll check the weather in Tokyo and get the latest news for you."},
        },
    },
    {
        Role: core.Tool,
        Name: "get_weather",
        Parts: []core.Part{
            core.Text{Text: `{"temperature": 22, "condition": "partly cloudy", "humidity": 65}`},
        },
    },
    {
        Role: core.Tool,
        Name: "get_news",
        Parts: []core.Part{
            core.Text{Text: `{"headlines": ["Tech summit begins", "Market update", "Sports news"]}`},
        },
    },
    {
        Role: core.Assistant,
        Parts: []core.Part{
            core.Text{Text: "Here's the current information:\n\n**Weather in Tokyo:**\n- Temperature: 22°C\n- Partly cloudy\n- Humidity: 65%\n\n**Latest News:**\n1. Tech summit begins\n2. Market update\n3. Sports news"},
        },
    },
}
```

## Best Practices

### 1. Message Ordering

Always maintain proper message order:

```go
// Good: Proper role alternation
goodOrder := []core.Message{
    {Role: core.System, Parts: []core.Part{core.Text{Text: "..."}}},
    {Role: core.User, Parts: []core.Part{core.Text{Text: "..."}}},
    {Role: core.Assistant, Parts: []core.Part{core.Text{Text: "..."}}},
    {Role: core.User, Parts: []core.Part{core.Text{Text: "..."}}},
}

// Bad: Consecutive same roles (except for tool results)
badOrder := []core.Message{
    {Role: core.User, Parts: []core.Part{core.Text{Text: "..."}}},
    {Role: core.User, Parts: []core.Part{core.Text{Text: "..."}}}, // Bad!
}
```

### 2. Content Structure

Structure content for clarity:

```go
// Good: Well-structured multimodal message
wellStructured := core.Message{
    Role: core.User,
    Parts: []core.Part{
        core.Text{Text: "Please analyze this data:"},
        core.File{Source: dataFile, Name: "data.csv"},
        core.Text{Text: "And compare with this chart:"},
        core.ImageURL{URL: chartURL},
        core.Text{Text: "Focus on: trends, anomalies, and predictions"},
    },
}

// Good: Clear separation of concerns
func createAnalysisRequest(code string, tests string) core.Message {
    return core.Message{
        Role: core.User,
        Parts: []core.Part{
            core.Text{Text: "Review this code:"},
            core.Text{Text: "```go\n" + code + "\n```"},
            core.Text{Text: "And its tests:"},
            core.Text{Text: "```go\n" + tests + "\n```"},
            core.Text{Text: "Identify: missing test cases, potential bugs, and improvements"},
        },
    }
}
```

### 3. Token Optimization

Be mindful of token usage:

```go
// Token-efficient message construction
func optimizeMessages(messages []core.Message, maxTokens int) []core.Message {
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
    
    // Trim if needed
    if estimatedTokens > maxTokens {
        // Keep system message and recent messages
        return trimConversation(messages, maxTokens)
    }
    
    return messages
}
```

### 4. Error Context

Include context for better error handling:

```go
// Good: Include context in error scenarios
func createErrorContext(err error, operation string) core.Message {
    return core.Message{
        Role: core.User,
        Parts: []core.Part{
            core.Text{Text: fmt.Sprintf("I encountered an error during %s:", operation)},
            core.Text{Text: fmt.Sprintf("Error: %v", err)},
            core.Text{Text: "Can you help me understand and fix this issue?"},
        },
    }
}
```

### 5. Multimodal Optimization

Optimize multimodal content:

```go
// Good: Optimize images based on use case
func prepareImageForAnalysis(imagePath string, analysisType string) (core.Part, error) {
    switch analysisType {
    case "detailed":
        // High resolution for detailed analysis
        return core.ImageURL{URL: uploadImage(imagePath), Detail: "high"}, nil
    case "quick":
        // Low resolution for quick checks
        return core.ImageURL{URL: uploadImage(imagePath), Detail: "low"}, nil
    default:
        // Auto for general use
        return core.ImageURL{URL: uploadImage(imagePath), Detail: "auto"}, nil
    }
}
```

## Summary

The message system in GAI provides:

1. **Flexibility**: Support for any combination of content types
2. **Type Safety**: Compile-time checking prevents errors
3. **Provider Agnostic**: Same format works everywhere
4. **Multimodal**: Native support for text, images, audio, video, and files
5. **Extensible**: New content types can be added

Key takeaways:
- Use appropriate roles for different participants
- Leverage multimodal capabilities when needed
- Structure messages for clarity and efficiency
- Optimize for token usage
- Maintain proper conversation flow

Next, explore:
- [Providers](./providers.md) - Understanding provider abstraction
- [Streaming](./streaming.md) - Real-time response handling
- [Tools](./tools.md) - Function calling with messages