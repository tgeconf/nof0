package hyperliquid

import (
	"context"
	"fmt"
	"math"
	"strings"
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

// GetMarketData orchestrates the full aggregation flow for a symbol.
func (c *Client) GetMarketData(ctx context.Context, symbol string) (*Data, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Retrieve market info first to obtain canonical symbol casing.
	info, err := c.GetMarketInfo(ctx, symbol)
	if err != nil {
		return nil, err
	}

	// Fetch candles in sequence. Hyperliquid does not expose bulk endpoints yet, so requests are serial.
	intradayKlines, err := c.GetKlines(ctx, info.Symbol, intradayInterval, intradayLookback)
	if err != nil {
		return nil, err
	}
	longerKlines, err := c.GetKlines(ctx, info.Symbol, longerInterval, longerLookback)
	if err != nil {
		return nil, err
	}

	price, err := c.getCurrentPriceForCanonical(ctx, info.Symbol)
	if err != nil {
		return nil, err
	}

	var marketInfo *MarketInfo
	marketInfo = info

	intraday := calculateIntradayData(intradayKlines)
	longerTerm := calculateLongerTermData(longerKlines)

	var change1h float64
	if len(intradayKlines) > intradayChangeLookback {
		reference := intradayKlines[len(intradayKlines)-1-intradayChangeLookback].Close
		change1h = calculatePriceChange(price, reference)
	}

	var change4h float64
	if len(longerKlines) > priceChange4hLookback {
		reference := longerKlines[len(longerKlines)-1-priceChange4hLookback].Close
		change4h = calculatePriceChange(price, reference)
	}

	var currentEMA20 float64
	var currentMACD float64
	var currentRSI7 float64
	if intraday != nil {
		currentEMA20 = latestNonNaN(intraday.EMA20Values)
		currentMACD = latestNonNaN(intraday.MACDValues)
		currentRSI7 = latestNonNaN(intraday.RSI7Values)
	}

	oi := &OIData{}
	if marketInfo != nil {
		oi.Latest = marketInfo.OpenInterest
		// TODO: replace with historical open interest averaging once the endpoint is available.
		oi.Average = marketInfo.OpenInterest
	}

	data := &Data{
		Symbol:            marketInfo.Symbol,
		CurrentPrice:      price,
		PriceChange1h:     change1h,
		PriceChange4h:     change4h,
		CurrentEMA20:      currentEMA20,
		CurrentMACD:       currentMACD,
		CurrentRSI7:       currentRSI7,
		OpenInterest:      oi,
		FundingRate:       marketInfo.FundingRate,
		IntradaySeries:    intraday,
		LongerTermContext: longerTerm,
	}

	return data, nil
}

func calculatePriceChange(currentPrice, previousPrice float64) float64 {
	if previousPrice == 0 {
		return 0
	}
	return (currentPrice - previousPrice) / previousPrice * 100
}

func calculateIntradayData(klines []Kline) *IntradayData {
	if len(klines) == 0 {
		return nil
	}
	closes := extractCloses(klines)

	data := &IntradayData{
		MidPrices: lastN(closes, intradaySeriesLength),
	}

	ema20 := CalculateEMA(closes, 20)
	data.EMA20Values = lastN(ema20, intradaySeriesLength)

	macd, _, _ := CalculateMACD(closes)
	data.MACDValues = lastN(macd, intradaySeriesLength)

	rsi7 := CalculateRSI(closes, 7)
	data.RSI7Values = lastN(rsi7, intradaySeriesLength)

	rsi14 := CalculateRSI(closes, 14)
	data.RSI14Values = lastN(rsi14, intradaySeriesLength)

	return data
}

func calculateLongerTermData(klines []Kline) *LongerTermData {
	if len(klines) == 0 {
		return nil
	}
	closes := extractCloses(klines)

	ema20 := CalculateEMA(closes, 20)
	ema50 := CalculateEMA(closes, 50)
	atr3 := CalculateATR(klines, 3)
	atr14 := CalculateATR(klines, 14)
	macd, _, _ := CalculateMACD(closes)
	rsi14 := CalculateRSI(closes, 14)

	avgWindow := minInt(20, len(klines))
	var volumeSum float64
	for _, k := range klines[len(klines)-avgWindow:] {
		volumeSum += k.Volume
	}

	data := &LongerTermData{
		EMA20:         latestNonNaN(ema20),
		EMA50:         latestNonNaN(ema50),
		ATR3:          latestNonNaN(atr3),
		ATR14:         latestNonNaN(atr14),
		CurrentVolume: klines[len(klines)-1].Volume,
		AverageVolume: volumeSum / float64(avgWindow),
		MACDValues:    lastN(macd, intradaySeriesLength),
		RSI14Values:   lastN(rsi14, intradaySeriesLength),
	}
	return data
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

func extractCloses(klines []Kline) []float64 {
	out := make([]float64, len(klines))
	for i, k := range klines {
		out[i] = k.Close
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
	return 0
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
