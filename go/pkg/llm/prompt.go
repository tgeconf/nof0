package llm

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
)

// PromptTemplate wraps a text/template loaded from disk with optional function map.
type PromptTemplate struct {
	path  string
	funcs template.FuncMap

	mu   sync.RWMutex
	tmpl *template.Template
	hash string
}

// NewPromptTemplate parses the template at path using the provided template functions.
func NewPromptTemplate(path string, funcs template.FuncMap) (*PromptTemplate, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("prompt template path is empty")
	}
	t := &PromptTemplate{
		path:  path,
		funcs: funcs,
	}
	if err := t.reload(); err != nil {
		return nil, err
	}
	return t, nil
}

// Render executes the template with the provided data and returns the rendered string.
func (t *PromptTemplate) Render(data any) (string, error) {
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
func (t *PromptTemplate) Reload() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.reload()
}

func (t *PromptTemplate) reload() error {
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
func (t *PromptTemplate) Digest() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.hash
}

// DigestString returns the sha256 digest for the provided string.
func DigestString(s string) string {
	return computeDigest([]byte(s))
}

func computeDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
