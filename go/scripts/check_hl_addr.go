package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"nof0-api/internal/config"

	"github.com/ethereum/go-ethereum/crypto"
)

type infoReq struct {
	Type string `json:"type"`
	User string `json:"user"`
}

func queryInfo(url, addr string) (map[string]any, error) {
	body, _ := json.Marshal(infoReq{Type: "clearinghouseState", User: addr})
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c := &http.Client{Timeout: 10 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]any{"raw": string(b), "status": resp.Status}, nil
	}
	out["http_status"] = resp.Status
	return out, nil
}

func main() {
	// Ensure default exchange config (and .env) is loaded before reading env vars.
	_ = config.MustLoadExchange()

	pk := os.Getenv("HYPERLIQUID_PRIVATE_KEY")
	if pk == "" {
		fmt.Println("HYPERLIQUID_PRIVATE_KEY not set in env/.env")
		os.Exit(1)
	}
	keyHex := strings.TrimPrefix(strings.TrimSpace(pk), "0x")
	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		fmt.Printf("decode private key error: %v\n", err)
		os.Exit(1)
	}
	apiWallet := strings.ToLower(crypto.PubkeyToAddress(key.PublicKey).Hex())
	mainAddr := strings.ToLower(strings.TrimSpace(os.Getenv("HYPERLIQUID_MAIN_ADDRESS")))

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("API Wallet (from private key): %s\n", apiWallet)
	if mainAddr != "" {
		fmt.Printf("Main Account (HYPERLIQUID_MAIN_ADDRESS): %s\n", mainAddr)
	} else {
		fmt.Println("Main Account: (not set - using API wallet as main account)")
	}
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	// Check if using API wallet mode
	if mainAddr != "" && mainAddr != apiWallet {
		fmt.Println("⚠️  API WALLET MODE DETECTED")
		fmt.Println("Your setup uses an API wallet to trade on behalf of a main account.")
		fmt.Println("")
		fmt.Println("IMPORTANT: The API wallet MUST be registered with your main account!")
		fmt.Println("")
		fmt.Println("To register the API wallet:")
		fmt.Println("1. Go to https://app.hyperliquid-testnet.xyz/ (for testnet)")
		fmt.Println("   or https://app.hyperliquid.xyz/ (for mainnet)")
		fmt.Println("2. Connect with your MAIN ACCOUNT wallet")
		fmt.Printf("   (address: %s)\n", mainAddr)
		fmt.Println("3. Go to Settings → API")
		fmt.Println("4. Click 'Add API Wallet'")
		fmt.Printf("5. Enter this address: %s\n", apiWallet)
		fmt.Println("6. Approve the transaction")
		fmt.Println("")
	}

	// Probe testnet and mainnet info endpoints
	testnet := "https://api.hyperliquid-testnet.xyz/info"
	mainnet := "https://api.hyperliquid.xyz/info"

	checkAddr := apiWallet
	if mainAddr != "" {
		checkAddr = mainAddr
	}

	fmt.Printf("Checking account state for: %s\n\n", checkAddr)

	fmt.Println("--- TESTNET ---")
	if m, err := queryInfo(testnet, checkAddr); err == nil {
		fmt.Printf("Status: %v\n", m["http_status"])
		if state, ok := m["assetPositions"]; ok {
			fmt.Printf("Asset Positions: %v\n", state)
		}
		if mv, ok := m["marginSummary"]; ok {
			fmt.Printf("Margin Summary: %v\n", mv)
		}
	} else {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Println("\n--- MAINNET ---")
	if m, err := queryInfo(mainnet, checkAddr); err == nil {
		fmt.Printf("Status: %v\n", m["http_status"])
		if state, ok := m["assetPositions"]; ok {
			fmt.Printf("Asset Positions: %v\n", state)
		}
		if mv, ok := m["marginSummary"]; ok {
			fmt.Printf("Margin Summary: %v\n", mv)
		}
	} else {
		fmt.Printf("Error: %v\n", err)
	}
}
