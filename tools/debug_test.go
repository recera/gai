package tools

import (
	"encoding/json"
	"reflect"
	"testing"
)

// DebugTestStruct is an exported struct for testing schema generation
type DebugTestStruct struct {
	StringField string  `json:"string_field"`
	IntField    int     `json:"int_field"`
	FloatField  float64 `json:"float_field"`
	BoolField   bool    `json:"bool_field"`
}

func TestDebugSchema(t *testing.T) {
	schema, err := GenerateSchema(reflect.TypeOf(DebugTestStruct{}))
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	// Pretty print the schema for debugging
	var schemaObj interface{}
	if err := json.Unmarshal(schema, &schemaObj); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	pretty, err := json.MarshalIndent(schemaObj, "", "  ")
	if err != nil {
		t.Fatalf("Failed to format schema: %v", err)
	}

	t.Logf("Generated schema:\n%s", string(pretty))
	
	// Also test with BasicStruct from schema_test.go
	schema2, err := GenerateSchema(reflect.TypeOf(BasicStruct{}))
	if err != nil {
		t.Fatalf("Failed to generate schema for BasicStruct: %v", err)
	}
	
	var schemaObj2 interface{}
	if err := json.Unmarshal(schema2, &schemaObj2); err != nil {
		t.Fatalf("Failed to unmarshal schema2: %v", err)
	}
	
	pretty2, err := json.MarshalIndent(schemaObj2, "", "  ")
	if err != nil {
		t.Fatalf("Failed to format schema2: %v", err)
	}
	
	t.Logf("BasicStruct schema:\n%s", string(pretty2))
}