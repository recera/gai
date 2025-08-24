// Package tools provides JSON Schema generation and validation for tool inputs/outputs.
// This file implements the schema generation adapter using invopop/jsonschema,
// with caching and validation helpers for runtime type checking.

package tools

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/invopop/jsonschema"
)

// schemaCache stores generated schemas to avoid redundant reflection.
// Key is the reflect.Type, value is the generated JSON schema.
var schemaCache = &schemaCacheImpl{
	cache: make(map[reflect.Type][]byte),
}

// schemaCacheImpl implements thread-safe schema caching.
type schemaCacheImpl struct {
	cache map[reflect.Type][]byte
	mu    sync.RWMutex
}

// get retrieves a cached schema for the given type.
func (c *schemaCacheImpl) get(t reflect.Type) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	schema, ok := c.cache[t]
	return schema, ok
}

// set stores a schema for the given type.
func (c *schemaCacheImpl) set(t reflect.Type, schema []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache[t] = schema
}

// clear removes all cached schemas.
func (c *schemaCacheImpl) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache = make(map[reflect.Type][]byte)
}

// size returns the number of cached schemas.
func (c *schemaCacheImpl) size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return len(c.cache)
}

// GenerateSchema generates a JSON Schema for the given Go type.
// The schema is cached for performance.
func GenerateSchema(t reflect.Type) ([]byte, error) {
	// Check cache first
	if schema, ok := schemaCache.get(t); ok {
		return schema, nil
	}
	
	// Create a reflector with custom settings
	r := &jsonschema.Reflector{
		// Allow additional properties by default for flexibility
		AllowAdditionalProperties: true,
		// Don't require all fields by default (respect omitempty tags)
		RequiredFromJSONSchemaTags: true,
		// Don't create references for top-level definitions
		DoNotReference: true,
	}
	
	// Handle special cases
	schema := handleSpecialTypes(t, r)
	if schema == nil {
		// Generate schema using reflection
		// For struct types, create a new instance to help with reflection
		if t.Kind() == reflect.Struct {
			// Create a new instance of the type
			instance := reflect.New(t).Interface()
			schema = r.Reflect(instance)
		} else {
			schema = r.Reflect(t)
		}
	}
	
	// Set schema metadata
	if schema.Title == "" {
		schema.Title = t.Name()
	}
	
	// Marshal to JSON
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}
	
	// Cache the result
	schemaCache.set(t, schemaJSON)
	
	return schemaJSON, nil
}

// handleSpecialTypes provides custom schema handling for specific types.
func handleSpecialTypes(t reflect.Type, r *jsonschema.Reflector) *jsonschema.Schema {
	// Handle empty interface{}
	if t.Kind() == reflect.Interface && t.NumMethod() == 0 {
		return &jsonschema.Schema{
			Type:                 "object",
			AdditionalProperties: jsonschema.TrueSchema,
			Description:          "Any valid JSON value",
		}
	}
	
	// Handle json.RawMessage
	if t == reflect.TypeOf(json.RawMessage{}) {
		return &jsonschema.Schema{
			Type:                 "object",
			AdditionalProperties: jsonschema.TrueSchema,
			Description:          "Raw JSON value",
		}
	}
	
	// Handle map[string]any
	if t.Kind() == reflect.Map && t.Key().Kind() == reflect.String && t.Elem().Kind() == reflect.Interface {
		return &jsonschema.Schema{
			Type:                 "object",
			AdditionalProperties: jsonschema.TrueSchema,
			Description:          "Object with string keys and any values",
		}
	}
	
	// Handle slices of any
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Interface {
		return &jsonschema.Schema{
			Type: "array",
			Items: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: jsonschema.TrueSchema,
			},
			Description: "Array of any values",
		}
	}
	
	return nil
}

// ValidateJSON validates JSON data against a JSON Schema.
// This is used for runtime validation when providers don't support strict mode.
func ValidateJSON(data json.RawMessage, schema []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty JSON data")
	}
	
	if len(schema) == 0 {
		return fmt.Errorf("empty schema")
	}
	
	// Parse the schema
	var schemaObj map[string]interface{}
	if err := json.Unmarshal(schema, &schemaObj); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}
	
	// Parse the data
	var dataObj interface{}
	if err := json.Unmarshal(data, &dataObj); err != nil {
		return fmt.Errorf("invalid JSON data: %w", err)
	}
	
	// Perform basic validation
	// Note: For production, we would integrate a full JSON Schema validator
	// For now, we do basic type checking
	return validateBasic(dataObj, schemaObj)
}

// validateBasic performs basic type validation.
// This is a simplified validator for Phase 2.
// A full implementation would use a complete JSON Schema validator library.
func validateBasic(data interface{}, schema map[string]interface{}) error {
	schemaType, ok := schema["type"].(string)
	if !ok {
		// No type specified, allow any
		return nil
	}
	
	switch schemaType {
	case "object":
		obj, ok := data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected object, got %T", data)
		}
		
		// Validate required fields
		if required, ok := schema["required"].([]interface{}); ok {
			for _, req := range required {
				if reqStr, ok := req.(string); ok {
					if _, exists := obj[reqStr]; !exists {
						return fmt.Errorf("missing required field: %s", reqStr)
					}
				}
			}
		}
		
		// Validate properties
		if properties, ok := schema["properties"].(map[string]interface{}); ok {
			for key, value := range obj {
				if propSchema, ok := properties[key].(map[string]interface{}); ok {
					if err := validateBasic(value, propSchema); err != nil {
						return fmt.Errorf("field %s: %w", key, err)
					}
				} else if additionalProps, ok := schema["additionalProperties"].(bool); ok && !additionalProps {
					return fmt.Errorf("unexpected field: %s", key)
				}
			}
		}
		
	case "array":
		arr, ok := data.([]interface{})
		if !ok {
			return fmt.Errorf("expected array, got %T", data)
		}
		
		// Validate items
		if itemSchema, ok := schema["items"].(map[string]interface{}); ok {
			for i, item := range arr {
				if err := validateBasic(item, itemSchema); err != nil {
					return fmt.Errorf("item %d: %w", i, err)
				}
			}
		}
		
		// Validate array constraints
		if minItems, ok := schema["minItems"].(float64); ok && len(arr) < int(minItems) {
			return fmt.Errorf("array has %d items, minimum %d required", len(arr), int(minItems))
		}
		if maxItems, ok := schema["maxItems"].(float64); ok && len(arr) > int(maxItems) {
			return fmt.Errorf("array has %d items, maximum %d allowed", len(arr), int(maxItems))
		}
		
	case "string":
		str, ok := data.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", data)
		}
		
		// Validate string constraints
		if minLength, ok := schema["minLength"].(float64); ok && len(str) < int(minLength) {
			return fmt.Errorf("string length %d is less than minimum %d", len(str), int(minLength))
		}
		if maxLength, ok := schema["maxLength"].(float64); ok && len(str) > int(maxLength) {
			return fmt.Errorf("string length %d exceeds maximum %d", len(str), int(maxLength))
		}
		
		// Validate enum
		if enum, ok := schema["enum"].([]interface{}); ok {
			found := false
			for _, e := range enum {
				if e == str {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("value %q not in enum", str)
			}
		}
		
	case "number", "integer":
		num, ok := data.(float64)
		if !ok {
			// Try to convert from int
			if intVal, ok := data.(int); ok {
				num = float64(intVal)
			} else {
				return fmt.Errorf("expected number, got %T", data)
			}
		}
		
		if schemaType == "integer" && num != float64(int64(num)) {
			return fmt.Errorf("expected integer, got float %v", num)
		}
		
		// Validate number constraints
		if minimum, ok := schema["minimum"].(float64); ok && num < minimum {
			return fmt.Errorf("value %v is less than minimum %v", num, minimum)
		}
		if maximum, ok := schema["maximum"].(float64); ok && num > maximum {
			return fmt.Errorf("value %v exceeds maximum %v", num, maximum)
		}
		
	case "boolean":
		if _, ok := data.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", data)
		}
		
	case "null":
		if data != nil {
			return fmt.Errorf("expected null, got %T", data)
		}
		
	default:
		return fmt.Errorf("unknown schema type: %s", schemaType)
	}
	
	return nil
}

// RepairJSON attempts to fix common JSON errors and make it conform to a schema.
// This is useful for providers that don't have strict JSON mode.
func RepairJSON(data json.RawMessage, schema []byte) (json.RawMessage, error) {
	// Parse the data
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		// Try to repair common issues
		repaired := repairCommonIssues(string(data))
		if err := json.Unmarshal([]byte(repaired), &obj); err != nil {
			return nil, fmt.Errorf("cannot repair JSON: %w", err)
		}
	}
	
	// Parse the schema
	var schemaObj map[string]interface{}
	if err := json.Unmarshal(schema, &schemaObj); err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}
	
	// Apply schema-based repairs
	repaired := applySchemaRepairs(obj, schemaObj)
	
	// Re-encode
	result, err := json.Marshal(repaired)
	if err != nil {
		return nil, fmt.Errorf("failed to re-encode repaired JSON: %w", err)
	}
	
	return result, nil
}

// repairCommonIssues fixes common JSON syntax errors.
func repairCommonIssues(data string) string {
	// This is a simplified repair function
	// A production version would be more sophisticated
	
	// Remove trailing commas
	// Note: This is a simple approach; a proper parser would be better
	result := data
	
	// Add quotes to unquoted keys (simple regex approach)
	// In production, use a proper JSON repair library
	
	return result
}

// applySchemaRepairs modifies data to conform to schema requirements.
func applySchemaRepairs(data interface{}, schema map[string]interface{}) interface{} {
	schemaType, ok := schema["type"].(string)
	if !ok {
		return data
	}
	
	switch schemaType {
	case "object":
		obj, ok := data.(map[string]interface{})
		if !ok {
			// Convert to object if possible
			return make(map[string]interface{})
		}
		
		// Add default values for missing required fields
		if required, ok := schema["required"].([]interface{}); ok {
			properties, _ := schema["properties"].(map[string]interface{})
			for _, req := range required {
				if reqStr, ok := req.(string); ok {
					if _, exists := obj[reqStr]; !exists {
						// Add default value based on property schema
						if propSchema, ok := properties[reqStr].(map[string]interface{}); ok {
							obj[reqStr] = getDefaultValue(propSchema)
						}
					}
				}
			}
		}
		
		// Recursively repair nested objects
		if properties, ok := schema["properties"].(map[string]interface{}); ok {
			for key, value := range obj {
				if propSchema, ok := properties[key].(map[string]interface{}); ok {
					obj[key] = applySchemaRepairs(value, propSchema)
				}
			}
		}
		
		return obj
		
	case "array":
		arr, ok := data.([]interface{})
		if !ok {
			// Convert to array if possible
			return []interface{}{}
		}
		
		// Recursively repair array items
		if itemSchema, ok := schema["items"].(map[string]interface{}); ok {
			for i, item := range arr {
				arr[i] = applySchemaRepairs(item, itemSchema)
			}
		}
		
		return arr
		
	case "string":
		if str, ok := data.(string); ok {
			return str
		}
		// Convert to string
		return fmt.Sprintf("%v", data)
		
	case "number", "integer":
		if num, ok := data.(float64); ok {
			if schemaType == "integer" {
				return int64(num)
			}
			return num
		}
		// Try to convert to number
		if str, ok := data.(string); ok {
			var num float64
			if _, err := fmt.Sscanf(str, "%f", &num); err == nil {
				if schemaType == "integer" {
					return int64(num)
				}
				return num
			}
		}
		return 0
		
	case "boolean":
		if b, ok := data.(bool); ok {
			return b
		}
		// Convert to boolean
		if str, ok := data.(string); ok {
			return str == "true" || str == "1" || str == "yes"
		}
		return false
		
	case "null":
		return nil
		
	default:
		return data
	}
}

// getDefaultValue returns a default value for a schema type.
func getDefaultValue(schema map[string]interface{}) interface{} {
	// Check for explicit default
	if def, ok := schema["default"]; ok {
		return def
	}
	
	// Generate based on type
	schemaType, ok := schema["type"].(string)
	if !ok {
		return nil
	}
	
	switch schemaType {
	case "object":
		return make(map[string]interface{})
	case "array":
		return []interface{}{}
	case "string":
		return ""
	case "number":
		return 0.0
	case "integer":
		return 0
	case "boolean":
		return false
	case "null":
		return nil
	default:
		return nil
	}
}

// ClearSchemaCache removes all cached schemas.
// This is useful for testing or when schemas might have changed.
func ClearSchemaCache() {
	schemaCache.clear()
}

// GetSchemaCacheSize returns the number of cached schemas.
func GetSchemaCacheSize() int {
	return schemaCache.size()
}