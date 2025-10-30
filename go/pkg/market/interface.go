package market

import (
	"context"
	"time"

	"nof0-api/pkg/market/hyperliquid"
)

const defaultRequestTimeout = 8 * time.Second

// MarketDataProvider describes an exchange market data source.
type MarketDataProvider interface {
	Get(symbol string) (*hyperliquid.Data, error)
	GetCurrentPrice(symbol string) (float64, error)
}

// HyperliquidProvider implements MarketDataProvider via Hyperliquid endpoints.
type HyperliquidProvider struct {
	client  *hyperliquid.Client
	timeout time.Duration
}

// ProviderOption customises the Hyperliquid provider.
type ProviderOption func(*HyperliquidProvider)

// WithClient injects a custom Hyperliquid client.
func WithClient(client *hyperliquid.Client) ProviderOption {
	return func(p *HyperliquidProvider) {
		if client != nil {
			p.client = client
		}
	}
}

// WithTimeout overrides the per-request timeout.
func WithTimeout(timeout time.Duration) ProviderOption {
	return func(p *HyperliquidProvider) {
		if timeout > 0 {
			p.timeout = timeout
		}
	}
}

// NewHyperliquidProvider constructs a default provider.
func NewHyperliquidProvider(opts ...ProviderOption) MarketDataProvider {
	provider := &HyperliquidProvider{
		client:  hyperliquid.NewClient(),
		timeout: defaultRequestTimeout,
	}
	for _, opt := range opts {
		opt(provider)
	}
	return provider
}

// Get assembles the full market snapshot for the supplied symbol.
func (p *HyperliquidProvider) Get(symbol string) (*hyperliquid.Data, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	return p.client.GetMarketData(ctx, symbol)
}

// GetCurrentPrice returns the latest mid price.
func (p *HyperliquidProvider) GetCurrentPrice(symbol string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	return p.client.GetCurrentPrice(ctx, symbol)
}
