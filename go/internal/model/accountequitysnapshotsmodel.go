package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AccountEquitySnapshotsModel = (*customAccountEquitySnapshotsModel)(nil)

type (
	// AccountEquitySnapshotsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAccountEquitySnapshotsModel.
	AccountEquitySnapshotsModel interface {
		accountEquitySnapshotsModel
	}

	customAccountEquitySnapshotsModel struct {
		*defaultAccountEquitySnapshotsModel
	}
)

// NewAccountEquitySnapshotsModel returns a model for the database table.
func NewAccountEquitySnapshotsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AccountEquitySnapshotsModel {
	return &customAccountEquitySnapshotsModel{
		defaultAccountEquitySnapshotsModel: newAccountEquitySnapshotsModel(conn, c, opts...),
	}
}
