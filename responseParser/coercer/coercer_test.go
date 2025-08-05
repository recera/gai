package coercer

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

type SimpleStruct struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type SnakeCaseStruct struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	UserID    int    `json:"user_id"`
}

type TimeStruct struct {
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

func TestCoerce(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		target      interface{}
		expected    interface{}
		options     CoerceOptions
		shouldError bool
	}{
		{
			name: "basic string to int coercion",
			input: map[string]interface{}{
				"name":  "test",
				"value": "42",
			},
			target:   &SimpleStruct{},
			expected: &SimpleStruct{Name: "test", Value: 42},
			options:  DefaultOptions(),
		},
		{
			name: "case insensitive matching",
			input: map[string]interface{}{
				"Name":  "test",
				"Value": 42,
			},
			target:   &SimpleStruct{},
			expected: &SimpleStruct{Name: "test", Value: 42},
			options:  DefaultOptions(),
		},
		{
			name: "snake case matching",
			input: map[string]interface{}{
				"first_name": "John",
				"last_name":  "Doe",
				"user_id":    123,
			},
			target: &SnakeCaseStruct{},
			expected: &SnakeCaseStruct{
				FirstName: "John",
				LastName:  "Doe",
				UserID:    123,
			},
			options: DefaultOptions(),
		},
		{
			name: "camel case to snake case",
			input: map[string]interface{}{
				"firstName": "John",
				"lastName":  "Doe",
				"userId":    123,
			},
			target: &SnakeCaseStruct{},
			expected: &SnakeCaseStruct{
				FirstName: "John",
				LastName:  "Doe",
				UserID:    123,
			},
			options: DefaultOptions(),
		},
		{
			name: "boolean string coercion",
			input: map[string]interface{}{
				"name":  "test",
				"value": "true",
			},
			target:      &SimpleStruct{},
			shouldError: true, // "true" can't be coerced directly to int
			options:     DefaultOptions(),
		},
		{
			name: "string to time coercion",
			input: map[string]interface{}{
				"created": "2023-01-01T12:00:00Z",
				"updated": "2023-02-15",
			},
			target: &TimeStruct{},
			expected: &TimeStruct{
				Created: mustParseTime(t, "2006-01-02T15:04:05Z", "2023-01-01T12:00:00Z"),
				Updated: mustParseTime(t, "2006-01-02", "2023-02-15"),
			},
			options: DefaultOptions(),
		},
		{
			name: "strict options",
			input: map[string]interface{}{
				"name":  "test",
				"value": "42",
			},
			target:      &SimpleStruct{},
			shouldError: true, // Strict mode doesn't allow string to int coercion
			options:     StrictOptions(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Coerce(tt.input, tt.target, tt.options)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// For time structs, convert to maps for comparison
			if _, ok := tt.target.(*TimeStruct); ok {
				targetMap, expectedMap := structToMap(t, tt.target), structToMap(t, tt.expected)
				if !reflect.DeepEqual(targetMap, expectedMap) {
					t.Errorf("Result = %v, want %v", targetMap, expectedMap)
				}
			} else if !reflect.DeepEqual(tt.target, tt.expected) {
				t.Errorf("Result = %v, want %v", tt.target, tt.expected)
			}
		})
	}
}

func TestUnmarshalAndCoerce(t *testing.T) {
	tests := []struct {
		name        string
		inputJSON   string
		target      interface{}
		expected    interface{}
		options     CoerceOptions
		shouldError bool
	}{
		{
			name:      "valid json",
			inputJSON: `{"name": "test", "value": 42}`,
			target:    &SimpleStruct{},
			expected:  &SimpleStruct{Name: "test", Value: 42},
			options:   DefaultOptions(),
		},
		{
			name:      "string to int conversion",
			inputJSON: `{"name": "test", "value": "42"}`,
			target:    &SimpleStruct{},
			expected:  &SimpleStruct{Name: "test", Value: 42},
			options:   DefaultOptions(),
		},
		{
			name:        "invalid json",
			inputJSON:   `{"name": "test", "value": }`,
			target:      &SimpleStruct{},
			shouldError: true,
			options:     DefaultOptions(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UnmarshalAndCoerce([]byte(tt.inputJSON), tt.target, tt.options)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(tt.target, tt.expected) {
				t.Errorf("Result = %v, want %v", tt.target, tt.expected)
			}
		})
	}
}

func TestDeepPreprocess(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "string to int conversion",
			input:    "42",
			expected: int64(42),
		},
		{
			name:     "string to float conversion",
			input:    "3.14",
			expected: float64(3.14),
		},
		{
			name:     "string to boolean conversion",
			input:    "true",
			expected: true,
		},
		{
			name: "map with numeric strings",
			input: map[string]interface{}{
				"int":    "42",
				"float":  "3.14",
				"bool":   "true",
				"string": "hello",
			},
			expected: map[string]interface{}{
				"int":    int64(42),
				"float":  float64(3.14),
				"bool":   true,
				"string": "hello",
			},
		},
		{
			name: "nested structures",
			input: map[string]interface{}{
				"array": []interface{}{"1", "2", "3.14", "true"},
				"obj": map[string]interface{}{
					"value": "42",
				},
			},
			expected: map[string]interface{}{
				"array": []interface{}{int64(1), int64(2), float64(3.14), true},
				"obj": map[string]interface{}{
					"value": int64(42),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processed, err := deepPreprocess(tt.input)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// For comparison of maps/slices, convert both to JSON and compare
			processedJSON, _ := json.Marshal(processed)
			expectedJSON, _ := json.Marshal(tt.expected)

			if !reflect.DeepEqual(processedJSON, expectedJSON) {
				t.Errorf("Result = %s, want %s", string(processedJSON), string(expectedJSON))
			}
		})
	}
}

// Helper function to parse time or fail the test
func mustParseTime(t *testing.T, layout, value string) time.Time {
	tm, err := time.Parse(layout, value)
	if err != nil {
		t.Fatalf("Failed to parse time %q: %v", value, err)
	}
	return tm
}

// Helper function to convert struct to map for easier comparison
func structToMap(t *testing.T, v interface{}) map[string]interface{} {
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Failed to marshal struct: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal struct: %v", err)
	}

	return result
}
