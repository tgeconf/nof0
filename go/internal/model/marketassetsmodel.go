package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ MarketAssetsModel = (*customMarketAssetsModel)(nil)

type (
	// MarketAssetsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMarketAssetsModel.
	MarketAssetsModel interface {
		marketAssetsModel
	}

	customMarketAssetsModel struct {
		*defaultMarketAssetsModel
	}
)

// NewMarketAssetsModel returns a model for the database table.
func NewMarketAssetsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) MarketAssetsModel {
	return &customMarketAssetsModel{
		defaultMarketAssetsModel: newMarketAssetsModel(conn, c, opts...),
	}
}
