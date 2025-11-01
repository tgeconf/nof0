package hyperliquid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPositions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "ok",
				"data": {
					"assetPositions": [
						{
							"coin": "BTC",
							"entryPx": "50000.0",
							"leverage": {"type": "cross", "value": 5},
							"liquidationPx": "45000.0",
							"positionValue": "5000.0",
							"returnOnEquity": "0.05",
							"szi": "0.1",
							"unrealizedPnl": "100.0"
						}
					],
					"marginSummary": {
						"accountValue": "10000.0",
						"totalMarginUsed": "500.0",
						"totalNtlPos": "2000.0",
						"totalRawUsd": "9500.0"
					},
					"crossMarginSummary": {
						"accountValue": "10000.0",
						"totalMarginUsed": "500.0",
						"totalNtlPos": "2000.0",
						"totalRawUsd": "9500.0"
					}
				}
			}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		positions, err := client.GetPositions(context.Background())
		assert.NoError(t, err)
		assert.Len(t, positions, 1)
		assert.Equal(t, "BTC", positions[0].Coin)
		assert.Equal(t, "0.1", positions[0].Szi)
	})

	t.Run("empty_positions", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "ok",
				"data": {
					"assetPositions": [],
					"marginSummary": {
						"accountValue": "10000.0",
						"totalMarginUsed": "0",
						"totalNtlPos": "0",
						"totalRawUsd": "10000.0"
					},
					"crossMarginSummary": {
						"accountValue": "10000.0",
						"totalMarginUsed": "0",
						"totalNtlPos": "0",
						"totalRawUsd": "10000.0"
					}
				}
			}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		positions, err := client.GetPositions(context.Background())
		assert.NoError(t, err)
		assert.Empty(t, positions)
	})
}

func TestClosePosition(t *testing.T) {
	t.Run("position_not_found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "ok",
				"data": {
					"assetPositions": [],
					"marginSummary": {
						"accountValue": "10000.0",
						"totalMarginUsed": "0",
						"totalNtlPos": "0",
						"totalRawUsd": "10000.0"
					},
					"crossMarginSummary": {
						"accountValue": "10000.0",
						"totalMarginUsed": "0",
						"totalNtlPos": "0",
						"totalRawUsd": "10000.0"
					}
				}
			}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		err = client.ClosePosition(context.Background(), "BTC")
		assert.NoError(t, err) // No error when position doesn't exist
	})
}

func TestUpdateLeverage(t *testing.T) {
	t.Run("invalid_leverage_zero", func(t *testing.T) {
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)

		err = client.UpdateLeverage(context.Background(), 1, true, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "leverage must be positive")
	})

	t.Run("invalid_leverage_negative", func(t *testing.T) {
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)

		err = client.UpdateLeverage(context.Background(), 1, true, -5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "leverage must be positive")
	})
}

func TestDecimalsForString(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected int
	}{
		{"integer", "100", 0},
		{"one_decimal", "100.5", 1},
		{"two_decimals", "100.52", 2},
		{"many_decimals", "100.123456", 6},
		{"no_decimal_point", "12345", 0},
		{"trailing_zeros", "100.500", 3},
		{"with_spaces", "  100.5  ", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decimalsForString(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrimTrailingZeros(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"no_zeros", "100.5", "100.5"},
		{"trailing_zeros", "100.500", "100.5"},
		{"all_zeros_after_decimal", "100.000", "100"},
		{"zero", "0.000", "0"},
		{"empty", "", ""},
		{"decimal_point_only", "100.", "100"},
		{"multiple_trailing_zeros", "100.123000", "100.123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimTrailingZeros(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}
