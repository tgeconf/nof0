package manager

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"nof0-api/pkg/exchange"
	executorpkg "nof0-api/pkg/executor"
	"nof0-api/pkg/llm"
	"nof0-api/pkg/market"
)

// ExecutorFactory abstracts executor construction so Manager stays decoupled
// from concrete executor wiring (local vs RPC, prompt template selection, etc.).
type ExecutorFactory interface {
	NewExecutor(traderCfg TraderConfig) (executorpkg.Executor, error)
}

// BasicExecutorFactory is a minimal factory that adapts a TraderConfig into
// an executor.Config and constructs a local executor.BasicExecutor.
type BasicExecutorFactory struct {
	llmClient llm.LLMClient
}

// NewBasicExecutorFactory returns a factory that builds local executors using
// the provided LLM client.
func NewBasicExecutorFactory(client llm.LLMClient) *BasicExecutorFactory {
	return &BasicExecutorFactory{llmClient: client}
}

// NewExecutor implements ExecutorFactory.
func (f *BasicExecutorFactory) NewExecutor(traderCfg TraderConfig) (executorpkg.Executor, error) {
	if f == nil || f.llmClient == nil {
		return nil, errors.New("manager: executor factory requires llm client")
	}

	// Adapt risk/interval fields from TraderConfig → executor.Config.
	// Ensure durations are populated to avoid zero timeout.
	interval := traderCfg.DecisionInterval
	if interval <= 0 {
		if d, err := time.ParseDuration(traderCfg.DecisionIntervalRaw); err == nil && d > 0 {
			interval = d
		} else {
			interval = 3 * time.Minute
		}
	}
	intervalRaw := traderCfg.DecisionIntervalRaw
	if intervalRaw == "" {
		intervalRaw = "3m"
	}
	ec := &executorpkg.Config{
		BTCETHLeverage:         traderCfg.RiskParams.BTCETHLeverage,
		AltcoinLeverage:        traderCfg.RiskParams.AltcoinLeverage,
		MinConfidence:          traderCfg.RiskParams.MinConfidence,
		MinRiskReward:          traderCfg.RiskParams.MinRiskRewardRatio,
		MaxPositions:           traderCfg.RiskParams.MaxPositions,
		DecisionIntervalRaw:    intervalRaw,
		DecisionInterval:       interval,
		DecisionTimeoutRaw:     "60s",
		DecisionTimeout:        60 * time.Second,
		MaxConcurrentDecisions: 1,
		AllowedTraderIDs:       []string{traderCfg.ID},
	}
	// executor.NewExecutor validates config.
	exec, err := executorpkg.NewExecutor(ec, f.llmClient, traderCfg.PromptTemplate)
	if err != nil {
		return nil, err
	}
	return exec, nil
}

// Manager is the orchestration layer that coordinates virtual traders,
// executors and providers.
type Manager struct {
	mu sync.RWMutex

	config *Config

	traders map[string]*VirtualTrader // Trader ID → instance

	// Provider registries resolved at startup (see internal/svc for wiring).
	exchangeProviders map[string]exchange.Provider
	marketProviders   map[string]market.Provider

	executorFactory ExecutorFactory

	stopChan chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// NewManager constructs a Manager with injected dependencies.
func NewManager(
	cfg *Config,
	execFactory ExecutorFactory,
	exch map[string]exchange.Provider,
	mkts map[string]market.Provider,
) *Manager {
	if cfg == nil {
		cfg = &Config{}
	}
	m := &Manager{
		config:            cfg,
		traders:           make(map[string]*VirtualTrader),
		exchangeProviders: make(map[string]exchange.Provider),
		marketProviders:   make(map[string]market.Provider),
		executorFactory:   execFactory,
		stopChan:          make(chan struct{}),
	}
	for k, v := range exch {
		m.exchangeProviders[k] = v
	}
	for k, v := range mkts {
		m.marketProviders[k] = v
	}
	return m
}

// InitializeManager loads configuration and returns a Manager instance.
// Note: provider registries and executor factory can be injected later
// via NewManager or dedicated setters if needed by the application wiring.
func InitializeManager(configPath string) (*Manager, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	return NewManager(cfg, nil, nil, nil), nil
}

// RegisterTrader creates a VirtualTrader from the provided configuration and
// attaches providers and executor instances.
func (m *Manager) RegisterTrader(cfg TraderConfig) (*VirtualTrader, error) {
	if m == nil {
		return nil, errors.New("manager: nil manager")
	}
	if err := (&Config{Traders: []TraderConfig{cfg}}).Validate(); err != nil {
		// Reuse config validation on a temporary wrapper to validate the trader.
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.traders[cfg.ID]; exists {
		return nil, fmt.Errorf("manager: trader %s already registered", cfg.ID)
	}

	// Resolve providers by ID as declared in manager config.
	ex, ok := m.exchangeProviders[cfg.ExchangeProvider]
	if !ok {
		return nil, fmt.Errorf("manager: unknown exchange provider %q for trader %s", cfg.ExchangeProvider, cfg.ID)
	}
	mk, ok := m.marketProviders[cfg.MarketProvider]
	if !ok {
		return nil, fmt.Errorf("manager: unknown market provider %q for trader %s", cfg.MarketProvider, cfg.ID)
	}
	if m.executorFactory == nil {
		return nil, errors.New("manager: executorFactory is not set")
	}
	exec, err := m.executorFactory.NewExecutor(cfg)
	if err != nil {
		return nil, fmt.Errorf("manager: create executor for trader %s: %w", cfg.ID, err)
	}

	vt := &VirtualTrader{
		ID:               cfg.ID,
		Name:             cfg.Name,
		Exchange:         cfg.ExchangeProvider,
		ExchangeProvider: ex,
		MarketProvider:   mk,
		Executor:         exec,
		PromptTemplate:   cfg.PromptTemplate,
		RiskParams:       cfg.RiskParams,
		ResourceAlloc: ResourceAllocation{
			AllocationPct: cfg.AllocationPct,
		},
		State:            TraderStateStopped,
		DecisionInterval: cfg.DecisionInterval,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	m.traders[cfg.ID] = vt
	if cfg.AutoStart {
		_ = vt.Start()
	}
	return vt, nil
}

// UnregisterTrader stops and removes a trader from registry.
func (m *Manager) UnregisterTrader(traderID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.traders[traderID]
	if !ok {
		return fmt.Errorf("manager: trader %s not found", traderID)
	}
	_ = t.Stop() // Best-effort stop; ignore error for MVP.
	delete(m.traders, traderID)
	return nil
}

// GetActiveTraders returns a stable-ordered slice of currently active traders.
func (m *Manager) GetActiveTraders() []*VirtualTrader {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*VirtualTrader, 0, len(m.traders))
	for _, t := range m.traders {
		if t.IsActive() {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// RunTradingLoop executes the main orchestration loop (minimal skeleton).
func (m *Manager) RunTradingLoop(ctx context.Context) error {
	if m == nil {
		return errors.New("manager: nil manager")
	}
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.stopChan:
			return nil
		case <-ticker.C:
			traders := m.GetActiveTraders()
			for _, t := range traders {
				if !t.ShouldMakeDecision() {
					continue
				}
				// Build context (can be enriched later with synced state).
				out, err := t.Executor.GetFullDecision(&executorpkg.Context{
					CurrentTime:     time.Now().UTC().Format(time.RFC3339),
					RuntimeMinutes:  0,
					CallCount:       0,
					Performance:     nil,
					BTCETHLeverage:  t.RiskParams.BTCETHLeverage,
					AltcoinLeverage: t.RiskParams.AltcoinLeverage,
				})
				if err == nil && out != nil {
					for i := range out.Decisions {
						d := out.Decisions[i]
						_ = m.ExecuteDecision(t, &d)
					}
				}
				t.RecordDecision(time.Now())
				_ = m.SyncTraderPositions(t.ID)
			}
		}
	}
}

// Stop signals the main loop to exit.
func (m *Manager) Stop() { m.stopOnce.Do(func() { close(m.stopChan) }) }

// ExecuteDecision executes a single decision using trader's exchange provider.
// Placeholder MVP: perform basic validation and return nil.
func (m *Manager) ExecuteDecision(trader *VirtualTrader, decision *executorpkg.Decision) error {
	if trader == nil || decision == nil {
		return errors.New("manager: execute decision requires trader and decision")
	}
	if decision.Symbol == "" {
		return errors.New("manager: decision missing symbol")
	}
	if decision.PositionSizeUSD < 0 {
		return errors.New("manager: decision position size must be non-negative")
	}

	// Close actions shortcut via provider.
	if decision.Action == "close_long" || decision.Action == "close_short" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return trader.ExchangeProvider.ClosePosition(ctx, decision.Symbol)
	}

	if decision.Action != "open_long" && decision.Action != "open_short" {
		// Ignore non-trade actions (e.g., hold/wait).
		return nil
	}

	// Enforce per-trader caps.
	if trader.RiskParams.MaxPositionSizeUSD > 0 && decision.PositionSizeUSD > trader.RiskParams.MaxPositionSizeUSD+1e-6 {
		return fmt.Errorf("manager: decision size %.2f exceeds max_position_size_usd %.2f", decision.PositionSizeUSD, trader.RiskParams.MaxPositionSizeUSD)
	}

	// Resolve leverage preference.
	lev := decision.Leverage
	if lev <= 0 {
		if isBTCorETH(decision.Symbol) {
			lev = trader.RiskParams.BTCETHLeverage
		} else {
			lev = trader.RiskParams.AltcoinLeverage
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assetIdx, err := trader.ExchangeProvider.GetAssetIndex(ctx, decision.Symbol)
	if err == nil && lev > 0 {
		_ = trader.ExchangeProvider.UpdateLeverage(ctx, assetIdx, true, lev)
	}

	// Determine price: use decision price or query market snapshot.
	price := decision.EntryPrice
	if !(price > 0) {
		snap, err := trader.MarketProvider.Snapshot(ctx, decision.Symbol)
		if err != nil {
			return fmt.Errorf("manager: fetch market snapshot for %s: %w", decision.Symbol, err)
		}
		price = snap.Price.Last
	}
	if !(price > 0) {
		return fmt.Errorf("manager: invalid price resolved for %s", decision.Symbol)
	}

	// Compute size and direction.
	qty := decision.PositionSizeUSD / price
	if qty <= 0 || math.IsNaN(qty) || math.IsInf(qty, 0) {
		return fmt.Errorf("manager: invalid position size for %s: qty=%.6f", decision.Symbol, qty)
	}
	isBuy := decision.Action == "open_long"

	// Submit a limit IOC order approximating a marketable order.
	order := exchange.Order{
		Asset:      assetIdx,
		IsBuy:      isBuy,
		LimitPx:    fmt.Sprintf("%.8f", price),
		Sz:         fmt.Sprintf("%.8f", qty),
		ReduceOnly: false,
		OrderType:  exchange.OrderType{Limit: &exchange.LimitOrderType{TIF: "Ioc"}},
	}
	if _, err := trader.ExchangeProvider.PlaceOrder(ctx, order); err != nil {
		return fmt.Errorf("manager: place order %s %s: %w", decision.Symbol, decision.Action, err)
	}
	return nil
}

// SyncAllPositions updates cached account/position state for all traders (stub).
func (m *Manager) SyncAllPositions() error {
	m.mu.RLock()
	ids := make([]string, 0, len(m.traders))
	for id := range m.traders {
		ids = append(ids, id)
	}
	m.mu.RUnlock()
	for _, id := range ids {
		_ = m.SyncTraderPositions(id)
	}
	return nil
}

// SyncTraderPositions updates a single trader's cached state (stub).
func (m *Manager) SyncTraderPositions(traderID string) error {
	m.mu.RLock()
	t := m.traders[traderID]
	m.mu.RUnlock()
	if t == nil {
		return fmt.Errorf("manager: sync positions: trader %s not found", traderID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	acct, err := t.ExchangeProvider.GetAccountState(ctx)
	if err != nil {
		return err
	}
	// Parse commonly used fields from strings.
	acctVal := parseFloat(acct.MarginSummary.AccountValue)
	marginUsed := parseFloat(acct.MarginSummary.TotalMarginUsed)
	var unreal float64
	for i := range acct.AssetPositions {
		unreal += parseFloat(acct.AssetPositions[i].UnrealizedPnl)
	}

	t.mu.Lock()
	t.ResourceAlloc.CurrentEquityUSD = acctVal
	t.ResourceAlloc.MarginUsedUSD = marginUsed
	t.ResourceAlloc.UnrealizedPnLUSD = unreal
	t.ResourceAlloc.AvailableBalanceUSD = math.Max(0, acctVal-marginUsed)
	t.UpdatedAt = time.Now()
	t.mu.Unlock()
	return nil
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	return 0
}

func isBTCorETH(symbol string) bool {
	switch symbol {
	case "BTC", "ETH", "BTCUSDT", "ETHUSDT":
		return true
	default:
		return false
	}
}
