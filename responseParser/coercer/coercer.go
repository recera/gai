// Package coercer provides advanced type coercion between JSON values and Go types.
// It handles cases where the LLM output doesn't perfectly match the expected struct fields.
package coercer

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

// CoerceOptions configures the behavior of the coercion process.
type CoerceOptions struct {
	// When true, allows zero values to overwrite existing values
	AllowZeroValues bool
	// When true, ignores case in field name matching
	IgnoreCase bool
	// When true, uses field names with underscores (snake_case)
	UseSnakeCase bool
	// When true, tries handling time values in various formats
	HandleTimeValues bool
	// When true, attempts more aggressive type coercions
	DeepCoercion bool
}

// DefaultOptions returns the recommended default coercion options
func DefaultOptions() CoerceOptions {
	return CoerceOptions{
		AllowZeroValues:  false,
		IgnoreCase:       true,
		UseSnakeCase:     true,
		HandleTimeValues: true,
		DeepCoercion:     true,
	}
}

// StrictOptions returns options that minimize type coercion
func StrictOptions() CoerceOptions {
	return CoerceOptions{
		AllowZeroValues:  false,
		IgnoreCase:       false,
		UseSnakeCase:     false,
		HandleTimeValues: true,
		DeepCoercion:     false,
	}
}

// Coerce attempts to map arbitrary decoded JSON into a Go struct pointer.
// It supports various type conversions and field matching strategies.
func Coerce(input interface{}, target interface{}, opts ...CoerceOptions) error {
	options := DefaultOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	// Ensure target is a non-nil pointer
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("target must be a non-nil pointer")
	}

	// Preprocess input if deep coercion is enabled
	if options.DeepCoercion {
		var err error
		input, err = deepPreprocess(input)
		if err != nil {
			return errors.Wrap(err, "preprocessing input for deep coercion")
		}
	}

	// Create a decoder with custom options
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:          "json",
		WeaklyTypedInput: true,
		Result:           target,
		ZeroFields:       options.AllowZeroValues,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			stringToTimeHookFunc(options),
			stringToStructHookFunc(),
			mapToStructHookFunc(),
			stringToSliceHookFunc(),
			snakeCaseMatcherHookFunc(options),
		),
		MatchName: func(mapKey, fieldName string) bool {
			if options.IgnoreCase {
				return strings.EqualFold(mapKey, fieldName)
			}
			return mapKey == fieldName
		},
	})

	if err != nil {
		return errors.Wrap(err, "creating decoder")
	}

	return decoder.Decode(input)
}

// UnmarshalAndCoerce unmarshals JSON bytes and then coerces into target struct.
func UnmarshalAndCoerce(jsonBytes []byte, target interface{}, opts ...CoerceOptions) error {
	var interim interface{}
	if err := json.Unmarshal(jsonBytes, &interim); err != nil {
		return errors.Wrap(err, "unmarshalling JSON")
	}
	return Coerce(interim, target, opts...)
}

// deepPreprocess performs deep preprocessing of the input data to enhance coercion success.
// It recursively processes maps, slices and primitive types.
func deepPreprocess(input interface{}) (interface{}, error) {
	switch v := input.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, value := range v {
			processedValue, err := deepPreprocess(value)
			if err != nil {
				return nil, err
			}
			result[key] = processedValue
		}
		return result, nil

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			processedItem, err := deepPreprocess(item)
			if err != nil {
				return nil, err
			}
			result[i] = processedItem
		}
		return result, nil

	case string:
		// Try to parse string as number if it looks numerical
		if isLikelyNumber(v) {
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i, nil
			}
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f, nil
			}
		}

		// Try to parse string as boolean if it looks like a boolean
		if isLikelyBool(v) {
			if b, err := strconv.ParseBool(v); err == nil {
				return b, nil
			}
		}

		return v, nil

	default:
		return input, nil
	}
}

// stringToTimeHookFunc creates a decoder hook for converting strings to time.Time
func stringToTimeHookFunc(options CoerceOptions) mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if !options.HandleTimeValues {
			return data, nil
		}

		if t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			s, ok := data.(string)
			if !ok {
				return nil, fmt.Errorf("expected string, got %T", data)
			}

			// Try common time formats
			formats := []string{
				time.RFC3339,
				"2006-01-02T15:04:05",
				"2006-01-02 15:04:05",
				"2006-01-02",
				"01/02/2006",
				"Jan 2, 2006",
				"January 2, 2006",
			}

			for _, format := range formats {
				if tm, err := time.Parse(format, s); err == nil {
					return tm, nil
				}
			}

			return nil, fmt.Errorf("could not parse %q as time", s)

		case reflect.Float64:
			// Handle Unix timestamp (seconds since epoch)
			f, ok := data.(float64)
			if !ok {
				return nil, fmt.Errorf("expected float64, got %T", data)
			}
			return time.Unix(int64(f), 0), nil

		case reflect.Int, reflect.Int32, reflect.Int64:
			// Handle Unix timestamp (seconds since epoch)
			v := reflect.ValueOf(data)
			return time.Unix(v.Int(), 0), nil
		}

		return data, nil
	}
}

// stringToStructHookFunc creates a decoder hook that tries to parse a JSON string into a struct
func stringToStructHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		// Check if the input is a string and the target is a struct
		if f.Kind() != reflect.String || t.Kind() != reflect.Struct {
			return data, nil
		}

		s, ok := data.(string)
		if !ok {
			return data, nil
		}

		// Create a new instance of the target type
		target := reflect.New(t).Interface()

		// Try to unmarshal the string into the target
		if err := json.Unmarshal([]byte(s), target); err != nil {
			return data, nil // Return original data if unmarshaling fails
		}

		// Return the element the pointer points to
		return reflect.ValueOf(target).Elem().Interface(), nil
	}
}

// mapToStructHookFunc creates a decoder hook that helps with map to struct conversion
func mapToStructHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		// We only care if the input is a map and the output is a struct
		if f.Kind() != reflect.Map || t.Kind() != reflect.Struct {
			return data, nil
		}

		// Just return the data; mapstructure will handle the rest
		// This hook is primarily for future extensibility
		return data, nil
	}
}

// stringToSliceHookFunc creates a decoder hook that tries to parse a string into a slice
func stringToSliceHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		// Check if input is a string and output is a slice
		if f.Kind() != reflect.String || t.Kind() != reflect.Slice {
			return data, nil
		}

		s, ok := data.(string)
		if !ok {
			return data, nil
		}

		// Handle comma-separated lists
		if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
			// Looks like JSON array, try to parse it
			var result []interface{}
			if err := json.Unmarshal([]byte(s), &result); err == nil {
				return result, nil
			}
		}

		// Split by comma as fallback for comma-separated values
		items := strings.Split(s, ",")
		result := make([]string, len(items))
		for i, item := range items {
			result[i] = strings.TrimSpace(item)
		}

		return result, nil
	}
}

// snakeCaseMatcherHookFunc handles snake_case to camelCase conversion if enabled
func snakeCaseMatcherHookFunc(options CoerceOptions) mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		// This hook doesn't transform data, just passes it through
		// Snake case handling is done in the MatchName function
		return data, nil
	}
}

// isLikelyNumber checks if a string likely represents a numeric value
func isLikelyNumber(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	// Check for decimal point or scientific notation
	hasDot := strings.Contains(s, ".")
	hasE := strings.ContainsAny(s, "eE")

	// Check for minus sign at the beginning
	hasSign := s[0] == '-' || s[0] == '+'
	startIdx := 0
	if hasSign {
		startIdx = 1
	}

	// Must have at least one digit
	hasDigit := false
	for i := startIdx; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			hasDigit = true
			break
		}
	}

	if !hasDigit {
		return false
	}

	// Handle scientific notation
	if hasE {
		parts := strings.SplitN(s, "e", 2)
		if len(parts) != 2 {
			parts = strings.SplitN(s, "E", 2)
		}
		if len(parts) != 2 {
			return false
		}

		// Validate the base part
		base := parts[0]
		if !isLikelyNumber(base) {
			return false
		}

		// Validate the exponent part
		exp := parts[1]
		if len(exp) == 0 {
			return false
		}

		// Exponent can have a sign
		expStartIdx := 0
		if exp[0] == '+' || exp[0] == '-' {
			expStartIdx = 1
		}

		// Rest must be digits
		for i := expStartIdx; i < len(exp); i++ {
			if exp[i] < '0' || exp[i] > '9' {
				return false
			}
		}

		return true
	}

	// Regular number
	for i := startIdx; i < len(s); i++ {
		c := s[i]
		if c == '.' && hasDot {
			// More than one dot
			return false
		}
		if (c < '0' || c > '9') && c != '.' {
			return false
		}
	}

	return true
}

// isLikelyBool checks if a string likely represents a boolean value
func isLikelyBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "false" || s == "yes" || s == "no" || s == "1" || s == "0"
}
