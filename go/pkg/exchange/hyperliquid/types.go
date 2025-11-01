package hyperliquid

import (
	"encoding/json"
	"fmt"

	"nof0-api/pkg/exchange"
)

// ActionType enumerates supported exchange actions.
type ActionType string

const (
	// ActionTypeOrder submits one or more orders.
	ActionTypeOrder ActionType = "order"
	// ActionTypeCancel cancels specific orders by oid.
	ActionTypeCancel ActionType = "cancel"
	// ActionTypeCancelAll cancels all resting orders for an asset.
	ActionTypeCancelAll ActionType = "cancelAll"
	// ActionTypeUpdateLeverage adjusts leverage settings.
	ActionTypeUpdateLeverage ActionType = "updateLeverage"
)

// Action encodes the payload sent to the Hyperliquid exchange endpoint.
type Action struct {
	Type      ActionType        `json:"type" msgpack:"type"`
	Orders    []orderPayload    `json:"orders,omitempty" msgpack:"orders,omitempty"`
	Cancels   []cancelPayload   `json:"cancels,omitempty" msgpack:"cancels,omitempty"`
	CancelAll *CancelAllPayload `json:"cancelAll,omitempty" msgpack:"cancelAll,omitempty"`
	Grouping  string            `json:"grouping,omitempty" msgpack:"grouping,omitempty"`
	Asset     *int              `json:"asset,omitempty" msgpack:"asset,omitempty"`
	IsCross   *bool             `json:"isCross,omitempty" msgpack:"isCross,omitempty"`
	Leverage  int               `json:"leverage,omitempty" msgpack:"leverage,omitempty"`
}

type orderPayload struct {
	Asset      int              `json:"a" msgpack:"a"`
	IsBuy      bool             `json:"b" msgpack:"b"`
	LimitPx    string           `json:"p" msgpack:"p"`
	Sz         string           `json:"s" msgpack:"s"`
	ReduceOnly bool             `json:"r" msgpack:"r"`
	OrderType  orderTypePayload `json:"t" msgpack:"t"`
	Cloid      string           `json:"c,omitempty" msgpack:"c,omitempty"`
	TriggerPx  string           `json:"triggerPx,omitempty" msgpack:"triggerPx,omitempty"`
	TriggerRel string           `json:"triggerRel,omitempty" msgpack:"triggerRel,omitempty"`
}

type orderTypePayload struct {
	Limit   *limitOrderPayload   `json:"limit,omitempty" msgpack:"limit,omitempty"`
	Trigger *triggerOrderPayload `json:"trigger,omitempty" msgpack:"trigger,omitempty"`
}

type limitOrderPayload struct {
	TIF string `json:"tif" msgpack:"tif"`
}

type triggerOrderPayload struct {
	IsMarket   bool   `json:"isMarket" msgpack:"isMarket"`
	TriggerPx  string `json:"triggerPx" msgpack:"triggerPx"`
	Tpsl       string `json:"tpsl,omitempty" msgpack:"tpsl,omitempty"`
	TriggerRel string `json:"triggerRel,omitempty" msgpack:"triggerRel,omitempty"`
}

// Cancel identifies an order to cancel (public API input).
type Cancel struct {
	Asset int
	Oid   int64
}

// cancelPayload identifies an order to cancel.
type cancelPayload struct {
	Asset int   `json:"a" msgpack:"a"`
	Oid   int64 `json:"o" msgpack:"o"`
}

// CancelAllPayload captures cancel-all arguments.
type CancelAllPayload struct {
	Asset int `json:"asset" msgpack:"asset"`
}

// ExchangeRequest is the signed request envelope for exchange actions.
type ExchangeRequest struct {
	Action       Action    `json:"action"`
	Nonce        int64     `json:"nonce"`
	Signature    Signature `json:"signature"`
	VaultAddress string    `json:"vaultAddress,omitempty"`
	ExpiresAfter *int64    `json:"expiresAfter,omitempty"`
}

// Signature represents an ECDSA signature.
type Signature struct {
	R string `json:"r"`
	S string `json:"s"`
	V int    `json:"v"`
}

// InfoRequest targets read-only endpoints that do not require signatures.
type InfoRequest struct {
	Type string `json:"type"`
	User string `json:"user,omitempty"`
	// For vaultDetails endpoint
	VaultAddress string `json:"vaultAddress,omitempty"`
}

// AccountStateResponse wraps account state returned by Hyperliquid.
type AccountStateResponse struct {
	Status string                 `json:"status"`
	Data   *exchange.AccountState `json:"data"`
}

// MetaResponse contains high level asset metadata.
type MetaResponse struct {
	Universe []AssetUniverseEntry `json:"universe"`
}

// MetaAndAssetCtxsResponse includes universe meta plus per-asset context.
type MetaAndAssetCtxsResponse struct {
	Universe  []AssetUniverseEntry `json:"universe"`
	AssetCtxs []AssetCtx           `json:"assetCtxs"`
}

// UnmarshalJSON handles both legacy array and object payloads.
func (m *MetaAndAssetCtxsResponse) UnmarshalJSON(data []byte) error {
	// Primary format: {"universe":[...],"assetCtxs":[...]}
	type alias MetaAndAssetCtxsResponse
	var object alias
	if err := json.Unmarshal(data, &object); err == nil && (len(object.Universe) > 0 || len(object.AssetCtxs) > 0) {
		m.Universe = object.Universe
		m.AssetCtxs = object.AssetCtxs
		return nil
	}

	// Fallback: [ {"universe":[...]}, [{"..."}] ]
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("hyperliquid: metaAndAssetCtxs decode: %w", err)
	}
	if len(raw) == 0 {
		return fmt.Errorf("hyperliquid: metaAndAssetCtxs empty payload")
	}
	var universeHolder struct {
		Universe []AssetUniverseEntry `json:"universe"`
	}
	if err := json.Unmarshal(raw[0], &universeHolder); err != nil {
		return fmt.Errorf("hyperliquid: metaAndAssetCtxs universe: %w", err)
	}
	m.Universe = universeHolder.Universe

	if len(raw) > 1 {
		if err := json.Unmarshal(raw[1], &m.AssetCtxs); err != nil {
			return fmt.Errorf("hyperliquid: metaAndAssetCtxs assetCtxs: %w", err)
		}
	}
	return nil
}

// AssetUniverseEntry describes asset listing info from the meta endpoint.
type AssetUniverseEntry struct {
	Name         string  `json:"name"`
	SzDecimals   int     `json:"szDecimals"`
	MaxLeverage  float64 `json:"maxLeverage"`
	MarginTable  int     `json:"marginTableId"`
	OnlyIsolated bool    `json:"onlyIsolated"`
	IsDelisted   bool    `json:"isDelisted"`
}

// AssetCtx provides contextual info such as funding and mark price.
type AssetCtx struct {
	Funding      string   `json:"funding"`
	OpenInterest string   `json:"openInterest"`
	PrevDayPx    string   `json:"prevDayPx"`
	DayNtlVlm    string   `json:"dayNtlVlm"`
	DayBaseVlm   string   `json:"dayBaseVlm"`
	Premium      string   `json:"premium"`
	OraclePx     string   `json:"oraclePx"`
	MarkPx       string   `json:"markPx"`
	MidPx        string   `json:"midPx"`
	ImpactPxs    []string `json:"impactPxs"`
}

// AssetInfo aggregates convenience metadata for trading use cases.
type AssetInfo struct {
	Name         string
	SzDecimals   int
	MaxLeverage  float64
	MarginTable  int
	OnlyIsolated bool
	IsDelisted   bool
	Index        int
	MarkPx       string
	MidPx        string
	OraclePx     string
	ImpactPxs    []string
}

// NonceResponse returns the exchange nonce used for signing.
type NonceResponse struct {
	Status string       `json:"status"`
	Data   NoncePayload `json:"data"`
}

// NoncePayload contains the numeric nonce.
type NoncePayload struct {
	Nonce int64 `json:"nonce"`
}

// EIP712Domain matches the domain separator definition.
type EIP712Domain struct {
	Name              string
	Version           string
	ChainID           int
	VerifyingContract string
}

// SubAccounts (info endpoint: type=subAccounts) response items.
type SubAccount struct {
	Name               string                `json:"name"`
	SubAccountUser     string                `json:"subAccountUser"`
	Master             string                `json:"master"`
	ClearinghouseState exchange.AccountState `json:"clearinghouseState"`
	SpotState          *SpotState            `json:"spotState,omitempty"`
}

type SpotState struct {
	Balances []SpotBalance `json:"balances"`
}

type SpotBalance struct {
	Coin     string `json:"coin"`
	Token    int    `json:"token"`
	Total    string `json:"total"`
	Hold     string `json:"hold"`
	EntryNtl string `json:"entryNtl"`
}

// VaultDetails (info endpoint: type=vaultDetails) response.
type VaultDetails struct {
	Name                  string          `json:"name"`
	VaultAddress          string          `json:"vaultAddress"`
	Leader                string          `json:"leader"`
	Description           string          `json:"description"`
	APR                   float64         `json:"apr"`
	Followers             []VaultFollower `json:"followers"`
	MaxDistributable      float64         `json:"maxDistributable"`
	MaxWithdrawable       float64         `json:"maxWithdrawable"`
	IsClosed              bool            `json:"isClosed"`
	AllowDeposits         bool            `json:"allowDeposits"`
	AlwaysCloseOnWithdraw bool            `json:"alwaysCloseOnWithdraw"`
}

type VaultFollower struct {
	User          string `json:"user"`
	VaultEquity   string `json:"vaultEquity"`
	PnL           string `json:"pnl"`
	AllTimePnL    string `json:"allTimePnl"`
	DaysFollowing int    `json:"daysFollowing"`
}
