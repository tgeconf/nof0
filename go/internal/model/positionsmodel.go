package model

import (
	"context"
	"fmt"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ PositionsModel = (*customPositionsModel)(nil)

// PositionRecord provides a nullable-safe representation of a row in the
// positions table. Nullable numeric fields become pointers so callers can
// easily detect unset values while working with idiomatic Go types.
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

type (
	// PositionsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPositionsModel.
	PositionsModel interface {
		positionsModel
		ActiveByModels(ctx context.Context, modelIDs []string) (map[string][]PositionRecord, error)
	}

	customPositionsModel struct {
		*defaultPositionsModel
	}
)

// NewPositionsModel returns a model for the database table.
func NewPositionsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) PositionsModel {
	return &customPositionsModel{
		defaultPositionsModel: newPositionsModel(conn, c, opts...),
	}
}

// ActiveByModels returns all open positions grouped by model ID. When modelIDs
// is empty, it returns every open position.
func (m *customPositionsModel) ActiveByModels(ctx context.Context, modelIDs []string) (map[string][]PositionRecord, error) {
	const baseQuery = `
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

	finalQuery := fmt.Sprintf(baseQuery, clause)

	var rows []Positions
	if err := m.QueryRowsNoCacheCtx(ctx, &rows, finalQuery, args...); err != nil {
		return nil, fmt.Errorf("positions.ActiveByModels query: %w", err)
	}

	result := make(map[string][]PositionRecord)
	for i := range rows {
		rec := buildPositionRecord(&rows[i])
		result[rows[i].ModelId] = append(result[rows[i].ModelId], rec)
	}
	return result, nil
}

func buildPositionRecord(row *Positions) PositionRecord {
	rec := PositionRecord{
		ID:               row.Id,
		ModelID:          row.ModelId,
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
	return rec
}
