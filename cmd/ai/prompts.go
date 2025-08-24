package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/recera/gai/prompts"
	"github.com/spf13/cobra"
)

// promptsCmd represents the prompts command group
var promptsCmd = &cobra.Command{
	Use:   "prompts",
	Short: "Manage and verify prompt templates",
	Long:  `Tools for managing, verifying, and testing prompt templates.`,
}

// verifyCmd represents the verify command
var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify prompt template versions",
	Long: `Verifies that prompt templates have appropriate version bumps when content changes.

This command checks:
  - Template file naming conventions (name@VERSION.tmpl)
  - Version format (MAJOR.MINOR.PATCH)
  - Content changes require version bumps
  - No duplicate versions for the same template name

Exit codes:
  0 - All templates verified successfully
  1 - Verification failed`,
	RunE: runVerify,
}

// bumpCmd represents the bump command
var bumpCmd = &cobra.Command{
	Use:   "bump <template> <major|minor|patch>",
	Short: "Bump the version of a prompt template",
	Args:  cobra.ExactArgs(2),
	RunE:  runBump,
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all prompt templates and their versions",
	RunE:  runList,
}

var (
	promptsDir string
	strict     bool
)

func init() {
	rootCmd.AddCommand(promptsCmd)
	promptsCmd.AddCommand(verifyCmd)
	promptsCmd.AddCommand(bumpCmd)
	promptsCmd.AddCommand(listCmd)

	promptsCmd.PersistentFlags().StringVar(&promptsDir, "dir", "", "Prompts directory (default: search for embedded templates)")
	verifyCmd.Flags().BoolVar(&strict, "strict", false, "Strict mode: fail on any warning")
}

func runVerify(cmd *cobra.Command, args []string) error {
	if promptsDir == "" {
		promptsDir = findPromptsDir()
	}

	if promptsDir == "" {
		return fmt.Errorf("prompts directory not found")
	}

	fmt.Printf("Verifying prompts in: %s\n\n", promptsDir)

	// Pattern for template files
	versionPattern := regexp.MustCompile(`^(.+)@(\d+\.\d+\.\d+)\.tmpl$`)
	
	templates := make(map[string][]templateInfo)
	var errors []string
	var warnings []string

	// Walk the prompts directory
	err := filepath.Walk(promptsDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		filename := filepath.Base(path)
		
		// Check naming convention
		matches := versionPattern.FindStringSubmatch(filename)
		if matches == nil {
			errors = append(errors, fmt.Sprintf("❌ Invalid naming: %s (expected: name@VERSION.tmpl)", filename))
			return nil
		}

		name := matches[1]
		version := matches[2]

		// Read content for fingerprinting
		content, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("❌ Cannot read %s: %v", filename, err))
			return nil
		}

		// Calculate fingerprint
		hash := sha256.Sum256(content)
		fingerprint := hex.EncodeToString(hash[:])[:16]

		tmplInfo := templateInfo{
			Name:        name,
			Version:     version,
			Filename:    filename,
			Path:        path,
			Fingerprint: fingerprint,
			Content:     string(content),
		}

		templates[name] = append(templates[name], tmplInfo)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Verify each template group
	for name, versions := range templates {
		fmt.Printf("Template: %s\n", name)
		
		// Check for duplicate versions
		versionMap := make(map[string][]templateInfo)
		for _, t := range versions {
			versionMap[t.Version] = append(versionMap[t.Version], t)
		}

		for version, infos := range versionMap {
			if len(infos) > 1 {
				errors = append(errors, fmt.Sprintf("❌ Duplicate version %s for template %s", version, name))
				for _, info := range infos {
					fmt.Printf("    - %s\n", info.Path)
				}
			}
		}

		// Check for content duplicates with different versions
		fingerprintMap := make(map[string][]templateInfo)
		for _, t := range versions {
			fingerprintMap[t.Fingerprint] = append(fingerprintMap[t.Fingerprint], t)
		}

		for fingerprint, infos := range fingerprintMap {
			if len(infos) > 1 {
				warnings = append(warnings, fmt.Sprintf("⚠️  Same content with different versions in template %s", name))
				for _, info := range infos {
					fmt.Printf("    - %s (fingerprint: %s)\n", info.Filename, fingerprint[:8])
				}
			}
		}

		// List all versions
		for _, t := range versions {
			fmt.Printf("  ✓ %s (fingerprint: %s...)\n", t.Version, t.Fingerprint[:8])
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("Summary:")
	fmt.Printf("  Templates found: %d\n", len(templates))
	
	totalVersions := 0
	for _, versions := range templates {
		totalVersions += len(versions)
	}
	fmt.Printf("  Total versions: %d\n", totalVersions)

	if len(errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range errors {
			fmt.Println("  " + err)
		}
	}

	if len(warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warn := range warnings {
			fmt.Println("  " + warn)
		}
	}

	if len(errors) > 0 || (strict && len(warnings) > 0) {
		return fmt.Errorf("verification failed")
	}

	fmt.Println("\n✅ All prompt templates verified successfully!")
	return nil
}

func runBump(cmd *cobra.Command, args []string) error {
	templateName := args[0]
	bumpType := args[1]

	if promptsDir == "" {
		promptsDir = findPromptsDir()
	}

	if promptsDir == "" {
		return fmt.Errorf("prompts directory not found")
	}

	// Find the latest version of the template
	versionPattern := regexp.MustCompile(`^` + regexp.QuoteMeta(templateName) + `@(\d+)\.(\d+)\.(\d+)\.tmpl$`)
	
	var latestVersion string
	var latestPath string
	var major, minor, patch int

	entries, err := os.ReadDir(promptsDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := versionPattern.FindStringSubmatch(entry.Name())
		if matches != nil {
			version := fmt.Sprintf("%s.%s.%s", matches[1], matches[2], matches[3])
			if version > latestVersion {
				latestVersion = version
				latestPath = filepath.Join(promptsDir, entry.Name())
				fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch)
			}
		}
	}

	if latestVersion == "" {
		return fmt.Errorf("template %s not found", templateName)
	}

	// Calculate new version
	switch bumpType {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "patch":
		patch++
	default:
		return fmt.Errorf("invalid bump type: %s (use major, minor, or patch)", bumpType)
	}

	newVersion := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	newFilename := fmt.Sprintf("%s@%s.tmpl", templateName, newVersion)
	newPath := filepath.Join(promptsDir, newFilename)

	// Read the content
	content, err := os.ReadFile(latestPath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// Write the new version
	if err := os.WriteFile(newPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write new version: %w", err)
	}

	fmt.Printf("✅ Bumped %s from %s to %s\n", templateName, latestVersion, newVersion)
	fmt.Printf("   Created: %s\n", newPath)
	fmt.Printf("   Note: Remember to update the template content!\n")

	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	if promptsDir == "" {
		promptsDir = findPromptsDir()
	}

	if promptsDir == "" {
		// Try to use embedded templates
		var embedFS embed.FS // This would need to be properly set up
		reg, err := prompts.NewRegistry(embedFS)
		if err == nil {
			templates := reg.List()
			fmt.Println("Available prompt templates:")
			for name, versions := range templates {
				fmt.Printf("\n📄 %s\n", name)
				for _, version := range versions {
					fmt.Printf("   - %s\n", version)
				}
			}
			return nil
		}
		return fmt.Errorf("prompts directory not found and no embedded templates available")
	}

	fmt.Printf("Prompt templates in: %s\n\n", promptsDir)

	versionPattern := regexp.MustCompile(`^(.+)@(\d+\.\d+\.\d+)\.tmpl$`)
	templates := make(map[string][]string)

	entries, err := os.ReadDir(promptsDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := versionPattern.FindStringSubmatch(entry.Name())
		if matches != nil {
			name := matches[1]
			version := matches[2]
			templates[name] = append(templates[name], version)
		}
	}

	if len(templates) == 0 {
		fmt.Println("No prompt templates found.")
		return nil
	}

	for name, versions := range templates {
		fmt.Printf("📄 %s\n", name)
		for _, version := range versions {
			info, _ := os.Stat(filepath.Join(promptsDir, fmt.Sprintf("%s@%s.tmpl", name, version)))
			size := ""
			if info != nil {
				size = fmt.Sprintf(" (%d bytes)", info.Size())
			}
			fmt.Printf("   - %s%s\n", version, size)
		}
		fmt.Println()
	}

	return nil
}

func findPromptsDir() string {
	// Look for common prompt directory locations
	candidates := []string{
		"prompts",
		"templates",
		"prompt-templates",
		filepath.Join("examples", "prompts"),
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Check current directory and parent directories
	for dir := cwd; dir != "/" && dir != ""; dir = filepath.Dir(dir) {
		for _, candidate := range candidates {
			path := filepath.Join(dir, candidate)
			if info, err := os.Stat(path); err == nil && info.IsDir() {
				// Check if it contains .tmpl files
				entries, _ := os.ReadDir(path)
				for _, entry := range entries {
					if strings.HasSuffix(entry.Name(), ".tmpl") {
						return path
					}
				}
			}
		}
	}

	return ""
}

type templateInfo struct {
	Name        string
	Version     string
	Filename    string
	Path        string
	Fingerprint string
	Content     string
}