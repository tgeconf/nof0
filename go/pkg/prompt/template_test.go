package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestTemplateRender(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "example.tmpl")
	err := os.WriteFile(templatePath, []byte("hello {{ .Name }} - {{ toUpper .Role }}"), 0o600)
	assert.NoError(t, err, "write template should succeed")

	funcs := template.FuncMap{
		"toUpper": strings.ToUpper,
	}
	tpl, err := NewTemplate(templatePath, funcs)
	assert.NoError(t, err, "NewTemplate should not error")
	assert.NotNil(t, tpl, "template should not be nil")

	out, err := tpl.Render(map[string]any{"Name": "Alice", "Role": "trader"})
	assert.NoError(t, err, "Render should not error")
	assert.Equal(t, "hello Alice - TRADER", out, "rendered output should match expected")
}

func TestTemplateReload(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "reload.tmpl")
	err := os.WriteFile(templatePath, []byte("v1"), 0o600)
	assert.NoError(t, err, "write template should succeed")

	tpl, err := NewTemplate(templatePath, nil)
	assert.NoError(t, err, "NewTemplate should not error")
	assert.NotNil(t, tpl, "template should not be nil")

	out, err := tpl.Render(nil)
	assert.NoError(t, err, "Render should not error")
	assert.Equal(t, "v1", out, "initial render should be v1")

	digestV1 := tpl.Digest()
	assert.NotEmpty(t, digestV1, "digest should not be empty")

	err = os.WriteFile(templatePath, []byte("v2"), 0o600)
	assert.NoError(t, err, "rewrite template should succeed")

	err = tpl.Reload()
	assert.NoError(t, err, "Reload should not error")

	out, err = tpl.Render(nil)
	assert.NoError(t, err, "Render after reload should not error")
	assert.Equal(t, "v2", out, "reloaded render should be v2")

	digestV2 := tpl.Digest()
	assert.NotEqual(t, digestV1, digestV2, "digest should change after reload")
}
