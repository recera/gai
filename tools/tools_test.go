package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

// Test types for schema generation
type SimpleInput struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type SimpleOutput struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

type ComplexInput struct {
	Required    string            `json:"required"`
	Optional    string            `json:"optional,omitempty"`
	Number      float64           `json:"number"`
	List        []string          `json:"list"`
	Nested      NestedStruct      `json:"nested"`
	Map         map[string]int    `json:"map"`
	Interface   interface{}       `json:"interface"`
	RawJSON     json.RawMessage   `json:"raw_json"`
	Enum        string            `json:"enum" jsonschema:"enum=option1,enum=option2,enum=option3"`
	MinMax      int               `json:"min_max" jsonschema:"minimum=1,maximum=100"`
}

type NestedStruct struct {
	Field1 string `json:"field1"`
	Field2 bool   `json:"field2"`
}

func TestNewTool(t *testing.T) {
	// Test creating a simple tool
	tool := New[SimpleInput, SimpleOutput](
		"test_tool",
		"A test tool",
		func(ctx context.Context, in SimpleInput, meta Meta) (SimpleOutput, error) {
			return SimpleOutput{
				Message: fmt.Sprintf("Hello, %s! You are %d years old.", in.Name, in.Age),
				Success: true,
			}, nil
		},
	)

	if tool.Name() != "test_tool" {
		t.Errorf("Expected name 'test_tool', got '%s'", tool.Name())
	}

	if tool.Description() != "A test tool" {
		t.Errorf("Expected description 'A test tool', got '%s'", tool.Description())
	}
}

func TestToolPanics(t *testing.T) {
	// Test that New panics with empty name
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for empty name")
		}
	}()
	
	New[SimpleInput, SimpleOutput](
		"",
		"Description",
		func(ctx context.Context, in SimpleInput, meta Meta) (SimpleOutput, error) {
			return SimpleOutput{}, nil
		},
	)
}

func TestToolPanicsNilExecute(t *testing.T) {
	// Test that New panics with nil execute function
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil execute function")
		}
	}()
	
	New[SimpleInput, SimpleOutput](
		"test",
		"Description",
		nil,
	)
}

func TestToolExecution(t *testing.T) {
	tool := New[SimpleInput, SimpleOutput](
		"greet",
		"Greets a person",
		func(ctx context.Context, in SimpleInput, meta Meta) (SimpleOutput, error) {
			if in.Name == "" {
				return SimpleOutput{}, errors.New("name is required")
			}
			return SimpleOutput{
				Message: fmt.Sprintf("Hello, %s!", in.Name),
				Success: true,
			}, nil
		},
	)

	// Test successful execution
	input := json.RawMessage(`{"name": "Alice", "age": 30}`)
	result, err := tool.Exec(context.Background(), input, Meta{CallID: "test-1"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output, ok := result.(SimpleOutput)
	if !ok {
		t.Fatalf("Expected SimpleOutput, got %T", result)
	}

	if output.Message != "Hello, Alice!" {
		t.Errorf("Expected 'Hello, Alice!', got '%s'", output.Message)
	}

	if !output.Success {
		t.Error("Expected success to be true")
	}

	// Test error case
	emptyInput := json.RawMessage(`{"name": "", "age": 0}`)
	_, err = tool.Exec(context.Background(), emptyInput, Meta{CallID: "test-2"})
	if err == nil {
		t.Error("Expected error for empty name")
	}

	// Test invalid JSON
	invalidInput := json.RawMessage(`{"name": "Bob", "age": "not a number"}`)
	_, err = tool.Exec(context.Background(), invalidInput, Meta{CallID: "test-3"})
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestToolWithOptions(t *testing.T) {
	tool := NewWithOptions[SimpleInput, SimpleOutput](
		"options_tool",
		"Tool with options",
		func(ctx context.Context, in SimpleInput, meta Meta) (SimpleOutput, error) {
			return SimpleOutput{Message: "Done", Success: true}, nil
		},
		Timeout[SimpleInput, SimpleOutput](30),
		Retryable[SimpleInput, SimpleOutput](false),
		Cacheable[SimpleInput, SimpleOutput](true),
		MaxInputSize[SimpleInput, SimpleOutput](1024),
		MaxOutputSize[SimpleInput, SimpleOutput](2048),
	)

	typedTool := tool.(*Tool[SimpleInput, SimpleOutput])
	
	if typedTool.Timeout() != 30 {
		t.Errorf("Expected timeout 30, got %d", typedTool.Timeout())
	}
	
	if typedTool.IsRetryable() {
		t.Error("Expected retryable to be false")
	}
	
	if !typedTool.IsCacheable() {
		t.Error("Expected cacheable to be true")
	}
	
	if typedTool.maxInputSize != 1024 {
		t.Errorf("Expected maxInputSize 1024, got %d", typedTool.maxInputSize)
	}
	
	if typedTool.maxOutputSize != 2048 {
		t.Errorf("Expected maxOutputSize 2048, got %d", typedTool.maxOutputSize)
	}
}

func TestToolInputSizeLimit(t *testing.T) {
	tool := New[SimpleInput, SimpleOutput](
		"limited_tool",
		"Tool with size limits",
		func(ctx context.Context, in SimpleInput, meta Meta) (SimpleOutput, error) {
			return SimpleOutput{Message: "OK", Success: true}, nil
		},
	).(*Tool[SimpleInput, SimpleOutput])
	
	tool.WithMaxInputSize(10) // Very small limit
	
	largeInput := json.RawMessage(`{"name": "This is a very long name", "age": 100}`)
	_, err := tool.Exec(context.Background(), largeInput, Meta{})
	if err == nil {
		t.Error("Expected error for input exceeding size limit")
	}
}

func TestToolOutputSizeLimit(t *testing.T) {
	tool := New[SimpleInput, SimpleOutput](
		"output_limited_tool",
		"Tool with output size limits",
		func(ctx context.Context, in SimpleInput, meta Meta) (SimpleOutput, error) {
			// Create a large message
			message := ""
			for i := 0; i < 1000; i++ {
				message += "x"
			}
			return SimpleOutput{Message: message, Success: true}, nil
		},
	).(*Tool[SimpleInput, SimpleOutput])
	
	tool.WithMaxOutputSize(10) // Very small limit
	
	input := json.RawMessage(`{"name": "Test", "age": 30}`)
	_, err := tool.Exec(context.Background(), input, Meta{})
	if err == nil {
		t.Error("Expected error for output exceeding size limit")
	}
}

func TestToolContextCancellation(t *testing.T) {
	tool := New[SimpleInput, SimpleOutput](
		"slow_tool",
		"Tool that takes time",
		func(ctx context.Context, in SimpleInput, meta Meta) (SimpleOutput, error) {
			select {
			case <-time.After(100 * time.Millisecond):
				return SimpleOutput{Message: "Done", Success: true}, nil
			case <-ctx.Done():
				return SimpleOutput{}, ctx.Err()
			}
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	input := json.RawMessage(`{"name": "Test", "age": 30}`)
	_, err := tool.Exec(ctx, input, Meta{})
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

func TestRegistry(t *testing.T) {
	reg := NewRegistry()

	tool1 := New[SimpleInput, SimpleOutput](
		"tool1",
		"First tool",
		func(ctx context.Context, in SimpleInput, meta Meta) (SimpleOutput, error) {
			return SimpleOutput{Message: "Tool 1", Success: true}, nil
		},
	)

	tool2 := New[SimpleInput, SimpleOutput](
		"tool2",
		"Second tool",
		func(ctx context.Context, in SimpleInput, meta Meta) (SimpleOutput, error) {
			return SimpleOutput{Message: "Tool 2", Success: true}, nil
		},
	)

	// Test registration
	if err := reg.Register(tool1); err != nil {
		t.Errorf("Failed to register tool1: %v", err)
	}

	if err := reg.Register(tool2); err != nil {
		t.Errorf("Failed to register tool2: %v", err)
	}

	// Test duplicate registration
	if err := reg.Register(tool1); err == nil {
		t.Error("Expected error for duplicate registration")
	}

	// Test nil registration
	if err := reg.Register(nil); err == nil {
		t.Error("Expected error for nil tool registration")
	}

	// Test retrieval
	retrieved, ok := reg.Get("tool1")
	if !ok {
		t.Error("Failed to retrieve tool1")
	}
	if retrieved.Name() != "tool1" {
		t.Errorf("Retrieved wrong tool: %s", retrieved.Name())
	}

	// Test non-existent tool
	_, ok = reg.Get("nonexistent")
	if ok {
		t.Error("Should not find non-existent tool")
	}

	// Test listing
	names := reg.List()
	if len(names) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(names))
	}

	// Test getting all tools
	allTools := reg.All()
	if len(allTools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(allTools))
	}

	// Test clearing
	reg.Clear()
	names = reg.List()
	if len(names) != 0 {
		t.Errorf("Expected 0 tools after clear, got %d", len(names))
	}
}

func TestDefaultRegistry(t *testing.T) {
	// Clear the default registry first
	DefaultRegistry.Clear()

	tool := New[SimpleInput, SimpleOutput](
		"default_tool",
		"Tool in default registry",
		func(ctx context.Context, in SimpleInput, meta Meta) (SimpleOutput, error) {
			return SimpleOutput{Message: "Default", Success: true}, nil
		},
	)

	// Test registration with package-level functions
	if err := Register(tool); err != nil {
		t.Errorf("Failed to register tool: %v", err)
	}

	// Test retrieval
	retrieved, ok := Get("default_tool")
	if !ok {
		t.Error("Failed to retrieve tool from default registry")
	}
	if retrieved.Name() != "default_tool" {
		t.Errorf("Retrieved wrong tool: %s", retrieved.Name())
	}

	// Test listing
	names := List()
	found := false
	for _, name := range names {
		if name == "default_tool" {
			found = true
			break
		}
	}
	if !found {
		t.Error("default_tool not found in list")
	}

	// Test getting all
	allTools := All()
	if len(allTools) == 0 {
		t.Error("Expected at least one tool")
	}

	// Clean up
	DefaultRegistry.Clear()
}

func TestSchemaGeneration(t *testing.T) {
	tool := New[ComplexInput, SimpleOutput](
		"schema_tool",
		"Tool for testing schema generation",
		func(ctx context.Context, in ComplexInput, meta Meta) (SimpleOutput, error) {
			return SimpleOutput{Message: "OK", Success: true}, nil
		},
	)

	// Get input schema
	inSchema := tool.InSchemaJSON()
	if len(inSchema) == 0 {
		t.Error("Input schema is empty")
	}

	// Verify it's valid JSON
	var schemaObj map[string]interface{}
	if err := json.Unmarshal(inSchema, &schemaObj); err != nil {
		t.Errorf("Invalid JSON schema: %v", err)
	}

	// Check that it has expected properties
	if schemaObj["type"] != "object" {
		t.Errorf("Expected type 'object', got %v", schemaObj["type"])
	}

	// Get output schema
	outSchema := tool.OutSchemaJSON()
	if len(outSchema) == 0 {
		t.Error("Output schema is empty")
	}

	// Test that schemas are cached (should return same slice)
	inSchema2 := tool.InSchemaJSON()
	if !reflect.DeepEqual(inSchema, inSchema2) {
		t.Error("Schema should be cached")
	}
}

func TestMetaInformation(t *testing.T) {
	var capturedMeta Meta

	tool := New[SimpleInput, SimpleOutput](
		"meta_tool",
		"Tool that captures meta information",
		func(ctx context.Context, in SimpleInput, meta Meta) (SimpleOutput, error) {
			capturedMeta = meta
			return SimpleOutput{Message: "Captured", Success: true}, nil
		},
	)

	messages := []core.Message{
		{Role: core.User, Parts: []core.Part{core.Text{Text: "Hello"}}},
	}

	meta := Meta{
		CallID:     "call-123",
		Messages:   messages,
		StepNumber: 5,
		Provider:   "test-provider",
		Metadata:   map[string]any{"key": "value"},
	}

	input := json.RawMessage(`{"name": "Test", "age": 30}`)
	_, err := tool.Exec(context.Background(), input, meta)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify meta was passed correctly
	if capturedMeta.CallID != "call-123" {
		t.Errorf("Expected CallID 'call-123', got '%s'", capturedMeta.CallID)
	}

	if capturedMeta.StepNumber != 5 {
		t.Errorf("Expected StepNumber 5, got %d", capturedMeta.StepNumber)
	}

	if capturedMeta.Provider != "test-provider" {
		t.Errorf("Expected Provider 'test-provider', got '%s'", capturedMeta.Provider)
	}

	if len(capturedMeta.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(capturedMeta.Messages))
	}

	if val, ok := capturedMeta.Metadata["key"].(string); !ok || val != "value" {
		t.Error("Metadata not passed correctly")
	}
}

func TestConcurrentToolExecution(t *testing.T) {
	tool := New[SimpleInput, SimpleOutput](
		"concurrent_tool",
		"Tool for concurrent testing",
		func(ctx context.Context, in SimpleInput, meta Meta) (SimpleOutput, error) {
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			return SimpleOutput{
				Message: fmt.Sprintf("Processed %s", in.Name),
				Success: true,
			}, nil
		},
	)

	// Run multiple concurrent executions
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			input := json.RawMessage(fmt.Sprintf(`{"name": "User%d", "age": %d}`, id, id*10))
			result, err := tool.Exec(context.Background(), input, Meta{CallID: fmt.Sprintf("call-%d", id)})
			if err != nil {
				t.Errorf("Goroutine %d failed: %v", id, err)
			}
			
			output, ok := result.(SimpleOutput)
			if !ok {
				t.Errorf("Goroutine %d: wrong output type", id)
			}
			
			expectedMsg := fmt.Sprintf("Processed User%d", id)
			if output.Message != expectedMsg {
				t.Errorf("Goroutine %d: expected '%s', got '%s'", id, expectedMsg, output.Message)
			}
			
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestEmptyInterfaceHandling(t *testing.T) {
	// Test tool with interface{} input/output
	type FlexibleInput struct {
		Data interface{} `json:"data"`
	}
	
	type FlexibleOutput struct {
		Result interface{} `json:"result"`
	}
	
	tool := New[FlexibleInput, FlexibleOutput](
		"flexible_tool",
		"Tool with interface{} fields",
		func(ctx context.Context, in FlexibleInput, meta Meta) (FlexibleOutput, error) {
			return FlexibleOutput{Result: in.Data}, nil
		},
	)
	
	// Test with various input types
	inputs := []string{
		`{"data": "string"}`,
		`{"data": 123}`,
		`{"data": true}`,
		`{"data": null}`,
		`{"data": {"nested": "object"}}`,
		`{"data": ["array", "of", "values"]}`,
	}
	
	for _, input := range inputs {
		result, err := tool.Exec(context.Background(), json.RawMessage(input), Meta{})
		if err != nil {
			t.Errorf("Failed to execute with input %s: %v", input, err)
		}
		
		_, ok := result.(FlexibleOutput)
		if !ok {
			t.Errorf("Wrong output type for input %s", input)
		}
	}
}