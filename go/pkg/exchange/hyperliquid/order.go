package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"nof0-api/pkg/exchange"
)

const (
	defaultAggressiveBuyLimit  = "999999999"
	defaultAggressiveSellLimit = "0.00000001"
)

var (
	errInvalidAsset = errors.New("hyperliquid: asset index must be non-negative")
	errInvalidPrice = errors.New("hyperliquid: price must be positive")
	errInvalidSize  = errors.New("hyperliquid: size must be positive")
)

type frontendOpenOrder struct {
	Coin       string `json:"coin"`
	Side       string `json:"side"`
	LimitPx    string `json:"limitPx"`
	Sz         string `json:"sz"`
	OrigSz     string `json:"origSz"`
	Oid        int64  `json:"oid"`
	Timestamp  int64  `json:"timestamp"`
	Cloid      string `json:"cloid"`
	ReduceOnly bool   `json:"reduceOnly"`
	OrderType  any    `json:"orderType"`
}

func buildPlaceOrderAction(orders []exchange.Order) (Action, error) {
	payloads := make([]orderPayload, len(orders))
	for i, order := range orders {
		if err := validateOrder(order); err != nil {
			return Action{}, fmt.Errorf("order[%d]: %w", i, err)
		}
		payload, err := convertOrder(order)
		if err != nil {
			return Action{}, fmt.Errorf("order[%d]: %w", i, err)
		}
		payloads[i] = payload
	}
	return Action{
		Type:     ActionTypeOrder,
		Grouping: "na",
		Orders:   payloads,
	}, nil
}

func buildCancelAction(cancels []Cancel) Action {
	payloads := make([]cancelPayload, len(cancels))
	for i, cancel := range cancels {
		payloads[i] = cancelPayload{Asset: cancel.Asset, Oid: cancel.Oid}
	}
	return Action{
		Type:    ActionTypeCancel,
		Cancels: payloads,
	}
}

// GetOpenOrders returns currently resting orders.
func (c *Client) GetOpenOrders(ctx context.Context) ([]exchange.OrderStatus, error) {
	infoAddr := c.getInfoAddress()
	if infoAddr == "" {
		return nil, fmt.Errorf("hyperliquid: client address unavailable")
	}

	// Use json.RawMessage to capture raw response first
	var raw json.RawMessage
	if err := c.doInfoRequest(ctx, InfoRequest{
		Type: "frontendOpenOrders",
		User: infoAddr,
	}, &raw); err != nil {
		return nil, err
	}

	// Try to determine if response is an array or an object
	var orders []frontendOpenOrder
	trimmed := bytes.TrimSpace(raw)

	if len(trimmed) > 0 && trimmed[0] == '[' {
		// Response is a direct array
		if err := json.Unmarshal(raw, &orders); err != nil {
			return nil, fmt.Errorf("hyperliquid: decode open orders array: %w", err)
		}
	} else {
		// Response is an object with status and data fields
		var resp struct {
			Status string              `json:"status"`
			Data   []frontendOpenOrder `json:"data"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, fmt.Errorf("hyperliquid: decode open orders object: %w", err)
		}
		if strings.ToLower(resp.Status) != "ok" {
			return nil, fmt.Errorf("hyperliquid: frontendOpenOrders status %q", resp.Status)
		}
		orders = resp.Data
	}

	results := make([]exchange.OrderStatus, 0, len(orders))
	for _, raw := range orders {
		status := exchange.OrderStatus{
			Order: exchange.OrderInfo{
				Coin:      raw.Coin,
				Side:      raw.Side,
				LimitPx:   raw.LimitPx,
				Sz:        raw.Sz,
				Oid:       raw.Oid,
				Timestamp: raw.Timestamp,
				OrigSz:    raw.OrigSz,
				Cloid:     raw.Cloid,
			},
			Status:          "open",
			StatusTimestamp: raw.Timestamp,
		}
		results = append(results, status)
	}
	return results, nil
}

// validateOrder ensures order parameters meet basic exchange constraints.
func validateOrder(order exchange.Order) error {
	if order.Asset < 0 {
		return errInvalidAsset
	}
	if strings.TrimSpace(order.Sz) == "" || !isPositiveDecimal(order.Sz) {
		return errInvalidSize
	}
	// Accept trigger-only orders without a limit price
	if order.OrderType.Trigger != nil || strings.TrimSpace(order.TriggerPx) != "" {
		// For trigger orders, require a valid trigger price
		if !isPositiveDecimal(order.TriggerPx) {
			return fmt.Errorf("hyperliquid: trigger price must be positive")
		}
	} else {
		// Otherwise require a valid limit price
		if strings.TrimSpace(order.LimitPx) == "" || !isPositiveDecimal(order.LimitPx) {
			return errInvalidPrice
		}
	}
	if len(order.Cloid) > 128 {
		return fmt.Errorf("hyperliquid: cloid longer than 128 characters")
	}
	return nil
}

func isPositiveDecimal(value string) bool {
	v := new(big.Rat)
	if _, ok := v.SetString(strings.TrimSpace(value)); !ok {
		return false
	}
	return v.Sign() > 0
}

func isZeroDecimal(value string) bool {
	s := strings.TrimSpace(value)
	if s == "" {
		return true
	}
	s = strings.TrimPrefix(s, "+")
	s = strings.TrimPrefix(s, "-")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.TrimLeft(s, "0")
	return s == ""
}

func convertOrder(order exchange.Order) (orderPayload, error) {
	// Build base payload
	payload := orderPayload{
		Asset:      order.Asset,
		IsBuy:      order.IsBuy,
		LimitPx:    order.LimitPx,
		Sz:         order.Sz,
		ReduceOnly: order.ReduceOnly,
		Cloid:      order.Cloid,
	}

	// Prefer explicit trigger order if provided
	if order.OrderType.Trigger != nil || (strings.TrimSpace(order.TriggerPx) != "" && order.OrderType.Limit == nil) {
		if strings.TrimSpace(order.TriggerPx) == "" {
			return orderPayload{}, fmt.Errorf("hyperliquid: trigger order requires trigger price")
		}
		payload.OrderType = orderTypePayload{
			Trigger: &triggerOrderPayload{
				IsMarket:  order.OrderType.Trigger != nil && order.OrderType.Trigger.IsMarket,
				TriggerPx: order.TriggerPx,
				Tpsl: func() string {
					if order.OrderType.Trigger != nil {
						return order.OrderType.Trigger.Tpsl
					}
					return ""
				}(),
				TriggerRel: order.TriggerRel,
			},
		}
		// HL expects triggerPx inside orderType.trigger. Do not set top-level fields.
		return payload, nil
	}

	// Fallback to limit order
	if order.OrderType.Limit == nil {
		return orderPayload{}, fmt.Errorf("hyperliquid: order type not specified (limit or trigger)")
	}
	payload.OrderType = orderTypePayload{
		Limit: &limitOrderPayload{TIF: order.OrderType.Limit.TIF},
	}
	// Do not set top-level TriggerPx/TriggerRel for limit orders as they are not documented in the API
	return payload, nil
}

func aggressiveLimitPrice(isBuy bool) string {
	if isBuy {
		return defaultAggressiveBuyLimit
	}
	return defaultAggressiveSellLimit
}
