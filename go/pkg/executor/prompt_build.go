package executor

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	market "nof0-api/pkg/market"
)

// buildPromptInputs renders dynamic sections used by the executor prompt template.
func buildPromptInputs(cfg *Config, ctx *Context) PromptInputs {
	now := time.Now().UTC().Format(time.RFC3339)
	current := ctx.CurrentTime
	if strings.TrimSpace(current) == "" {
		current = now
	}

	return PromptInputs{
		CurrentTime:     current,
		RuntimeMinutes:  ctx.RuntimeMinutes,
		SharpeRatio:     safePerf(ctx.Performance).SharpeRatio,
		AccountOverview: formatAccount(ctx.Account),
		OpenPositions:   formatPositions(ctx.Positions),
		RiskBudget:      formatRiskBudget(cfg, ctx),
		PerformanceView: formatPerformance(ctx.Performance),
		CandidateCoins:  formatCandidates(ctx.CandidateCoins),
		MarketSnapshots: formatMarketJSON(ctx.MarketDataMap),
	}
}

func formatAccount(a AccountInfo) string {
	return fmt.Sprintf("equity=%.2f, avail=%.2f, pnl=%.2f (%.2f%%), margin=%.2f (%.2f%%), positions=%d",
		a.TotalEquity, a.AvailableBalance, a.TotalPnL, a.TotalPnLPct, a.MarginUsed, a.MarginUsedPct, a.PositionCount,
	)
}

func formatPositions(positions []PositionInfo) string {
	if len(positions) == 0 {
		return "(none)"
	}
	// Stable sorting for reproducibility
	items := make([]string, 0, len(positions))
	for _, p := range positions {
		items = append(items, fmt.Sprintf("%s %s qty=%.4f lev=%dx entry=%.4f mark=%.4f upnl=%.2f(%.2f%%) liq=%.4f",
			p.Symbol, p.Side, p.Quantity, p.Leverage, p.EntryPrice, p.MarkPrice, p.UnrealizedPnL, p.UnrealizedPnLPct, p.LiquidationPrice,
		))
	}
	sort.Strings(items)
	return strings.Join(items, "\n")
}

func formatCandidates(cands []CandidateCoin) string {
	if len(cands) == 0 {
		return "(none)"
	}
	items := make([]string, 0, len(cands))
	for _, c := range cands {
		src := strings.Join(c.Sources, ",")
		items = append(items, fmt.Sprintf("%s [%s]", c.Symbol, src))
	}
	sort.Strings(items)
	return strings.Join(items, ", ")
}

func formatPerformance(p *PerformanceView) string {
	if p == nil {
		return "(n/a)"
	}
	return fmt.Sprintf("sharpe=%.3f, win_rate=%.1f%%, trades=%d, recent_rate=%.3f, updated=%s",
		p.SharpeRatio, p.WinRate*100, p.TotalTrades, p.RecentTradesRate, p.UpdatedAt.UTC().Format(time.RFC3339),
	)
}

func formatRiskBudget(cfg *Config, ctx *Context) string {
	remaining := cfg.MaxPositions - len(ctx.Positions)
	if remaining < 0 {
		remaining = 0
	}
	return fmt.Sprintf("max_positions=%d (remaining=%d), min_confidence=%d, min_rr=%.2f",
		cfg.MaxPositions, remaining, cfg.MinConfidence, cfg.MinRiskReward,
	)
}

func formatMarketJSON(snaps map[string]*market.Snapshot) string {
	if len(snaps) == 0 {
		return "{}"
	}
	// Reduce payload: include selected fields only
	type Lite struct {
		Price    float64            `json:"price"`
		Change1h float64            `json:"change_1h"`
		Change4h float64            `json:"change_4h"`
		EMA      map[string]float64 `json:"ema,omitempty"`
		RSI      map[string]float64 `json:"rsi,omitempty"`
		MACD     float64            `json:"macd,omitempty"`
		OILatest *float64           `json:"oi_latest,omitempty"`
		Funding  *float64           `json:"funding,omitempty"`
	}
	out := make(map[string]Lite, len(snaps))
	for sym, s := range snaps {
		var oi *float64
		if s.OpenInterest != nil {
			oi = &s.OpenInterest.Latest
		}
		var funding *float64
		if s.Funding != nil {
			funding = &s.Funding.Rate
		}
		out[sym] = Lite{
			Price:    s.Price.Last,
			Change1h: s.Change.OneHour,
			Change4h: s.Change.FourHour,
			EMA:      s.Indicators.EMA,
			RSI:      s.Indicators.RSI,
			MACD:     s.Indicators.MACD,
			OILatest: oi,
			Funding:  funding,
		}
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func safePerf(p *PerformanceView) *PerformanceView {
	if p != nil {
		return p
	}
	return &PerformanceView{}
}
