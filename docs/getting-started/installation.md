# Installation Guide

This guide walks you through installing and setting up GAI (Go AI Framework) for your Go project.

## Prerequisites

### System Requirements

- **Go 1.23+** (required for generics and latest language features)
- **Git** (for cloning repositories and version control)
- **Make** (optional, for convenience commands)

### Verify Go Installation

```bash
go version
```

You should see output like:
```
go version go1.23.0 linux/amd64
```

If Go is not installed, download it from [golang.org](https://golang.org/dl/).

## Installation Methods

### Method 1: Go Module (Recommended)

The simplest way to add GAI to your project:

```bash
# Initialize a new Go module (if you haven't already)
go mod init your-project-name

# Add GAI to your project
go get github.com/recera/gai@latest

# Install specific packages as needed
go get github.com/recera/gai/providers/openai@latest
go get github.com/recera/gai/providers/anthropic@latest
go get github.com/recera/gai/providers/groq@latest
go get github.com/recera/gai/tools@latest
go get github.com/recera/gai/middleware@latest
```

### Method 2: Clone and Build from Source

For development or to get the latest features:

```bash
# Clone the repository
git clone https://github.com/recera/gai.git
cd gai

# Install dependencies
go mod download

# Run tests to verify installation
go test ./...

# Build the CLI tool
go build -o ai ./cmd/ai

# Install CLI globally (optional)
go install github.com/recera/gai/cmd/ai@latest
```

### Method 3: Using Go Install (CLI Only)

If you only need the CLI tool:

```bash
# Install the GAI CLI globally
go install github.com/recera/gai/cmd/ai@latest

# Verify installation
ai version
```

## API Key Setup

GAI requires API keys for various providers. Set up the providers you plan to use:

### OpenAI

1. Visit [platform.openai.com](https://platform.openai.com)
2. Create an account or sign in
3. Navigate to [API Keys](https://platform.openai.com/api-keys)
4. Click "Create new secret key"
5. Copy the key (starts with `sk-`)

```bash
export OPENAI_API_KEY="sk-your-key-here"
```

### Anthropic Claude

1. Visit [console.anthropic.com](https://console.anthropic.com)
2. Create an account or sign in
3. Navigate to API Keys
4. Generate a new API key

```bash
export ANTHROPIC_API_KEY="sk-ant-your-key-here"
```

### Google Gemini

1. Visit [ai.google.dev](https://ai.google.dev)
2. Sign in with your Google account
3. Navigate to "Get API key"
4. Create a new API key

```bash
export GOOGLE_API_KEY="AI-your-key-here"
```

### Groq

1. Visit [console.groq.com](https://console.groq.com)
2. Create an account or sign in
3. Navigate to API Keys
4. Generate a new API key

```bash
export GROQ_API_KEY="gsk-your-key-here"
```

### Additional Providers

```bash
# xAI (Grok)
export XAI_API_KEY="xai-your-key-here"

# ElevenLabs (TTS)
export ELEVENLABS_API_KEY="your-key-here"

# Deepgram (STT)  
export DEEPGRAM_API_KEY="your-key-here"
```

### Persistent Environment Variables

Create a `.env` file in your project root:

```bash
# .env file
OPENAI_API_KEY=sk-your-openai-key
ANTHROPIC_API_KEY=sk-ant-your-anthropic-key
GOOGLE_API_KEY=AI-your-google-key
GROQ_API_KEY=gsk-your-groq-key
ELEVENLABS_API_KEY=your-elevenlabs-key
```

**Important**: Add `.env` to your `.gitignore` to avoid committing API keys:

```bash
echo ".env" >> .gitignore
```

## Verification

### Test Installation with Hello World

Create `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "github.com/recera/gai/core"
    "github.com/recera/gai/providers/openai"
)

func main() {
    // Load API key from environment
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("Please set OPENAI_API_KEY environment variable")
    }
    
    // Create provider
    provider := openai.New(
        openai.WithAPIKey(apiKey),
        openai.WithModel("gpt-4o-mini"), // Fast and cost-effective for testing
    )
    
    // Test with simple request
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Say 'GAI installation successful!'"},
                },
            },
        },
        MaxTokens: 20,
    })
    
    if err != nil {
        log.Fatalf("Error: %v", err)
    }
    
    fmt.Printf("✅ Success! Response: %s\n", response.Text)
    fmt.Printf("📊 Tokens used: %d\n", response.Usage.TotalTokens)
}
```

Run the test:

```bash
go mod tidy
go run main.go
```

You should see output like:
```
✅ Success! Response: GAI installation successful!
📊 Tokens used: 15
```

### Test with Local Provider (Ollama)

If you prefer to test without API keys, install Ollama:

```bash
# Install Ollama (see https://ollama.ai for your platform)
curl -fsSL https://ollama.ai/install.sh | sh

# Pull a model
ollama pull llama3.2

# Start Ollama service
ollama serve
```

Test with Ollama:

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
    // Create Ollama provider (no API key needed)
    provider := ollama.New(
        ollama.WithBaseURL("http://localhost:11434"),
        ollama.WithModel("llama3.2"),
    )
    
    // Test request
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Say hello from Ollama!"},
                },
            },
        },
        MaxTokens: 50,
    })
    
    if err != nil {
        log.Fatalf("Error: %v", err)
    }
    
    fmt.Printf("✅ Ollama Success! Response: %s\n", response.Text)
}
```

### Test Ultra-Fast Groq Provider

Test the native Groq provider with ultra-fast inference:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"
    
    "github.com/recera/gai/core"
    "github.com/recera/gai/providers/groq"
)

func main() {
    // Create Groq provider
    provider := groq.New(
        groq.WithAPIKey(os.Getenv("GROQ_API_KEY")),
        groq.WithModel("llama-3.1-8b-instant"),
    )
    
    // Time the request to see ultra-fast performance
    start := time.Now()
    
    response, err := provider.GenerateText(context.Background(), core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Write a haiku about speed"},
                },
            },
        },
        MaxTokens: 50,
    })
    
    duration := time.Since(start)
    
    if err != nil {
        log.Fatalf("Error: %v", err)
    }
    
    fmt.Printf("⚡ Groq Response (%.2fs): %s\n", duration.Seconds(), response.Text)
}
```

## IDE Setup

### VS Code

For the best Go development experience with GAI:

1. Install the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.go)
2. Enable Go modules support
3. Install go tools: `Ctrl+Shift+P` → "Go: Install/Update Tools"

Recommended VS Code settings (`.vscode/settings.json`):

```json
{
    "go.useLanguageServer": true,
    "go.autocompleteUnimportedPackages": true,
    "go.gocodeAutoBuild": false,
    "go.lintOnSave": "package",
    "go.formatTool": "goimports",
    "go.testFlags": ["-v"],
    "go.testTimeout": "120s"
}
```

### GoLand / IntelliJ IDEA

GAI works seamlessly with JetBrains IDEs:

1. Ensure Go plugin is installed and enabled
2. Import the GAI project or create a new Go project
3. Enable Go modules support in Settings → Go → Go Modules

## CLI Tool Setup

The GAI CLI provides development utilities:

```bash
# Install CLI globally
go install github.com/recera/gai/cmd/ai@latest

# Verify installation
ai version

# Get help
ai help

# Start development server
ai dev serve
```

The development server provides:
- Interactive web UI: http://localhost:8080
- REST API: http://localhost:8080/api/generate
- Streaming endpoints: http://localhost:8080/api/chat
- Health check: http://localhost:8080/api/health

## Troubleshooting

### Common Issues

#### "module not found" errors
```bash
# Ensure you're in a Go module directory
go mod init your-project-name
go get github.com/recera/gai@latest
```

#### "Go version too old" errors
```bash
# Check Go version
go version

# Update Go if needed (download from golang.org)
# Or use Go version manager like g or gvm
```

#### API key issues
```bash
# Verify environment variables are set
echo $OPENAI_API_KEY

# Test API key directly
curl -H "Authorization: Bearer $OPENAI_API_KEY" \
     https://api.openai.com/v1/models
```

#### Network/proxy issues
```bash
# Set Go proxy if needed
go env -w GOPROXY=https://proxy.golang.org,direct

# Or use direct access
go env -w GOPROXY=direct

# Check Go module settings
go env GOPROXY GOSUMDB
```

#### Import path issues
```bash
# Clean module cache if needed
go clean -modcache

# Re-download dependencies
go mod download
```

### Getting Help

- Check our [Troubleshooting Guide](../troubleshooting/common-issues.md)
- Visit [GitHub Issues](https://github.com/recera/gai/issues)
- Join our [Discord Community](https://discord.gg/gai)
- Read the [FAQ](../troubleshooting/faq.md)

## Next Steps

Now that GAI is installed and verified:

1. **[Quick Start Tutorial](./quickstart.md)** - Build your first AI app in 5 minutes
2. **[Core Concepts](../core-concepts/architecture.md)** - Understand GAI's architecture
3. **[Provider Guides](../providers/)** - Deep dive into specific providers
4. **[Examples](../../examples/)** - Explore comprehensive examples
5. **[Best Practices](../guides/best-practices.md)** - Learn production patterns

## Update GAI

To update to the latest version:

```bash
# Update to latest version
go get -u github.com/recera/gai@latest

# Update specific packages
go get -u github.com/recera/gai/providers/openai@latest

# Update CLI tool
go install github.com/recera/gai/cmd/ai@latest

# Verify update
go list -m github.com/recera/gai
```

## Uninstall

To remove GAI from your project:

```bash
# Remove from go.mod
go mod edit -droprequire github.com/recera/gai

# Clean unused dependencies
go mod tidy

# Remove CLI tool
rm $(which ai)
```

---

**Need help?** Check our [troubleshooting guide](../troubleshooting/) or [open an issue](https://github.com/recera/gai/issues).