package responseParser

import (
	"fmt"
	"reflect"
	"strings"
)

// generateFieldsOutput recursively builds the string for fields within a JSON object.
func generateFieldsOutput(b *strings.Builder, typ reflect.Type, indentLevel int) {
	var fieldsToProcess []reflect.StructField
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		// Skip fields explicitly ignored by JSON marshalling
		if getJSONFieldName(field) == "-" {
			continue
		}
		// Skip unexported fields as they are not marshalled by encoding/json by default
		if !field.IsExported() {
			continue
		}
		fieldsToProcess = append(fieldsToProcess, field)
	}

	baseIndent := strings.Repeat("  ", indentLevel)
	fieldIndent := baseIndent + "  "

	for i, field := range fieldsToProcess {
		jsonName := getJSONFieldName(field)
		description := getFieldDescription(field)
		fieldType := field.Type

		b.WriteString(fieldIndent + "\"" + jsonName + "\": ")

		// Determine the kind of the field, dereferencing pointers for kind-switch
		kindCheckType := fieldType
		if kindCheckType.Kind() == reflect.Ptr {
			kindCheckType = kindCheckType.Elem()
		}

		switch kindCheckType.Kind() {
		case reflect.String, reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			b.WriteString(mapGoTypeToJSONType(fieldType)) // Pass original type for mapping
			if description != "" {
				b.WriteString(", // " + description)
			}
		case reflect.Struct:
			b.WriteString("{\n")
			generateFieldsOutput(b, kindCheckType, indentLevel+1) // Use dereferenced type for recursion
			b.WriteString(fieldIndent + "}")
			if description != "" { // Description for the struct field itself
				b.WriteString(", // " + description)
			}
		case reflect.Slice, reflect.Array:
			b.WriteString("[\n")
			elemType := kindCheckType.Elem() // Element type of the slice/array

			// Determine the actual element type, dereferencing if it's a pointer (e.g., []*User)
			actualElemTypeForKindCheck := elemType
			if actualElemTypeForKindCheck.Kind() == reflect.Ptr {
				actualElemTypeForKindCheck = actualElemTypeForKindCheck.Elem()
			}

			elemLineDesc := "" // Description to be shown on the representative element line

			if actualElemTypeForKindCheck.Kind() == reflect.Struct {
				b.WriteString(fieldIndent + "  {") // Start of the example object in array

				var subFieldsToProcess []reflect.StructField
				for k := 0; k < actualElemTypeForKindCheck.NumField(); k++ {
					sf := actualElemTypeForKindCheck.Field(k)
					if getJSONFieldName(sf) == "-" || !sf.IsExported() {
						continue
					}
					subFieldsToProcess = append(subFieldsToProcess, sf)
				}

				objectDescCandidate := "" // Description from within the object's fields

				for si, subField := range subFieldsToProcess {
					subJSONName := getJSONFieldName(subField)
					b.WriteString("\"" + subJSONName + "\": " + mapGoTypeToJSONType(subField.Type))

					subFieldDesc := getFieldDescription(subField)
					if subFieldDesc != "" && objectDescCandidate == "" {
						// Capture the first description found in the object's fields
						objectDescCandidate = subFieldDesc
					}

					if si < len(subFieldsToProcess)-1 {
						b.WriteString(", ")
					}
				}
				b.WriteString("}") // End of the example object

				// Determine description for the array element line
				if objectDescCandidate != "" {
					elemLineDesc = objectDescCandidate
				} else if description != "" { // Fallback to the slice field's own description
					elemLineDesc = description
				}
				if elemLineDesc != "" {
					b.WriteString(", // " + elemLineDesc)
				}
				b.WriteString("\n" + fieldIndent + "  ...")
			} else { // Slice of basic types
				b.WriteString(fieldIndent + "  " + mapGoTypeToJSONType(elemType))
				if description != "" { // Description of the slice field itself
					elemLineDesc = description
					b.WriteString(", // " + elemLineDesc)
				}
				b.WriteString("\n" + fieldIndent + "  ...")
			}
			b.WriteString("\n" + fieldIndent + "]")
			// Per user examples, description for arrays (even slice field's own) appears on the element line.
			// If it's a slice field like `MyThings []Thing `desc:"..."``, and Thing has no field with desc,
			// then "desc for MyThings" will be used on the `Thing{...}, // desc for MyThings` line.
			// No separate description after the closing bracket `]` is added for the slice field.
		default:
			b.WriteString(mapGoTypeToJSONType(fieldType)) // Fallback for other types
			if description != "" {
				b.WriteString(", // " + description)
			}
		}

		if i < len(fieldsToProcess)-1 {
			b.WriteString(",\n")
		} else {
			b.WriteString("\n")
		}
	}
}

// ResponseInstructions generates guidance describing the JSON format expected
// based on the provided Go struct. This helper is a provider-agnostic fallback
// when strict JSON modes are unavailable. Prefer provider strict schema modes
// where supported (e.g., OpenAI json_schema, Gemini response schema) for higher determinism.
func ResponseInstructions(s interface{}) (string, error) {
	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() { // Handle nil pointer if necessary, or let Elem() panic
			return "", fmt.Errorf("input cannot be a nil pointer")
		}
		val = val.Elem() // Dereference pointer to get struct
	}

	if val.Kind() != reflect.Struct {
		return "", fmt.Errorf("input must be a struct or a pointer to a struct, got %s", val.Kind())
	}
	typ := val.Type()

	var b strings.Builder
	b.WriteString("\nFormat your response as a valid JSON object. Follow the below format:\n\n")
	b.WriteString("{\n")
	generateFieldsOutput(&b, typ, 0) // Initial indent level is 0
	b.WriteString("}")

	return b.String(), nil
}

func getJSONFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name == "" { // Handles cases like `json:",omitempty"`
		return field.Name
	}
	return name
}

// getFieldDescription extracts the description from the "description" struct tag.
func getFieldDescription(field reflect.StructField) string {
	return field.Tag.Get("desc")
}

// mapGoTypeToJSONType maps basic Go types (or their Ptr counterparts) to JSON type strings.
func mapGoTypeToJSONType(goType reflect.Type) string {
	// Dereference pointer types to get the underlying kind
	actualType := goType
	if actualType.Kind() == reflect.Ptr {
		actualType = actualType.Elem()
	}

	switch actualType.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	// Complex types like struct and slice are handled by generateFieldsOutput's main logic.
	// This function is primarily for the "type" part in "key: type".
	case reflect.Struct:
		return "object" // General placeholder if ever directly needed
	case reflect.Slice, reflect.Array:
		return "array" // General placeholder
	default:
		name := actualType.Name()
		if name != "" {
			return name // e.g. for custom types not directly mappable
		}
		return "unknown"
	}
}
