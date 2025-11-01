package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	configYAML := `
btc_eth_leverage: 20
altcoin_leverage: 10
min_confidence: 75
min_risk_reward: 3.5
max_positions: 4
decision_interval: 2m
decision_timeout: 45s
max_concurrent_decisions: 2
allowed_trader_ids:
  - trader_alpha
  - trader_beta
signing_key: ${EXEC_SIGNING_KEY}
overrides:
  trader_alpha:
    min_confidence: 80
`
	path := filepath.Join(dir, "executor.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("EXEC_SIGNING_KEY", " secret ")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.DecisionInterval.String() != "2m0s" {
		t.Fatalf("DecisionInterval = %s, want 2m0s", cfg.DecisionInterval)
	}
	if cfg.DecisionTimeout.String() != "45s" {
		t.Fatalf("DecisionTimeout = %s, want 45s", cfg.DecisionTimeout)
	}
	if cfg.MaxConcurrentDecisions != 2 {
		t.Fatalf("MaxConcurrentDecisions = %d, want 2", cfg.MaxConcurrentDecisions)
	}
	if cfg.SigningKey != "secret" {
		t.Fatalf("SigningKey not trimmed/expanded: %q", cfg.SigningKey)
	}
	if cfg.Overrides["trader_alpha"].MinConfidence == nil || *cfg.Overrides["trader_alpha"].MinConfidence != 80 {
		t.Fatalf("Override min_confidence not parsed: %+v", cfg.Overrides["trader_alpha"])
	}
	expectedIDs := []string{"trader_alpha", "trader_beta"}
	for i, id := range expectedIDs {
		if cfg.AllowedTraderIDs[i] != id {
			t.Fatalf("AllowedTraderIDs[%d] = %q, want %q", i, cfg.AllowedTraderIDs[i], id)
		}
	}
}

func TestLoadConfigInvalidMinRiskReward(t *testing.T) {
	dir := t.TempDir()
	configYAML := `
btc_eth_leverage: 20
altcoin_leverage: 10
min_confidence: 75
min_risk_reward: -1
max_positions: 4
`
	path := filepath.Join(dir, "executor.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "min_risk_reward") {
		t.Fatalf("expected min_risk_reward validation error, got %v", err)
	}
}

func TestLoadConfigDuplicateTraderIDs(t *testing.T) {
	dir := t.TempDir()
	configYAML := `
btc_eth_leverage: 20
altcoin_leverage: 10
min_confidence: 75
min_risk_reward: 3.5
max_positions: 4
allowed_trader_ids:
  - trader_alpha
  - trader_alpha
`
	path := filepath.Join(dir, "executor.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate trader id error, got %v", err)
	}
}

func TestValidateFails(t *testing.T) {
	dir := t.TempDir()
	configYAML := `
btc_eth_leverage: 0
altcoin_leverage: -1
min_confidence: 150
min_risk_reward: -2
max_positions: 0
decision_interval: 1s
decision_timeout: 1s
`
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "btc_eth_leverage") {
		t.Fatalf("unexpected error: %v", err)
	}
}
