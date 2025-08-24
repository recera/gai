package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Golden test structures - these should never change to ensure backward compatibility
type GoldenSimple struct {
	Name    string  `json:"name"`
	Age     int     `json:"age"`
	Score   float64 `json:"score"`
	Active  bool    `json:"active"`
}

type GoldenNested struct {
	ID      string       `json:"id"`
	Details GoldenDetail `json:"details"`
	Tags    []string     `json:"tags"`
}

type GoldenDetail struct {
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type GoldenWithValidation struct {
	RequiredField string  `json:"required_field" jsonschema:"required"`
	OptionalField string  `json:"optional_field,omitempty"`
	MinValue      int     `json:"min_value" jsonschema:"minimum=0"`
	MaxValue      int     `json:"max_value" jsonschema:"maximum=100"`
	EnumField     string  `json:"enum_field" jsonschema:"enum=small,enum=medium,enum=large"`
	PatternField  string  `json:"pattern_field" jsonschema:"pattern=^[A-Z][0-9]+$"`
}

type GoldenArraysAndMaps struct {
	StringList []string               `json:"string_list"`
	IntList    []int                  `json:"int_list"`
	ObjectList []GoldenSimple         `json:"object_list"`
	StringMap  map[string]string      `json:"string_map"`
	AnyMap     map[string]interface{} `json:"any_map"`
}

func TestGoldenSchemas(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping golden tests in short mode")
	}

	tests := []struct {
		name string
		typ  reflect.Type
		file string
	}{
		{
			name: "Simple struct",
			typ:  reflect.TypeOf(GoldenSimple{}),
			file: "golden_simple.json",
		},
		{
			name: "Nested struct",
			typ:  reflect.TypeOf(GoldenNested{}),
			file: "golden_nested.json",
		},
		{
			name: "Struct with validation",
			typ:  reflect.TypeOf(GoldenWithValidation{}),
			file: "golden_validation.json",
		},
		{
			name: "Arrays and maps",
			typ:  reflect.TypeOf(GoldenArraysAndMaps{}),
			file: "golden_arrays_maps.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate schema
			schema, err := GenerateSchema(tt.typ)
			if err != nil {
				t.Fatalf("Failed to generate schema: %v", err)
			}

			// Pretty format for readability
			var schemaObj interface{}
			if err := json.Unmarshal(schema, &schemaObj); err != nil {
				t.Fatalf("Failed to unmarshal schema: %v", err)
			}

			prettySchema, err := json.MarshalIndent(schemaObj, "", "  ")
			if err != nil {
				t.Fatalf("Failed to format schema: %v", err)
			}

			goldenPath := filepath.Join("testdata", tt.file)

			if os.Getenv("UPDATE_GOLDEN") == "true" {
				// Update golden files mode
				if err := os.MkdirAll("testdata", 0755); err != nil {
					t.Fatalf("Failed to create testdata directory: %v", err)
				}

				if err := os.WriteFile(goldenPath, prettySchema, 0644); err != nil {
					t.Fatalf("Failed to write golden file: %v", err)
				}

				t.Logf("Updated golden file: %s", goldenPath)
			} else {
				// Compare with golden file
				golden, err := os.ReadFile(goldenPath)
				if err != nil {
					if os.IsNotExist(err) {
						// Create the golden file if it doesn't exist
						if err := os.MkdirAll("testdata", 0755); err != nil {
							t.Fatalf("Failed to create testdata directory: %v", err)
						}

						if err := os.WriteFile(goldenPath, prettySchema, 0644); err != nil {
							t.Fatalf("Failed to create golden file: %v", err)
						}

						t.Logf("Created golden file: %s", goldenPath)
						return
					}
					t.Fatalf("Failed to read golden file: %v", err)
				}

				// Compare schemas
				if string(prettySchema) != string(golden) {
					t.Errorf("Schema mismatch for %s\nGenerated:\n%s\n\nExpected:\n%s",
						tt.name, string(prettySchema), string(golden))
				}
			}
		})
	}
}

func TestGoldenToolExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping golden tests in short mode")
	}

	// Create a tool with golden types
	tool := New[GoldenSimple, GoldenSimple](
		"golden_tool",
		"Tool for golden testing",
		func(ctx context.Context, in GoldenSimple, meta Meta) (GoldenSimple, error) {
			return GoldenSimple{
				Name:   "Processed: " + in.Name,
				Age:    in.Age + 1,
				Score:  in.Score * 2,
				Active: !in.Active,
			}, nil
		},
	)

	// Test data
	testCases := []struct {
		name     string
		input    string
		expected GoldenSimple
	}{
		{
			name:  "Basic input",
			input: `{"name": "John", "age": 30, "score": 95.5, "active": true}`,
			expected: GoldenSimple{
				Name:   "Processed: John",
				Age:    31,
				Score:  191.0,
				Active: false,
			},
		},
		{
			name:  "Zero values",
			input: `{"name": "", "age": 0, "score": 0.0, "active": false}`,
			expected: GoldenSimple{
				Name:   "Processed: ",
				Age:    1,
				Score:  0.0,
				Active: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tool.Exec(context.Background(), json.RawMessage(tc.input), Meta{})
			if err != nil {
				t.Fatalf("Execution failed: %v", err)
			}

			output, ok := result.(GoldenSimple)
			if !ok {
				t.Fatalf("Wrong output type: %T", result)
			}

			if !reflect.DeepEqual(output, tc.expected) {
				t.Errorf("Output mismatch\nGot:      %+v\nExpected: %+v", output, tc.expected)
			}
		})
	}
}

func TestGoldenComplexToolExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping golden tests in short mode")
	}

	// Create a tool with complex golden types
	tool := New[GoldenNested, GoldenArraysAndMaps](
		"complex_golden_tool",
		"Complex tool for golden testing",
		func(ctx context.Context, in GoldenNested, meta Meta) (GoldenArraysAndMaps, error) {
			return GoldenArraysAndMaps{
				StringList: in.Tags,
				IntList:    []int{1, 2, 3},
				ObjectList: []GoldenSimple{
					{Name: in.ID, Age: 25, Score: 100.0, Active: true},
				},
				StringMap: map[string]string{
					"id":          in.ID,
					"description": in.Details.Description,
				},
				AnyMap: in.Details.Metadata,
			}, nil
		},
	)

	// Test input
	input := GoldenNested{
		ID: "test-123",
		Details: GoldenDetail{
			Description: "Test description",
			Metadata: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
		},
		Tags: []string{"tag1", "tag2", "tag3"},
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Execute tool
	result, err := tool.Exec(context.Background(), inputJSON, Meta{CallID: "golden-test"})
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	output, ok := result.(GoldenArraysAndMaps)
	if !ok {
		t.Fatalf("Wrong output type: %T", result)
	}

	// Verify output
	if len(output.StringList) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(output.StringList))
	}

	if len(output.IntList) != 3 {
		t.Errorf("Expected 3 ints, got %d", len(output.IntList))
	}

	if len(output.ObjectList) != 1 {
		t.Errorf("Expected 1 object, got %d", len(output.ObjectList))
	}

	if output.StringMap["id"] != "test-123" {
		t.Errorf("Expected id 'test-123', got '%s'", output.StringMap["id"])
	}

	if len(output.AnyMap) != 3 {
		t.Errorf("Expected 3 metadata entries, got %d", len(output.AnyMap))
	}
}

func TestGoldenValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping golden tests in short mode")
	}

	// Generate schema for validation struct
	schema, err := GenerateSchema(reflect.TypeOf(GoldenWithValidation{}))
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}
	
	// Debug: Check what's in the schema
	if testing.Verbose() {
		schemaJSON, _ := json.MarshalIndent(schema, "", "  ")
		t.Logf("Generated schema:\n%s", schemaJSON)
	}

	// Test cases for validation
	testCases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "Valid input",
			input: `{
				"required_field": "value",
				"optional_field": "optional",
				"min_value": 10,
				"max_value": 90,
				"enum_field": "medium",
				"pattern_field": "A123"
			}`,
			wantErr: false,
		},
		{
			name: "Missing required field",
			input: `{
				"optional_field": "optional",
				"min_value": 10,
				"max_value": 90,
				"enum_field": "medium",
				"pattern_field": "A123"
			}`,
			wantErr: true,
		},
		{
			name: "Invalid enum value",
			input: `{
				"required_field": "value",
				"min_value": 10,
				"max_value": 90,
				"enum_field": "invalid",
				"pattern_field": "A123"
			}`,
			wantErr: true,
		},
		{
			name: "Value below minimum",
			input: `{
				"required_field": "value",
				"min_value": -5,
				"max_value": 90,
				"enum_field": "small",
				"pattern_field": "A123"
			}`,
			wantErr: true,
		},
		{
			name: "Value above maximum",
			input: `{
				"required_field": "value",
				"min_value": 10,
				"max_value": 150,
				"enum_field": "large",
				"pattern_field": "A123"
			}`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateJSON(json.RawMessage(tc.input), schema)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateJSON() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}