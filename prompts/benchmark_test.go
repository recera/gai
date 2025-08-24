package prompts

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Benchmark embedded filesystem for tests
//
//go:embed testdata/*.tmpl
var benchFS embed.FS

// BenchmarkRegistryCreation benchmarks creating a new registry.
func BenchmarkRegistryCreation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		reg, err := NewRegistry(benchFS)
		if err != nil {
			b.Fatal(err)
		}
		_ = reg
	}
}

// BenchmarkRenderSimple benchmarks rendering a simple template.
func BenchmarkRenderSimple(b *testing.B) {
	reg, err := NewRegistry(benchFS)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	data := map[string]any{"Name": "Alice"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, id, err := reg.Render(ctx, "greet", "1.0.0", data)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
		_ = id
	}
}

// BenchmarkRenderComplex benchmarks rendering a complex template with helpers.
func BenchmarkRenderComplex(b *testing.B) {
	reg, err := NewRegistry(benchFS)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	data := map[string]any{
		"Audience": "executives",
		"Length":   "brief",
		"Topics":   []string{"sales", "revenue", "growth", "market", "competition"},
		"Style":    "professional",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, id, err := reg.Render(ctx, "summarize", "1.2.0", data)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
		_ = id
	}
}

// BenchmarkRenderWithHelpers benchmarks templates using various helper functions.
func BenchmarkRenderWithHelpers(b *testing.B) {
	reg, err := NewRegistry(benchFS)
	if err != nil {
		b.Fatal(err)
	}

	// Add a template that uses multiple helpers
	tmpl := &Template{
		Name:    "bench",
		Version: "1.0.0",
		Content: `{{upper .Title}}
{{indent 4 .Body}}
{{join ", " .Items}}
{{json .Data}}`,
		Fingerprint: "test",
		Source:      "test",
		LoadedAt:    time.Now(),
	}

	reg.mu.Lock()
	reg.templates["bench@1.0.0"] = tmpl
	reg.versionIndex["bench"] = []string{"1.0.0"}
	reg.mu.Unlock()

	ctx := context.Background()
	data := map[string]any{
		"Title": "Important Report",
		"Body":  "This is a multi-line\nbody text that needs\nproper indentation.",
		"Items": []string{"item1", "item2", "item3", "item4", "item5"},
		"Data": map[string]any{
			"key1": "value1",
			"key2": 42,
			"key3": true,
			"nested": map[string]string{
				"deep": "value",
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, _, err := reg.Render(ctx, "bench", "1.0.0", data)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}

// BenchmarkFingerprinting benchmarks SHA-256 fingerprint computation.
func BenchmarkFingerprinting(b *testing.B) {
	contents := [][]byte{
		[]byte("Small content"),
		[]byte("Medium content with more text to hash including some special characters !@#$%^&*()"),
		[]byte("Large content " + string(make([]byte, 1024))), // 1KB
		[]byte("XLarge content " + string(make([]byte, 10240))), // 10KB
	}

	for _, content := range contents {
		size := len(content)
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				fp := computeFingerprint(content)
				_ = fp
			}
		})
	}
}

// BenchmarkGet benchmarks retrieving templates from the registry.
func BenchmarkGet(b *testing.B) {
	reg, err := NewRegistry(benchFS)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tmpl, err := reg.Get("greet", "1.0.0")
		if err != nil {
			b.Fatal(err)
		}
		_ = tmpl
	}
}

// BenchmarkList benchmarks listing all templates.
func BenchmarkList(b *testing.B) {
	reg, err := NewRegistry(benchFS)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		templates := reg.List()
		_ = templates
	}
}

// BenchmarkValidate benchmarks template validation.
func BenchmarkValidate(b *testing.B) {
	reg, err := NewRegistry(benchFS)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := reg.Validate("greet", "1.0.0")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVersionSorting benchmarks semantic version sorting.
func BenchmarkVersionSorting(b *testing.B) {
	versions := []string{
		"2.10.5", "1.0.0", "1.2.0", "1.10.0", "1.2.1",
		"3.0.0", "2.0.0", "1.5.3", "1.5.10", "2.1.0",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Make a copy to avoid mutating the original
		v := make([]string, len(versions))
		copy(v, versions)
		sortVersions(v)
	}
}

// BenchmarkReload benchmarks reloading templates from override directory.
func BenchmarkReload(b *testing.B) {
	tmpDir := b.TempDir()

	// Create some override templates
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("template%d@1.0.0.tmpl", i)
		content := fmt.Sprintf("Template %d content", i)
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			b.Fatal(err)
		}
	}

	reg, err := NewRegistry(benchFS, WithOverrideDir(tmpDir))
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := reg.Reload(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConcurrentRender benchmarks concurrent template rendering.
func BenchmarkConcurrentRender(b *testing.B) {
	reg, err := NewRegistry(benchFS)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	data := map[string]any{"Name": "User"}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result, id, err := reg.Render(ctx, "greet", "1.0.0", data)
			if err != nil {
				b.Fatal(err)
			}
			_ = result
			_ = id
		}
	})
}

// BenchmarkOverrideVsEmbedded benchmarks performance difference between override and embedded templates.
func BenchmarkOverrideVsEmbedded(b *testing.B) {
	tmpDir := b.TempDir()

	// Create an override template
	overrideContent := "Override: {{.Name}}"
	overridePath := filepath.Join(tmpDir, "override@1.0.0.tmpl")
	if err := os.WriteFile(overridePath, []byte(overrideContent), 0644); err != nil {
		b.Fatal(err)
	}

	reg, err := NewRegistry(benchFS, WithOverrideDir(tmpDir))
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	data := map[string]any{"Name": "Test"}

	b.Run("embedded", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result, _, err := reg.Render(ctx, "greet", "1.0.0", data)
			if err != nil {
				b.Fatal(err)
			}
			_ = result
		}
	})

	b.Run("override", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result, _, err := reg.Render(ctx, "override", "1.0.0", data)
			if err != nil {
				b.Fatal(err)
			}
			_ = result
		}
	})
}

// BenchmarkTemplateCache benchmarks the effect of template caching.
func BenchmarkTemplateCache(b *testing.B) {
	reg, err := NewRegistry(benchFS)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	// Test with different cache hit patterns
	b.Run("same_template", func(b *testing.B) {
		data := map[string]any{"Name": "User"}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result, _, err := reg.Render(ctx, "greet", "1.0.0", data)
			if err != nil {
				b.Fatal(err)
			}
			_ = result
		}
	})

	b.Run("rotating_templates", func(b *testing.B) {
		templates := []string{"greet", "summarize", "analyze"}
		data := map[string]any{
			"Name":     "User",
			"Audience": "general",
			"Length":   "short",
			"Topics":   []string{"test"},
			"Context":  "benchmark",
			"Parameters": map[string]string{"mode": "test"},
			"Focus":    "performance",
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			tmplName := templates[i%len(templates)]
			result, _, err := reg.Render(ctx, tmplName, "1.0.0", data)
			if err != nil {
				b.Fatal(err)
			}
			_ = result
		}
	})
}

// BenchmarkExport benchmarks exporting templates to JSON.
func BenchmarkExport(b *testing.B) {
	reg, err := NewRegistry(benchFS)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		if err := reg.Export(&buf); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStats benchmarks gathering registry statistics.
func BenchmarkStats(b *testing.B) {
	reg, err := NewRegistry(benchFS)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		stats := reg.Stats()
		_ = stats
	}
}