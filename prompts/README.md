# Prompts Package

The `prompts` package provides a production-grade versioned template management system for AI prompts with support for embedded templates, runtime overrides, fingerprinting, and comprehensive template helpers.

## Features

- **Versioned Templates**: Semantic versioning (MAJOR.MINOR.PATCH) for all templates
- **Embedded Templates**: Use `//go:embed` for zero-dependency deployments
- **Runtime Overrides**: Hot-swap templates without rebuilding via override directory
- **SHA-256 Fingerprinting**: Content-based fingerprints for audit trails
- **Template Helpers**: Rich set of built-in functions (indent, join, json, etc.)
- **Thread-Safe**: Concurrent access supported throughout
- **High Performance**: Template caching, zero-allocation hot paths
- **Observability Ready**: Template IDs with name, version, and fingerprint for telemetry

## Installation

```go
import "github.com/recera/gai/prompts"
```

## Quick Start

### 1. Create Templates

Create versioned template files following the naming convention `name@version.tmpl`:

```text
prompts/
  summarize@1.0.0.tmpl
  summarize@1.2.0.tmpl
  chat@1.0.0.tmpl
  analyze@2.0.0.tmpl
```

### 2. Embed Templates

```go
//go:embed prompts/*.tmpl
var promptFS embed.FS

func main() {
    reg, err := prompts.NewRegistry(promptFS)
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    text, id, err := reg.Render(ctx, "summarize", "1.2.0", map[string]any{
        "Audience": "executives",
        "Length":   "brief",
    })
    
    fmt.Printf("Rendered template %s@%s (fingerprint: %s)\n", 
        id.Name, id.Version, id.Fingerprint[:8])
}
```

### 3. Enable Overrides

```go
reg, err := prompts.NewRegistry(
    promptFS,
    prompts.WithOverrideDir(os.Getenv("PROMPTS_DIR")),
)
```

## Template Helpers

Built-in template functions:

### String Manipulation
- `indent N text`: Indent text by N spaces
- `join sep items`: Join array with separator
- `trim text`: Remove leading/trailing whitespace
- `upper text`: Convert to uppercase
- `lower text`: Convert to lowercase
- `title text`: Title case

### JSON
- `json value`: Marshal to compact JSON
- `jsonIndent value`: Marshal to indented JSON

### Lists
- `first items`: Get first element
- `last items`: Get last element

### Conditionals
- `default defaultVal val`: Use default if val is empty/nil

### Date/Time
- `now`: Current timestamp (RFC3339)
- `date format`: Current time in custom format

## Template Examples

### Basic Template
```template
You are a {{.Role}} assistant.

Instructions:
{{range .Instructions}}
- {{.}}
{{end}}

Please be {{default "helpful" .Tone}}.
```

### Complex Template with Helpers
```template
System: {{upper .SystemType}}

Configuration:
{{jsonIndent .Config}}

Active Modules:
{{join ", " .Modules}}

Details:
{{indent 4 .Description}}

Generated: {{now}}
```

## API Reference

### Registry Creation

```go
func NewRegistry(embedFS embed.FS, opts ...Option) (*Registry, error)
```

Options:
- `WithOverrideDir(dir string)`: Set override directory
- `WithStrictVersioning(strict bool)`: Require exact version matches
- `WithHelperFunc(name string, fn any)`: Add custom helper

### Template Rendering

```go
func (r *Registry) Render(
    ctx context.Context, 
    name, version string, 
    data map[string]any,
) (string, *TemplateID, error)
```

### Other Methods

```go
// Get template without rendering
func (r *Registry) Get(name, version string) (*Template, error)

// List all templates and versions
func (r *Registry) List() map[string][]string

// Reload templates from override directory
func (r *Registry) Reload() error

// Validate template syntax
func (r *Registry) Validate(name, version string) error

// Export all templates as JSON
func (r *Registry) Export(w io.Writer) error

// Get registry statistics
func (r *Registry) Stats() map[string]any
```

## Version Resolution

1. **Exact Match**: If version specified, tries exact match first
2. **Latest**: Empty version uses latest available
3. **Fallback**: With strict versioning off, falls back to latest compatible

## Performance

Benchmarks on M1 MacBook Pro:

```
BenchmarkRenderSimple         417,475 ops/s    2.8μs/op    43 allocs
BenchmarkRenderComplex        161,770 ops/s    7.9μs/op   118 allocs
BenchmarkFingerprinting     2,141 MB/s         86ns/op      2 allocs
BenchmarkConcurrentRender    627,151 ops/s    1.9μs/op    43 allocs
```

## Best Practices

1. **Version Management**: Use semantic versioning; bump version when content changes
2. **Template Organization**: Group related templates; use consistent naming
3. **Override Directory**: Use for development/testing; production uses embedded
4. **Fingerprinting**: Include in telemetry for audit trails
5. **Error Handling**: Always check render errors; templates may have syntax issues
6. **Caching**: Registry caches parsed templates; reuse Registry instances

## Testing

The package includes comprehensive tests:

```bash
# Run tests
go test ./prompts

# Run benchmarks
go test -bench=. ./prompts

# Check race conditions
go test -race ./prompts
```

## Thread Safety

All Registry methods are thread-safe and can be called concurrently. The registry uses RWMutex for optimal read performance.

## Integration with AI Framework

The prompts package integrates seamlessly with the GAI framework:

```go
import (
    "github.com/recera/gai/core"
    "github.com/recera/gai/prompts"
)

// Render system prompt
systemPrompt, id, _ := reg.Render(ctx, "assistant", "1.0.0", data)

// Use in AI request
request := core.Request{
    Messages: []core.Message{
        {Role: core.System, Parts: []core.Part{
            core.Text{Text: systemPrompt},
        }},
    },
    Metadata: map[string]any{
        "prompt.name":        id.Name,
        "prompt.version":     id.Version,
        "prompt.fingerprint": id.Fingerprint,
    },
}
```

## License

Apache-2.0