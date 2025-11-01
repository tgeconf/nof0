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

	_ "nof0-api/internal/bootstrap/dotenv"

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
	addr := strings.ToLower(crypto.PubkeyToAddress(key.PublicKey).Hex())
	fmt.Printf("Derived address: %s\n", addr)

	// Probe testnet and mainnet info endpoints
	testnet := "https://api.hyperliquid-testnet.xyz/info"
	mainnet := "https://api.hyperliquid.xyz/info"
	if m, err := queryInfo(testnet, addr); err == nil {
		fmt.Printf("Testnet clearinghouseState: %v\n", m)
	} else {
		fmt.Printf("Testnet info error: %v\n", err)
	}
	if m, err := queryInfo(mainnet, addr); err == nil {
		fmt.Printf("Mainnet clearinghouseState: %v\n", m)
	} else {
		fmt.Printf("Mainnet info error: %v\n", err)
	}
}
