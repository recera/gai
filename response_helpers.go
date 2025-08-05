package gai

import (
	"github.com/collinshill/gai/responseParser"
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