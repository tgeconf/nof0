package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TradesModel = (*customTradesModel)(nil)

type (
	// TradesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTradesModel.
	TradesModel interface {
		tradesModel
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
