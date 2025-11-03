package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ MarketAssetCtxModel = (*customMarketAssetCtxModel)(nil)

type (
	// MarketAssetCtxModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMarketAssetCtxModel.
	MarketAssetCtxModel interface {
		marketAssetCtxModel
	}

	customMarketAssetCtxModel struct {
		*defaultMarketAssetCtxModel
	}
)

// NewMarketAssetCtxModel returns a model for the database table.
func NewMarketAssetCtxModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) MarketAssetCtxModel {
	return &customMarketAssetCtxModel{
		defaultMarketAssetCtxModel: newMarketAssetCtxModel(conn, c, opts...),
	}
}
