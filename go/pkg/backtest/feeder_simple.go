package backtest

import (
	"context"

	"nof0-api/pkg/market"
)

// PriceFeeder emits snapshots built from a static price series.
type PriceFeeder struct {
	symbol string
	prices []float64
	idx    int
}

func NewPriceFeeder(symbol string, prices []float64) *PriceFeeder {
	return &PriceFeeder{symbol: symbol, prices: prices}
}

func (f *PriceFeeder) Next(ctx context.Context, symbol string) (*market.Snapshot, bool, error) {
	if f.idx >= len(f.prices) {
		return nil, false, nil
	}
	px := f.prices[f.idx]
	f.idx++
	var oneHour, fourHour float64 // fractional change ratios (0.01 == +1%)
	if f.idx >= 2 {
		prev := f.prices[f.idx-2]
		if prev != 0 {
			oneHour = (px - prev) / prev
			fourHour = oneHour // simple mirror for demo
		}
	}
	snap := &market.Snapshot{
		Symbol: symbol,
		Price:  market.PriceInfo{Last: px},
		Change: market.ChangeInfo{OneHour: oneHour, FourHour: fourHour},
		Indicators: market.IndicatorInfo{
			EMA: map[string]float64{"EMA2": px},
			RSI: map[string]float64{"RSI2": 50},
		},
	}
	return snap, true, nil
}
