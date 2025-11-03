package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ConversationsModel = (*customConversationsModel)(nil)

type (
	// ConversationsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customConversationsModel.
	ConversationsModel interface {
		conversationsModel
	}

	customConversationsModel struct {
		*defaultConversationsModel
	}
)

// NewConversationsModel returns a model for the database table.
func NewConversationsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) ConversationsModel {
	return &customConversationsModel{
		defaultConversationsModel: newConversationsModel(conn, c, opts...),
	}
}
