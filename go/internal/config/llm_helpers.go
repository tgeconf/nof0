package config

import (
	"fmt"
	"path/filepath"

	"nof0-api/pkg/llm"
)

// MustLoadLLM loads etc/llm.yaml from the project root and panics on error.
func MustLoadLLM() *llm.Config {
	root := MustProjectRoot()
	path := filepath.Join(root, "etc", "llm.yaml")
	cfg, err := llm.LoadConfig(path)
	if err != nil {
		panic(fmt.Errorf("load llm config %s: %w", path, err))
	}
	return cfg
}
