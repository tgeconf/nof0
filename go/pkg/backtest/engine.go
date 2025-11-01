package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"

	"nof0-api/pkg/exchange"
	"nof0-api/pkg/market"
)

// Feeder yields sequential market snapshots for a symbol.
type Feeder interface {
	Next(ctx context.Context, symbol string) (*market.Snapshot, bool, error)
}

// Strategy maps a snapshot into a set of orders to place on the exchange.
type Strategy interface {
	Decide(ctx context.Context, snap *market.Snapshot) ([]exchange.Order, error)
}

// Engine wires a Feeder, a Strategy, and an exchange.Provider to simulate a session.
type Engine struct {
	Feeder   Feeder
	Strategy Strategy
	Exch     exchange.Provider
	Symbol   string

	// Optional simulation parameters
	InitialEquity float64 // defaults to 100000 if zero
	FeeBps        float64 // per-trade fee in basis points (e.g., 2.0 for 0.02%)
	SlippageBps   float64 // execution slippage in bps applied to mid/last

	// Optional: write JSON report to this path
	OutputPath string
}

// Result summarizes a simulation run.
type Result struct {
	Steps       int
	OrdersSent  int
	Trades      int
	Wins        int
	WinRate     float64
	RealizedPNL float64
	UnrealPNL   float64
	TotalPNL    float64
	MaxDDPct    float64
	Sharpe      float64
	EquityCurve []float64
	Details     []TradeDetail
}

func (e *Engine) Run(ctx context.Context) (*Result, error) {
	if e.Feeder == nil || e.Strategy == nil || e.Exch == nil || e.Symbol == "" {
		return nil, fmt.Errorf("backtest: engine not fully configured")
	}
	res := &Result{}
	eq0 := e.InitialEquity
	if eq0 <= 0 {
		eq0 = 100000
	}
	pf := &portfolio{cash: eq0, feeBps: e.FeeBps, slippageBps: e.SlippageBps}
	lastEquity := eq0
	for {
		snap, ok, err := e.Feeder.Next(ctx, e.Symbol)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		res.Steps++
		orders, err := e.Strategy.Decide(ctx, snap)
		if err != nil {
			return nil, err
		}
		px := snap.Price.Last
		for _, ord := range orders {
			// Derive numeric size & execution price
			sz, _ := strconv.ParseFloat(ord.Sz, 64)
			execPx := applySlippage(px, e.SlippageBps, ord.IsBuy)
			realized, fee, tradeCompleted := pf.apply(ord.IsBuy, execPx, sz)
			if tradeCompleted {
				res.Trades++
				if realized > 0 {
					res.Wins++
				}
			}
			if _, err := e.Exch.PlaceOrder(ctx, ord); err != nil {
				return nil, err
			}
			res.OrdersSent++
			// Append detail for every order
			res.Details = append(res.Details, TradeDetail{
				Step:     res.Steps,
				Side:     sideStr(ord.IsBuy),
				Price:    execPx,
				Qty:      sz,
				Fee:      fee,
				Realized: realized,
				Position: pf.pos,
			})
		}
		equity := pf.equity(px)
		res.EquityCurve = append(res.EquityCurve, equity)
		lastEquity = equity
	}
	res.RealizedPNL = pf.realized
	res.UnrealPNL = pf.unrealized
	res.TotalPNL = res.RealizedPNL + res.UnrealPNL
	if res.Trades > 0 {
		res.WinRate = float64(res.Wins) / float64(res.Trades)
	}
	res.MaxDDPct = maxDrawdownPct(append([]float64{eq0}, res.EquityCurve...))
	res.Sharpe = sharpe(res.EquityCurve)

	if e.OutputPath != "" {
		if err := writeReport(e.OutputPath, res); err != nil {
			return nil, err
		}
	}
	_ = lastEquity
	return res, nil
}

func applySlippage(px, bps float64, isBuy bool) float64 {
	if bps == 0 {
		return px
	}
	m := 1 + bps/10000.0
	if isBuy {
		return px * m
	}
	return px / m
}

func maxDrawdownPct(series []float64) float64 {
	peak := series[0]
	mdd := 0.0
	for _, v := range series {
		if v > peak {
			peak = v
		}
		dd := (peak - v) / peak
		if dd > mdd {
			mdd = dd
		}
	}
	return mdd * 100
}

func sharpe(equity []float64) float64 {
	if len(equity) < 2 {
		return 0
	}
	rets := make([]float64, 0, len(equity)-1)
	for i := 1; i < len(equity); i++ {
		if equity[i-1] == 0 {
			continue
		}
		rets = append(rets, equity[i]/equity[i-1]-1)
	}
	if len(rets) == 0 {
		return 0
	}
	m := 0.0
	for _, r := range rets {
		m += r
	}
	m /= float64(len(rets))
	v := 0.0
	for _, r := range rets {
		d := r - m
		v += d * d
	}
	v /= float64(len(rets))
	sd := math.Sqrt(v)
	if sd == 0 {
		return 0
	}
	return m / sd * math.Sqrt(float64(len(rets)))
}

func writeReport(path string, r *Result) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// TradeDetail records per-order execution for analysis.
type TradeDetail struct {
	Step     int     `json:"step"`
	Side     string  `json:"side"` // buy/sell
	Price    float64 `json:"price"`
	Qty      float64 `json:"qty"`
	Fee      float64 `json:"fee"`
	Realized float64 `json:"realized"` // realized PnL contributed by this order
	Position float64 `json:"position"` // signed position after this order
}

func sideStr(buy bool) string {
	if buy {
		return "buy"
	}
	return "sell"
}
