package sim

import (
	"context"
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

	err = p.ClosePosition(ctx, "BTC")
	assert.NoError(t, err, "ClosePosition should not error")

	pos, err = p.GetPositions(ctx)
	assert.NoError(t, err, "GetPositions should not error")
	assert.Len(t, pos, 1, "should still have 1 position entry")
	assert.Equal(t, "0", pos[0].Szi, "closed position size should be 0")
}
