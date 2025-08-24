// Package main demonstrates usage of the prompts package with versioned templates,
// overrides, and template helpers.
package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/recera/gai/prompts"
)

// Embed your prompt templates
//
//go:embed prompts/*.tmpl
var promptFS embed.FS

func main() {
	// Example 1: Basic usage with embedded templates
	basicExample()

	// Example 2: Using overrides for development
	overrideExample()

	// Example 3: Complex template with helpers
	complexExample()

	// Example 4: Template versioning
	versioningExample()

	// Example 5: Observability integration
	observabilityExample()
}

func basicExample() {
	fmt.Println("=== Basic Example ===")

	// Create registry with embedded templates
	reg, err := prompts.NewRegistry(promptFS)
	if err != nil {
		log.Fatal(err)
	}

	// Render a simple chat assistant prompt
	ctx := context.Background()
	data := map[string]any{
		"Personality": []string{"friendly", "helpful", "concise"},
		"Expertise":   []string{"programming", "mathematics", "science"},
		"Style":       "professional",
	}

	text, id, err := reg.Render(ctx, "chat_assistant", "1.0.0", data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Template: %s@%s\n", id.Name, id.Version)
	fmt.Printf("Fingerprint: %s\n", id.Fingerprint[:16]+"...")
	fmt.Printf("Rendered:\n%s\n\n", text)
}

func overrideExample() {
	fmt.Println("=== Override Example ===")

	// Use environment variable for override directory
	overrideDir := os.Getenv("PROMPTS_DIR")
	if overrideDir == "" {
		overrideDir = "./prompt_overrides"
	}

	// Create registry with override support
	reg, err := prompts.NewRegistry(
		promptFS,
		prompts.WithOverrideDir(overrideDir),
	)
	if err != nil {
		log.Fatal(err)
	}

	// List available templates
	templates := reg.List()
	fmt.Println("Available templates:")
	for name, versions := range templates {
		fmt.Printf("  %s: %v\n", name, versions)
	}
	fmt.Println()
}

func complexExample() {
	fmt.Println("=== Complex Template Example ===")

	reg, err := prompts.NewRegistry(promptFS)
	if err != nil {
		log.Fatal(err)
	}

	// Code review template with structured data
	ctx := context.Background()
	data := map[string]any{
		"Language":    "Go",
		"Repository":  "github.com/company/project",
		"ReviewType":  "Pull Request",
		"FocusAreas":  []string{"security", "performance", "testing"},
		"Standards": map[string]any{
			"line_length":    120,
			"test_coverage":  80,
			"cyclomatic":     10,
		},
		"IgnorePatterns": []string{"vendor/*", "*.pb.go"},
		"Tone":          "constructive",
	}

	text, id, err := reg.Render(ctx, "code_reviewer", "1.0.0", data)
	if err != nil {
		// Try fallback to latest version if exact version not found
		text, id, err = reg.Render(ctx, "code_reviewer", "", data)
		if err != nil {
			log.Printf("Warning: code_reviewer template not found: %v", err)
			return
		}
	}

	fmt.Printf("Using template: %s@%s\n", id.Name, id.Version)
	fmt.Printf("Output:\n%s\n\n", text)
}

func versioningExample() {
	fmt.Println("=== Versioning Example ===")

	// Create registry with strict versioning
	reg, err := prompts.NewRegistry(
		promptFS,
		prompts.WithStrictVersioning(false), // Allow fallback to latest
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	templates := []struct {
		name    string
		version string
	}{
		{"chat_assistant", "1.0.0"},  // Exact version
		{"chat_assistant", ""},        // Latest version
		{"chat_assistant", "2.0.0"},   // Non-existent (will use latest)
	}

	for _, tmpl := range templates {
		data := map[string]any{"Style": "test"}
		
		_, id, err := reg.Render(ctx, tmpl.name, tmpl.version, data)
		if err != nil {
			fmt.Printf("  %s@%s: ERROR - %v\n", tmpl.name, tmpl.version, err)
			continue
		}
		
		requestedVer := tmpl.version
		if requestedVer == "" {
			requestedVer = "latest"
		}
		
		fmt.Printf("  Requested %s@%s -> Got %s@%s\n", 
			tmpl.name, requestedVer, id.Name, id.Version)
	}
	fmt.Println()
}

func observabilityExample() {
	fmt.Println("=== Observability Example ===")

	reg, err := prompts.NewRegistry(promptFS)
	if err != nil {
		log.Fatal(err)
	}

	// Get registry statistics
	stats := reg.Stats()
	fmt.Println("Registry Statistics:")
	fmt.Printf("  Total Templates: %v\n", stats["total_templates"])
	fmt.Printf("  Unique Names: %v\n", stats["unique_names"])
	fmt.Printf("  Embedded: %v\n", stats["embedded"])
	fmt.Printf("  Overrides: %v\n", stats["overrides"])

	// Render and track template usage
	ctx := context.Background()
	data := map[string]any{
		"DataType":   "sales",
		"Dataset":    "Q4-2024",
		"TimePeriod": "Last Quarter",
		"Metrics": []map[string]string{
			{"Name": "Revenue", "Description": "Total sales revenue"},
			{"Name": "Units", "Description": "Units sold"},
		},
	}

	text, id, err := reg.Render(ctx, "data_analyst", "1.0.0", data)
	if err != nil {
		// Template might not exist in embedded FS
		fmt.Printf("  Template not found: %v\n", err)
		return
	}

	// In production, you would send these to your telemetry system
	fmt.Printf("\nTelemetry Data:\n")
	fmt.Printf("  prompt.name: %s\n", id.Name)
	fmt.Printf("  prompt.version: %s\n", id.Version)
	fmt.Printf("  prompt.fingerprint: %s\n", id.Fingerprint)
	fmt.Printf("  prompt.length: %d\n", len(text))
	fmt.Println()
}

// Example of custom helper function
func customHelperExample() {
	fmt.Println("=== Custom Helper Example ===")

	// Add a custom markdown helper
	markdownList := func(items []string) string {
		result := ""
		for _, item := range items {
			result += fmt.Sprintf("* %s\n", item)
		}
		return result
	}

	_, err := prompts.NewRegistry(
		promptFS,
		prompts.WithHelperFunc("markdown_list", markdownList),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Now templates can use {{markdown_list .Items}}
	fmt.Println("Registry created with custom helper function")
}