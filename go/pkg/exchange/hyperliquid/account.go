package hyperliquid

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"nof0-api/pkg/exchange"
)

// GetAccountState fetches the clearinghouse state for the signer address.
func (c *Client) GetAccountState(ctx context.Context) (*exchange.AccountState, error) {
	if c.address == "" {
		return nil, fmt.Errorf("hyperliquid: client address unavailable")
	}
	var resp AccountStateResponse
	if err := c.doInfoRequest(ctx, InfoRequest{
		Type: "clearinghouseState",
		User: c.address,
	}, &resp); err != nil {
		return nil, err
	}
	if strings.ToLower(resp.Status) != "ok" {
		return nil, fmt.Errorf("hyperliquid: clearinghouseState status %q", resp.Status)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("hyperliquid: clearinghouseState missing data")
	}
	return resp.Data, nil
}

// GetAccountValue returns the account value parsed as float64.
func (c *Client) GetAccountValue(ctx context.Context) (float64, error) {
	state, err := c.GetAccountState(ctx)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseFloat(state.MarginSummary.AccountValue, 64)
	if err != nil {
		return 0, fmt.Errorf("hyperliquid: parse account value: %w", err)
	}
	return value, nil
}
