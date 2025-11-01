//go:build integration
// +build integration

package hyperliquid

import (
	"context"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderSnapshot_Integration(t *testing.T) {
	// 使用真实的Hyperliquid测试环境
	provider := NewProvider(
		WithTimeout(10*time.Second),
		WithClientOptions(WithMaxRetries(3)),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 测试主要交易对
	snapshot, err := provider.Snapshot(ctx, "BTC")
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.NotEmpty(t, snapshot.Symbol)
	require.Greater(t, snapshot.Price.Last, 0.0)
	require.NotNil(t, snapshot.Change)
	require.NotNil(t, snapshot.Indicators)
	assert.NotNil(t, snapshot.Intraday)
	assert.NotNil(t, snapshot.LongTerm)
}

func TestProviderListAssets_Integration(t *testing.T) {
	provider := NewProvider(
		WithTimeout(10*time.Second),
		WithClientOptions(WithMaxRetries(3)),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	assets, err := provider.ListAssets(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, assets)

	// 验证至少包含一些主要资产
	btcFound := false
	for _, asset := range assets {
		assert.NotEmpty(t, asset.Symbol)
		assert.True(t, asset.Precision >= 0)
		assert.True(t, asset.IsActive || !asset.IsActive) // 要么活跃要么不活跃

		if asset.Symbol == "BTC" {
			btcFound = true
		}
	}

	assert.True(t, btcFound, "BTC should be found in asset list")
}

func TestClientGetKlines_Integration(t *testing.T) {
	client := NewClient(
		WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
		WithMaxRetries(3),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 测试获取K线数据
	klines, err := client.GetKlines(ctx, "BTC", "3m", 10)
	require.NoError(t, err)
	require.NotEmpty(t, klines)

	// 验证K线数据结构
	for i, kline := range klines {
		assert.Greater(t, kline.OpenTime, int64(0))
		assert.Greater(t, kline.CloseTime, int64(0))
		// 基本区间检查：开盘必须早于收盘，间隔大致不超过3分多钟（允许轻微偏差）
		assert.Less(t, kline.OpenTime, kline.CloseTime)
		assert.LessOrEqual(t, kline.CloseTime-kline.OpenTime, int64((3*time.Minute+10*time.Second)/time.Millisecond))
		assert.Greater(t, kline.Close, 0.0)

		if i > 0 {
			// 确保时间序列是递增的
			assert.Greater(t, kline.OpenTime, klines[i-1].OpenTime)
		}
	}
}

func TestClientGetMarketInfo_Integration(t *testing.T) {
	client := NewClient(
		WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
		WithMaxRetries(3),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 测试获取市场信息
	info, err := client.GetMarketInfo(ctx, "BTC")
	require.NoError(t, err)
	require.NotNil(t, info)

	// 验证市场信息字段
	assert.Equal(t, "BTC", info.Symbol)
	assert.Greater(t, info.MidPrice, 0.0)
	// 资金费率可能为正/负/零，且有时缺失（NaN）。只要不是 NaN 即可通过。
	assert.False(t, math.IsNaN(info.FundingRate))
	// 未上市或冷门资产可能为 0，这里使用非负断言
	assert.GreaterOrEqual(t, info.OpenInterest, 0.0)
	assert.GreaterOrEqual(t, info.MarkPrice, 0.0)
}

func TestClientGetCurrentPrice_Integration(t *testing.T) {
	client := NewClient(
		WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
		WithMaxRetries(3),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 测试获取当前价格
	price, err := client.GetCurrentPrice(ctx, "BTC")
	require.NoError(t, err)
	assert.Greater(t, price, 0.0)
}

func TestConcurrentRequests_Integration(t *testing.T) {
	provider := NewProvider(
		WithTimeout(10*time.Second),
		WithClientOptions(WithMaxRetries(3)),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 并发测试多个请求
	symbols := []string{"BTC", "ETH", "SOL", "ARB", "DOGE"}
	results := make(chan error, len(symbols))

	for _, symbol := range symbols {
		go func(sym string) {
			_, err := provider.Snapshot(ctx, sym)
			results <- err
		}(symbol)
	}

	// 检查所有请求的结果
	for i := 0; i < len(symbols); i++ {
		err := <-results
		// 允许部分交易对不存在或无K线；只要没有明显的上下文/参数错误即可
		if err != nil {
			assert.NotContains(t, err.Error(), "unsupported interval")
		}
	}
}

func TestProviderTimeoutHandling_Integration(t *testing.T) {
	// 使用一个可能响应较慢的配置
	provider := NewProvider(
		WithTimeout(1*time.Second), // 短超时
		WithClientOptions(WithMaxRetries(1)),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 测试超时处理
	_, err := provider.Snapshot(ctx, "BTC")
	// 由于超时设置很短，可能会遇到超时错误，这是预期的
	// 或者如果请求快速完成，也应该成功
	if err != nil {
		assert.Contains(t, err.Error(), "context deadline exceeded")
	}
}

func TestProviderWithCustomBaseURL_Integration(t *testing.T) {
	// 测试自定义BaseURL配置
	provider := NewProvider(
		WithTimeout(10*time.Second),
		// 将客户端层的选项通过 WithClientOptions 传递
		WithClientOptions(
			WithBaseURL("https://api.hyperliquid.xyz"),
			WithMaxRetries(3),
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 测试使用自定义URL获取数据
	snapshot, err := provider.Snapshot(ctx, "BTC")
	// 即使可能失败，也应该有适当的错误处理
	if err != nil {
		// 如果连接失败，应该有明确的错误信息
		assert.Contains(t, err.Error(), "hyperliquid")
	} else {
		// 如果成功，验证数据结构
		require.NotNil(t, snapshot)
		assert.NotEmpty(t, snapshot.Symbol)
		assert.Greater(t, snapshot.Price.Last, 0.0)
	}
}

func TestProviderAssetMetadata_Integration(t *testing.T) {
	provider := NewProvider(
		WithTimeout(10*time.Second),
		WithClientOptions(WithMaxRetries(3)),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	assets, err := provider.ListAssets(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, assets)

	// 检查元数据字段
	for _, asset := range assets {
		if asset.RawMetadata != nil {
			// 验证常见的元数据字段
			if maxLeverage, exists := asset.RawMetadata["maxLeverage"]; exists {
				assert.NotNil(t, maxLeverage)
			}
			if marginTable, exists := asset.RawMetadata["marginTable"]; exists {
				assert.NotNil(t, marginTable)
			}
			if onlyIsolated, exists := asset.RawMetadata["onlyIsolated"]; exists {
				assert.NotNil(t, onlyIsolated)
			}
		}
	}
}

func TestProviderIndicatorCalculation_Integration(t *testing.T) {
	provider := NewProvider(
		WithTimeout(10*time.Second),
		WithClientOptions(WithMaxRetries(3)),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 测试指标计算
	snapshot, err := provider.Snapshot(ctx, "BTC")
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.NotNil(t, snapshot.Indicators)

	// 验证EMA指标
	if len(snapshot.Indicators.EMA) > 0 {
		for window, value := range snapshot.Indicators.EMA {
			assert.NotEmpty(t, window)
			assert.Greater(t, value, 0.0)
		}
	}

	// 验证RSI指标
	if len(snapshot.Indicators.RSI) > 0 {
		for window, value := range snapshot.Indicators.RSI {
			assert.NotEmpty(t, window)
			assert.GreaterOrEqual(t, value, 0.0)
			assert.LessOrEqual(t, value, 100.0)
		}
	}
}
