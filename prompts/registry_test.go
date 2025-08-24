package prompts

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test embedded filesystem
//
//go:embed testdata/*.tmpl
var testFS embed.FS

// TestNewRegistry tests creating a new registry with embedded templates.
func TestNewRegistry(t *testing.T) {
	reg, err := NewRegistry(testFS)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Check that templates were loaded
	templates := reg.List()
	if len(templates) == 0 {
		t.Error("expected templates to be loaded from embedded FS")
	}

	// Verify specific templates exist
	expectedTemplates := []string{"summarize", "greet", "analyze"}
	for _, name := range expectedTemplates {
		if versions, exists := templates[name]; !exists {
			t.Errorf("expected template %q to exist", name)
		} else if len(versions) == 0 {
			t.Errorf("expected template %q to have versions", name)
		}
	}
}

// TestRender tests template rendering with data.
func TestRender(t *testing.T) {
	reg, err := NewRegistry(testFS)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	tests := []struct {
		name     string
		template string
		version  string
		data     map[string]any
		want     string
		wantErr  bool
	}{
		{
			name:     "simple template",
			template: "greet",
			version:  "1.0.0",
			data:     map[string]any{"Name": "Alice"},
			want:     "Hello, Alice!",
			wantErr:  false,
		},
		{
			name:     "template with helpers",
			template: "summarize",
			version:  "1.0.0",
			data: map[string]any{
				"Audience": "executives",
				"Length":   "brief",
				"Topics":   []string{"sales", "revenue", "growth"},
			},
			wantErr: false,
		},
		{
			name:     "missing template",
			template: "nonexistent",
			version:  "1.0.0",
			data:     map[string]any{},
			wantErr:  true,
		},
		{
			name:     "empty version (latest)",
			template: "greet",
			version:  "",
			data:     map[string]any{"Name": "Bob"},
			wantErr:  false,
		},
	}

	ctx := context.Background()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, id, err := reg.Render(ctx, tc.template, tc.version, tc.data)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if id == nil {
				t.Error("expected template ID, got nil")
			} else {
				if id.Name != tc.template {
					t.Errorf("ID name = %q, want %q", id.Name, tc.template)
				}
				if tc.version != "" && id.Version != tc.version {
					t.Errorf("ID version = %q, want %q", id.Version, tc.version)
				}
				if id.Fingerprint == "" {
					t.Error("expected fingerprint to be set")
				}
			}

			if tc.want != "" && result != tc.want {
				t.Errorf("result = %q, want %q", result, tc.want)
			}
		})
	}
}

// TestOverrideDirectory tests loading templates from an override directory.
func TestOverrideDirectory(t *testing.T) {
	// Create temp directory for overrides
	tmpDir := t.TempDir()

	// Create an override template
	overrideContent := "Override: {{.Message}}"
	overridePath := filepath.Join(tmpDir, "greet@2.0.0.tmpl")
	if err := os.WriteFile(overridePath, []byte(overrideContent), 0644); err != nil {
		t.Fatalf("failed to write override template: %v", err)
	}

	// Create registry with override directory
	reg, err := NewRegistry(testFS, WithOverrideDir(tmpDir))
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Render the overridden template
	ctx := context.Background()
	result, id, err := reg.Render(ctx, "greet", "2.0.0", map[string]any{"Message": "Test"})
	if err != nil {
		t.Fatalf("failed to render override template: %v", err)
	}

	if result != "Override: Test" {
		t.Errorf("result = %q, want %q", result, "Override: Test")
	}

	if id.Version != "2.0.0" {
		t.Errorf("version = %q, want %q", id.Version, "2.0.0")
	}

	// Check template source
	tmpl, err := reg.Get("greet", "2.0.0")
	if err != nil {
		t.Fatalf("failed to get template: %v", err)
	}

	if tmpl.Source != "override" {
		t.Errorf("source = %q, want %q", tmpl.Source, "override")
	}
}

// TestFingerprinting tests that fingerprints are computed correctly.
func TestFingerprinting(t *testing.T) {
	content1 := []byte("Hello, world!")
	content2 := []byte("Hello, world!") // Same content
	content3 := []byte("Goodbye, world!")

	fp1 := computeFingerprint(content1)
	fp2 := computeFingerprint(content2)
	fp3 := computeFingerprint(content3)

	if fp1 != fp2 {
		t.Error("identical content should have same fingerprint")
	}

	if fp1 == fp3 {
		t.Error("different content should have different fingerprints")
	}

	// Verify it's a valid hex string
	if len(fp1) != 64 { // SHA-256 produces 32 bytes = 64 hex chars
		t.Errorf("fingerprint length = %d, want 64", len(fp1))
	}
}

// TestVersionSorting tests semantic version sorting.
func TestVersionSorting(t *testing.T) {
	versions := []string{"2.0.0", "1.0.0", "1.2.0", "1.10.0", "1.2.1"}
	sortVersions(versions)

	expected := []string{"1.0.0", "1.2.0", "1.2.1", "1.10.0", "2.0.0"}
	for i, v := range versions {
		if v != expected[i] {
			t.Errorf("versions[%d] = %q, want %q", i, v, expected[i])
		}
	}
}

// TestHelperFunctions tests template helper functions.
func TestHelperFunctions(t *testing.T) {
	reg, err := NewRegistry(testFS)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	tests := []struct {
		name     string
		template string
		data     map[string]any
		contains []string
	}{
		{
			name:     "indent helper",
			template: "{{indent 4 .Text}}",
			data:     map[string]any{"Text": "line1\nline2"},
			contains: []string{"    line1", "    line2"},
		},
		{
			name:     "join helper",
			template: "{{join \", \" .Items}}",
			data:     map[string]any{"Items": []string{"a", "b", "c"}},
			contains: []string{"a, b, c"},
		},
		{
			name:     "json helper",
			template: "{{json .Data}}",
			data:     map[string]any{"Data": map[string]string{"key": "value"}},
			contains: []string{`"key":"value"`},
		},
		{
			name:     "upper/lower helpers",
			template: "{{upper .Text}} {{lower .Text}}",
			data:     map[string]any{"Text": "Hello"},
			contains: []string{"HELLO", "hello"},
		},
		{
			name:     "default helper",
			template: "{{default \"N/A\" .Missing}}",
			data:     map[string]any{},
			contains: []string{"N/A"},
		},
	}

	ctx := context.Background()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary in-memory template
			tmpl := &Template{
				Name:        "test",
				Version:     "1.0.0",
				Content:     tc.template,
				Fingerprint: computeFingerprint([]byte(tc.template)),
				Source:      "test",
				LoadedAt:    time.Now(),
			}

			reg.mu.Lock()
			reg.templates["test@1.0.0"] = tmpl
			reg.versionIndex["test"] = []string{"1.0.0"}
			reg.mu.Unlock()

			result, _, err := reg.Render(ctx, "test", "1.0.0", tc.data)
			if err != nil {
				t.Fatalf("failed to render: %v", err)
			}

			for _, want := range tc.contains {
				if !strings.Contains(result, want) {
					t.Errorf("result should contain %q, got %q", want, result)
				}
			}

			// Clean up
			reg.mu.Lock()
			delete(reg.templates, "test@1.0.0")
			delete(reg.versionIndex, "test")
			reg.mu.Unlock()
		})
	}
}

// TestReload tests reloading templates from override directory.
func TestReload(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial override
	v1Content := "Version 1"
	v1Path := filepath.Join(tmpDir, "dynamic@1.0.0.tmpl")
	if err := os.WriteFile(v1Path, []byte(v1Content), 0644); err != nil {
		t.Fatalf("failed to write v1: %v", err)
	}

	reg, err := NewRegistry(testFS, WithOverrideDir(tmpDir))
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Verify v1 is loaded
	tmpl, err := reg.Get("dynamic", "1.0.0")
	if err != nil {
		t.Fatalf("failed to get v1: %v", err)
	}
	if tmpl.Content != v1Content {
		t.Errorf("content = %q, want %q", tmpl.Content, v1Content)
	}

	// Update override
	v2Content := "Version 2"
	if err := os.WriteFile(v1Path, []byte(v2Content), 0644); err != nil {
		t.Fatalf("failed to update template: %v", err)
	}

	// Reload
	if err := reg.Reload(); err != nil {
		t.Fatalf("failed to reload: %v", err)
	}

	// Verify updated content
	tmpl, err = reg.Get("dynamic", "1.0.0")
	if err != nil {
		t.Fatalf("failed to get updated template: %v", err)
	}
	if tmpl.Content != v2Content {
		t.Errorf("content = %q, want %q", tmpl.Content, v2Content)
	}
}

// TestStrictVersioning tests strict version matching.
func TestStrictVersioning(t *testing.T) {
	reg, err := NewRegistry(testFS, WithStrictVersioning(true))
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	ctx := context.Background()

	// With strict versioning, empty version should fail
	_, _, err = reg.Render(ctx, "greet", "", map[string]any{"Name": "Test"})
	if err == nil {
		t.Error("expected error with empty version in strict mode")
	}

	// Non-existent version should fail
	_, _, err = reg.Render(ctx, "greet", "99.0.0", map[string]any{"Name": "Test"})
	if err == nil {
		t.Error("expected error with non-existent version in strict mode")
	}
}

// TestValidate tests template validation.
func TestValidate(t *testing.T) {
	reg, err := NewRegistry(testFS)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Valid template should pass
	if err := reg.Validate("greet", "1.0.0"); err != nil {
		t.Errorf("valid template failed validation: %v", err)
	}

	// Add invalid template for testing
	reg.mu.Lock()
	reg.templates["invalid@1.0.0"] = &Template{
		Name:        "invalid",
		Version:     "1.0.0",
		Content:     "{{.Name", // Missing closing braces
		Fingerprint: "test",
		Source:      "test",
		LoadedAt:    time.Now(),
	}
	reg.versionIndex["invalid"] = []string{"1.0.0"}
	reg.mu.Unlock()

	// Invalid template should fail
	if err := reg.Validate("invalid", "1.0.0"); err == nil {
		t.Error("invalid template passed validation")
	}
}

// TestExport tests exporting templates.
func TestExport(t *testing.T) {
	reg, err := NewRegistry(testFS)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	var buf bytes.Buffer
	if err := reg.Export(&buf); err != nil {
		t.Fatalf("failed to export: %v", err)
	}

	// Verify it's valid JSON
	var exported map[string]any
	if err := json.Unmarshal(buf.Bytes(), &exported); err != nil {
		t.Fatalf("export is not valid JSON: %v", err)
	}

	// Check structure
	if _, ok := exported["templates"]; !ok {
		t.Error("export missing 'templates' field")
	}
	if _, ok := exported["exported"]; !ok {
		t.Error("export missing 'exported' field")
	}
}

// TestStats tests registry statistics.
func TestStats(t *testing.T) {
	tmpDir := t.TempDir()

	// Add override template
	overridePath := filepath.Join(tmpDir, "override@1.0.0.tmpl")
	if err := os.WriteFile(overridePath, []byte("override"), 0644); err != nil {
		t.Fatalf("failed to write override: %v", err)
	}

	reg, err := NewRegistry(testFS, WithOverrideDir(tmpDir))
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	stats := reg.Stats()

	// Verify stats structure
	if total, ok := stats["total_templates"].(int); !ok || total == 0 {
		t.Error("expected positive total_templates")
	}
	if unique, ok := stats["unique_names"].(int); !ok || unique == 0 {
		t.Error("expected positive unique_names")
	}
	if embedded, ok := stats["embedded"].(int); !ok || embedded == 0 {
		t.Error("expected positive embedded count")
	}
	if overrides, ok := stats["overrides"].(int); !ok || overrides == 0 {
		t.Error("expected positive overrides count")
	}
	if dir, ok := stats["override_dir"].(string); !ok || dir != tmpDir {
		t.Errorf("override_dir = %q, want %q", dir, tmpDir)
	}
}

// TestConcurrentAccess tests thread-safe operations.
func TestConcurrentAccess(t *testing.T) {
	reg, err := NewRegistry(testFS)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	ctx := context.Background()
	done := make(chan bool)

	// Concurrent renders
	for i := 0; i < 10; i++ {
		go func(n int) {
			defer func() { done <- true }()

			data := map[string]any{"Name": "User"}
			_, _, err := reg.Render(ctx, "greet", "1.0.0", data)
			if err != nil {
				t.Errorf("concurrent render %d failed: %v", n, err)
			}
		}(i)
	}

	// Concurrent gets
	for i := 0; i < 10; i++ {
		go func(n int) {
			defer func() { done <- true }()

			_, err := reg.Get("greet", "1.0.0")
			if err != nil {
				t.Errorf("concurrent get %d failed: %v", n, err)
			}
		}(i)
	}

	// Concurrent lists
	for i := 0; i < 10; i++ {
		go func(n int) {
			defer func() { done <- true }()

			templates := reg.List()
			if len(templates) == 0 {
				t.Errorf("concurrent list %d returned empty", n)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 30; i++ {
		<-done
	}
}

// TestCustomHelperFunc tests adding custom helper functions.
func TestCustomHelperFunc(t *testing.T) {
	customHelper := func(s string) string {
		return "custom: " + s
	}

	reg, err := NewRegistry(testFS, WithHelperFunc("custom", customHelper))
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Add test template using custom helper
	tmpl := &Template{
		Name:        "custom",
		Version:     "1.0.0",
		Content:     "{{custom .Value}}",
		Fingerprint: "test",
		Source:      "test",
		LoadedAt:    time.Now(),
	}

	reg.mu.Lock()
	reg.templates["custom@1.0.0"] = tmpl
	reg.versionIndex["custom"] = []string{"1.0.0"}
	reg.mu.Unlock()

	ctx := context.Background()
	result, _, err := reg.Render(ctx, "custom", "1.0.0", map[string]any{"Value": "test"})
	if err != nil {
		t.Fatalf("failed to render with custom helper: %v", err)
	}

	if result != "custom: test" {
		t.Errorf("result = %q, want %q", result, "custom: test")
	}
}