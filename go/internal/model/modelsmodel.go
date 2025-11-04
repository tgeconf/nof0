package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ModelsModel = (*customModelsModel)(nil)

type (
	// ModelsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customModelsModel.
	ModelsModel interface {
		modelsModel
	}

	customModelsModel struct {
		*defaultModelsModel
	}
)

// NewModelsModel returns a model for the database table.
func NewModelsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) ModelsModel {
	return &customModelsModel{
		defaultModelsModel: newModelsModel(conn, c, opts...),
	}
}
