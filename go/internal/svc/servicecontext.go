package svc

import (
	"log"

	_ "github.com/jackc/pgx/v5/stdlib" // register pgx driver
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"nof0-api/internal/config"
	"nof0-api/internal/data"
	"nof0-api/internal/model"
	exchangepkg "nof0-api/pkg/exchange"
	executorpkg "nof0-api/pkg/executor"
	managerpkg "nof0-api/pkg/manager"
	marketpkg "nof0-api/pkg/market"
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
	ExchangeConfig         *exchangepkg.Config
	ExchangeProviders      map[string]exchangepkg.Provider
	DefaultExchange        exchangepkg.Provider
	MarketConfig           *marketpkg.Config
	MarketProviders        map[string]marketpkg.Provider
	DefaultMarket          marketpkg.Provider
	ManagerTraderExchange  map[string]exchangepkg.Provider
	ManagerTraderMarket    map[string]marketpkg.Provider

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

	if cfg := c.Exchange.Config; cfg != nil {
		providers, err := cfg.BuildProviders()
		if err != nil {
			log.Fatalf("failed to init exchange providers: %v", err)
		}
		svc.ExchangeConfig = cfg
		svc.ExchangeProviders = providers
		if cfg.Default != "" {
			svc.DefaultExchange = providers[cfg.Default]
		}
	}

	if cfg := c.Market.Config; cfg != nil {
		providers, err := cfg.BuildProviders()
		if err != nil {
			log.Fatalf("failed to init market providers: %v", err)
		}
		svc.MarketConfig = cfg
		svc.MarketProviders = providers
		if cfg.Default != "" {
			svc.DefaultMarket = providers[cfg.Default]
		}
	}

	if svc.ManagerConfig != nil {
		svc.ManagerTraderExchange = make(map[string]exchangepkg.Provider, len(svc.ManagerConfig.Traders))
		svc.ManagerTraderMarket = make(map[string]marketpkg.Provider, len(svc.ManagerConfig.Traders))
		for i := range svc.ManagerConfig.Traders {
			trader := &svc.ManagerConfig.Traders[i]
			if trader.ExchangeProvider != "" {
				provider, ok := svc.ExchangeProviders[trader.ExchangeProvider]
				if !ok {
					log.Fatalf("manager trader %s references unknown exchange provider %s", trader.ID, trader.ExchangeProvider)
				}
				svc.ManagerTraderExchange[trader.ID] = provider
			} else if svc.DefaultExchange != nil {
				svc.ManagerTraderExchange[trader.ID] = svc.DefaultExchange
			} else {
				log.Fatalf("manager trader %s has no exchange provider and no default configured", trader.ID)
			}

			marketProviderID := trader.MarketProvider
			if marketProviderID == "" && svc.MarketConfig != nil {
				marketProviderID = svc.MarketConfig.Default
			}
			if marketProviderID != "" {
				provider, ok := svc.MarketProviders[marketProviderID]
				if !ok {
					log.Fatalf("manager trader %s references unknown market provider %s", trader.ID, marketProviderID)
				}
				svc.ManagerTraderMarket[trader.ID] = provider
			} else if svc.DefaultMarket != nil {
				svc.ManagerTraderMarket[trader.ID] = svc.DefaultMarket
			}
		}
	}

	if cfg := c.Exchange.Config; cfg != nil {
		providers, err := cfg.BuildProviders()
		if err != nil {
			log.Fatalf("failed to init exchange providers: %v", err)
		}
		svc.ExchangeConfig = cfg
		svc.ExchangeProviders = providers
	}

	if cfg := c.Market.Config; cfg != nil {
		providers, err := cfg.BuildProviders()
		if err != nil {
			log.Fatalf("failed to init market providers: %v", err)
		}
		svc.MarketConfig = cfg
		svc.MarketProviders = providers
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
