package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TraderStateModel = (*customTraderStateModel)(nil)

type (
	// TraderStateModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTraderStateModel.
	TraderStateModel interface {
		traderStateModel
	}

	customTraderStateModel struct {
		*defaultTraderStateModel
	}
)

// NewTraderStateModel returns a model for the database table.
func NewTraderStateModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) TraderStateModel {
	return &customTraderStateModel{
		defaultTraderStateModel: newTraderStateModel(conn, c, opts...),
	}
}
