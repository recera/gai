package gai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
)

// PromptTemplate wraps Go's text/template for LLM prompt generation
type PromptTemplate struct {
	t *template.Template
}

// NewPromptTemplate creates a new prompt template from the given text
func NewPromptTemplate(text string) (*PromptTemplate, error) {
	tmpl, err := template.New("prompt").Parse(text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	return &PromptTemplate{t: tmpl}, nil
}

// NewPromptTemplateWithFuncs creates a new prompt template with custom functions
func NewPromptTemplateWithFuncs(text string, funcs template.FuncMap) (*PromptTemplate, error) {
	tmpl, err := template.New("prompt").Funcs(funcs).Parse(text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	return &PromptTemplate{t: tmpl}, nil
}

// Render executes the template with the given data
func (pt *PromptTemplate) Render(data any) (string, error) {
	var buf bytes.Buffer
	if err := pt.t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}
	return buf.String(), nil
}

// MustRender renders the template and panics on error (useful for tests)
func (pt *PromptTemplate) MustRender(data any) string {
	result, err := pt.Render(data)
	if err != nil {
		panic(err)
	}
	return result
}

// Helper functions to use templates with LLMCallParts

// RenderSystemTemplate renders a template and sets it as the system message
func RenderSystemTemplate(p *LLMCallParts, tpl *PromptTemplate, data any) error {
	text, err := tpl.Render(data)
	if err != nil {
		return fmt.Errorf("failed to render system template: %w", err)
	}
	p.WithSystem(text)
	return nil
}

// RenderUserTemplate renders a template and adds it as a user message
func RenderUserTemplate(p *LLMCallParts, tpl *PromptTemplate, data any) error {
	text, err := tpl.Render(data)
	if err != nil {
		return fmt.Errorf("failed to render user template: %w", err)
	}
	p.WithUserMessage(text)
	return nil
}

// RenderAssistantTemplate renders a template and adds it as an assistant message
func RenderAssistantTemplate(p *LLMCallParts, tpl *PromptTemplate, data any) error {
	text, err := tpl.Render(data)
	if err != nil {
		return fmt.Errorf("failed to render assistant template: %w", err)
	}
	p.WithAssistantMessage(text)
	return nil
}

// Common template functions that can be useful
var CommonTemplateFuncs = template.FuncMap{
	"json": func(v interface{}) string {
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("\"%v\"", v)
		}
		return string(b)
	},
	"quote": func(s string) string {
		return fmt.Sprintf("%q", s)
	},
	"list": func(items ...interface{}) []interface{} {
		return items
	},
	"dict": func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("dict requires even number of arguments")
		}
		dict := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict keys must be strings")
			}
			dict[key] = values[i+1]
		}
		return dict, nil
	},
}

// MustParseTemplate parses a template and panics on error (useful for package-level templates)
func MustParseTemplate(text string) *PromptTemplate {
	tpl, err := NewPromptTemplate(text)
	if err != nil {
		panic(err)
	}
	return tpl
}

// MustParseTemplateWithFuncs parses a template with functions and panics on error
func MustParseTemplateWithFuncs(text string, funcs template.FuncMap) *PromptTemplate {
	tpl, err := NewPromptTemplateWithFuncs(text, funcs)
	if err != nil {
		panic(err)
	}
	return tpl
}
