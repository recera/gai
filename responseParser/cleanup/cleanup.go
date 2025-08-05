// Package cleanup provides preprocessing functions for LLM responses
// to prepare them for JSON parsing.
package cleanup

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// Options configures the behavior of the cleanup process
type Options struct {
	// When true, strips markdown code blocks and formatting
	StripMarkdown bool
	// When true, tries to extract JSON from mixed text
	ExtractJSON bool
	// When true, attempts to fix incomplete JSON structures
	FixIncomplete bool
}

// DefaultOptions returns the recommended default cleanup options
func DefaultOptions() Options {
	return Options{
		StripMarkdown: true,
		ExtractJSON:   true,
		FixIncomplete: true,
	}
}

// Process applies preprocessing steps to an LLM response based on options
func Process(raw string, opts Options) (string, error) {
	if raw == "" {
		return "", nil
	}

	result := raw

	// Step 1: Strip markdown if enabled
	if opts.StripMarkdown {
		result = stripMarkdown(result)
	}

	// Step 2: Extract JSON if enabled
	if opts.ExtractJSON {
		extracted, err := extractJSON(result)
		if err != nil {
			return "", errors.Wrap(err, "extracting JSON")
		}
		result = extracted
	}

	// Step 3: Fix incomplete JSON if enabled
	if opts.FixIncomplete {
		result = fixIncompleteJSON(result)
	}

	return strings.TrimSpace(result), nil
}

// stripMarkdown removes markdown formatting, especially code fences
func stripMarkdown(text string) string {
	// Try to extract content from code blocks first (```json ... ```)
	codeBlockRegex := regexp.MustCompile("(?s)```(?:json|javascript|js)?\\s*\\n?(.*?)\\n?```")
	if matches := codeBlockRegex.FindStringSubmatch(text); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Try to extract content from inline code blocks (`...`)
	inlineCodeRegex := regexp.MustCompile("`([^`]+)`")
	if matches := inlineCodeRegex.FindStringSubmatch(text); len(matches) > 1 {
		potentialJSON := strings.TrimSpace(matches[1])
		if isLikelyJSON(potentialJSON) {
			return potentialJSON
		}
	}

	// No code blocks found, return the original text
	return text
}

// extractJSON finds and extracts JSON from text that might contain other content
func extractJSON(text string) (string, error) {
	// Find the first occurrence of { or [
	startIdx := strings.IndexAny(text, "{[")
	if startIdx == -1 {
		// No JSON-like structure found
		return text, nil
	}

	// Special case: if the text starts with { or [, assume it's already JSON
	if startIdx == 0 {
		return text, nil
	}

	// Extract everything from the first { or [
	return text[startIdx:], nil
}

// fixIncompleteJSON attempts to fix incomplete JSON by balancing delimiters
func fixIncompleteJSON(text string) string {
	if text == "" {
		return text
	}

	// Check if the text starts with { or [
	if text[0] != '{' && text[0] != '[' {
		return text
	}

	stack := []rune{}
	inString := false
	var stringDelim rune

	// First pass: traverse the string to detect unbalanced delimiters
	for _, ch := range text {
		if inString {
			if ch == '\\' {
				// Skip the next character if it's escaped
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
			} else {
				// Mismatched closing delimiter - indicating malformed JSON
				// We don't try to fix this case as it would be ambiguous
			}
		}
	}

	// If we have unbalanced open delimiters, add the missing closing ones
	if len(stack) > 0 {
		for i := len(stack) - 1; i >= 0; i-- {
			text += string(stack[i])
		}
	}

	return text
}

// isLikelyJSON does a quick check if text looks like it might be JSON
func isLikelyJSON(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}

	// Check for JSON object or array
	if (text[0] == '{' && text[len(text)-1] == '}') ||
		(text[0] == '[' && text[len(text)-1] == ']') {
		return true
	}

	// Check if it contains : which is common in JSON
	return strings.Contains(text, ":")
}
