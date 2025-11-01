package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestTemplateRender(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "example.tmpl")
	err := os.WriteFile(templatePath, []byte("hello {{ .Name }} - {{ toUpper .Role }}"), 0o600)
	if err != nil {
		t.Fatalf("write template: %v", err)
	}

	funcs := template.FuncMap{
		"toUpper": strings.ToUpper,
	}
	tpl, err := NewTemplate(templatePath, funcs)
	if err != nil {
		t.Fatalf("NewTemplate error: %v", err)
	}

	out, err := tpl.Render(map[string]any{"Name": "Alice", "Role": "trader"})
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if out != "hello Alice - TRADER" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestTemplateReload(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "reload.tmpl")
	if err := os.WriteFile(templatePath, []byte("v1"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	tpl, err := NewTemplate(templatePath, nil)
	if err != nil {
		t.Fatalf("NewTemplate: %v", err)
	}
	out, err := tpl.Render(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if out != "v1" {
		t.Fatalf("expected v1, got %q", out)
	}
	digestV1 := tpl.Digest()
	if digestV1 == "" {
		t.Fatalf("expected non-empty digest")
	}

	if err := os.WriteFile(templatePath, []byte("v2"), 0o600); err != nil {
		t.Fatalf("rewrite template: %v", err)
	}
	if err := tpl.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	out, err = tpl.Render(nil)
	if err != nil {
		t.Fatalf("Render after reload: %v", err)
	}
	if out != "v2" {
		t.Fatalf("expected v2, got %q", out)
	}
	if tpl.Digest() == digestV1 {
		t.Fatalf("digest did not change after reload")
	}
}
