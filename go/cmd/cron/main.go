package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"nof0-api/internal/cli"
	"nof0-api/internal/config"
	"nof0-api/pkg/exchange"
	"nof0-api/pkg/market"

	// Import for side-effects: registers hyperliquid providers
	_ "nof0-api/pkg/exchange/hyperliquid"
	hyperliquidExchange "nof0-api/pkg/exchange/hyperliquid"
	_ "nof0-api/pkg/market/exchanges/hyperliquid"
)

const (
	marketInterval   = 2 * time.Minute  // Market data monitoring interval
	exchangeInterval = 10 * time.Minute // Exchange API monitoring interval
	apiTimeout       = 5 * time.Second  // Timeout for individual API calls
	shutdownTimeout  = 10 * time.Second // Grace period for shutdown
)

var monitoredSymbols = []string{"BTC", "ETH", "SOL"}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Println("[main] Starting cron monitor...")

	// Load application configuration
	var appCfg *config.Config
	var err error
	configPath := "etc/nof0.yaml"
	appCfg, err = config.Load(configPath)
	if err != nil {
		log.Printf("[main] Warning: Failed to load app config: %v", err)
		log.Printf("[main] Using default configuration")
		appCfg = &config.Config{Env: "test"} // Default fallback
	}

	// Log configuration information
	log.Printf("[main] Configuration loaded:")
	for _, line := range cli.ConfigSummaryLines(appCfg) {
		log.Printf("  - %s", line)
	}

	marketCfg := appCfg.Market.Value
	marketPath := appCfg.Market.File
	if marketCfg == nil {
		marketCfg = config.MustLoadMarket()
		if marketPath == "" {
			marketPath = "etc/market.yaml (default)"
		}
	}

	exchangeCfg := appCfg.Exchange.Value
	exchangePath := appCfg.Exchange.File
	if exchangeCfg == nil {
		exchangeCfg = config.MustLoadExchange()
		if exchangePath == "" {
			exchangePath = "etc/exchange.yaml (default)"
		}
	}

	log.Printf("  - Market Config Path: %s", marketPath)
	log.Printf("  - Exchange Config Path: %s", exchangePath)
	log.Printf("  - Monitored Symbols: %v", monitoredSymbols)
	log.Printf("  - Monitoring Intervals: market=%s, exchange=%s", marketInterval, exchangeInterval)

	// Build market providers
	marketProviders, err := marketCfg.BuildProviders()
	if err != nil {
		log.Fatalf("[main] Failed to build market providers: %v", err)
	}

	// Get default market provider
	marketProvider, ok := marketProviders[marketCfg.Default]
	if !ok {
		log.Fatalf("[main] Default market provider %q not found", marketCfg.Default)
	}

	// Apply test environment defaults: use testnet endpoints for all providers
	if appCfg.IsTestEnv() {
		for _, provider := range exchangeCfg.Providers {
			provider.Testnet = true
		}
	}

	// Build exchange providers
	exchangeProviders, err := exchangeCfg.BuildProviders()
	if err != nil {
		log.Fatalf("[main] Failed to build exchange providers: %v", err)
	}

	// Get default exchange provider
	exchangeProvider, ok := exchangeProviders[exchangeCfg.Default]
	if !ok {
		log.Fatalf("[main] Default exchange provider %q not found", exchangeCfg.Default)
	}

	// Create context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create wait group for goroutines
	var wg sync.WaitGroup

	// Start market monitoring task
	wg.Add(1)
	go func() {
		defer wg.Done()
		runMarketMonitor(ctx, marketProvider)
	}()

	// Start exchange monitoring task
	wg.Add(1)
	go func() {
		defer wg.Done()
		runExchangeMonitor(ctx, exchangeProvider)
	}()

	log.Println("[main] Cron monitor started. Press Ctrl+C to stop.")

	// Wait for signal
	<-ctx.Done()
	log.Println("[main] Shutdown signal received, stopping tasks...")

	// Give tasks time to complete current work
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("[main] All tasks stopped cleanly")
	case <-shutdownCtx.Done():
		log.Println("[main] Shutdown timeout exceeded, forcing exit")
	}

	log.Println("[main] Cron monitor stopped")
}

// runMarketMonitor runs market data monitoring on a schedule
func runMarketMonitor(ctx context.Context, provider market.Provider) {
	ticker := time.NewTicker(marketInterval)
	defer ticker.Stop()

	// Run once immediately on startup
	monitorMarket(ctx, provider)

	for {
		select {
		case <-ctx.Done():
			log.Println("[market] Stopping market monitor")
			return
		case <-ticker.C:
			monitorMarket(ctx, provider)
		}
	}
}

// runExchangeMonitor runs exchange API monitoring on a schedule
func runExchangeMonitor(ctx context.Context, provider exchange.Provider) {
	ticker := time.NewTicker(exchangeInterval)
	defer ticker.Stop()

	// Run once immediately on startup
	monitorExchange(ctx, provider)

	for {
		select {
		case <-ctx.Done():
			log.Println("[exchange] Stopping exchange monitor")
			return
		case <-ticker.C:
			monitorExchange(ctx, provider)
		}
	}
}

// monitorMarket calls market data interfaces and logs results
func monitorMarket(parentCtx context.Context, provider market.Provider) {
	// Check if parent context is already cancelled
	if parentCtx.Err() != nil {
		return
	}

	// List assets
	func() {
		ctx, cancel := context.WithTimeout(parentCtx, apiTimeout)
		defer cancel()

		start := time.Now()
		assets, err := provider.ListAssets(ctx)
		elapsed := time.Since(start)

		if err != nil {
			log.Printf("[market.list_assets] [ERROR] %v, took %dms", err, elapsed.Milliseconds())
			return
		}

		log.Printf("[market.list_assets] [OK] found %d assets, took %dms", len(assets), elapsed.Milliseconds())
	}()

	// Get snapshots for monitored symbols
	for _, symbol := range monitoredSymbols {
		func(sym string) {
			ctx, cancel := context.WithTimeout(parentCtx, apiTimeout)
			defer cancel()

			start := time.Now()
			snapshot, err := provider.Snapshot(ctx, sym)
			elapsed := time.Since(start)

			if err != nil {
				log.Printf("[market.snapshot.%s] [ERROR] %v, took %dms", sym, err, elapsed.Milliseconds())
				return
			}

			// Validate data
			if snapshot.Price.Last <= 0 {
				log.Printf("[market.snapshot.%s] [WARN] invalid price=%f, took %dms", sym, snapshot.Price.Last, elapsed.Milliseconds())
				return
			}

			log.Printf("[market.snapshot.%s] [OK] price=%.2f, change_1h=%.2f%%, change_4h=%.2f%%, took %dms",
				sym,
				snapshot.Price.Last,
				snapshot.Change.OneHour*100,
				snapshot.Change.FourHour*100,
				elapsed.Milliseconds())

			// Log indicators
			if len(snapshot.Indicators.EMA) > 0 {
				log.Printf("  - Indicators.EMA: %v", snapshot.Indicators.EMA)
			}
			if snapshot.Indicators.MACD != 0 {
				log.Printf("  - Indicators.MACD: %.4f", snapshot.Indicators.MACD)
			}
			if len(snapshot.Indicators.RSI) > 0 {
				log.Printf("  - Indicators.RSI: %v", snapshot.Indicators.RSI)
			}

			// Log open interest
			if snapshot.OpenInterest != nil {
				log.Printf("  - OpenInterest: latest=%.2f, average=%.2f",
					snapshot.OpenInterest.Latest, snapshot.OpenInterest.Average)
			}

			// Log funding rate
			if snapshot.Funding != nil {
				log.Printf("  - Funding: rate=%.4f%%", snapshot.Funding.Rate*100)
			}

			// Log time series statistics
			if snapshot.Intraday != nil && len(snapshot.Intraday.Prices) > 0 {
				log.Printf("  - Intraday: %d data points", len(snapshot.Intraday.Prices))
			}
			if snapshot.LongTerm != nil && len(snapshot.LongTerm.Prices) > 0 {
				log.Printf("  - LongTerm: %d data points", len(snapshot.LongTerm.Prices))
			}
		}(symbol)
	}
}

// monitorExchange calls exchange read-only interfaces and logs results
func monitorExchange(parentCtx context.Context, provider exchange.Provider) {
	// Check if parent context is already cancelled
	if parentCtx.Err() != nil {
		return
	}

	// Get open orders
	func() {
		ctx, cancel := context.WithTimeout(parentCtx, apiTimeout)
		defer cancel()

		start := time.Now()
		orders, err := provider.GetOpenOrders(ctx)
		elapsed := time.Since(start)

		if err != nil {
			log.Printf("[exchange.open_orders] [ERROR] %v, took %dms", err, elapsed.Milliseconds())
			return
		}

		log.Printf("[exchange.open_orders] [OK] %d orders, took %dms", len(orders), elapsed.Milliseconds())
	}()

	// Get positions
	func() {
		ctx, cancel := context.WithTimeout(parentCtx, apiTimeout)
		defer cancel()

		start := time.Now()
		positions, err := provider.GetPositions(ctx)
		elapsed := time.Since(start)

		if err != nil {
			log.Printf("[exchange.positions] [ERROR] %v, took %dms", err, elapsed.Milliseconds())
			return
		}

		log.Printf("[exchange.positions] [OK] %d positions, took %dms", len(positions), elapsed.Milliseconds())

		// Log position details
		for _, pos := range positions {
			entryPx := "N/A"
			if pos.EntryPx != nil {
				entryPx = *pos.EntryPx
			}
			log.Printf("  - %s: size=%s, entry=%s, unrealized_pnl=%s",
				pos.Coin, pos.Szi, entryPx, pos.UnrealizedPnl)
		}
	}()

	// Get account state
	func() {
		ctx, cancel := context.WithTimeout(parentCtx, apiTimeout)
		defer cancel()

		start := time.Now()
		state, err := provider.GetAccountState(ctx)
		elapsed := time.Since(start)

		if err != nil {
			log.Printf("[exchange.account_state] [ERROR] %v, took %dms", err, elapsed.Milliseconds())
			return
		}

		if state == nil {
			log.Printf("[exchange.account_state] [WARN] received nil state, took %dms", elapsed.Milliseconds())
			return
		}

		log.Printf("[exchange.account_state] [OK] account_value=%s, total_margin=%s, took %dms",
			state.MarginSummary.AccountValue, state.MarginSummary.TotalMarginUsed, elapsed.Milliseconds())
	}()

	// Get account value
	func() {
		ctx, cancel := context.WithTimeout(parentCtx, apiTimeout)
		defer cancel()

		start := time.Now()
		value, err := provider.GetAccountValue(ctx)
		elapsed := time.Since(start)

		if err != nil {
			log.Printf("[exchange.account_value] [ERROR] %v, took %dms", err, elapsed.Milliseconds())
			return
		}

		log.Printf("[exchange.account_value] [OK] value=%.2f, took %dms", value, elapsed.Milliseconds())
	}()

	// Test GetAssetIndex for monitored symbols
	for _, symbol := range monitoredSymbols {
		func(sym string) {
			ctx, cancel := context.WithTimeout(parentCtx, apiTimeout)
			defer cancel()

			start := time.Now()
			idx, err := provider.GetAssetIndex(ctx, sym)
			elapsed := time.Since(start)

			if err != nil {
				log.Printf("[exchange.asset_index.%s] [ERROR] %v, took %dms", sym, err, elapsed.Milliseconds())
				return
			}

			log.Printf("[exchange.asset_index.%s] [OK] index=%d, took %dms", sym, idx, elapsed.Milliseconds())
		}(symbol)
	}

	// Test Hyperliquid-specific read-only methods
	if hlProvider, ok := provider.(*hyperliquidExchange.Provider); ok {
		// Test FormatSize for monitored symbols
		for _, symbol := range monitoredSymbols {
			func(sym string) {
				ctx, cancel := context.WithTimeout(parentCtx, apiTimeout)
				defer cancel()

				start := time.Now()
				formatted, err := hlProvider.FormatSize(ctx, sym, 1.23456789)
				elapsed := time.Since(start)

				if err != nil {
					log.Printf("[hyperliquid.format_size.%s] [ERROR] %v, took %dms", sym, err, elapsed.Milliseconds())
					return
				}

				log.Printf("[hyperliquid.format_size.%s] [OK] 1.23456789 -> %s, took %dms", sym, formatted, elapsed.Milliseconds())
			}(symbol)
		}

		// Test FormatPrice for monitored symbols
		for _, symbol := range monitoredSymbols {
			func(sym string) {
				ctx, cancel := context.WithTimeout(parentCtx, apiTimeout)
				defer cancel()

				start := time.Now()
				formatted, err := hlProvider.FormatPrice(ctx, sym, 12345.6789)
				elapsed := time.Since(start)

				if err != nil {
					log.Printf("[hyperliquid.format_price.%s] [ERROR] %v, took %dms", sym, err, elapsed.Milliseconds())
					return
				}

				log.Printf("[hyperliquid.format_price.%s] [OK] 12345.6789 -> %s, took %dms", sym, formatted, elapsed.Milliseconds())
			}(symbol)
		}
	}
}
