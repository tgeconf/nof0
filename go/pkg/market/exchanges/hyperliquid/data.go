package hyperliquid

import (
	"context"
	"fmt"
	"math"
	"strings"

	"nof0-api/pkg/market"
	"nof0-api/pkg/market/indicators"
)

const (
	intradayInterval       = "3m"
	intradayLookback       = 40
	intradaySeriesLength   = 10
	intradayChangeLookback = 20
	longerInterval         = "4h"
	longerLookback         = 60
	priceChange4hLookback  = 1
)

func (c *Client) buildSnapshot(ctx context.Context, symbol string) (*market.Snapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	info, err := c.GetMarketInfo(ctx, symbol)
	if err != nil {
		return nil, err
	}

	intradayKlines, err := c.GetKlines(ctx, info.Symbol, intradayInterval, intradayLookback)
	if err != nil {
		return nil, err
	}
	longerKlines, err := c.GetKlines(ctx, info.Symbol, longerInterval, longerLookback)
	if err != nil {
		return nil, err
	}

	lastPrice, err := c.getCurrentPriceForCanonical(ctx, info.Symbol)
	if err != nil {
		return nil, err
	}

	intradaySeries, intradaySignals := buildIntradaySeries(intradayKlines)
	longerSeries, longerSignals := buildLongerSeries(longerKlines)

	change1h := calculatePriceChange(lastPrice, priceAt(intradayKlines, intradayChangeLookback))
	change4h := calculatePriceChange(lastPrice, priceAt(longerKlines, priceChange4hLookback))

	indicatorEMA := make(map[string]float64)
	indicatorRSI := make(map[string]float64)

	if intradaySignals != nil {
		if !math.IsNaN(intradaySignals.ema20) {
			indicatorEMA["EMA20"] = intradaySignals.ema20
		}
		if !math.IsNaN(intradaySignals.rsi7) {
			indicatorRSI["RSI7"] = intradaySignals.rsi7
		}
	}
	if longerSignals != nil {
		if !math.IsNaN(longerSignals.ema20) {
			indicatorEMA["EMA20_Long"] = longerSignals.ema20
		}
		if !math.IsNaN(longerSignals.ema50) {
			indicatorEMA["EMA50"] = longerSignals.ema50
		}
		if !math.IsNaN(longerSignals.rsi14) {
			indicatorRSI["RSI14"] = longerSignals.rsi14
		}
	}

	indicator := market.IndicatorInfo{
		EMA: indicatorEMA,
		RSI: indicatorRSI,
	}
	if intradaySignals != nil && !math.IsNaN(intradaySignals.macd) {
		indicator.MACD = intradaySignals.macd
	} else if longerSignals != nil && !math.IsNaN(longerSignals.macd) {
		indicator.MACD = longerSignals.macd
	}

	var funding *market.FundingInfo
	if !math.IsNaN(info.FundingRate) && info.FundingRate != 0 {
		funding = &market.FundingInfo{
			Rate: info.FundingRate,
		}
	}

	var openInterest *market.OpenInterestInfo
	if info.OpenInterest != 0 {
		openInterest = &market.OpenInterestInfo{
			Latest:  info.OpenInterest,
			Average: info.OpenInterest, // TODO: replace with historical averaging when Hyperliquid exposes the series.
		}
	}

	snapshot := &market.Snapshot{
		Symbol: info.Symbol,
		Price: market.PriceInfo{
			Last: lastPrice,
		},
		Change: market.ChangeInfo{
			OneHour:  change1h,
			FourHour: change4h,
		},
		Indicators:   indicator,
		OpenInterest: openInterest,
		Funding:      funding,
		Intraday:     intradaySeries,
		LongTerm:     longerSeries,
	}

	return snapshot, nil
}

func (c *Client) getCurrentPriceForCanonical(ctx context.Context, symbol string) (float64, error) {
	var response AllMidsResponse
	if err := c.doRequest(ctx, InfoRequest{Type: "allMids"}, &response); err != nil {
		return 0, err
	}
	if price, ok := response[symbol]; ok {
		return parseFloat(price)
	}
	if price, ok := response[strings.ToUpper(symbol)]; ok {
		return parseFloat(price)
	}
	return 0, fmt.Errorf("hyperliquid: price for %s not available", symbol)
}

type indicatorSnapshot struct {
	ema20 float64
	ema50 float64
	macd  float64
	rsi7  float64
	rsi14 float64
}

func buildIntradaySeries(klines []Kline) (*market.SeriesBundle, *indicatorSnapshot) {
	if len(klines) == 0 {
		return nil, nil
	}
	closes := extractCloses(klines)
	volumes := extractVolumes(klines)

	ema20 := indicators.EMA(closes, 20)
	macd, _, _ := indicators.MACD(closes)
	rsi7 := indicators.RSI(closes, 7)
	rsi14 := indicators.RSI(closes, 14)

	series := &market.SeriesBundle{
		Prices: lastN(closes, intradaySeriesLength),
		EMA:    map[string][]float64{"EMA20": lastN(ema20, intradaySeriesLength)},
		MACD:   lastN(macd, intradaySeriesLength),
		RSI: map[string][]float64{
			"RSI7":  lastN(rsi7, intradaySeriesLength),
			"RSI14": lastN(rsi14, intradaySeriesLength),
		},
		Volume: lastN(volumes, intradaySeriesLength),
	}

	snapshot := &indicatorSnapshot{
		ema20: latestNonNaN(ema20),
		macd:  latestNonNaN(macd),
		rsi7:  latestNonNaN(rsi7),
		rsi14: latestNonNaN(rsi14),
	}
	return series, snapshot
}

func buildLongerSeries(klines []Kline) (*market.SeriesBundle, *indicatorSnapshot) {
	if len(klines) == 0 {
		return nil, nil
	}
	closes := extractCloses(klines)
	volumes := extractVolumes(klines)

	ema20 := indicators.EMA(closes, 20)
	ema50 := indicators.EMA(closes, 50)
	macd, _, _ := indicators.MACD(closes)
	rsi14 := indicators.RSI(closes, 14)

	atrInput := convertForATR(klines)
	atr3 := indicators.ATR(atrInput, 3)
	atr14 := indicators.ATR(atrInput, 14)

	series := &market.SeriesBundle{
		Prices: lastN(closes, intradaySeriesLength),
		EMA: map[string][]float64{
			"EMA20": lastN(ema20, intradaySeriesLength),
			"EMA50": lastN(ema50, intradaySeriesLength),
		},
		MACD: lastN(macd, intradaySeriesLength),
		RSI: map[string][]float64{
			"RSI14": lastN(rsi14, intradaySeriesLength),
		},
		ATR: map[string][]float64{
			"ATR3":  lastN(atr3, intradaySeriesLength),
			"ATR14": lastN(atr14, intradaySeriesLength),
		},
		Volume: lastN(volumes, intradaySeriesLength),
	}

	snapshot := &indicatorSnapshot{
		ema20: latestNonNaN(ema20),
		ema50: latestNonNaN(ema50),
		macd:  latestNonNaN(macd),
		rsi14: latestNonNaN(rsi14),
	}
	return series, snapshot
}

// calculatePriceChange returns the fractional change (e.g., 0.01 == +1%).
func calculatePriceChange(currentPrice, previousPrice float64) float64 {
	if previousPrice == 0 {
		return 0
	}
	return (currentPrice - previousPrice) / previousPrice
}

func priceAt(klines []Kline, stepsBack int) float64 {
	if len(klines) == 0 || stepsBack <= 0 || len(klines) <= stepsBack {
		return 0
	}
	return klines[len(klines)-1-stepsBack].Close
}

func extractCloses(klines []Kline) []float64 {
	out := make([]float64, len(klines))
	for i, k := range klines {
		out[i] = k.Close
	}
	return out
}

func extractVolumes(klines []Kline) []float64 {
	out := make([]float64, len(klines))
	for i, k := range klines {
		out[i] = k.Volume
	}
	return out
}

func convertForATR(klines []Kline) []indicators.Kline {
	out := make([]indicators.Kline, len(klines))
	for i, k := range klines {
		out[i] = indicators.Kline{
			High:  k.High,
			Low:   k.Low,
			Close: k.Close,
		}
	}
	return out
}

func lastN(values []float64, count int) []float64 {
	if len(values) == 0 {
		return []float64{}
	}
	if len(values) <= count {
		return append([]float64(nil), values...)
	}
	return append([]float64(nil), values[len(values)-count:]...)
}

func latestNonNaN(values []float64) float64 {
	for i := len(values) - 1; i >= 0; i-- {
		if !math.IsNaN(values[i]) {
			return values[i]
		}
	}
	return math.NaN()
}
