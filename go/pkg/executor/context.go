package executor

import (
	"context"
	"fmt"

	market "nof0-api/pkg/market"
)

// BuildContext constructs a Context by enriching base fields with market snapshots and optional OI/performance data.
func BuildContext(ctx context.Context, base *Context, mktProvider market.Provider) (*Context, error) {
	if base == nil {
		return nil, fmt.Errorf("executor: base context is required")
	}
	out := *base // shallow copy
	if out.MarketDataMap == nil {
		out.MarketDataMap = make(map[string]*market.Snapshot, len(base.CandidateCoins)+len(base.Positions))
	}
	// Ensure existing positions' symbols are included in candidates
	syms := make(map[string]struct{})
	for _, c := range base.CandidateCoins {
		syms[c.Symbol] = struct{}{}
	}
	for _, p := range base.Positions {
		syms[p.Symbol] = struct{}{}
	}
	// Fetch snapshots if provider supplied
	if mktProvider != nil {
		for sym := range syms {
			if _, ok := out.MarketDataMap[sym]; ok {
				continue
			}
			snap, err := mktProvider.Snapshot(ctx, sym)
			if err != nil {
				// tolerate individual symbol failure; log/skip in future
				continue
			}
			out.MarketDataMap[sym] = snap
		}
	}
	return &out, nil
}
