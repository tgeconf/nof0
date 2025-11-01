package sim

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"nof0-api/pkg/exchange"
)

// Provider is a minimal in-memory simulator implementing exchange.Provider.
type Provider struct {
	mu          sync.Mutex
	nextAssetID int
	// maps canonical coin -> asset index
	assetIndex map[string]int
	// positions by coin symbol (canonical)
	positions map[string]exchange.Position
	// last known price per coin (string value)
	lastPx map[string]string
	// per-asset leverage
	leverage map[int]exchange.Leverage
}

// New constructs a new simulator instance.
func New() *Provider {
	return &Provider{
		nextAssetID: 1,
		assetIndex:  make(map[string]int),
		positions:   make(map[string]exchange.Position),
		lastPx:      make(map[string]string),
		leverage:    make(map[int]exchange.Leverage),
	}
}

func canonical(coin string) string { return strings.ToUpper(strings.TrimSpace(coin)) }

func (p *Provider) GetAssetIndex(ctx context.Context, coin string) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	c := canonical(coin)
	if id, ok := p.assetIndex[c]; ok {
		return id, nil
	}
	id := p.nextAssetID
	p.nextAssetID++
	p.assetIndex[c] = id
	return id, nil
}

func (p *Provider) PlaceOrder(ctx context.Context, order exchange.Order) (*exchange.OrderResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Find coin by asset index
	var coin string
	for c, id := range p.assetIndex {
		if id == order.Asset {
			coin = c
			break
		}
	}
	if coin == "" {
		return nil, fmt.Errorf("sim: unknown asset index %d", order.Asset)
	}

	// Update position size
	pos := p.positions[coin]
	pos.Coin = coin
	// szi is signed float in string form
	cur, _ := strconv.ParseFloat(pos.Szi, 64)
	sz, _ := strconv.ParseFloat(order.Sz, 64)
	if order.IsBuy {
		cur += sz
	} else {
		cur -= sz
	}
	pos.Szi = strconv.FormatFloat(cur, 'f', -1, 64)
	// keep a copy for entry price
	entryPx := order.LimitPx
	pos.EntryPx = &entryPx
	// store last price
	p.lastPx[coin] = order.LimitPx
	// keep leverage if set
	if lev, ok := p.leverage[order.Asset]; ok {
		pos.Leverage = lev
	}
	p.positions[coin] = pos

	// Synchronously mark as filled
	resp := &exchange.OrderResponse{
		Status: "ok",
		Response: exchange.OrderResponseData{
			Type: "order",
			Data: exchange.OrderResponseDataDetail{
				Statuses: []exchange.OrderStatusResponse{{
					Filled: &exchange.FilledOrder{TotalSz: order.Sz, AvgPx: order.LimitPx, Oid: 1},
				}},
			},
		},
	}
	return resp, nil
}

func (p *Provider) CancelOrder(ctx context.Context, asset int, oid int64) error { return nil }

func (p *Provider) GetOpenOrders(ctx context.Context) ([]exchange.OrderStatus, error) {
	return nil, nil
}

func (p *Provider) GetPositions(ctx context.Context) ([]exchange.Position, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]exchange.Position, 0, len(p.positions))
	for _, pos := range p.positions {
		out = append(out, pos)
	}
	return out, nil
}

func (p *Provider) ClosePosition(ctx context.Context, coin string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	c := canonical(coin)
	pos := p.positions[c]
	pos.Szi = "0"
	p.positions[c] = pos
	return nil
}

func (p *Provider) UpdateLeverage(ctx context.Context, asset int, isCross bool, leverage int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.leverage[asset] = exchange.Leverage{Type: map[bool]string{true: "cross", false: "isolated"}[isCross], Value: leverage}
	return nil
}

func (p *Provider) GetAccountState(ctx context.Context) (*exchange.AccountState, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	state := &exchange.AccountState{
		MarginSummary:      exchange.MarginSummary{AccountValue: "100000", TotalMarginUsed: "0", TotalNtlPos: "0", TotalRawUSD: "0"},
		CrossMarginSummary: exchange.CrossMarginSummary{AccountValue: "100000", TotalMarginUsed: "0", TotalNtlPos: "0", TotalRawUSD: "0"},
	}
	for _, pos := range p.positions {
		state.AssetPositions = append(state.AssetPositions, pos)
	}
	return state, nil
}

func (p *Provider) GetAccountValue(ctx context.Context) (float64, error) {
	return 100000.0, nil
}

// Registry hook for exchange.Config
func init() {
	exchange.RegisterProvider("sim", func(name string, cfg *exchange.ProviderConfig) (exchange.Provider, error) {
		return New(), nil
	})
}
