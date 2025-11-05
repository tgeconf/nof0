package hyperliquid

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"nof0-api/pkg/exchange"
)

// GetPositions returns live positions by delegating to account state.
func (c *Client) GetPositions(ctx context.Context) ([]exchange.Position, error) {
	state, err := c.GetAccountState(ctx)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, nil
	}
	return state.AssetPositions, nil
}

// ClosePosition submits a reduce-only order to flatten the specified coin.
func (c *Client) ClosePosition(ctx context.Context, coin string) (*exchange.OrderResponse, error) {
	positions, err := c.GetPositions(ctx)
	if err != nil {
		return nil, err
	}
	var target *exchange.Position
	for i := range positions {
		if strings.EqualFold(positions[i].Coin, coin) {
			target = &positions[i]
			break
		}
	}
	if target == nil {
		return nil, nil
	}

	assetIdx, err := c.GetAssetIndex(ctx, coin)
	if err != nil {
		return nil, err
	}
	info, err := c.GetAssetInfo(ctx, coin)
	if err != nil {
		return nil, err
	}
	order, shouldExecute, err := buildCloseOrder(assetIdx, info.MarkPx, *target)
	if err != nil {
		return nil, err
	}
	if !shouldExecute {
		return nil, nil
	}
	resp, err := c.PlaceOrder(ctx, order)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdateLeverage adjusts leverage for a given asset.
func (c *Client) UpdateLeverage(ctx context.Context, asset int, isCross bool, leverage int) error {
	if leverage <= 0 {
		return fmt.Errorf("hyperliquid: leverage must be positive")
	}
	action := Action{
		Type:     ActionTypeUpdateLeverage,
		Asset:    &asset,
		IsCross:  &isCross,
		Leverage: leverage,
	}
	return c.doExchangeRequest(ctx, action, nil)
}

func buildCloseOrder(assetIdx int, markPx string, pos exchange.Position) (exchange.Order, bool, error) {
	rawSize := strings.TrimSpace(pos.Szi)
	if rawSize == "" || isZeroDecimal(rawSize) {
		return exchange.Order{}, false, nil
	}

	isShort := strings.HasPrefix(rawSize, "-")
	size := trimSign(rawSize)
	if size == "" || isZeroDecimal(size) {
		return exchange.Order{}, false, nil
	}

	limit := computeCloseLimit(markPx, isShort)

	order := exchange.Order{
		Asset:      assetIdx,
		IsBuy:      isShort,
		LimitPx:    limit,
		Sz:         size,
		ReduceOnly: true,
		OrderType: exchange.OrderType{
			Limit: &exchange.LimitOrderType{TIF: "Ioc"},
		},
	}
	return order, true, nil
}

const closePriceSlippage = 0.005

var (
	closeMultiplierBuy  = big.NewRat(1005, 1000)
	closeMultiplierSell = big.NewRat(995, 1000)
)

func computeCloseLimit(mark string, isBuy bool) string {
	trimmed := strings.TrimSpace(mark)
	if trimmed != "" && isPositiveDecimal(trimmed) {
		price := new(big.Rat)
		if _, ok := price.SetString(trimmed); ok && price.Sign() > 0 {
			multiplier := closeMultiplierSell
			if isBuy {
				multiplier = closeMultiplierBuy
			}
			result := new(big.Rat).Mul(price, multiplier)
			if result.Sign() > 0 {
				// Round to 5 significant figures for submission consistency
				f, _ := new(big.Rat).Set(result).Float64()
				if f > 0 {
					return RoundPriceToSigFigs(f, 5)
				}
			}
		}
	}
	return aggressiveLimitPrice(isBuy)
}

func trimSign(value string) string {
	s := strings.TrimSpace(value)
	for len(s) > 0 {
		if s[0] == '+' || s[0] == '-' {
			s = strings.TrimSpace(s[1:])
			continue
		}
		break
	}
	return s
}

func decimalsForString(value string) int {
	value = strings.TrimSpace(value)
	if idx := strings.Index(value, "."); idx >= 0 {
		return len(value[idx+1:])
	}
	return 0
}

func trimTrailingZeros(value string) string {
	if value == "" {
		return value
	}
	value = strings.TrimRight(value, "0")
	value = strings.TrimRight(value, ".")
	if value == "" {
		return "0"
	}
	return value
}
