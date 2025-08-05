// Package parser provides utilities for robust parsing of JSON-like strings.
// It handles common edge cases in LLM output such as non-standard quotes,
// comments, trailing commas, and missing delimiters.
package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Options configures the behavior of the JSON parser
type Options struct {
	// When true, uses relaxed parsing (comments, trailing commas, etc.)
	Relaxed bool
	// When true, attempts to autocomplete partial JSON
	Autocomplete bool
	// When true, fixes common formatting issues in JSON
	FixFormatting bool
}

// DefaultOptions returns the recommended default parser options
func DefaultOptions() Options {
	return Options{
		Relaxed:       true,
		Autocomplete:  true,
		FixFormatting: true,
	}
}

// ToJSON converts LLM response text to canonical JSON bytes.
// It handles JSON-like formats and various edge cases in LLM outputs.
func ToJSON(raw string, opts Options) ([]byte, error) {
	if raw == "" {
		return nil, errors.New("empty input")
	}

	// Attempt to fix formatting issues if enabled
	input := raw
	if opts.FixFormatting {
		input = fixCommonFormatting(input)
	}

	// Try direct JSON parsing first (fast path)
	rawBytes := []byte(input)
	if json.Valid(rawBytes) {
		// Already valid JSON, just normalize it to ensure consistent output
		var parsed interface{}
		if err := json.Unmarshal(rawBytes, &parsed); err != nil {
			// This should never happen as we already validated the JSON
			return nil, errors.Wrap(err, "unmarshalling validated JSON")
		}
		return json.Marshal(parsed)
	}

	// If relaxed parsing is enabled, try YAML-based parsing
	// which handles single quotes, trailing commas, etc.
	if opts.Relaxed {
		var yamlData interface{}
		if err := yaml.Unmarshal([]byte(input), &yamlData); err == nil {
			// Convert from YAML representation to JSON
			jsonBytes, err := json.Marshal(yamlData)
			if err == nil && json.Valid(jsonBytes) {
				return jsonBytes, nil
			}
		}
	}

	// If autocomplete is enabled, try to fix incomplete structures
	if opts.Autocomplete {
		// Apply more aggressive autocorrection for malformed JSON
		corrected := autocompleteMalformedJSON(input)
		if corrected != input {
			// Re-attempt parsing with the corrected input
			return ToJSON(corrected, Options{
				Relaxed:       opts.Relaxed,
				Autocomplete:  false, // Prevent infinite recursion
				FixFormatting: opts.FixFormatting,
			})
		}
	}

	// If we've tried everything and still failed, attempt character-by-character parsing
	if opts.Relaxed {
		result, err := characterByCharacterParse(input)
		if err == nil {
			return result, nil
		}
	}

	return nil, errors.New("failed to parse input as JSON with all available methods")
}

// fixCommonFormatting addresses common formatting issues in LLM-generated JSON
func fixCommonFormatting(input string) string {
	// Replace all variations of double quotes with standard double quotes
	input = regexp.MustCompile("[\u201c\u201d]").ReplaceAllString(input, "\"")

	// Replace single quotes with double quotes, but not within a string
	// This is complex so we'll use a more basic approach that catches most cases
	input = replaceSingleQuotes(input)

	// Replace JavaScript-style comments
	input = removeJSComments(input)

	// Fix common LLM errors like using => instead of :
	input = regexp.MustCompile(`(\s*)(=>)(\s*)`).ReplaceAllString(input, "$1:$3")

	// Fix trailing commas before closing braces/brackets
	input = regexp.MustCompile(`,(\s*[\}\]])`).ReplaceAllString(input, "$1")

	// Fix missing quotes around keys
	input = fixUnquotedKeys(input)

	// Fix boolean and null values that might be capitalized
	input = regexp.MustCompile(`(?i)\btrue\b`).ReplaceAllString(input, "true")
	input = regexp.MustCompile(`(?i)\bfalse\b`).ReplaceAllString(input, "false")
	input = regexp.MustCompile(`(?i)\bnull\b`).ReplaceAllString(input, "null")

	return input
}

// replaceSingleQuotes replaces single quotes with double quotes, being careful
// not to replace single quotes inside double-quoted strings
func replaceSingleQuotes(input string) string {
	var result bytes.Buffer
	inDoubleQuotes := false
	inSingleQuotes := false
	lastChar := ' '

	for _, ch := range input {
		if ch == '"' && lastChar != '\\' {
			inDoubleQuotes = !inDoubleQuotes
		} else if ch == '\'' && lastChar != '\\' {
			if inDoubleQuotes {
				// Don't change single quotes inside double quotes
				result.WriteRune(ch)
			} else {
				if inSingleQuotes {
					result.WriteRune('"')
					inSingleQuotes = false
				} else {
					result.WriteRune('"')
					inSingleQuotes = true
				}
				lastChar = ch
				continue
			}
		}

		result.WriteRune(ch)
		lastChar = ch
	}

	return result.String()
}

// removeJSComments removes JavaScript-style comments from JSON
func removeJSComments(input string) string {
	// Remove single line comments
	re := regexp.MustCompile(`//.*?\n`)
	input = re.ReplaceAllString(input, "\n")

	// Remove multi-line comments
	re = regexp.MustCompile(`(?s)/\*.*?\*/`)
	input = re.ReplaceAllString(input, "")

	return input
}

// fixUnquotedKeys adds quotes to keys that should be quoted in JSON
func fixUnquotedKeys(input string) string {
	// This regex looks for patterns like {key: or {key, or key:
	re := regexp.MustCompile(`([{,]\s*)([a-zA-Z_][a-zA-Z0-9_]*)(\s*:)`)
	return re.ReplaceAllString(input, `$1"$2"$3`)
}

// autocompleteMalformedJSON attempts to fix incomplete JSON structures
func autocompleteMalformedJSON(input string) string {
	stack := []rune{}
	inString := false
	var stringDelim rune
	var result strings.Builder

	for _, ch := range input {
		result.WriteRune(ch)

		if inString {
			if ch == '\\' {
				// Skip next character if it's an escape sequence
				continue
			}
			if ch == stringDelim {
				inString = false
			}
			continue
		}

		switch ch {
		case '"', '\'':
			inString = true
			stringDelim = ch
		case '{':
			stack = append(stack, '}')
		case '[':
			stack = append(stack, ']')
		case '}', ']':
			if len(stack) > 0 && stack[len(stack)-1] == ch {
				stack = stack[:len(stack)-1]
			}
		}
	}

	// Add missing closing delimiters
	for i := len(stack) - 1; i >= 0; i-- {
		result.WriteRune(stack[i])
	}

	return result.String()
}

// characterByCharacterParse implements a more lenient parser that tries to parse
// the input character by character, handling more edge cases
func characterByCharacterParse(input string) ([]byte, error) {
	// Placeholder implementation - in a real implementation, this would be
	// a state machine that processes character by character

	// Convert special characters to their JSON equivalents
	var buffer bytes.Buffer
	inString := false
	prevChar := ' '

	for i, ch := range input {
		switch {
		case ch == '"' && prevChar != '\\':
			inString = !inString
			buffer.WriteRune(ch)
		case inString:
			// Inside a string, write character as is
			buffer.WriteRune(ch)
		case ch == '\'' && prevChar != '\\':
			// Convert single quotes to double quotes outside strings
			buffer.WriteRune('"')
		case ch == ',' && (i+1 < len(input) && (input[i+1] == ']' || input[i+1] == '}')):
			// Skip trailing commas
			continue
		default:
			buffer.WriteRune(ch)
		}
		prevChar = ch
	}

	result := buffer.String()

	// Try to parse the result
	if json.Valid([]byte(result)) {
		var parsed interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			return nil, err
		}
		return json.Marshal(parsed)
	}

	return nil, fmt.Errorf("character-by-character parsing failed")
}
