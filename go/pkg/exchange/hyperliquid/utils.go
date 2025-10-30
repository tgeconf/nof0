package hyperliquid

import (
	"context"
	"fmt"
	"strings"
)

// GetAssetIndex resolves the exchange asset index for the given coin.
func (c *Client) GetAssetIndex(ctx context.Context, coin string) (int, error) {
	key := canonicalAssetKey(coin)
	if key == "" {
		return 0, fmt.Errorf("hyperliquid: empty coin symbol")
	}

	if idx, ok := c.cachedAssetIndex(key); ok {
		return idx, nil
	}
	if err := c.refreshAssetDirectory(ctx); err != nil {
		return 0, err
	}
	if idx, ok := c.cachedAssetIndex(key); ok {
		return idx, nil
	}
	return 0, fmt.Errorf("hyperliquid: asset %s not found", coin)
}

// GetAssetInfo returns cached asset metadata.
func (c *Client) GetAssetInfo(ctx context.Context, coin string) (*AssetInfo, error) {
	key := canonicalAssetKey(coin)
	if key == "" {
		return nil, fmt.Errorf("hyperliquid: empty coin symbol")
	}
	if info, ok := c.cachedAssetInfo(key); ok {
		return &info, nil
	}
	if err := c.refreshAssetDirectory(ctx); err != nil {
		return nil, err
	}
	if info, ok := c.cachedAssetInfo(key); ok {
		return &info, nil
	}
	return nil, fmt.Errorf("hyperliquid: asset info %s not found", coin)
}

func (c *Client) cachedAssetIndex(key string) (int, bool) {
	c.assetMu.RLock()
	defer c.assetMu.RUnlock()
	idx, ok := c.assetIndex[key]
	return idx, ok
}

func (c *Client) cachedAssetInfo(key string) (AssetInfo, bool) {
	c.assetMu.RLock()
	defer c.assetMu.RUnlock()
	info, ok := c.assetInfo[key]
	return info, ok
}

func (c *Client) refreshAssetDirectory(ctx context.Context) error {
	var resp MetaAndAssetCtxsResponse
	if err := c.doInfoRequest(ctx, InfoRequest{Type: "metaAndAssetCtxs"}, &resp); err != nil {
		return err
	}
	if len(resp.Universe) == 0 {
		return fmt.Errorf("hyperliquid: metaAndAssetCtxs response contained no assets")
	}

	index := make(map[string]int, len(resp.Universe))
	info := make(map[string]AssetInfo, len(resp.Universe))
	for idx, entry := range resp.Universe {
		key := canonicalAssetKey(entry.Name)
		if key == "" {
			continue
		}
		var ctx AssetCtx
		if idx < len(resp.AssetCtxs) {
			ctx = resp.AssetCtxs[idx]
		}
		info[key] = AssetInfo{
			Name:         entry.Name,
			SzDecimals:   entry.SzDecimals,
			MaxLeverage:  entry.MaxLeverage,
			MarginTable:  entry.MarginTable,
			OnlyIsolated: entry.OnlyIsolated,
			IsDelisted:   entry.IsDelisted,
			Index:        idx,
			MarkPx:       ctx.MarkPx,
			MidPx:        ctx.MidPx,
			OraclePx:     ctx.OraclePx,
			ImpactPxs:    append([]string(nil), ctx.ImpactPxs...),
		}
		index[key] = idx
	}

	c.assetMu.Lock()
	c.assetIndex = index
	c.assetInfo = info
	c.assetMu.Unlock()
	return nil
}

func canonicalAssetKey(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}
