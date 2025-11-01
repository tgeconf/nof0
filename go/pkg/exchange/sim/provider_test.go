package sim

import (
	"context"
	"testing"

	"nof0-api/pkg/exchange"
)

func TestSimProvider_BasicFlow(t *testing.T) {
	p := New()
	ctx := context.Background()

	asset, err := p.GetAssetIndex(ctx, "BTC")
	if err != nil {
		t.Fatalf("GetAssetIndex: %v", err)
	}

	// Set leverage, then place an order
	if err := p.UpdateLeverage(ctx, asset, true, 10); err != nil {
		t.Fatalf("UpdateLeverage: %v", err)
	}

	_, err = p.PlaceOrder(ctx, exchange.Order{Asset: asset, IsBuy: true, LimitPx: "50000", Sz: "0.01"})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	pos, err := p.GetPositions(ctx)
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(pos) != 1 {
		t.Fatalf("expected 1 position, got %d", len(pos))
	}
	if pos[0].Szi != "0.01" {
		t.Fatalf("unexpected size %s", pos[0].Szi)
	}

	if err := p.ClosePosition(ctx, "BTC"); err != nil {
		t.Fatalf("ClosePosition: %v", err)
	}
	pos, _ = p.GetPositions(ctx)
	if pos[0].Szi != "0" {
		t.Fatalf("expected closed size 0, got %s", pos[0].Szi)
	}
}
