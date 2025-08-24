package tools

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Test structures for schema generation
type BasicStruct struct {
	StringField string  `json:"string_field"`
	IntField    int     `json:"int_field"`
	FloatField  float64 `json:"float_field"`
	BoolField   bool    `json:"bool_field"`
}

type StructWithOmitempty struct {
	Required string `json:"required"`
	Optional string `json:"optional,omitempty"`
}

type StructWithArrays struct {
	StringArray []string               `json:"string_array"`
	IntArray    []int                  `json:"int_array"`
	StructArray []BasicStruct          `json:"struct_array"`
	MapField    map[string]interface{} `json:"map_field"`
}

type NestedStructure struct {
	Level1 struct {
		Level2 struct {
			Level3 string `json:"level3"`
		} `json:"level2"`
	} `json:"level1"`
}

type StructWithTags struct {
	EnumField    string  `json:"enum_field" jsonschema:"enum=red,enum=green,enum=blue"`
	MinMaxInt    int     `json:"min_max_int" jsonschema:"minimum=1,maximum=100"`
	MinMaxFloat  float64 `json:"min_max_float" jsonschema:"minimum=0.0,maximum=1.0"`
	PatternField string  `json:"pattern_field" jsonschema:"pattern=^[A-Z][a-z]+$"`
	Description  string  `json:"description" jsonschema:"description=This is a description field"`
}

type RecursiveStruct struct {
	Name     string           `json:"name"`
	Children []RecursiveStruct `json:"children"`
}

func TestGenerateSchemaBasic(t *testing.T) {
	schema, err := GenerateSchema(reflect.TypeOf(BasicStruct{}))
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	// Check type
	if schemaMap["type"] != "object" {
		t.Errorf("Expected type 'object', got %v", schemaMap["type"])
	}

	// Check properties exist
	properties, ok := schemaMap["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties not found or wrong type")
	}

	// Check each field
	expectedFields := []string{"string_field", "int_field", "float_field", "bool_field"}
	for _, field := range expectedFields {
		if _, exists := properties[field]; !exists {
			t.Errorf("Field %s not found in schema", field)
		}
	}

	// Check field types
	if stringField, ok := properties["string_field"].(map[string]interface{}); ok {
		if stringField["type"] != "string" {
			t.Errorf("string_field should have type 'string', got %v", stringField["type"])
		}
	}

	if intField, ok := properties["int_field"].(map[string]interface{}); ok {
		if intField["type"] != "integer" {
			t.Errorf("int_field should have type 'integer', got %v", intField["type"])
		}
	}

	if floatField, ok := properties["float_field"].(map[string]interface{}); ok {
		if floatField["type"] != "number" {
			t.Errorf("float_field should have type 'number', got %v", floatField["type"])
		}
	}

	if boolField, ok := properties["bool_field"].(map[string]interface{}); ok {
		if boolField["type"] != "boolean" {
			t.Errorf("bool_field should have type 'boolean', got %v", boolField["type"])
		}
	}
}

func TestGenerateSchemaWithOmitempty(t *testing.T) {
	schema, err := GenerateSchema(reflect.TypeOf(StructWithOmitempty{}))
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	// The omitempty tag should affect whether fields are required
	// This behavior depends on the jsonschema library configuration
	properties, ok := schemaMap["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties not found")
	}

	// Both fields should be in properties
	if _, exists := properties["required"]; !exists {
		t.Error("Required field not in properties")
	}

	if _, exists := properties["optional"]; !exists {
		t.Error("Optional field not in properties")
	}
}

func TestGenerateSchemaWithArrays(t *testing.T) {
	schema, err := GenerateSchema(reflect.TypeOf(StructWithArrays{}))
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	properties, ok := schemaMap["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties not found")
	}

	// Check string array
	if stringArray, ok := properties["string_array"].(map[string]interface{}); ok {
		if stringArray["type"] != "array" {
			t.Errorf("string_array should have type 'array', got %v", stringArray["type"])
		}
		if items, ok := stringArray["items"].(map[string]interface{}); ok {
			if items["type"] != "string" {
				t.Errorf("string_array items should have type 'string', got %v", items["type"])
			}
		}
	}

	// Check map field
	if mapField, ok := properties["map_field"].(map[string]interface{}); ok {
		if mapField["type"] != "object" {
			t.Errorf("map_field should have type 'object', got %v", mapField["type"])
		}
	}
}

func TestGenerateSchemaWithNested(t *testing.T) {
	schema, err := GenerateSchema(reflect.TypeOf(NestedStructure{}))
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	// Navigate through nested structure
	properties, ok := schemaMap["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties not found")
	}

	level1, ok := properties["level1"].(map[string]interface{})
	if !ok {
		t.Fatal("level1 not found")
	}

	if level1["type"] != "object" {
		t.Errorf("level1 should be object, got %v", level1["type"])
	}
}

func TestSchemaCache(t *testing.T) {
	// Clear cache first
	ClearSchemaCache()
	
	initialSize := GetSchemaCacheSize()
	if initialSize != 0 {
		t.Errorf("Expected empty cache, got size %d", initialSize)
	}

	// Generate schema for the first time
	schema1, err := GenerateSchema(reflect.TypeOf(BasicStruct{}))
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	// Cache should have one entry
	if GetSchemaCacheSize() != 1 {
		t.Errorf("Expected cache size 1, got %d", GetSchemaCacheSize())
	}

	// Generate same schema again (should use cache)
	schema2, err := GenerateSchema(reflect.TypeOf(BasicStruct{}))
	if err != nil {
		t.Fatalf("Failed to generate schema from cache: %v", err)
	}

	// Should be the same
	if !reflect.DeepEqual(schema1, schema2) {
		t.Error("Cached schema should be identical")
	}

	// Cache size should still be 1
	if GetSchemaCacheSize() != 1 {
		t.Errorf("Expected cache size 1, got %d", GetSchemaCacheSize())
	}

	// Generate schema for different type
	_, err = GenerateSchema(reflect.TypeOf(StructWithArrays{}))
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	// Cache should have two entries
	if GetSchemaCacheSize() != 2 {
		t.Errorf("Expected cache size 2, got %d", GetSchemaCacheSize())
	}

	// Clear cache
	ClearSchemaCache()
	if GetSchemaCacheSize() != 0 {
		t.Error("Cache should be empty after clear")
	}
}

func TestValidateJSON(t *testing.T) {
	schema := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"},
			"email": {"type": "string"}
		},
		"required": ["name", "age"]
	}`)

	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "Valid data",
			data:    `{"name": "John", "age": 30, "email": "john@example.com"}`,
			wantErr: false,
		},
		{
			name:    "Missing required field",
			data:    `{"name": "John"}`,
			wantErr: true,
		},
		{
			name:    "Wrong type",
			data:    `{"name": "John", "age": "thirty"}`,
			wantErr: true,
		},
		{
			name:    "Extra fields allowed",
			data:    `{"name": "John", "age": 30, "extra": "field"}`,
			wantErr: false,
		},
		{
			name:    "Empty data",
			data:    ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSON(json.RawMessage(tt.data), schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateJSONArrays(t *testing.T) {
	schema := []byte(`{
		"type": "array",
		"items": {"type": "string"},
		"minItems": 1,
		"maxItems": 3
	}`)

	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "Valid array",
			data:    `["one", "two"]`,
			wantErr: false,
		},
		{
			name:    "Empty array",
			data:    `[]`,
			wantErr: true, // minItems: 1
		},
		{
			name:    "Too many items",
			data:    `["one", "two", "three", "four"]`,
			wantErr: true, // maxItems: 3
		},
		{
			name:    "Wrong item type",
			data:    `[1, 2, 3]`,
			wantErr: true,
		},
		{
			name:    "Not an array",
			data:    `"string"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSON(json.RawMessage(tt.data), schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateJSONNumbers(t *testing.T) {
	schema := []byte(`{
		"type": "object",
		"properties": {
			"int_value": {
				"type": "integer",
				"minimum": 0,
				"maximum": 100
			},
			"float_value": {
				"type": "number",
				"minimum": 0.0,
				"maximum": 1.0
			}
		}
	}`)

	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "Valid numbers",
			data:    `{"int_value": 50, "float_value": 0.5}`,
			wantErr: false,
		},
		{
			name:    "Integer out of range",
			data:    `{"int_value": 150, "float_value": 0.5}`,
			wantErr: true,
		},
		{
			name:    "Float out of range",
			data:    `{"int_value": 50, "float_value": 1.5}`,
			wantErr: true,
		},
		{
			name:    "Float for integer field",
			data:    `{"int_value": 50.5, "float_value": 0.5}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSON(json.RawMessage(tt.data), schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateJSONEnum(t *testing.T) {
	schema := []byte(`{
		"type": "object",
		"properties": {
			"color": {
				"type": "string",
				"enum": ["red", "green", "blue"]
			}
		}
	}`)

	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "Valid enum value",
			data:    `{"color": "red"}`,
			wantErr: false,
		},
		{
			name:    "Invalid enum value",
			data:    `{"color": "yellow"}`,
			wantErr: true,
		},
		{
			name:    "Empty string",
			data:    `{"color": ""}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSON(json.RawMessage(tt.data), schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRepairJSON(t *testing.T) {
	schema := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string", "default": "Unknown"},
			"age": {"type": "integer", "default": 0},
			"active": {"type": "boolean", "default": false}
		},
		"required": ["name", "age"]
	}`)

	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name:  "Add missing required fields",
			input: `{}`,
			expected: map[string]interface{}{
				"name": "Unknown",
				"age":  float64(0),
			},
		},
		{
			name:  "Convert wrong types",
			input: `{"name": 123, "age": "30", "active": "yes"}`,
			expected: map[string]interface{}{
				"name":   "123",
				"age":    float64(30),
				"active": true,
			},
		},
		{
			name:  "Keep valid data",
			input: `{"name": "John", "age": 25, "active": true}`,
			expected: map[string]interface{}{
				"name":   "John",
				"age":    float64(25),
				"active": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repaired, err := RepairJSON(json.RawMessage(tt.input), schema)
			if err != nil {
				t.Fatalf("RepairJSON() error = %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(repaired, &result); err != nil {
				t.Fatalf("Failed to unmarshal repaired JSON: %v", err)
			}

			for key, expectedValue := range tt.expected {
				if result[key] != expectedValue {
					t.Errorf("Key %s: expected %v, got %v", key, expectedValue, result[key])
				}
			}
		})
	}
}

func TestSpecialTypeSchemas(t *testing.T) {
	tests := []struct {
		name     string
		typ      reflect.Type
		checkFn  func(map[string]interface{}) bool
	}{
		{
			name: "interface{}",
			typ:  reflect.TypeOf((*interface{})(nil)).Elem(),
			checkFn: func(schema map[string]interface{}) bool {
				return schema["type"] == "object"
			},
		},
		{
			name: "json.RawMessage",
			typ:  reflect.TypeOf(json.RawMessage{}),
			checkFn: func(schema map[string]interface{}) bool {
				return schema["type"] == "object"
			},
		},
		{
			name: "map[string]interface{}",
			typ:  reflect.TypeOf(map[string]interface{}{}),
			checkFn: func(schema map[string]interface{}) bool {
				return schema["type"] == "object"
			},
		},
		{
			name: "[]interface{}",
			typ:  reflect.TypeOf([]interface{}{}),
			checkFn: func(schema map[string]interface{}) bool {
				return schema["type"] == "array"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := GenerateSchema(tt.typ)
			if err != nil {
				t.Fatalf("Failed to generate schema: %v", err)
			}

			var schemaMap map[string]interface{}
			if err := json.Unmarshal(schema, &schemaMap); err != nil {
				t.Fatalf("Failed to unmarshal schema: %v", err)
			}

			if !tt.checkFn(schemaMap) {
				t.Errorf("Schema check failed for %s", tt.name)
			}
		})
	}
}

func TestGetDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]interface{}
		expected interface{}
	}{
		{
			name:     "String with default",
			schema:   map[string]interface{}{"type": "string", "default": "hello"},
			expected: "hello",
		},
		{
			name:     "String without default",
			schema:   map[string]interface{}{"type": "string"},
			expected: "",
		},
		{
			name:     "Integer without default",
			schema:   map[string]interface{}{"type": "integer"},
			expected: 0,
		},
		{
			name:     "Number without default",
			schema:   map[string]interface{}{"type": "number"},
			expected: 0.0,
		},
		{
			name:     "Boolean without default",
			schema:   map[string]interface{}{"type": "boolean"},
			expected: false,
		},
		{
			name:     "Array without default",
			schema:   map[string]interface{}{"type": "array"},
			expected: []interface{}{},
		},
		{
			name:     "Object without default",
			schema:   map[string]interface{}{"type": "object"},
			expected: make(map[string]interface{}),
		},
		{
			name:     "Null type",
			schema:   map[string]interface{}{"type": "null"},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDefaultValue(tt.schema)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}