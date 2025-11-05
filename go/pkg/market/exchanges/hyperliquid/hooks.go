package hyperliquid

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	marketpkg "nof0-api/pkg/market"
)

// persistSnapshot writes the given snapshot to the persistence hook (if configured)
// and logs errors without blocking the data path.
func (p *Provider) persistSnapshot(ctx context.Context, symbol string, snap *marketpkg.Snapshot) {
	if p.persistence == nil || snap == nil {
		return
	}
	if err := p.persistence.RecordSnapshot(ctx, p.providerName(), snap); err != nil {
		logx.WithContext(ctx).Errorf("hyperliquid: persist snapshot provider=%s symbol=%s err=%v", p.providerName(), symbol, err)
	}
}

// persistAssets writes asset metadata via persistence hook when last refresh exceeds refreshInterval.
func (p *Provider) persistAssets(ctx context.Context, assets []marketpkg.Asset, refreshInterval time.Duration, lastRefreshed map[string]time.Time) {
	if p.persistence == nil || len(assets) == 0 {
		return
	}
	if refreshInterval > 0 {
		if ts := lastRefreshed[p.providerName()]; !ts.IsZero() && time.Since(ts) < refreshInterval {
			return
		}
	}
	if err := p.persistence.UpsertAssets(ctx, p.providerName(), assets); err != nil {
		logx.WithContext(ctx).Errorf("hyperliquid: persist assets provider=%s err=%v", p.providerName(), err)
		return
	}
	lastRefreshed[p.providerName()] = time.Now()
}
