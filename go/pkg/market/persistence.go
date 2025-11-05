package market

import "context"

// Persistence hooks allow providers to persist market data to external stores.
type Persistence interface {
	// UpsertAssets persists static asset metadata for the provider.
	UpsertAssets(ctx context.Context, provider string, assets []Asset) error
	// RecordSnapshot persists a single market snapshot (price/latest context).
	RecordSnapshot(ctx context.Context, provider string, snapshot *Snapshot) error
}
