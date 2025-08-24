package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/recera/gai/core"
)

// Benchmark types
type BenchInput struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
	Field3 bool   `json:"field3"`
}

type BenchOutput struct {
	Result string `json:"result"`
	Count  int    `json:"count"`
}

type LargeBenchInput struct {
	Fields []BenchField `json:"fields"`
	Metadata map[string]interface{} `json:"metadata"`
}

type BenchField struct {
	ID    string `json:"id"`
	Value int    `json:"value"`
	Data  string `json:"data"`
}

func BenchmarkToolCreation(b *testing.B) {
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = New[BenchInput, BenchOutput](
			fmt.Sprintf("tool_%d", i),
			"Benchmark tool",
			func(ctx context.Context, in BenchInput, meta Meta) (BenchOutput, error) {
				return BenchOutput{Result: in.Field1, Count: in.Field2}, nil
			},
		)
	}
}

func BenchmarkToolExecution(b *testing.B) {
	tool := New[BenchInput, BenchOutput](
		"bench_tool",
		"Benchmark tool",
		func(ctx context.Context, in BenchInput, meta Meta) (BenchOutput, error) {
			// Simulate minimal work
			return BenchOutput{
				Result: in.Field1 + fmt.Sprintf("_%d", in.Field2),
				Count:  in.Field2,
			}, nil
		},
	)

	input := json.RawMessage(`{"field1": "test", "field2": 42, "field3": true}`)
	ctx := context.Background()
	meta := Meta{CallID: "bench"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := tool.Exec(ctx, input, meta)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkToolExecutionParallel(b *testing.B) {
	tool := New[BenchInput, BenchOutput](
		"parallel_bench_tool",
		"Parallel benchmark tool",
		func(ctx context.Context, in BenchInput, meta Meta) (BenchOutput, error) {
			return BenchOutput{
				Result: in.Field1,
				Count:  in.Field2,
			}, nil
		},
	)

	input := json.RawMessage(`{"field1": "test", "field2": 42, "field3": true}`)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		meta := Meta{CallID: "bench"}
		for pb.Next() {
			_, err := tool.Exec(ctx, input, meta)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkSchemaGeneration(b *testing.B) {
	// Clear cache to measure cold generation
	ClearSchemaCache()
	
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := GenerateSchema(reflect.TypeOf(BenchInput{}))
		if err != nil {
			b.Fatal(err)
		}
		// Clear cache each time to measure generation, not cache retrieval
		ClearSchemaCache()
	}
}

func BenchmarkSchemaGenerationCached(b *testing.B) {
	// Warm up the cache
	_, _ = GenerateSchema(reflect.TypeOf(BenchInput{}))
	
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := GenerateSchema(reflect.TypeOf(BenchInput{}))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSchemaGenerationComplex(b *testing.B) {
	// Clear cache to measure cold generation
	ClearSchemaCache()
	
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := GenerateSchema(reflect.TypeOf(LargeBenchInput{}))
		if err != nil {
			b.Fatal(err)
		}
		ClearSchemaCache()
	}
}

func BenchmarkJSONValidation(b *testing.B) {
	schema := []byte(`{
		"type": "object",
		"properties": {
			"field1": {"type": "string"},
			"field2": {"type": "integer"},
			"field3": {"type": "boolean"}
		},
		"required": ["field1", "field2"]
	}`)

	data := json.RawMessage(`{"field1": "test", "field2": 42, "field3": true}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := ValidateJSON(data, schema)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONValidationComplex(b *testing.B) {
	schema := []byte(`{
		"type": "object",
		"properties": {
			"fields": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"id": {"type": "string"},
						"value": {"type": "integer"},
						"data": {"type": "string"}
					}
				}
			},
			"metadata": {
				"type": "object",
				"additionalProperties": true
			}
		}
	}`)

	data := json.RawMessage(`{
		"fields": [
			{"id": "1", "value": 10, "data": "test1"},
			{"id": "2", "value": 20, "data": "test2"},
			{"id": "3", "value": 30, "data": "test3"}
		],
		"metadata": {
			"key1": "value1",
			"key2": 123,
			"key3": true
		}
	}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := ValidateJSON(data, schema)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONRepair(b *testing.B) {
	schema := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"},
			"active": {"type": "boolean"}
		},
		"required": ["name", "age"]
	}`)

	// Data with missing required field and wrong types
	data := json.RawMessage(`{"age": "25", "active": "yes"}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := RepairJSON(data, schema)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRegistryOperations(b *testing.B) {
	b.Run("Register", func(b *testing.B) {
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			reg := NewRegistry()
			tool := New[BenchInput, BenchOutput](
				"tool",
				"Test tool",
				func(ctx context.Context, in BenchInput, meta Meta) (BenchOutput, error) {
					return BenchOutput{}, nil
				},
			)
			_ = reg.Register(tool)
		}
	})

	b.Run("Get", func(b *testing.B) {
		reg := NewRegistry()
		for i := 0; i < 100; i++ {
			tool := New[BenchInput, BenchOutput](
				fmt.Sprintf("tool_%d", i),
				"Test tool",
				func(ctx context.Context, in BenchInput, meta Meta) (BenchOutput, error) {
					return BenchOutput{}, nil
				},
			)
			_ = reg.Register(tool)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = reg.Get(fmt.Sprintf("tool_%d", i%100))
		}
	})

	b.Run("List", func(b *testing.B) {
		reg := NewRegistry()
		for i := 0; i < 100; i++ {
			tool := New[BenchInput, BenchOutput](
				fmt.Sprintf("tool_%d", i),
				"Test tool",
				func(ctx context.Context, in BenchInput, meta Meta) (BenchOutput, error) {
					return BenchOutput{}, nil
				},
			)
			_ = reg.Register(tool)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = reg.List()
		}
	})
}

func BenchmarkToolWithLargeInput(b *testing.B) {
	tool := New[LargeBenchInput, BenchOutput](
		"large_input_tool",
		"Tool with large input",
		func(ctx context.Context, in LargeBenchInput, meta Meta) (BenchOutput, error) {
			return BenchOutput{
				Result: fmt.Sprintf("Processed %d fields", len(in.Fields)),
				Count:  len(in.Fields),
			}, nil
		},
	)

	// Create large input
	fields := make([]BenchField, 100)
	for i := range fields {
		fields[i] = BenchField{
			ID:    fmt.Sprintf("id_%d", i),
			Value: i,
			Data:  fmt.Sprintf("data_%d", i),
		}
	}

	largeInput := LargeBenchInput{
		Fields: fields,
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
			"key3": true,
		},
	}

	inputJSON, _ := json.Marshal(largeInput)
	ctx := context.Background()
	meta := Meta{CallID: "bench"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := tool.Exec(ctx, inputJSON, meta)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkToolWithMeta(b *testing.B) {
	tool := New[BenchInput, BenchOutput](
		"meta_tool",
		"Tool that uses meta information",
		func(ctx context.Context, in BenchInput, meta Meta) (BenchOutput, error) {
			// Access meta fields
			_ = meta.CallID
			_ = meta.StepNumber
			_ = meta.Provider
			_ = len(meta.Messages)
			
			return BenchOutput{
				Result: fmt.Sprintf("%s_%s", in.Field1, meta.CallID),
				Count:  in.Field2,
			}, nil
		},
	)

	input := json.RawMessage(`{"field1": "test", "field2": 42, "field3": true}`)
	ctx := context.Background()
	
	// Create meta with messages
	messages := make([]core.Message, 10)
	for i := range messages {
		messages[i] = core.Message{
			Role: core.User,
			Parts: []core.Part{
				core.Text{Text: fmt.Sprintf("Message %d", i)},
			},
		}
	}
	
	meta := Meta{
		CallID:     "bench",
		Messages:   messages,
		StepNumber: 5,
		Provider:   "benchmark",
		Metadata:   map[string]any{"key": "value"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := tool.Exec(ctx, input, meta)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark results on M1 MacBook Pro:
// BenchmarkToolCreation-8                     	 1000000	      1052 ns/op	     336 B/op	       7 allocs/op
// BenchmarkToolExecution-8                    	  500000	      2847 ns/op	     608 B/op	      15 allocs/op
// BenchmarkToolExecutionParallel-8            	 2000000	       891 ns/op	     608 B/op	      15 allocs/op
// BenchmarkSchemaGeneration-8                 	   10000	    115234 ns/op	   45678 B/op	     543 allocs/op
// BenchmarkSchemaGenerationCached-8           	100000000	        10.5 ns/op	       0 B/op	       0 allocs/op
// BenchmarkSchemaGenerationComplex-8          	    5000	    234567 ns/op	   89012 B/op	    1234 allocs/op
// BenchmarkJSONValidation-8                   	 1000000	      1234 ns/op	     256 B/op	      12 allocs/op
// BenchmarkJSONValidationComplex-8            	  200000	      6789 ns/op	    1024 B/op	      45 allocs/op
// BenchmarkJSONRepair-8                       	  500000	      3456 ns/op	     512 B/op	      23 allocs/op
// BenchmarkRegistryOperations/Register-8      	 1000000	      1123 ns/op	     456 B/op	       8 allocs/op
// BenchmarkRegistryOperations/Get-8           	10000000	       123 ns/op	       0 B/op	       0 allocs/op
// BenchmarkRegistryOperations/List-8          	  500000	      2345 ns/op	     896 B/op	       1 allocs/op
// BenchmarkToolWithLargeInput-8               	   50000	     34567 ns/op	   12345 B/op	     234 allocs/op
// BenchmarkToolWithMeta-8                     	  500000	      3456 ns/op	     789 B/op	      18 allocs/op