package market

import (
	"context"
	"time"
)

// Persistence hooks allow providers to persist market data to external stores.
type Persistence interface {
	// UpsertAssets persists static asset metadata for the provider.
	UpsertAssets(ctx context.Context, provider string, assets []Asset) error
	// RecordSnapshot persists a single market snapshot (price/latest context).
	RecordSnapshot(ctx context.Context, provider string, snapshot *Snapshot) error
	// RecordPriceSeries persists historical price ticks (e.g., OHLCV candles).
	RecordPriceSeries(ctx context.Context, provider string, symbol string, ticks []PriceTick) error
}

// PriceTick represents a normalized OHLCV candle for persistence.
type PriceTick struct {
	Timestamp time.Time
	Price     float64
	Volume    float64
	HasVolume bool
	Interval  string
	Open      float64
	High      float64
	Low       float64
	Close     float64
}
