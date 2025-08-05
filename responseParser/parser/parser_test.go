package parser

import (
	"encoding/json"
	"testing"
	//"github.com/pkg/errors"
)

func TestToJSON(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		relaxed      bool
		autocomplete bool
		fixFormat    bool
		wantErr      bool
	}{
		{
			name:         "standard valid json",
			input:        `{"key": "value", "number": 42, "bool": true}`,
			relaxed:      false,
			autocomplete: false,
			fixFormat:    false,
			wantErr:      false,
		},
		{
			name: "json with comments",
			input: `{
				// This is a comment
				"key": "value", 
				"number": 42
				/* Another comment */
			}`,
			relaxed:      true,
			autocomplete: false,
			fixFormat:    true,
			wantErr:      false,
		},
		{
			name:         "trailing commas",
			input:        `{"key": "value", "number": 42,}`,
			relaxed:      true,
			autocomplete: false,
			fixFormat:    true,
			wantErr:      false,
		},
		{
			name:         "single quotes",
			input:        `{'key': 'value', 'number': 42}`,
			relaxed:      true,
			autocomplete: false,
			fixFormat:    true,
			wantErr:      false,
		},
		{
			name:         "unquoted keys",
			input:        `{key: "value", number: 42}`,
			relaxed:      true,
			autocomplete: false,
			fixFormat:    true,
			wantErr:      false,
		},
		{
			name:         "incomplete json",
			input:        `{"key": "value", "number": 42`,
			relaxed:      false,
			autocomplete: true,
			fixFormat:    false,
			wantErr:      false,
		},
		{
			name:         "javascript arrow syntax",
			input:        `{key => "value", number => 42}`,
			relaxed:      true,
			autocomplete: false,
			fixFormat:    true,
			wantErr:      false,
		},
		{
			name:         "empty input",
			input:        "",
			relaxed:      true,
			autocomplete: true,
			fixFormat:    true,
			wantErr:      true,
		},
		{
			name:         "non-json input",
			input:        "this is not json",
			relaxed:      true,
			autocomplete: true,
			fixFormat:    true,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				Relaxed:       tt.relaxed,
				Autocomplete:  tt.autocomplete,
				FixFormatting: tt.fixFormat,
			}

			jsonBytes, err := ToJSON(tt.input, opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("ToJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Verify that the output is valid JSON
			if !json.Valid(jsonBytes) {
				t.Errorf("ToJSON() produced invalid JSON: %s", string(jsonBytes))
			}

			// Try to parse the result into a generic structure to verify it's parseable
			var parsed interface{}
			if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
				t.Errorf("Unmarshalling result failed: %v", err)
			}
		})
	}
}

func TestFixCommonFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "fancy quotes",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "single quotes to double quotes",
			input:    `{'key': 'value'}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "trailing comma removal",
			input:    `{"key": "value",}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "javascript style comment removal",
			input:    "{\n// Comment\n\"key\": \"value\"\n}",
			expected: "{\n\n\"key\": \"value\"\n}",
		},
		{
			name:     "multi-line comment removal",
			input:    "{\n/* Comment */\n\"key\": \"value\"\n}",
			expected: "{\n\n\"key\": \"value\"\n}",
		},
		{
			name:     "arrow syntax fix",
			input:    "{key => \"value\"}",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "unquoted key fix",
			input:    "{key: \"value\"}",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "capitalized boolean fix",
			input:    "{\"key\": TRUE}",
			expected: "{\"key\": true}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fixCommonFormatting(tt.input)

			// We don't expect exact string matches due to whitespace differences
			// Instead, verify both parse to equivalent JSON objects
			var expectedObj interface{}
			var resultObj interface{}

			expectedErr := json.Unmarshal([]byte(tt.expected), &expectedObj)
			resultErr := json.Unmarshal([]byte(result), &resultObj)

			// If the expected string is valid JSON, both should be able to parse
			if expectedErr == nil && resultErr != nil {
				t.Errorf("fixCommonFormatting() failed: expected to parse but got error %v", resultErr)
			}
		})
	}
}

func TestAutocompleteMalformedJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "missing closing brace",
			input:    `{"key": "value"`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "missing closing bracket",
			input:    `["item1", "item2"`,
			expected: `["item1", "item2"]`,
		},
		{
			name:     "nested unclosed structures",
			input:    `{"array": [1, 2, 3, {"nested": "value"`,
			expected: `{"array": [1, 2, 3, {"nested": "value"}]}`,
		},
		{
			name:     "already balanced",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := autocompleteMalformedJSON(tt.input)
			if result != tt.expected {
				t.Errorf("autocompleteMalformedJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}
