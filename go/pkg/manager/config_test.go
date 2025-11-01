package manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	managerPromptAgg := filepath.Join(dir, "prompts/manager/aggressive_short.tmpl")
	managerPromptCon := filepath.Join(dir, "prompts/manager/conservative_long.tmpl")
	if err := os.MkdirAll(filepath.Dir(managerPromptAgg), 0o700); err != nil {
		t.Fatalf("mkdir prompts: %v", err)
	}
	if err := os.WriteFile(managerPromptAgg, []byte("aggressive short prompt"), 0o600); err != nil {
		t.Fatalf("write aggressive prompt: %v", err)
	}
	if err := os.WriteFile(managerPromptCon, []byte("conservative long prompt"), 0o600); err != nil {
		t.Fatalf("write conservative prompt: %v", err)
	}

	configYAML := `
manager:
  total_equity_usd: 10000
  reserve_equity_pct: 10
  allocation_strategy: performance_based
  rebalance_interval: 2h
  state_storage_backend: file
  state_storage_path: ./state/manager.json

traders:
  - id: trader_a
    name: Aggressive Short
    exchange_provider: " hyperliquid_primary "
    market_provider: " hl_market "
    prompt_template: prompts/manager/aggressive_short.tmpl
    decision_interval: 4m
    allocation_pct: 40
    auto_start: true
    risk_params:
      max_positions: 3
      max_position_size_usd: 500
      max_margin_usage_pct: 60
      btc_eth_leverage: 20
      altcoin_leverage: 10
      min_risk_reward_ratio: 3.0
      min_confidence: 75
      stop_loss_enabled: true
      take_profit_enabled: true

  - id: trader_b
    name: Conservative Long
    exchange_provider: hyperliquid_secondary
    market_provider: hl_market
    prompt_template: prompts/manager/conservative_long.tmpl
    decision_interval: 5m
    allocation_pct: 30
    auto_start: false
    risk_params:
      max_positions: 2
      max_position_size_usd: 300
      max_margin_usage_pct: 50
      btc_eth_leverage: 10
      altcoin_leverage: 5
      min_risk_reward_ratio: 2.5
      min_confidence: 80
      stop_loss_enabled: true
      take_profit_enabled: true

monitoring:
  update_interval: 15s
  metrics_exporter: prometheus
`
	path := filepath.Join(dir, "manager.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write manager config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Manager.RebalanceInterval.String() != "2h0m0s" {
		t.Fatalf("RebalanceInterval = %s", cfg.Manager.RebalanceInterval)
	}
	if cfg.Traders[0].DecisionInterval.String() != "4m0s" {
		t.Fatalf("DecisionInterval = %s", cfg.Traders[0].DecisionInterval)
	}
	if cfg.Traders[0].ExchangeProvider != "hyperliquid_primary" {
		t.Fatalf("ExchangeProvider not trimmed: %q", cfg.Traders[0].ExchangeProvider)
	}
	if cfg.Traders[0].MarketProvider != "hl_market" {
		t.Fatalf("MarketProvider not trimmed: %q", cfg.Traders[0].MarketProvider)
	}
	wantStatePath := filepath.Join(dir, "state/manager.json")
	if cfg.Manager.StateStoragePath != wantStatePath {
		t.Fatalf("StateStoragePath = %q, want %q", cfg.Manager.StateStoragePath, wantStatePath)
	}
}

func TestAllocationValidation(t *testing.T) {
	dir := t.TempDir()
	promptPath := filepath.Join(dir, "prompt.tmpl")
	if err := os.WriteFile(promptPath, []byte("generic prompt"), 0o600); err != nil {
		t.Fatalf("write prompt: %v", err)
	}

	configYAML := `
manager:
  total_equity_usd: 1000
  reserve_equity_pct: 0
  allocation_strategy: equal
  rebalance_interval: 1h
  state_storage_backend: file
  state_storage_path: state.json

traders:
  - id: t1
    name: Trader1
    exchange_provider: ex
    market_provider: market_a
    prompt_template: prompt.tmpl
    decision_interval: 3m
    allocation_pct: 80
    auto_start: true
    risk_params:
      max_positions: 1
      max_position_size_usd: 100
      max_margin_usage_pct: 50
      btc_eth_leverage: 10
      altcoin_leverage: 5
      min_risk_reward_ratio: 2
      min_confidence: 70
      stop_loss_enabled: true
      take_profit_enabled: true

  - id: t2
    name: Trader2
    exchange_provider: ex
    market_provider: market_a
    prompt_template: prompt.tmpl
    decision_interval: 3m
    allocation_pct: 30
    auto_start: true
    risk_params:
      max_positions: 1
      max_position_size_usd: 100
      max_margin_usage_pct: 50
      btc_eth_leverage: 10
      altcoin_leverage: 5
      min_risk_reward_ratio: 2
      min_confidence: 70
      stop_loss_enabled: true
      take_profit_enabled: true

monitoring:
  update_interval: 10s
  metrics_exporter: prometheus
`
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "allocation") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfigMissingPrompt(t *testing.T) {
	dir := t.TempDir()
	configYAML := `
manager:
  total_equity_usd: 1000
  reserve_equity_pct: 0
  allocation_strategy: equal
  rebalance_interval: 1h
  state_storage_backend: file
  state_storage_path: state.json

traders:
  - id: t1
    name: Trader1
    exchange_provider: ex
    market_provider: market_a
    prompt_template: missing.tmpl
    decision_interval: 3m
    allocation_pct: 50
    auto_start: true
    risk_params:
      max_positions: 1
      max_position_size_usd: 100
      max_margin_usage_pct: 50
      btc_eth_leverage: 10
      altcoin_leverage: 5
      min_risk_reward_ratio: 2
      min_confidence: 70
      stop_loss_enabled: true
      take_profit_enabled: true

monitoring:
  update_interval: 10s
  metrics_exporter: prometheus
`
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "prompt_template") {
		t.Fatalf("expected prompt_template error, got %v", err)
	}
}

func TestMissingMarketProvider(t *testing.T) {
	dir := t.TempDir()
	promptPath := filepath.Join(dir, "prompt.tmpl")
	if err := os.WriteFile(promptPath, []byte("generic prompt"), 0o600); err != nil {
		t.Fatalf("write prompt: %v", err)
	}

	configYAML := `
manager:
  total_equity_usd: 1000
  reserve_equity_pct: 0
  allocation_strategy: equal
  rebalance_interval: 1h
  state_storage_backend: file
  state_storage_path: state.json

traders:
  - id: t1
    name: Trader1
    exchange_provider: ex
    prompt_template: prompt.tmpl
    decision_interval: 3m
    allocation_pct: 50
    auto_start: true
    risk_params:
      max_positions: 1
      max_position_size_usd: 100
      max_margin_usage_pct: 50
      btc_eth_leverage: 10
      altcoin_leverage: 5
      min_risk_reward_ratio: 2
      min_confidence: 70
      stop_loss_enabled: true
      take_profit_enabled: true

monitoring:
  update_interval: 10s
  metrics_exporter: prometheus
`
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "market_provider") {
		t.Fatalf("expected market_provider error, got %v", err)
	}
}
