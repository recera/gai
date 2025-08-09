package gai

import (
	"context"
	"encoding/json"

	"github.com/recera/gai/core"
	"github.com/recera/gai/responseParser"
)

// ResponseInstructions generates instructions for the LLM to format its response
// according to the provided struct. This is useful for structured outputs.
func ResponseInstructions(s interface{}) (string, error) {
	return responseParser.ResponseInstructions(s)
}

// ParseInto parses an LLM response into the provided struct.
// It uses intelligent parsing that can handle various edge cases and malformed JSON.
func ParseInto(raw string, target interface{}) error {
	return responseParser.ParseInto(raw, target)
}

// GenerateObject attempts to retrieve a structured JSON object of type T.
// If the provider supports strict JSON schema modes, it will be engaged using
// the JSON Schema derived from T. Otherwise, it falls back to tolerant parsing.
func GenerateObject[T any](ctx context.Context, c LLMClient, parts core.LLMCallParts) (T, core.TokenUsage, error) {
	var zero T
	// Try provider strict mode via ProviderOpts when possible. We encode a JSON Schema for T
	// as a minimal, provider-agnostic map. For now, we rely on ToolFromStruct-like schema
	// generation semantics exposed via tooling.go (JSONSchema generation for structs).
	schemaTool, err := ToolFromType[T]("object")
	if err == nil && schemaTool.JSONSchema != nil {
		// For OpenAI strict object mode: response_format = { type: "json_schema", json_schema: { name, schema, strict:true } }
		if parts.Provider == "openai" {
			rf := map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "gai_object", "schema": schemaTool.JSONSchema, "strict": true}}
			if parts.ProviderOpts == nil {
				parts.ProviderOpts = map[string]any{}
			}
			parts.ProviderOpts["response_format"] = rf
		}
		// Gemini can use response_mime_type + schema (to be implemented in provider). We just set ProviderOpts hint.
		if parts.Provider == "gemini" {
			if parts.ProviderOpts == nil {
				parts.ProviderOpts = map[string]any{}
			}
			parts.ProviderOpts["response_schema"] = schemaTool.JSONSchema
		}
	}

	resp, err := c.GetCompletion(ctx, parts)
	if err != nil {
		return zero, core.TokenUsage{}, err
	}
	// Fast path: try strict JSON unmarshal first
	if err := json.Unmarshal([]byte(resp.Content), &zero); err == nil {
		return zero, resp.Usage, nil
	}
	// Fallback to tolerant parser
	out, perr := responseParser.Parse[T](resp.Content)
	if perr == nil {
		return out, resp.Usage, nil
	}
	return zero, resp.Usage, perr
}
