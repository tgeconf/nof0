package repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// PositionRecord mirrors the positions table while normalising nullable fields.
type PositionRecord struct {
	ID               string
	ModelID          string
	ExchangeProvider string
	Symbol           string
	Side             string
	Status           string
	EntryOid         *int64
	RiskUsd          *float64
	Confidence       *float64
	IndexCol         []byte
	ExitPlan         []byte
	EntryTimeMs      int64
	EntryPrice       float64
	TpOid            *int64
	Margin           *float64
	WaitForFill      bool
	SlOid            *int64
	CurrentPrice     *float64
	ClosedPnl        *float64
	LiquidationPrice *float64
	Commission       *float64
	Leverage         *float64
	Slippage         *float64
	Quantity         float64
	UnrealizedPnl    *float64
}

// PositionsRepo fetches open/closed positions aggregated by model.
type PositionsRepo interface {
	// ActiveByModels returns all open positions keyed by model ID.
	ActiveByModels(ctx context.Context, modelIDs []string) (map[string][]PositionRecord, error)
}

type positionsRepo struct {
	conn sqlx.SqlConn
}

func newPositionsRepo(deps Dependencies) PositionsRepo {
	return &positionsRepo{
		conn: deps.DBConn,
	}
}

func (r *positionsRepo) ActiveByModels(ctx context.Context, modelIDs []string) (map[string][]PositionRecord, error) {
	query := `
SELECT
    id,
    model_id,
    exchange_provider,
    symbol,
    side,
    status,
    entry_oid,
    risk_usd,
    confidence,
    index_col,
    exit_plan,
    entry_time_ms,
    entry_price,
    tp_oid,
    margin,
    wait_for_fill,
    sl_oid,
    current_price,
    closed_pnl,
    liquidation_price,
    commission,
    leverage,
    slippage,
    quantity,
    unrealized_pnl
FROM public.positions
WHERE status = 'open'
%s
ORDER BY model_id, symbol`

	var (
		args   []any
		clause string
	)
	if len(modelIDs) > 0 {
		clause = "AND model_id = ANY($1)"
		args = append(args, pq.Array(modelIDs))
	}

	finalQuery := fmt.Sprintf(query, clause)
	var rows []positionRow
	if err := r.conn.QueryRowsCtx(ctx, &rows, finalQuery, args...); err != nil {
		return nil, fmt.Errorf("positionsRepo.ActiveByModels query: %w", err)
	}

	result := make(map[string][]PositionRecord)
	for _, row := range rows {
		rec := PositionRecord{
			ID:               row.ID,
			ModelID:          row.ModelID,
			ExchangeProvider: row.ExchangeProvider,
			Symbol:           row.Symbol,
			Side:             row.Side,
			Status:           row.Status,
			EntryTimeMs:      row.EntryTimeMs,
			EntryPrice:       row.EntryPrice,
			WaitForFill:      row.WaitForFill,
			Quantity:         row.Quantity,
		}
		if row.EntryOid.Valid {
			value := row.EntryOid.Int64
			rec.EntryOid = &value
		}
		if row.RiskUsd.Valid {
			value := row.RiskUsd.Float64
			rec.RiskUsd = &value
		}
		if row.Confidence.Valid {
			value := row.Confidence.Float64
			rec.Confidence = &value
		}
		if row.IndexCol.Valid {
			rec.IndexCol = []byte(row.IndexCol.String)
		}
		if row.ExitPlan.Valid {
			rec.ExitPlan = []byte(row.ExitPlan.String)
		}
		if row.TpOid.Valid {
			value := row.TpOid.Int64
			rec.TpOid = &value
		}
		if row.Margin.Valid {
			value := row.Margin.Float64
			rec.Margin = &value
		}
		if row.SlOid.Valid {
			value := row.SlOid.Int64
			rec.SlOid = &value
		}
		if row.CurrentPrice.Valid {
			value := row.CurrentPrice.Float64
			rec.CurrentPrice = &value
		}
		if row.ClosedPnl.Valid {
			value := row.ClosedPnl.Float64
			rec.ClosedPnl = &value
		}
		if row.LiquidationPrice.Valid {
			value := row.LiquidationPrice.Float64
			rec.LiquidationPrice = &value
		}
		if row.Commission.Valid {
			value := row.Commission.Float64
			rec.Commission = &value
		}
		if row.Leverage.Valid {
			value := row.Leverage.Float64
			rec.Leverage = &value
		}
		if row.Slippage.Valid {
			value := row.Slippage.Float64
			rec.Slippage = &value
		}
		if row.UnrealizedPnl.Valid {
			value := row.UnrealizedPnl.Float64
			rec.UnrealizedPnl = &value
		}
		result[row.ModelID] = append(result[row.ModelID], rec)
	}

	return result, nil
}

type positionRow struct {
	ID               string          `db:"id"`
	ModelID          string          `db:"model_id"`
	ExchangeProvider string          `db:"exchange_provider"`
	Symbol           string          `db:"symbol"`
	Side             string          `db:"side"`
	Status           string          `db:"status"`
	EntryOid         sql.NullInt64   `db:"entry_oid"`
	RiskUsd          sql.NullFloat64 `db:"risk_usd"`
	Confidence       sql.NullFloat64 `db:"confidence"`
	IndexCol         sql.NullString  `db:"index_col"`
	ExitPlan         sql.NullString  `db:"exit_plan"`
	EntryTimeMs      int64           `db:"entry_time_ms"`
	EntryPrice       float64         `db:"entry_price"`
	TpOid            sql.NullInt64   `db:"tp_oid"`
	Margin           sql.NullFloat64 `db:"margin"`
	WaitForFill      bool            `db:"wait_for_fill"`
	SlOid            sql.NullInt64   `db:"sl_oid"`
	CurrentPrice     sql.NullFloat64 `db:"current_price"`
	ClosedPnl        sql.NullFloat64 `db:"closed_pnl"`
	LiquidationPrice sql.NullFloat64 `db:"liquidation_price"`
	Commission       sql.NullFloat64 `db:"commission"`
	Leverage         sql.NullFloat64 `db:"leverage"`
	Slippage         sql.NullFloat64 `db:"slippage"`
	Quantity         float64         `db:"quantity"`
	UnrealizedPnl    sql.NullFloat64 `db:"unrealized_pnl"`
}
