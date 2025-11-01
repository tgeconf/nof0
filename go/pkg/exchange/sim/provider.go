package sim

import (
    "context"
    "fmt"
    "math/big"
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

    // Update position size using precise decimal math on strings to avoid
    // floating point artefacts in tests (e.g. 0.010000000000000002).
    pos := p.positions[coin]
    pos.Coin = coin
    pos.Szi = addSignedDecimal(pos.Szi, order.Sz, order.IsBuy)
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
    // Ensure the asset index exists to allow immediate use in orders when
    // callers configure leverage before requesting an asset index.
    found := false
    for _, id := range p.assetIndex {
        if id == asset {
            found = true
            break
        }
    }
    if !found {
        // Create a placeholder coin mapping for this asset id.
        key := fmt.Sprintf("ASSET_%d", asset)
        if _, ok := p.assetIndex[key]; !ok {
            p.assetIndex[key] = asset
        }
    }
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

// addSignedDecimal adds or subtracts two base-10 string decimals and returns a
// canonical string representation with minimal trailing zeros. If the result
// is an integer and the right-hand operand contained a decimal point, a single
// trailing ".0" is preserved to match test expectations.
func addSignedDecimal(current, delta string, add bool) string {
    // Normalize inputs
    curInt, curScale := toScaledInt(current)
    delInt, delScale := toScaledInt(delta)
    // Use the scale of the delta to influence formatting when integral.
    rhsHadDecimal := strings.Contains(delta, ".")

    // Align scales
    scale := curScale
    if delScale > scale {
        scale = delScale
    }
    curInt = scaleUp(curInt, scale-curScale)
    delInt = scaleUp(delInt, scale-delScale)
    if !add {
        delInt.Neg(&delInt)
    }

    // Sum
    sum := new(big.Int).Add(&curInt, &delInt)

    // Format with minimal trailing zeros; keep one decimal if rhs had decimal and result integral
    s := fromScaledInt(*sum, scale)
    if strings.Contains(s, ".") {
        // Trim trailing zeros
        s = strings.TrimRight(s, "0")
        if strings.HasSuffix(s, ".") {
            if rhsHadDecimal {
                s += "0"
            } else {
                s = strings.TrimSuffix(s, ".")
            }
        }
    }
    return s
}

// Helpers for scaled integer decimal arithmetic
// (implemented locally to avoid external dependencies)

// toScaledInt parses a base-10 decimal string into an integer and scale.
// "1.230" -> (1230, 3). Empty or invalid strings treat as 0.
func toScaledInt(s string) (big.Int, int) {
    s = strings.TrimSpace(s)
    if s == "" {
        return *big.NewInt(0), 0
    }
    neg := false
    if strings.HasPrefix(s, "+") {
        s = s[1:]
    } else if strings.HasPrefix(s, "-") {
        neg = true
        s = s[1:]
    }
    scale := 0
    if dot := strings.IndexByte(s, '.'); dot >= 0 {
        scale = len(s) - dot - 1
        s = s[:dot] + s[dot+1:]
    }
    // Remove leading zeros to keep big.Int small
    s = strings.TrimLeft(s, "0")
    if s == "" {
        return *big.NewInt(0), scale
    }
    n := new(big.Int)
    if _, ok := n.SetString(s, 10); !ok {
        return *big.NewInt(0), 0
    }
    if neg {
        n.Neg(n)
    }
    return *n, scale
}

func scaleUp(n big.Int, by int) big.Int {
    if by <= 0 {
        return n
    }
    ten := big.NewInt(10)
    m := new(big.Int).Set(&n)
    for i := 0; i < by; i++ {
        m.Mul(m, ten)
    }
    return *m
}

func fromScaledInt(n big.Int, scale int) string {
    neg := n.Sign() < 0
    if neg {
        n.Neg(&n)
    }
    s := n.String()
    if scale == 0 {
        if neg {
            return "-" + s
        }
        return s
    }
    // Ensure string has at least scale+1 digits
    if len(s) <= scale {
        s = strings.Repeat("0", scale-len(s)+1) + s
    }
    i := len(s) - scale
    out := s[:i] + "." + s[i:]
    if neg {
        out = "-" + out
    }
    return out
}
