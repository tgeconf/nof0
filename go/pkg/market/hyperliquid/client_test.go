package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCalculateEMA(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 6}
	result := CalculateEMA(data, 3)
	require.Len(t, result, len(data))
	require.True(t, mathIsNaN(result[0]))
	require.True(t, mathIsNaN(result[1]))
	require.InDelta(t, 2.0, result[2], 1e-9)
	require.InDelta(t, 3.0, result[3], 1e-9)
	require.InDelta(t, 4.0, result[4], 1e-9)
	require.InDelta(t, 5.0, result[5], 1e-9)
}

func TestCalculateMACD(t *testing.T) {
	closes := []float64{100, 101, 102, 103, 105, 107, 106, 108, 110, 111, 112, 115, 117, 119, 118, 120, 121, 123, 125, 124, 126, 127, 129, 130, 132, 133, 134, 135, 136, 138, 139, 141, 140, 142, 144, 143, 145, 147, 149, 148, 150, 151, 149, 148, 150, 152, 151, 153, 154, 156, 155, 157, 158, 160, 161, 159, 158, 157, 159, 160}
	macd, signal, hist := CalculateMACD(closes)
	require.Len(t, macd, len(closes))
	require.Len(t, signal, len(closes))
	require.Len(t, hist, len(closes))

	last := len(closes) - 1
	require.InDelta(t, 5.582947, macd[last], 1e-6)
	require.InDelta(t, 6.307087, signal[last], 1e-6)
	require.InDelta(t, -0.724141, hist[last], 1e-6)
}

func TestCalculateRSI(t *testing.T) {
	closes := []float64{100, 101, 102, 103, 105, 107, 106, 108, 110, 111, 112, 115, 117, 119, 118, 120, 121, 123, 125, 124, 126, 127, 129, 130, 132, 133, 134, 135, 136, 138, 139, 141, 140, 142, 144, 143, 145, 147, 149, 148, 150, 151, 149, 148, 150, 152, 151, 153, 154, 156, 155, 157, 158, 160, 161, 159, 158, 157, 159, 160}
	rsi := CalculateRSI(closes, 14)
	require.Len(t, rsi, len(closes))
	require.InDelta(t, 73.084185, rsi[len(rsi)-1], 1e-6)
}

func TestCalculateATR(t *testing.T) {
	base := int64(1_700_000_000_000)
	var klines []Kline
	closes := []float64{100, 101, 102, 104, 103, 105, 107, 106, 108, 110, 112, 111, 113, 115, 114, 116, 118, 117, 119, 121}
	for i, close := range closes {
		klines = append(klines, Kline{
			OpenTime:  base + int64(i)*180_000,
			Open:      close - 0.5,
			High:      close + 1.5,
			Low:       close - 1.5,
			Close:     close,
			Volume:    100 + float64(i),
			CloseTime: base + int64(i+1)*180_000 - 1,
		})
	}
	atr := CalculateATR(klines, 14)
	require.Len(t, atr, len(klines))
	require.InDelta(t, 3.326525, atr[len(atr)-1], 1e-6)
}

func TestGetKlines(t *testing.T) {
	server, client := newMockHyperliquidServer(t)
	defer server.Close()

	ctx := context.Background()
	klines, err := client.GetKlines(ctx, "BTC", "3m", 20)
	require.NoError(t, err)
	require.Len(t, klines, 20)
	require.True(t, klines[0].OpenTime < klines[len(klines)-1].OpenTime)
	require.InDelta(t, 131.0, klines[0].Close, 1e-9)
	require.InDelta(t, 150.0, klines[len(klines)-1].Close, 1e-9)
}

func TestGetMarketInfo(t *testing.T) {
	server, client := newMockHyperliquidServer(t)
	defer server.Close()

	ctx := context.Background()
	info, err := client.GetMarketInfo(ctx, "btc")
	require.NoError(t, err)
	require.Equal(t, "BTC", info.Symbol)
	require.InDelta(t, 150.0, info.MidPrice, 1e-9)
	require.InDelta(t, 0.000125, info.FundingRate, 1e-9)
	require.InDelta(t, 150.0, info.OpenInterest, 1e-9)
}

func TestGetMarketData(t *testing.T) {
	server, client := newMockHyperliquidServer(t)
	defer server.Close()

	ctx := context.Background()
	data, err := client.GetMarketData(ctx, "BTCUSDT")
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Equal(t, "BTC", data.Symbol)
	require.InDelta(t, 150.0, data.CurrentPrice, 1e-9)
	require.InDelta(t, 15.384615, data.PriceChange1h, 1e-6)
	require.InDelta(t, 0.671141, data.PriceChange4h, 1e-6)
	require.NotNil(t, data.OpenInterest)
	require.InDelta(t, 150.0, data.OpenInterest.Latest, 1e-9)
	require.NotNil(t, data.IntradaySeries)
	require.Len(t, data.IntradaySeries.MidPrices, 10)
	require.Len(t, data.IntradaySeries.EMA20Values, 10)
	require.NotNil(t, data.LongerTermContext)
	require.Len(t, data.LongerTermContext.MACDValues, 10)
	require.InDelta(t, 0.000125, data.FundingRate, 1e-9)
}

func TestGetMarketDataMixedCaseSymbol(t *testing.T) {
	server, client := newMockHyperliquidServer(t)
	defer server.Close()

	ctx := context.Background()
	data, err := client.GetMarketData(ctx, "KPEPEUSDT")
	require.NoError(t, err)
	require.Equal(t, "kPEPE", data.Symbol)
	require.InDelta(t, 0.00095, data.CurrentPrice, 1e-9)
	require.NotNil(t, data.IntradaySeries)
	require.NotNil(t, data.LongerTermContext)
}

func TestGetKlinesMixedCaseSymbol(t *testing.T) {
	server, client := newMockHyperliquidServer(t)
	defer server.Close()

	ctx := context.Background()
	klines, err := client.GetKlines(ctx, "KPEPE", "3m", 20)
	require.NoError(t, err)
	require.Len(t, klines, 20)
	require.InDelta(t, 0.001, klines[0].Close, 1e-6)
	require.InDelta(t, 0.00119, klines[len(klines)-1].Close, 1e-6)
}

// --- helpers ---

func newMockHyperliquidServer(t *testing.T) (*httptest.Server, *Client) {
	t.Helper()

	intradaySeries := map[string][]float64{
		"BTC":   makeSequence(111.0, 150.0, 1.0),
		"kPEPE": makeSequence(0.00080, 0.00119, 0.00001),
	}
	longerSeries := map[string][]float64{
		"BTC":   makeSequence(91.0, 150.0, 1.0),
		"kPEPE": makeSequence(0.00060, 0.00119, 0.00001),
	}

	candlePayload := make(map[string]map[string][]map[string]interface{})
	for symbol := range intradaySeries {
		candlePayload[symbol] = map[string][]map[string]interface{}{
			"3m": buildCandlePayload(symbol, "3m", 180_000, intradaySeries[symbol]),
			"4h": buildCandlePayload(symbol, "4h", int64(4*time.Hour/time.Millisecond), longerSeries[symbol]),
		}
	}

	metaPayload := []interface{}{
		map[string]interface{}{
			"universe": []map[string]interface{}{
				{"name": "BTC"},
				{"name": "kPEPE"},
			},
		},
		[]map[string]interface{}{
			{
				"funding":      "0.000125",
				"openInterest": "150",
				"prevDayPx":    "149.5",
				"dayNtlVlm":    "2500000",
				"dayBaseVlm":   "1234.56",
				"premium":      "0.0001",
				"oraclePx":     "149.7",
				"markPx":       "150.2",
				"midPx":        "150.0",
				"impactPxs":    []string{"149.9", "150.1"},
			},
			{
				"funding":      "0.000045",
				"openInterest": "9876.54",
				"prevDayPx":    "0.00094",
				"dayNtlVlm":    "85234.1234",
				"dayBaseVlm":   "123.456",
				"premium":      "0.00002",
				"oraclePx":     "0.00093",
				"markPx":       "0.00095",
				"midPx":        "0.00095",
				"impactPxs":    []string{"0.00094", "0.00096"},
			},
		},
	}

	allMids := map[string]string{
		"BTC":   "150",
		"kPEPE": "0.00095",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req InfoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		switch req.Type {
		case "candleSnapshot":
			params, _ := req.Req.(map[string]interface{})
			interval, _ := params["interval"].(string)
			coin := fmt.Sprintf("%v", params["coin"])
			payloads, ok := candlePayload[coin]
			if !ok {
				for canonical, data := range candlePayload {
					if strings.EqualFold(canonical, coin) {
						payloads = data
						ok = true
						break
					}
				}
			}
			if !ok {
				http.Error(w, "coin not mocked", http.StatusBadRequest)
				return
			}
			data, ok := payloads[interval]
			if !ok {
				http.Error(w, "interval not mocked", http.StatusBadRequest)
				return
			}
			writeJSON(w, data)
		case "metaAndAssetCtxs":
			writeJSON(w, metaPayload)
		case "allMids":
			writeJSON(w, allMids)
		default:
			http.Error(w, "unsupported type", http.StatusBadRequest)
		}
	}))

	client := NewClient(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithMaxRetries(0),
	)

	return server, client
}

func buildCandlePayload(symbol, interval string, stepMillis int64, closes []float64) []map[string]interface{} {
	base := int64(1_700_000_000_000)
	payload := make([]map[string]interface{}, len(closes))
	for i, close := range closes {
		delta := 1.0
		if close < 1 {
			delta = close * 0.05
			if delta == 0 {
				delta = 0.0001
			}
		}
		high := close + delta
		low := close - delta
		if low < 0 {
			low = 0
		}
		open := close - delta/2
		payload[i] = map[string]interface{}{
			"t": base + int64(i)*stepMillis,
			"T": base + int64(i+1)*stepMillis - 1,
			"s": symbol,
			"i": interval,
			"o": formatFloat(open),
			"c": formatFloat(close),
			"h": formatFloat(high),
			"l": formatFloat(low),
			"v": formatFloat(100 + float64(i)),
		}
	}
	return payload
}

func formatFloat(val float64) string {
	return strconv.FormatFloat(val, 'f', -1, 64)
}

func makeSequence(start, end, step float64) []float64 {
	var out []float64
	for v := start; v <= end+1e-9; v += step {
		out = append(out, v)
	}
	return out
}

func writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func mathIsNaN(f float64) bool {
	return f != f
}
