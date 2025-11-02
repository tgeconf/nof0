package executor

import (
	"fmt"
	"strings"
	"time"
)

// ValidateDecisions applies sanity checks against configuration and current context.
func ValidateDecisions(cfg *Config, ctx *Context, decisions []Decision) error {
	if cfg == nil {
		return fmt.Errorf("executor: missing config for validation")
	}
	for i, d := range decisions {
		action := strings.TrimSpace(d.Action)
		symbol := strings.TrimSpace(d.Symbol)
		switch action {
		case "open_long", "open_short":
			if symbol == "" {
				return fmt.Errorf("decision[%d]: symbol is required", i)
			}
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
			if action == "open_long" {
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
			// Leverage caps (config) and asset-level cap if available; take the minimum
			capLev := cfg.AltcoinLeverage
			if isBTCETH(d.Symbol) {
				capLev = cfg.MajorCoinLeverage
			}
			if ctx != nil && ctx.AssetMeta != nil {
				if meta, ok := ctx.AssetMeta[d.Symbol]; ok && meta.MaxLeverage > 0 {
					ml := int(meta.MaxLeverage)
					if ml < capLev {
						capLev = ml
					}
				}
			}
			if d.Leverage > capLev {
				return fmt.Errorf("decision[%d]: leverage %dx exceeds cap %dx", i, d.Leverage, capLev)
			}

			// Extended guards (enabled only when context provides non-zero values)
			if ctx != nil {
				// Liquidity threshold: OI * Price ≥ threshold (new opens only)
				if ctx.LiquidityThresholdUSD > 0 && ctx.MarketDataMap != nil {
					if snap, ok := ctx.MarketDataMap[d.Symbol]; ok && snap != nil && snap.OpenInterest != nil && snap.Price.Last > 0 {
						oiValueUSD := snap.OpenInterest.Latest * snap.Price.Last
						if oiValueUSD+1e-9 < ctx.LiquidityThresholdUSD {
							return fmt.Errorf("decision[%d]: %s illiquid: oi*price %.2f < threshold %.2f", i, d.Symbol, oiValueUSD, ctx.LiquidityThresholdUSD)
						}
					}
				}

				// Position value band by category (equity multiples)
				if ctx.Account.TotalEquity > 0 {
					equity := ctx.Account.TotalEquity
					if isBTCETH(d.Symbol) {
						if ctx.BTCETHPositionValueMinMultiple > 0 {
							minV := equity * ctx.BTCETHPositionValueMinMultiple
							if d.PositionSizeUSD+1e-9 < minV {
								return fmt.Errorf("decision[%d]: position_size_usd %.2f below BTC/ETH min %.2f (%.2fx equity)", i, d.PositionSizeUSD, minV, ctx.BTCETHPositionValueMinMultiple)
							}
						}
						if ctx.BTCETHPositionValueMaxMultiple > 0 {
							maxV := equity * ctx.BTCETHPositionValueMaxMultiple
							if d.PositionSizeUSD-1e-9 > maxV {
								return fmt.Errorf("decision[%d]: position_size_usd %.2f exceeds BTC/ETH max %.2f (%.2fx equity)", i, d.PositionSizeUSD, maxV, ctx.BTCETHPositionValueMaxMultiple)
							}
						}
					} else {
						if ctx.AltPositionValueMinMultiple > 0 {
							minV := equity * ctx.AltPositionValueMinMultiple
							if d.PositionSizeUSD+1e-9 < minV {
								return fmt.Errorf("decision[%d]: position_size_usd %.2f below alt min %.2f (%.2fx equity)", i, d.PositionSizeUSD, minV, ctx.AltPositionValueMinMultiple)
							}
						}
						if ctx.AltPositionValueMaxMultiple > 0 {
							maxV := equity * ctx.AltPositionValueMaxMultiple
							if d.PositionSizeUSD-1e-9 > maxV {
								return fmt.Errorf("decision[%d]: position_size_usd %.2f exceeds alt max %.2f (%.2fx equity)", i, d.PositionSizeUSD, maxV, ctx.AltPositionValueMaxMultiple)
							}
						}
					}
				}

				// Margin usage cap after new position margin
				if ctx.MaxMarginUsagePct > 0 && ctx.Account.TotalEquity > 0 && d.Leverage > 0 {
					newMargin := d.PositionSizeUSD / float64(d.Leverage)
					used := ctx.Account.MarginUsed + newMargin
					usagePct := 100 * (used / ctx.Account.TotalEquity)
					if usagePct > ctx.MaxMarginUsagePct+1e-9 {
						return fmt.Errorf("decision[%d]: margin usage %.2f%% exceeds cap %.2f%% after new position", i, usagePct, ctx.MaxMarginUsagePct)
					}
				}

				// Cooldown after close
				if ctx.CooldownAfterClose > 0 && ctx.RecentlyClosed != nil {
					if ts, ok := ctx.RecentlyClosed[d.Symbol]; ok && !ts.IsZero() {
						if time.Since(ts) < ctx.CooldownAfterClose {
							return fmt.Errorf("decision[%d]: %s in cooldown window (%s remaining)", i, d.Symbol, (ctx.CooldownAfterClose - time.Since(ts)).Truncate(time.Second))
						}
					}
				}
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
			if symbol == "" {
				return fmt.Errorf("decision[%d]: symbol is required", i)
			}
			if ctx == nil {
				return fmt.Errorf("decision[%d]: context required to validate close action", i)
			}
			wantSide := "long"
			if action == "close_short" {
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
