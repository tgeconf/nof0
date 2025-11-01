package executor

import (
	"fmt"
	"strings"
)

// ValidateDecisions applies sanity checks against configuration and current context.
func ValidateDecisions(cfg *Config, ctx *Context, decisions []Decision) error {
	if cfg == nil {
		return fmt.Errorf("executor: missing config for validation")
	}
	for i, d := range decisions {
		if strings.TrimSpace(d.Symbol) == "" {
			return fmt.Errorf("decision[%d]: symbol is required", i)
		}
		switch d.Action {
		case "open_long", "open_short":
			if d.Leverage <= 0 {
				return fmt.Errorf("decision[%d]: leverage must be positive", i)
			}
			if d.PositionSizeUSD <= 0 {
				return fmt.Errorf("decision[%d]: position_size_usd must be positive", i)
			}
			if d.StopLoss <= 0 || d.TakeProfit <= 0 || d.EntryPrice <= 0 {
				return fmt.Errorf("decision[%d]: entry/stop_loss/take_profit must be positive", i)
			}
			if d.Confidence < 0 || d.Confidence > 100 {
				return fmt.Errorf("decision[%d]: confidence must be 0-100", i)
			}
			if d.Confidence < cfg.MinConfidence {
				return fmt.Errorf("decision[%d]: confidence below threshold", i)
			}
			// Price relationship & RR check
			if d.Action == "open_long" {
				if !(d.TakeProfit > d.EntryPrice && d.EntryPrice > d.StopLoss) {
					return fmt.Errorf("decision[%d]: long requires TP>entry>SL", i)
				}
				rr := (d.TakeProfit - d.EntryPrice) / (d.EntryPrice - d.StopLoss)
				if rr < cfg.MinRiskReward {
					return fmt.Errorf("decision[%d]: reward/risk %.2f below min %.2f", i, rr, cfg.MinRiskReward)
				}
			} else { // open_short
				if !(d.StopLoss > d.EntryPrice && d.EntryPrice > d.TakeProfit) {
					return fmt.Errorf("decision[%d]: short requires SL>entry>TP", i)
				}
				rr := (d.EntryPrice - d.TakeProfit) / (d.StopLoss - d.EntryPrice)
				if rr < cfg.MinRiskReward {
					return fmt.Errorf("decision[%d]: reward/risk %.2f below min %.2f", i, rr, cfg.MinRiskReward)
				}
			}
			// Leverage caps
			if isBTCETH(d.Symbol) {
				if d.Leverage > cfg.BTCETHLeverage {
					return fmt.Errorf("decision[%d]: leverage %dx exceeds BTC/ETH cap %dx", i, d.Leverage, cfg.BTCETHLeverage)
				}
			} else if d.Leverage > cfg.AltcoinLeverage {
				return fmt.Errorf("decision[%d]: leverage %dx exceeds alt cap %dx", i, d.Leverage, cfg.AltcoinLeverage)
			}
			// Position count
			if ctx != nil && len(ctx.Positions) >= cfg.MaxPositions {
				return fmt.Errorf("decision[%d]: max_positions reached (%d)", i, cfg.MaxPositions)
			}
			// No pyramiding / hedging: disallow opening if any position already exists on the symbol
			if ctx != nil {
				for _, p := range ctx.Positions {
					if strings.EqualFold(p.Symbol, d.Symbol) {
						return fmt.Errorf("decision[%d]: position already exists on %s; no add/hedge allowed", i, d.Symbol)
					}
				}
			}
			// Risk and size caps from context (if provided by Manager)
			if ctx != nil && ctx.Account.TotalEquity > 0 && ctx.MaxRiskPct > 0 {
				maxRiskUSD := ctx.Account.TotalEquity * (ctx.MaxRiskPct / 100.0)
				if d.RiskUSD > maxRiskUSD+1e-9 { // small epsilon
					return fmt.Errorf("decision[%d]: risk_usd %.2f exceeds max %.2f (%.2f%% of equity)", i, d.RiskUSD, maxRiskUSD, ctx.MaxRiskPct)
				}
			}
			if ctx != nil && ctx.MaxPositionSizeUSD > 0 {
				if d.PositionSizeUSD > ctx.MaxPositionSizeUSD+1e-9 {
					return fmt.Errorf("decision[%d]: position_size_usd %.2f exceeds cap %.2f", i, d.PositionSizeUSD, ctx.MaxPositionSizeUSD)
				}
			}
		case "close_long", "close_short":
			if ctx == nil {
				return fmt.Errorf("decision[%d]: context required to validate close action", i)
			}
			wantSide := "long"
			if d.Action == "close_short" {
				wantSide = "short"
			}
			has := false
			for _, p := range ctx.Positions {
				if strings.EqualFold(p.Symbol, d.Symbol) && strings.EqualFold(p.Side, wantSide) {
					has = true
					break
				}
			}
			if !has {
				return fmt.Errorf("decision[%d]: no matching %s position to close for %s", i, wantSide, d.Symbol)
			}
		case "hold", "wait":
			// ok
		default:
			return fmt.Errorf("decision[%d]: unknown action %q", i, d.Action)
		}
	}
	return nil
}

// isBTCETH moved to utils.go for single definition
