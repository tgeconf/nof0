package svc

import (
	"log"

	_ "github.com/jackc/pgx/v5/stdlib" // register pgx driver
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"nof0-api/internal/config"
	"nof0-api/internal/data"
	"nof0-api/internal/model"
	"nof0-api/pkg/confkit"
	exchangepkg "nof0-api/pkg/exchange"
	_ "nof0-api/pkg/exchange/hyperliquid"
	executorpkg "nof0-api/pkg/executor"
	llmpkg "nof0-api/pkg/llm"
	managerpkg "nof0-api/pkg/manager"
	marketpkg "nof0-api/pkg/market"
	_ "nof0-api/pkg/market/exchanges/hyperliquid"
)

type ServiceContext struct {
	Config config.Config

	DataLoader *data.DataLoader

	LLMConfig              *llmpkg.Config
	ExecutorConfig         *executorpkg.Config
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

func NewServiceContext(c config.Config, mainConfigPath string) *ServiceContext {
	svc := &ServiceContext{
		Config:     c,
		DataLoader: data.NewDataLoader(c.DataPath),
	}

	baseDir := confkit.BaseDir(mainConfigPath)

	// Load LLM config if specified
	if c.LLM.File != "" {
		llmCfg, err := llmpkg.LoadConfig(confkit.ResolvePath(baseDir, c.LLM.File))
		if err != nil {
			log.Fatalf("failed to load llm config: %v", err)
		}
		// Apply test environment defaults: use low-cost model for good quality
		if c.IsTestEnv() {
			llmCfg.DefaultModel = "google/gemini-2.5-flash-lite"
		}
		svc.LLMConfig = llmCfg
	}

	// Load Executor config if specified
	if c.Executor.File != "" {
		executorCfg, err := executorpkg.LoadConfig(confkit.ResolvePath(baseDir, c.Executor.File))
		if err != nil {
			log.Fatalf("failed to load executor config: %v", err)
		}
		svc.ExecutorConfig = executorCfg
	}

	// Load Manager config if specified
	if c.Manager.File != "" {
		managerCfg, err := managerpkg.LoadConfig(confkit.ResolvePath(baseDir, c.Manager.File))
		if err != nil {
			log.Fatalf("failed to load manager config: %v", err)
		}
		// Build prompt renderers for each trader
		renderers := make(map[string]*managerpkg.PromptRenderer, len(managerCfg.Traders))
		digests := make(map[string]string, len(managerCfg.Traders))
		for i := range managerCfg.Traders {
			trader := &managerCfg.Traders[i]
			renderer, err := managerpkg.NewPromptRenderer(trader.PromptTemplate)
			if err != nil {
				log.Fatalf("failed to init manager prompt renderer for trader %s: %v", trader.ID, err)
			}
			renderers[trader.ID] = renderer
			digests[trader.ID] = renderer.Digest()
		}
		svc.ManagerConfig = managerCfg
		svc.ManagerPromptRenderers = renderers
		svc.ManagerPromptDigests = digests
	}

	// Load Exchange config if specified
	if c.Exchange.File != "" {
		exchangeCfg, err := exchangepkg.LoadConfig(confkit.ResolvePath(baseDir, c.Exchange.File))
		if err != nil {
			log.Fatalf("failed to load exchange config: %v", err)
		}
		// Apply test environment defaults: use testnet endpoints for all providers
		if c.IsTestEnv() {
			for _, provider := range exchangeCfg.Providers {
				provider.Testnet = true
			}
		}
		providers, err := exchangeCfg.BuildProviders()
		if err != nil {
			log.Fatalf("failed to build exchange providers: %v", err)
		}
		svc.ExchangeConfig = exchangeCfg
		svc.ExchangeProviders = providers
		if exchangeCfg.Default != "" {
			svc.DefaultExchange = providers[exchangeCfg.Default]
		}
	}

	// Load Market config if specified
	if c.Market.File != "" {
		marketCfg, err := marketpkg.LoadConfig(confkit.ResolvePath(baseDir, c.Market.File))
		if err != nil {
			log.Fatalf("failed to load market config: %v", err)
		}
		providers, err := marketCfg.BuildProviders()
		if err != nil {
			log.Fatalf("failed to build market providers: %v", err)
		}
		svc.MarketConfig = marketCfg
		svc.MarketProviders = providers
		if marketCfg.Default != "" {
			svc.DefaultMarket = providers[marketCfg.Default]
		}
	}

	// Validate cross-module references: manager trader -> exchange/market providers
	if svc.ManagerConfig != nil {
		svc.ManagerTraderExchange = make(map[string]exchangepkg.Provider, len(svc.ManagerConfig.Traders))
		svc.ManagerTraderMarket = make(map[string]marketpkg.Provider, len(svc.ManagerConfig.Traders))
		for i := range svc.ManagerConfig.Traders {
			trader := &svc.ManagerConfig.Traders[i]
			// Strict mapping: manager config requires explicit provider IDs
			exProvider, ok := svc.ExchangeProviders[trader.ExchangeProvider]
			if !ok {
				log.Fatalf("manager trader %s references unknown exchange provider %s", trader.ID, trader.ExchangeProvider)
			}
			svc.ManagerTraderExchange[trader.ID] = exProvider

			mktProvider, ok := svc.MarketProviders[trader.MarketProvider]
			if !ok {
				log.Fatalf("manager trader %s references unknown market provider %s", trader.ID, trader.MarketProvider)
			}
			svc.ManagerTraderMarket[trader.ID] = mktProvider
		}
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
