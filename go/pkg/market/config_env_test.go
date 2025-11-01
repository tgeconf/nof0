package market_test

import (
	"os"
	"path/filepath"
	"testing"

	market "nof0-api/pkg/market"
	_ "nof0-api/pkg/market/exchanges/hyperliquid"
)

// Ensures env placeholders are expanded and durations parsed.
func TestMarketConfig_EnvExpansionAndDurations(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BASE_URL_VAR", "https://api.hyperliquid.test/info")
	t.Setenv("TOUT", "9s")
	t.Setenv("HTTP_TOUT", "13s")

	yaml := []byte(`
default: hp
providers:
  hp:
    type: hyperliquid
    base_url: ${BASE_URL_VAR}
    timeout: ${TOUT}
    http_timeout: ${HTTP_TOUT}
`)
	path := filepath.Join(dir, "market.yaml")
	if err := os.WriteFile(path, yaml, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := market.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	p := cfg.Providers["hp"]
	if p == nil {
		t.Fatalf("provider hp missing")
	}
	if p.BaseURL != "https://api.hyperliquid.test/info" {
		t.Fatalf("BaseURL not expanded, got %q", p.BaseURL)
	}
	if p.Timeout.String() != "9s" || p.HTTPTimeout.String() != "13s" {
		t.Fatalf("durations not parsed, timeout=%s http_timeout=%s", p.Timeout, p.HTTPTimeout)
	}
}
