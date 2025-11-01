package executor

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptRenderer(t *testing.T) {
	templatePath := filepath.Join("..", "..", "etc", "prompts", "executor", "default_prompt.tmpl")
	cfg := &Config{
		BTCETHLeverage:         20,
		AltcoinLeverage:        8,
		MinConfidence:          75,
		MinRiskReward:          3.2,
		MaxPositions:           3,
		DecisionIntervalRaw:    "3m",
		DecisionTimeoutRaw:     "60s",
		MaxConcurrentDecisions: 1,
	}
	renderer, err := NewPromptRenderer(cfg, templatePath)
	if err != nil {
		t.Fatalf("NewPromptRenderer error: %v", err)
	}

	out, err := renderer.Render(PromptInputs{
		CurrentTime:     "2025-11-01T08:00:00Z",
		RuntimeMinutes:  120,
		SharpeRatio:     1.45,
		AccountOverview: "Equity: $12000\nBalance: $11800",
		OpenPositions:   "- BTC short 0.1 @ 65000",
		RiskBudget:      "Available risk: $250 (25% of cap)",
		PerformanceView: "WinRate: 60%",
		CandidateCoins:  "- BTC\n- ETH\n- SOL",
		MarketSnapshots: `{"BTC":{"price":64000}}`,
	})
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	expectations := []string{
		"TIMESTAMP: 2025-11-01T08:00:00Z",
		"UPTIME_MINUTES: 120",
		"ROLLING_SHARPE: 1.45",
		"BTC/ETH default 20x",
		"Minimum reward-to-risk ratio: 3.20",
		"RISK_BUDGET:",
		"Respect per-trader limits defined by Manager",
		"Available risk: $250",
		`"BTC":{"price":64000}`,
		"minimum confidence 75",
	}
	for _, substr := range expectations {
		if !strings.Contains(out, substr) {
			t.Fatalf("rendered prompt missing substring %q\n--- prompt ---\n%s", substr, out)
		}
	}
}

func TestPromptRendererNilConfig(t *testing.T) {
	if _, err := NewPromptRenderer(nil, ""); err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestPromptRendererEmptyPath(t *testing.T) {
	cfg := &Config{
		BTCETHLeverage:         20,
		AltcoinLeverage:        8,
		MinConfidence:          75,
		MinRiskReward:          3.2,
		MaxPositions:           3,
		DecisionIntervalRaw:    "3m",
		DecisionTimeoutRaw:     "60s",
		MaxConcurrentDecisions: 1,
	}
	if _, err := NewPromptRenderer(cfg, " "); err == nil {
		t.Fatal("expected error for empty template path")
	}
}
