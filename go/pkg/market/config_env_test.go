package market_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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
	err := os.WriteFile(path, yaml, 0o600)
	assert.NoError(t, err, "write config should succeed")

	cfg, err := market.LoadConfig(path)
	assert.NoError(t, err, "LoadConfig should not error")
	assert.NotNil(t, cfg, "config should not be nil")

	p := cfg.Providers["hp"]
	assert.NotNil(t, p, "provider hp should exist")
	assert.Equal(t, "https://api.hyperliquid.test/info", p.BaseURL, "BaseURL should be expanded from env var")
	assert.Equal(t, "9s", p.Timeout.String(), "Timeout should be parsed as 9s")
	assert.Equal(t, "13s", p.HTTPTimeout.String(), "HTTPTimeout should be parsed as 13s")
}
