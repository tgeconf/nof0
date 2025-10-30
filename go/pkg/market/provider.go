package market

import "context"

// Provider exposes exchange-agnostic market data.
type Provider interface {
	// Snapshot returns a normalized market snapshot for the specified symbol.
	Snapshot(ctx context.Context, symbol string) (*Snapshot, error)
	// ListAssets returns all supported symbols along with metadata.
	ListAssets(ctx context.Context) ([]Asset, error)
}

// Snapshot captures a normalized market view for a trading symbol.
type Snapshot struct {
	Symbol       string            // Exchange symbol as traded
	Price        PriceInfo         // Latest price data
	Change       ChangeInfo        // Percentage changes across time windows
	Indicators   IndicatorInfo     // Calculated technical indicators
	OpenInterest *OpenInterestInfo // Derivatives interest data, if available
	Funding      *FundingInfo      // Perpetual funding information, if available
	Intraday     *SeriesBundle     // Short-term time series context
	LongTerm     *SeriesBundle     // Longer-term time series context
}

// Asset describes a tradeable instrument.
type Asset struct {
	Symbol      string         // Exchange-native symbol, e.g. "BTC", "kPEPE"
	Base        string         // Optional base asset
	Quote       string         // Optional quote asset
	Precision   int            // Price precision when available
	IsActive    bool           // Whether the asset is currently tradeable
	RawMetadata map[string]any // Exchange-specific fields for callers that need more detail
}

// PriceInfo holds last trade data.
type PriceInfo struct {
	Last float64
}

// ChangeInfo describes percentage changes over standard windows.
type ChangeInfo struct {
	OneHour  float64
	FourHour float64
}

// IndicatorInfo aggregates computed indicator values.
type IndicatorInfo struct {
	EMA  map[string]float64 // e.g., {"EMA20": 1234.5}
	MACD float64
	RSI  map[string]float64 // e.g., {"RSI7": 70.1}
}

// OpenInterestInfo reports derivatives open interest metrics.
type OpenInterestInfo struct {
	Latest  float64
	Average float64
}

// FundingInfo captures perpetual funding rate data.
type FundingInfo struct {
	Rate float64 // expressed as percentage (e.g., 0.01 == 1%)
}

// SeriesBundle provides supporting time series data for analysis layers.
type SeriesBundle struct {
	Prices []float64            // Ordered oldest → newest close prices
	EMA    map[string][]float64 // EMA series keyed by window
	MACD   []float64            // MACD values
	RSI    map[string][]float64 // RSI series keyed by window
	ATR    map[string][]float64 // ATR series keyed by window
	Volume []float64            // Volume series when available
}
