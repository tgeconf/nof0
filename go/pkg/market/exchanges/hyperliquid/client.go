package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURL          = "https://api.hyperliquid.xyz/info"
	defaultHTTPTimeout      = 10 * time.Second
	defaultMaxRetries       = 3
	defaultRetryBackoffBase = 150 * time.Millisecond
)

// ErrSymbolNotFound indicates that the requested symbol is not listed.
var ErrSymbolNotFound = errors.New("hyperliquid: symbol not found")

// Client wraps access to the Hyperliquid info endpoint.
type Client struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
	logger     *log.Logger

	symbolsMu        sync.RWMutex
	symbolIndex      map[string]string
	assetCtxBySymbol map[string]AssetCtx
	universeMeta     map[string]UniverseEntry
}

// Option configures a new Client.
type Option func(*Client)

// WithHTTPClient injects a custom http.Client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithBaseURL overrides the default info endpoint URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		if url != "" {
			c.baseURL = url
		}
	}
}

// WithMaxRetries adjusts the retry budget.
func WithMaxRetries(max int) Option {
	return func(c *Client) {
		if max >= 0 {
			c.maxRetries = max
		}
	}
}

// WithLogger injects a custom logger (defaults to log.Default()).
func WithLogger(l *log.Logger) Option {
	return func(c *Client) {
		if l != nil {
			c.logger = l
		}
	}
}

// NewClient constructs a Hyperliquid API client.
func NewClient(opts ...Option) *Client {
	httpClient := &http.Client{Timeout: defaultHTTPTimeout}
	client := &Client{
		baseURL:    defaultBaseURL,
		httpClient: httpClient,
		maxRetries: defaultMaxRetries,
		logger:     log.Default(),
	}
	for _, opt := range opts {
		opt(client)
	}
	if client.httpClient == nil {
		client.httpClient = httpClient
	}
	if client.logger == nil {
		client.logger = log.Default()
	}
	return client
}

// doRequest posts an InfoRequest and decodes the response into result.
func (c *Client) doRequest(ctx context.Context, req InfoRequest, result interface{}) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("hyperliquid: encode request: %w", err)
	}
	var lastErr error
	backoff := defaultRetryBackoffBase
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("hyperliquid: build request: %w", err)
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
				lastErr = fmt.Errorf("hyperliquid: read response: %w", readErr)
			} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				lastErr = fmt.Errorf("hyperliquid: http status %d: %s", resp.StatusCode, string(body))
			} else {
				if result != nil {
					if err := json.Unmarshal(body, result); err != nil {
						return fmt.Errorf("hyperliquid: decode response: %w", err)
					}
				}
				return nil
			}
		}

		if attempt < c.maxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
			continue
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("hyperliquid: request failed without error detail")
}

// logf prints debug output when a logger is configured.
func (c *Client) logf(format string, args ...interface{}) {
	if c.logger != nil {
		c.logger.Printf(format, args...)
	}
}

func (c *Client) canonicalFromCache(symbol string) (string, bool) {
	key := normalizeKey(symbol)
	if key == "" {
		return "", false
	}
	c.symbolsMu.RLock()
	canonical, ok := c.symbolIndex[key]
	c.symbolsMu.RUnlock()
	return canonical, ok
}

func (c *Client) assetCtxFromCache(symbol string) (string, AssetCtx, bool) {
	key := normalizeKey(symbol)
	if key == "" {
		return "", AssetCtx{}, false
	}
	c.symbolsMu.RLock()
	canonical, ok := c.symbolIndex[key]
	if !ok {
		c.symbolsMu.RUnlock()
		return "", AssetCtx{}, false
	}
	ctxData, ok := c.assetCtxBySymbol[canonical]
	c.symbolsMu.RUnlock()
	return canonical, ctxData, ok
}

func (c *Client) refreshSymbolDirectory(ctx context.Context) error {
	var payload MetaAndAssetCtxsResponse
	if err := c.doRequest(ctx, InfoRequest{Type: "metaAndAssetCtxs"}, &payload); err != nil {
		return err
	}

	index := make(map[string]string, len(payload.Universe))
	assetCtx := make(map[string]AssetCtx, len(payload.AssetCtxs))
	universe := make(map[string]UniverseEntry, len(payload.Universe))
	for i, entry := range payload.Universe {
		canonical := strings.TrimSpace(entry.Name)
		if canonical == "" {
			continue
		}
		key := normalizeKey(canonical)
		if key == "" {
			continue
		}
		index[key] = canonical
		if i < len(payload.AssetCtxs) {
			assetCtx[canonical] = payload.AssetCtxs[i]
		}
		universe[canonical] = entry
	}

	c.symbolsMu.Lock()
	c.symbolIndex = index
	c.assetCtxBySymbol = assetCtx
	c.universeMeta = universe
	c.symbolsMu.Unlock()
	return nil
}

func (c *Client) canonicalSymbolFor(ctx context.Context, symbol string) (string, error) {
	if canonical, ok := c.canonicalFromCache(symbol); ok {
		return canonical, nil
	}
	if err := c.refreshSymbolDirectory(ctx); err != nil {
		return "", err
	}
	if canonical, ok := c.canonicalFromCache(symbol); ok {
		return canonical, nil
	}
	return "", ErrSymbolNotFound
}

func normalizeKey(symbol string) string {
	trimmed := strings.TrimSpace(symbol)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) > 4 && strings.EqualFold(trimmed[len(trimmed)-4:], "USDT") {
		trimmed = trimmed[:len(trimmed)-4]
	}
	return strings.ToUpper(trimmed)
}
