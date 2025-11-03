package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ PriceLatestModel = (*customPriceLatestModel)(nil)

type (
	// PriceLatestModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPriceLatestModel.
	PriceLatestModel interface {
		priceLatestModel
	}

	customPriceLatestModel struct {
		*defaultPriceLatestModel
	}
)

// NewPriceLatestModel returns a model for the database table.
func NewPriceLatestModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) PriceLatestModel {
	return &customPriceLatestModel{
		defaultPriceLatestModel: newPriceLatestModel(conn, c, opts...),
	}
}
