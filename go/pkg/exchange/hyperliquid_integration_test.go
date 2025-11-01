//go:build integration

package exchange_test

import (
	"context"
	"net/http"
	"nof0-api/pkg/exchange"
	"os"
	"strings"
	"testing"
	"time"

	// Import for side-effects: registers hyperliquid providers
	_ "nof0-api/pkg/exchange/hyperliquid"

	hl "nof0-api/pkg/exchange/hyperliquid"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type HLIntegrationSuite struct {
	suite.Suite
	Provider *hl.Provider
	Coin     string
	AssetIdx int
}

func (s *HLIntegrationSuite) SetupSuite() {
	// Load exchange config via config module helpers (no env fallback; panic if missing).
	cfg := exchange.MustLoad()
	s.Coin = "BTC"

	for _, provider := range cfg.Providers {
		provider.Testnet = true
	}

	def := cfg.Default
	if def == "" {
		for k := range cfg.Providers {
			def = k
			break
		}
	}
	if p, ok := cfg.Providers[def]; ok {
		if p.Timeout == 0 {
			p.Timeout = 20 * time.Second
		}
	}
	providers, err := cfg.BuildProviders()
	s.Require().NoError(err, "BuildProviders(exchange)")
	prov, ok := providers[def]
	s.Require().True(ok, "default exchange provider not built")
	hp, ok := prov.(*hl.Provider)
	if !ok {
		s.T().Skip("default provider is not Hyperliquid; skipping integration tests")
	}
	s.Provider = hp

	// Resolve asset index strictly; failure here indicates symbol/env mismatch.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	idx, err := s.Provider.GetAssetIndex(ctx, s.Coin)
	s.Require().NoErrorf(err, "GetAssetIndex(%s)", s.Coin)
	s.Require().GreaterOrEqual(idx, 0, "asset index should be >= 0")
	s.AssetIdx = idx
}

func (s *HLIntegrationSuite) TearDownTest() {
	// Best-effort cleanup to keep account tidy across runs.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_ = s.Provider.CancelAllBySymbol(ctx, s.Coin)
	_ = s.Provider.ClosePosition(ctx, s.Coin)
}

// Strictly verify account state endpoint shape/behavior.
func (s *HLIntegrationSuite) Test_AccountState_Strict() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	state, err := s.Provider.GetAccountState(ctx)
	// Strict mode: endpoint must succeed with valid shape.
	s.Require().NoError(err, "GetAccountState must succeed with canonical response")
	s.Require().NotNil(state)
	s.Require().NotEmpty(state.MarginSummary.AccountValue)

	value, err := s.Provider.GetAccountValue(ctx)
	s.Require().NoError(err, "GetAccountValue")
	s.True(value >= 0, "account value should be non-negative")
}

// Utilities rely on live metadata and should succeed.
func (s *HLIntegrationSuite) Test_Utilities() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	price, err := s.Provider.FormatPrice(ctx, s.Coin, 12345.6789)
	s.Require().NoError(err)
	s.NotEmpty(price)

	size, err := s.Provider.FormatSize(ctx, s.Coin, 0.0012345)
	s.Require().NoError(err)
	s.NotEmpty(size)
}

// Writable endpoints: leverage update, IOC order, triggers, cleanup.
// This test purposely requires a minimally funded testnet account.
func (s *HLIntegrationSuite) Test_OrderLifecycle_Strict() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Leverage update should be allowed regardless of balance on many venues; assert strictly.
	s.Require().NoError(s.Provider.UpdateLeverage(ctx, s.AssetIdx, true, 2), "UpdateLeverage")

	// Require a minimum account value before placing real orders.
	value, err := s.Provider.GetAccountValue(ctx)
	s.Require().NoError(err, "GetAccountValue")

	minBal := 1.0
	s.Require().Truef(value >= minBal, "testnet account must have >= %.2f USD (have %.4f)", minBal, value)

	// Place a tiny IOC buy, assert an OK status.
	resp, err := s.Provider.IOCMarket(ctx, s.Coin, true, 0.001, 0.02, false)
	s.Require().NoError(err, "IOCMarket")
	s.Require().NotNil(resp)
	assert.Equal(s.T(), "ok", resp.Status, "IOC response status should be ok")

	// Optionally place a reduce-only stop-loss far away and then cancel-all.
	s.Require().NoError(s.Provider.SetStopLoss(ctx, s.Coin, "LONG", 0.001, 1_000_000), "SetStopLoss")

	// Cleanup: cancel-all and close position must succeed (best effort but strict for surfacing issues).
	s.Require().NoError(s.Provider.CancelAllBySymbol(ctx, s.Coin), "CancelAllBySymbol")
	s.Require().NoError(s.Provider.ClosePosition(ctx, s.Coin), "ClosePosition")
}

// Read-only endpoints from the API doc: subAccounts and vaultDetails.
func (s *HLIntegrationSuite) Test_InfoEndpoints_SubAccounts_Vault() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pk := os.Getenv("HYPERLIQUID_PRIVATE_KEY")
	key, err := crypto.HexToECDSA(strings.TrimPrefix(pk, "0x"))
	s.Require().NoError(err)
	user := strings.ToLower(crypto.PubkeyToAddress(key.PublicKey).Hex())

	// Build a direct client to call info helpers.
	c, err := hl.NewClient(pk, true, hl.WithHTTPClient(&http.Client{Timeout: 20 * time.Second}))
	s.Require().NoError(err)

	_, err = c.GetSubAccounts(ctx, user)
	s.Require().NoError(err)

	if vault := os.Getenv("HYPERLIQUID_VAULT_ADDRESS"); vault != "" {
		vd, err := c.GetVaultDetails(ctx, vault, "")
		s.Require().NoError(err)
		if strings.TrimSpace(vd.VaultAddress) != "" {
			s.Equal(strings.ToLower(vault), strings.ToLower(vd.VaultAddress))
		}
	}
}

func TestHLIntegrationSuite(t *testing.T) {
	suite.Run(t, new(HLIntegrationSuite))
}
