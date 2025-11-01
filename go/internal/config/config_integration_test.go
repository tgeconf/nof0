package config_test

import (
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	appconfig "nof0-api/internal/config"
	"nof0-api/internal/svc"
)

// genTestPrivKey returns a valid hex-encoded secp256k1 private key for tests.
func genTestPrivKey(t *testing.T) string {
	t.Helper()
	// Use a deterministic small scalar to avoid randomness in hermetic tests.
	// Not used for real signing on network calls in this test.
	one := big.NewInt(1)
	key := new(ecdsa.PrivateKey)
	key.PublicKey.Curve = crypto.S256()
	key.D = one
	key.PublicKey.X, key.PublicKey.Y = crypto.S256().ScalarBaseMult(one.Bytes())
	h := hex.EncodeToString(key.D.Bytes())
	// Left pad to 64 hex chars (32 bytes)
	if len(h) < 64 {
		h = strings.Repeat("0", 64-len(h)) + h
	}
	return h
}

func TestMustLoadAndProviders(t *testing.T) {
	// Compose a minimal main config in a temp dir that skips LLM section
	// and references the real etc/* module files via absolute paths.
	etcDir := filepath.Clean(filepath.Join("..", "..", "etc"))
	etcAbs, err := filepath.Abs(etcDir)
	if err != nil {
		t.Fatalf("Abs(%s) error: %v", etcDir, err)
	}
	exch := filepath.Join(etcAbs, "exchange.yaml")
	mkt := filepath.Join(etcAbs, "market.yaml")
	exec := filepath.Join(etcAbs, "executor.yaml")
	mgr := filepath.Join(etcAbs, "manager.yaml")

	// Provide env vars required by sub-configs.
	// Provide a valid-looking private key for Hyperliquid provider expansion.
	t.Setenv("HYPERLIQUID_PRIVATE_KEY", genTestPrivKey(t))

	mainYAML := []byte("" +
		"Name: test\n" +
		"Host: 127.0.0.1\n" +
		"Port: 0\n" +
		"DataPath: ../mcp/data\n" +
		"TTL:\n  Short: 10\n  Medium: 60\n  Long: 300\n\n" +
		"Executor:\n  File: " + exec + "\n\n" +
		"Manager:\n  File: " + mgr + "\n\n" +
		"Exchange:\n  File: " + exch + "\n\n" +
		"Market:\n  File: " + mkt + "\n")

	dir := t.TempDir()
	mainPath := filepath.Join(dir, "nof0.yaml")
	if err := os.WriteFile(mainPath, mainYAML, 0o600); err != nil {
		t.Fatalf("write temp main config: %v", err)
	}

	// Load the main config using the new simplified structure
	cfg, err := appconfig.Load(mainPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	// ServiceContext should now load module configs internally and wire trader -> providers strictly by ID
	sc := svc.NewServiceContext(*cfg, mainPath)

	// Verify that providers were built
	if len(sc.ExchangeProviders) == 0 {
		t.Fatalf("no exchange providers built")
	}
	if len(sc.MarketProviders) == 0 {
		t.Fatalf("no market providers built")
	}
	if len(sc.ManagerTraderExchange) == 0 || len(sc.ManagerTraderMarket) == 0 {
		t.Fatalf("manager trader provider mappings not initialised")
	}

	// Sanity: ensure all referenced providers exist in the maps.
	for traderID, p := range sc.ManagerTraderExchange {
		if p == nil {
			t.Fatalf("exchange provider nil for trader %s", traderID)
		}
	}
	for traderID, p := range sc.ManagerTraderMarket {
		if p == nil {
			t.Fatalf("market provider nil for trader %s", traderID)
		}
	}

	// Avoid linter complaining about unused imports in certain environments.
	_ = os.Getenv
}
