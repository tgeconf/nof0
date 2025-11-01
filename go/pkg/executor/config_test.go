package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	promptPath := filepath.Join(dir, "prompts/executor/default_prompt.tmpl")
	if err := os.MkdirAll(filepath.Dir(promptPath), 0o700); err != nil {
		t.Fatalf("mkdir prompts: %v", err)
	}
	if err := os.WriteFile(promptPath, []byte("base executor prompt"), 0o600); err != nil {
		t.Fatalf("write prompt: %v", err)
	}

	configYAML := `
model_alias: gpt-5
prompt_template: prompts/executor/default_prompt.tmpl
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
overrides:
  trader_alpha:
    min_confidence: 80
`
	path := filepath.Join(dir, "executor.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	wantPrompt := filepath.Join(dir, "prompts/executor/default_prompt.tmpl")
	if cfg.PromptTemplate != wantPrompt {
		t.Fatalf("PromptTemplate = %q, want %q", cfg.PromptTemplate, wantPrompt)
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
	if cfg.Overrides["trader_alpha"].MinConfidence == nil || *cfg.Overrides["trader_alpha"].MinConfidence != 80 {
		t.Fatalf("Override min_confidence not parsed: %+v", cfg.Overrides["trader_alpha"])
	}
}

func TestLoadConfigInvalidMinRiskReward(t *testing.T) {
	dir := t.TempDir()
	promptPath := filepath.Join(dir, "prompts/executor/default_prompt.tmpl")
	if err := os.MkdirAll(filepath.Dir(promptPath), 0o700); err != nil {
		t.Fatalf("mkdir prompts: %v", err)
	}
	if err := os.WriteFile(promptPath, []byte("base executor prompt"), 0o600); err != nil {
		t.Fatalf("write prompt: %v", err)
	}

	configYAML := `
model_alias: gpt-5
prompt_template: prompts/executor/default_prompt.tmpl
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

func TestLoadConfigMissingPrompt(t *testing.T) {
	dir := t.TempDir()
	configYAML := `
model_alias: gpt-5
prompt_template: prompts/executor/default_prompt.tmpl
btc_eth_leverage: 20
altcoin_leverage: 10
min_confidence: 75
min_risk_reward: 3.5
max_positions: 4
`
	path := filepath.Join(dir, "executor.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "prompt_template") {
		t.Fatalf("expected prompt_template error, got %v", err)
	}
}

func TestValidateFails(t *testing.T) {
	dir := t.TempDir()
	configYAML := `
model_alias: ""
prompt_template: ""
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
	if !strings.Contains(err.Error(), "model_alias") {
		t.Fatalf("unexpected error: %v", err)
	}
}
