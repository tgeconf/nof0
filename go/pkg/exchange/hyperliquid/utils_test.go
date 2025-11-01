package hyperliquid

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetAssetIndex(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"universe": [
					{"name": "BTC", "szDecimals": 5},
					{"name": "ETH", "szDecimals": 4}
				],
				"assetCtxs": [
					{"markPx": "50000.0"},
					{"markPx": "3000.0"}
				]
			}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		idx, err := client.GetAssetIndex(context.Background(), "BTC")
		assert.NoError(t, err)
		assert.Equal(t, 0, idx)

		idx, err = client.GetAssetIndex(context.Background(), "ETH")
		assert.NoError(t, err)
		assert.Equal(t, 1, idx)
	})

	t.Run("empty_coin", func(t *testing.T) {
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)

		_, err = client.GetAssetIndex(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty coin")
	})
}

func TestGetAssetInfo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"universe": [
					{
						"name": "BTC",
						"szDecimals": 5,
						"maxLeverage": 20.0,
						"marginTableId": 1,
						"onlyIsolated": false,
						"isDelisted": false
					}
				],
				"assetCtxs": [
					{
						"markPx": "50000.0",
						"midPx": "50001.0",
						"oraclePx": "49999.0",
						"impactPxs": ["49950.0", "50050.0"]
					}
				]
			}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		info, err := client.GetAssetInfo(context.Background(), "BTC")
		assert.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, "BTC", info.Name)
		assert.Equal(t, 5, info.SzDecimals)
		assert.Equal(t, 20.0, info.MaxLeverage)
		assert.Equal(t, "50000.0", info.MarkPx)
		assert.Equal(t, "50001.0", info.MidPx)
		assert.Len(t, info.ImpactPxs, 2)
	})

	t.Run("empty_coin", func(t *testing.T) {
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)

		_, err = client.GetAssetInfo(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty coin")
	})
}

func TestRefreshAssetDirectory(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"universe": [
					{"name": "BTC", "szDecimals": 5}
				],
				"assetCtxs": [
					{"markPx": "50000.0"}
				]
			}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		err = client.refreshAssetDirectory(context.Background())
		assert.NoError(t, err)

		// Verify cache was populated
		idx, ok := client.cachedAssetIndex("BTC")
		assert.True(t, ok)
		assert.Equal(t, 0, idx)
	})

	t.Run("empty_universe", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"universe": [], "assetCtxs": []}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		err = client.refreshAssetDirectory(context.Background())
		assert.Error(t, err)
		// The error message can vary, just check there's an error
	})
}

func TestCachedAssetIndex(t *testing.T) {
	client := &Client{
		assetIndex: map[string]int{
			"BTC": 0,
			"ETH": 1,
		},
	}

	t.Run("found", func(t *testing.T) {
		idx, ok := client.cachedAssetIndex("BTC")
		assert.True(t, ok)
		assert.Equal(t, 0, idx)
	})

	t.Run("not_found", func(t *testing.T) {
		_, ok := client.cachedAssetIndex("UNKNOWN")
		assert.False(t, ok)
	})
}

func TestCachedAssetInfo(t *testing.T) {
	client := &Client{
		assetInfo: map[string]AssetInfo{
			"BTC": {
				Name:       "BTC",
				SzDecimals: 5,
				MarkPx:     "50000.0",
			},
		},
	}

	t.Run("found", func(t *testing.T) {
		info, ok := client.cachedAssetInfo("BTC")
		assert.True(t, ok)
		assert.Equal(t, "BTC", info.Name)
		assert.Equal(t, 5, info.SzDecimals)
	})

	t.Run("not_found", func(t *testing.T) {
		_, ok := client.cachedAssetInfo("UNKNOWN")
		assert.False(t, ok)
	})
}

func TestMaybeRefreshAssetDirectory(t *testing.T) {
	t.Run("ttl_zero_no_refresh", func(t *testing.T) {
		client := &Client{
			assetTTL: 0,
		}
		err := client.maybeRefreshAssetDirectory(context.Background())
		assert.NoError(t, err)
	})

	t.Run("ttl_expired", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"universe": [{"name": "BTC", "szDecimals": 5}],
				"assetCtxs": [{"markPx": "50000.0"}]
			}`))
		}))
		defer server.Close()

		baseTime := time.Now()
		client, err := NewClient(
			"0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f",
			false,
			WithAssetCacheTTL(1*time.Second),
			WithClock(func() time.Time { return baseTime.Add(2 * time.Second) }),
		)
		assert.NoError(t, err)
		client.infoURL = server.URL

		// Set last refresh to old time
		client.assetLastRef = baseTime

		err = client.maybeRefreshAssetDirectory(context.Background())
		assert.NoError(t, err)
	})
}

func TestIsFinite(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected bool
	}{
		{"finite", 100.5, true},
		{"zero", 0.0, true},
		{"negative", -100.5, true},
		{"nan", math.NaN(), false},
		{"positive_inf", math.Inf(1), false},
		{"negative_inf", math.Inf(-1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFinite(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatPrice(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"universe": [{"name": "BTC", "szDecimals": 5}],
				"assetCtxs": [{"markPx": "50000.0"}]
			}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		price, err := client.FormatPrice(context.Background(), "BTC", 50123.456)
		assert.NoError(t, err)
		assert.NotEmpty(t, price)
	})

	t.Run("invalid_price_zero", func(t *testing.T) {
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)

		_, err = client.FormatPrice(context.Background(), "BTC", 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid price")
	})

	t.Run("invalid_price_negative", func(t *testing.T) {
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)

		_, err = client.FormatPrice(context.Background(), "BTC", -100)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid price")
	})
}
