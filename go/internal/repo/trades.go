package repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// TradeRecord provides a normalised view of the trades table.
type TradeRecord struct {
	ID                     string
	ModelID                string
	ExchangeProvider       string
	Symbol                 string
	Side                   string
	TradeType              *string
	TradeID                *string
	Quantity               *float64
	Leverage               *float64
	Confidence             *float64
	EntryPrice             *float64
	EntryTsMs              int64
	EntryHumanTime         *string
	EntrySz                *float64
	EntryTid               *int64
	EntryOid               *int64
	EntryCrossed           bool
	EntryLiquidation       []byte
	EntryCommissionDollars *float64
	EntryClosedPnl         *float64
	ExitPrice              *float64
	ExitTsMs               *int64
	ExitHumanTime          *string
	ExitSz                 *float64
	ExitTid                *int64
	ExitOid                *int64
	ExitCrossed            *bool
	ExitLiquidation        []byte
	ExitCommissionDollars  *float64
	ExitClosedPnl          *float64
	ExitPlan               []byte
	RealizedGrossPnl       *float64
	RealizedNetPnl         *float64
	TotalCommissionDollars *float64
}

// TradesRepo exposes read helpers for trade history queries.
type TradesRepo interface {
	// RecentByModel returns trades ordered by entry timestamp descending.
	RecentByModel(ctx context.Context, modelID string, limit int) ([]TradeRecord, error)
}

type tradesRepo struct {
	conn sqlx.SqlConn
}

func newTradesRepo(deps Dependencies) TradesRepo {
	return &tradesRepo{
		conn: deps.DBConn,
	}
}

func (r *tradesRepo) RecentByModel(ctx context.Context, modelID string, limit int) ([]TradeRecord, error) {
	if limit <= 0 {
		limit = 200
	}

	query := `
SELECT
    id,
    model_id,
    exchange_provider,
    symbol,
    side,
    trade_type,
    trade_id,
    quantity,
    leverage,
    confidence,
    entry_price,
    entry_ts_ms,
    entry_human_time,
    entry_sz,
    entry_tid,
    entry_oid,
    entry_crossed,
    entry_liquidation,
    entry_commission_dollars,
    entry_closed_pnl,
    exit_price,
    exit_ts_ms,
    exit_human_time,
    exit_sz,
    exit_tid,
    exit_oid,
    exit_crossed,
    exit_liquidation,
    exit_commission_dollars,
    exit_closed_pnl,
    exit_plan,
    realized_gross_pnl,
    realized_net_pnl,
    total_commission_dollars
FROM public.trades
WHERE model_id = $1
ORDER BY entry_ts_ms DESC
LIMIT $2`

	var rows []tradeRow
	if err := r.conn.QueryRowsCtx(ctx, &rows, query, modelID, limit); err != nil {
		return nil, fmt.Errorf("tradesRepo.RecentByModel query: %w", err)
	}

	result := make([]TradeRecord, 0, len(rows))
	for _, row := range rows {
		rec := TradeRecord{
			ID:               row.ID,
			ModelID:          row.ModelID,
			ExchangeProvider: row.ExchangeProvider,
			Symbol:           row.Symbol,
			Side:             row.Side,
			EntryTsMs:        row.EntryTsMs,
			EntryCrossed:     row.EntryCrossed,
		}
		if row.TradeType.Valid {
			value := row.TradeType.String
			rec.TradeType = &value
		}
		if row.TradeID.Valid {
			value := row.TradeID.String
			rec.TradeID = &value
		}
		if row.Quantity.Valid {
			value := row.Quantity.Float64
			rec.Quantity = &value
		}
		if row.Leverage.Valid {
			value := row.Leverage.Float64
			rec.Leverage = &value
		}
		if row.Confidence.Valid {
			value := row.Confidence.Float64
			rec.Confidence = &value
		}
		if row.EntryPrice.Valid {
			value := row.EntryPrice.Float64
			rec.EntryPrice = &value
		}
		if row.EntryHumanTime.Valid {
			value := row.EntryHumanTime.String
			rec.EntryHumanTime = &value
		}
		if row.EntrySz.Valid {
			value := row.EntrySz.Float64
			rec.EntrySz = &value
		}
		if row.EntryTid.Valid {
			value := row.EntryTid.Int64
			rec.EntryTid = &value
		}
		if row.EntryOid.Valid {
			value := row.EntryOid.Int64
			rec.EntryOid = &value
		}
		if row.EntryLiquidation.Valid {
			rec.EntryLiquidation = []byte(row.EntryLiquidation.String)
		}
		if row.EntryCommissionDollars.Valid {
			value := row.EntryCommissionDollars.Float64
			rec.EntryCommissionDollars = &value
		}
		if row.EntryClosedPnl.Valid {
			value := row.EntryClosedPnl.Float64
			rec.EntryClosedPnl = &value
		}
		if row.ExitPrice.Valid {
			value := row.ExitPrice.Float64
			rec.ExitPrice = &value
		}
		if row.ExitTsMs.Valid {
			value := row.ExitTsMs.Int64
			rec.ExitTsMs = &value
		}
		if row.ExitHumanTime.Valid {
			value := row.ExitHumanTime.String
			rec.ExitHumanTime = &value
		}
		if row.ExitSz.Valid {
			value := row.ExitSz.Float64
			rec.ExitSz = &value
		}
		if row.ExitTid.Valid {
			value := row.ExitTid.Int64
			rec.ExitTid = &value
		}
		if row.ExitOid.Valid {
			value := row.ExitOid.Int64
			rec.ExitOid = &value
		}
		if row.ExitCrossed.Valid {
			value := row.ExitCrossed.Bool
			rec.ExitCrossed = &value
		}
		if row.ExitLiquidation.Valid {
			rec.ExitLiquidation = []byte(row.ExitLiquidation.String)
		}
		if row.ExitCommissionDollars.Valid {
			value := row.ExitCommissionDollars.Float64
			rec.ExitCommissionDollars = &value
		}
		if row.ExitClosedPnl.Valid {
			value := row.ExitClosedPnl.Float64
			rec.ExitClosedPnl = &value
		}
		if row.ExitPlan.Valid {
			rec.ExitPlan = []byte(row.ExitPlan.String)
		}
		if row.RealizedGrossPnl.Valid {
			value := row.RealizedGrossPnl.Float64
			rec.RealizedGrossPnl = &value
		}
		if row.RealizedNetPnl.Valid {
			value := row.RealizedNetPnl.Float64
			rec.RealizedNetPnl = &value
		}
		if row.TotalCommissionDollars.Valid {
			value := row.TotalCommissionDollars.Float64
			rec.TotalCommissionDollars = &value
		}
		result = append(result, rec)
	}

	return result, nil
}

type tradeRow struct {
	ID                     string          `db:"id"`
	ModelID                string          `db:"model_id"`
	ExchangeProvider       string          `db:"exchange_provider"`
	Symbol                 string          `db:"symbol"`
	Side                   string          `db:"side"`
	TradeType              sql.NullString  `db:"trade_type"`
	TradeID                sql.NullString  `db:"trade_id"`
	Quantity               sql.NullFloat64 `db:"quantity"`
	Leverage               sql.NullFloat64 `db:"leverage"`
	Confidence             sql.NullFloat64 `db:"confidence"`
	EntryPrice             sql.NullFloat64 `db:"entry_price"`
	EntryTsMs              int64           `db:"entry_ts_ms"`
	EntryHumanTime         sql.NullString  `db:"entry_human_time"`
	EntrySz                sql.NullFloat64 `db:"entry_sz"`
	EntryTid               sql.NullInt64   `db:"entry_tid"`
	EntryOid               sql.NullInt64   `db:"entry_oid"`
	EntryCrossed           bool            `db:"entry_crossed"`
	EntryLiquidation       sql.NullString  `db:"entry_liquidation"`
	EntryCommissionDollars sql.NullFloat64 `db:"entry_commission_dollars"`
	EntryClosedPnl         sql.NullFloat64 `db:"entry_closed_pnl"`
	ExitPrice              sql.NullFloat64 `db:"exit_price"`
	ExitTsMs               sql.NullInt64   `db:"exit_ts_ms"`
	ExitHumanTime          sql.NullString  `db:"exit_human_time"`
	ExitSz                 sql.NullFloat64 `db:"exit_sz"`
	ExitTid                sql.NullInt64   `db:"exit_tid"`
	ExitOid                sql.NullInt64   `db:"exit_oid"`
	ExitCrossed            sql.NullBool    `db:"exit_crossed"`
	ExitLiquidation        sql.NullString  `db:"exit_liquidation"`
	ExitCommissionDollars  sql.NullFloat64 `db:"exit_commission_dollars"`
	ExitClosedPnl          sql.NullFloat64 `db:"exit_closed_pnl"`
	ExitPlan               sql.NullString  `db:"exit_plan"`
	RealizedGrossPnl       sql.NullFloat64 `db:"realized_gross_pnl"`
	RealizedNetPnl         sql.NullFloat64 `db:"realized_net_pnl"`
	TotalCommissionDollars sql.NullFloat64 `db:"total_commission_dollars"`
}
