package indicators

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEMA(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 6}
	result := EMA(data, 3)
	require.Len(t, result, len(data))
	require.True(t, math.IsNaN(result[0]))
	require.True(t, math.IsNaN(result[1]))
	require.InDelta(t, 2.0, result[2], 1e-9)
	require.InDelta(t, 3.0, result[3], 1e-9)
	require.InDelta(t, 4.0, result[4], 1e-9)
	require.InDelta(t, 5.0, result[5], 1e-9)
}

func TestMACD(t *testing.T) {
	closes := []float64{100, 101, 102, 103, 105, 107, 106, 108, 110, 111, 112, 115, 117, 119, 118, 120, 121, 123, 125, 124, 126, 127, 129, 130, 132, 133, 134, 135, 136, 138, 139, 141, 140, 142, 144, 143, 145, 147, 149, 148, 150, 151, 149, 148, 150, 152, 151, 153, 154, 156, 155, 157, 158, 160, 161, 159, 158, 157, 159, 160}
	macd, signal, hist := MACD(closes)
	require.Len(t, macd, len(closes))
	require.Len(t, signal, len(closes))
	require.Len(t, hist, len(closes))

	last := len(closes) - 1
	require.InDelta(t, 5.582947, macd[last], 1e-6)
	require.InDelta(t, 6.307087, signal[last], 1e-6)
	require.InDelta(t, -0.724141, hist[last], 1e-6)
}

func TestRSI(t *testing.T) {
	closes := []float64{100, 101, 102, 103, 105, 107, 106, 108, 110, 111, 112, 115, 117, 119, 118, 120, 121, 123, 125, 124, 126, 127, 129, 130, 132, 133, 134, 135, 136, 138, 139, 141, 140, 142, 144, 143, 145, 147, 149, 148, 150, 151, 149, 148, 150, 152, 151, 153, 154, 156, 155, 157, 158, 160, 161, 159, 158, 157, 159, 160}
	rsi := RSI(closes, 14)
	require.Len(t, rsi, len(closes))
	require.InDelta(t, 73.084185, rsi[len(rsi)-1], 1e-6)
}

func TestATR(t *testing.T) {
	closes := []float64{100, 101, 102, 104, 103, 105, 107, 106, 108, 110, 112, 111, 113, 115, 114, 116, 118, 117, 119, 121}
	klines := make([]Kline, len(closes))
	for i, close := range closes {
		klines[i] = Kline{
			High:  close + 1.5,
			Low:   close - 1.5,
			Close: close,
		}
	}

	atr := ATR(klines, 14)
	require.Len(t, atr, len(klines))
	require.InDelta(t, 3.326525, atr[len(atr)-1], 1e-6)
}
