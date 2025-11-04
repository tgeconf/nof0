package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TradesModel = (*customTradesModel)(nil)

// TradeRecord provides a nullable-safe representation of a trade row.
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

type (
	// TradesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTradesModel.
	TradesModel interface {
		tradesModel
		RecentByModel(ctx context.Context, modelID string, limit int) ([]TradeRecord, error)
	}

	customTradesModel struct {
		*defaultTradesModel
	}
)

// NewTradesModel returns a model for the database table.
func NewTradesModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) TradesModel {
	return &customTradesModel{
		defaultTradesModel: newTradesModel(conn, c, opts...),
	}
}

// RecentByModel returns trades for the given model ordered by entry timestamp
// descending. Limit defaults to 200 when non-positive.
func (m *customTradesModel) RecentByModel(ctx context.Context, modelID string, limit int) ([]TradeRecord, error) {
	if limit <= 0 {
		limit = 200
	}

	const query = `
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

	var rows []Trades
	if err := m.QueryRowsNoCacheCtx(ctx, &rows, query, modelID, limit); err != nil {
		return nil, fmt.Errorf("trades.RecentByModel query: %w", err)
	}

	result := make([]TradeRecord, 0, len(rows))
	for i := range rows {
		result = append(result, buildTradeRecord(&rows[i]))
	}
	return result, nil
}

func buildTradeRecord(row *Trades) TradeRecord {
	rec := TradeRecord{
		ID:               row.Id,
		ModelID:          row.ModelId,
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
	if row.TradeId.Valid {
		value := row.TradeId.String
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
	return rec
}
