package hyperliquid

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"nof0-api/pkg/exchange"
)

// FormatSize rounds a float quantity to the coin's szDecimals and returns a
// normalized decimal string (no scientific notation).
func (c *Client) FormatSize(ctx context.Context, coin string, qty float64) (string, error) {
	info, err := c.GetAssetInfo(ctx, coin)
	if err != nil {
		return "", err
	}
	if qty < 0 {
		qty = -qty
	}
	// round half up to szDecimals places
	pow := math.Pow(10, float64(info.SzDecimals))
	v := math.Round(qty*pow) / pow
	s := strconv.FormatFloat(v, 'f', info.SzDecimals, 64)
	// trim trailing zeros
	for len(s) > 0 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	if s == "" { // in case qty was 0 exactly
		s = "0"
	}
	return s, nil
}

// IOCMarket places an IOC limit order using a small slippage on the mid/mark
// price to simulate market execution.
// - slippage is a fraction, e.g. 0.01 = 1%.
func (c *Client) IOCMarket(ctx context.Context, coin string, isBuy bool, qty float64, slippage float64, reduceOnly bool) (*exchange.OrderResponse, error) {
	if slippage <= 0 {
		if c.defaultSlippage > 0 {
			slippage = c.defaultSlippage
		} else {
			slippage = 0.01
		}
	}
	idx, err := c.GetAssetIndex(ctx, coin)
	if err != nil {
		return nil, err
	}
	info, err := c.GetAssetInfo(ctx, coin)
	if err != nil {
		return nil, err
	}
	// pick a base price: prefer MidPx then MarkPx then OraclePx
	base := firstNonEmpty(info.MidPx, info.MarkPx, info.OraclePx)
	if base == "" {
		return nil, fmt.Errorf("hyperliquid: missing reference price for %s", coin)
	}
	px, err := strconv.ParseFloat(base, 64)
	if err != nil || !(px > 0) {
		return nil, fmt.Errorf("hyperliquid: invalid reference price %q for %s", base, coin)
	}
	if isBuy {
		px = px * (1 + slippage)
	} else {
		px = px * (1 - slippage)
	}
	sigs := c.priceSigFigs
	if sigs <= 0 {
		sigs = 5
	}
	price := RoundPriceToSigFigs(px, sigs)
	size, err := c.FormatSize(ctx, coin, qty)
	if err != nil {
		return nil, err
	}
	order := exchange.Order{
		Asset:      idx,
		IsBuy:      isBuy,
		LimitPx:    price,
		Sz:         size,
		ReduceOnly: reduceOnly,
		OrderType:  exchange.OrderType{Limit: &exchange.LimitOrderType{TIF: "Ioc"}},
	}
	resp, err := c.PlaceOrder(ctx, order)
	if err != nil {
		return nil, err
	}
	// Check for error status in response
	if resp.Status == "err" {
		// Try to extract detailed error message
		if len(resp.Response.Data.Statuses) > 0 && resp.Response.Data.Statuses[0].Error != "" {
			return resp, fmt.Errorf("hyperliquid: order rejected: %s", resp.Response.Data.Statuses[0].Error)
		}
		// Check if error message is in string format
		if resp.ErrorMessage != "" {
			return resp, fmt.Errorf("hyperliquid: order rejected: %s", resp.ErrorMessage)
		}
		// Return generic error if no details available
		return resp, fmt.Errorf("hyperliquid: order rejected with status 'err' (no error details provided)")
	}
	return resp, nil
}

// PlaceTriggerReduceOnly creates a reduce-only trigger order (TP/SL style).
// tpsl can be "tp" or "sl" for venues that accept the semantic.
func (c *Client) PlaceTriggerReduceOnly(ctx context.Context, coin string, isBuy bool, qty float64, triggerPx float64, tpsl string) error {
	if !(triggerPx > 0) {
		return fmt.Errorf("hyperliquid: trigger price must be positive")
	}
	idx, err := c.GetAssetIndex(ctx, coin)
	if err != nil {
		return err
	}
	size, err := c.FormatSize(ctx, coin, qty)
	if err != nil {
		return err
	}
	tp := RoundPriceToSigFigs(triggerPx, 5)
	// For trigger orders with isMarket, use an aggressive limit price as safety
	limitPx := aggressiveLimitPrice(isBuy)
	ord := exchange.Order{
		Asset:      idx,
		IsBuy:      isBuy,
		LimitPx:    limitPx,
		Sz:         size,
		ReduceOnly: true,
		TriggerPx:  tp,
		OrderType:  exchange.OrderType{Trigger: &exchange.TriggerOrderType{IsMarket: true, Tpsl: tpsl}},
	}
	_, err = c.PlaceOrder(ctx, ord)
	return err
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if len(v) > 0 {
			return v
		}
	}
	return ""
}
