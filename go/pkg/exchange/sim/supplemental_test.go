package sim

import (
	"context"
	"testing"

	"nof0-api/pkg/exchange"

	"github.com/stretchr/testify/assert"
)

func TestSimProvider_GetAssetIndex(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Test getting index for new coin
	t.Run("new_coin", func(t *testing.T) {
		index, err := p.GetAssetIndex(ctx, "BTC")
		assert.NoError(t, err)
		assert.Equal(t, 1, index)
	})

	// Test getting index for existing coin
	t.Run("existing_coin", func(t *testing.T) {
		index, err := p.GetAssetIndex(ctx, "BTC")
		assert.NoError(t, err)
		assert.Equal(t, 1, index) // Should return same index
	})

	// Test getting index for different coin
	t.Run("different_coin", func(t *testing.T) {
		index, err := p.GetAssetIndex(ctx, "ETH")
		assert.NoError(t, err)
		assert.Equal(t, 2, index) // Should get new index
	})

	// Test canonicalization
	t.Run("canonicalization", func(t *testing.T) {
		index1, err := p.GetAssetIndex(ctx, "btc")
		assert.NoError(t, err)
		index2, err := p.GetAssetIndex(ctx, "BTC")
		assert.NoError(t, err)
		assert.Equal(t, index1, index2) // Should be same index
	})
}

func TestSimProvider_PlaceOrder(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Setup asset index
	asset, err := p.GetAssetIndex(ctx, "BTC")
	assert.NoError(t, err)

	// Test placing buy order
	t.Run("buy_order", func(t *testing.T) {
		resp, err := p.PlaceOrder(ctx, exchange.Order{
			Asset:   asset,
			IsBuy:   true,
			LimitPx: "50000",
			Sz:      "0.01",
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)

		// Verify position was created
		positions, err := p.GetPositions(ctx)
		assert.NoError(t, err)
		assert.Len(t, positions, 1)
		assert.Equal(t, "BTC", positions[0].Coin)
		assert.Equal(t, "0.01", positions[0].Szi)
		assert.Equal(t, "50000", *positions[0].EntryPx)
	})

	// Test placing sell order
	t.Run("sell_order", func(t *testing.T) {
		resp, err := p.PlaceOrder(ctx, exchange.Order{
			Asset:   asset,
			IsBuy:   false,
			LimitPx: "45000",
			Sz:      "0.005",
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify position was updated
		positions, err := p.GetPositions(ctx)
		assert.NoError(t, err)
		assert.Len(t, positions, 1)
		assert.Equal(t, "BTC", positions[0].Coin)
		assert.Equal(t, "0.005", positions[0].Szi) // 0.01 - 0.005 = 0.005
	})

	// Test placing order with unknown asset
	t.Run("unknown_asset", func(t *testing.T) {
		resp, err := p.PlaceOrder(ctx, exchange.Order{
			Asset:   999, // Unknown asset
			IsBuy:   true,
			LimitPx: "50000",
			Sz:      "0.01",
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "unknown asset index")
	})
}

func TestSimProvider_CancelOrder(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Test that cancel order always succeeds (no-op implementation)
	t.Run("cancel_order", func(t *testing.T) {
		err := p.CancelOrder(ctx, 1, 12345)
		assert.NoError(t, err)
	})
}

func TestSimProvider_GetOpenOrders(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Test that get open orders always returns empty (no-op implementation)
	t.Run("get_open_orders", func(t *testing.T) {
		orders, err := p.GetOpenOrders(ctx)
		assert.NoError(t, err)
		assert.Nil(t, orders)
	})
}

func TestSimProvider_GetPositions(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Test getting positions when empty
	t.Run("empty_positions", func(t *testing.T) {
		positions, err := p.GetPositions(ctx)
		assert.NoError(t, err)
		assert.Len(t, positions, 0)
	})

	// Test getting positions after placing orders
	t.Run("with_positions", func(t *testing.T) {
		asset, err := p.GetAssetIndex(ctx, "BTC")
		assert.NoError(t, err)

		_, err = p.PlaceOrder(ctx, exchange.Order{
			Asset:   asset,
			IsBuy:   true,
			LimitPx: "50000",
			Sz:      "0.01",
		})
		assert.NoError(t, err)

		positions, err := p.GetPositions(ctx)
		assert.NoError(t, err)
		assert.Len(t, positions, 1)
		assert.Equal(t, "BTC", positions[0].Coin)
	})
}

func TestSimProvider_ClosePosition(t *testing.T) {
	p := New()
	ctx := context.Background()

	asset, err := p.GetAssetIndex(ctx, "BTC")
	assert.NoError(t, err)

	// Place an order first
	_, err = p.PlaceOrder(ctx, exchange.Order{
		Asset:   asset,
		IsBuy:   true,
		LimitPx: "50000",
		Sz:      "0.01",
	})
	assert.NoError(t, err)

	// Test closing position
	t.Run("close_position", func(t *testing.T) {
		err := p.ClosePosition(ctx, "BTC")
		assert.NoError(t, err)

		positions, err := p.GetPositions(ctx)
		assert.NoError(t, err)
		assert.Len(t, positions, 1)
		assert.Equal(t, "0", positions[0].Szi)
	})

	// Test closing position with canonicalization
	t.Run("close_position_canonical", func(t *testing.T) {
		err := p.ClosePosition(ctx, "btc")
		assert.NoError(t, err)

		positions, err := p.GetPositions(ctx)
		assert.NoError(t, err)
		assert.Len(t, positions, 1)
		assert.Equal(t, "0", positions[0].Szi)
	})
}

func TestSimProvider_UpdateLeverage(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Test updating leverage
	t.Run("update_leverage", func(t *testing.T) {
		err := p.UpdateLeverage(ctx, 1, true, 10)
		assert.NoError(t, err)

		err = p.UpdateLeverage(ctx, 1, false, 5)
		assert.NoError(t, err)

		// Verify leverage was set by checking that no error occurs
		// (the leverage is used internally when placing orders)
		_, err = p.PlaceOrder(ctx, exchange.Order{
			Asset:   1,
			IsBuy:   true,
			LimitPx: "50000",
			Sz:      "0.01",
		})
		assert.NoError(t, err)
	})
}

func TestSimProvider_GetAccountState(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Test getting account state when empty
	t.Run("empty_account_state", func(t *testing.T) {
		state, err := p.GetAccountState(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, state)
		assert.Equal(t, "100000", state.MarginSummary.AccountValue)
		assert.Equal(t, "0", state.MarginSummary.TotalMarginUsed)
		assert.Len(t, state.AssetPositions, 0)
	})

	// Test getting account state with positions
	t.Run("account_state_with_positions", func(t *testing.T) {
		asset, err := p.GetAssetIndex(ctx, "BTC")
		assert.NoError(t, err)

		_, err = p.PlaceOrder(ctx, exchange.Order{
			Asset:   asset,
			IsBuy:   true,
			LimitPx: "50000",
			Sz:      "0.01",
		})
		assert.NoError(t, err)

		state, err := p.GetAccountState(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, state)
		assert.Len(t, state.AssetPositions, 1)
		assert.Equal(t, "BTC", state.AssetPositions[0].Coin)
	})
}

func TestSimProvider_GetAccountValue(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Test getting account value
	t.Run("get_account_value", func(t *testing.T) {
		value, err := p.GetAccountValue(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 100000.0, value)
	})
}

func TestSimProvider_ConcurrentAccess(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Test that the provider is safe for concurrent access
	t.Run("concurrent_access", func(t *testing.T) {
		asset, err := p.GetAssetIndex(ctx, "BTC")
		assert.NoError(t, err)

		// Launch multiple goroutines to test concurrent access
		errChan := make(chan error, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_, err := p.PlaceOrder(ctx, exchange.Order{
					Asset:   asset,
					IsBuy:   true,
					LimitPx: "50000",
					Sz:      "0.001",
				})
				errChan <- err
			}()
		}

		// Check that all operations succeeded
		for i := 0; i < 10; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}

		// Verify final position
		positions, err := p.GetPositions(ctx)
		assert.NoError(t, err)
		assert.Len(t, positions, 1)
		assert.Equal(t, "0.01", positions[0].Szi) // 10 * 0.001 = 0.01
	})
}

func TestSimProvider_MultipleCoins(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Test managing positions in multiple coins
	t.Run("multiple_coins", func(t *testing.T) {
		// Get asset indices for different coins
		btcAsset, err := p.GetAssetIndex(ctx, "BTC")
		assert.NoError(t, err)

		ethAsset, err := p.GetAssetIndex(ctx, "ETH")
		assert.NoError(t, err)

		// Place orders in different coins
		_, err = p.PlaceOrder(ctx, exchange.Order{
			Asset:   btcAsset,
			IsBuy:   true,
			LimitPx: "50000",
			Sz:      "0.01",
		})
		assert.NoError(t, err)

		_, err = p.PlaceOrder(ctx, exchange.Order{
			Asset:   ethAsset,
			IsBuy:   true,
			LimitPx: "3000",
			Sz:      "1.0",
		})
		assert.NoError(t, err)

		// Verify both positions exist
		positions, err := p.GetPositions(ctx)
		assert.NoError(t, err)
		assert.Len(t, positions, 2)

		// Find BTC position
		var btcPos, ethPos exchange.Position
		for _, pos := range positions {
			if pos.Coin == "BTC" {
				btcPos = pos
			} else if pos.Coin == "ETH" {
				ethPos = pos
			}
		}

		assert.Equal(t, "BTC", btcPos.Coin)
		assert.Equal(t, "0.01", btcPos.Szi)
		assert.Equal(t, "ETH", ethPos.Coin)
		assert.Equal(t, "1.0", ethPos.Szi)

		// Close BTC position
		err = p.ClosePosition(ctx, "BTC")
		assert.NoError(t, err)

		// Verify only ETH position remains
		positions, err = p.GetPositions(ctx)
		assert.NoError(t, err)
		assert.Len(t, positions, 2) // Still 2 entries, but BTC position is closed

		for _, pos := range positions {
			if pos.Coin == "BTC" {
				assert.Equal(t, "0", pos.Szi)
			} else if pos.Coin == "ETH" {
				assert.Equal(t, "1.0", pos.Szi)
			}
		}
	})
}

func TestSimProvider_Init(t *testing.T) {
	// Test that init function registers the provider
	t.Run("provider_registration", func(t *testing.T) {
		// This test verifies that the init function properly registers the provider
		// by attempting to create a provider through the exchange package
		provider, err := exchange.GetProvider("sim", &exchange.ProviderConfig{})
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})
}
