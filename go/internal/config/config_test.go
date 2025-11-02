package config

import (
	"os"
	"path/filepath"
	"testing"

	"nof0-api/pkg/llm"
	"nof0-api/pkg/market"
)

// Test_moduleConfig_envExpansion verifies that module configs expand environment
// variables correctly when loaded directly via their LoadConfig functions.
func Test_moduleConfig_envExpansion(t *testing.T) {
	dir := t.TempDir()

	// Prepare llm.yaml using env placeholders
	llmYAML := []byte(`
base_url: ${ZENMUX_BASE_URL}
api_key: ${ZENMUX_API_KEY}
default_model: ${ZENMUX_DEFAULT_MODEL}
timeout: 2s
`)
	llmPath := filepath.Join(dir, "llm.yaml")
	if err := os.WriteFile(llmPath, llmYAML, 0o600); err != nil {
		t.Fatalf("write llm.yaml: %v", err)
	}

	// Prepare market.yaml using env placeholders for durations
	marketYAML := []byte(`
default: hyper
providers:
  hyper:
    type: hyperliquid
    testnet: true
    timeout: ${HLIQ_TIMEOUT}
    http_timeout: ${HLIQ_HTTP_TIMEOUT}
    max_retries: 2
`)
	mktPath := filepath.Join(dir, "market.yaml")
	if err := os.WriteFile(mktPath, marketYAML, 0o600); err != nil {
		t.Fatalf("write market.yaml: %v", err)
	}

	// Set envs consumed by the files above
	t.Setenv("ZENMUX_BASE_URL", "https://zenmux.example/api")
	t.Setenv("ZENMUX_API_KEY", "test-key")
	t.Setenv("ZENMUX_DEFAULT_MODEL", "gpt-x")
	t.Setenv("HLIQ_TIMEOUT", "7s")
	t.Setenv("HLIQ_HTTP_TIMEOUT", "11s")

	// Load LLM config and verify env expansion
	llmCfg, err := llm.LoadConfig(llmPath)
	if err != nil {
		t.Fatalf("llm.LoadConfig: %v", err)
	}
	if got := llmCfg.BaseURL; got != "https://zenmux.example/api" {
		t.Fatalf("LLM.BaseURL not expanded, got %q", got)
	}
	if got := llmCfg.APIKey; got != "test-key" {
		t.Fatalf("LLM.APIKey not expanded, got %q", got)
	}
	if got := llmCfg.DefaultModel; got != "gpt-x" {
		t.Fatalf("LLM.DefaultModel got %q", got)
	}

	// Load Market config and verify env expansion
	mktCfg, err := market.LoadConfig(mktPath)
	if err != nil {
		t.Fatalf("market.LoadConfig: %v", err)
	}
	p := mktCfg.Providers["hyper"]
	if p == nil {
		t.Fatalf("Market provider 'hyper' missing")
	}
	if !p.Testnet {
		t.Fatalf("Market Testnet flag not parsed")
	}
	if p.Timeout.String() != "7s" || p.HTTPTimeout.String() != "11s" {
		t.Fatalf("Market timeouts not parsed, got timeout=%s http_timeout=%s", p.Timeout, p.HTTPTimeout)
	}
}

func TestValidate_TTLBounds(t *testing.T) {
	cfg := &Config{}
	cfg.DataPath = "./data"
	cfg.TTL.Short = 0
	cfg.TTL.Medium = 60
	cfg.TTL.Long = 300
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected ttl.short validation error")
	}
}
