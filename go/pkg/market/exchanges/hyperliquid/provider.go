package hyperliquid

import (
	"context"
	"net/http"
	"sort"
	"time"

	"nof0-api/pkg/market"
)

const defaultProviderTimeout = 8 * time.Second

// Provider wraps Hyperliquid client calls behind the generic market.Provider contract.
type Provider struct {
	client  *Client
	timeout time.Duration
}

type providerConfig struct {
	timeout      time.Duration
	clientConfig []Option
}

// ProviderOption customises the Hyperliquid provider.
type ProviderOption func(*providerConfig)

// WithTimeout overrides the default per-call timeout.
func WithTimeout(timeout time.Duration) ProviderOption {
	return func(cfg *providerConfig) {
		if timeout > 0 {
			cfg.timeout = timeout
		}
	}
}

// WithClientOptions passes options to the underlying Hyperliquid client.
func WithClientOptions(options ...Option) ProviderOption {
	return func(cfg *providerConfig) {
		cfg.clientConfig = append(cfg.clientConfig, options...)
	}
}

// NewProvider constructs a Hyperliquid market provider.
func NewProvider(opts ...ProviderOption) *Provider {
	cfg := &providerConfig{
		timeout: defaultProviderTimeout,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	client := NewClient(cfg.clientConfig...)
	return &Provider{
		client:  client,
		timeout: cfg.timeout,
	}
}

func init() {
	market.RegisterProvider("hyperliquid", func(name string, cfg *market.ProviderConfig) (market.Provider, error) {
		opts := []ProviderOption{}
		clientOptions := []Option{}
		if cfg.Timeout > 0 {
			opts = append(opts, WithTimeout(cfg.Timeout))
		}
		if cfg.HTTPTimeout > 0 {
			clientOptions = append(clientOptions, WithHTTPClient(&http.Client{Timeout: cfg.HTTPTimeout}))
		}
		if cfg.Testnet {
			clientOptions = append(clientOptions, WithBaseURL(testnetBaseURL))
		}
		if cfg.MaxRetries > 0 {
			clientOptions = append(clientOptions, WithMaxRetries(cfg.MaxRetries))
		}
		if len(clientOptions) > 0 {
			opts = append(opts, WithClientOptions(clientOptions...))
		}
		return NewProvider(opts...), nil
	})
}

// Snapshot implements market.Provider by returning an aggregated market snapshot.
func (p *Provider) Snapshot(ctx context.Context, symbol string) (*market.Snapshot, error) {
	ctx, cancel := p.withTimeout(ctx)
	defer cancel()
	return p.client.buildSnapshot(ctx, symbol)
}

// ListAssets implements market.Provider by returning all supported symbols.
func (p *Provider) ListAssets(ctx context.Context) ([]market.Asset, error) {
	ctx, cancel := p.withTimeout(ctx)
	defer cancel()

	if err := p.client.refreshSymbolDirectory(ctx); err != nil {
		return nil, err
	}
	return p.collectAssets(), nil
}

func (p *Provider) collectAssets() []market.Asset {
	p.client.symbolsMu.RLock()
	defer p.client.symbolsMu.RUnlock()

	assets := make([]market.Asset, 0, len(p.client.symbolIndex))
	for _, canonical := range p.client.symbolIndex {
		meta := p.client.universeMeta[canonical]
		ctxData := p.client.assetCtxBySymbol[canonical]
		asset := market.Asset{
			Symbol:    canonical,
			Base:      "", // Hyperliquid uses single-coin naming; leave base/quote empty for now.
			Quote:     "",
			Precision: meta.SzDecimals,
			IsActive:  !meta.IsDelisted,
			RawMetadata: map[string]any{
				"maxLeverage":  meta.MaxLeverage,
				"marginTable":  meta.MarginTableID,
				"onlyIsolated": meta.OnlyIsolated,
				"funding":      ctxData.Funding,
				"openInterest": ctxData.OpenInterest,
				"dayBaseVlm":   ctxData.DayBaseVlm,
				"dayNtlVlm":    ctxData.DayNtlVlm,
				"prevDayPx":    ctxData.PrevDayPx,
			},
		}
		assets = append(assets, asset)
	}

	sort.Slice(assets, func(i, j int) bool {
		return assets[i].Symbol < assets[j].Symbol
	})
	return assets
}

func (p *Provider) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, p.timeout)
}
