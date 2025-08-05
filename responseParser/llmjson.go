// Package llmjson provides robust parsing of LLM responses into structured data.
// It's designed to handle various edge cases and malformed JSON that are common
// in LLM outputs, with intelligent recovery mechanisms.
package responseParser

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/collinshill/gai/core"
	"github.com/collinshill/gai/responseParser/cleanup"
	"github.com/collinshill/gai/responseParser/coercer"
	"github.com/collinshill/gai/responseParser/parser"

	"github.com/pkg/errors"
)

// ParseOptions configures the behavior of the Parse functions.
type ParseOptions struct {
	// When true, uses relaxed parsing for comments, trailing commas, etc.
	Relaxed bool
	// When true, attempts to extract JSON even if embedded in other text
	Extract bool
	// When true, attempts to autocomplete partial JSON (missing braces, etc.)
	Autocomplete bool
	// When true, tries to fix common LLM formatting issues in JSON
	FixFormat bool
	// When true, allows smart type coercion between similar types (string->int, etc.)
	AllowCoercion bool
}

// DefaultOptions returns the recommended default parsing options
func DefaultOptions() ParseOptions {
	return ParseOptions{
		Relaxed:       true,
		Extract:       true,
		Autocomplete:  true,
		FixFormat:     true,
		AllowCoercion: true,
	}
}

// StrictOptions returns parsing options that enforce valid JSON
func StrictOptions() ParseOptions {
	return ParseOptions{
		Relaxed:       false,
		Extract:       false,
		Autocomplete:  false,
		FixFormat:     false,
		AllowCoercion: false,
	}
}

// Parse attempts to decode an LLM response into the target Go type.
// It uses a multi-layer approach to handle common LLM response issues.
func Parse[T any](raw string, opts ...ParseOptions) (T, error) {
	var zero T
	options := DefaultOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	// Clean and normalize the raw input
	cleaned, err := cleanup.Process(raw, cleanup.Options{
		StripMarkdown: options.Extract,
		ExtractJSON:   options.Extract,
		FixIncomplete: options.Autocomplete,
	})
	if err != nil {
		return zero, errors.Wrap(err, "preprocessing LLM response")
	}

	// Check for empty response or non-JSON content
	if len(cleaned) == 0 || (cleaned[0] != '{' && cleaned[0] != '[') {
		return zero, errors.New("no JSON object/array found in response")
	}

	// Convert to canonical JSON
	jsonBytes, err := parser.ToJSON(cleaned, parser.Options{
		Relaxed:       options.Relaxed,
		Autocomplete:  options.Autocomplete,
		FixFormatting: options.FixFormat,
	})
	if err != nil {
		return zero, errors.Wrap(err, "converting to canonical JSON")
	}

	// Standard path: direct unmarshal
	var out T
	if err := json.Unmarshal(jsonBytes, &out); err == nil {
		return out, nil
	}

	// Fallback path: try coercion if enabled
	if !options.AllowCoercion {
		return zero, errors.New("failed to unmarshal JSON and coercion is disabled")
	}

	var generic interface{}
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return zero, errors.Wrap(err, "unmarshalling to generic interface")
	}

	if err := coercer.Coerce(generic, &out); err != nil {
		return zero, errors.Wrap(err, "coercing JSON values to target type")
	}

	return out, nil
}

// ParseInto decodes the LLM response into the value pointed to by target.
// It follows the same approach as Parse but uses an existing pointer.
func ParseInto(raw string, target interface{}, opts ...ParseOptions) error {
	// Ensure target is a non-nil pointer
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("target must be a non-nil pointer")
	}

	options := DefaultOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	// Clean and normalize the raw input
	cleaned, err := cleanup.Process(raw, cleanup.Options{
		StripMarkdown: options.Extract,
		ExtractJSON:   options.Extract,
		FixIncomplete: options.Autocomplete,
	})
	if err != nil {
		return errors.Wrap(err, "preprocessing LLM response")
	}

	// Check for empty response or non-JSON content
	if len(cleaned) == 0 {
		return errors.New("empty response after preprocessing")
	}
	if cleaned[0] != '{' && cleaned[0] != '[' {
		return fmt.Errorf("content doesn't appear to be JSON: %q", cleaned[:min(len(cleaned), 20)])
	}

	// Convert to canonical JSON
	jsonBytes, err := parser.ToJSON(cleaned, parser.Options{
		Relaxed:       options.Relaxed,
		Autocomplete:  options.Autocomplete,
		FixFormatting: options.FixFormat,
	})
	if err != nil {
		return errors.Wrap(err, "converting to canonical JSON")
	}

	// Standard path: direct unmarshal
	if err := json.Unmarshal(jsonBytes, target); err == nil {
		return nil
	}

	// Fallback path: try coercion if enabled
	if !options.AllowCoercion {
		return errors.New("failed to unmarshal JSON and coercion is disabled")
	}

	var generic interface{}
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return errors.Wrap(err, "unmarshalling to generic interface")
	}

	return coercer.Coerce(generic, target)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// LLMClient defines the interface for LLM clients that can generate completions
type LLMClient interface {
	// GetCompletion makes a completion request to the LLM with the provided context and arguments
	GetCompletion(context.Context, core.LLMCallParts) (core.LLMResponse, error)
}

// GetResponseObject gets a response from the LLM and parses it into the target struct v
// It will retry the LLM call once if parsing fails
func GetResponseObject(ctx context.Context, client LLMClient, v any, parts core.LLMCallParts) error {
	llmResponse, err := client.GetCompletion(ctx, parts)
	if err != nil {
		return fmt.Errorf("failed to get LLM response: %w", err)
	}
	response := llmResponse.Content
	if err := ParseInto(response, v); err == nil {
		return nil
	}

	// If we failed to parse the response, try again with a different prompt
	parts.Messages = append(parts.Messages, core.Message{
		Role: "user",
		Contents: []core.Content{
			core.TextContent{Text: "Your previous response was not valid JSON. Please fix it and respond with only valid JSON."},
		},
	})

	llmResponse, err = client.GetCompletion(ctx, parts)
	if err != nil {
		return fmt.Errorf("failed to get LLM response on retry: %w", err)
	}
	response = llmResponse.Content
	if err := ParseInto(response, v); err != nil {
		return fmt.Errorf("failed to parse LLM response: %w", err)
	}
	return nil
}
