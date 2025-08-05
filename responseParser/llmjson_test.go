package responseParser

import (
	"encoding/json"
	"reflect"
	"testing"
)

type SimpleStruct struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type ComplexStruct struct {
	ID        int          `json:"id"`
	Name      string       `json:"name"`
	Tags      []string     `json:"tags"`
	Active    bool         `json:"active"`
	Metadata  interface{}  `json:"metadata"`
	SubStruct SimpleStruct `json:"sub_struct"`
	Items     []Item       `json:"items"`
}

type Item struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected SimpleStruct
		wantErr  bool
	}{
		{
			name:     "valid json",
			input:    `{"name": "test", "value": 42}`,
			expected: SimpleStruct{Name: "test", Value: 42},
			wantErr:  false,
		},
		{
			name: "json with comments",
			input: `{
				// This is a comment
				"name": "test", 
				"value": 42
				/* Multi-line comment */
			}`,
			expected: SimpleStruct{Name: "test", Value: 42},
			wantErr:  false,
		},
		{
			name:     "string value for number",
			input:    `{"name": "test", "value": "42"}`,
			expected: SimpleStruct{Name: "test", Value: 42},
			wantErr:  false,
		},
		{
			name:     "markdown code block",
			input:    "```json\n{\"name\": \"test\", \"value\": 42}\n```",
			expected: SimpleStruct{Name: "test", Value: 42},
			wantErr:  false,
		},
		{
			name: "text with json",
			input: `Here is the JSON response:
			{
				"name": "test",
				"value": 42
			}`,
			expected: SimpleStruct{Name: "test", Value: 42},
			wantErr:  false,
		},
		{
			name:     "incomplete json",
			input:    `{"name": "test", "value": 42`,
			expected: SimpleStruct{Name: "test", Value: 42},
			wantErr:  false,
		},
		{
			name:     "single quotes",
			input:    `{'name': 'test', 'value': 42}`,
			expected: SimpleStruct{Name: "test", Value: 42},
			wantErr:  false,
		},
		{
			name:     "trailing comma",
			input:    `{"name": "test", "value": 42,}`,
			expected: SimpleStruct{Name: "test", Value: 42},
			wantErr:  false,
		},
		{
			name:     "no json data",
			input:    "There is no JSON here",
			expected: SimpleStruct{},
			wantErr:  true,
		},
		{
			name:     "empty input",
			input:    "",
			expected: SimpleStruct{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse[SimpleStruct](tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Parse() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseInto(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected SimpleStruct
		wantErr  bool
	}{
		{
			name:     "valid json",
			input:    `{"name": "test", "value": 42}`,
			expected: SimpleStruct{Name: "test", Value: 42},
			wantErr:  false,
		},
		{
			name:     "strict mode",
			input:    `{'name': 'test', "value": 42}`,
			expected: SimpleStruct{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result SimpleStruct

			if tt.name == "strict mode" {
				err := ParseInto(tt.input, &result, StrictOptions())
				if (err != nil) != tt.wantErr {
					t.Errorf("ParseInto() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				err := ParseInto(tt.input, &result)
				if (err != nil) != tt.wantErr {
					t.Errorf("ParseInto() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("ParseInto() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestComplexStructParsing(t *testing.T) {
	input := `{
		"id": 123,
		"name": "Complex Example",
		"tags": ["tag1", "tag2", "tag3"],
		"active": true,
		"metadata": {
			"created": "2023-01-01",
			"version": 2.5
		},
		"sub_struct": {
			"name": "Nested",
			"value": 99
		},
		"items": [
			{"id": 1, "label": "Item 1"},
			{"id": 2, "label": "Item 2"}
		]
	}`

	expected := ComplexStruct{
		ID:     123,
		Name:   "Complex Example",
		Tags:   []string{"tag1", "tag2", "tag3"},
		Active: true,
		Metadata: map[string]interface{}{
			"created": "2023-01-01",
			"version": 2.5,
		},
		SubStruct: SimpleStruct{
			Name:  "Nested",
			Value: 99,
		},
		Items: []Item{
			{ID: 1, Label: "Item 1"},
			{ID: 2, Label: "Item 2"},
		},
	}

	result, err := Parse[ComplexStruct](input)
	if err != nil {
		t.Errorf("Failed to parse complex struct: %v", err)
		return
	}

	// Compare fields manually since Metadata is interface{}
	if result.ID != expected.ID {
		t.Errorf("ID = %v, want %v", result.ID, expected.ID)
	}
	if result.Name != expected.Name {
		t.Errorf("Name = %v, want %v", result.Name, expected.Name)
	}
	if !reflect.DeepEqual(result.Tags, expected.Tags) {
		t.Errorf("Tags = %v, want %v", result.Tags, expected.Tags)
	}
	if result.Active != expected.Active {
		t.Errorf("Active = %v, want %v", result.Active, expected.Active)
	}
	if !reflect.DeepEqual(result.SubStruct, expected.SubStruct) {
		t.Errorf("SubStruct = %v, want %v", result.SubStruct, expected.SubStruct)
	}
	if len(result.Items) != len(expected.Items) {
		t.Errorf("Items length = %v, want %v", len(result.Items), len(expected.Items))
	} else {
		for i := range result.Items {
			if !reflect.DeepEqual(result.Items[i], expected.Items[i]) {
				t.Errorf("Item[%d] = %v, want %v", i, result.Items[i], expected.Items[i])
			}
		}
	}

	// Check metadata using JSON marshaling for comparison
	resultMetaBytes, _ := json.Marshal(result.Metadata)
	expectedMetaBytes, _ := json.Marshal(expected.Metadata)
	if !reflect.DeepEqual(resultMetaBytes, expectedMetaBytes) {
		t.Errorf("Metadata = %v, want %v", string(resultMetaBytes), string(expectedMetaBytes))
	}
}
