package market_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	market "nof0-api/pkg/market"
	_ "nof0-api/pkg/market/exchanges/hyperliquid"
)

func TestLoadMarketConfig(t *testing.T) {
	dir := t.TempDir()
	configYAML := `
default: hyperliquid
providers:
  hyperliquid:
    type: hyperliquid
    base_url: https://api.hyperliquid.xyz/info
    timeout: 6s
    http_timeout: 12s
    max_retries: 4
`
	path := filepath.Join(dir, "market.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := market.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Default != "hyperliquid" {
		t.Fatalf("unexpected default: %s", cfg.Default)
	}

	providers, err := cfg.BuildProviders()
	if err != nil {
		t.Fatalf("BuildProviders error: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	if _, ok := providers["hyperliquid"]; !ok {
		t.Fatalf("provider map missing hyperliquid")
	}
}

func TestMarketConfigInvalidType(t *testing.T) {
	dir := t.TempDir()
	configYAML := `
providers:
  demo:
    type: foobar
`
	path := filepath.Join(dir, "market.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := market.LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported type error, got %v", err)
	}
}
