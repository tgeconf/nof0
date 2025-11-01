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
	infoAddr := c.getInfoAddress()
	if infoAddr == "" {
		return nil, fmt.Errorf("hyperliquid: client address unavailable")
	}
	var state exchange.AccountState
	if err := c.doInfoRequest(ctx, InfoRequest{
		Type: "clearinghouseState",
		User: infoAddr,
	}, &state); err != nil {
		return nil, err
	}
	// Basic sanity check: margin summary should be present
	if strings.TrimSpace(state.MarginSummary.AccountValue) == "" && strings.TrimSpace(state.CrossMarginSummary.AccountValue) == "" {
		return nil, fmt.Errorf("hyperliquid: clearinghouseState missing fields")
	}
	return &state, nil
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
