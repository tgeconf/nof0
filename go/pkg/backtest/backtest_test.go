package backtest

import (
	"context"
	"math"
	"testing"

	simex "nof0-api/pkg/exchange/sim"

	"github.com/stretchr/testify/assert"
)

func TestBacktest_ThresholdWithSim(t *testing.T) {
	ctx := context.Background()
	exch := simex.New()
	assetID, err := exch.GetAssetIndex(ctx, "BTC")
	assert.NoError(t, err, "GetAssetIndex should not error")

	feeder := NewPriceFeeder("BTC", []float64{100, 101, 103, 102, 99, 100})
	strat := &ThresholdStrategy{AssetID: assetID, ThresholdP: 1.0, LotSz: "0.01"}

	e := &Engine{Feeder: feeder, Strategy: strat, Exch: exch, Symbol: "BTC", FeeBps: 2.0, SlippageBps: 1.0, InitialEquity: 100000}
	res, err := e.Run(ctx)
	assert.NoError(t, err, "Engine.Run should not error")
	assert.NotNil(t, res, "result should not be nil")

	assert.Equal(t, 6, res.Steps, "should run for 6 steps")
	assert.Greater(t, res.OrdersSent, 0, "should send some orders")
	assert.Len(t, res.EquityCurve, res.Steps, "equity curve length should match steps")

	// MaxDDPct, Sharpe are scenario-specific but should be finite numbers
	assert.False(t, res.MaxDDPct < 0 || math.IsNaN(res.MaxDDPct), "max drawdown should be non-negative and not NaN")
	assert.False(t, math.IsNaN(res.Sharpe), "sharpe ratio should not be NaN")
}
