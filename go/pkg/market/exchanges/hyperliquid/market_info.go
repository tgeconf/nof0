package hyperliquid

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// MarketInfo aggregates key market metrics returned by metaAndAssetCtxs.
type MarketInfo struct {
	Symbol       string  // Canonical Hyperliquid symbol
	MarkPrice    float64 // Mark price
	MidPrice     float64 // Mid price
	FundingRate  float64 // Funding rate (decimal, not percentage)
	OpenInterest float64 // Current open interest
	DayVolume    float64 // 24h base volume
}

// GetCurrentPrice returns the current mid price for the given symbol.
func (c *Client) GetCurrentPrice(ctx context.Context, symbol string) (float64, error) {
	canonical, err := c.canonicalSymbolFor(ctx, symbol)
	if err != nil {
		return 0, err
	}

	var response AllMidsResponse
	if err := c.doRequest(ctx, InfoRequest{Type: "allMids"}, &response); err != nil {
		return 0, err
	}
	val, ok := response[canonical]
	if !ok {
		// Attempt relaxed lookup by upper-casing as a fallback.
		val, ok = response[strings.ToUpper(canonical)]
	}
	if !ok {
		return 0, fmt.Errorf("hyperliquid: price for %s not found", canonical)
	}
	price, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("hyperliquid: parse price %q: %w", val, err)
	}
	return price, nil
}

// GetMarketInfo retrieves funding, open interest and related metrics.
func (c *Client) GetMarketInfo(ctx context.Context, symbol string) (*MarketInfo, error) {
	if err := c.refreshSymbolDirectory(ctx); err != nil {
		return nil, err
	}

	canonical, ctxData, ok := c.assetCtxFromCache(symbol)
	if !ok {
		return nil, ErrSymbolNotFound
	}
	mark, err := parseFloat(ctxData.MarkPx)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid: parse mark price: %w", err)
	}
	if math.IsNaN(mark) {
		return nil, fmt.Errorf("hyperliquid: missing mark price for %s", canonical)
	}
	mid, err := parseFloat(ctxData.MidPx)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid: parse mid price: %w", err)
	}
	if math.IsNaN(mid) {
		mid = mark
	}
	funding, err := parseFloat(ctxData.Funding)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid: parse funding: %w", err)
	}
	oi, err := parseFloat(ctxData.OpenInterest)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid: parse open interest: %w", err)
	}
	if math.IsNaN(oi) {
		oi = 0
	}
	dayVolume, err := parseFloat(ctxData.DayBaseVlm)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid: parse dayBase volume: %w", err)
	}
	if math.IsNaN(dayVolume) {
		dayVolume, err = parseFloat(ctxData.DayNtlVlm)
		if err != nil {
			return nil, fmt.Errorf("hyperliquid: parse dayNotional volume: %w", err)
		}
		if math.IsNaN(dayVolume) {
			dayVolume = 0
		}
	}

	return &MarketInfo{
		Symbol:       canonical,
		MarkPrice:    mark,
		MidPrice:     mid,
		FundingRate:  funding,
		OpenInterest: oi,
		DayVolume:    dayVolume,
	}, nil
}

func (c *Client) resolveSymbol(ctx context.Context, symbol string) (string, error) {
	return c.canonicalSymbolFor(ctx, symbol)
}

func parseFloat(val string) (float64, error) {
	if val == "" {
		return math.NaN(), nil
	}
	return strconv.ParseFloat(val, 64)
}
