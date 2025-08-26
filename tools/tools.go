// Package tools provides typed tool definitions and execution for AI frameworks.
// It supports automatic JSON Schema generation from Go types, parallel execution,
// and proper error handling with context propagation.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/obs"
)

// Meta provides context about a tool execution, including the call ID
// and the conversation history up to this point.
type Meta struct {
	// CallID uniquely identifies this tool call within a conversation
	CallID string
	// RequestID uniquely identifies the parent request
	RequestID string
	// IdempotencyScope for tool-specific deduplication (defaults to RequestID)
	IdempotencyScope string
	// Attempt number for this tool execution (1-based)
	Attempt int
	// Messages contains the conversation history up to this point
	Messages []core.Message
	// StepNumber indicates which step in a multi-step execution this is
	StepNumber int
	// Provider identifies which AI provider is calling the tool
	Provider string
	// Metadata contains arbitrary key-value pairs for telemetry
	Metadata map[string]any
}

// Handle is the interface that all tools must implement.
// It provides schema information and execution capabilities.
type Handle interface {
	// Name returns the unique identifier for this tool
	Name() string
	// Description returns a human-readable description of what the tool does
	Description() string
	// InSchemaJSON returns the JSON Schema for the tool's input parameters
	InSchemaJSON() []byte
	// OutSchemaJSON returns the JSON Schema for the tool's output
	OutSchemaJSON() []byte
	// Exec executes the tool with raw JSON input and returns the result
	Exec(ctx context.Context, raw json.RawMessage, meta Meta) (any, error)
}

// Tool represents a typed tool with specific input and output types.
// It provides type-safe execution and automatic schema generation.
type Tool[I any, O any] struct {
	name        string
	description string
	execute     func(context.Context, I, Meta) (O, error)
	inSchema    []byte
	outSchema   []byte
	mu          sync.RWMutex
	// Options for execution behavior
	timeout        int  // timeout in seconds, 0 means no timeout
	retryable      bool // whether this tool can be safely retried
	cacheable      bool // whether results can be cached
	maxInputSize   int  // maximum input size in bytes, 0 means no limit
	maxOutputSize  int  // maximum output size in bytes, 0 means no limit
}

// New creates a new typed tool with the given name, description, and execution function.
// The tool will automatically generate JSON schemas for the input and output types.
func New[I any, O any](
	name string,
	description string,
	execute func(context.Context, I, Meta) (O, error),
) Handle {
	if name == "" {
		panic("tools.New: name cannot be empty")
	}
	if execute == nil {
		panic("tools.New: execute function cannot be nil")
	}
	
	t := &Tool[I, O]{
		name:        name,
		description: description,
		execute:     execute,
		retryable:   true,  // default to retryable
		cacheable:   false, // default to not cacheable for safety
	}
	
	// Generate schemas lazily on first access
	return t
}

// Name returns the tool's unique identifier.
func (t *Tool[I, O]) Name() string {
	return t.name
}

// Description returns the tool's human-readable description.
func (t *Tool[I, O]) Description() string {
	return t.description
}

// InSchemaJSON returns the JSON Schema for the tool's input type.
// The schema is generated once and cached for performance.
func (t *Tool[I, O]) InSchemaJSON() []byte {
	t.mu.RLock()
	if t.inSchema != nil {
		t.mu.RUnlock()
		return t.inSchema
	}
	t.mu.RUnlock()
	
	t.mu.Lock()
	defer t.mu.Unlock()
	
	// Double-check after acquiring write lock
	if t.inSchema != nil {
		return t.inSchema
	}
	
	// Generate schema for input type
	var i I
	schema, err := GenerateSchema(reflect.TypeOf(i))
	if err != nil {
		// If schema generation fails, return a minimal schema
		t.inSchema = []byte(`{"type":"object"}`)
	} else {
		t.inSchema = schema
	}
	
	return t.inSchema
}

// OutSchemaJSON returns the JSON Schema for the tool's output type.
// The schema is generated once and cached for performance.
func (t *Tool[I, O]) OutSchemaJSON() []byte {
	t.mu.RLock()
	if t.outSchema != nil {
		t.mu.RUnlock()
		return t.outSchema
	}
	t.mu.RUnlock()
	
	t.mu.Lock()
	defer t.mu.Unlock()
	
	// Double-check after acquiring write lock
	if t.outSchema != nil {
		return t.outSchema
	}
	
	// Generate schema for output type
	var o O
	schema, err := GenerateSchema(reflect.TypeOf(o))
	if err != nil {
		// If schema generation fails, return a minimal schema
		t.outSchema = []byte(`{"type":"object"}`)
	} else {
		t.outSchema = schema
	}
	
	return t.outSchema
}

// Exec executes the tool with the given raw JSON input.
// It handles JSON unmarshaling, type validation, execution, and result marshaling.
// It also records observability metrics if configured.
func (t *Tool[I, O]) Exec(ctx context.Context, raw json.RawMessage, meta Meta) (any, error) {
	// Start tool span for observability
	startTime := time.Now()
	ctx, span := obs.StartToolSpan(ctx, obs.ToolSpanOptions{
		ToolName:   t.name,
		ToolID:     meta.CallID,
		InputSize:  len(raw),
		StepNumber: meta.StepNumber,
		Parallel:   false, // Will be set by runner if parallel
		RetryCount: 0,     // Will be incremented on retries
		Timeout:    time.Duration(t.timeout) * time.Second,
	})
	defer span.End()
	
	// Check input size limit
	if t.maxInputSize > 0 && len(raw) > t.maxInputSize {
		err := fmt.Errorf("input size %d exceeds maximum %d", len(raw), t.maxInputSize)
		obs.RecordError(span, err, "Input size validation failed")
		return nil, err
	}
	
	// Unmarshal input
	var input I
	if err := json.Unmarshal(raw, &input); err != nil {
		err = fmt.Errorf("failed to unmarshal input for tool %s: %w", t.name, err)
		obs.RecordError(span, err, "Input unmarshaling failed")
		return nil, err
	}
	
	// Validate input against schema if strict validation is enabled
	if err := ValidateJSON(raw, t.InSchemaJSON()); err != nil {
		err = fmt.Errorf("input validation failed for tool %s: %w", t.name, err)
		obs.RecordError(span, err, "Schema validation failed")
		return nil, err
	}
	
	// Execute the tool
	output, err := t.execute(ctx, input, meta)
	if err != nil {
		err = fmt.Errorf("tool %s execution failed: %w", t.name, err)
		obs.RecordError(span, err, "Tool execution failed")
		obs.RecordToolResult(span, false, 0, time.Since(startTime))
		
		// Record tool content with error for Braintrust display
		obs.RecordToolContent(span, t.name, raw, nil, err)
		
		return nil, err
	}
	
	// Check output size if needed (marshal to check size)
	outputSize := 0
	if t.maxOutputSize > 0 {
		outputJSON, err := json.Marshal(output)
		if err != nil {
			err = fmt.Errorf("failed to marshal output for tool %s: %w", t.name, err)
			obs.RecordError(span, err, "Output marshaling failed")
			return nil, err
		}
		outputSize = len(outputJSON)
		if outputSize > t.maxOutputSize {
			err := fmt.Errorf("output size %d exceeds maximum %d", outputSize, t.maxOutputSize)
			obs.RecordError(span, err, "Output size validation failed")
			return nil, err
		}
	} else {
		// Calculate output size for metrics even if not checking limit
		if outputJSON, err := json.Marshal(output); err == nil {
			outputSize = len(outputJSON)
		}
	}
	
	// Record successful execution
	obs.RecordToolResult(span, true, outputSize, time.Since(startTime))
	
	// Record tool content for Braintrust display
	obs.RecordToolContent(span, t.name, raw, output, nil)
	
	// Record metrics
	obs.RecordToolExecution(ctx, t.name, true, time.Since(startTime))
	
	return output, nil
}

// WithTimeout sets the execution timeout for the tool in seconds.
func (t *Tool[I, O]) WithTimeout(seconds int) *Tool[I, O] {
	t.timeout = seconds
	return t
}

// WithRetryable sets whether the tool can be safely retried on failure.
func (t *Tool[I, O]) WithRetryable(retryable bool) *Tool[I, O] {
	t.retryable = retryable
	return t
}

// WithCacheable sets whether the tool's results can be cached.
func (t *Tool[I, O]) WithCacheable(cacheable bool) *Tool[I, O] {
	t.cacheable = cacheable
	return t
}

// WithMaxInputSize sets the maximum allowed input size in bytes.
func (t *Tool[I, O]) WithMaxInputSize(bytes int) *Tool[I, O] {
	t.maxInputSize = bytes
	return t
}

// WithMaxOutputSize sets the maximum allowed output size in bytes.
func (t *Tool[I, O]) WithMaxOutputSize(bytes int) *Tool[I, O] {
	t.maxOutputSize = bytes
	return t
}

// IsRetryable returns whether the tool can be safely retried.
func (t *Tool[I, O]) IsRetryable() bool {
	return t.retryable
}

// IsCacheable returns whether the tool's results can be cached.
func (t *Tool[I, O]) IsCacheable() bool {
	return t.cacheable
}

// Timeout returns the tool's timeout in seconds (0 means no timeout).
func (t *Tool[I, O]) Timeout() int {
	return t.timeout
}

// ToolOption is a function that configures a tool.
type ToolOption[I any, O any] func(*Tool[I, O])

// WithOptions creates a new tool with the given options.
func NewWithOptions[I any, O any](
	name string,
	description string,
	execute func(context.Context, I, Meta) (O, error),
	opts ...ToolOption[I, O],
) Handle {
	t := New[I, O](name, description, execute).(*Tool[I, O])
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Timeout returns a ToolOption that sets the execution timeout.
func Timeout[I any, O any](seconds int) ToolOption[I, O] {
	return func(t *Tool[I, O]) {
		t.timeout = seconds
	}
}

// Retryable returns a ToolOption that sets whether the tool can be retried.
func Retryable[I any, O any](retryable bool) ToolOption[I, O] {
	return func(t *Tool[I, O]) {
		t.retryable = retryable
	}
}

// Cacheable returns a ToolOption that sets whether results can be cached.
func Cacheable[I any, O any](cacheable bool) ToolOption[I, O] {
	return func(t *Tool[I, O]) {
		t.cacheable = cacheable
	}
}

// MaxInputSize returns a ToolOption that sets the maximum input size.
func MaxInputSize[I any, O any](bytes int) ToolOption[I, O] {
	return func(t *Tool[I, O]) {
		t.maxInputSize = bytes
	}
}

// MaxOutputSize returns a ToolOption that sets the maximum output size.
func MaxOutputSize[I any, O any](bytes int) ToolOption[I, O] {
	return func(t *Tool[I, O]) {
		t.maxOutputSize = bytes
	}
}

// Registry manages a collection of tools and provides lookup capabilities.
type Registry struct {
	tools map[string]Handle
	mu    sync.RWMutex
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Handle),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool Handle) error {
	if tool == nil {
		return fmt.Errorf("cannot register nil tool")
	}
	
	name := tool.Name()
	if name == "" {
		return fmt.Errorf("cannot register tool with empty name")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s is already registered", name)
	}
	
	r.tools[name] = tool
	return nil
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Handle, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tool names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// All returns all registered tools.
func (r *Registry) All() []Handle {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tools := make([]Handle, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Clear removes all tools from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.tools = make(map[string]Handle)
}

// DefaultRegistry is the global tool registry.
var DefaultRegistry = NewRegistry()

// Register adds a tool to the default registry.
func Register(tool Handle) error {
	return DefaultRegistry.Register(tool)
}

// Get retrieves a tool from the default registry.
func Get(name string) (Handle, bool) {
	return DefaultRegistry.Get(name)
}

// List returns all tool names from the default registry.
func List() []string {
	return DefaultRegistry.List()
}

// All returns all tools from the default registry.
func All() []Handle {
	return DefaultRegistry.All()
}