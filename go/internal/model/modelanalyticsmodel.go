package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ModelAnalyticsModel = (*customModelAnalyticsModel)(nil)

type (
	// ModelAnalyticsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customModelAnalyticsModel.
	ModelAnalyticsModel interface {
		modelAnalyticsModel
	}

	customModelAnalyticsModel struct {
		*defaultModelAnalyticsModel
	}
)

// NewModelAnalyticsModel returns a model for the database table.
func NewModelAnalyticsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) ModelAnalyticsModel {
	return &customModelAnalyticsModel{
		defaultModelAnalyticsModel: newModelAnalyticsModel(conn, c, opts...),
	}
}
