package repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// AccountSnapshot captures the latest equity snapshot for a model.
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
}

// AccountingRepo exposes read helpers for account/equity related tables.
type AccountingRepo interface {
	// LatestSnapshots returns the freshest equity snapshot per model. When modelIDs
	// is empty it returns all known models.
	LatestSnapshots(ctx context.Context, modelIDs []string) (map[string]AccountSnapshot, error)
}

type accountingRepo struct {
	conn sqlx.SqlConn
}

func newAccountingRepo(deps Dependencies) AccountingRepo {
	return &accountingRepo{
		conn: deps.DBConn,
	}
}

func (r *accountingRepo) LatestSnapshots(ctx context.Context, modelIDs []string) (map[string]AccountSnapshot, error) {
	query := `
SELECT DISTINCT ON (model_id)
    model_id,
    ts_ms,
    dollar_equity,
    realized_pnl,
    total_unrealized_pnl,
    cum_pnl_pct,
    sharpe_ratio,
    since_inception_hourly_marker,
    since_inception_minute_marker
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

	finalQuery := fmt.Sprintf(query, clause)
	var rows []accountSnapshotRow
	if err := r.conn.QueryRowsCtx(ctx, &rows, finalQuery, args...); err != nil {
		return nil, fmt.Errorf("accountingRepo.LatestSnapshots query: %w", err)
	}

	result := make(map[string]AccountSnapshot, len(rows))
	for _, row := range rows {
		snapshot := AccountSnapshot{
			ModelID:            row.ModelID,
			TimestampMs:        row.TsMs,
			DollarEquity:       row.DollarEquity,
			RealizedPnl:        row.RealizedPnl,
			TotalUnrealizedPnl: row.TotalUnrealizedPnl,
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
		result[row.ModelID] = snapshot
	}
	return result, nil
}

type accountSnapshotRow struct {
	ModelID                    string          `db:"model_id"`
	TsMs                       int64           `db:"ts_ms"`
	DollarEquity               float64         `db:"dollar_equity"`
	RealizedPnl                float64         `db:"realized_pnl"`
	TotalUnrealizedPnl         float64         `db:"total_unrealized_pnl"`
	CumPnlPct                  sql.NullFloat64 `db:"cum_pnl_pct"`
	SharpeRatio                sql.NullFloat64 `db:"sharpe_ratio"`
	SinceInceptionHourlyMarker sql.NullInt64   `db:"since_inception_hourly_marker"`
	SinceInceptionMinuteMarker sql.NullInt64   `db:"since_inception_minute_marker"`
}
