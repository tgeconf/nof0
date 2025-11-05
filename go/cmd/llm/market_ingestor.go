package main

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	marketpkg "nof0-api/pkg/market"
)

// marketIngestor periodically fetches market snapshots and asset metadata so that
// persistence hooks can mirror them into Postgres/Redis. It operates on the set
// of filtered market providers already constrained to the allowed symbol list.
type marketIngestor struct {
	providers      map[string]marketpkg.Provider
	orderedNames   []string
	symbols        []string
	interval       time.Duration
	assetRefresh   time.Duration
	delayPerSymbol time.Duration
	assetsAt       map[string]time.Time
}

const (
	defaultSnapshotTimeout = 8 * time.Second
	defaultAssetsTimeout   = 20 * time.Second
)

func newMarketIngestor(providers map[string]marketpkg.Provider, symbols []string, interval, assetRefresh, delay time.Duration) *marketIngestor {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if assetRefresh < 0 {
		assetRefresh = 0
	}
	if delay < 0 {
		delay = 0
	}
	ordered := make([]string, 0, len(providers))
	for name := range providers {
		if providers[name] == nil {
			continue
		}
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)
	uniqueSymbols := make([]string, 0, len(symbols))
	seen := make(map[string]struct{}, len(symbols))
	for _, sym := range symbols {
		sym = strings.ToUpper(strings.TrimSpace(sym))
		if sym == "" {
			continue
		}
		if _, ok := seen[sym]; ok {
			continue
		}
		seen[sym] = struct{}{}
		uniqueSymbols = append(uniqueSymbols, sym)
	}
	return &marketIngestor{
		providers:      providers,
		orderedNames:   ordered,
		symbols:        uniqueSymbols,
		interval:       interval,
		assetRefresh:   assetRefresh,
		delayPerSymbol: delay,
		assetsAt:       make(map[string]time.Time, len(providers)),
	}
}

// run starts the ingestion loop and blocks until the context is cancelled.
func (m *marketIngestor) run(ctx context.Context) {
	if m == nil || len(m.orderedNames) == 0 || len(m.symbols) == 0 {
		return
	}
	m.refreshAssets(ctx, true)
	m.refreshSnapshots(ctx)
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.refreshAssets(ctx, false)
			m.refreshSnapshots(ctx)
		}
	}
}

func (m *marketIngestor) refreshAssets(ctx context.Context, force bool) {
	if m.assetRefresh == 0 && !force {
		return
	}
	now := time.Now()
	for _, name := range m.orderedNames {
		if !force && m.assetRefresh > 0 {
			if last, ok := m.assetsAt[name]; ok && now.Sub(last) < m.assetRefresh {
				continue
			}
		}
		prov := m.providers[name]
		if prov == nil {
			continue
		}
		reqCtx, cancel := context.WithTimeout(ctx, defaultAssetsTimeout)
		_, err := prov.ListAssets(reqCtx)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logx.WithContext(ctx).Errorf("market ingest: list assets provider=%s err=%v", name, err)
			continue
		}
		m.assetsAt[name] = time.Now()
	}
}

func (m *marketIngestor) refreshSnapshots(ctx context.Context) {
	for _, name := range m.orderedNames {
		prov := m.providers[name]
		if prov == nil {
			continue
		}
		for _, symbol := range m.symbols {
			if ctx.Err() != nil {
				return
			}
			reqCtx, cancel := context.WithTimeout(ctx, defaultSnapshotTimeout)
			if _, err := prov.Snapshot(reqCtx, symbol); err != nil && reqCtx.Err() == nil {
				logx.WithContext(ctx).Errorf("market ingest: snapshot provider=%s symbol=%s err=%v", name, symbol, err)
			}
			cancel()
			if m.delayPerSymbol > 0 {
				if !sleepWithContext(ctx, m.delayPerSymbol) {
					return
				}
			}
		}
	}
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
