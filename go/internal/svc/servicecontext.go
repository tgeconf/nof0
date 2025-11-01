package svc

import (
	"log"

	_ "github.com/jackc/pgx/v5/stdlib" // register pgx driver
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"nof0-api/internal/config"
	"nof0-api/internal/data"
	"nof0-api/internal/model"
	executorpkg "nof0-api/pkg/executor"
	managerpkg "nof0-api/pkg/manager"
)

type ServiceContext struct {
	Config config.Config

	DataLoader *data.DataLoader

	ExecutorConfig         *executorpkg.Config
	ExecutorPrompt         *executorpkg.PromptRenderer
	ExecutorPromptDigest   string
	ManagerConfig          *managerpkg.Config
	ManagerPromptRenderers map[string]*managerpkg.PromptRenderer
	ManagerPromptDigests   map[string]string

	// Optional DB models (injected but unused by handlers/logic for now)
	DBConn                      sqlx.SqlConn
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
}

func NewServiceContext(c config.Config) *ServiceContext {
	svc := &ServiceContext{
		Config:     c,
		DataLoader: data.NewDataLoader(c.DataPath),
	}

	if cfg := c.Executor.Config; cfg != nil {
		renderer, err := executorpkg.NewPromptRenderer(cfg)
		if err != nil {
			log.Fatalf("failed to init executor prompt renderer: %v", err)
		}
		svc.ExecutorConfig = cfg
		svc.ExecutorPrompt = renderer
		svc.ExecutorPromptDigest = renderer.Digest()
	}

	if cfg := c.Manager.Config; cfg != nil {
		renderers := make(map[string]*managerpkg.PromptRenderer, len(cfg.Traders))
		digests := make(map[string]string, len(cfg.Traders))
		for i := range cfg.Traders {
			trader := &cfg.Traders[i]
			renderer, err := managerpkg.NewPromptRenderer(trader.PromptTemplate)
			if err != nil {
				log.Fatalf("failed to init manager prompt renderer for trader %s: %v", trader.ID, err)
			}
			renderers[trader.ID] = renderer
			digests[trader.ID] = renderer.Digest()
		}
		svc.ManagerConfig = cfg
		svc.ManagerPromptRenderers = renderers
		svc.ManagerPromptDigests = digests
	}

	// Only inject DB models when DSN provided; business logic still uses DataLoader.
	if c.Postgres.DSN != "" {
		conn := sqlx.NewSqlConn("pgx", c.Postgres.DSN)
		svc.DBConn = conn
		svc.ModelsModel = model.NewModelsModel(conn)
		svc.SymbolsModel = model.NewSymbolsModel(conn)
		svc.PriceTicksModel = model.NewPriceTicksModel(conn)
		svc.PriceLatestModel = model.NewPriceLatestModel(conn)
		svc.AccountsModel = model.NewAccountsModel(conn)
		svc.AccountEquitySnapshotsModel = model.NewAccountEquitySnapshotsModel(conn)
		svc.PositionsModel = model.NewPositionsModel(conn)
		svc.TradesModel = model.NewTradesModel(conn)
		svc.ModelAnalyticsModel = model.NewModelAnalyticsModel(conn)
		svc.ConversationsModel = model.NewConversationsModel(conn)
		svc.ConversationMessagesModel = model.NewConversationMessagesModel(conn)
	}
	return svc
}
