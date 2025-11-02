package market_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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
    testnet: false
    timeout: 6s
    http_timeout: 12s
    max_retries: 4
`
	path := filepath.Join(dir, "market.yaml")
	err := os.WriteFile(path, []byte(configYAML), 0o600)
	assert.NoError(t, err, "write config should succeed")

	cfg, err := market.LoadConfig(path)
	assert.NoError(t, err, "LoadConfig should not error")
	assert.NotNil(t, cfg, "config should not be nil")
	assert.Equal(t, "hyperliquid", cfg.Default, "default should be hyperliquid")

	providers, err := cfg.BuildProviders()
	assert.NoError(t, err, "BuildProviders should not error")
	assert.Len(t, providers, 1, "should have 1 provider")
	assert.Contains(t, providers, "hyperliquid", "provider map should contain hyperliquid")
	assert.False(t, cfg.Providers["hyperliquid"].Testnet)
}

func TestMarketConfigInvalidType(t *testing.T) {
	dir := t.TempDir()
	configYAML := `
providers:
  demo:
    type: foobar
`
	path := filepath.Join(dir, "market.yaml")
	err := os.WriteFile(path, []byte(configYAML), 0o600)
	assert.NoError(t, err, "write config should succeed")

	_, err = market.LoadConfig(path)
	assert.Error(t, err, "LoadConfig should error for unsupported type")
	assert.Contains(t, err.Error(), "unsupported", "error should mention unsupported")
}

// TestMarketConfigValidationEdgeCases tests various validation edge cases.
func TestMarketConfigValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		wantErr     bool
		errContains string
	}{
		{
			name: "empty provider name",
			configYAML: `
providers:
  "":
    type: hyperliquid
`,
			wantErr:     true,
			errContains: "name cannot be empty",
		},
		{
			name: "negative timeout",
			configYAML: `
providers:
  test:
    type: hyperliquid
    timeout: -5s
`,
			wantErr:     true,
			errContains: "timeout must be positive",
		},
		{
			name: "default provider not found",
			configYAML: `
default: nonexistent
providers:
  test:
    type: hyperliquid
`,
			wantErr:     true,
			errContains: "default provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "market.yaml")
			err := os.WriteFile(path, []byte(tt.configYAML), 0o600)
			assert.NoError(t, err)

			_, err = market.LoadConfig(path)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
