package hyperliquid

import (
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestClientOptions(t *testing.T) {
	// Test WithHTTPClient
	t.Run("WithHTTPClient", func(t *testing.T) {
		customClient := &http.Client{Timeout: 10 * time.Second}
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false, WithHTTPClient(customClient))
		assert.NoError(t, err)
		assert.Equal(t, customClient, client.httpClient)
	})

	t.Run("WithHTTPClient_nil", func(t *testing.T) {
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false, WithHTTPClient(nil))
		assert.NoError(t, err)
		assert.NotNil(t, client.httpClient)
	})

	// Test WithLogger
	t.Run("WithLogger", func(t *testing.T) {
		customLogger := log.New(nil, "", 0)
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false, WithLogger(customLogger))
		assert.NoError(t, err)
		assert.Equal(t, customLogger, client.logger)
	})

	t.Run("WithLogger_nil", func(t *testing.T) {
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false, WithLogger(nil))
		assert.NoError(t, err)
		assert.NotNil(t, client.logger)
	})

	// Test WithVaultAddress
	t.Run("WithVaultAddress", func(t *testing.T) {
		validAddress := "0x742d35Cc6634C0532925a3b8D4C0cD9D7fD703b0"
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false, WithVaultAddress(validAddress))
		assert.NoError(t, err)
		// Compare against canonical checksum representation
		assert.Equal(t, common.HexToAddress(validAddress).Hex(), client.vault)
	})

	t.Run("WithVaultAddress_invalid", func(t *testing.T) {
		invalidAddress := "invalid_address"
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false, WithVaultAddress(invalidAddress))
		assert.NoError(t, err)
		assert.Empty(t, client.vault)
	})

	// Test WithClock
	t.Run("WithClock", func(t *testing.T) {
		customClock := func() time.Time { return time.Unix(1234567890, 0) }
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false, WithClock(customClock))
		assert.NoError(t, err)
		assert.NotNil(t, client.clock)
	})

	t.Run("WithClock_nil", func(t *testing.T) {
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false, WithClock(nil))
		assert.NoError(t, err)
		assert.NotNil(t, client.clock)
	})

	// Test WithDefaultSlippage
	t.Run("WithDefaultSlippage", func(t *testing.T) {
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false, WithDefaultSlippage(0.02))
		assert.NoError(t, err)
		assert.InDelta(t, 0.02, client.defaultSlippage, 1e-12)
	})

	t.Run("WithDefaultSlippage_zero", func(t *testing.T) {
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false, WithDefaultSlippage(0))
		assert.NoError(t, err)
		assert.Equal(t, 0.0, client.defaultSlippage)
	})

	// Test WithPriceSigFigs
	t.Run("WithPriceSigFigs", func(t *testing.T) {
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false, WithPriceSigFigs(4))
		assert.NoError(t, err)
		assert.Equal(t, 4, client.priceSigFigs)
	})

	t.Run("WithPriceSigFigs_invalid", func(t *testing.T) {
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false, WithPriceSigFigs(0))
		assert.NoError(t, err)
		assert.Equal(t, 5, client.priceSigFigs) // Default value
	})

	// Test WithAssetCacheTTL
	t.Run("WithAssetCacheTTL", func(t *testing.T) {
		ttl := 5 * time.Minute
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false, WithAssetCacheTTL(ttl))
		assert.NoError(t, err)
		assert.Equal(t, ttl, client.assetTTL)
	})

	t.Run("WithAssetCacheTTL_zero", func(t *testing.T) {
		client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false, WithAssetCacheTTL(0))
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), client.assetTTL)
	})
}
