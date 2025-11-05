package exchange

import "context"

// Provider exposes trading capabilities in an exchange-agnostic fashion.
type Provider interface {
	// Order management.
	PlaceOrder(ctx context.Context, order Order) (*OrderResponse, error)
	CancelOrder(ctx context.Context, asset int, oid int64) error
	GetOpenOrders(ctx context.Context) ([]OrderStatus, error)

	// Position management.
	GetPositions(ctx context.Context) ([]Position, error)
	ClosePosition(ctx context.Context, coin string) (*OrderResponse, error)
	UpdateLeverage(ctx context.Context, asset int, isCross bool, leverage int) error

	// Account information.
	GetAccountState(ctx context.Context) (*AccountState, error)
	GetAccountValue(ctx context.Context) (float64, error)

	// Utilities.
	GetAssetIndex(ctx context.Context, coin string) (int, error)
}
