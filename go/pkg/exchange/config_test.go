package exchange_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := exchange.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Default != "hyperliquid_main" {
		t.Fatalf("unexpected default: %s", cfg.Default)
	}

	providers, err := cfg.BuildProviders()
	if err != nil {
		t.Fatalf("BuildProviders error: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	if _, ok := providers["hyperliquid_main"]; !ok {
		t.Fatalf("provider map missing hyperliquid_main")
	}
}

func TestLoadConfigRequiresPrivateKey(t *testing.T) {
	dir := t.TempDir()
	configYAML := `
providers:
  hyperliquid_main:
    type: hyperliquid
`
	path := filepath.Join(dir, "exchange.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := exchange.LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "private_key") {
		t.Fatalf("expected private_key error, got %v", err)
	}
}
