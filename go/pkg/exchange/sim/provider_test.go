package sim

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"nof0-api/pkg/exchange"
)

func TestSimProvider_BasicFlow(t *testing.T) {
	p := New()
	ctx := context.Background()

	asset, err := p.GetAssetIndex(ctx, "BTC")
	assert.NoError(t, err, "GetAssetIndex should not error")

	// Set leverage, then place an order
	err = p.UpdateLeverage(ctx, asset, true, 10)
	assert.NoError(t, err, "UpdateLeverage should not error")

	_, err = p.PlaceOrder(ctx, exchange.Order{Asset: asset, IsBuy: true, LimitPx: "50000", Sz: "0.01"})
	assert.NoError(t, err, "PlaceOrder should not error")

	pos, err := p.GetPositions(ctx)
	assert.NoError(t, err, "GetPositions should not error")
	assert.Len(t, pos, 1, "should have 1 position")
	assert.Equal(t, "0.01", pos[0].Szi, "position size should be 0.01")

	resp, err := p.ClosePosition(ctx, "BTC")
	assert.NoError(t, err, "ClosePosition should not error")
	assert.NotNil(t, resp, "ClosePosition response should not be nil")

	pos, err = p.GetPositions(ctx)
	assert.NoError(t, err, "GetPositions should not error")
	assert.Len(t, pos, 0, "position list should be empty after close")
}

func TestSimProvider_IOCMarket(t *testing.T) {
	p := New()
	ctx := context.Background()

	err := p.SetMarkPrice(ctx, "ETH", 3889.65)
	assert.NoError(t, err, "SetMarkPrice should not error")

	resp, err := p.IOCMarket(ctx, "ETH", true, 0.25, 0.01, false)
	assert.NoError(t, err, "IOCMarket should not error")
	assert.NotNil(t, resp, "IOCMarket response should not be nil")

	positions, err := p.GetPositions(ctx)
	assert.NoError(t, err, "GetPositions should not error after IOCMarket")
	assert.Len(t, positions, 1, "should have one position recorded")
	assert.Equal(t, "ETH", positions[0].Coin, "position coin should be canonical symbol")
	assert.InDelta(t, 0.25, parseDecimal(t, positions[0].Szi), 1e-9, "position size should match order quantity")
	assert.InDelta(t, 3889.65*(1+0.01), parseDecimal(t, resp.Response.Data.Statuses[0].Filled.AvgPx), 1e-6, "filled price should apply slippage to mark")

	// Exercise reduce-only flow.
	_, err = p.IOCMarket(ctx, "ETH", false, 0.25, 0.01, true)
	assert.NoError(t, err, "reduce-only IOCMarket should not error")
	positions, err = p.GetPositions(ctx)
	assert.NoError(t, err, "GetPositions should succeed")
	assert.Len(t, positions, 0, "reduce-only order should flat position")
}

func TestSimProvider_FormattersAndCancelAll(t *testing.T) {
	p := New()
	ctx := context.Background()

	price, err := p.FormatPrice(ctx, "BTC", 12345.6789)
	assert.NoError(t, err, "FormatPrice should not error for positive prices")
	assert.Equal(t, "12345.6789", price, "FormatPrice should trim trailing zeros")

	size, err := p.FormatSize(ctx, "BTC", 0.125)
	assert.NoError(t, err, "FormatSize should not error for positive sizes")
	assert.Equal(t, "0.125", size, "FormatSize should trim trailing zeros")

	assert.NoError(t, p.CancelAllBySymbol(ctx, "BTC"), "CancelAllBySymbol should be a no-op")

	_, err = p.FormatPrice(ctx, "BTC", 0)
	assert.Error(t, err, "FormatPrice should error for non-positive prices")
	_, err = p.FormatSize(ctx, "BTC", -1)
	assert.Error(t, err, "FormatSize should error for non-positive sizes")
}

func TestSimProvider_ReduceOnlyClamp(t *testing.T) {
	p := New()
	ctx := context.Background()

	asset, err := p.GetAssetIndex(ctx, "BTC")
	assert.NoError(t, err, "GetAssetIndex should not error")

	_, err = p.PlaceOrder(ctx, exchange.Order{Asset: asset, IsBuy: true, LimitPx: "50000", Sz: "1"})
	assert.NoError(t, err, "initial PlaceOrder should not error")

	resp, err := p.IOCMarket(ctx, "BTC", false, 2, 0.01, true)
	assert.NoError(t, err, "reduce-only IOCMarket should not error")
	assert.NotNil(t, resp, "response should not be nil")
	if assert.NotNil(t, resp.Response.Data.Statuses[0].Filled, "filled data should be present") {
		assert.Equal(t, "1", resp.Response.Data.Statuses[0].Filled.TotalSz, "filled size should clamp to existing position")
	}

	positions, err := p.GetPositions(ctx)
	assert.NoError(t, err, "GetPositions should succeed")
	assert.Len(t, positions, 0, "position should be fully closed without flipping direction")
}

func parseDecimal(t *testing.T, s string) float64 {
	t.Helper()
	f, err := strconv.ParseFloat(s, 64)
	assert.NoError(t, err, "string should parse as float")
	return f
}
