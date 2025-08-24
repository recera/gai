// Package prompts provides a versioned template management system with hot-reload
// support, fingerprinting, and runtime overrides.
package prompts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/recera/gai/obs"
)

// TemplateID contains metadata about a rendered template.
type TemplateID struct {
	Name        string `json:"name"`        // Template name without version
	Version     string `json:"version"`     // Semantic version (MAJOR.MINOR.PATCH)
	Fingerprint string `json:"fingerprint"` // SHA-256 hash of template content
}

// Template represents a versioned prompt template.
type Template struct {
	Name        string
	Version     string
	Content     string
	Fingerprint string
	Source      string // "embedded" or "override"
	LoadedAt    time.Time
}

// Registry manages versioned prompt templates with support for embedded
// and override templates.
type Registry struct {
	mu sync.RWMutex

	// embedFS is the embedded filesystem containing default templates
	embedFS fs.FS

	// overrideDir is the optional directory for runtime overrides
	overrideDir string

	// templates cached by "name@version" key
	templates map[string]*Template

	// versionIndex maps template names to available versions
	versionIndex map[string][]string

	// funcMap contains template helper functions
	funcMap template.FuncMap

	// watchOverrides enables filesystem watching (future enhancement)
	watchOverrides bool

	// strictVersioning requires exact version matches
	strictVersioning bool
}

// Option configures a Registry.
type Option func(*Registry)

// WithOverrideDir sets the directory to check for template overrides.
func WithOverrideDir(dir string) Option {
	return func(r *Registry) {
		r.overrideDir = dir
	}
}

// WithStrictVersioning requires exact version matches (no fallback to latest).
func WithStrictVersioning(strict bool) Option {
	return func(r *Registry) {
		r.strictVersioning = strict
	}
}

// WithHelperFunc adds a custom template helper function.
func WithHelperFunc(name string, fn any) Option {
	return func(r *Registry) {
		r.funcMap[name] = fn
	}
}

// versionPattern matches template filenames like "name@1.2.3.tmpl"
var versionPattern = regexp.MustCompile(`^(.+)@(\d+\.\d+\.\d+)\.tmpl$`)

// NewRegistry creates a new template registry.
func NewRegistry(embedFS embed.FS, opts ...Option) (*Registry, error) {
	r := &Registry{
		embedFS:      embedFS,
		templates:    make(map[string]*Template),
		versionIndex: make(map[string][]string),
		funcMap:      defaultFuncMap(),
	}

	for _, opt := range opts {
		opt(r)
	}

	// Load embedded templates
	if err := r.loadEmbedded(); err != nil {
		return nil, fmt.Errorf("failed to load embedded templates: %w", err)
	}

	// Load override templates if directory is specified
	if r.overrideDir != "" {
		if err := r.loadOverrides(); err != nil {
			// Non-fatal: overrides are optional
			// Log warning in production
		}
	}

	return r, nil
}

// loadEmbedded loads templates from the embedded filesystem.
func (r *Registry) loadEmbedded() error {
	return fs.WalkDir(r.embedFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		matches := versionPattern.FindStringSubmatch(filepath.Base(path))
		if matches == nil {
			// Skip non-versioned templates
			return nil
		}

		name := matches[1]
		version := matches[2]

		content, err := fs.ReadFile(r.embedFS, path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		tmpl := &Template{
			Name:        name,
			Version:     version,
			Content:     string(content),
			Fingerprint: computeFingerprint(content),
			Source:      "embedded",
			LoadedAt:    time.Now(),
		}

		key := fmt.Sprintf("%s@%s", name, version)
		r.templates[key] = tmpl

		// Update version index
		if r.versionIndex[name] == nil {
			r.versionIndex[name] = []string{}
		}
		r.versionIndex[name] = append(r.versionIndex[name], version)

		return nil
	})
}

// loadOverrides loads templates from the override directory.
func (r *Registry) loadOverrides() error {
	if r.overrideDir == "" {
		return nil
	}

	info, err := os.Stat(r.overrideDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Override directory doesn't exist, that's OK
		}
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("override path is not a directory: %s", r.overrideDir)
	}

	entries, err := os.ReadDir(r.overrideDir)
	if err != nil {
		return fmt.Errorf("failed to read override directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}

		matches := versionPattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		name := matches[1]
		version := matches[2]

		path := filepath.Join(r.overrideDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read override %s: %w", path, err)
		}

		tmpl := &Template{
			Name:        name,
			Version:     version,
			Content:     string(content),
			Fingerprint: computeFingerprint(content),
			Source:      "override",
			LoadedAt:    time.Now(),
		}

		key := fmt.Sprintf("%s@%s", name, version)
		r.templates[key] = tmpl

		// Update version index if this is a new version
		found := false
		for _, v := range r.versionIndex[name] {
			if v == version {
				found = true
				break
			}
		}
		if !found {
			r.versionIndex[name] = append(r.versionIndex[name], version)
		}
	}

	// Sort versions for each template
	for name := range r.versionIndex {
		sortVersions(r.versionIndex[name])
	}

	return nil
}

// Render renders a template with the given data.
func (r *Registry) Render(ctx context.Context, name, version string, data map[string]any) (string, *TemplateID, error) {
	startTime := time.Now()
	
	// Extract data keys for observability
	dataKeys := make([]string, 0, len(data))
	for k := range data {
		dataKeys = append(dataKeys, k)
	}
	
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Handle empty version - use latest
	if version == "" && !r.strictVersioning {
		versions, exists := r.versionIndex[name]
		if !exists || len(versions) == 0 {
			return "", nil, fmt.Errorf("template %q not found", name)
		}
		version = versions[len(versions)-1] // Latest version (already sorted)
	}

	key := fmt.Sprintf("%s@%s", name, version)
	tmpl, exists := r.templates[key]
	cacheHit := exists
	
	if !exists {
		// If strict versioning is off, try to find the latest compatible version
		if !r.strictVersioning {
			compatVersion := r.findCompatibleVersion(name, version)
			if compatVersion != "" {
				key = fmt.Sprintf("%s@%s", name, compatVersion)
				tmpl = r.templates[key]
				cacheHit = tmpl != nil
			}
		}

		if tmpl == nil {
			return "", nil, fmt.Errorf("template %q version %q not found", name, version)
		}
	}

	// Start prompt span for observability
	_, span := obs.StartPromptSpan(ctx, obs.PromptSpanOptions{
		Name:        name,
		Version:     tmpl.Version,
		Fingerprint: tmpl.Fingerprint,
		DataKeys:    dataKeys,
		Override:    tmpl.Source == "override",
		CacheHit:    cacheHit,
	})
	defer span.End()

	// Parse and execute template
	t, err := template.New(name).Funcs(r.funcMap).Parse(tmpl.Content)
	if err != nil {
		obs.RecordError(span, err, "Template parsing failed")
		return "", nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		obs.RecordError(span, err, "Template execution failed")
		return "", nil, fmt.Errorf("failed to execute template: %w", err)
	}

	id := &TemplateID{
		Name:        tmpl.Name,
		Version:     tmpl.Version,
		Fingerprint: tmpl.Fingerprint,
	}

	// Record metrics
	obs.RecordPromptRender(ctx, name, tmpl.Version, cacheHit, time.Since(startTime))
	obs.RecordCacheHit(ctx, "prompt", cacheHit)

	return buf.String(), id, nil
}

// Get retrieves a template without rendering it.
func (r *Registry) Get(name, version string) (*Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if version == "" && !r.strictVersioning {
		versions, exists := r.versionIndex[name]
		if !exists || len(versions) == 0 {
			return nil, fmt.Errorf("template %q not found", name)
		}
		version = versions[len(versions)-1]
	}

	key := fmt.Sprintf("%s@%s", name, version)
	tmpl, exists := r.templates[key]
	if !exists {
		return nil, fmt.Errorf("template %q version %q not found", name, version)
	}

	// Return a copy to prevent mutation
	copy := *tmpl
	return &copy, nil
}

// List returns all available templates and their versions.
func (r *Registry) List() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]string)
	for name, versions := range r.versionIndex {
		result[name] = append([]string(nil), versions...)
	}
	return result
}

// Reload reloads templates from the override directory.
func (r *Registry) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear override templates
	for key, tmpl := range r.templates {
		if tmpl.Source == "override" {
			delete(r.templates, key)
		}
	}

	// Rebuild version index
	r.versionIndex = make(map[string][]string)
	for _, tmpl := range r.templates {
		if tmpl.Source == "embedded" {
			if r.versionIndex[tmpl.Name] == nil {
				r.versionIndex[tmpl.Name] = []string{}
			}
			r.versionIndex[tmpl.Name] = append(r.versionIndex[tmpl.Name], tmpl.Version)
		}
	}

	// Reload overrides
	if r.overrideDir != "" {
		if err := r.loadOverrides(); err != nil {
			return err
		}
	}

	return nil
}

// findCompatibleVersion finds the latest compatible version based on semantic versioning.
func (r *Registry) findCompatibleVersion(name, requestedVersion string) string {
	versions, exists := r.versionIndex[name]
	if !exists || len(versions) == 0 {
		return ""
	}

	// For now, just return the latest version
	// In the future, implement proper semver compatibility checking
	return versions[len(versions)-1]
}

// computeFingerprint computes the SHA-256 hash of template content.
func computeFingerprint(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// sortVersions sorts semantic versions in ascending order.
func sortVersions(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) < 0
	})
}

// compareVersions compares two semantic versions.
// Returns -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	for i := 0; i < 3; i++ {
		var p1, p2 int
		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &p1)
		}
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &p2)
		}

		if p1 < p2 {
			return -1
		}
		if p1 > p2 {
			return 1
		}
	}

	return 0
}

// defaultFuncMap returns the default template helper functions.
func defaultFuncMap() template.FuncMap {
	return template.FuncMap{
		// String manipulation
		"indent": func(spaces int, text string) string {
			padding := strings.Repeat(" ", spaces)
			lines := strings.Split(text, "\n")
			for i := range lines {
				if lines[i] != "" {
					lines[i] = padding + lines[i]
				}
			}
			return strings.Join(lines, "\n")
		},

		"join": func(sep string, items []string) string {
			return strings.Join(items, sep)
		},

		"trim": strings.TrimSpace,
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,

		// JSON helpers
		"json": func(v any) (string, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},

		"jsonIndent": func(v any) (string, error) {
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return "", err
			}
			return string(b), nil
		},

		// List operations
		"first": func(items []any) any {
			if len(items) > 0 {
				return items[0]
			}
			return nil
		},

		"last": func(items []any) any {
			if len(items) > 0 {
				return items[len(items)-1]
			}
			return nil
		},

		// Conditional helpers
		"default": func(defaultVal, val any) any {
			if val == nil || val == "" {
				return defaultVal
			}
			return val
		},

		// Date/time
		"now": func() string {
			return time.Now().Format(time.RFC3339)
		},

		"date": func(format string) string {
			return time.Now().Format(format)
		},
	}
}

// Validate checks if a template can be parsed without errors.
func (r *Registry) Validate(name, version string) error {
	tmpl, err := r.Get(name, version)
	if err != nil {
		return err
	}

	_, err = template.New(name).Funcs(r.funcMap).Parse(tmpl.Content)
	return err
}

// Export writes all templates to a writer in a format suitable for backup.
func (r *Registry) Export(w io.Writer) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	type export struct {
		Templates map[string]*Template `json:"templates"`
		Exported  time.Time            `json:"exported"`
	}

	data := export{
		Templates: r.templates,
		Exported:  time.Now(),
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// Stats returns statistics about the registry.
func (r *Registry) Stats() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	embeddedCount := 0
	overrideCount := 0
	for _, tmpl := range r.templates {
		if tmpl.Source == "embedded" {
			embeddedCount++
		} else {
			overrideCount++
		}
	}

	return map[string]any{
		"total_templates": len(r.templates),
		"unique_names":    len(r.versionIndex),
		"embedded":        embeddedCount,
		"overrides":       overrideCount,
		"override_dir":    r.overrideDir,
	}
}