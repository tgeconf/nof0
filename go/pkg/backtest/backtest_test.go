package backtest

import (
	"context"
	"testing"

	"math"
	simex "nof0-api/pkg/exchange/sim"
)

func TestBacktest_ThresholdWithSim(t *testing.T) {
	ctx := context.Background()
	exch := simex.New()
	assetID, err := exch.GetAssetIndex(ctx, "BTC")
	if err != nil {
		t.Fatalf("GetAssetIndex: %v", err)
	}

	feeder := NewPriceFeeder("BTC", []float64{100, 101, 103, 102, 99, 100})
	strat := &ThresholdStrategy{AssetID: assetID, ThresholdP: 1.0, LotSz: "0.01"}

	e := &Engine{Feeder: feeder, Strategy: strat, Exch: exch, Symbol: "BTC", FeeBps: 2.0, SlippageBps: 1.0, InitialEquity: 100000}
	res, err := e.Run(ctx)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Steps != 6 {
		t.Fatalf("steps=%d", res.Steps)
	}
	if res.OrdersSent == 0 {
		t.Fatalf("expected some orders, got 0")
	}
	if len(res.EquityCurve) != res.Steps {
		t.Fatalf("equity curve length mismatch")
	}
	// MaxDDPct, Sharpe are scenario-specific but should be finite numbers
	if res.MaxDDPct < 0 || math.IsNaN(res.MaxDDPct) {
		t.Fatalf("invalid max drawdown: %v", res.MaxDDPct)
	}
	if math.IsNaN(res.Sharpe) {
		t.Fatalf("invalid sharpe: %v", res.Sharpe)
	}
}
