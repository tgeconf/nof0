package hyperliquid

import (
	"context"
	"net/http"

	"nof0-api/pkg/exchange"
)

// Provider wraps Client to satisfy the exchange.Provider interface.
type Provider struct {
	client *Client
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
