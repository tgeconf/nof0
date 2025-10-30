package indicators

import "math"

// EMA produces the exponential moving average for the supplied prices.
func EMA(prices []float64, period int) []float64 {
	if period <= 0 || len(prices) == 0 {
		return []float64{}
	}
	result := make([]float64, len(prices))
	for i := range result {
		result[i] = math.NaN()
	}
	if len(prices) < period {
		return result
	}
	multiplier := 2.0 / float64(period+1)

	start := -1
	var seed float64
	for i := period - 1; i < len(prices); i++ {
		windowValid := true
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			if math.IsNaN(prices[j]) {
				windowValid = false
				break
			}
			sum += prices[j]
		}
		if windowValid {
			start = i
			seed = sum / float64(period)
			break
		}
	}
	if start == -1 {
		return result
	}
	result[start] = seed

	for i := start + 1; i < len(prices); i++ {
		if math.IsNaN(prices[i]) {
			result[i] = result[i-1]
			continue
		}
		prev := result[i-1]
		if math.IsNaN(prev) {
			prev = seed
		}
		result[i] = (prices[i]-prev)*multiplier + prev
	}
	return result
}

// MACD returns MACD, signal, and histogram series.
func MACD(prices []float64) ([]float64, []float64, []float64) {
	if len(prices) == 0 {
		return []float64{}, []float64{}, []float64{}
	}
	ema12 := EMA(prices, 12)
	ema26 := EMA(prices, 26)

	macd := make([]float64, len(prices))
	for i := range prices {
		if math.IsNaN(ema12[i]) || math.IsNaN(ema26[i]) {
			macd[i] = math.NaN()
		} else {
			macd[i] = ema12[i] - ema26[i]
		}
	}

	signal := EMA(macd, 9)
	hist := make([]float64, len(prices))
	for i := range hist {
		if math.IsNaN(macd[i]) || math.IsNaN(signal[i]) {
			hist[i] = math.NaN()
		} else {
			hist[i] = macd[i] - signal[i]
		}
	}
	return macd, signal, hist
}

// RSI computes the Relative Strength Index across the supplied prices.
func RSI(prices []float64, period int) []float64 {
	if period <= 0 || len(prices) == 0 {
		return []float64{}
	}
	rsi := make([]float64, len(prices))
	for i := range rsi {
		rsi[i] = math.NaN()
	}
	if len(prices) <= period {
		return rsi
	}

	var gainSum, lossSum float64
	for i := 1; i <= period; i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gainSum += change
		} else {
			lossSum -= change
		}
	}

	avgGain := gainSum / float64(period)
	avgLoss := lossSum / float64(period)

	rsi[period] = computeRSI(avgGain, avgLoss)

	for i := period + 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		gain := math.Max(change, 0)
		loss := math.Max(-change, 0)

		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)

		rsi[i] = computeRSI(avgGain, avgLoss)
	}
	return rsi
}

// ATR computes the Average True Range across the Kline series.
func ATR(klines []Kline, period int) []float64 {
	if period <= 0 || len(klines) == 0 {
		return []float64{}
	}
	tr := make([]float64, len(klines))
	for i := range klines {
		if i == 0 {
			tr[i] = klines[i].High - klines[i].Low
			continue
		}
		highLow := klines[i].High - klines[i].Low
		highClose := math.Abs(klines[i].High - klines[i-1].Close)
		lowClose := math.Abs(klines[i].Low - klines[i-1].Close)
		tr[i] = math.Max(highLow, math.Max(highClose, lowClose))
	}
	return EMA(tr, period)
}

// Kline represents OHLCV input for ATR calculations.
type Kline struct {
	High  float64
	Low   float64
	Close float64
}

func computeRSI(avgGain, avgLoss float64) float64 {
	switch {
	case avgLoss == 0 && avgGain == 0:
		return 50.0
	case avgLoss == 0:
		return 100.0
	case avgGain == 0:
		return 0.0
	default:
		rs := avgGain / avgLoss
		return 100.0 - (100.0 / (1.0 + rs))
	}
}
