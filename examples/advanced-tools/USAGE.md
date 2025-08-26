# Quick Usage Guide

## Setup

1. **Install Dependencies**
   ```bash
   cd examples/advanced-tools
   go mod tidy
   ```

2. **Set Environment Variables**
   ```bash
   export OPENAI_API_KEY="your-openai-api-key"
   export GROQ_API_KEY="your-groq-api-key"
   ```

3. **Build and Run**
   ```bash
   go build -o advanced-tools .
   ./advanced-tools
   ```

## Key Features Demonstrated

### ✨ Advanced Tools
- **Weather Analyzer**: Multi-location weather analysis with insights
- **Data Processor**: Statistical analysis and pattern recognition

### 🛑 StopWhen Conditions
- `MaxSteps(n)` - Stop after n steps
- `NoMoreTools()` - Stop when no tools needed
- `UntilToolSeen("tool_name")` - Stop after specific tool
- `CombineConditions(...)` - Multiple conditions

### 🚀 Model Comparison  
- **GPT-4o-mini**: Reliable, thorough analysis
- **Kimi-K2-Instruct**: Ultra-fast inference (when working)

## Expected Output

```
🚀 Advanced AI Agent Examples with GPT-5-mini vs Kimi-K2-Instruct
================================================================================

🎯 Research & Analysis Workflow
============================================================
🤖 Testing with GPT-5-mini (OpenAI)
🌦️  [Tool Execution] Weather Analysis: current for [Tokyo London Sydney]  
✅ Completed in 8.26s
📊 Steps: 1, Tools: 2, Tokens: 426
💡 Final Response: [Detailed weather analysis and recommendations]

🚀 Testing with Kimi-K2-Instruct (Groq)
🌦️  [Tool Execution] Weather Analysis: comparison for [Tokyo London Sydney]
❌ Error: Tool call ID issue (known technical limitation)
```

## Troubleshooting

### Missing API Keys
```
❌ Missing environment variable: OPENAI_API_KEY
```
**Solution**: Set your API keys in environment variables

### Build Errors
**Solution**: Run `go mod tidy` and ensure you're in the correct directory

### Tool Call Errors (Groq)
**Known Issue**: Groq provider needs tool call ID fixes in the framework

## Customization

### Add Your Own Tools
```go
myTool := tools.New[MyInput, MyOutput](
    "my_tool",
    "Description of what it does",
    func(ctx context.Context, input MyInput, meta tools.Meta) (MyOutput, error) {
        // Your implementation
        return MyOutput{...}, nil
    },
)
```

### Custom StopConditions
```go
request := core.Request{
    // ... other fields
    StopWhen: core.CombineConditions(
        core.MaxSteps(5),
        core.UntilToolSeen("my_critical_tool"),
    ),
}
```

## Performance Notes

- **OpenAI GPT-4o-mini**: 8-12s per scenario, very reliable
- **Groq Kimi-K2**: Sub-second when working, needs tool call fixes
- **Memory Usage**: Minimal, tools are stateless
- **Token Usage**: ~400-500 tokens per complex scenario