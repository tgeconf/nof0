package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func baseCfg() *Config {
	return &Config{
		BTCETHLeverage:         20,
		AltcoinLeverage:        10,
		MinConfidence:          75,
		MinRiskReward:          3.0,
		MaxPositions:           2,
		DecisionIntervalRaw:    "3m",
		DecisionTimeoutRaw:     "60s",
		MaxConcurrentDecisions: 1,
	}
}

func TestValidateDecisions_OpenLong_OK(t *testing.T) {
	cfg := baseCfg()
	ctx := &Context{Positions: nil}
	d := Decision{
		Symbol:          "BTC",
		Action:          "open_long",
		Leverage:        10,
		PositionSizeUSD: 100,
		EntryPrice:      100,
		StopLoss:        95,
		TakeProfit:      115,
		Confidence:      80,
	}
	err := ValidateDecisions(cfg, ctx, []Decision{d})
	assert.NoError(t, err, "ValidateDecisions should not error for valid open_long decision")
}

func TestValidateDecisions_RR_Fails(t *testing.T) {
	cfg := baseCfg()
	ctx := &Context{}
	d := Decision{
		Symbol:          "ETH",
		Action:          "open_long",
		Leverage:        5,
		PositionSizeUSD: 100,
		EntryPrice:      100,
		StopLoss:        90,
		TakeProfit:      105, // RR = 0.5 < 3.0
		Confidence:      90,
	}
	err := ValidateDecisions(cfg, ctx, []Decision{d})
	assert.Error(t, err, "ValidateDecisions should fail due to insufficient risk/reward ratio")
}

func TestValidateDecisions_LeverageCap_Fails(t *testing.T) {
	cfg := baseCfg()
	ctx := &Context{}
	d := Decision{
		Symbol:          "PEPE",
		Action:          "open_long",
		Leverage:        50, // exceeds alt cap
		PositionSizeUSD: 100,
		EntryPrice:      1,
		StopLoss:        0.9,
		TakeProfit:      1.5,
		Confidence:      90,
	}
	err := ValidateDecisions(cfg, ctx, []Decision{d})
	assert.Error(t, err, "ValidateDecisions should fail due to leverage cap exceeded")
}

func TestValidateDecisions_MaxPositions_Fails(t *testing.T) {
	cfg := baseCfg()
	ctx := &Context{Positions: []PositionInfo{{Symbol: "A"}, {Symbol: "B"}}} // already 2
	d := Decision{
		Symbol:          "C",
		Action:          "open_long",
		Leverage:        2,
		PositionSizeUSD: 100,
		EntryPrice:      10,
		StopLoss:        9,
		TakeProfit:      13,
		Confidence:      80,
	}
	err := ValidateDecisions(cfg, ctx, []Decision{d})
	assert.Error(t, err, "ValidateDecisions should fail due to max positions limit exceeded")
}

func TestValidateDecisions_NoAddOrHedge(t *testing.T) {
	cfg := baseCfg()
	ctx := &Context{Positions: []PositionInfo{{Symbol: "BTC", Side: "long"}}}
	d := Decision{
		Symbol:          "BTC",
		Action:          "open_short",
		Leverage:        2,
		PositionSizeUSD: 100,
		EntryPrice:      10,
		StopLoss:        11,
		TakeProfit:      7,
		Confidence:      80,
	}
	err := ValidateDecisions(cfg, ctx, []Decision{d})
	assert.Error(t, err, "ValidateDecisions should reject hedging positions")
}

func TestValidateDecisions_RiskAndSizeCaps(t *testing.T) {
	cfg := baseCfg()
	ctx := &Context{Account: AccountInfo{TotalEquity: 10000}, MaxRiskPct: 2, MaxPositionSizeUSD: 150}
	// risk within 2% of equity (=200), size within 150
	ok := Decision{Symbol: "ETH", Action: "open_short", Leverage: 3, PositionSizeUSD: 150, EntryPrice: 100, StopLoss: 110, TakeProfit: 70, Confidence: 90, RiskUSD: 100}
	err := ValidateDecisions(cfg, ctx, []Decision{ok})
	assert.NoError(t, err, "ValidateDecisions should not error for valid risk and size caps")

	badRisk := ok
	badRisk.RiskUSD = 500
	err = ValidateDecisions(cfg, ctx, []Decision{badRisk})
	assert.Error(t, err, "ValidateDecisions should fail due to risk cap exceeded")

	badSize := ok
	badSize.PositionSizeUSD = 151
	err = ValidateDecisions(cfg, ctx, []Decision{badSize})
	assert.Error(t, err, "ValidateDecisions should fail due to position size cap exceeded")
}

func TestValidateDecisions_Close_NoPosition_Fails(t *testing.T) {
	cfg := baseCfg()
	ctx := &Context{Positions: nil}
	d := Decision{Symbol: "BTC", Action: "close_long"}
	err := ValidateDecisions(cfg, ctx, []Decision{d})
	assert.Error(t, err, "ValidateDecisions should fail when closing non-existent position")
}

func TestValidateDecisions_Close_WithMatching_Passes(t *testing.T) {
	cfg := baseCfg()
	ctx := &Context{Positions: []PositionInfo{{Symbol: "BTC", Side: "long"}}}
	d := Decision{Symbol: "BTC", Action: "close_long"}
	err := ValidateDecisions(cfg, ctx, []Decision{d})
	assert.NoError(t, err, "ValidateDecisions should not error for valid close position")
}
