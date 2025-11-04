package model

import (
	"context"
	"fmt"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AccountEquitySnapshotsModel = (*customAccountEquitySnapshotsModel)(nil)

// AccountSnapshot captures the latest equity snapshot for a model with nullable
// metrics represented via pointers.
type AccountSnapshot struct {
	ModelID                    string
	TimestampMs                int64
	DollarEquity               float64
	RealizedPnl                float64
	TotalUnrealizedPnl         float64
	CumPnlPct                  *float64
	SharpeRatio                *float64
	SinceInceptionHourlyMarker *int64
	SinceInceptionMinuteMarker *int64
	Metadata                   string
}

type (
	// AccountEquitySnapshotsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAccountEquitySnapshotsModel.
	AccountEquitySnapshotsModel interface {
		accountEquitySnapshotsModel
		LatestSnapshots(ctx context.Context, modelIDs []string) (map[string]AccountSnapshot, error)
	}

	customAccountEquitySnapshotsModel struct {
		*defaultAccountEquitySnapshotsModel
	}
)

// NewAccountEquitySnapshotsModel returns a model for the database table.
func NewAccountEquitySnapshotsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AccountEquitySnapshotsModel {
	return &customAccountEquitySnapshotsModel{
		defaultAccountEquitySnapshotsModel: newAccountEquitySnapshotsModel(conn, c, opts...),
	}
}

// LatestSnapshots loads the newest equity snapshot per model. When modelIDs is
// empty it returns all known models.
func (m *customAccountEquitySnapshotsModel) LatestSnapshots(ctx context.Context, modelIDs []string) (map[string]AccountSnapshot, error) {
	const baseQuery = `
SELECT DISTINCT ON (model_id)
    model_id,
    ts_ms,
    dollar_equity,
    realized_pnl,
    total_unrealized_pnl,
    cum_pnl_pct,
    sharpe_ratio,
    since_inception_hourly_marker,
    since_inception_minute_marker,
    metadata
FROM public.account_equity_snapshots
%s
ORDER BY model_id, ts_ms DESC`

	var (
		args   []any
		clause string
	)

	if len(modelIDs) > 0 {
		clause = "WHERE model_id = ANY($1)"
		args = append(args, pq.Array(modelIDs))
	}

	finalQuery := fmt.Sprintf(baseQuery, clause)
	var rows []AccountEquitySnapshots
	if err := m.QueryRowsNoCacheCtx(ctx, &rows, finalQuery, args...); err != nil {
		return nil, fmt.Errorf("accountEquitySnapshots.LatestSnapshots query: %w", err)
	}

	result := make(map[string]AccountSnapshot, len(rows))
	for i := range rows {
		result[rows[i].ModelId] = buildAccountSnapshot(&rows[i])
	}
	return result, nil
}

func buildAccountSnapshot(row *AccountEquitySnapshots) AccountSnapshot {
	snapshot := AccountSnapshot{
		ModelID:            row.ModelId,
		TimestampMs:        row.TsMs,
		DollarEquity:       row.DollarEquity,
		RealizedPnl:        row.RealizedPnl,
		TotalUnrealizedPnl: row.TotalUnrealizedPnl,
		Metadata:           row.Metadata,
	}
	if row.CumPnlPct.Valid {
		value := row.CumPnlPct.Float64
		snapshot.CumPnlPct = &value
	}
	if row.SharpeRatio.Valid {
		value := row.SharpeRatio.Float64
		snapshot.SharpeRatio = &value
	}
	if row.SinceInceptionHourlyMarker.Valid {
		value := row.SinceInceptionHourlyMarker.Int64
		snapshot.SinceInceptionHourlyMarker = &value
	}
	if row.SinceInceptionMinuteMarker.Valid {
		value := row.SinceInceptionMinuteMarker.Int64
		snapshot.SinceInceptionMinuteMarker = &value
	}
	return snapshot
}
