package gai

import (
	"fmt"
	"os"
)

// BuildActionPrompt reads a prompt template from a file and appends response format instructions.
// This is a utility function for combining file-based prompts with structured output requirements.
// Note: This function performs file I/O and should be used outside of the core client logic.
func BuildActionPrompt(filePath string, responseStruct any) (string, error) {
	// Read the prompt file
	prompt, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file: %w", err)
	}
	instructions, err := ResponseInstructions(responseStruct)
	if err != nil {
		return "", fmt.Errorf("failed to generate instructions: %w", err)
	}
	return string(prompt) + instructions, nil
}

// StringFromPath reads a file and returns its content as a string.
// This is a convenience function for loading text files.
func StringFromPath(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(data), nil
}