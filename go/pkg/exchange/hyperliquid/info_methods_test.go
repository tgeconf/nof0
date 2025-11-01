package hyperliquid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetSubAccounts_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
            {
              "name": "Test",
              "subAccountUser": "0x035605fc2f24d65300227189025e90a0d947f16c",
              "master": "0x8c967e73e6b15087c42a10d344cff4c96d877f1d",
              "clearinghouseState": {
                "marginSummary": {"accountValue": "29.78001", "totalNtlPos": "0.0", "totalRawUsd": "29.78001", "totalMarginUsed": "0.0"},
                "crossMarginSummary": {"accountValue": "29.78001", "totalNtlPos": "0.0", "totalRawUsd": "29.78001", "totalMarginUsed": "0.0"},
                "assetPositions": []
              },
              "spotState": {"balances": [{"coin": "USDC", "token": 0, "total": "0.22", "hold": "0.0", "entryNtl": "0.0"}]}
            }
        ]`))
	}))
	defer server.Close()

	client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
	require.NoError(t, err)
	client.infoURL = server.URL

	out, err := client.GetSubAccounts(context.Background(), "0x8c967e73e6b15087c42a10d344cff4c96d877f1d")
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "Test", out[0].Name)
	require.Equal(t, "29.78001", out[0].ClearinghouseState.MarginSummary.AccountValue)
}

func TestGetSubAccounts_InvalidUser(t *testing.T) {
	client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
	require.NoError(t, err)
	_, err = client.GetSubAccounts(context.Background(), "not-an-address")
	require.Error(t, err)
}

func TestGetVaultDetails_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
          "name": "Test",
          "vaultAddress": "0xdfc24b077bc1425ad1dea75bcb6f8158e10df303",
          "leader": "0x677d831aef5328190852e24f13c46cac05f984e7",
          "description": "A vault",
          "apr": 0.12,
          "followers": [],
          "maxDistributable": 10.0,
          "maxWithdrawable": 5.0,
          "isClosed": false,
          "allowDeposits": true,
          "alwaysCloseOnWithdraw": false
        }`))
	}))
	defer server.Close()

	client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
	require.NoError(t, err)
	client.infoURL = server.URL

	out, err := client.GetVaultDetails(context.Background(), "0xdfc24b077bc1425ad1dea75bcb6f8158e10df303", "")
	require.NoError(t, err)
	require.Equal(t, "Test", out.Name)
	require.Equal(t, "0xdfc24b077bc1425ad1dea75bcb6f8158e10df303", out.VaultAddress)
}

func TestGetVaultDetails_InvalidVault(t *testing.T) {
	client, err := NewClient("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a741b52d7c5d5095e2f", false)
	require.NoError(t, err)
	_, err = client.GetVaultDetails(context.Background(), "not-an-address", "")
	require.Error(t, err)
}
