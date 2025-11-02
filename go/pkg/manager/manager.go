package manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"nof0-api/pkg/exchange"
	executorpkg "nof0-api/pkg/executor"
	"nof0-api/pkg/journal"
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
		MajorCoinLeverage:      traderCfg.RiskParams.MajorCoinLeverage,
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
	exec, err := executorpkg.NewExecutor(ec, f.llmClient, traderCfg.ExecutorTemplate, traderCfg.Model)
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
	tempCfg := &Config{
		Manager:    ManagerConfig{},
		Traders:    []TraderConfig{cfg},
		Monitoring: MonitoringConfig{},
	}
	if m.config != nil {
		tempCfg.Manager = m.config.Manager
		tempCfg.Monitoring = m.config.Monitoring
	}
	if err := tempCfg.Validate(); err != nil {
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
		ExecGuards:       cfg.ExecGuards,
		ResourceAlloc: ResourceAllocation{
			AllocationPct: cfg.AllocationPct,
		},
		State:            TraderStateStopped,
		DecisionInterval: cfg.DecisionInterval,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Cooldown:         make(map[string]time.Time),
		JournalEnabled:   cfg.JournalEnabled,
	}
	if cfg.JournalEnabled {
		dir := cfg.JournalDir
		if strings.TrimSpace(dir) == "" {
			dir = fmt.Sprintf("journal/%s", cfg.ID)
		}
		vt.Journal = journal.NewWriter(dir)
		logx.Infof("manager: trader %s journaling enabled dir=%s", cfg.ID, dir)
	}

	m.traders[cfg.ID] = vt
	if cfg.AutoStart {
		_ = vt.Start()
	}
	logx.Infof("manager: registered trader id=%s name=%s allocation=%.2f%% exchange=%s market=%s model=%s auto_start=%t", vt.ID, vt.Name, cfg.AllocationPct, cfg.ExchangeProvider, cfg.MarketProvider, cfg.Model, cfg.AutoStart)
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
	logx.Infof("manager: unregistered trader id=%s", traderID)
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
	logx.WithContext(ctx).Infof("manager: trading loop starting tick=1s active_traders=%d", len(m.GetActiveTraders()))
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logx.WithContext(ctx).Infof("manager: trading loop stopping (context): %v", ctx.Err())
			return ctx.Err()
		case <-m.stopChan:
			logx.WithContext(ctx).Infof("manager: trading loop stopping (stop signal)")
			return nil
		case <-ticker.C:
			traders := m.GetActiveTraders()
			for _, t := range traders {
				if !t.ShouldMakeDecision() {
					continue
				}
				cycleStart := time.Now()
				// Sharpe gating
				if t.ExecGuards.SharpePauseThreshold != 0 && t.ExecGuards.PauseDurationOnBreach > 0 && t.Performance != nil {
					if t.Performance.SharpeRatio < t.ExecGuards.SharpePauseThreshold {
						t.mu.Lock()
						if t.PauseUntil.Before(time.Now()) {
							t.PauseUntil = time.Now().Add(t.ExecGuards.PauseDurationOnBreach)
						}
						t.mu.Unlock()
						logx.WithContext(ctx).Infof("manager: trader %s paused for Sharpe gating until %s", t.ID, t.PauseUntil.Format(time.RFC3339))
						continue
					}
				}
				// Build richer executor context and refresh performance view.
				perfView := t.Performance.ToExecutorView()
				t.Executor.UpdatePerformance(perfView)

				ectx := m.buildExecutorContext(t)
				out, decisionErr := t.Executor.GetFullDecision(&ectx)

				// Prepare journaling containers
				var decisionsJSON string
				var actions []map[string]any
				allOK := true
				decisionCount := 0
				if out != nil {
					decisionCount = len(out.Decisions)
					if b, e := json.Marshal(out.Decisions); e == nil {
						decisionsJSON = string(b)
					}
					// Close actions first, then open actions; cap new opens by remaining slots.
					decisions := sortDecisionsCloseFirst(out.Decisions)
					// remaining slots by max positions
					remaining := t.RiskParams.MaxPositions - len(ectx.Positions)
					if remaining < 0 {
						remaining = 0
					}
					// also enforce per-cycle cap if configured (>0)
					cycleCap := t.ExecGuards.MaxNewPositionsPerCycle
					if cycleCap > 0 && cycleCap < remaining {
						remaining = cycleCap
					}
					decisions = capNewOpenDecisions(decisions, remaining)
					for i := range decisions {
						d := decisions[i]
						execErr := m.ExecuteDecision(t, &d)
						act := map[string]any{
							"symbol":            d.Symbol,
							"action":            d.Action,
							"leverage":          d.Leverage,
							"position_size_usd": d.PositionSizeUSD,
							"entry_price":       d.EntryPrice,
							"stop_loss":         d.StopLoss,
							"take_profit":       d.TakeProfit,
							"confidence":        d.Confidence,
							"result":            "ok",
						}
						if execErr != nil {
							act["result"] = "error"
							act["error"] = execErr.Error()
							allOK = false
							logx.WithContext(ctx).Errorf("manager: trader %s decision action=%s symbol=%s error=%v", t.ID, d.Action, d.Symbol, execErr)
						}
						actions = append(actions, act)
					}
				} else {
					allOK = false
					if decisionErr != nil {
						logx.WithContext(ctx).Errorf("manager: trader %s decision generation failed: %v", t.ID, decisionErr)
					}
				}

				// Update lightweight performance snapshot (success ratio proxy)
				if t.Performance == nil {
					t.Performance = &PerformanceMetrics{}
				}
				succ := 0
				for _, a := range actions {
					if a["result"] == "ok" {
						succ++
					}
				}
				total := len(actions)
				t.Performance.TotalTrades += total
				if total > 0 {
					t.Performance.WinRate = float64(succ) / float64(total)
				}
				t.Performance.UpdatedAt = time.Now()

				// Journal the cycle if configured
				if t.Journal != nil && t.JournalEnabled {
					if jErr := m.writeJournalRecord(t, &ectx, out, decisionsJSON, actions, decisionErr, allOK); jErr != nil {
						logx.WithContext(ctx).Errorf("manager: trader %s journal write failed: %v", t.ID, jErr)
					} else {
						logx.WithContext(ctx).Infof("manager: trader %s journal written prompt_digest=%s", t.ID, outPromptDigest(out))
					}
				}
				t.RecordDecision(time.Now())
				if syncErr := m.SyncTraderPositions(t.ID); syncErr != nil {
					logx.WithContext(ctx).Errorf("manager: trader %s sync positions error: %v", t.ID, syncErr)
				}
				logx.WithContext(ctx).Infof("manager: cycle trader=%s decisions=%d actions=%d ok=%t duration=%s", t.ID, decisionCount, len(actions), allOK && decisionErr == nil, time.Since(cycleStart).String())
			}
		}
	}
}

// Stop signals the main loop to exit.
func (m *Manager) Stop() {
	m.stopOnce.Do(func() {
		logx.Info("manager: stop signal emitted")
		close(m.stopChan)
	})
}

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
		// Attempt to cancel resting orders via optional extension
		if p, ok := trader.ExchangeProvider.(interface {
			CancelAllBySymbol(context.Context, string) error
		}); ok {
			_ = p.CancelAllBySymbol(ctx, decision.Symbol)
		}
		if err := trader.ExchangeProvider.ClosePosition(ctx, decision.Symbol); err != nil {
			return err
		}
		logx.Infof("manager: trader %s closed position symbol=%s action=%s", trader.ID, decision.Symbol, decision.Action)
		// Mark cooldown timestamp on successful close
		trader.mu.Lock()
		trader.Cooldown[decision.Symbol] = time.Now()
		trader.mu.Unlock()
		return nil
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
			lev = trader.RiskParams.MajorCoinLeverage
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

	// Format price/size via provider if available
	priceStr := fmt.Sprintf("%.8f", price)
	sizeStr := fmt.Sprintf("%.8f", qty)
	if p, ok := trader.ExchangeProvider.(interface {
		FormatPrice(context.Context, string, float64) (string, error)
	}); ok {
		if s, err := p.FormatPrice(ctx, decision.Symbol, price); err == nil && s != "" {
			priceStr = s
		}
	}
	if p, ok := trader.ExchangeProvider.(interface {
		FormatSize(context.Context, string, float64) (string, error)
	}); ok {
		if s, err := p.FormatSize(ctx, decision.Symbol, qty); err == nil && s != "" {
			sizeStr = s
		}
	}

	// Submit a limit IOC order approximating a marketable order.
	cloid := buildCloid(trader.ID, decision.Symbol, decision.Action, qty, time.Now())
	order := exchange.Order{
		Asset:      assetIdx,
		IsBuy:      isBuy,
		LimitPx:    priceStr,
		Sz:         sizeStr,
		ReduceOnly: false,
		OrderType:  exchange.OrderType{Limit: &exchange.LimitOrderType{TIF: "Ioc"}},
		Cloid:      cloid,
	}
	if _, err := trader.ExchangeProvider.PlaceOrder(ctx, order); err != nil {
		return fmt.Errorf("manager: place order %s %s: %w", decision.Symbol, decision.Action, err)
	}
	logx.Infof("manager: trader %s submitted %s order symbol=%s notional=%.2f usd qty=%.6f cloid=%s", trader.ID, decision.Action, decision.Symbol, decision.PositionSizeUSD, qty, cloid)
	// Configure reduce-only SL/TP best-effort
	side := "LONG"
	if !isBuy { // open_short
		side = "SHORT"
	}
	// Best-effort SL/TP via optional provider extension.
	if p, ok := trader.ExchangeProvider.(interface {
		SetStopLoss(context.Context, string, string, float64, float64) error
		SetTakeProfit(context.Context, string, string, float64, float64) error
	}); ok {
		_ = p.SetStopLoss(ctx, decision.Symbol, side, qty, decision.StopLoss)
		_ = p.SetTakeProfit(ctx, decision.Symbol, side, qty, decision.TakeProfit)
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
	logx.Infof("manager: trader %s equity=%.2f usd margin_used=%.2f usd avail=%.2f usd unreal_pnl=%.2f usd", traderID, acctVal, marginUsed, t.ResourceAlloc.AvailableBalanceUSD, unreal)
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

func outPromptDigest(out *executorpkg.FullDecision) string {
	if out == nil {
		return ""
	}
	if strings.TrimSpace(out.UserPrompt) == "" {
		return ""
	}
	return llm.DigestString(out.UserPrompt)
}

func parsePtrFloat(ps *string) float64 {
	if ps == nil {
		return 0
	}
	return parseFloat(*ps)
}

func isBTCorETH(symbol string) bool {
	switch symbol {
	case "BTC", "ETH", "BTCUSDT", "ETHUSDT":
		return true
	default:
		return false
	}
}

func (m *Manager) writeJournalRecord(t *VirtualTrader, ectx *executorpkg.Context, out *executorpkg.FullDecision, decisionsJSON string, actions []map[string]any, callErr error, allOK bool) error {
	if t == nil || t.Journal == nil || ectx == nil {
		return nil
	}
	acc := map[string]any{
		"equity":      ectx.Account.TotalEquity,
		"available":   ectx.Account.AvailableBalance,
		"used_margin": ectx.Account.MarginUsed,
		"used_pct":    ectx.Account.MarginUsedPct,
		"positions":   ectx.Account.PositionCount,
	}
	pos := make([]map[string]any, 0, len(ectx.Positions))
	for _, p := range ectx.Positions {
		pos = append(pos, map[string]any{
			"symbol": p.Symbol,
			"side":   p.Side,
			"qty":    p.Quantity,
			"lev":    p.Leverage,
			"entry":  p.EntryPrice,
			"mark":   p.MarkPrice,
			"upnl":   p.UnrealizedPnL,
			"liq":    p.LiquidationPrice,
		})
	}
	marketDigest := make(map[string]any, len(ectx.MarketDataMap))
	for sym, s := range ectx.MarketDataMap {
		if s == nil {
			continue
		}
		md := map[string]any{
			"price": s.Price.Last,
			"chg1h": s.Change.OneHour,
			"chg4h": s.Change.FourHour,
		}
		if s.OpenInterest != nil {
			md["oi_latest"] = s.OpenInterest.Latest
		}
		if s.Funding != nil {
			md["funding"] = s.Funding.Rate
		}
		marketDigest[sym] = md
	}

	cot := ""
	promptDigest := ""
	if out != nil {
		cot = out.CoTTrace
		if s := strings.TrimSpace(out.UserPrompt); s != "" {
			promptDigest = llm.DigestString(s)
		}
	}
	// candidates list as strings for compactness
	var cand []string
	for _, c := range ectx.CandidateCoins {
		cand = append(cand, c.Symbol)
	}
	rec := &journal.CycleRecord{
		TraderID:      t.ID,
		PromptDigest:  promptDigest,
		CoTTrace:      cot,
		DecisionsJSON: decisionsJSON,
		Account:       acc,
		Positions:     pos,
		Candidates:    cand,
		MarketDigest:  marketDigest,
		Actions:       actions,
		Success:       allOK && callErr == nil,
	}
	if callErr != nil {
		rec.ErrorMessage = callErr.Error()
	}
	_, err := t.Journal.WriteCycle(rec)
	return err
}

// buildCloid creates a stable client order id for idempotent intent submission.
func buildCloid(traderID, symbol, action string, qty float64, now time.Time) string {
	// Bucket time to minute to avoid collision across cycles; include rounded qty to 6 dp.
	ts := now.UTC().Format("20060102T1504")
	return fmt.Sprintf("%s|%s|%s|%.6f|%s", traderID, strings.ToUpper(symbol), action, qty, ts)
}

// buildExecutorContext collects a richer snapshot for the executor prompt and validation.
func (m *Manager) buildExecutorContext(t *VirtualTrader) executorpkg.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1) Account and positions
	acctState, _ := t.ExchangeProvider.GetAccountState(ctx)
	positionsRaw, _ := t.ExchangeProvider.GetPositions(ctx)

	// Normalize account info
	account := executorpkg.AccountInfo{}
	if acctState != nil {
		account.TotalEquity = parseFloat(acctState.MarginSummary.AccountValue)
		account.MarginUsed = parseFloat(acctState.MarginSummary.TotalMarginUsed)
		account.AvailableBalance = account.TotalEquity - account.MarginUsed
		// Aggregate unrealized PnL
		for i := range acctState.AssetPositions {
			account.TotalPnL += parseFloat(acctState.AssetPositions[i].UnrealizedPnl)
		}
		if account.TotalEquity != 0 {
			account.MarginUsedPct = 100 * (account.MarginUsed / account.TotalEquity)
			account.TotalPnLPct = 100 * (account.TotalPnL / account.TotalEquity)
		}
	}

	// Normalize positions (first pass: collect symbols and static fields)
	positions := make([]executorpkg.PositionInfo, 0, len(positionsRaw))
	symbols := make(map[string]struct{})
	for i := range positionsRaw {
		p := positionsRaw[i]
		side := "long"
		qty := parseFloat(p.Szi)
		if qty < 0 {
			side = "short"
			qty = -qty
		}
		positions = append(positions, executorpkg.PositionInfo{
			Symbol:           p.Coin,
			Side:             side,
			EntryPrice:       parsePtrFloat(p.EntryPx),
			MarkPrice:        0,
			Quantity:         qty,
			Leverage:         p.Leverage.Value,
			UnrealizedPnL:    parseFloat(p.UnrealizedPnl),
			LiquidationPrice: parsePtrFloat(p.LiquidationPx),
		})
		symbols[p.Coin] = struct{}{}
	}
	account.PositionCount = len(positions)

	// 2) Candidate set (basic Top-N by |1h change|) and market snapshots
	candidates := m.selectCandidates(ctx, t, 0)
	snaps := map[string]*market.Snapshot{}
	for sym := range symbols {
		if s, err := t.MarketProvider.Snapshot(ctx, sym); err == nil && s != nil {
			snaps[sym] = s
		}
	}
	// Snapshots for candidates
	for _, c := range candidates {
		if _, ok := snaps[c.Symbol]; ok { // already fetched
			continue
		}
		if s, err := t.MarketProvider.Snapshot(ctx, c.Symbol); err == nil && s != nil {
			snaps[c.Symbol] = s
		}
	}

	// Second pass: enrich mark price and pnl pct from snapshots
	for i := range positions {
		pi := &positions[i]
		if pi.EntryPrice > 0 {
			if s, ok := snaps[pi.Symbol]; ok && s != nil && s.Price.Last > 0 {
				pi.MarkPrice = s.Price.Last
				pi.UnrealizedPnLPct = 100 * (pi.MarkPrice - pi.EntryPrice) / pi.EntryPrice
			}
		}
	}

	// 3) Asset meta (max leverage, precision) for present symbols
	assetMeta := map[string]executorpkg.AssetMeta{}
	if assets, err := t.MarketProvider.ListAssets(ctx); err == nil {
		for _, a := range assets {
			if _, want := symbols[a.Symbol]; !want {
				continue
			}
			ml := 0.0
			onlyIso := false
			if a.RawMetadata != nil {
				if v, ok := a.RawMetadata["maxLeverage"]; ok {
					switch x := v.(type) {
					case float64:
						ml = x
					case int:
						ml = float64(x)
					}
				}
				if v, ok := a.RawMetadata["onlyIsolated"]; ok {
					if b, ok := v.(bool); ok {
						onlyIso = b
					}
				}
			}
			assetMeta[a.Symbol] = executorpkg.AssetMeta{MaxLeverage: ml, Precision: a.Precision, OnlyIsolated: onlyIso}
		}
	}

	// 4) Compose executor context
	return executorpkg.Context{
		CurrentTime:       time.Now().UTC().Format(time.RFC3339),
		RuntimeMinutes:    0,
		CallCount:         0,
		Account:           account,
		Positions:         positions,
		CandidateCoins:    candidates,
		MarketDataMap:     snaps,
		OpenInterestMap:   nil,
		Performance:       t.Performance.ToExecutorView(),
		MajorCoinLeverage: t.RiskParams.MajorCoinLeverage,
		AltcoinLeverage:   t.RiskParams.AltcoinLeverage,
		AssetMeta:         assetMeta,
		// Optional guards sourced from trader risk params when enabled
		MaxMarginUsagePct: func() float64 {
			if t.ExecGuards.EnableMarginUsageGuard == nil || *t.ExecGuards.EnableMarginUsageGuard {
				return t.RiskParams.MaxMarginUsagePct
			}
			return 0
		}(),
		LiquidityThresholdUSD: func() float64 {
			if t.ExecGuards.EnableLiquidityGuard == nil || *t.ExecGuards.EnableLiquidityGuard {
				return t.ExecGuards.LiquidityThresholdUSD
			}
			return 0
		}(),
		BTCETHPositionValueMinMultiple: func() float64 {
			if t.ExecGuards.EnableValueBandGuard == nil || *t.ExecGuards.EnableValueBandGuard {
				return t.ExecGuards.BTCETHMinEquityMultiple
			}
			return 0
		}(),
		BTCETHPositionValueMaxMultiple: func() float64 {
			if t.ExecGuards.EnableValueBandGuard == nil || *t.ExecGuards.EnableValueBandGuard {
				return t.ExecGuards.BTCETHMaxEquityMultiple
			}
			return 0
		}(),
		AltPositionValueMinMultiple: func() float64 {
			if t.ExecGuards.EnableValueBandGuard == nil || *t.ExecGuards.EnableValueBandGuard {
				return t.ExecGuards.AltMinEquityMultiple
			}
			return 0
		}(),
		AltPositionValueMaxMultiple: func() float64 {
			if t.ExecGuards.EnableValueBandGuard == nil || *t.ExecGuards.EnableValueBandGuard {
				return t.ExecGuards.AltMaxEquityMultiple
			}
			return 0
		}(),
		CooldownAfterClose: func() time.Duration {
			if t.ExecGuards.EnableCooldownGuard == nil || *t.ExecGuards.EnableCooldownGuard {
				return t.ExecGuards.CooldownAfterClose
			}
			return 0
		}(),
	}
}

// selectCandidates picks up to limit candidates using a simple heuristic (|1h change| ranking).
// If limit == 0, uses ExecGuards.CandidateLimit (defaults to 10 when <=0). Applies liquidity threshold when enabled.
func (m *Manager) selectCandidates(ctx context.Context, t *VirtualTrader, limit int) []executorpkg.CandidateCoin {
	if limit <= 0 {
		limit = t.ExecGuards.CandidateLimit
		if limit <= 0 {
			limit = 10
		}
	}
	assets, err := t.MarketProvider.ListAssets(ctx)
	if err != nil || len(assets) == 0 {
		return nil
	}
	// Fetch snapshots (keep to first 200 assets to bound cost)
	type item struct {
		sym   string
		score float64
	}
	ranked := make([]item, 0, limit*3)
	count := 0
	for _, a := range assets {
		if !a.IsActive {
			continue
		}
		s, err := t.MarketProvider.Snapshot(ctx, a.Symbol)
		if err != nil || s == nil {
			continue
		}
		// Liquidity threshold if enabled
		if (t.ExecGuards.EnableLiquidityGuard == nil || *t.ExecGuards.EnableLiquidityGuard) && t.ExecGuards.LiquidityThresholdUSD > 0 {
			if s.OpenInterest != nil {
				if s.OpenInterest.Latest*s.Price.Last+1e-9 < t.ExecGuards.LiquidityThresholdUSD {
					continue
				}
			}
		}
		score := s.Change.OneHour
		if score < 0 {
			score = -score
		}
		ranked = append(ranked, item{sym: a.Symbol, score: score})
		count++
		if count >= 200 {
			break
		}
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]executorpkg.CandidateCoin, 0, len(ranked))
	for _, it := range ranked {
		out = append(out, executorpkg.CandidateCoin{Symbol: it.sym, Sources: []string{"rank_1h_abs"}})
	}
	return out
}

// sortDecisionsCloseFirst returns decisions ordered by priority: close_* first, then open_*.
func sortDecisionsCloseFirst(ds []executorpkg.Decision) []executorpkg.Decision {
	out := make([]executorpkg.Decision, len(ds))
	copy(out, ds)
	sort.SliceStable(out, func(i, j int) bool {
		pri := priority(out[i].Action)
		prj := priority(out[j].Action)
		if pri != prj {
			return pri < prj
		}
		return out[i].Symbol < out[j].Symbol
	})
	return out
}

func priority(action string) int {
	switch action {
	case "close_long", "close_short":
		return 0
	case "open_long", "open_short":
		return 1
	default:
		return 2
	}
}

// capNewOpenDecisions limits the number of new open actions to remainingSlots; non-open actions are kept.
func capNewOpenDecisions(ds []executorpkg.Decision, remainingSlots int) []executorpkg.Decision {
	if remainingSlots <= 0 {
		// Keep only non-open actions (e.g., closes/hold/wait)
		out := make([]executorpkg.Decision, 0, len(ds))
		for _, d := range ds {
			if d.Action != "open_long" && d.Action != "open_short" {
				out = append(out, d)
			}
		}
		return out
	}
	opens := 0
	out := make([]executorpkg.Decision, 0, len(ds))
	for _, d := range ds {
		if d.Action == "open_long" || d.Action == "open_short" {
			if opens >= remainingSlots {
				continue
			}
			opens++
		}
		out = append(out, d)
	}
	return out
}
