package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/zeromicro/go-zero/core/logx"

	"nof0-api/pkg/confkit"
	exchangepkg "nof0-api/pkg/exchange"
	_ "nof0-api/pkg/exchange/hyperliquid"
	llmpkg "nof0-api/pkg/llm"
	managerpkg "nof0-api/pkg/manager"
	marketpkg "nof0-api/pkg/market"
	_ "nof0-api/pkg/market/exchanges/hyperliquid"
)

type filteredMarket struct {
	marketpkg.Provider
	allowed map[string]struct{}
}

func newFilteredMarket(base marketpkg.Provider, symbols []string) (*filteredMarket, error) {
	if base == nil {
		return nil, fmt.Errorf("filtered market: base provider is nil")
	}
	set := make(map[string]struct{}, len(symbols))
	for _, sym := range symbols {
		sym = strings.TrimSpace(sym)
		if sym == "" {
			continue
		}
		set[strings.ToUpper(sym)] = struct{}{}
	}
	if len(set) == 0 {
		return nil, fmt.Errorf("filtered market: allowed symbol list is empty")
	}
	return &filteredMarket{
		Provider: base,
		allowed:  set,
	}, nil
}

func (f *filteredMarket) Snapshot(ctx context.Context, symbol string) (*marketpkg.Snapshot, error) {
	if !f.isAllowed(symbol) {
		return nil, fmt.Errorf("filtered market: symbol %s not allowed", symbol)
	}
	return f.Provider.Snapshot(ctx, symbol)
}

func (f *filteredMarket) ListAssets(ctx context.Context) ([]marketpkg.Asset, error) {
	assets, err := f.Provider.ListAssets(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]marketpkg.Asset, 0, len(f.allowed))
	for _, asset := range assets {
		if f.isAllowed(asset.Symbol) {
			filtered = append(filtered, asset)
		}
	}
	return filtered, nil
}

func (f *filteredMarket) isAllowed(symbol string) bool {
	if symbol == "" {
		return false
	}
	_, ok := f.allowed[strings.ToUpper(symbol)]
	return ok
}

func parseSymbols(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		field = strings.ToUpper(field)
		if _, exists := seen[field]; exists {
			continue
		}
		seen[field] = struct{}{}
		out = append(out, field)
	}
	return out
}

func fatalf(format string, args ...interface{}) {
	logx.Errorf(format, args...)
	os.Exit(1)
}

func adaptManagerConfig(cfg *managerpkg.Config, totalEquity float64, allowed []string) error {
	if cfg == nil {
		return fmt.Errorf("manager config is nil")
	}
	if totalEquity <= 0 {
		return fmt.Errorf("total equity must be positive, got %.2f", totalEquity)
	}
	if len(cfg.Traders) == 0 {
		return fmt.Errorf("manager config has no traders defined")
	}
	if len(allowed) == 0 {
		return fmt.Errorf("allowed symbol list cannot be empty")
	}

	cfg.Manager.TotalEquityUSD = totalEquity
	cfg.Manager.ReserveEquityPct = 0

	perTraderEquity := totalEquity / float64(len(cfg.Traders))
	allocationRemaining := 100.0
	for i := range cfg.Traders {
		tr := &cfg.Traders[i]
		// Force allocation to split evenly across all configured traders; last trader absorbs rounding.
		if i == len(cfg.Traders)-1 {
			tr.AllocationPct = allocationRemaining
		} else {
			share := 100.0 / float64(len(cfg.Traders))
			tr.AllocationPct = share
			allocationRemaining -= share
		}

		// Cap max position size to per-trader equity.
		maxSize := perTraderEquity
		if maxSize <= 0 {
			maxSize = totalEquity
		}
		if tr.RiskParams.MaxPositionSizeUSD > maxSize {
			tr.RiskParams.MaxPositionSizeUSD = maxSize
		}
		if tr.RiskParams.MaxPositions > len(allowed) {
			tr.RiskParams.MaxPositions = len(allowed)
			if tr.RiskParams.MaxPositions == 0 {
				tr.RiskParams.MaxPositions = 1
			}
		}
		if tr.ExecGuards.CandidateLimit <= 0 || tr.ExecGuards.CandidateLimit > len(allowed) {
			tr.ExecGuards.CandidateLimit = len(allowed)
		}
	}
	return cfg.Validate()
}

func main() {
	var (
		exchangePath = flag.String("exchange-config", "etc/exchange.yaml", "path to exchange provider configuration")
		marketPath   = flag.String("market-config", "etc/market.yaml", "path to market provider configuration")
		llmPath      = flag.String("llm-config", "etc/llm.yaml", "path to llm client configuration")
		managerPath  = flag.String("manager-config", "etc/manager.yaml", "path to manager configuration")
		allowedRaw   = flag.String("symbols", "BTC,ETH", "comma-separated list of tradable symbols")
		totalEquity  = flag.Float64("equity", 100.0, "total deployable equity in USD")
	)
	flag.Parse()
	logx.MustSetup(logx.LogConf{})
	logx.DisableStat()

	allowedSymbols := parseSymbols(*allowedRaw)
	if len(allowedSymbols) == 0 {
		fatalf("no tradable symbols provided; use --symbols to specify at least one")
	}

	confkit.LoadDotenvOnce()

	exchangeCfg, err := exchangepkg.LoadConfig(*exchangePath)
	if err != nil {
		fatalf("load exchange config: %v", err)
	}
	exchangeProviders, err := exchangeCfg.BuildProviders()
	if err != nil {
		fatalf("build exchange providers: %v", err)
	}

	marketCfg, err := marketpkg.LoadConfig(*marketPath)
	if err != nil {
		fatalf("load market config: %v", err)
	}
	marketProviders, err := marketCfg.BuildProviders()
	if err != nil {
		fatalf("build market providers: %v", err)
	}
	filteredMarkets := make(map[string]marketpkg.Provider, len(marketProviders))
	for name, provider := range marketProviders {
		wrapped, wrapErr := newFilteredMarket(provider, allowedSymbols)
		if wrapErr != nil {
			fatalf("wrap market provider %s: %v", name, wrapErr)
		}
		filteredMarkets[name] = wrapped
	}

	llmCfg, err := llmpkg.LoadConfig(*llmPath)
	if err != nil {
		fatalf("load llm config: %v", err)
	}
	llmClient, err := llmpkg.NewClient(llmCfg)
	if err != nil {
		fatalf("initialise llm client: %v", err)
	}
	defer func() {
		_ = llmClient.Close()
	}()

	managerCfg, err := managerpkg.LoadConfig(*managerPath)
	if err != nil {
		fatalf("load manager config: %v", err)
	}
	if err := adaptManagerConfig(managerCfg, *totalEquity, allowedSymbols); err != nil {
		fatalf("adapt manager config: %v", err)
	}

	execFactory := managerpkg.NewBasicExecutorFactory(llmClient)
	mgr := managerpkg.NewManager(managerCfg, execFactory, exchangeProviders, filteredMarkets)

	for _, traderCfg := range managerCfg.Traders {
		vt, regErr := mgr.RegisterTrader(traderCfg)
		if regErr != nil {
			fatalf("register trader %s: %v", traderCfg.ID, regErr)
		}
		logx.Infof("registered trader %s (%s) using exchange=%s market=%s", vt.ID, vt.Name, traderCfg.ExchangeProvider, traderCfg.MarketProvider)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logx.Infof("received signal %s, shutting down manager loop", sig)
		cancel()
		mgr.Stop()
	}()

	logx.Infof("starting manager loop with equity=%.2f USD, symbols=%s", *totalEquity, strings.Join(allowedSymbols, ","))
	if err := mgr.RunTradingLoop(ctx); err != nil && err != context.Canceled {
		fatalf("manager loop exited with error: %v", err)
	}
	logx.Info("manager loop stopped")
}
