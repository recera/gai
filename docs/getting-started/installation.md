# Installation Guide

This comprehensive guide will walk you through installing and setting up GAI in your Go project. We'll cover everything from basic installation to advanced configuration options.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Installation Methods](#installation-methods)
- [Environment Setup](#environment-setup)
- [Verification](#verification)
- [IDE Setup](#ide-setup)
- [Troubleshooting](#troubleshooting)

## Prerequisites

Before installing GAI, ensure you have the following:

### Required Software

#### Go Language
GAI requires Go 1.22 or later for full feature support, including generics.

```bash
# Check your Go version
go version

# Should output: go version go1.22.x or higher
```

If you need to install or upgrade Go:
- **macOS**: `brew install go` or download from [golang.org](https://golang.org)
- **Linux**: Use your package manager or download from [golang.org](https://golang.org)
- **Windows**: Download the installer from [golang.org](https://golang.org)

#### Git
Git is required to clone repositories and manage dependencies.

```bash
# Check Git installation
git --version
```

### System Requirements

- **Memory**: Minimum 2GB RAM (4GB+ recommended for local models with Ollama)
- **Disk Space**: 100MB for GAI + space for any local models
- **Network**: Internet connection for cloud providers
- **OS**: Linux, macOS, or Windows with WSL2

## Installation Methods

### Method 1: Go Modules (Recommended)

The simplest way to add GAI to your project is using Go modules.

#### New Project Setup

```bash
# Create a new project directory
mkdir my-ai-app
cd my-ai-app

# Initialize Go module
go mod init github.com/yourusername/my-ai-app

# Install GAI
go get github.com/yourusername/gai@latest

# Install specific provider packages as needed
go get github.com/yourusername/gai/providers/openai@latest
go get github.com/yourusername/gai/providers/anthropic@latest
go get github.com/yourusername/gai/providers/gemini@latest
go get github.com/yourusername/gai/providers/ollama@latest
```

#### Existing Project Setup

```bash
# Navigate to your project
cd your-project

# Add GAI to your project
go get github.com/yourusername/gai@latest

# Update your go.mod
go mod tidy
```

### Method 2: Specific Version Installation

To install a specific version of GAI:

```bash
# Install a specific version
go get github.com/yourusername/gai@v1.0.0

# Install a pre-release version
go get github.com/yourusername/gai@v1.1.0-beta.1

# Install from a specific commit
go get github.com/yourusername/gai@commithash

# Install from a branch
go get github.com/yourusername/gai@feature-branch
```

### Method 3: Local Development Setup

For contributing to GAI or local development:

```bash
# Clone the repository
git clone https://github.com/yourusername/gai.git
cd gai

# Install dependencies
go mod download

# Build the project
go build ./...

# Run tests to verify installation
go test ./...

# In your project, use replace directive
cd ../your-project
go mod edit -replace github.com/yourusername/gai=../gai
```

### Method 4: Using GAI CLI (Optional)

GAI provides an optional CLI for project scaffolding:

```bash
# Install the GAI CLI
go install github.com/yourusername/gai/cmd/gai@latest

# Create a new project with GAI
gai new my-ai-project

# This creates a new project with:
# - Configured go.mod
# - Example code
# - Environment template
# - Docker setup (optional)
```

## Environment Setup

### API Keys Configuration

GAI requires API keys for cloud providers. Set them up as environment variables:

#### Creating an Environment File

Create a `.env` file in your project root:

```bash
# .env
# OpenAI Configuration
OPENAI_API_KEY=sk-...your-key-here...
OPENAI_ORG_ID=org-...optional...
OPENAI_MODEL=gpt-4

# Anthropic Configuration
ANTHROPIC_API_KEY=sk-ant-...your-key-here...
ANTHROPIC_MODEL=claude-3-opus-20240229

# Google Gemini Configuration
GOOGLE_API_KEY=...your-key-here...
GEMINI_MODEL=gemini-1.5-pro

# Ollama Configuration (Local)
OLLAMA_HOST=http://localhost:11434
OLLAMA_MODEL=llama3.2

# Groq Configuration
GROQ_API_KEY=gsk_...your-key-here...

# ElevenLabs (TTS)
ELEVENLABS_API_KEY=...your-key-here...

# Deepgram (STT)
DEEPGRAM_API_KEY=...your-key-here...

# Optional: Observability
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_SERVICE_NAME=my-ai-app

# Optional: Custom Settings
GAI_LOG_LEVEL=debug
GAI_TIMEOUT=60s
GAI_MAX_RETRIES=3
```

#### Loading Environment Variables

Use a package like `godotenv` to load the `.env` file:

```go
package main

import (
    "log"
    "os"
    
    "github.com/joho/godotenv"
    "github.com/yourusername/gai/providers/openai"
)

func init() {
    // Load .env file
    if err := godotenv.Load(); err != nil {
        log.Printf("Warning: .env file not found")
    }
}

func main() {
    // API key is now available
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY not set")
    }
    
    provider := openai.New(
        openai.WithAPIKey(apiKey),
    )
    // ... rest of your code
}
```

#### System-wide Environment Variables

Alternatively, set environment variables system-wide:

**macOS/Linux:**
```bash
# Add to ~/.bashrc, ~/.zshrc, or ~/.profile
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."

# Reload shell configuration
source ~/.bashrc  # or ~/.zshrc
```

**Windows:**
```powershell
# PowerShell (permanent)
[System.Environment]::SetEnvironmentVariable('OPENAI_API_KEY','sk-...','User')

# Command Prompt (temporary)
set OPENAI_API_KEY=sk-...
```

### Obtaining API Keys

Here's how to get API keys for each provider:

#### OpenAI
1. Visit [platform.openai.com](https://platform.openai.com)
2. Sign up or log in
3. Navigate to API Keys section
4. Create a new API key
5. Set usage limits and restrictions

#### Anthropic
1. Visit [console.anthropic.com](https://console.anthropic.com)
2. Create an account
3. Go to API Keys
4. Generate a new key
5. Note: Anthropic requires approval for production use

#### Google Gemini
1. Visit [makersuite.google.com](https://makersuite.google.com)
2. Sign in with Google account
3. Get API key from the console
4. Enable the Generative Language API

#### Groq
1. Visit [console.groq.com](https://console.groq.com)
2. Sign up for an account
3. Navigate to API Keys
4. Create and copy your key

## Verification

### Basic Installation Test

Create a test file to verify your installation:

```go
// test_install.go
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
        openai.WithModel("gpt-3.5-turbo"),
    )
    
    // Test request
    ctx := context.Background()
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Say 'GAI is installed correctly!'"},
                },
            },
        },
        MaxTokens: 50,
    })
    
    if err != nil {
        log.Fatalf("Installation test failed: %v", err)
    }
    
    fmt.Println("✅ Success:", response.Text)
    fmt.Printf("📊 Tokens used: %d\n", response.Usage.TotalTokens)
}
```

Run the test:
```bash
go run test_install.go
```

Expected output:
```
✅ Success: GAI is installed correctly!
📊 Tokens used: 15
```

### Advanced Verification

Test multiple providers and features:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "github.com/yourusername/gai/core"
    "github.com/yourusername/gai/providers/openai"
    "github.com/yourusername/gai/providers/anthropic"
    "github.com/yourusername/gai/providers/ollama"
    "github.com/yourusername/gai/tools"
)

func main() {
    ctx := context.Background()
    
    // Test each provider
    providers := map[string]core.Provider{
        "OpenAI": openai.New(
            openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        ),
        "Anthropic": anthropic.New(
            anthropic.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
        ),
        "Ollama": ollama.New(
            ollama.WithBaseURL("http://localhost:11434"),
        ),
    }
    
    for name, provider := range providers {
        fmt.Printf("\nTesting %s...\n", name)
        testProvider(ctx, provider)
    }
    
    // Test tools
    fmt.Println("\nTesting tools system...")
    testTools()
    
    fmt.Println("\n✅ All tests passed! GAI is properly installed.")
}

func testProvider(ctx context.Context, provider core.Provider) {
    response, err := provider.GenerateText(ctx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "Return the number 42"},
                },
            },
        },
    })
    
    if err != nil {
        fmt.Printf("  ⚠️  Provider test failed: %v\n", err)
        return
    }
    
    fmt.Printf("  ✓ Response received: %s\n", response.Text)
}

func testTools() {
    // Define a simple tool
    type Input struct {
        Number int `json:"number"`
    }
    type Output struct {
        Result int `json:"result"`
    }
    
    tool := tools.New[Input, Output](
        "double",
        "Doubles a number",
        func(ctx context.Context, in Input, meta tools.Meta) (Output, error) {
            return Output{Result: in.Number * 2}, nil
        },
    )
    
    fmt.Printf("  ✓ Tool created: %s\n", tool.Name())
}
```

## IDE Setup

### Visual Studio Code

Install recommended extensions for the best experience:

```json
// .vscode/extensions.json
{
    "recommendations": [
        "golang.go",
        "ms-vscode.makefile-tools",
        "streetsidesoftware.code-spell-checker",
        "yzhang.markdown-all-in-one"
    ]
}
```

Configure VS Code settings:

```json
// .vscode/settings.json
{
    "go.lintTool": "golangci-lint",
    "go.lintFlags": [
        "--fast"
    ],
    "go.testFlags": ["-v"],
    "go.buildTags": "",
    "go.generateTestsFlags": ["-template", "testify"],
    "gopls": {
        "experimentalPostfixCompletions": true,
        "analyses": {
            "unusedparams": true,
            "shadow": true
        }
    }
}
```

### GoLand / IntelliJ IDEA

1. Open your project
2. Go to **File → Project Structure**
3. Set SDK to Go 1.22+
4. Enable Go Modules integration
5. Configure run configurations with environment variables

### Vim/Neovim

Add to your configuration:

```vim
" For vim-go
let g:go_def_mode='gopls'
let g:go_info_mode='gopls'
let g:go_fmt_command = "goimports"
let g:go_auto_type_info = 1

" For coc.nvim
" Install coc-go extension
:CocInstall coc-go
```

## Troubleshooting

### Common Installation Issues

#### Issue: Module not found
```
go: github.com/yourusername/gai: module not found
```

**Solution:**
```bash
# Clear module cache
go clean -modcache

# Re-download modules
go mod download

# If using private repo, configure Git
go env -w GOPRIVATE=github.com/yourusername
```

#### Issue: Version conflicts
```
go: conflicting versions of module
```

**Solution:**
```bash
# Update all dependencies
go get -u ./...

# Or specify exact versions
go get github.com/yourusername/gai@v1.0.0
```

#### Issue: API key not found
```
OPENAI_API_KEY not set
```

**Solution:**
```bash
# Check environment variable
echo $OPENAI_API_KEY

# Set it if missing
export OPENAI_API_KEY="your-key"

# Or use .env file with godotenv
```

#### Issue: Connection refused (Ollama)
```
connection refused: http://localhost:11434
```

**Solution:**
```bash
# Start Ollama service
ollama serve

# Or check if it's running
curl http://localhost:11434/api/tags
```

### Platform-Specific Issues

#### macOS
- If using Homebrew, ensure paths are correct: `echo $PATH`
- For M1/M2 Macs, ensure you have the ARM64 version of Go

#### Linux
- Install build essentials: `sudo apt-get install build-essential`
- For Ubuntu/Debian: `sudo apt-get install golang-go`

#### Windows
- Use WSL2 for best compatibility
- Or ensure Git Bash is used for terminal commands
- Set GOPROXY if behind corporate firewall

### Getting Help

If you encounter issues:

1. Check the [FAQ](../troubleshooting/faq.md)
2. Search [existing issues](https://github.com/yourusername/gai/issues)
3. Join our [Discord community](https://discord.gg/gai)
4. Create a [new issue](https://github.com/yourusername/gai/issues/new) with:
   - Go version (`go version`)
   - GAI version (`go list -m github.com/yourusername/gai`)
   - Error messages
   - Minimal reproduction code

## Next Steps

Now that GAI is installed, you're ready to:

1. Follow the [Quick Start Tutorial](./quickstart.md)
2. Explore [Basic Examples](./basic-examples.md)
3. Learn about [Core Concepts](../core-concepts/architecture.md)
4. Build your first [AI Application](../tutorials/chatbot.md)

---

**Congratulations! You've successfully installed GAI. Let's build something amazing! 🚀**