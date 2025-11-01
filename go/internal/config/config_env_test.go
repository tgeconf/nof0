package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Test_hydrateSections_withEnvAndSectionFiles verifies env expansion and
// per-section hydration without going through go-zero conf.Load.
func Test_hydrateSections_withEnvAndSectionFiles(t *testing.T) {
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

	// Prepare market.yaml using env placeholders for base_url and durations
	marketYAML := []byte(`
default: hyper
providers:
  hyper:
    type: hyperliquid
    base_url: ${HLIQ_BASE}
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
	t.Setenv("HLIQ_BASE", "https://api.hyperliquid.local/info")
	t.Setenv("HLIQ_TIMEOUT", "7s")
	t.Setenv("HLIQ_HTTP_TIMEOUT", "11s")

	// Construct top-level config and hydrate sections
	cfg := &Config{
		DataPath: "./data",
		TTL:      CacheTTL{Short: 10, Medium: 60, Long: 300},
		LLM:      LLMSection{File: "llm.yaml"},
		Market:   MarketSection{File: "market.yaml"},
	}
	if err := cfg.hydrateSections(filepath.Join(dir, "nof0.yaml")); err != nil {
		t.Fatalf("hydrateSections: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if cfg.LLM.Config == nil {
		t.Fatalf("LLM.Config not hydrated")
	}
	if got := cfg.LLM.Config.BaseURL; got != "https://zenmux.example/api" {
		t.Fatalf("LLM.BaseURL not expanded, got %q", got)
	}
	if got := cfg.LLM.Config.APIKey; got != "test-key" {
		t.Fatalf("LLM.APIKey not expanded, got %q", got)
	}
	if got := cfg.LLM.Config.DefaultModel; got != "gpt-x" {
		t.Fatalf("LLM.DefaultModel got %q", got)
	}

	if cfg.Market.Config == nil {
		t.Fatalf("Market.Config not hydrated")
	}
	p := cfg.Market.Config.Providers["hyper"]
	if p == nil {
		t.Fatalf("Market provider 'hyper' missing")
	}
	if got := p.BaseURL; got != "https://api.hyperliquid.local/info" {
		t.Fatalf("Market BaseURL not expanded, got %q", got)
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
