package manager

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestManagerPromptRenderer(t *testing.T) {
	templatePath := filepath.Join("..", "..", "etc", "prompts", "manager", "aggressive_short.tmpl")
	renderer, err := NewPromptRenderer(templatePath)
	assert.NoError(t, err, "NewPromptRenderer should not error")
	assert.NotNil(t, renderer, "renderer should not be nil")

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
	assert.NoError(t, err, "Render should not error")
	assert.NotEmpty(t, out, "rendered output should not be empty")

	expectations := []string{
		"Trader ID: trader_aggressive_short",
		"Decision Interval: 3m0s",
		"Allocation %: 40.00",
		"max_positions=3",
		"Min Confidence: 75",
		`{"market":"bearish"}`,
	}
	for _, substr := range expectations {
		assert.Contains(t, out, substr, "rendered prompt should contain %q", substr)
	}
}

func TestManagerPromptRendererMissingTrader(t *testing.T) {
	templatePath := filepath.Join("..", "..", "etc", "prompts", "manager", "aggressive_short.tmpl")
	renderer, err := NewPromptRenderer(templatePath)
	assert.NoError(t, err, "NewPromptRenderer should not error")
	assert.NotNil(t, renderer, "renderer should not be nil")

	_, err = renderer.Render(ManagerPromptInputs{})
	assert.Error(t, err, "Render should error for missing trader data")
}
