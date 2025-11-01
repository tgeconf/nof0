package hyperliquid

import (
	"context"
	"net/http"
	"strings"

	"nof0-api/pkg/exchange"
)

// Provider wraps Client to satisfy the exchange.Provider interface.
// clientAPI captures the subset of client behavior the provider relies on.
type clientAPI interface {
	PlaceOrder(ctx context.Context, order exchange.Order) (*exchange.OrderResponse, error)
	CancelOrder(ctx context.Context, asset int, oid int64) error
	GetOpenOrders(ctx context.Context) ([]exchange.OrderStatus, error)
	GetPositions(ctx context.Context) ([]exchange.Position, error)
	ClosePosition(ctx context.Context, coin string) error
	UpdateLeverage(ctx context.Context, asset int, isCross bool, leverage int) error
	GetAccountState(ctx context.Context) (*exchange.AccountState, error)
	GetAccountValue(ctx context.Context) (float64, error)
	GetAssetIndex(ctx context.Context, coin string) (int, error)

	// Convenience methods used by the provider
	IOCMarket(ctx context.Context, coin string, isBuy bool, qty float64, slippage float64, reduceOnly bool) (*exchange.OrderResponse, error)
	PlaceTriggerReduceOnly(ctx context.Context, coin string, isBuy bool, qty float64, triggerPrice float64, tpsl string) error
	CancelAllOrders(ctx context.Context, asset int) error
	FormatSize(ctx context.Context, coin string, qty float64) (string, error)
	FormatPrice(ctx context.Context, coin string, price float64) (string, error)
}

type Provider struct {
	client clientAPI
}

// NewProvider constructs a Hyperliquid exchange provider.
func NewProvider(privateKeyHex string, isTestnet bool, opts ...ClientOption) (*Provider, error) {
	client, err := NewClient(privateKeyHex, isTestnet, opts...)
	if err != nil {
		return nil, err
	}
	return &Provider{client: client}, nil
}

func init() {
	exchange.RegisterProvider("hyperliquid", func(name string, cfg *exchange.ProviderConfig) (exchange.Provider, error) {
		opts := []ClientOption{}
		if cfg.Timeout > 0 {
			opts = append(opts, WithHTTPClient(&http.Client{Timeout: cfg.Timeout}))
		}
		if cfg.VaultAddress != "" {
			opts = append(opts, WithVaultAddress(cfg.VaultAddress))
		}
		if cfg.MainAddress != "" {
			opts = append(opts, WithMainAddress(cfg.MainAddress))
		}
		return NewProvider(cfg.PrivateKey, cfg.Testnet, opts...)
	})
}

// PlaceOrder delegates to the underlying client.
func (p *Provider) PlaceOrder(ctx context.Context, order exchange.Order) (*exchange.OrderResponse, error) {
	return p.client.PlaceOrder(ctx, order)
}

// CancelOrder cancels a single order.
func (p *Provider) CancelOrder(ctx context.Context, asset int, oid int64) error {
	return p.client.CancelOrder(ctx, asset, oid)
}

// GetOpenOrders returns currently resting orders.
func (p *Provider) GetOpenOrders(ctx context.Context) ([]exchange.OrderStatus, error) {
	return p.client.GetOpenOrders(ctx)
}

// GetPositions fetches all open positions.
func (p *Provider) GetPositions(ctx context.Context) ([]exchange.Position, error) {
	return p.client.GetPositions(ctx)
}

// ClosePosition attempts to close an open position.
func (p *Provider) ClosePosition(ctx context.Context, coin string) error {
	return p.client.ClosePosition(ctx, coin)
}

// UpdateLeverage updates leverage configuration.
func (p *Provider) UpdateLeverage(ctx context.Context, asset int, isCross bool, leverage int) error {
	return p.client.UpdateLeverage(ctx, asset, isCross, leverage)
}

// GetAccountState returns current account state.
func (p *Provider) GetAccountState(ctx context.Context) (*exchange.AccountState, error) {
	return p.client.GetAccountState(ctx)
}

// GetAccountValue returns parsed account value.
func (p *Provider) GetAccountValue(ctx context.Context) (float64, error) {
	return p.client.GetAccountValue(ctx)
}

// GetAssetIndex resolves asset index for a symbol.
func (p *Provider) GetAssetIndex(ctx context.Context, coin string) (int, error) {
	return p.client.GetAssetIndex(ctx, coin)
}

// Convenience wrappers (not part of the generic exchange.Provider interface)

// IOCMarket places an IOC order using a small price slippage as market.
func (p *Provider) IOCMarket(ctx context.Context, coin string, isBuy bool, qty float64, slippage float64, reduceOnly bool) (*exchange.OrderResponse, error) {
	return p.client.IOCMarket(ctx, coin, isBuy, qty, slippage, reduceOnly)
}

// SetStopLoss places a reduce-only trigger order as stop loss.
// positionSide: "LONG" or "SHORT".
func (p *Provider) SetStopLoss(ctx context.Context, coin string, positionSide string, qty float64, stopPrice float64) error {
	isBuy := strings.EqualFold(positionSide, "SHORT") // buy to cover short
	return p.client.PlaceTriggerReduceOnly(ctx, coin, isBuy, qty, stopPrice, "sl")
}

// SetTakeProfit places a reduce-only trigger order as take profit.
// positionSide: "LONG" or "SHORT".
func (p *Provider) SetTakeProfit(ctx context.Context, coin string, positionSide string, qty float64, takeProfit float64) error {
	isBuy := strings.EqualFold(positionSide, "SHORT")
	return p.client.PlaceTriggerReduceOnly(ctx, coin, isBuy, qty, takeProfit, "tp")
}

// CancelAllBySymbol cancels all resting orders for the given symbol.
func (p *Provider) CancelAllBySymbol(ctx context.Context, coin string) error {
	idx, err := p.client.GetAssetIndex(ctx, coin)
	if err != nil {
		return err
	}
	return p.client.CancelAllOrders(ctx, idx)
}

// FormatSize rounds a float quantity to szDecimals and returns a string.
func (p *Provider) FormatSize(ctx context.Context, coin string, qty float64) (string, error) {
	return p.client.FormatSize(ctx, coin, qty)
}

// FormatPrice rounds and formats a price using the client's configured
// significant figures, after verifying the asset exists.
func (p *Provider) FormatPrice(ctx context.Context, coin string, price float64) (string, error) {
	return p.client.FormatPrice(ctx, coin, price)
}
