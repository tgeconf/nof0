package backtest

import (
	"context"
	"strconv"

	"nof0-api/pkg/exchange"
	"nof0-api/pkg/market"
)

// ThresholdStrategy buys when price increases above threshold% vs previous, sells when decreases below -threshold%.
type ThresholdStrategy struct {
	AssetID    int
	LastPrice  float64
	ThresholdP float64 // percent
	LotSz      string  // order size string (e.g., "0.01")
}

func (s *ThresholdStrategy) Decide(ctx context.Context, snap *market.Snapshot) ([]exchange.Order, error) {
	px := snap.Price.Last
	if s.LastPrice == 0 {
		s.LastPrice = px
		return nil, nil
	}
	pct := 0.0
	if s.LastPrice != 0 {
		pct = (px - s.LastPrice) / s.LastPrice * 100
	}
	s.LastPrice = px
	if pct >= s.ThresholdP {
		return []exchange.Order{{Asset: s.AssetID, IsBuy: true, LimitPx: strconv.FormatFloat(px, 'f', -1, 64), Sz: s.LotSz}}, nil
	}
	if pct <= -s.ThresholdP {
		return []exchange.Order{{Asset: s.AssetID, IsBuy: false, LimitPx: strconv.FormatFloat(px, 'f', -1, 64), Sz: s.LotSz}}, nil
	}
	return nil, nil
}
