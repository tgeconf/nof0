# Engine Layer Data & Type Reference

This document describes the runtime “engine” packages under `pkg/` – `exchange`, `market`, `llm`, `executor`, `manager`, and `journal`. It captures how data moves between them, enumerates the major entities, flags which fields come straight from primary data stores (Postgres, Redis, external APIs, or configuration) versus those computed from that data, and records the formulas or derivation rules.

> **Legend**  
> - **Primary (Exchange API)** – raw values fetched from a live exchange (currently Hyperliquid) via `exchange.Provider`, persisted later in `positions`, `price_ticks`, etc.  
> - **Primary (Market API)** – market data fetched via `market.Provider`; cached in Redis keys such as `nof0:price:latest:{symbol}`.  
> - **Primary (DB/Redis)** – values loaded from Postgres/Redis, e.g., analytics snapshots once the JSON loader is replaced.  
> - **Primary (Config)** – static settings from YAML configs (`etc/*.yaml`).  
> - **Primary (Runtime)** – user input or direct LLM outputs captured without transformations.  
> - **Derived** – computed from primary data in-process; formulas noted in the tables.

---

## 0. Configuration Chain Snapshot

`etc/nof0.yaml` acts as the root manifest and points each subsystem to its YAML file:

- **LLM** → `etc/llm.yaml` (base URL `https://zenmux.ai/api/v1`, default model `gpt-5`, timeout `60s`, retry budget `3`, API key via `${ZENMUX_API_KEY}`).
- **Executor** → `etc/executor.yaml` (baseline: `major_coin_leverage: 20`, `altcoin_leverage: 10`, `min_confidence: 75`, `min_risk_reward: 3.0`, `max_positions: 4`, `decision_interval: 3m`, `decision_timeout: 60s`).
- **Manager** → `etc/manager.yaml` (book size `10000` USD with `reserve_equity_pct: 10`, allocation strategy `performance_based`, state persisted to `../data/manager_state.json`, two sample traders configured).
- **Exchange** → `etc/exchange.yaml` (default provider `hyperliquid_testnet`; credentials supplied via `${HYPERLIQUID_PRIVATE_KEY}`, `${HYPERLIQUID_MAIN_ADDRESS}`, optional `${HYPERLIQUID_VAULT_ADDRESS}`).
- **Market** → `etc/market.yaml` (default provider `hyperliquid_testnet`; per-call timeout `8s`, HTTP timeout `10s`, retry budget `3`, mirrored mainnet entry `hyperliquid`).

Environment placeholders (`${...}`) are expanded before validation, so runtime deployments must ensure the appropriate secrets are injected. When DB/Redis connectivity is enabled (see `Postgres`/`Cache` blocks in `etc/nof0.yaml`), hydrated values should align with the data contracts documented below.

---

## 1. High-Level Data Flow

1. **Manager** loads `manager.Config` (Primary Config) and builds `VirtualTrader` instances. Each trader is wired with an `exchange.Provider`, `market.Provider`, `executor.Executor`, and optional `journal.Writer`.
2. **Manager** periodically calls `exchange.Provider` to fetch account/position state (Primary Exchange API) and normalises it into `executor.Context`.
3. **Manager** enriches the context with market snapshots and asset metadata via `market.Provider` (Primary Market API) and injects guardrails from config (Derived).
4. **Executor** renders prompts using the context, calls `llm.LLMClient.ChatStructured`, and receives `Decision` payloads (Primary Runtime). Validation derives additional metrics (Derived) before handing decisions back.
5. **Manager** executes validated decisions through `exchange.Provider.PlaceOrder` / `ClosePosition`, updating optional stop-loss/take-profit orders. Order responses remain Primary Exchange API data.
6. **Manager** and **Executor** update `PerformanceMetrics`, `PerformanceView`, and guard state (Derived) and write each cycle to `journal.Writer` for audit (Primary Runtime + Derived).
7. **LLM** layer wraps configuration for external LLM providers (Primary Config) and handles retries/logging. Responses can be cached (future Redis) but are currently transient.

---

## 2. Package Reference

### 2.1 `pkg/exchange`

**Purpose.** Normalised trading interface around venue-specific providers (Hyperliquid today). Downstream consumers (manager/executor) never touch venue-specific JSON.

**Provider Interface.**

- `PlaceOrder`, `CancelOrder`, `GetOpenOrders`
- `GetPositions`, `ClosePosition`, `UpdateLeverage`
- `GetAccountState`, `GetAccountValue`
- `GetAssetIndex`
- Optional extensions (Hyperliquid): `IOCMarket`, `FormatPrice`, `FormatSize`, `SetStopLoss`, `SetTakeProfit`, `CancelAllBySymbol`, `SetMarkPrice`

**Configuration Entities.**

| Entity | Field | Description | Provenance |
|--------|-------|-------------|------------|
| `Config` | `Default` | Default provider alias (`hyperliquid_testnet`). | Primary Config (`etc/exchange.yaml`) |
| `Config` | `Providers` | Named provider configs (`hyperliquid_testnet`, `paper_trading`). | Primary Config |
| `ProviderConfig` | `Type`, `PrivateKey`, `APIKey`, `APISecret`, `Passphrase`, `VaultAddress`, `MainAddress`, `Testnet` | Credentials and environment flags (Hyperliquid pulls `${HYPERLIQUID_*}` env vars; simulator requires none). | Primary Config (env-expanded) |
| `ProviderConfig` | `Timeout` | Request timeout parsed from `TimeoutRaw` (e.g., `30s` for Hyperliquid testnet). | Derived (`time.ParseDuration`) |

**Trading Entities.**

| Type | Field | Description | Provenance / Formula |
|------|-------|-------------|----------------------|
| `OrderType` | `Limit`, `Trigger` | Optional sub-structures passed through to venue. | Primary Runtime (manager/executor input) |
| `LimitOrderType` | `TIF` | Time-in-force (Alo/Ioc/Gtc). | Primary Runtime |
| `TriggerOrderType` | `IsMarket`, `Tpsl`, `TriggerRel` | Trigger semantics. | Primary Runtime |
| `BuilderInfo` | `Name`, `FeeBps` | Routing metadata for fee tiers. | Primary Runtime / Config overrides |
| `Order` | `Asset`, `IsBuy`, `LimitPx`, `Sz`, `ReduceOnly`, `OrderType`, `Cloid`, `TriggerPx`, `TriggerRel`, `Grouping`, `Builder` | Canonical order request. | Primary Runtime (constructed by manager) |
| `Position` | `Coin`, `EntryPx`, `PositionValue`, `Szi`, `UnrealizedPnl`, `ReturnOnEquity`, `Leverage`, `LiquidationPx` | Live position snapshot. | Primary Exchange API (`GetPositions`) |
| `Leverage` | `Type`, `Value` | Instrument leverage mode. | Primary Exchange API |
| `AccountState` | `MarginSummary`, `CrossMarginSummary`, `AssetPositions` | Top-level account info. | Primary Exchange API |
| `MarginSummary` / `CrossMarginSummary` | `AccountValue`, `TotalMarginUsed`, `TotalNtlPos`, `TotalRawUsd` | Margin metrics (strings). | Primary Exchange API; converted to floats by manager. |
| `OrderStatus` | `Order`, `Status`, `StatusTimestamp` | Pending/open order info. | Primary Exchange API |
| `OrderInfo` | `Coin`, `Side`, `LimitPx`, `Sz`, `Oid`, `Timestamp`, `OrigSz`, `Cloid` | Order metadata. | Primary Exchange API |
| `Fill` | `AvgPx`, `TotalSz`, `LimitPx`, `Sz`, `Oid`, `Crossed`, `Fee`, `Tid`, `Timestamp` | Execution fill. | Primary Exchange API |
| `OrderResponse` | `Status`, `Response`, `ErrorMessage` | Submission result; `ErrorMessage` populated when venue returns string. | `Status`, `Response`: Primary Exchange API; `ErrorMessage`: Derived parsing fallback |
| `OrderResponseData` | `Type`, `Data` | Wrapper around statuses. | Primary Exchange API |
| `OrderStatusResponse` | `Resting`, `Filled`, `Error` | per-order result. | Primary Exchange API |

**Hyperliquid Provider Highlights.**

- `Action`, `Cancel`, `CancelByCloid`, `Modify`, `ExchangeRequest`, `Signature` mirror Hyperliquid JSON (Primary Exchange API structures).
- `AssetUniverseEntry`, `AssetCtx`, `AssetInfo` feed into `market.Asset` raw metadata (Primary Exchange API → Derived aggregator).
- `NoncePayload` provides signing nonces (Primary Exchange API).

### 2.2 `pkg/market`

**Purpose.** Normalise market data (price snapshots, indicators, series) across venues.

**Provider Interface.**  
`Snapshot(ctx, symbol)` ➞ `Snapshot`; `ListAssets(ctx)` ➞ `[]Asset`.

**Configuration Entities.**

| Entity | Field | Description | Provenance |
|--------|-------|-------------|------------|
| `Config` | `Default`, `Providers` | Provider registry (default `hyperliquid_testnet`, companion mainnet entry `hyperliquid`). | Primary Config (`etc/market.yaml`) |
| `ProviderConfig` | `Type`, `Testnet`, `Mode`, `MaxRetries` | Provider settings (`testnet: true/false`, `max_retries: 3`). | Primary Config |
| `ProviderConfig` | `Timeout`, `HTTPTimeout` | Durations parsed from raw strings (`timeout: 8s`, `http_timeout: 10s`). | Derived |

**Market Entities.**

| Type | Field | Description | Provenance / Formula |
|------|-------|-------------|----------------------|
| `Snapshot` | `Symbol` | Exchange-native symbol. | Primary Market API |
| | `Price.Last` | Last trade / mark. | Primary Market API (cached in Redis `nof0:price:latest:*`) |
| | `Change.OneHour`, `Change.FourHour` | Fractional returns. | Derived by provider from price history (`(close_now - close_prev)/close_prev`). |
| | `Indicators.EMA` | EMA map keyed by window. | Derived by provider from OHLCV series. |
| | `Indicators.MACD` | MACD value. | Derived (EMA12-EMA26). |
| | `Indicators.RSI` | RSI values keyed by period. | Derived (Wilder smoothing). |
| | `OpenInterest.Latest`, `Average` | OI metrics where venue supports. | Primary Market API (cached Redis `nof0:oi:{symbol}`) |
| | `Funding.Rate` | Perpetual funding (decimal). | Primary Market API |
| | `Intraday`, `LongTerm` | Bundled historical series. | Derived packaging of OHLCV / indicator arrays from provider data. |
| `Asset` | `Symbol`, `Base`, `Quote`, `Precision`, `IsActive` | Static symbol metadata. | Primary Market API |
| | `RawMetadata` | Venue-specific map (e.g., `maxLeverage`, `onlyIsolated`). | Primary Market API |
| `SeriesBundle` | `Prices`, `EMA`, `MACD`, `RSI`, `ATR`, `Volume` | Historical arrays for signal generation. | Derived from OHLCV caches / `price_ticks` view. |

**Hyperliquid Adapter Notes.**

- `Kline` represents raw candle rows (Primary Market API).  
- `MetaAndAssetCtxsResponse` merges universe + per-asset contexts; `UniverseEntry` surfaces to `Asset.RawMetadata`.  
- `AssetCtx.OpenInterest`, `AssetCtx.MarkPx` feed `Snapshot.OpenInterest` and `Snapshot.Price`.

### 2.3 `pkg/llm`

**Purpose.** Wrap external LLM providers (Zenmux/OpenAI-compatible) with configuration, logging, retries, and structured output helpers.

**Client Interface.**

- `Chat`, `ChatStream`, `ChatStructured`, `GetConfig`, `Close`.

**Configuration Entities.**

| Type | Field | Description | Provenance |
|------|-------|-------------|------------|
| `Config` | `BaseURL`, `APIKey`, `DefaultModel`, `Timeout`, `MaxRetries`, `LogLevel` | Core settings (defaults: `https://zenmux.ai/api/v1`, `${ZENMUX_API_KEY}`, model `gpt-5`, timeout `60s`, retries `3`, log level `info`). | Primary Config (env overrides allowed) |
| `Config` | `Timeout` | Request timeout parsed from YAML/env. | Derived |
| `Config` | `MaxRetries` | Retry count. | Primary Config/env |
| `Config` | `Models` | Alias map (`name` → `ModelConfig`) — sample entries: `gpt-5`, `claude-sonnet-4.5`, `deepseek-chat`. | Primary Config |
| `Config` | `RoutingDefaults` | Default routing for `zenmux/auto`. | Primary Config |
| `ModelConfig` | `Provider`, `ModelName`, `Temperature`, `MaxCompletionTokens`, `TopP` | Per-alias defaults (e.g., `gpt-5` → provider `openai`, temp 0.7, tokens 4096). | Primary Config |

**Chat Entities.**

| Type | Field | Description | Provenance / Formula |
|------|-------|-------------|----------------------|
| `ChatRequest` | `Model` | Target model alias or provider/model string. | Primary Runtime (executor input) |
| | `Messages` | Conversation turns. | Primary Runtime |
| | `Temperature`, `TopP`, `MaxCompletionTokens`, `Stream` | Generation parameters. | Primary Runtime or derived from `ModelConfig` defaults. |
| | `ResponseFormat` | JSON schema contract for structured decoding. | Primary Runtime |
| | `Routing` | Zenmux routing instructions. | Primary Runtime/Config (defaults inserted) |
| `Message` | `Role`, `Content`, `Name`, `ToolCallID` | Conversation unit. | Primary Runtime |
| `ChatResponse` | `ID`, `Model`, `Choices`, `Usage`, `Created`, `RawJSON`, `Tier`, `Fingerprint` | Completion metadata. | Primary LLM API |
| `Choice` | `Index`, `Message`, `FinishReason`, `ToolCalls` | Single candidate. | Primary LLM API |
| `ToolCall` | `ID`, `Type`, `Function` | Structured tool results. | Primary LLM API |
| `Usage` | `PromptTokens`, `CompletionTokens`, `TotalTokens` | Token accounting. | Primary LLM API |
| `StreamResponse`, `StreamChoice`, `Delta` | Streaming chunks. | Primary LLM API |

**Client Runtime Structures.**

- `RetryHandler` tracks retry strategy (Derived from config).  
- `Logger` attaches metadata to requests/responses (Derived summarisation).  
- `ResolveModelID`, `ParseModelID` produce provider/model strings (Derived string helpers).

### 2.4 `pkg/executor`

**Purpose.** Convert normalised account/market context into LLM prompts and validate `Decision` outputs.

**Configuration Highlights (`Config`).**

| Field | Description | Provenance |
|-------|-------------|------------|
| `MajorCoinLeverage`, `AltcoinLeverage`, `MinConfidence`, `MinRiskReward`, `MaxPositions`, `MaxConcurrentDecisions`, `AllowedTraderIDs`, `SigningKey` | Executor operating thresholds (defaults: 20×, 10×, confidence 75, risk/reward 3.0, max positions 4, concurrency 1, signing key empty). | Primary Config (`etc/executor.yaml`; adapted per trader if overrides supplied) |
| `DecisionInterval`, `DecisionTimeout` | Durations parsed from raw strings (`3m`, `60s`). | Derived |
| `Overrides` | Per-trader symbol overrides. | Primary Config |

**Context & Supporting Types.**

| Type | Field | Description | Provenance / Formula |
|------|-------|-------------|----------------------|
| `PositionInfo` | `Symbol`, `Side`, `EntryPrice`, `Quantity`, `Leverage`, `UnrealizedPnL`, `LiquidationPrice` | Normalised from `exchange.Position`. | Primary Exchange API (normalised by manager) |
| | `MarkPrice` | Latest mark. | Primary Market API (`Snapshot.Price.Last`) |
| | `UnrealizedPnLPct` | %-based unrealised PnL. | Derived: `100 * (MarkPrice - EntryPrice) / EntryPrice` |
| | `MarginUsed` | Margin allocated per position. | Derived: `Quantity * EntryPrice / Leverage` (manager to populate). |
| | `UpdateTime` | Last refresh timestamp. | Derived (`time.Now().Unix()`) when sync occurs. |
| `AccountInfo` | `TotalEquity`, `AvailableBalance`, `TotalPnL`, `MarginUsed` | From `exchange.AccountState`. | Primary Exchange API (string→float) |
| | `TotalPnLPct` | Equity return. | Derived: `100 * TotalPnL / TotalEquity` |
| | `MarginUsedPct` | Margin use %. | Derived: `100 * MarginUsed / TotalEquity` |
| | `PositionCount` | Open positions. | Derived length of `Positions`. |
| `CandidateCoin` | `Symbol`, `Sources` | Candidate list for prompts (heuristics). | Derived (manager ranking). |
| `OpenInterest` | `Latest`, `Average` | Additional OI data if not provided in `market.Snapshot`. | Primary Market API (future) |
| `PerformanceView` | `SharpeRatio`, `WinRate`, `TotalTrades`, `RecentTradesRate`, `UpdatedAt` | Aggregated KPIs shown in prompts. | Derived from `manager.PerformanceMetrics` |
| `AssetMeta` | `MaxLeverage`, `Precision`, `OnlyIsolated` | Venue-specific constraints from `market.Asset.RawMetadata`. | Primary Market API |
| `Context` | `CurrentTime`, `RuntimeMinutes`, `CallCount` | Scheduler metadata. | Derived (manager loop) |
| | `Account`, `Positions`, `CandidateCoins`, `MarketDataMap`, `OpenInterestMap` | Consolidated domain state. | Mixed (Primary Exchange / Market / Derived ranking) |
| | `Performance` | Optional pointer. | Derived |
| | `MajorCoinLeverage`, `AltcoinLeverage`, guard fields (`MaxRiskPct`, `MaxPositionSizeUSD`, `LiquidityThresholdUSD`, `MaxMarginUsagePct`, `ValueBand*`, `CooldownAfterClose`, `RecentlyClosed`) | Configuration-sourced guardrails. | Derived from `manager.TraderConfig.ExecGuards` + runtime cooldown map |
| `Decision` | `Symbol`, `Action`, `Leverage`, `PositionSizeUSD`, `EntryPrice`, `StopLoss`, `TakeProfit`, `Confidence`, `RiskUSD`, `Reasoning`, `InvalidationCondition` | Structured LLM output. | Primary Runtime (LLM) except: `Leverage` may default to guard defaults; `RiskUSD` derived as `PositionSizeUSD * (1/Leverage)` when missing. |
| `FullDecision` | `UserPrompt`, `CoTTrace`, `Decisions`, `Timestamp` | Prompt echo + LLM results. | `UserPrompt`: Derived from template rendering; `Decisions`: Primary Runtime; `Timestamp`: Derived (`time.Now()`). |

**Key Flows.**

- `BuildContext` merges base context with live market snapshots (`market.Provider`).  
- `ValidateDecisions` enforces guardrails, computing new margin usage and liquidity checks (Derived).  
- `mapDecisionContract` converts structured JSON (`decisionContract`) into `Decision`, calculating defaults such as side inference (Derived).  
- Failures tracked via `BasicExecutor.failures` for retry heuristics (Derived counters).

### 2.5 `pkg/manager`

**Purpose.** Orchestrates traders, wiring exchange/market providers, executors, guardrails, and journaling.

**Configuration Entities.**

| Type | Field | Description | Provenance |
|------|-------|-------------|------------|
| `Config` | `Manager`, `Traders`, `Monitoring` | Top-level configuration. | Primary Config |
| `ManagerConfig` | `TotalEquityUSD`, `ReserveEquityPct`, `AllocationStrategy`, `StateStorageBackend`, `StateStoragePath` | Portfolio policy. | Primary Config |
| | `RebalanceInterval` | Parsed from `RebalanceIntervalRaw`. | Derived |
| `TraderConfig` | `ID`, `Name`, `ExchangeProvider`, `MarketProvider`, `OrderStyle`, `MarketIOCSlippageBps`, `PromptTemplate`, `ExecutorTemplate`, `Model`, `DecisionInterval`, `RiskParams`, `ExecGuards`, `AllocationPct`, `AutoStart`, `JournalEnabled`, `JournalDir` | Trader-specific wiring. | Primary Config (paths env-resolved) |
| | `DecisionInterval` | Parsed duration. | Derived |
| `RiskParameters` | `MaxPositions`, `MaxPositionSizeUSD`, `MaxMarginUsagePct`, `MajorCoinLeverage`, `AltcoinLeverage`, `MinRiskRewardRatio`, `MinConfidence`, `StopLossEnabled`, `TakeProfitEnabled` | Risk caps (sample: aggressive trader 3 positions / 500 USD cap / 60 % margin / 20× majors / 10× alts; conservative trader 2 / 300 USD / 50 % / 10× / 5×). | Primary Config |
| `ExecGuards` | `MaxNewPositionsPerCycle`, `LiquidityThresholdUSD`, `MaxMarginUsagePct` | Execution guardrails (sample config leaves these unset → defaults disable guards). | Primary Config |
| | `BTCETHMinEquityMultiple`, `BTCETHMaxEquityMultiple`, `AltMinEquityMultiple`, `AltMaxEquityMultiple` | Value band guardrails. | Primary Config |
| | `CooldownAfterClose`, `PauseDurationOnBreach` | Durations parsed from raw strings. | Derived |
| | `Enable*Guard`, `CandidateLimit`, `SharpePauseThreshold` | Feature toggles, heuristics. | Primary Config |
| `MonitoringConfig` | `UpdateInterval`, `AlertWebhook`, `MetricsExporter` | Monitoring outputs (sample: `update_interval: 15s`, `metrics_exporter: prometheus`, webhook empty by default). | `UpdateInterval`: Derived; others Primary Config |

**Runtime Entities.**

| Type | Field | Description | Provenance / Formula |
|------|-------|-------------|----------------------|
| `Manager` | `config`, `traders`, `exchangeProviders`, `marketProviders`, `executorFactory`, `stopChan`, `wg` | Orchestration state. | Derived runtime wiring |
| `ExecutorFactory` | `NewExecutor` | Builds executors from trader config. | Derived (adapts config) |
| `VirtualTrader` | `ID`, `Name`, `Exchange`, `ExchangeProvider`, `MarketProvider`, `Executor`, `PromptTemplate`, `OrderStyle`, `MarketIOCSlippageBps`, `RiskParams`, `ExecGuards`, `DecisionInterval`, `CreatedAt`, `UpdatedAt`, `State`, `Performance`, `LastDecisionAt`, `Cooldown`, `Journal`, `JournalEnabled`, `PauseUntil`, `ResourceAlloc` | Trader state container. | Mix: from config (Primary), runtime updates (Derived). |
| `TraderState` | Enum (`running`, `paused`, `stopped`, `error`). | Derived from lifecycle. |
| `ResourceAllocation` | `AllocatedEquityUSD`, `AllocationPct` | From config (Primary). |
| | `CurrentEquityUSD`, `AvailableBalanceUSD`, `MarginUsedUSD`, `UnrealizedPnLUSD` | Derived from `exchange.AccountState`. |
| | `IsOverAllocated()` | Derived check: `MarginUsedUSD > AllocatedEquityUSD`. |
| `PerformanceMetrics` | `TotalPnLUSD`, `TotalPnLPct`, `SharpeRatio`, `WinRate`, `TotalTrades`, `WinningTrades`, `LosingTrades`, `AvgWinUSD`, `AvgLossUSD`, `MaxDrawdownPct`, `CurrentDrawdownPct`, `UpdatedAt` | Derived from trade history / execution outcomes (Primary data will shift to DB `trades` & analytics tables). |

**Key Flows.**

- `RunTradingLoop` schedules decision cycles, enforces Sharpe-based pauses, and coordinates execution.  
- `buildExecutorContext` fetches Primary data (`exchange.Provider`, `market.Provider`), computes derived metrics (`UnrealizedPnLPct`, guard toggles), and feeds `executor.Context`.  
- `selectCandidates` ranks assets by absolute 1h move, applying liquidity guard threshold (Derived).  
- `ExecuteDecision` translates `Decision` into `exchange.Order`, computes size/price strings (Derived), and attaches optional SL/TP via provider extensions.  
- `SyncTraderPositions` updates `ResourceAllocation` from Primary exchange data, ultimately destined for DB/Redis persistence.

### 2.6 `pkg/journal`

**Purpose.** Persist decision-cycle artefacts to disk (or future object storage) for auditability.

| Type | Field | Description | Provenance / Formula |
|------|-------|-------------|----------------------|
| `CycleRecord` | `Timestamp` | Cycle completion time. | Derived (`time.Now()` unless provided) |
| | `TraderID` | Trader identifier. | Primary Runtime (manager) |
| | `CycleNumber` | Auto-increment per writer. | Derived counter |
| | `PromptDigest` | SHA-256 digest of prompt. | Derived via `llm.DigestString` |
| | `CoTTrace` | Chain-of-thought trace if available. | Primary Runtime (LLM response) |
| | `DecisionsJSON` | JSON serialisation of decisions. | Derived (`json.Marshal`) |
| | `Account` | Map snapshot (equity, balances). | Derived from `executor.Context.Account` (Primary exchange data) |
| | `Positions` | Slice of position maps. | Derived from `executor.Context.Positions` |
| | `Candidates` | Candidate symbol list. | Derived from selection heuristics |
| | `MarketDigest` | Reduced market snapshot (price/changes/funding). | Derived from `executor.Context.MarketDataMap` |
| | `Actions` | Execution outcomes (result/error). | Derived (manager execution) |
| | `Success` | Cycle status flag. | Derived (all actions success & no decision error) |
| | `ErrorMessage` | Failure cause. | Derived (error string) |
| | `Extra` | Free-form metadata. | Primary Runtime |
| `Writer` | `dir`, `seq`, `nowFn` | Journal writer settings. | `dir`: Primary Config (trader); `seq`: Derived counter |

---

## 3. Cross-Package Relationships

- **Manager ↔ Exchange.** `Manager` relies on `exchange.Provider` for account snapshots, live positions, leverage updates, and order execution. Exchange responses are treated as Primary data and optionally persisted to Postgres tables (`positions`, `trades`) and Redis caches (`nof0:positions:{model_id}`) by downstream services.
- **Manager ↔ Market.** `Manager` queries `market.Provider` for both `Snapshot` (prices, funding, OI) and `Asset` metadata. These feed executor risk checks and candidate selection. Market data should be cached in Redis (`nof0:price:latest:*`, `nof0:market:snapshot:{symbol}`) to avoid overloading external APIs.
- **Manager ↔ Executor.** `Manager.buildExecutorContext` converts Primary exchange/market data into `executor.Context`, adds guardrail thresholds (Derived), and calls `Executor.GetFullDecision`. `Executor.UpdatePerformance` receives `PerformanceView` derived from `PerformanceMetrics`.
- **Executor ↔ LLM.** `Executor` renders prompts, pushes `ChatRequest` to `llm.Client`, and decodes structured responses using `ChatStructured`. Model selection (`modelAlias`) and prompt hashing ensure auditability.
- **Manager ↔ Journal.** After each cycle, `Manager.writeJournalRecord` serialises consolidated state into `journal.CycleRecord`. Journals serve as the source of truth for investigations until DB ingestion is live.
- **Future DB/Redis Alignment.**  
  - Exchange data (positions, account state) → Postgres (`positions`, `account_equity_snapshots`) with Redis mirrors (`nof0:positions:{model_id}`).  
  - Market snapshots → Redis (`nof0:price:latest`, `nof0:market:snapshot`).  
  - Decisions & journal entries → object storage / `journal` table once introduced.  
  - Performance metrics → derived analytics tables (`model_analytics`).

---

## 4. Derived Metric Formulas & Guards (Quick Reference)

- **Position Unrealised PnL %**: `100 * (MarkPrice - EntryPrice) / EntryPrice`.
- **Account Margin Usage %**: `100 * MarginUsed / TotalEquity`.
- **Account PnL %**: `100 * TotalPnL / TotalEquity`.
- **Risk USD** (if absent): `PositionSizeUSD / Leverage`.
- **Liquidity Check**: `Snapshot.OpenInterest.Latest * Snapshot.Price.Last >= LiquidityThresholdUSD`.
- **BTC/ETH Value Band Guard**: `position_notional / equity` must fall within `[BTCETHMinEquityMultiple, BTCETHMaxEquityMultiple]`.
- **Alt Value Band Guard**: same formula with alt thresholds.
- **Cooldown Guard**: disallow re-entry until `last_close + CooldownAfterClose`.
- **Sharpe Pause**: if `PerformanceMetrics.SharpeRatio < SharpePauseThreshold`, pause trader for `PauseDurationOnBreach`.

---

## 5. Implementation Notes & Gaps

- `executor.PositionInfo.MarginUsed` is defined but currently populated as zero; manager should fill it using `qty * entryPrice / leverage` to tighten margin guard calculations.
- Liquidity filtering in `selectCandidates` only applies when Open Interest is available. For spot-only providers, guard falls back to raw change ranking.
- Journaling currently writes to the local filesystem. Introduce rotation and offloading once concurrency increases.
- Sample configs route both traders through `hyperliquid_testnet`; production rollouts must override `exchange_provider` / `market_provider` and inject production credentials via `${HYPERLIQUID_*}`.
- `PerformanceMetrics` is updated opportunistically after decision execution; integrating realised trade data from Postgres will provide authoritative metrics.
- `LLM` auto-routing (`zenmux/auto`) remains behind a fallback due to upstream instability; revisit after Zenmux bugfix (tracked for post-2025-12-01).

This reference should remain the single source of truth when extending the engine layer. Update it whenever new fields are added to the structs documented above or when data provenance changes (e.g., when moving from JSON loader to Postgres/Redis backends).
