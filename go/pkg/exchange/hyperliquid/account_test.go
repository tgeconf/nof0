package hyperliquid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAccountState(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "ok",
				"data": {
					"assetPositions": [],
					"crossMarginSummary": {
						"accountValue": "10000.5"
					},
					"marginSummary": {
						"accountValue": "10000.5",
						"totalMarginUsed": "500.0",
						"totalNtlPos": "2000.0",
						"totalRawUsd": "9500.5"
					},
					"withdrawable": "9000.0"
				}
			}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		state, err := client.GetAccountState(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, state)
		assert.Equal(t, "10000.5", state.MarginSummary.AccountValue)
	})

	t.Run("empty_address", func(t *testing.T) {
		client := &Client{address: ""}
		_, err := client.GetAccountState(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "address unavailable")
	})

	t.Run("non_ok_status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "error"}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		_, err = client.GetAccountState(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status")
	})

	t.Run("missing_data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		_, err = client.GetAccountState(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing data")
	})
}

func TestGetAccountValue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "ok",
				"data": {
					"assetPositions": [],
					"marginSummary": {
						"accountValue": "12345.67"
					}
				}
			}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		value, err := client.GetAccountValue(context.Background())
		assert.NoError(t, err)
		assert.InDelta(t, 12345.67, value, 0.001)
	})

	t.Run("invalid_value_format", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "ok",
				"data": {
					"marginSummary": {
						"accountValue": "invalid"
					}
				}
			}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		_, err = client.GetAccountValue(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parse")
	})
}

func TestDoInfoRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok", "data": {"key": "value"}}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		var result map[string]interface{}
		err = client.doInfoRequest(context.Background(), InfoRequest{Type: "test"}, &result)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("http_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "bad request"}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		var result map[string]interface{}
		err = client.doInfoRequest(context.Background(), InfoRequest{Type: "test"}, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "http status")
	})

	t.Run("invalid_json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid json`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		var result map[string]interface{}
		err = client.doInfoRequest(context.Background(), InfoRequest{Type: "test"}, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "decode")
	})

	t.Run("nil_result", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}))
		defer server.Close()

		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
		assert.NoError(t, err)
		client.infoURL = server.URL

		err = client.doInfoRequest(context.Background(), InfoRequest{Type: "test"}, nil)
		assert.NoError(t, err)
	})
}
