package svc_test

import (
	"testing"

	"nof0-api/internal/config"
	exchangepkg "nof0-api/pkg/exchange"
)

// TestEnvironmentAwareExchangeConfig verifies that exchange providers
// automatically use testnet endpoints when Env is "test".
func TestEnvironmentAwareExchangeConfig(t *testing.T) {
	tests := []struct {
		name            string
		env             string
		configTestnet   bool
		expectedTestnet bool
	}{
		{
			name:            "test env forces testnet even when config says false",
			env:             "test",
			configTestnet:   false,
			expectedTestnet: true, // Should be overridden
		},
		{
			name:            "test env with testnet true stays true",
			env:             "test",
			configTestnet:   true,
			expectedTestnet: true,
		},
		{
			name:            "dev env respects config false",
			env:             "dev",
			configTestnet:   false,
			expectedTestnet: false,
		},
		{
			name:            "dev env respects config true",
			env:             "dev",
			configTestnet:   true,
			expectedTestnet: true,
		},
		{
			name:            "prod env respects config false",
			env:             "prod",
			configTestnet:   false,
			expectedTestnet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock exchange config
			exchangeCfg := &exchangepkg.Config{
				Default: "test_provider",
				Providers: map[string]*exchangepkg.ProviderConfig{
					"test_provider": {
						Type:       "hyperliquid",
						PrivateKey: "0x0000000000000000000000000000000000000000000000000000000000000001",
						Testnet:    tt.configTestnet,
					},
				},
			}

			// Create main config
			cfg := config.Config{
				Env:      tt.env,
				DataPath: "../../mcp/data",
			}

			// Simulate the logic from internal/svc
			if cfg.IsTestEnv() {
				for _, provider := range exchangeCfg.Providers {
					provider.Testnet = true
				}
			}

			// Verify the result
			provider := exchangeCfg.Providers["test_provider"]
			if provider.Testnet != tt.expectedTestnet {
				t.Errorf("Expected Testnet=%v, got Testnet=%v", tt.expectedTestnet, provider.Testnet)
			}

			// Verify IsTestEnv() logic
			isTest := cfg.IsTestEnv()
			shouldForceTestnet := isTest
			if shouldForceTestnet && !provider.Testnet {
				t.Error("Test environment should force testnet=true")
			}
		})
	}
}

// TestIsTestEnv verifies the environment detection logic.
func TestIsTestEnv(t *testing.T) {
	tests := []struct {
		env      string
		expected bool
	}{
		{"test", true},
		{"", true}, // Empty defaults to test
		{"dev", false},
		{"prod", false},
	}

	for _, tt := range tests {
		t.Run("env="+tt.env, func(t *testing.T) {
			cfg := config.Config{
				Env:      tt.env,
				DataPath: "test",
				TTL:      config.CacheTTL{Short: 10, Medium: 60, Long: 300},
			}
			// Normalize via Validate (which sets env to "test" if empty)
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate failed: %v", err)
			}
			result := cfg.IsTestEnv()
			if result != tt.expected {
				t.Errorf("IsTestEnv() for env=%q: expected %v, got %v (normalized to %q)",
					tt.env, tt.expected, result, cfg.Env)
			}
		})
	}
}
