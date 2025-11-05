package hyperliquid

import (
	"context"
	"testing"

	"nof0-api/pkg/exchange"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of the Client interface
type MockClient struct {
	mock.Mock
}

func (m *MockClient) PlaceOrder(ctx context.Context, order exchange.Order) (*exchange.OrderResponse, error) {
	args := m.Called(ctx, order)
	return args.Get(0).(*exchange.OrderResponse), args.Error(1)
}

func (m *MockClient) CancelOrder(ctx context.Context, asset int, oid int64) error {
	args := m.Called(ctx, asset, oid)
	return args.Error(0)
}

func (m *MockClient) GetOpenOrders(ctx context.Context) ([]exchange.OrderStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).([]exchange.OrderStatus), args.Error(1)
}

func (m *MockClient) GetPositions(ctx context.Context) ([]exchange.Position, error) {
	args := m.Called(ctx)
	return args.Get(0).([]exchange.Position), args.Error(1)
}

func (m *MockClient) ClosePosition(ctx context.Context, coin string) (*exchange.OrderResponse, error) {
	args := m.Called(ctx, coin)
	var resp *exchange.OrderResponse
	if v := args.Get(0); v != nil {
		resp = v.(*exchange.OrderResponse)
	}
	return resp, args.Error(1)
}

func (m *MockClient) UpdateLeverage(ctx context.Context, asset int, isCross bool, leverage int) error {
	args := m.Called(ctx, asset, isCross, leverage)
	return args.Error(0)
}

func (m *MockClient) GetAccountState(ctx context.Context) (*exchange.AccountState, error) {
	args := m.Called(ctx)
	return args.Get(0).(*exchange.AccountState), args.Error(1)
}

func (m *MockClient) GetAccountValue(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockClient) GetAssetIndex(ctx context.Context, coin string) (int, error) {
	args := m.Called(ctx, coin)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockClient) IOCMarket(ctx context.Context, coin string, isBuy bool, qty float64, slippage float64, reduceOnly bool) (*exchange.OrderResponse, error) {
	args := m.Called(ctx, coin, isBuy, qty, slippage, reduceOnly)
	return args.Get(0).(*exchange.OrderResponse), args.Error(1)
}

func (m *MockClient) PlaceTriggerReduceOnly(ctx context.Context, coin string, isBuy bool, qty float64, triggerPrice float64, tpsl string) error {
	args := m.Called(ctx, coin, isBuy, qty, triggerPrice, tpsl)
	return args.Error(0)
}

func (m *MockClient) CancelAllOrders(ctx context.Context, asset int) error {
	args := m.Called(ctx, asset)
	return args.Error(0)
}

func (m *MockClient) CancelByCloid(ctx context.Context, asset int, cloid string) error {
	args := m.Called(ctx, asset, cloid)
	return args.Error(0)
}

func (m *MockClient) CancelOrdersByCloid(ctx context.Context, cancels []CancelByCloid) error {
	args := m.Called(ctx, cancels)
	return args.Error(0)
}

func (m *MockClient) ModifyOrder(ctx context.Context, req ModifyOrderRequest) (*exchange.OrderResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*exchange.OrderResponse), args.Error(1)
}

func (m *MockClient) ModifyOrders(ctx context.Context, requests []ModifyOrderRequest) (*exchange.OrderResponse, error) {
	args := m.Called(ctx, requests)
	return args.Get(0).(*exchange.OrderResponse), args.Error(1)
}

func (m *MockClient) FormatSize(ctx context.Context, coin string, qty float64) (string, error) {
	args := m.Called(ctx, coin, qty)
	return args.Get(0).(string), args.Error(1)
}

func (m *MockClient) FormatPrice(ctx context.Context, coin string, price float64) (string, error) {
	args := m.Called(ctx, coin, price)
	return args.Get(0).(string), args.Error(1)
}

func TestNewProvider(t *testing.T) {
	// Test successful provider creation
	t.Run("successful_creation", func(t *testing.T) {
		provider, err := NewProvider("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.NotNil(t, provider.client)
	})

	// Test provider creation with invalid private key
	t.Run("invalid_private_key", func(t *testing.T) {
		provider, err := NewProvider("", false)
		assert.Error(t, err)
		assert.Nil(t, provider)
	})

	// Test provider creation with options
	t.Run("with_options", func(t *testing.T) {
		provider, err := NewProvider("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false,
			WithDefaultSlippage(0.02),
			WithPriceSigFigs(4))
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		// Inspect concrete client settings via type assertion
		c, ok := provider.client.(*Client)
		assert.True(t, ok)
		assert.InDelta(t, 0.02, c.defaultSlippage, 1e-12)
		assert.Equal(t, 4, c.priceSigFigs)
	})
}

func TestProviderPlaceOrder(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()
	order := exchange.Order{
		Asset:   1,
		IsBuy:   true,
		LimitPx: "50000",
		Sz:      "0.01",
	}

	// Test successful order placement
	t.Run("successful_order", func(t *testing.T) {
		expectedResponse := &exchange.OrderResponse{Status: "success"}
		mockClient.On("PlaceOrder", ctx, order).Return(expectedResponse, nil)

		resp, err := provider.PlaceOrder(ctx, order)
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, resp)
		mockClient.AssertExpectations(t)
	})

	// Test order placement failure
	t.Run("order_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("PlaceOrder", ctx, order).Return((*exchange.OrderResponse)(nil), assert.AnError)

		resp, err := provider.PlaceOrder(ctx, order)
		assert.Error(t, err)
		assert.Nil(t, resp)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderCancelOrder(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()

	// Test successful cancellation
	t.Run("successful_cancel", func(t *testing.T) {
		mockClient.On("CancelOrder", ctx, 1, int64(12345)).Return(nil)

		err := provider.CancelOrder(ctx, 1, 12345)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	// Test cancellation failure
	t.Run("cancel_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("CancelOrder", ctx, 1, int64(12345)).Return(assert.AnError)

		err := provider.CancelOrder(ctx, 1, 12345)
		assert.Error(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderGetOpenOrders(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()

	// Test successful retrieval
	t.Run("successful_get", func(t *testing.T) {
		expectedOrders := []exchange.OrderStatus{{
			Order:  exchange.OrderInfo{Oid: 12345, Sz: "0.01"},
			Status: "open",
		}}
		mockClient.On("GetOpenOrders", ctx).Return(expectedOrders, nil)

		orders, err := provider.GetOpenOrders(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedOrders, orders)
		mockClient.AssertExpectations(t)
	})

	// Test retrieval failure
	t.Run("get_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("GetOpenOrders", ctx).Return([]exchange.OrderStatus(nil), assert.AnError)

		orders, err := provider.GetOpenOrders(ctx)
		assert.Error(t, err)
		assert.Nil(t, orders)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderGetPositions(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()

	// Test successful retrieval
	t.Run("successful_get", func(t *testing.T) {
		expectedPositions := []exchange.Position{{Coin: "BTC", Szi: "0.01"}}
		mockClient.On("GetPositions", ctx).Return(expectedPositions, nil)

		positions, err := provider.GetPositions(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedPositions, positions)
		mockClient.AssertExpectations(t)
	})

	// Test retrieval failure
	t.Run("get_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("GetPositions", ctx).Return([]exchange.Position(nil), assert.AnError)

		positions, err := provider.GetPositions(ctx)
		assert.Error(t, err)
		assert.Nil(t, positions)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderClosePosition(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()

	// Test successful close
	t.Run("successful_close", func(t *testing.T) {
		mockClient.On("ClosePosition", ctx, "BTC").Return(&exchange.OrderResponse{Status: "ok"}, nil)

		resp, err := provider.ClosePosition(ctx, "BTC")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		mockClient.AssertExpectations(t)
	})

	// Test close failure
	t.Run("close_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("ClosePosition", ctx, "BTC").Return((*exchange.OrderResponse)(nil), assert.AnError)

		resp, err := provider.ClosePosition(ctx, "BTC")
		assert.Error(t, err)
		assert.Nil(t, resp)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderUpdateLeverage(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()

	// Test successful update
	t.Run("successful_update", func(t *testing.T) {
		mockClient.On("UpdateLeverage", ctx, 1, true, 10).Return(nil)

		err := provider.UpdateLeverage(ctx, 1, true, 10)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	// Test update failure
	t.Run("update_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("UpdateLeverage", ctx, 1, true, 10).Return(assert.AnError)

		err := provider.UpdateLeverage(ctx, 1, true, 10)
		assert.Error(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderGetAccountState(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()

	// Test successful retrieval
	t.Run("successful_get", func(t *testing.T) {
		expectedState := &exchange.AccountState{MarginSummary: exchange.MarginSummary{AccountValue: "100", TotalMarginUsed: "0"}}
		mockClient.On("GetAccountState", ctx).Return(expectedState, nil)

		state, err := provider.GetAccountState(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedState, state)
		mockClient.AssertExpectations(t)
	})

	// Test retrieval failure
	t.Run("get_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("GetAccountState", ctx).Return((*exchange.AccountState)(nil), assert.AnError)

		state, err := provider.GetAccountState(ctx)
		assert.Error(t, err)
		assert.Nil(t, state)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderGetAccountValue(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()

	// Test successful retrieval
	t.Run("successful_get", func(t *testing.T) {
		expectedValue := 1000.0
		mockClient.On("GetAccountValue", ctx).Return(expectedValue, nil)

		value, err := provider.GetAccountValue(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, value)
		mockClient.AssertExpectations(t)
	})

	// Test retrieval failure
	t.Run("get_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("GetAccountValue", ctx).Return(0.0, assert.AnError)

		value, err := provider.GetAccountValue(ctx)
		assert.Error(t, err)
		assert.Equal(t, 0.0, value)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderGetAssetIndex(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()

	// Test successful retrieval
	t.Run("successful_get", func(t *testing.T) {
		expectedIndex := 1
		mockClient.On("GetAssetIndex", ctx, "BTC").Return(expectedIndex, nil)

		index, err := provider.GetAssetIndex(ctx, "BTC")
		assert.NoError(t, err)
		assert.Equal(t, expectedIndex, index)
		mockClient.AssertExpectations(t)
	})

	// Test retrieval failure
	t.Run("get_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("GetAssetIndex", ctx, "BTC").Return(0, assert.AnError)

		index, err := provider.GetAssetIndex(ctx, "BTC")
		assert.Error(t, err)
		assert.Equal(t, 0, index)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderIOCMarket(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()

	// Test successful IOC market order
	t.Run("successful_ioc", func(t *testing.T) {
		expectedResponse := &exchange.OrderResponse{Status: "success"}
		mockClient.On("IOCMarket", ctx, "BTC", true, 0.01, 0.001, false).Return(expectedResponse, nil)

		resp, err := provider.IOCMarket(ctx, "BTC", true, 0.01, 0.001, false)
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, resp)
		mockClient.AssertExpectations(t)
	})

	// Test IOC market order failure
	t.Run("ioc_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("IOCMarket", ctx, "BTC", true, 0.01, 0.001, false).Return((*exchange.OrderResponse)(nil), assert.AnError)

		resp, err := provider.IOCMarket(ctx, "BTC", true, 0.01, 0.001, false)
		assert.Error(t, err)
		assert.Nil(t, resp)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderSetStopLoss(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()

	// Test successful stop loss placement
	t.Run("successful_stop_loss", func(t *testing.T) {
		// For LONG positions, stop loss should SELL to close -> isBuy=false
		mockClient.On("PlaceTriggerReduceOnly", ctx, "BTC", false, 0.01, 45000.0, "sl").Return(nil)

		err := provider.SetStopLoss(ctx, "BTC", "LONG", 0.01, 45000.0)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	// Test stop loss placement failure
	t.Run("stop_loss_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("PlaceTriggerReduceOnly", ctx, "BTC", false, 0.01, 45000.0, "sl").Return(assert.AnError)

		err := provider.SetStopLoss(ctx, "BTC", "LONG", 0.01, 45000.0)
		assert.Error(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderSetTakeProfit(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()

	// Test successful take profit placement
	t.Run("successful_take_profit", func(t *testing.T) {
		// For LONG positions, take profit is also a SELL -> isBuy=false
		mockClient.On("PlaceTriggerReduceOnly", ctx, "BTC", false, 0.01, 55000.0, "tp").Return(nil)

		err := provider.SetTakeProfit(ctx, "BTC", "LONG", 0.01, 55000.0)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	// Test take profit placement failure
	t.Run("take_profit_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("PlaceTriggerReduceOnly", ctx, "BTC", false, 0.01, 55000.0, "tp").Return(assert.AnError)

		err := provider.SetTakeProfit(ctx, "BTC", "LONG", 0.01, 55000.0)
		assert.Error(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderCancelAllBySymbol(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()

	// Test successful cancel all
	t.Run("successful_cancel_all", func(t *testing.T) {
		mockClient.On("GetAssetIndex", ctx, "BTC").Return(1, nil)
		mockClient.On("CancelAllOrders", ctx, 1).Return(nil)

		err := provider.CancelAllBySymbol(ctx, "BTC")
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	// Test cancel all failure due to asset index error
	t.Run("asset_index_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("GetAssetIndex", ctx, "BTC").Return(0, assert.AnError)

		err := provider.CancelAllBySymbol(ctx, "BTC")
		assert.Error(t, err)
		mockClient.AssertExpectations(t)
	})

	// Test cancel all failure due to cancellation error
	t.Run("cancellation_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("GetAssetIndex", ctx, "BTC").Return(1, nil)
		mockClient.On("CancelAllOrders", ctx, 1).Return(assert.AnError)

		err := provider.CancelAllBySymbol(ctx, "BTC")
		assert.Error(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderFormatSize(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()

	// Test successful formatting
	t.Run("successful_format", func(t *testing.T) {
		expectedSize := "0.010"
		mockClient.On("FormatSize", ctx, "BTC", 0.01).Return(expectedSize, nil)

		size, err := provider.FormatSize(ctx, "BTC", 0.01)
		assert.NoError(t, err)
		assert.Equal(t, expectedSize, size)
		mockClient.AssertExpectations(t)
	})

	// Test formatting failure
	t.Run("format_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("FormatSize", ctx, "BTC", 0.01).Return("", assert.AnError)

		size, err := provider.FormatSize(ctx, "BTC", 0.01)
		assert.Error(t, err)
		assert.Empty(t, size)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderFormatPrice(t *testing.T) {
	// Create mock client
	mockClient := &MockClient{}
	provider := &Provider{client: mockClient}

	ctx := context.Background()

	// Test successful formatting
	t.Run("successful_format", func(t *testing.T) {
		expectedPrice := "50000.00"
		mockClient.On("FormatPrice", ctx, "BTC", 50000.0).Return(expectedPrice, nil)

		price, err := provider.FormatPrice(ctx, "BTC", 50000.0)
		assert.NoError(t, err)
		assert.Equal(t, expectedPrice, price)
		mockClient.AssertExpectations(t)
	})

	// Test formatting failure
	t.Run("format_failure", func(t *testing.T) {
		mockClient.ExpectedCalls = nil // Reset expectations
		mockClient.On("FormatPrice", ctx, "BTC", 50000.0).Return("", assert.AnError)

		price, err := provider.FormatPrice(ctx, "BTC", 50000.0)
		assert.Error(t, err)
		assert.Empty(t, price)
		mockClient.AssertExpectations(t)
	})
}

func TestProviderInit(t *testing.T) {
	// Test that init function registers the provider
	t.Run("provider_registration", func(t *testing.T) {
		// This test verifies that the init function properly registers the provider
		// by attempting to create a provider through the exchange package
		provider, err := exchange.GetProvider("hyperliquid", &exchange.ProviderConfig{
			PrivateKey: "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f",
			Testnet:    false,
		})
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})
}
