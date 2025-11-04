package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ DecisionCyclesModel = (*customDecisionCyclesModel)(nil)

type (
	// DecisionCyclesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customDecisionCyclesModel.
	DecisionCyclesModel interface {
		decisionCyclesModel
	}

	customDecisionCyclesModel struct {
		*defaultDecisionCyclesModel
	}
)

// NewDecisionCyclesModel returns a model for the database table.
func NewDecisionCyclesModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) DecisionCyclesModel {
	return &customDecisionCyclesModel{
		defaultDecisionCyclesModel: newDecisionCyclesModel(conn, c, opts...),
	}
}
