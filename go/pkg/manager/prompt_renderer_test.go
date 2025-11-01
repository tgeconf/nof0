package manager

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManagerPromptRenderer(t *testing.T) {
	templatePath := filepath.Join("..", "..", "etc", "prompts", "manager", "aggressive_short.tmpl")
	renderer, err := NewPromptRenderer(templatePath)
	if err != nil {
		t.Fatalf("NewPromptRenderer error: %v", err)
	}

	trader := &TraderConfig{
		ID:               "trader_aggressive_short",
		Name:             "Aggressive Short",
		ExchangeProvider: "hyperliquid",
		MarketProvider:   "hl_market",
		PromptTemplate:   templatePath,
		DecisionInterval: 3 * time.Minute,
		AllocationPct:    40,
		AutoStart:        true,
		RiskParams: RiskParameters{
			MaxPositions:       3,
			MaxPositionSizeUSD: 500,
			MaxMarginUsagePct:  60,
			BTCETHLeverage:     20,
			AltcoinLeverage:    10,
			MinRiskRewardRatio: 3.0,
			MinConfidence:      75,
			StopLossEnabled:    true,
			TakeProfitEnabled:  true,
		},
	}

	out, err := renderer.Render(ManagerPromptInputs{
		Trader:      trader,
		ContextJSON: `{"market":"bearish"}`,
	})
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	expectations := []string{
		"Trader ID: trader_aggressive_short",
		"Decision Interval: 3m0s",
		"Allocation %: 40.00",
		"max_positions=3",
		"Min Confidence: 75",
		`{"market":"bearish"}`,
	}
	for _, substr := range expectations {
		if !strings.Contains(out, substr) {
			t.Fatalf("rendered prompt missing substring %q\n--- prompt ---\n%s", substr, out)
		}
	}
}

func TestManagerPromptRendererMissingTrader(t *testing.T) {
	templatePath := filepath.Join("..", "..", "etc", "prompts", "manager", "aggressive_short.tmpl")
	renderer, err := NewPromptRenderer(templatePath)
	if err != nil {
		t.Fatalf("NewPromptRenderer error: %v", err)
	}
	if _, err := renderer.Render(ManagerPromptInputs{}); err == nil {
		t.Fatal("expected error for missing trader data")
	}
}
