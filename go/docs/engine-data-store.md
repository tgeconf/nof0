# Engine Layer Data Store Integration Plan

This document provides a comprehensive, module-by-module analysis of where database (Postgres) and cache (Redis) operations should be integrated into the `pkg/` layer. It identifies specific integration points for both **write operations** (primary focus) and **read operations** to ensure proper data persistence, caching, and retrieval across the engine layer.

**Reference Documents:**
- `docs/engine.md` - Engine layer data structure and flow reference
- `internal/model/` - Postgres table repository operations
- `internal/cache/keys.go` - Redis cache key structure definitions

---

## Table of Contents

1. [Overview](#overview)
2. [Database Schema Reference](#database-schema-reference)
3. [Redis Cache Structure Reference](#redis-cache-structure-reference)
4. [Module-by-Module Integration Analysis](#module-by-module-integration-analysis)
   - [4.1 pkg/manager](#41-pkgmanager)
   - [4.2 pkg/executor](#42-pkgexecutor)
   - [4.3 pkg/exchange](#43-pkgexchange)
   - [4.4 pkg/market](#44-pkgmarket)
   - [4.5 pkg/journal](#45-pkgjournal)
   - [4.6 pkg/llm](#46-pkgllm)
5. [Implementation Priority](#implementation-priority)
6. [Data Flow Diagrams](#data-flow-diagrams)

---

## Overview

The engine layer (`pkg/`) currently operates primarily in-memory with limited persistence to the filesystem (journal files). To enable production-grade operations, analytics, and state recovery, we need to integrate:

1. **Postgres writes** via `internal/model/*` - Persistent storage for positions, trades, account snapshots, decisions, and analytics
2. **Redis writes** via `internal/cache/keys.go` - Fast caching for real-time queries, market data, and computed metrics
3. **Postgres reads** - Load historical state for recovery, analytics computation, and context enrichment
4. **Redis reads** - Read cached market data, positions, and analytics to reduce database load

**Key Principles:**
- **Write operations** take priority: capture all critical state changes
- **Cache-aside pattern**: write to DB first, then update cache
- **Idempotency**: use guard keys and deduplication where appropriate
- **Non-blocking**: cache failures should not block core trading logic

---

## Database Schema Reference

Based on `internal/model/`, the following tables are available:

| Table | Purpose | Key Fields |
|-------|---------|------------|
| `positions` | Open and closed positions | `model_id`, `symbol`, `status`, `entry_price`, `quantity`, `entry_time_ms` |
| `trades` | Completed trades (entry + exit) | `model_id`, `symbol`, `entry_price`, `exit_price`, `realized_net_pnl` |
| `account_equity_snapshots` | Periodic account equity snapshots | `model_id`, `ts_ms`, `dollar_equity`, `realized_pnl`, `sharpe_ratio` |
| `models` | Trader/model registry | `id`, `name`, `exchange_provider`, `config` |
| `decision_cycles` | LLM decision cycle records | `model_id`, `cycle_ts_ms`, `prompt_digest`, `decisions_json` |
| `conversations` | LLM conversation history | `model_id`, `conversation_id`, `started_at` |
| `conversation_messages` | Individual messages in conversations | `conversation_id`, `role`, `content`, `tokens` |
| `market_assets` | Static asset metadata | `provider`, `symbol`, `base`, `quote`, `precision` |
| `market_asset_ctx` | Volatile market context | `provider`, `symbol`, `mark_price`, `funding_rate`, `oi` |
| `price_latest` | Latest price ticks | `provider`, `symbol`, `price`, `ts_ms` |
| `price_ticks` | Historical price ticks | `provider`, `symbol`, `price`, `ts_ms` |
| `trader_state` | Trader runtime state | `trader_id`, `state`, `last_decision_at`, `metadata` |
| `model_analytics` | Computed analytics per model | `model_id`, `metric_name`, `metric_value`, `computed_at` |

---

## Redis Cache Structure Reference

Based on `internal/cache/keys.go`, the following cache keys are defined:

### Price & Market Keys
- `nof0:price:latest:{symbol}` - Latest price for a symbol (TTL: short ~10s)
- `nof0:price:latest:{provider}:{symbol}` - Provider-scoped latest price
- `nof0:crypto_prices` - Aggregated prices map (TTL: short)
- `nof0:market:asset:{provider}:{symbol}` - Static asset metadata (TTL: long ~300s)
- `nof0:market:ctx:{provider}:{symbol}` - Volatile market context: funding, OI (TTL: medium ~60s)

### Position Keys
- `nof0:positions:{model_id}` - Hash of open positions for a model (TTL: ~30s)
- `nof0:lock:positions:{model_id}` - Recompute lock (TTL: ~5s)

### Trade Keys
- `nof0:trades:recent:{model_id}` - Recent trades list (TTL: medium)
- `nof0:trades:stream` - Redis Stream for trade fan-out (persistent)
- `nof0:ingest:trade:{trade_id}` - Idempotency guard (TTL: 24h)

### Analytics & Leaderboard Keys
- `nof0:leaderboard` - ZSet for model ranking by PnL
- `nof0:leaderboard:cache` - Pre-rendered leaderboard payload (TTL: medium)
- `nof0:since_inception:{model_id}` - Since-inception metrics (TTL: long)
- `nof0:analytics:{model_id}` - Analytics payload (TTL: ~600s)
- `nof0:analytics:all` - Aggregated analytics (TTL: long)

### Decision & Conversation Keys
- `nof0:decision:last:{model_id}` - Last decision summary (TTL: medium)
- `nof0:conversations:{model_id}` - Conversation list (TTL: long)

### Trader State Keys
- `nof0:trader:{trader_id}:state` - Cached trader state (TTL: medium)
- `nof0:sim:orders:{trader_id}` - Simulator orders (TTL: medium)
- `nof0:sim:balances:{trader_id}` - Simulator balances (TTL: medium)

---

## Module-by-Module Integration Analysis

### 4.1 pkg/manager

**File Locations:** `pkg/manager/manager.go`, `pkg/manager/trader.go`

**Responsibilities:**
- Orchestrates virtual traders
- Executes decisions via exchange providers
- Syncs account state and positions
- Tracks performance metrics
- Coordinates decision cycles

#### 4.1.1 Database Write Operations

| Location | Operation | Table | Description | Implementation Notes |
|----------|-----------|-------|-------------|---------------------|
| `manager.go:ExecuteDecision` | **INSERT/UPDATE** | `positions` | Create new position record when order is placed | After successful `PlaceOrder`, insert position with `status='open'`, `entry_oid`, `entry_time_ms`, `symbol`, `side`, `quantity`, `entry_price`, `leverage` |
| `manager.go:ExecuteDecision` | **UPDATE** | `positions` | Update position when filled | After order fill confirmation, update `wait_for_fill=false`, capture actual `entry_price` from fill data |
| `manager.go:ExecuteDecision` | **UPDATE** | `positions` | Attach SL/TP order IDs | After `SetStopLoss`/`SetTakeProfit`, update `sl_oid`, `tp_oid` in position record |
| `manager.go:ClosePosition` | **UPDATE** | `positions` | Mark position as closed | Update `status='closed'`, `exit_time_ms`, `closed_pnl`, `commission` |
| `manager.go:ClosePosition` | **INSERT** | `trades` | Create trade record on close | Insert complete trade record with entry/exit details, `realized_gross_pnl`, `realized_net_pnl`, `total_commission_dollars` |
| `manager.go:SyncTraderPositions` | **UPDATE** | `positions` | Sync live position data | Update `current_price`, `unrealized_pnl`, `liquidation_price` for all open positions |
| `manager.go:SyncTraderPositions` | **INSERT** | `account_equity_snapshots` | Periodic equity snapshot | Insert snapshot with `ts_ms`, `dollar_equity`, `realized_pnl`, `total_unrealized_pnl`, `sharpe_ratio`, `cum_pnl_pct` |
| `manager.go:RegisterTrader` | **INSERT/UPDATE** | `models` | Register new trader | Insert model record with `id`, `name`, `exchange_provider`, `market_provider`, `config` JSON |
| `manager.go:UpdatePerformanceMetrics` | **INSERT/UPDATE** | `model_analytics` | Store computed metrics | Insert/update rows for `sharpe_ratio`, `win_rate`, `total_trades`, `max_drawdown_pct` as separate metric records |
| `manager.go:RunTradingLoop` | **UPDATE** | `trader_state` | Update trader state | After each decision cycle, update `last_decision_at`, `state`, `metadata` JSON (cooldowns, pause_until) |
| `manager.go:writeJournalRecord` | **INSERT** | `decision_cycles` | Persist decision cycle | Insert cycle record with `model_id`, `cycle_ts_ms`, `prompt_digest`, `decisions_json`, `account_snapshot`, `success`, `error_message` |

**Key Implementation Points:**
- **manager.go:254-299 (`RunTradingLoop`)**: After calling `t.Executor.GetFullDecision`, write decision cycle to `decision_cycles` table
- **manager.go:ExecuteDecision** (new method needed): Wrap order execution with position DB writes
- **manager.go:SyncTraderPositions** (new method needed): Periodic sync job to:
  1. Fetch positions from exchange via `GetPositions()`
  2. Update `positions` table with current prices/PnL
  3. Insert `account_equity_snapshots` row
  4. Update performance metrics in `model_analytics`

#### 4.1.2 Database Read Operations

| Location | Operation | Table | Description | Implementation Notes |
|----------|-----------|-------|-------------|---------------------|
| `manager.go:buildExecutorContext` | **SELECT** | `positions` | Load open positions for context | Query `positionsModel.ActiveByModels()` to enrich executor context with DB-confirmed positions |
| `manager.go:UpdatePerformanceMetrics` | **SELECT** | `trades` | Load recent trades for metrics | Query `tradesModel.RecentByModel()` to compute win rate, Sharpe ratio |
| `manager.go:RegisterTrader` | **SELECT** | `trader_state` | Restore trader state on restart | Load last known state, cooldowns, pause windows |
| `manager.go:selectCandidates` | **SELECT** | `positions` | Check for open positions before candidate selection | Verify no conflicting positions exist |

#### 4.1.3 Redis Write Operations

| Location | Operation | Key Pattern | Description | TTL | Implementation Notes |
|----------|-----------|-------------|-------------|-----|---------------------|
| `manager.go:ExecuteDecision` | **HSET** | `nof0:positions:{model_id}` | Cache position snapshot | ~30s | After DB write, update hash with `{symbol: position_json}` |
| `manager.go:ClosePosition` | **HDEL** | `nof0:positions:{model_id}` | Remove closed position from cache | - | Delete symbol field from hash |
| `manager.go:ClosePosition` | **XADD** | `nof0:trades:stream` | Publish trade to stream | persistent | Add trade event for downstream consumers |
| `manager.go:ClosePosition` | **LPUSH + LTRIM** | `nof0:trades:recent:{model_id}` | Append to recent trades | medium | Keep last 100 trades in list |
| `manager.go:UpdatePerformanceMetrics` | **SET** | `nof0:analytics:{model_id}` | Cache computed analytics | ~600s | Serialize full `PerformanceMetrics` JSON |
| `manager.go:UpdatePerformanceMetrics` | **ZADD** | `nof0:leaderboard` | Update leaderboard ranking | persistent | Score = `total_pnl_pct` |
| `manager.go:SyncTraderPositions` | **SET** | `nof0:since_inception:{model_id}` | Cache since-inception stats | long | Store cumulative PnL%, Sharpe, total trades |
| `manager.go:RunTradingLoop` | **SET** | `nof0:trader:{trader_id}:state` | Cache trader state | medium | Store `{state, last_decision_at, pause_until}` JSON |
| `manager.go:writeJournalRecord` | **SET** | `nof0:decision:last:{model_id}` | Cache last decision summary | medium | Store digest of latest decision for UI |

**Key Implementation Points:**
- Use **cache-aside pattern**: write to Postgres first, then update Redis
- Positions cache should be atomic: use Redis transaction (MULTI/EXEC) when updating multiple fields
- Trade stream enables real-time notifications and downstream processing
- Leaderboard ZADD should be idempotent (score is absolute value)

#### 4.1.4 Redis Read Operations

| Location | Operation | Key Pattern | Description | Implementation Notes |
|----------|-----------|-------------|-------------|---------------------|
| `manager.go:buildExecutorContext` | **HGETALL** | `nof0:positions:{model_id}` | Try cache before DB query | Fallback to DB if cache miss |
| `manager.go:UpdatePerformanceMetrics` | **GET** | `nof0:analytics:{model_id}` | Check cached analytics | Skip recompute if fresh (<5min) |
| `manager.go:selectCandidates` | **HGETALL** | `nof0:positions:{model_id}` | Fast position check | Verify symbol not in cache before ranking candidates |

---

### 4.2 pkg/executor

**File Locations:** `pkg/executor/executor.go`, `pkg/executor/context.go`, `pkg/executor/validator.go`

**Responsibilities:**
- Renders prompts from context
- Calls LLM for structured decisions
- Validates decisions against guardrails
- Tracks validation failures

#### 4.2.1 Database Write Operations

| Location | Operation | Table | Description | Implementation Notes |
|----------|-----------|-------|-------------|---------------------|
| `executor.go:GetFullDecision` | **INSERT** | `conversations` | Start new conversation | Create conversation record with `model_id`, `started_at` |
| `executor.go:GetFullDecision` | **INSERT** | `conversation_messages` | Log prompt message | Insert message with `role='system'`, `content=promptStr`, `tokens=prompt_tokens` |
| `executor.go:GetFullDecision` | **INSERT** | `conversation_messages` | Log LLM response | Insert message with `role='assistant'`, `content=decisions_json`, `tokens=completion_tokens` |

**Key Implementation Points:**
- **executor.go:100-136 (`GetFullDecision`)**: After calling `llm.ChatStructured`, write conversation turn to `conversations` and `conversation_messages`
- Conversation tracking enables prompt debugging and LLM cost analysis
- Store token counts from `Usage` fields in response

#### 4.2.2 Database Read Operations

| Location | Operation | Table | Description | Implementation Notes |
|----------|-----------|-------|-------------|---------------------|
| `executor.go:UpdatePerformance` | **SELECT** | `model_analytics` | Load cached performance metrics | Alternative to in-memory performance view |

**Note:** Executor primarily operates on in-memory context. DB reads are minimal in this layer.

#### 4.2.3 Redis Write Operations

| Location | Operation | Key Pattern | Description | TTL | Implementation Notes |
|----------|-----------|-------------|-------------|-----|---------------------|
| `executor.go:GetFullDecision` | **SET** | `nof0:decision:last:{model_id}` | Cache decision summary | medium | Store `{symbol, action, confidence, timestamp}` JSON after validation passes |

**Key Implementation Points:**
- Only cache validated decisions (after `ValidateDecisions` succeeds)
- Decision cache enables fast UI updates without querying full decision_cycles table

#### 4.2.4 Redis Read Operations

| Location | Operation | Key Pattern | Description | Implementation Notes |
|----------|-----------|-------------|-------------|---------------------|
| `executor.go:GetFullDecision` | **GET** | `nof0:conversations:{model_id}` | Load recent conversation context | Optional: retrieve last N messages for context continuity |

---

### 4.3 pkg/exchange

**File Locations:** `pkg/exchange/hyperliquid/provider.go`, `pkg/exchange/hyperliquid/order.go`, `pkg/exchange/hyperliquid/position.go`, `pkg/exchange/hyperliquid/account.go`

**Responsibilities:**
- Place/cancel orders on exchange
- Fetch positions and account state
- Manage leverage and risk parameters
- Execute IOC market orders

#### 4.3.1 Database Write Operations

**Note:** Exchange layer should remain stateless and not write to DB directly. The **manager layer** is responsible for persisting exchange responses. However, exchange providers may emit **events** or **callbacks** that manager consumes.

| Location | Operation | Table | Description | Implementation Notes |
|----------|-----------|-------|-------------|---------------------|
| N/A | - | - | Exchange layer does not write to DB | Manager layer handles all persistence |

**Design Principle:** Keep exchange providers pure and side-effect-free. Manager orchestrates all state changes.

#### 4.3.2 Database Read Operations

| Location | Operation | Table | Description | Implementation Notes |
|----------|-----------|-------|-------------|---------------------|
| N/A | - | - | Exchange layer reads from live exchange only | No DB reads in this layer |

#### 4.3.3 Redis Write Operations

**Note:** Similar to DB writes, exchange layer should not cache directly. Manager handles caching after receiving exchange responses.

| Location | Operation | Key Pattern | Description | Implementation Notes |
|----------|-----------|-------------|-------------|---------------------|
| N/A | - | - | Exchange layer does not write to cache | Manager layer handles caching |

#### 4.3.4 Redis Read Operations

| Location | Operation | Key Pattern | Description | Implementation Notes |
|----------|-----------|-------------|-------------|---------------------|
| N/A | - | - | Exchange layer queries live exchange API | No cache reads |

**Summary:** Exchange layer is a pure adapter. All data persistence and caching happens in the **manager layer** after consuming exchange responses.

---

### 4.4 pkg/market

**File Locations:** `pkg/market/provider.go`, `pkg/market/exchanges/hyperliquid/provider.go`, `pkg/market/exchanges/hyperliquid/data.go`, `pkg/market/indicators/indicators.go`

**Responsibilities:**
- Fetch market snapshots (price, funding, OI)
- Compute technical indicators (EMA, MACD, RSI, ATR)
- List available assets with metadata

#### 4.4.1 Database Write Operations

| Location | Operation | Table | Description | Implementation Notes |
|----------|-----------|-------|-------------|---------------------|
| `hyperliquid/provider.go:ListAssets` | **INSERT/UPDATE** | `market_assets` | Persist static asset metadata | Upsert on `(provider, symbol)` with `base`, `quote`, `precision`, `is_active`, `max_leverage`, `only_isolated` |
| `hyperliquid/provider.go:Snapshot` | **INSERT/UPDATE** | `market_asset_ctx` | Update volatile market context | Upsert on `(provider, symbol, ts_ms)` with `mark_price`, `funding_rate`, `open_interest`, `volume_24h` |
| `hyperliquid/data.go:fetchKlines` | **INSERT** | `price_ticks` | Store historical price ticks | Insert OHLCV candles for indicator computation and backtesting |
| `hyperliquid/provider.go:Snapshot` | **UPDATE** | `price_latest` | Update latest price | Upsert on `(provider, symbol)` with `price`, `ts_ms` |

**Key Implementation Points:**
- **provider.go:Snapshot**: After computing indicators, write snapshot components to DB:
  1. `price_latest` - Latest mark price
  2. `market_asset_ctx` - Funding, OI, 1h/4h changes
- **provider.go:ListAssets**: Called on startup or refresh - cache results in both DB and Redis
- Use **bulk inserts** for price_ticks to minimize write overhead

#### 4.4.2 Database Read Operations

| Location | Operation | Table | Description | Implementation Notes |
|----------|-----------|-------|-------------|---------------------|
| `indicators/indicators.go:ComputeIndicators` | **SELECT** | `price_ticks` | Load historical candles for indicators | Query last 200+ candles ordered by `ts_ms` |
| `hyperliquid/provider.go:Snapshot` | **SELECT** | `market_asset_ctx` | Check last snapshot timestamp | Avoid redundant API calls if cached context is fresh (<30s) |

#### 4.4.3 Redis Write Operations

| Location | Operation | Key Pattern | Description | TTL | Implementation Notes |
|----------|-----------|-------------|-------------|-----|---------------------|
| `hyperliquid/provider.go:Snapshot` | **SET** | `nof0:price:latest:{provider}:{symbol}` | Cache latest price | short (~10s) | Store simple `{price, ts_ms}` JSON |
| `hyperliquid/provider.go:Snapshot` | **HSET** | `nof0:crypto_prices` | Add to aggregated prices | short | Field = `{provider}:{symbol}`, Value = price |
| `hyperliquid/provider.go:Snapshot` | **SET** | `nof0:market:ctx:{provider}:{symbol}` | Cache market context | medium (~60s) | Store `{funding, oi, change_1h, change_4h, indicators}` JSON |
| `hyperliquid/provider.go:ListAssets` | **SET** | `nof0:market:asset:{provider}:{symbol}` | Cache static metadata | long (~300s) | Store `{max_leverage, precision, is_active}` JSON |

**Key Implementation Points:**
- **Write-through cache**: Update Redis immediately after fetching from exchange
- Use **pipelining** to batch multiple SET operations when caching multiple symbols
- `crypto_prices` hash enables atomic reads of all latest prices
- Consider using **Redis Streams** for real-time price updates to subscribers

#### 4.4.4 Redis Read Operations

| Location | Operation | Key Pattern | Description | Implementation Notes |
|----------|-----------|-------------|-------------|---------------------|
| `hyperliquid/provider.go:Snapshot` | **GET** | `nof0:market:ctx:{provider}:{symbol}` | Try cache before API call | Return cached snapshot if fresh (<30s) |
| `hyperliquid/provider.go:ListAssets` | **MGET** | `nof0:market:asset:{provider}:{symbol}` | Batch read asset metadata | Reduce DB queries on startup |
| `indicators/indicators.go:ComputeIndicators` | **GET** | `nof0:market:ctx:{provider}:{symbol}` | Check for precomputed indicators | Skip recomputation if cached |

**Caching Strategy:**
- **L1 Cache (Redis)**: Short-lived, fast reads for real-time queries
- **L2 Cache (Postgres)**: Persistent storage for historical analysis
- Cache invalidation: TTL-based (short for prices, medium for indicators, long for static metadata)

---

### 4.5 pkg/journal

**File Locations:** `pkg/journal/journal.go`

**Responsibilities:**
- Persist decision cycle records to disk
- Provide audit trail for LLM decisions
- Enable post-mortem analysis

#### 4.5.1 Database Write Operations

| Location | Operation | Table | Description | Implementation Notes |
|----------|-----------|-------|-------------|---------------------|
| `journal.go:WriteCycle` | **INSERT** | `decision_cycles` | Mirror journal to DB | Insert cycle record with `model_id`, `cycle_ts_ms`, `prompt_digest`, `decisions_json`, `cot_trace`, `account_snapshot`, `success`, `error_message` |

**Key Implementation Points:**
- **journal.go:46-65 (`WriteCycle`)**: After writing to filesystem, also insert to `decision_cycles` table
- **Dual persistence**: Keep filesystem journal for debugging + DB for querying/analytics
- Consider **async write** to DB to avoid blocking journal writer
- Use **transaction** to ensure atomicity with related `conversation_messages` inserts

#### 4.5.2 Database Read Operations

| Location | Operation | Table | Description | Implementation Notes |
|----------|-----------|-------|-------------|---------------------|
| N/A | - | - | Journal writer does not read | Read operations happen in analytics/API layer |

#### 4.5.3 Redis Write Operations

| Location | Operation | Key Pattern | Description | TTL | Implementation Notes |
|----------|-----------|-------------|-------------|-----|---------------------|
| `journal.go:WriteCycle` | **SET** | `nof0:decision:last:{model_id}` | Cache latest cycle summary | medium | Store `{cycle_number, timestamp, success, symbol, action}` JSON |

**Key Implementation Points:**
- Cache only successful cycles to decision:last key
- Include error message in cache for failed cycles (for UI alerting)

#### 4.5.4 Redis Read Operations

| Location | Operation | Key Pattern | Description | Implementation Notes |
|----------|-----------|-------------|-------------|---------------------|
| N/A | - | - | No cache reads in journal layer | Consumers query cache directly |

---

### 4.6 pkg/llm

**File Locations:** `pkg/llm/client.go`, `pkg/llm/structured.go`, `pkg/llm/logger.go`

**Responsibilities:**
- Wrap LLM API calls (Zenmux/OpenAI)
- Handle retries and timeouts
- Log requests/responses
- Parse structured outputs

#### 4.6.1 Database Write Operations

| Location | Operation | Table | Description | Implementation Notes |
|----------|-----------|-------|-------------|---------------------|
| `client.go:Chat` | **INSERT** | `conversation_messages` | Log LLM request | Insert message with `role`, `content`, `tokens` (if executor provides conversation_id) |
| `client.go:Chat` | **INSERT** | `conversation_messages` | Log LLM response | Insert response message |

**Note:** LLM layer is typically stateless. Conversation persistence should be handled by the **executor layer** which has trader context.

#### 4.6.2 Database Read Operations

| Location | Operation | Table | Description | Implementation Notes |
|----------|-----------|-------|-------------|---------------------|
| N/A | - | - | LLM client does not read from DB | No context retrieval in this layer |

#### 4.6.3 Redis Write Operations

| Location | Operation | Key Pattern | Description | TTL | Implementation Notes |
|----------|-----------|-------------|-------------|-----|---------------------|
| `structured.go:ChatStructured` | **SET** | `nof0:llm:response:{digest}` | Cache structured responses (future) | long | Optional: cache LLM responses by prompt digest to avoid redundant calls |

**Future Enhancement:**
- Implement **response caching** for identical prompts (by prompt digest)
- Requires careful cache invalidation when context changes significantly
- Not critical for MVP but valuable for cost reduction

#### 4.6.4 Redis Read Operations

| Location | Operation | Key Pattern | Description | Implementation Notes |
|----------|-----------|-------------|-------------|---------------------|
| `structured.go:ChatStructured` | **GET** | `nof0:llm:response:{digest}` | Check cached response | Return cached result if prompt digest matches |

**Summary:** LLM layer remains mostly stateless. Persistence is delegated to executor/manager layers.

---

## Implementation Priority

### Phase 1: Critical Writes (P0 - Week 1-2)

**Goal:** Capture all trading state for production operations and recovery

1. **Manager - Position Lifecycle**
   - [ ] `positions` table: INSERT on entry, UPDATE on fill/close
   - [ ] `trades` table: INSERT on position close
   - [ ] Redis `positions:{model_id}` hash: HSET on entry, HDEL on close

2. **Manager - Account Snapshots**
   - [ ] `account_equity_snapshots`: INSERT periodic snapshots (every 5min)
   - [ ] Redis `analytics:{model_id}`: SET after snapshot

3. **Manager - Decision Cycles**
   - [ ] `decision_cycles`: INSERT after each LLM call
   - [ ] Redis `decision:last:{model_id}`: SET after validation

**Success Criteria:**
- All positions persisted to DB and cached in Redis
- Account equity tracked every 5 minutes
- Decision history queryable from DB

---

### Phase 2: Market Data Caching (P1 - Week 2-3)

**Goal:** Reduce external API calls and improve response times

1. **Market - Price Caching**
   - [ ] Redis `price:latest:{provider}:{symbol}`: SET after fetch
   - [ ] Redis `crypto_prices` hash: HSET for aggregated view
   - [ ] `price_latest` table: INSERT/UPDATE on fetch

2. **Market - Asset Metadata**
   - [ ] `market_assets`: INSERT/UPDATE on ListAssets
   - [ ] Redis `market:asset:{provider}:{symbol}`: SET with long TTL

3. **Market - Market Context**
   - [ ] `market_asset_ctx`: INSERT periodic updates
   - [ ] Redis `market:ctx:{provider}:{symbol}`: SET with medium TTL

**Success Criteria:**
- 90%+ cache hit rate for market snapshots
- Price data queryable for backtesting
- Asset metadata loaded from cache on startup

---

### Phase 3: Analytics & Performance (P1 - Week 3-4)

**Goal:** Enable real-time analytics and leaderboard

1. **Manager - Performance Metrics**
   - [ ] `model_analytics`: INSERT/UPDATE computed metrics
   - [ ] Redis `analytics:{model_id}`: SET full payload
   - [ ] Redis `since_inception:{model_id}`: SET cumulative stats
   - [ ] Redis `leaderboard` ZSet: ZADD on metric update

2. **Manager - Trader State**
   - [ ] `trader_state`: UPDATE on state changes
   - [ ] Redis `trader:{trader_id}:state`: SET with medium TTL

**Success Criteria:**
- Leaderboard updates within 1 minute of trades
- Analytics API serves from cache (< 50ms p99)
- Trader state recoverable after restart

---

### Phase 4: Conversation & Debugging (P2 - Week 4-5)

**Goal:** Enable LLM debugging and cost tracking

1. **Executor - Conversation Tracking**
   - [ ] `conversations`: INSERT on decision start
   - [ ] `conversation_messages`: INSERT prompt + response
   - [ ] Redis `conversations:{model_id}`: cache recent conversation IDs

2. **Journal - DB Mirroring**
   - [ ] `decision_cycles`: INSERT from journal writer
   - [ ] Link decision_cycles to conversations table

**Success Criteria:**
- Full prompt/response history queryable
- Token usage trackable per model
- Decision cycles linked to conversations

---

### Phase 5: Advanced Features (P3 - Week 5+)

**Goal:** Optimize and enhance

1. **Trade Stream**
   - [ ] Redis `trades:stream`: XADD on trade close
   - [ ] Consumer group for downstream processing

2. **LLM Response Caching**
   - [ ] Redis `llm:response:{digest}`: cache by prompt digest
   - [ ] Implement cache warming for common prompts

3. **Price Tick Bulk Inserts**
   - [ ] `price_ticks`: batch insert historical data
   - [ ] Background job to backfill from exchange

**Success Criteria:**
- Real-time trade notifications working
- LLM cache hit rate > 20% for repeated patterns
- Historical price data available for indicators


## Data Flow Diagrams

### Position Lifecycle Data Flow

#### A. Opening a Position

```
Step 1: Manager.RunTradingLoop
  |
  +--> Step 2: Executor.GetFullDecision
         |
         +--> Step 3: LLM.ChatStructured
                |
                +--> Returns: Decision {symbol, action, size, leverage...}
  |
  +--> Step 4: Manager.ExecuteDecision(decision)
         |
         +--> 4.1: Exchange.PlaceOrder(order)
         |      |
         |      +--> Returns: OrderResponse {oid, status}
         |
         +--> 4.2: DB Write - INSERT INTO positions
         |          Fields: model_id, symbol, status='open', entry_oid,
         |                  entry_price, quantity, leverage, entry_time_ms,
         |                  wait_for_fill=true
         |
         +--> 4.3: Redis Write - HSET positions:{model_id}
         |          Field: {symbol}
         |          Value: {position_json with all fields}
         |          TTL: 30s on hash
         |
         +--> 4.4: Exchange.SetStopLoss(symbol, stop_price)
         |      |
         |      +--> Returns: sl_oid
         |
         +--> 4.5: Exchange.SetTakeProfit(symbol, tp_price)
         |      |
         |      +--> Returns: tp_oid
         |
         +--> 4.6: DB Write - UPDATE positions
         |          SET sl_oid={sl_oid}, tp_oid={tp_oid}
         |          WHERE id={position_id}
         |
         +--> 4.7: Redis Write - HSET positions:{model_id}
                    Field: {symbol}
                    Value: {updated_position_json with sl/tp oids}
```

#### B. Closing a Position

```
Step 1: Manager.ClosePosition(trader, symbol)
  |
  +--> Step 2: Exchange.ClosePosition(symbol)
         |
         +--> Returns: FillResponse {exit_price, exit_time, commission...}
  |
  +--> Step 3: DB Write - UPDATE positions
  |              SET status='closed',
  |                  exit_time_ms={exit_time},
  |                  closed_pnl={calculated_pnl},
  |                  commission={total_commission}
  |              WHERE model_id={model_id} AND symbol={symbol}
  |
  +--> Step 4: DB Write - INSERT INTO trades
  |              Fields: model_id, symbol, side, trade_type,
  |                      entry_price, entry_ts_ms, entry_oid,
  |                      exit_price, exit_ts_ms, exit_oid,
  |                      quantity, leverage, confidence,
  |                      realized_gross_pnl, realized_net_pnl,
  |                      total_commission_dollars
  |
  +--> Step 5: Redis Write - HDEL positions:{model_id}
  |              Field: {symbol}
  |              (Remove from positions hash)
  |
  +--> Step 6: Redis Write - XADD trades:stream
  |              Message: {trade_event with full trade data}
  |              (Stream for downstream consumers)
  |
  +--> Step 7: Redis Write - LPUSH trades:recent:{model_id}
  |              Value: {trade_summary}
  |              Then: LTRIM to keep only last 100 trades
  |              TTL: medium (60s) on list
  |
  +--> Step 8: Update Performance Metrics
                (Triggers separate analytics flow)
```

---

### Market Data Flow

```
Step 1: Market.Provider.Snapshot(symbol)
  |
  +--> Step 2: Check Cache - Redis GET market:ctx:{provider}:{symbol}
         |
         +--> [Cache Hit] --> Return cached snapshot (skip to Step 8)
         |
         +--> [Cache Miss] --> Continue to Step 3
  |
  +--> Step 3: Exchange API Call
  |      |
  |      +--> Fetch: mark_price, funding_rate, open_interest,
  |                  volume_24h, recent_trades
  |      |
  |      +--> Returns: Raw market data
  |
  +--> Step 4: DB Read - SELECT FROM price_ticks
  |              WHERE provider={provider} AND symbol={symbol}
  |              ORDER BY ts_ms DESC
  |              LIMIT 200
  |              (Get historical candles for indicator computation)
  |
  +--> Step 5: Compute Indicators
  |      |
  |      +--> Calculate: EMA (20, 50, 200)
  |      +--> Calculate: MACD (12, 26, 9)
  |      +--> Calculate: RSI (7, 14)
  |      +--> Calculate: ATR (14)
  |      +--> Calculate: 1h/4h price changes
  |
  +--> Step 6: DB Write - INSERT/UPDATE price_latest
  |              ON CONFLICT (provider, symbol)
  |              DO UPDATE SET price={mark_price}, ts_ms={current_ts}
  |
  +--> Step 7: DB Write - INSERT/UPDATE market_asset_ctx
  |              ON CONFLICT (provider, symbol, ts_ms)
  |              DO UPDATE SET mark_price, funding_rate, open_interest,
  |                            volume_24h, change_1h, change_4h
  |
  +--> Step 8: Redis Write - SET price:latest:{provider}:{symbol}
  |              Value: {price, ts_ms}
  |              TTL: 10s (short)
  |
  +--> Step 9: Redis Write - HSET crypto_prices
  |              Field: {provider}:{symbol}
  |              Value: {price}
  |              TTL: 10s (short) on entire hash
  |
  +--> Step 10: Redis Write - SET market:ctx:{provider}:{symbol}
  |               Value: {full snapshot JSON with price, funding, OI,
  |                       indicators, changes}
  |               TTL: 60s (medium)
  |
  +--> Step 11: Return Snapshot object to caller
```

**Cache Strategy:**
- **First call**: API fetch + DB write + Redis cache (warm cache)
- **Subsequent calls within TTL**: Redis cache hit (no API/DB query)
- **After TTL expiry**: Repeat from Step 3

---

### Decision Cycle Flow

```
Step 1: Manager.RunTradingLoop (every 1 second)
  |
  +--> Check: trader.ShouldMakeDecision()
         |
         +--> [No] --> Skip this trader, continue loop
         |
         +--> [Yes] --> Continue to Step 2
  |
  +--> Step 2: Manager.buildExecutorContext(trader)
         |
         +--> 2.1: Load Positions
         |      |
         |      +--> Try Cache: Redis HGETALL positions:{model_id}
         |      |      |
         |      |      +--> [Hit] --> Parse position JSON from hash
         |      |      |
         |      |      +--> [Miss] --> Fallback to DB
         |      |
         |      +--> DB Read: SELECT FROM positions
         |      |              WHERE model_id={model_id} AND status='open'
         |      |
         |      +--> Returns: []PositionInfo
         |
         +--> 2.2: Fetch Market Data for Candidates
         |      |
         |      +--> For each candidate symbol:
         |      |      Market.Provider.Snapshot(symbol)
         |      |      --> [See Market Data Flow above]
         |      |
         |      +--> Returns: map[symbol]*Snapshot
         |
         +--> 2.3: Fetch Account State
         |      |
         |      +--> Exchange.GetAccountState()
         |      |      --> Returns: equity, margin_used, available_balance
         |      |
         |      +--> Calculate: margin_pct, equity_pct, position_count
         |
         +--> 2.4: Assemble Context
                |
                +--> Returns: executor.Context {
                       CurrentTime, RuntimeMinutes, CallCount,
                       Account, Positions, CandidateCoins,
                       MarketDataMap, OpenInterestMap,
                       MajorCoinLeverage, AltcoinLeverage,
                       MaxRiskPct, MaxPositionSizeUSD, ...guardrails
                     }
  |
  +--> Step 3: Executor.GetFullDecision(context)
         |
         +--> 3.1: Render Prompt
         |      |
         |      +--> Template Engine: substitute context data
         |      +--> Generate: prompt_text (5-10KB)
         |      +--> Calculate: prompt_digest (SHA256)
         |
         +--> 3.2: LLM Call
         |      |
         |      +--> LLM.ChatStructured(prompt, response_format)
         |      |      Model: gpt-5 or claude-sonnet-4.5
         |      |      Timeout: 60s
         |      |
         |      +--> Returns: Decision {
         |             symbol, action, position_size_usd,
         |             leverage, entry_price, stop_loss, take_profit,
         |             confidence, risk_usd, reasoning,
         |             invalidation_condition
         |           }
         |
         +--> 3.3: Validate Decision
         |      |
         |      +--> Check: confidence >= min_confidence
         |      +--> Check: position_count < max_positions
         |      +--> Check: margin_usage + new_margin <= max_margin_pct
         |      +--> Check: liquidity >= liquidity_threshold
         |      +--> Check: position_size within value bands
         |      +--> Check: symbol not in cooldown
         |      |
         |      +--> [Invalid] --> Return error, mark validation_failed
         |      |
         |      +--> [Valid] --> Continue
         |
         +--> 3.4: Return FullDecision
                |
                +--> Returns: {
                       UserPrompt, CoTTrace, Decisions, Timestamp
                     }
  |
  +--> Step 4: Manager.writeJournalRecord(cycle_data)
         |
         +--> 4.1: DB Write - INSERT INTO decision_cycles
         |          Fields: model_id, cycle_ts_ms, prompt_digest,
         |                  decisions_json, cot_trace,
         |                  account_snapshot, positions_snapshot,
         |                  candidates, market_digest,
         |                  success, error_message
         |
         +--> 4.2: Redis Write - SET decision:last:{model_id}
         |          Value: {cycle_number, timestamp, success,
         |                  symbol, action, confidence, error_msg}
         |          TTL: 60s (medium)
         |
         +--> 4.3: Journal.WriteCycle(record)
                |
                +--> Write: journal/{trader_id}/cycle_{ts}_{seq}.json
                +--> Also: INSERT INTO decision_cycles (duplicate for safety)
  |
  +--> Step 5: [If decision valid] Manager.ExecuteDecision(decision)
         |
         +--> --> [See Position Lifecycle Flow - Opening Position]
  |
  +--> Step 6: Update Trader State
         |
         +--> trader.RecordDecision(timestamp)
         |
         +--> DB Write: UPDATE trader_state
         |              SET last_decision_at={timestamp},
         |                  metadata={cooldowns, pause_until, ...}
         |              WHERE trader_id={trader_id}
         |
         +--> Redis Write: SET trader:{trader_id}:state
                    Value: {state, last_decision_at, pause_until}
                    TTL: 60s (medium)
```

**Cycle Timing:**
- Decision interval: 3 minutes (configurable per trader)
- LLM timeout: 60 seconds
- Market data TTL: 60 seconds (reused across cycles)
- Total cycle time: typically 5-15 seconds (including LLM call)

---

## Appendix: Code Integration Checklist

### For each write operation, ensure:
- [ ] DB write completes **before** cache update (cache-aside pattern)
- [ ] Error handling: log failures but don't block critical path
- [ ] Idempotency: use `INSERT ... ON CONFLICT` or guard keys where needed
- [ ] Transactions: group related writes (e.g., position + trade)
- [ ] Metrics: emit counters for writes (success/failure)

### For each read operation, ensure:
- [ ] Try cache first (Redis), fallback to DB
- [ ] Set appropriate cache TTLs based on data volatility
- [ ] Handle cache misses gracefully
- [ ] Consider read-through caching for hot paths
- [ ] Emit latency metrics for both cache and DB reads

### Testing checklist:
- [ ] Unit tests: verify correct table/key patterns
- [ ] Integration tests: verify DB+cache consistency
- [ ] Load tests: ensure writes don't block trading loop
- [ ] Failure tests: verify graceful degradation when DB/Redis down
- [ ] Recovery tests: verify state restoration after restart

---

**Document Version:** 1.0
**Last Updated:** 2025-11-04
**Maintainer:** Trading System Team
