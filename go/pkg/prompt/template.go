package prompt

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"text/template"
)

// Template wraps a text/template loaded from disk with optional function map.
type Template struct {
	path  string
	funcs template.FuncMap

	mu   sync.RWMutex
	tmpl *template.Template
	hash string
}

// NewTemplate parses the template at path using the provided template functions.
func NewTemplate(path string, funcs template.FuncMap) (*Template, error) {
	if path == "" {
		return nil, fmt.Errorf("prompt template path is empty")
	}
	t := &Template{
		path:  path,
		funcs: funcs,
	}
	if err := t.reload(); err != nil {
		return nil, err
	}
	return t, nil
}

// Render executes the template with the provided data and returns the rendered string.
func (t *Template) Render(data any) (string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tmpl == nil {
		return "", fmt.Errorf("prompt template %q not parsed", t.path)
	}

	var buf bytes.Buffer
	if err := t.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute prompt template %q: %w", t.path, err)
	}
	return buf.String(), nil
}

// Reload reparses the underlying template from disk. This can be used when files change.
func (t *Template) Reload() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.reload()
}

func (t *Template) reload() error {
	data, err := os.ReadFile(t.path)
	if err != nil {
		return fmt.Errorf("read prompt template %q: %w", t.path, err)
	}
	t.hash = computeDigest(data)

	name := filepath.Base(t.path)
	tmpl := template.New(name).Option("missingkey=error")
	if len(t.funcs) > 0 {
		tmpl = tmpl.Funcs(t.funcs)
	}
	if _, err := tmpl.Parse(string(data)); err != nil {
		return fmt.Errorf("parse prompt template %q: %w", t.path, err)
	}
	t.tmpl = tmpl
	return nil
}

// Digest returns the sha256 hash of the template content.
func (t *Template) Digest() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.hash
}
