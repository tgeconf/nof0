package config

import (
	"fmt"
	"path/filepath"

	"nof0-api/pkg/exchange"
)

// MustLoadExchange loads etc/exchange.yaml from the project root and panics on error.
// It isolates exchange config to avoid requiring other sections (LLM, Executor, etc.)
// when tests only need the exchange providers.
func MustLoadExchange() *exchange.Config {
	root := MustProjectRoot()
	path := filepath.Join(root, "etc", "exchange.yaml")
	cfg, err := exchange.LoadConfig(path)
	if err != nil {
		panic(fmt.Errorf("load exchange config %s: %w", path, err))
	}
	return cfg
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
