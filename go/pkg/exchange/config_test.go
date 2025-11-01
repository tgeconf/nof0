package exchange_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	exchange "nof0-api/pkg/exchange"
	_ "nof0-api/pkg/exchange/hyperliquid"
)

const testPrivateKey = "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f"

func TestLoadConfigAndBuildProviders(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("EXCHANGE_PRIVATE_KEY", testPrivateKey)
	t.Cleanup(func() {
		os.Unsetenv("EXCHANGE_PRIVATE_KEY")
	})

	configYAML := `
default: hyperliquid_main
providers:
  hyperliquid_main:
    type: hyperliquid
    private_key: ${EXCHANGE_PRIVATE_KEY}
    timeout: 45s
    testnet: true
    vault_address: 0x0000000000000000000000000000000000000000
`
	path := filepath.Join(dir, "exchange.yaml")
	err := os.WriteFile(path, []byte(configYAML), 0o600)
	assert.NoError(t, err, "write config should succeed")

	cfg, err := exchange.LoadConfig(path)
	assert.NoError(t, err, "LoadConfig should not error")
	assert.NotNil(t, cfg, "config should not be nil")
	assert.Equal(t, "hyperliquid_main", cfg.Default, "default should be hyperliquid_main")

	providers, err := cfg.BuildProviders()
	assert.NoError(t, err, "BuildProviders should not error")
	assert.Len(t, providers, 1, "should have 1 provider")
	assert.Contains(t, providers, "hyperliquid_main", "provider map should contain hyperliquid_main")
}

func TestLoadConfigRequiresPrivateKey(t *testing.T) {
	dir := t.TempDir()
	configYAML := `
providers:
  hyperliquid_main:
    type: hyperliquid
`
	path := filepath.Join(dir, "exchange.yaml")
	err := os.WriteFile(path, []byte(configYAML), 0o600)
	assert.NoError(t, err, "write config should succeed")

	_, err = exchange.LoadConfig(path)
	assert.Error(t, err, "LoadConfig should error for missing private_key")
	assert.Contains(t, err.Error(), "private_key", "error should mention private_key")
}
