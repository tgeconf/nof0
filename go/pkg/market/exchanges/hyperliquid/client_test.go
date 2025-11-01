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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nof0-api/pkg/market"
)

func TestProviderSnapshot(t *testing.T) {
	server, provider := newMockProvider(t)
	defer server.Close()

	ctx := context.Background()
	snapshot, err := provider.Snapshot(ctx, "BTCUSDT")
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Equal(t, "BTC", snapshot.Symbol)
	require.InDelta(t, 150.0, snapshot.Price.Last, 1e-9)
	require.InDelta(t, 15.384615, snapshot.Change.OneHour, 1e-6)
	require.InDelta(t, 0.671141, snapshot.Change.FourHour, 1e-6)
	require.NotNil(t, snapshot.OpenInterest)
	require.InDelta(t, 150.0, snapshot.OpenInterest.Latest, 1e-9)
	require.NotNil(t, snapshot.Intraday)
	require.NotNil(t, snapshot.LongTerm)
	require.NotEmpty(t, snapshot.Indicators.EMA)
}

func TestProviderSnapshotMixedCase(t *testing.T) {
	server, provider := newMockProvider(t)
	defer server.Close()

	ctx := context.Background()
	snapshot, err := provider.Snapshot(ctx, "kpepeusdt")
	require.NoError(t, err)
	require.Equal(t, "kPEPE", snapshot.Symbol)
	require.InDelta(t, 0.00095, snapshot.Price.Last, 1e-9)
	require.NotNil(t, snapshot.Intraday)
	require.NotNil(t, snapshot.LongTerm)
}

func TestProviderListAssets(t *testing.T) {
	server, provider := newMockProvider(t)
	defer server.Close()

	ctx := context.Background()
	assets, err := provider.ListAssets(ctx)
	require.NoError(t, err)
	require.Len(t, assets, 2)
	require.Contains(t, []string{assets[0].Symbol, assets[1].Symbol}, "BTC")
	require.Contains(t, []string{assets[0].Symbol, assets[1].Symbol}, "kPEPE")
}

func TestClientGetKlines(t *testing.T) {
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

func TestClientGetMarketInfo(t *testing.T) {
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

// TestClientGetMarketInfoErrors tests error handling in GetMarketInfo.
func TestClientGetMarketInfoErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		symbol      string
		wantErr     bool
		errContains string
	}{
		{
			name: "missing mark price",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					payload := []interface{}{
						map[string]interface{}{
							"universe": []map[string]interface{}{
								{"name": "TEST", "szDecimals": 5, "maxLeverage": 40, "marginTableId": 56, "isDelisted": false},
							},
						},
						[]map[string]interface{}{
							{
								"funding":      "0.0001",
								"openInterest": "100",
								"markPx":       "", // Missing mark price
								"midPx":        "100",
							},
						},
					}
					writeJSON(w, payload)
				}))
			},
			symbol:      "TEST",
			wantErr:     true,
			errContains: "missing mark price",
		},
		{
			name: "symbol not found",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					payload := []interface{}{
						map[string]interface{}{"universe": []map[string]interface{}{}},
						[]map[string]interface{}{},
					}
					writeJSON(w, payload)
				}))
			},
			symbol:      "NOTFOUND",
			wantErr:     true,
			errContains: "symbol not found",
		},
		{
			name: "invalid funding rate",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					payload := []interface{}{
						map[string]interface{}{
							"universe": []map[string]interface{}{
								{"name": "BAD", "szDecimals": 5, "maxLeverage": 40, "marginTableId": 56, "isDelisted": false},
							},
						},
						[]map[string]interface{}{
							{
								"funding":      "invalid",
								"openInterest": "100",
								"markPx":       "100",
								"midPx":        "100",
							},
						},
					}
					writeJSON(w, payload)
				}))
			},
			symbol:      "BAD",
			wantErr:     true,
			errContains: "parse funding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			client := NewClient(
				WithBaseURL(server.URL),
				WithHTTPClient(server.Client()),
			)

			ctx := context.Background()
			info, err := client.GetMarketInfo(ctx, tt.symbol)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, info)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, info)
			}
		})
	}
}

func TestListAssetsReflectsDelistedStatus(t *testing.T) {
	server, client := newMockHyperliquidServer(t)
	defer server.Close()

	ctx := context.Background()
	require.NoError(t, client.refreshSymbolDirectory(ctx))
	client.symbolsMu.Lock()
	entry := client.universeMeta["kPEPE"]
	entry.IsDelisted = true
	client.universeMeta["kPEPE"] = entry
	client.symbolsMu.Unlock()

	provider := &Provider{client: client, timeout: defaultProviderTimeout}
	assets := provider.collectAssets()
	var pepe market.Asset
	for _, asset := range assets {
		if asset.Symbol == "kPEPE" {
			pepe = asset
			break
		}
	}
	require.False(t, pepe.IsActive)
}

// --- helpers ---

func newMockProvider(t *testing.T) (*httptest.Server, *Provider) {
	t.Helper()
	server, client := newMockHyperliquidServer(t)
	provider := &Provider{
		client:  client,
		timeout: defaultProviderTimeout,
	}
	return server, provider
}
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
				{"name": "BTC", "szDecimals": 5, "maxLeverage": 40, "marginTableId": 56, "isDelisted": false},
				{"name": "kPEPE", "szDecimals": 0, "maxLeverage": 10, "marginTableId": 52, "isDelisted": false},
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

// TestNewProvider tests the NewProvider constructor and options.
func TestNewProvider(t *testing.T) {
	tests := []struct {
		name         string
		opts         []ProviderOption
		wantTimeout  time.Duration
		validateFunc func(*testing.T, *Provider)
	}{
		{
			name:        "default configuration",
			opts:        nil,
			wantTimeout: defaultProviderTimeout,
		},
		{
			name:        "custom timeout",
			opts:        []ProviderOption{WithTimeout(5 * time.Second)},
			wantTimeout: 5 * time.Second,
		},
		{
			name: "with client options",
			opts: []ProviderOption{
				WithClientOptions(WithMaxRetries(3)),
				WithTimeout(10 * time.Second),
			},
			wantTimeout: 10 * time.Second,
			validateFunc: func(t *testing.T, p *Provider) {
				assert.Equal(t, 3, p.client.maxRetries)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewProvider(tt.opts...)

			assert.NotNil(t, provider)
			assert.NotNil(t, provider.client)
			assert.Equal(t, tt.wantTimeout, provider.timeout)

			if tt.validateFunc != nil {
				tt.validateFunc(t, provider)
			}
		})
	}
}

// TestProviderWithTimeout tests the withTimeout helper.
func TestProviderWithTimeout(t *testing.T) {
	provider := NewProvider(WithTimeout(3 * time.Second))

	tests := []struct {
		name      string
		inputCtx  context.Context
		expectNil bool
	}{
		{
			name:      "nil context creates background",
			inputCtx:  nil,
			expectNil: false,
		},
		{
			name:      "existing context gets timeout",
			inputCtx:  context.Background(),
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := provider.withTimeout(tt.inputCtx)
			defer cancel()

			assert.NotNil(t, ctx)
			deadline, ok := ctx.Deadline()
			assert.True(t, ok, "context should have deadline")
			assert.True(t, time.Until(deadline) <= 3*time.Second)
		})
	}
}

// TestClientDoRequestRetry tests the retry logic in doRequest.
func TestClientDoRequestRetry(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		maxRetries  int
		wantErr     bool
		errContains string
		expectCalls int
	}{
		{
			name: "successful after retry",
			setupServer: func() *httptest.Server {
				callCount := 0
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					callCount++
					if callCount < 2 {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					writeJSON(w, map[string]string{"BTC": "150"})
				}))
			},
			maxRetries:  2,
			wantErr:     false,
			expectCalls: 2,
		},
		{
			name: "fail after max retries",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadGateway)
				}))
			},
			maxRetries:  1,
			wantErr:     true,
			errContains: "http status 502",
		},
		{
			name: "context timeout during retry",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(200 * time.Millisecond)
					writeJSON(w, map[string]string{})
				}))
			},
			maxRetries:  2,
			wantErr:     true,
			errContains: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			client := NewClient(
				WithBaseURL(server.URL),
				WithHTTPClient(server.Client()),
				WithMaxRetries(tt.maxRetries),
			)

			ctx := context.Background()
			if tt.name == "context timeout during retry" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}

			var result AllMidsResponse
			err := client.doRequest(ctx, InfoRequest{Type: "allMids"}, &result)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestClientGetCurrentPrice tests the GetCurrentPrice method.
func TestClientGetCurrentPrice(t *testing.T) {
	tests := []struct {
		name        string
		symbol      string
		wantPrice   float64
		wantErr     bool
		errContains string
	}{
		{
			name:      "successful price retrieval",
			symbol:    "BTC",
			wantPrice: 150.0,
			wantErr:   false,
		},
		{
			name:        "symbol not found",
			symbol:      "UNKNOWN",
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:      "case insensitive symbol",
			symbol:    "btc",
			wantPrice: 150.0,
			wantErr:   false,
		},
	}

	server, client := newMockHyperliquidServer(t)
	defer server.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			price, err := client.GetCurrentPrice(ctx, tt.symbol)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.wantPrice, price, 1e-9)
			}
		})
	}
}
