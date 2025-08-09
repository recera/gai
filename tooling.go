package gai

import (
	"context"
	"fmt"
	"reflect"

	rp "github.com/recera/gai/responseParser"
)

// ToolGenOptions controls how tool schemas are generated from Go structs
type ToolGenOptions struct {
	// Description applied to the overall tool; if empty we'll attempt to infer from type name
	Description string
	// Additional free-form documentation appended to the description
	Doc string
	// If true, allow extra fields that are not in the struct
	AdditionalProperties bool
}

// ToolFromStruct generates a provider-native tool definition from a Go struct value.
// Field descriptions are taken from the `desc:"..."` tag. Required fields are those
// without `omitempty` in their `json:"..."` tag and that are not pointers.
func ToolFromStruct(name string, s interface{}, opts ...ToolGenOptions) (ToolDefinition, error) {
	if name == "" {
		return ToolDefinition{}, fmt.Errorf("tool name is required")
	}
	var options ToolGenOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	t := reflect.TypeOf(s)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ToolDefinition{}, fmt.Errorf("ToolFromStruct expects a struct or pointer to struct, got %s", t.Kind())
	}

	schema := map[string]interface{}{
		"type":                 "object",
		"properties":           map[string]interface{}{},
		"required":             []string{},
		"additionalProperties": options.AdditionalProperties,
	}

	properties := schema["properties"].(map[string]interface{})
	required := schema["required"].([]string)

	// Build properties recursively
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" { // unexported
			continue
		}
		jsonName, omitempty := jsonFieldName(field)
		if jsonName == "-" {
			continue
		}
		prop := buildJSONSchemaForType(field.Type, field.Tag.Get("desc"))
		if prop != nil {
			properties[jsonName] = prop
			// required if not omitempty and not pointer
			if !omitempty && field.Type.Kind() != reflect.Ptr {
				required = append(required, jsonName)
			}
		}
	}

	// Fix empty required to be omitted for cleanliness
	if len(required) == 0 {
		delete(schema, "required")
	} else {
		schema["required"] = required
	}

	// Tool description
	desc := options.Description
	if options.Doc != "" {
		if desc != "" {
			desc += "\n\n" + options.Doc
		} else {
			desc = options.Doc
		}
	}

	return ToolDefinition{
		Name:        name,
		Description: desc,
		JSONSchema:  schema,
	}, nil
}

// ToolFromType is a generic helper to generate a tool from a type parameter
func ToolFromType[T any](name string, opts ...ToolGenOptions) (ToolDefinition, error) {
	var zero T
	return ToolFromStruct(name, zero, opts...)
}

func jsonFieldName(field reflect.StructField) (name string, omitempty bool) {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name, false
	}
	parts := splitCSV(tag)
	if len(parts) == 0 || parts[0] == "" {
		return field.Name, containsString(parts, "omitempty")
	}
	return parts[0], containsString(parts, "omitempty")
}

// buildJSONSchemaForType returns a JSON Schema (as map) for a Go type
func buildJSONSchemaForType(t reflect.Type, desc string) map[string]interface{} {
	// Resolve pointer
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		m := map[string]interface{}{"type": "string"}
		if desc != "" {
			m["description"] = desc
		}
		return m
	case reflect.Bool:
		m := map[string]interface{}{"type": "boolean"}
		if desc != "" {
			m["description"] = desc
		}
		return m
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		m := map[string]interface{}{"type": "integer"}
		if desc != "" {
			m["description"] = desc
		}
		return m
	case reflect.Float32, reflect.Float64:
		m := map[string]interface{}{"type": "number"}
		if desc != "" {
			m["description"] = desc
		}
		return m
	case reflect.Slice, reflect.Array:
		item := buildJSONSchemaForType(t.Elem(), "")
		if item == nil {
			item = map[string]interface{}{"type": "string"}
		}
		m := map[string]interface{}{
			"type":  "array",
			"items": item,
		}
		if desc != "" {
			m["description"] = desc
		}
		return m
	case reflect.Map:
		// generic object map
		m := map[string]interface{}{"type": "object"}
		if desc != "" {
			m["description"] = desc
		}
		return m
	case reflect.Struct:
		// Handle well-known structs like time.Time as string/date-time
		if t.PkgPath() == "time" && t.Name() == "Time" {
			m := map[string]interface{}{"type": "string", "format": "date-time"}
			if desc != "" {
				m["description"] = desc
			}
			return m
		}
		// Nested object schema
		nested := map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
		props := nested["properties"].(map[string]interface{})
		var req []string
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			name, om := jsonFieldName(f)
			if name == "-" {
				continue
			}
			p := buildJSONSchemaForType(f.Type, f.Tag.Get("desc"))
			if p != nil {
				props[name] = p
			}
			if !om && f.Type.Kind() != reflect.Ptr {
				req = append(req, name)
			}
		}
		if len(req) > 0 {
			nested["required"] = req
		}
		if desc != "" {
			nested["description"] = desc
		}
		return nested
	default:
		return map[string]interface{}{"type": "string"}
	}
}

func splitCSV(tag string) []string {
	if tag == "" {
		return nil
	}
	var parts []string
	start := 0
	for i := 0; i < len(tag); i++ {
		if tag[i] == ',' {
			parts = append(parts, tag[start:i])
			start = i + 1
		}
	}
	parts = append(parts, tag[start:])
	return parts
}

func containsString(parts []string, needle string) bool {
	for _, p := range parts {
		if p == needle {
			return true
		}
	}
	return false
}

// GetResponseObjectViaTools builds a tool from v's type and drives the tool-calling loop.
// When the model calls the tool with JSON arguments, they are parsed into v, and the call ends.
func GetResponseObjectViaTools(ctx context.Context, c LLMClient, parts LLMCallParts, toolName string, v any, opts ...ToolGenOptions) error {
	tool, err := ToolFromStruct(toolName, v, opts...)
	if err != nil {
		return err
	}
	parts.Tools = []ToolDefinition{tool}
	_, err = c.RunWithTools(ctx, parts, func(call ToolCall) (string, error) {
		if call.Name != toolName {
			return "", fmt.Errorf("unexpected tool: %s", call.Name)
		}
		// Parse tool arguments directly into v
		if err := rp.ParseInto(call.Arguments, v); err != nil {
			return "", err
		}
		// Returning an empty string since the tool's purpose is to carry structured output
		return "", nil
	})
	return err
}
