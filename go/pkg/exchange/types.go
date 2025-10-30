package exchange

// Core trading domain types shared across exchange implementations.
// These structures mirror the Hyperliquid API payloads while remaining exchange-agnostic
// to keep the public interface consistent if additional venues are added later.

// OrderSide represents order direction.
type OrderSide string

const (
	// OrderSideBuy executes a buy.
	OrderSideBuy OrderSide = "buy"
	// OrderSideSell executes a sell.
	OrderSideSell OrderSide = "sell"
)

// OrderType captures optional order configuration such as limit parameters.
type OrderType struct {
	Limit *LimitOrderType `json:"limit,omitempty"`
}

// LimitOrderType defines limit order specific fields.
type LimitOrderType struct {
	TIF string `json:"tif"` // Valid values: "Alo", "Ioc", "Gtc"
}

// Order describes a normalized order request.
type Order struct {
	Asset      int       `json:"asset"`                // Exchange-specific asset index.
	IsBuy      bool      `json:"isBuy"`                // true for buy, false for sell.
	LimitPx    string    `json:"limitPx"`              // Limit price as string to avoid precision loss.
	Sz         string    `json:"sz"`                   // Order size as string to avoid precision loss.
	ReduceOnly bool      `json:"reduceOnly"`           // Indicates whether the order only reduces position.
	OrderType  OrderType `json:"orderType"`            // Order execution parameters.
	Cloid      string    `json:"cloid,omitempty"`      // Optional client order identifier.
	TriggerPx  string    `json:"triggerPx,omitempty"`  // Optional trigger price for advanced orders.
	TriggerRel string    `json:"triggerRel,omitempty"` // Optional trigger relation (e.g. "lte").
}

// Position captures live position details.
type Position struct {
	Coin           string   `json:"coin"`
	EntryPx        string   `json:"entryPx"`
	PositionValue  string   `json:"positionValue"`
	Szi            string   `json:"szi"`            // Signed position size.
	UnrealizedPnl  string   `json:"unrealizedPnl"`  // Unrealised profit & loss.
	ReturnOnEquity string   `json:"returnOnEquity"` // ROE in percentage string.
	Leverage       Leverage `json:"leverage"`
	LiquidationPx  string   `json:"liquidationPx,omitempty"`
}

// Leverage contains leverage settings for an instrument.
type Leverage struct {
	Type  string `json:"type"`  // "cross" or "isolated".
	Value int    `json:"value"` // Leverage multiplier.
}

// AccountState summarizes a trading account.
type AccountState struct {
	MarginSummary      MarginSummary      `json:"marginSummary"`
	CrossMarginSummary CrossMarginSummary `json:"crossMarginSummary"`
	AssetPositions     []Position         `json:"assetPositions"`
}

// MarginSummary consolidates margin metrics.
type MarginSummary struct {
	AccountValue    string `json:"accountValue"`
	TotalMarginUsed string `json:"totalMarginUsed"`
	TotalNtlPos     string `json:"totalNtlPos"`
	TotalRawUSD     string `json:"totalRawUsd"`
}

// CrossMarginSummary mirrors margin data for cross mode.
type CrossMarginSummary struct {
	AccountValue    string `json:"accountValue"`
	TotalMarginUsed string `json:"totalMarginUsed"`
	TotalNtlPos     string `json:"totalNtlPos"`
	TotalRawUSD     string `json:"totalRawUsd"`
}

// OrderStatus conveys order lifecycle information.
type OrderStatus struct {
	Order           OrderInfo `json:"order"`
	Status          string    `json:"status"`
	StatusTimestamp int64     `json:"statusTimestamp"`
}

// OrderInfo stores metadata about an individual order.
type OrderInfo struct {
	Coin      string `json:"coin"`
	Side      string `json:"side"`
	LimitPx   string `json:"limitPx"`
	Sz        string `json:"sz"`
	Oid       int64  `json:"oid"`
	Timestamp int64  `json:"timestamp"`
	OrigSz    string `json:"origSz"`
	Cloid     string `json:"cloid,omitempty"`
}

// Fill describes a match executed against an order.
type Fill struct {
	AvgPx     string `json:"avgPx"`
	TotalSz   string `json:"totalSz"`
	LimitPx   string `json:"limitPx"`
	Sz        string `json:"sz"`
	Oid       int64  `json:"oid"`
	Crossed   bool   `json:"crossed"`
	Fee       string `json:"fee"`
	Tid       int64  `json:"tid"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// OrderResponse captures the standard exchange response after an order submission.
type OrderResponse struct {
	Status   string            `json:"status"` // "ok" or "err".
	Response OrderResponseData `json:"response"`
}

// OrderResponseData wraps the response payload.
type OrderResponseData struct {
	Type string                  `json:"type"` // Typically "order".
	Data OrderResponseDataDetail `json:"data"`
}

// OrderResponseDataDetail contains the per-order statuses.
type OrderResponseDataDetail struct {
	Statuses []OrderStatusResponse `json:"statuses"`
}

// OrderStatusResponse tracks the status of an individual order request.
type OrderStatusResponse struct {
	Resting *RestingOrder `json:"resting,omitempty"`
	Filled  *FilledOrder  `json:"filled,omitempty"`
	Error   string        `json:"error,omitempty"`
}

// RestingOrder represents an order that is currently resting on the book.
type RestingOrder struct {
	Oid int64 `json:"oid"`
}

// FilledOrder represents a fully matched order.
type FilledOrder struct {
	TotalSz string `json:"totalSz"`
	AvgPx   string `json:"avgPx"`
	Oid     int64  `json:"oid"`
}
