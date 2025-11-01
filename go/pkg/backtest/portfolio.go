package backtest

import (
	"math"
)

// portfolio tracks PnL with simple fee/slippage.
type portfolio struct {
	cash        float64
	pos         float64 // signed size in base units
	avgCost     float64 // average price of current position
	realized    float64
	unrealized  float64
	feeBps      float64
	slippageBps float64
}

// apply processes an order at given execution price and quantity.
// Returns (realized PnL for closed portion, fee charged, tradeCompleted-when-close-occurs).
func (p *portfolio) apply(isBuy bool, execPx float64, qty float64) (realized float64, fee float64, tradeCompleted bool) {
	if qty <= 0 || execPx <= 0 {
		return 0, 0, false
	}
	side := 1.0
	if !isBuy {
		side = -1.0
	}
	// trading fee in quote currency
	fee = p.fee(execPx, qty)
	// existing position sign
	posSign := 0.0
	if p.pos > 0 {
		posSign = 1
	} else if p.pos < 0 {
		posSign = -1
	}

	// If position increases in same direction
	if posSign == 0 || posSign == side {
		newPos := p.pos + side*qty
		// weighted avg cost update
		if p.pos == 0 {
			p.avgCost = execPx
		} else {
			totalQty := math.Abs(p.pos) + qty
			if totalQty > 0 {
				p.avgCost = (p.avgCost*math.Abs(p.pos) + execPx*qty) / totalQty
			}
		}
		p.pos = newPos
		p.cash -= fee
		return 0, fee, false
	}

	// Opposite direction: close part or all
	closeQty := math.Min(math.Abs(p.pos), qty)
	// realized PnL depends on previous sign
	if p.pos > 0 { // closing long by selling
		realized = (execPx - p.avgCost) * closeQty
	} else { // closing short by buying
		realized = (p.avgCost - execPx) * closeQty
	}
	p.cash += realized
	p.realized += realized
	p.cash -= fee
	tradeCompleted = true

	remaining := qty - closeQty
	if remaining == 0 {
		// position may go to zero or reduce
		if closeQty == math.Abs(p.pos) {
			p.pos = 0
			p.avgCost = 0
		} else {
			// reduced but same sign remains
			if p.pos > 0 {
				p.pos -= closeQty
			} else {
				p.pos += closeQty
			}
		}
		return realized, fee, tradeCompleted
	}
	// Crossed through zero: open new position with remaining in the new side
	if closeQty == math.Abs(p.pos) {
		p.pos = 0
		p.avgCost = 0
	} else {
		if p.pos > 0 {
			p.pos -= closeQty
		} else {
			p.pos += closeQty
		}
	}
	// Open new position with remaining amount
	if side > 0 { // buy
		p.pos += remaining
	} else {
		p.pos -= remaining
	}
	p.avgCost = execPx
	return realized, fee, tradeCompleted
}

func (p *portfolio) equity(lastPx float64) float64 {
	p.unrealized = 0
	if p.pos > 0 {
		p.unrealized = (lastPx - p.avgCost) * p.pos
	} else if p.pos < 0 {
		p.unrealized = (p.avgCost - lastPx) * math.Abs(p.pos)
	}
	return p.cash + p.unrealized
}

func (p *portfolio) fee(px, qty float64) float64 {
	if p.feeBps == 0 {
		return 0
	}
	return px * qty * (p.feeBps / 10000.0)
}
