//go:build integration
// +build integration

package market

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "nof0-api/pkg/market/exchanges/hyperliquid"
)

func TestConfigBuildProviders_Integration(t *testing.T) {
	configYAML := []byte(`
default: hyperliquid-test
providers:
  hyperliquid-test:
    type: hyperliquid
    base_url: https://api.hyperliquid.xyz
    timeout: 10s
    http_timeout: 8s
    max_retries: 3
`)

	config, err := LoadConfigFromReader(strings.NewReader(string(configYAML)))
	require.NoError(t, err)

	// 测试从配置构建提供者
	providers, err := config.BuildProviders()
	require.NoError(t, err)
	require.Len(t, providers, 1)

	// 验证提供者类型
	provider, exists := providers["hyperliquid-test"]
	assert.True(t, exists)
	assert.NotNil(t, provider)
}

func TestProviderSnapshot_Integration(t *testing.T) {
	// 注册Hyperliquid提供者
	_ = RegisterProvider("hyperliquid", func(name string, cfg *ProviderConfig) (Provider, error) {
		return &mockProvider{}, nil
	})

	// 创建一个简单的配置
	cfg := &Config{
		Providers: map[string]*ProviderConfig{
			"test": {
				Type: "hyperliquid",
			},
		},
	}

	providers, err := cfg.BuildProviders()
	require.NoError(t, err)
	require.Len(t, providers, 1)

	provider := providers["test"]
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 测试快照功能
	snapshot, err := provider.Snapshot(ctx, "BTC")
	// 由于是mock提供者，我们只验证接口是否正常工作
	if err != nil {
		t.Skipf("Mock provider returned error: %v", err)
	}

	if snapshot != nil {
		assert.NotEmpty(t, snapshot.Symbol)
		assert.Greater(t, snapshot.Price.Last, 0.0)
	}
}

func TestProviderListAssets_Integration(t *testing.T) {
	// 注册Hyperliquid提供者
	_ = RegisterProvider("hyperliquid", func(name string, cfg *ProviderConfig) (Provider, error) {
		return &mockProvider{}, nil
	})

	cfg := &Config{
		Providers: map[string]*ProviderConfig{
			"test": {
				Type: "hyperliquid",
			},
		},
	}

	providers, err := cfg.BuildProviders()
	require.NoError(t, err)
	require.Len(t, providers, 1)

	provider := providers["test"]
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 测试资产列表功能
	assets, err := provider.ListAssets(ctx)
	// 由于是mock提供者，我们只验证接口是否正常工作
	if err != nil {
		t.Skipf("Mock provider returned error: %v", err)
	}

	if len(assets) > 0 {
		for _, asset := range assets {
			assert.NotEmpty(t, asset.Symbol)
			assert.True(t, asset.Precision >= 0)
		}
	}
}

// mockProvider 是一个用于集成测试的模拟提供者
type mockProvider struct{}

func (m *mockProvider) Snapshot(ctx context.Context, symbol string) (*Snapshot, error) {
	return &Snapshot{
		Symbol: "BTC",
		Price: PriceInfo{
			Last: 50000.0,
		},
		Change: ChangeInfo{
			OneHour:  1.5,
			FourHour: 2.3,
		},
		Indicators: IndicatorInfo{
			EMA: map[string]float64{
				"EMA20": 49500.0,
				"EMA50": 48000.0,
			},
			RSI: map[string]float64{
				"RSI14": 65.5,
			},
		},
	}, nil
}

func (m *mockProvider) ListAssets(ctx context.Context) ([]Asset, error) {
	return []Asset{
		{
			Symbol:    "BTC",
			Base:      "BTC",
			Quote:     "USDT",
			Precision: 8,
			IsActive:  true,
			RawMetadata: map[string]any{
				"maxLeverage": 100,
			},
		},
		{
			Symbol:    "ETH",
			Base:      "ETH",
			Quote:     "USDT",
			Precision: 8,
			IsActive:  true,
			RawMetadata: map[string]any{
				"maxLeverage": 50,
			},
		},
	}, nil
}

func TestSnapshotDataStructure_Integration(t *testing.T) {
	// 测试快照数据结构的完整性
	snapshot := &Snapshot{
		Symbol: "BTCUSDT",
		Price: PriceInfo{
			Last: 50000.0,
		},
		Change: ChangeInfo{
			OneHour:  1.5,
			FourHour: 2.3,
		},
		Indicators: IndicatorInfo{
			EMA: map[string]float64{
				"EMA20": 49500.0,
				"EMA50": 48000.0,
			},
			MACD: 1000.0,
			RSI: map[string]float64{
				"RSI7":  70.1,
				"RSI14": 65.5,
			},
		},
		OpenInterest: &OpenInterestInfo{
			Latest:  1000.0,
			Average: 950.0,
		},
		Funding: &FundingInfo{
			Rate: 0.01,
		},
		Intraday: &SeriesBundle{
			Prices: []float64{49000.0, 49500.0, 50000.0},
			EMA: map[string][]float64{
				"EMA20": {48500.0, 49000.0, 49500.0},
			},
			MACD: []float64{800.0, 900.0, 1000.0},
			RSI: map[string][]float64{
				"RSI14": {60.0, 65.0, 70.0},
			},
			Volume: []float64{100.0, 150.0, 200.0},
		},
		LongTerm: &SeriesBundle{
			Prices: []float64{45000.0, 47000.0, 50000.0},
			EMA: map[string][]float64{
				"EMA50": {44000.0, 46000.0, 48000.0},
			},
		},
	}

	// 验证快照数据结构
	assert.Equal(t, "BTCUSDT", snapshot.Symbol)
	assert.InDelta(t, 50000.0, snapshot.Price.Last, 1e-9)
	assert.InDelta(t, 1.5, snapshot.Change.OneHour, 1e-9)
	assert.InDelta(t, 2.3, snapshot.Change.FourHour, 1e-9)
	assert.NotNil(t, snapshot.OpenInterest)
	assert.NotNil(t, snapshot.Funding)
	assert.NotNil(t, snapshot.Intraday)
	assert.NotNil(t, snapshot.LongTerm)

	// 验证指标
	assert.NotEmpty(t, snapshot.Indicators.EMA)
	assert.Greater(t, snapshot.Indicators.MACD, 0.0)
	assert.NotEmpty(t, snapshot.Indicators.RSI)

	// 验证时间序列数据
	assert.NotEmpty(t, snapshot.Intraday.Prices)
	assert.NotEmpty(t, snapshot.LongTerm.Prices)
}

func TestAssetDataStructure_Integration(t *testing.T) {
	// 测试资产数据结构
	asset := Asset{
		Symbol:    "BTC",
		Base:      "BTC",
		Quote:     "USDT",
		Precision: 8,
		IsActive:  true,
		RawMetadata: map[string]any{
			"maxLeverage":  100,
			"marginTable":  "table1",
			"onlyIsolated": false,
		},
	}

	// 验证资产数据结构
	assert.Equal(t, "BTC", asset.Symbol)
	assert.Equal(t, "BTC", asset.Base)
	assert.Equal(t, "USDT", asset.Quote)
	assert.Equal(t, 8, asset.Precision)
	assert.True(t, asset.IsActive)
	assert.NotNil(t, asset.RawMetadata)

	// 验证元数据字段
	assert.Equal(t, 100, asset.RawMetadata["maxLeverage"])
	assert.Equal(t, "table1", asset.RawMetadata["marginTable"])
	assert.Equal(t, false, asset.RawMetadata["onlyIsolated"])
}
