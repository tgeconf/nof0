//go:build integration
// +build integration

package indicators

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEMACalculation_Integration(t *testing.T) {
	// 测试EMA计算的完整性
	prices := []float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110}
	period := 5

	ema := EMA(prices, period)
	require.Len(t, ema, len(prices))

	// 验证EMA值的合理性
	validCount := 0
	for i, value := range ema {
		if !isNaN(value) {
			validCount++
			if i > 0 {
				prev := ema[i-1]
				if !isNaN(prev) {
					// EMA应该相对平滑，不会剧烈波动
					diff := value - prev
					assert.InDelta(t, 0, diff, 10.0, "EMA should not have extreme jumps at index %d", i)
				}
			}
		}
	}

	// 应该有合理的有效值数量
	assert.Greater(t, validCount, 0)
}

func TestMACDCalculation_Integration(t *testing.T) {
	// 测试MACD计算的完整性
	prices := []float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115}

	macd, signal, hist := MACD(prices)
	require.Len(t, macd, len(prices))
	require.Len(t, signal, len(prices))
	require.Len(t, hist, len(prices))

	// 验证MACD计算的正确性
	validCount := 0
	for i := range prices {
		macdVal := macd[i]
		signalVal := signal[i]
		histVal := hist[i]

		if !isNaN(macdVal) && !isNaN(signalVal) && !isNaN(histVal) {
			validCount++
			// 直方图应该是MACD减去信号线
			assert.InDelta(t, histVal, macdVal-signalVal, 1e-6, "Histogram should equal MACD - Signal at index %d", i)
		}
	}

	// 应该有合理的有效值数量
	assert.Greater(t, validCount, 0, "MACD calculation should have valid values")
}

func TestRSICalculation_Integration(t *testing.T) {
	// 测试RSI计算的完整性
	prices := []float64{100, 102, 101, 103, 104, 102, 105, 106, 104, 107, 108, 106, 109, 110, 108}
	period := 14

	rsi := RSI(prices, period)
	require.Len(t, rsi, len(prices))

	// 验证RSI值在0-100范围内
	validCount := 0
	for _, value := range rsi {
		if !isNaN(value) {
			validCount++
			assert.GreaterOrEqual(t, value, 0.0, "RSI should be >= 0")
			assert.LessOrEqual(t, value, 100.0, "RSI should be <= 100")
		}
	}

	// 应该有合理的有效值数量
	assert.Greater(t, validCount, 0)
}

func TestATRCalculation_Integration(t *testing.T) {
	// 测试ATR计算的完整性
	klines := []Kline{
		{High: 105, Low: 95, Close: 100},
		{High: 108, Low: 98, Close: 103},
		{High: 106, Low: 96, Close: 101},
		{High: 109, Low: 99, Close: 104},
		{High: 107, Low: 97, Close: 102},
	}
	period := 3

	atr := ATR(klines, period)
	require.Len(t, atr, len(klines))

	// 验证ATR值的合理性
	validCount := 0
	for _, value := range atr {
		if !isNaN(value) {
			validCount++
			assert.Greater(t, value, 0.0, "ATR should be positive")
		}
	}

	// 应该有合理的有效值数量
	assert.Greater(t, validCount, 0)
}

func TestIndicatorConsistency_Integration(t *testing.T) {
	// 测试不同指标之间的一致性
	prices := []float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110}

	// 计算不同周期的EMA
	ema20 := EMA(prices, 20)
	ema50 := EMA(prices, 50)

	// 计算RSI
	rsi14 := RSI(prices, 14)

	// 验证指标之间的逻辑关系
	validEMACount := 0
	validRSICount := 0
	for i := range prices {
		if !isNaN(ema20[i]) && !isNaN(ema50[i]) {
			validEMACount++
		}
		if !isNaN(rsi14[i]) {
			validRSICount++
		}
	}

	// 验证长周期EMA应该更平滑
	if validEMACount > 0 {
		// EMA50的变化应该比EMA20更平缓
		var ema20Changes, ema50Changes []float64
		for i := 1; i < len(prices); i++ {
			if !isNaN(ema20[i]) && !isNaN(ema20[i-1]) {
				ema20Changes = append(ema20Changes, ema20[i]-ema20[i-1])
			}
			if !isNaN(ema50[i]) && !isNaN(ema50[i-1]) {
				ema50Changes = append(ema50Changes, ema50[i]-ema50[i-1])
			}
		}

		if len(ema20Changes) > 0 && len(ema50Changes) > 0 {
			// 计算平均变化幅度
			avg20Change := 0.0
			avg50Change := 0.0
			for _, change := range ema20Changes {
				avg20Change += abs(change)
			}
			for _, change := range ema50Changes {
				avg50Change += abs(change)
			}
			avg20Change /= float64(len(ema20Changes))
			avg50Change /= float64(len(ema50Changes))

			// EMA50的平均变化应该小于或等于EMA20的平均变化
			assert.LessOrEqual(t, avg50Change, avg20Change, "EMA50 should be smoother than EMA20")
		}
	}

	// RSI应该在合理范围内
	assert.Greater(t, validRSICount, 0, "RSI should have valid values")
}

func TestEdgeCases_Integration(t *testing.T) {
	// 测试边界情况
	emptyPrices := []float64{}
	zeroPeriod := 0
	negativePeriod := -1
	singlePrice := []float64{100}

	// 测试空数据
	assert.Empty(t, EMA(emptyPrices, 5))
	assert.Empty(t, RSI(emptyPrices, 5))
	assert.Empty(t, ATR(nil, 5))

	// 测试无效周期
	assert.Empty(t, EMA(singlePrice, zeroPeriod))
	assert.Empty(t, EMA(singlePrice, negativePeriod))
	assert.Empty(t, RSI(singlePrice, zeroPeriod))
	assert.Empty(t, RSI(singlePrice, negativePeriod))

	// 测试单个价格
	ema := EMA(singlePrice, 5)
	assert.Empty(t, ema)

	rsi := RSI(singlePrice, 5)
	assert.Empty(t, rsi)
}

func TestPerformance_Integration(t *testing.T) {
	// 测试大量数据的性能
	largePrices := make([]float64, 1000)
	for i := range largePrices {
		largePrices[i] = float64(1000 + i)
	}

	// 测试EMA性能
	startTime := time.Now()
	ema := EMA(largePrices, 20)
	emaTime := time.Since(startTime)

	// 测试RSI性能
	startTime = time.Now()
	rsi := RSI(largePrices, 14)
	rsiTime := time.Since(startTime)

	// 验证计算结果
	assert.Len(t, ema, len(largePrices))
	assert.Len(t, rsi, len(largePrices))

	// 验证性能（应该在合理时间内完成）
	assert.Less(t, emaTime, 100*time.Millisecond, "EMA calculation should be fast")
	assert.Less(t, rsiTime, 100*time.Millisecond, "RSI calculation should be fast")
}

func TestNaNHandling_Integration(t *testing.T) {
	// 测试NaN值处理
	pricesWithNaN := []float64{100, 101, math.NaN(), 103, 104, 105}

	ema := EMA(pricesWithNaN, 3)
	rsi := RSI(pricesWithNaN, 3)

	// 验证NaN值的处理
	assert.Len(t, ema, len(pricesWithNaN))
	assert.Len(t, rsi, len(pricesWithNaN))

	// 检查是否有合理的有效值
	validEMACount := 0
	validRSICount := 0
	for i := range pricesWithNaN {
		if !isNaN(ema[i]) {
			validEMACount++
		}
		if !isNaN(rsi[i]) {
			validRSICount++
		}
	}

	// 应该有一些有效值
	assert.Greater(t, validEMACount, 0)
	assert.Greater(t, validRSICount, 0)
}

// 辅助函数
func isNaN(f float64) bool {
	return math.IsNaN(f)
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
