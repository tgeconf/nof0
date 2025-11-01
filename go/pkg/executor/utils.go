package executor

import (
	"strings"
)

// sanitizeResponse performs minimal cleanup prior to parsing.
func sanitizeResponse(s string) string {
	s = strings.TrimSpace(s)
	// strip UTF-8 BOM if present
	s = strings.TrimPrefix(s, "\uFEFF")
	return s
}

// decisionContract mirrors the structured JSON contract expected from the LLM.
type decisionContract struct {
	Signal                string  `json:"signal"`
	Symbol                string  `json:"symbol"`
	Leverage              int     `json:"leverage"`
	PositionSizeUSD       float64 `json:"position_size_usd"`
	EntryPrice            float64 `json:"entry_price"`
	StopLoss              float64 `json:"stop_loss"`
	TakeProfit            float64 `json:"take_profit"`
	RiskUSD               float64 `json:"risk_usd"`
	Confidence            int     `json:"confidence"`
	InvalidationCondition string  `json:"invalidation_condition"`
	Reasoning             string  `json:"reasoning"`
}

// mapDecisionContract converts the LLM contract into internal Decision format.
func mapDecisionContract(d decisionContract, positions []PositionInfo) Decision {
	action := strings.ToLower(strings.TrimSpace(d.Signal))
	mapped := "hold"
	switch action {
	case "buy_to_enter":
		mapped = "open_long"
	case "sell_to_enter":
		mapped = "open_short"
	case "hold":
		mapped = "hold"
	case "close":
		// Infer side from current positions
		side := inferSide(positions, d.Symbol)
		if side == "short" {
			mapped = "close_short"
		} else {
			mapped = "close_long"
		}
	default:
		// leave as hold
	}
	return Decision{
		Symbol:                d.Symbol,
		Action:                mapped,
		Leverage:              d.Leverage,
		PositionSizeUSD:       d.PositionSizeUSD,
		EntryPrice:            d.EntryPrice,
		StopLoss:              d.StopLoss,
		TakeProfit:            d.TakeProfit,
		Confidence:            d.Confidence,
		RiskUSD:               d.RiskUSD,
		Reasoning:             d.Reasoning,
		InvalidationCondition: d.InvalidationCondition,
	}
}

func inferSide(positions []PositionInfo, symbol string) string {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	for _, p := range positions {
		if strings.EqualFold(p.Symbol, sym) {
			s := strings.ToLower(p.Side)
			if s == "short" {
				return "short"
			}
			return "long"
		}
	}
	return ""
}

func isBTCETH(sym string) bool {
	s := strings.ToUpper(strings.TrimSpace(sym))
	return s == "BTC" || s == "ETH"
}
