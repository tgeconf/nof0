package repo

import (
	"errors"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"nof0-api/internal/model"
	cacheutil "nof0-api/internal/repo/cache"
)

// Dependencies bundles the generated goctl models and shared infrastructure
// required by repository implementations.
type Dependencies struct {
	DBConn     sqlx.SqlConn
	CachedConn *sqlc.CachedConn
	Cache      cache.Cache
	TTL        cacheutil.TTLSet

	ModelsModel                 model.ModelsModel
	SymbolsModel                model.SymbolsModel
	PriceTicksModel             model.PriceTicksModel
	PriceLatestModel            model.PriceLatestModel
	AccountsModel               model.AccountsModel
	AccountEquitySnapshotsModel model.AccountEquitySnapshotsModel
	PositionsModel              model.PositionsModel
	TradesModel                 model.TradesModel
	ModelAnalyticsModel         model.ModelAnalyticsModel
	ConversationsModel          model.ConversationsModel
	ConversationMessagesModel   model.ConversationMessagesModel
	DecisionCyclesModel         model.DecisionCyclesModel
	MarketAssetsModel           model.MarketAssetsModel
	MarketAssetCtxModel         model.MarketAssetCtxModel
	TraderStateModel            model.TraderStateModel
}

// Set exposes strongly typed repositories to application logic.
type Set struct {
	Accounting AccountingRepo
	Positions  PositionsRepo
	Trades     TradesRepo
}

// New constructs the repository set, validating required dependencies.
func New(deps Dependencies) (*Set, error) {
	if deps.DBConn == nil {
		return nil, errors.New("repo: missing DBConn dependency")
	}

	accounting := newAccountingRepo(deps)
	positions := newPositionsRepo(deps)
	trades := newTradesRepo(deps)

	return &Set{
		Accounting: accounting,
		Positions:  positions,
		Trades:     trades,
	}, nil
}
