package hyperliquid

import (
	"encoding/json"
	"fmt"
)

// Data represents the aggregated market snapshot returned by Hyperliquid.
type Data struct {
	Symbol            string          // Trading symbol, e.g. "BTC"
	CurrentPrice      float64         // Latest traded price
	PriceChange1h     float64         // Percentage change over the past hour
	PriceChange4h     float64         // Percentage change over the past 4 hours
	CurrentEMA20      float64         // Latest 20-period EMA value
	CurrentMACD       float64         // Latest MACD value
	CurrentRSI7       float64         // Latest 7-period RSI
	OpenInterest      *OIData         // Open interest metrics
	FundingRate       float64         // Funding rate percentage
	IntradaySeries    *IntradayData   // Recent intraday series based on 3m candles
	LongerTermContext *LongerTermData // Longer-term context based on 4h candles
}

// OIData captures open interest statistics.
type OIData struct {
	Latest  float64 // Latest reported open interest
	Average float64 // Averaged open interest over the selected lookback
}

// IntradayData groups short-term series derived from 3m candles.
type IntradayData struct {
	MidPrices   []float64 // Closing price series (oldest to newest)
	EMA20Values []float64 // 20-period EMA series
	MACDValues  []float64 // MACD values
	RSI7Values  []float64 // 7-period RSI values
	RSI14Values []float64 // 14-period RSI values
}

// LongerTermData holds indicators computed from 4h candles.
type LongerTermData struct {
	EMA20         float64   // 20-period EMA
	EMA50         float64   // 50-period EMA
	ATR3          float64   // 3-period ATR
	ATR14         float64   // 14-period ATR
	CurrentVolume float64   // Latest candle volume
	AverageVolume float64   // Average volume across recent candles
	MACDValues    []float64 // Recent MACD values (oldest to newest)
	RSI14Values   []float64 // Recent RSI14 values (oldest to newest)
}

// Kline represents a single OHLCV candlestick.
type Kline struct {
	OpenTime  int64   // Open time in milliseconds
	Open      float64 // Open price
	High      float64 // High price
	Low       float64 // Low price
	Close     float64 // Close price
	Volume    float64 // Traded volume
	CloseTime int64   // Close time in milliseconds
}

// InfoRequest is the shared envelope for Hyperliquid info endpoint requests.
type InfoRequest struct {
	Type string      `json:"type"`
	Req  interface{} `json:"req,omitempty"`
}

// CandleSnapshotRequest carries parameters for the candleSnapshot request.
type CandleSnapshotRequest struct {
	Coin      string `json:"coin"`
	Interval  string `json:"interval"` // e.g. "3m", "4h"
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}

// CandleResponse mirrors the payload returned from candleSnapshot requests.
type CandleResponse []struct {
	T      int64   `json:"t"`        // Open timestamp (ms)
	TClose int64   `json:"T"`        // Close timestamp (ms)
	S      string  `json:"s"`        // Symbol
	I      string  `json:"i"`        // Interval
	O      float64 `json:"o,string"` // Open price
	C      float64 `json:"c,string"` // Close price
	H      float64 `json:"h,string"` // High price
	L      float64 `json:"l,string"` // Low price
	V      float64 `json:"v,string"` // Volume
}

// MetaAndAssetCtxsResponse contains market meta data and per-asset contexts.
type MetaAndAssetCtxsResponse struct {
	Universe  []UniverseEntry
	AssetCtxs []AssetCtx
}

// UniverseEntry enumerates tradable assets on Hyperliquid.
type UniverseEntry struct {
	Name string `json:"name"`
}

// AssetCtx holds per-symbol market context such as funding and open interest.
type AssetCtx struct {
	Funding      string   `json:"funding"`
	OpenInterest string   `json:"openInterest"`
	PrevDayPx    string   `json:"prevDayPx"`
	DayNtlVlm    string   `json:"dayNtlVlm"`
	DayBaseVlm   string   `json:"dayBaseVlm"`
	Premium      string   `json:"premium"`
	OraclePx     string   `json:"oraclePx"`
	MarkPx       string   `json:"markPx"`
	MidPx        string   `json:"midPx"`
	ImpactPxs    []string `json:"impactPxs"`
}

// UnmarshalJSON customises parsing to accommodate both documented and live API payloads.
func (m *MetaAndAssetCtxsResponse) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch len(raw) {
	case 0:
		return fmt.Errorf("unexpected metaAndAssetCtxs payload: empty array")
	case 1:
		var meta struct {
			Universe  []UniverseEntry `json:"universe"`
			AssetCtxs []AssetCtx      `json:"assetCtxs"`
		}
		if err := json.Unmarshal(raw[0], &meta); err != nil {
			return err
		}
		m.Universe = meta.Universe
		m.AssetCtxs = meta.AssetCtxs
	default:
		var meta struct {
			Universe []UniverseEntry `json:"universe"`
		}
		if err := json.Unmarshal(raw[0], &meta); err != nil {
			return err
		}
		var assetCtxs []AssetCtx
		if err := json.Unmarshal(raw[1], &assetCtxs); err != nil {
			return err
		}
		m.Universe = meta.Universe
		m.AssetCtxs = assetCtxs
	}
	return nil
}

// AllMidsResponse maps symbols to their current mid prices.
type AllMidsResponse map[string]string
