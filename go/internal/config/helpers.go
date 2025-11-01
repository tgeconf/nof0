package config

import (
	"nof0-api/pkg/exchange"
	"nof0-api/pkg/executor"
	"nof0-api/pkg/llm"
	"nof0-api/pkg/manager"
	"nof0-api/pkg/market"
)

// MustLoadExchange loads etc/exchange.yaml from the project root and panics on error.
// It isolates exchange config to avoid requiring other sections (LLM, Executor, etc.)
// when tests only need the exchange providers.
func MustLoadExchange() *exchange.Config {
	return exchange.MustLoad()
}

// MustBuildExchangeProviders loads exchange config from the default path
// and builds provider instances; returns the map and default provider name.
func MustBuildExchangeProviders() (map[string]exchange.Provider, string) {
	cfg := MustLoadExchange()
	providers, err := cfg.BuildProviders()
	if err != nil {
		panic(err)
	}
	return providers, cfg.Default
}

// MustLoadExecutor loads the default executor configuration and panics on error.
func MustLoadExecutor() *executor.Config {
	return executor.MustLoad()
}

// MustLoadLLM loads etc/llm.yaml from the project root and panics on error.
func MustLoadLLM() *llm.Config {
	return llm.MustLoad()
}

// MustLoadManager loads the default manager configuration and panics on error.
func MustLoadManager() *manager.Config {
	return manager.MustLoad()
}

// MustLoadMarket loads the default market configuration and panics on error.
func MustLoadMarket() *market.Config {
	return market.MustLoad()
}
