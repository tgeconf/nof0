package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"nof0-api/pkg/exchange"
)

const (
	mainnetInfoURL     = "https://api.hyperliquid.xyz/info"
	mainnetExchangeURL = "https://api.hyperliquid.xyz/exchange"
	testnetInfoURL     = "https://api.hyperliquid-testnet.xyz/info"
	testnetExchangeURL = "https://api.hyperliquid-testnet.xyz/exchange"

	defaultHTTPTimeout  = 30 * time.Second
	defaultRetryBackoff = 200 * time.Millisecond
	maxRetryAttempts    = 3
)

// Client coordinates signed requests against Hyperliquid exchange endpoints.
type Client struct {
	infoURL     string
	exchangeURL string
	httpClient  *http.Client
	signer      Signer
	address     string // API wallet address (derived from signer)
	mainAddress string // Main account address (for info requests when using API wallet)
	isTestnet   bool
	logger      *log.Logger
	clock       func() time.Time
	vault       string

	assetMu    sync.RWMutex
	assetIndex map[string]int
	assetInfo  map[string]AssetInfo

	// Trade defaults / formatting
	defaultSlippage float64
	priceSigFigs    int

	// Asset directory cache
	assetTTL     time.Duration
	assetLastRef time.Time
}

// ClientOption customises the Hyperliquid client.
type ClientOption func(*Client)

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// WithLogger attaches a custom logger (defaults to log.Default()).
func WithLogger(logger *log.Logger) ClientOption {
	return func(c *Client) {
		if logger != nil {
			c.logger = logger
		}
	}
}

// WithVaultAddress configures a vault address for signing requests.
func WithVaultAddress(addr string) ClientOption {
	return func(c *Client) {
		if common.IsHexAddress(addr) {
			c.vault = common.HexToAddress(addr).Hex()
		}
	}
}

// WithMainAddress configures the main account address for info requests.
// This is used when the API wallet (agent wallet) is different from the main account.
// Info requests must use the main account's public address, while exchange requests
// are signed by the API wallet on behalf of the main account.
func WithMainAddress(addr string) ClientOption {
	return func(c *Client) {
		if common.IsHexAddress(addr) {
			c.mainAddress = common.HexToAddress(addr).Hex()
		}
	}
}

// WithClock overrides the time source (primarily for testing).
func WithClock(clock func() time.Time) ClientOption {
	return func(c *Client) {
		if clock != nil {
			c.clock = clock
		}
	}
}

// WithDefaultSlippage configures a default slippage fraction used by helpers
// when caller does not specify one (e.g. 0.01 = 1%).
func WithDefaultSlippage(slippage float64) ClientOption {
	return func(c *Client) {
		if slippage > 0 {
			c.defaultSlippage = slippage
		}
	}
}

// WithPriceSigFigs sets the default number of price significant figures
// used by helper methods when formatting prices.
func WithPriceSigFigs(sigfigs int) ClientOption {
	return func(c *Client) {
		if sigfigs >= 1 {
			c.priceSigFigs = sigfigs
		}
	}
}

// WithAssetCacheTTL sets a time-to-live for the asset directory cache.
// When positive, the client refreshes asset metadata after TTL elapses.
func WithAssetCacheTTL(ttl time.Duration) ClientOption {
	return func(c *Client) {
		if ttl > 0 {
			c.assetTTL = ttl
		}
	}
}

// getInfoAddress returns the address to use for info requests.
// If mainAddress is configured (API wallet scenario), it returns mainAddress.
// Otherwise, it returns the signer's address.
func (c *Client) getInfoAddress() string {
	if c.mainAddress != "" {
		return c.mainAddress
	}
	return c.address
}

// NewClient constructs a Hyperliquid trading client using the provided private key.
func NewClient(privateKeyHex string, isTestnet bool, opts ...ClientOption) (*Client, error) {
	if privateKeyHex == "" {
		return nil, fmt.Errorf("hyperliquid: private key is required")
	}

	signer, err := NewPrivateKeySigner(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid: create signer: %w", err)
	}

	client := &Client{
		infoURL:     mainnetInfoURL,
		exchangeURL: mainnetExchangeURL,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
		signer:       signer,
		address:      signer.GetAddress(),
		isTestnet:    isTestnet,
		logger:       log.Default(),
		clock:        time.Now,
		assetIndex:   make(map[string]int),
		assetInfo:    make(map[string]AssetInfo),
		priceSigFigs: 5,
	}
	if isTestnet {
		client.infoURL = testnetInfoURL
		client.exchangeURL = testnetExchangeURL
	}
	for _, opt := range opts {
		opt(client)
	}
	if client.httpClient == nil {
		client.httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	if client.logger == nil {
		client.logger = log.Default()
	}
	if client.clock == nil {
		client.clock = time.Now
	}
	return client, nil
}

// PlaceOrder submits a single order to the exchange endpoint.
func (c *Client) PlaceOrder(ctx context.Context, order exchange.Order) (*exchange.OrderResponse, error) {
	return c.PlaceOrders(ctx, []exchange.Order{order})
}

// PlaceOrders submits multiple orders atomically.
func (c *Client) PlaceOrders(ctx context.Context, orders []exchange.Order) (*exchange.OrderResponse, error) {
	if len(orders) == 0 {
		return nil, fmt.Errorf("hyperliquid: at least one order required")
	}
	action, err := buildPlaceOrderAction(orders)
	if err != nil {
		return nil, err
	}
	var resp exchange.OrderResponse
	if err := c.doExchangeRequest(ctx, action, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CancelOrder cancels a single resting order.
func (c *Client) CancelOrder(ctx context.Context, asset int, oid int64) error {
	action := buildCancelAction([]Cancel{{Asset: asset, Oid: oid}})
	return c.doExchangeRequest(ctx, action, nil)
}

// CancelOrders executes batch cancellations.
func (c *Client) CancelOrders(ctx context.Context, cancels []Cancel) error {
	if len(cancels) == 0 {
		return nil
	}
	action := buildCancelAction(cancels)
	return c.doExchangeRequest(ctx, action, nil)
}

// CancelAllOrders cancels all resting orders for the specified asset.
func (c *Client) CancelAllOrders(ctx context.Context, asset int) error {
	// Get all open orders
	orders, err := c.GetOpenOrders(ctx)
	if err != nil {
		return fmt.Errorf("hyperliquid: failed to get open orders: %w", err)
	}

	// Filter orders for the specified asset and build cancel list
	var cancels []Cancel
	for _, order := range orders {
		// Get the asset index for this order's coin
		orderAsset, err := c.GetAssetIndex(ctx, order.Order.Coin)
		if err != nil {
			// Skip orders we can't identify
			continue
		}
		// Only cancel orders matching the target asset
		if orderAsset == asset {
			cancels = append(cancels, Cancel{
				Asset: asset,
				Oid:   order.Order.Oid,
			})
		}
	}

	// If no orders to cancel, return success
	if len(cancels) == 0 {
		return nil
	}

	// Cancel the filtered orders using the standard cancel action
	return c.CancelOrders(ctx, cancels)
}

// doInfoRequest queries the public info endpoint.
func (c *Client) doInfoRequest(ctx context.Context, req InfoRequest, result interface{}) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("hyperliquid: encode info request: %w", err)
	}
	backoff := defaultRetryBackoff
	var lastErr error
	for attempt := 0; attempt < maxRetryAttempts; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.infoURL, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("hyperliquid: build info request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastErr = err
		} else {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				lastErr = fmt.Errorf("hyperliquid: read info response: %w", readErr)
			} else if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
				lastErr = fmt.Errorf("hyperliquid: info http status %d: %s", resp.StatusCode, string(body))
			} else if result != nil {
				if err := json.Unmarshal(body, result); err != nil {
					return fmt.Errorf("hyperliquid: decode info response: %w", err)
				}
				return nil
			} else {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("hyperliquid: info request failed")
}

// GetSubAccounts retrieves the list of subaccounts for a master user address.
func (c *Client) GetSubAccounts(ctx context.Context, user string) ([]SubAccount, error) {
	if !common.IsHexAddress(user) {
		return nil, fmt.Errorf("hyperliquid: invalid user address %q", user)
	}
	var out []SubAccount
	err := c.doInfoRequest(ctx, InfoRequest{Type: "subAccounts", User: common.HexToAddress(user).Hex()}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetVaultDetails retrieves vault details by address (optionally scoped by user).
func (c *Client) GetVaultDetails(ctx context.Context, vaultAddress string, user string) (*VaultDetails, error) {
	if !common.IsHexAddress(vaultAddress) {
		return nil, fmt.Errorf("hyperliquid: invalid vault address %q", vaultAddress)
	}
	req := InfoRequest{Type: "vaultDetails", VaultAddress: common.HexToAddress(vaultAddress).Hex()}
	if user != "" {
		if !common.IsHexAddress(user) {
			return nil, fmt.Errorf("hyperliquid: invalid user address %q", user)
		}
		req.User = common.HexToAddress(user).Hex()
	}
	var out VaultDetails
	if err := c.doInfoRequest(ctx, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// doExchangeRequest signs and submits an exchange action.
func (c *Client) doExchangeRequest(ctx context.Context, action Action, result interface{}) error {
	exchangeReq, err := c.signAction(action)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(exchangeReq)
	if err != nil {
		return fmt.Errorf("hyperliquid: encode exchange request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.exchangeURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("hyperliquid: build exchange request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("hyperliquid: read exchange response: %w", readErr)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
		return fmt.Errorf("hyperliquid: exchange http status %d: %s", resp.StatusCode, string(body))
	}
	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("hyperliquid: decode exchange response: %w", err)
		}
	}
	return nil
}

// signAction builds the EIP-712 payload and signs it.
func (c *Client) signAction(action Action) (*ExchangeRequest, error) {
	now := c.clock
	if now == nil {
		now = time.Now
	}
	nonce := now().UnixMilli()
	exchangeReq, err := signAction(action, c.signer, nonce, c.vault, !c.isTestnet)
	if err != nil {
		return nil, err
	}
	return exchangeReq, nil
}

func (c *Client) logf(format string, args ...interface{}) {
	if c.logger != nil {
		c.logger.Printf(format, args...)
	}
}
