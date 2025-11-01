package executor

import "testing"

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
	if err := ValidateDecisions(cfg, ctx, []Decision{d}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	if err := ValidateDecisions(cfg, ctx, []Decision{d}); err == nil {
		t.Fatal("expected rr failure")
	}
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
	if err := ValidateDecisions(cfg, ctx, []Decision{d}); err == nil {
		t.Fatal("expected leverage cap failure")
	}
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
	if err := ValidateDecisions(cfg, ctx, []Decision{d}); err == nil {
		t.Fatal("expected max positions failure")
	}
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
	if err := ValidateDecisions(cfg, ctx, []Decision{d}); err == nil {
		t.Fatal("expected hedging to be rejected")
	}
}

func TestValidateDecisions_RiskAndSizeCaps(t *testing.T) {
	cfg := baseCfg()
	ctx := &Context{Account: AccountInfo{TotalEquity: 10000}, MaxRiskPct: 2, MaxPositionSizeUSD: 150}
	// risk within 2% of equity (=200), size within 150
	ok := Decision{Symbol: "ETH", Action: "open_short", Leverage: 3, PositionSizeUSD: 150, EntryPrice: 100, StopLoss: 110, TakeProfit: 70, Confidence: 90, RiskUSD: 100}
	if err := ValidateDecisions(cfg, ctx, []Decision{ok}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	badRisk := ok
	badRisk.RiskUSD = 500
	if err := ValidateDecisions(cfg, ctx, []Decision{badRisk}); err == nil {
		t.Fatal("expected risk cap failure")
	}
	badSize := ok
	badSize.PositionSizeUSD = 151
	if err := ValidateDecisions(cfg, ctx, []Decision{badSize}); err == nil {
		t.Fatal("expected size cap failure")
	}
}

func TestValidateDecisions_Close_NoPosition_Fails(t *testing.T) {
	cfg := baseCfg()
	ctx := &Context{Positions: nil}
	d := Decision{Symbol: "BTC", Action: "close_long"}
	if err := ValidateDecisions(cfg, ctx, []Decision{d}); err == nil {
		t.Fatal("expected close without position to fail")
	}
}

func TestValidateDecisions_Close_WithMatching_Passes(t *testing.T) {
	cfg := baseCfg()
	ctx := &Context{Positions: []PositionInfo{{Symbol: "BTC", Side: "long"}}}
	d := Decision{Symbol: "BTC", Action: "close_long"}
	if err := ValidateDecisions(cfg, ctx, []Decision{d}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
