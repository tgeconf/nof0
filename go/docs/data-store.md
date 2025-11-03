# Postgres & Redis Data Design

This document captures the authoritative storage design for the NOF0 trading platform. It combines the domain definitions surfaced in `internal/types`, runtime behaviour described in `docs/engine.md`, and the baseline schema proposed in `docs/data-architecture.md`. It is intended to guide the production build-out as we migrate from JSON loaders to persistent infrastructure.

## 1. Scope and Inputs

- **Configuration chain** (`etc/nof0.yaml`): maps each subsystem to its YAML file. TTL classes (`Short=10s`, `Medium=60s`, `Long=300s`) inform Redis expirations.
- **Runtime domains** (`internal/types`, `docs/engine.md`): positions, trades, analytics, conversations, market snapshots.
- **Existing migrations** (`migrations/001_domain.sql`, `002_refresh_helpers.sql`): seed Postgres tables and materialized views.

Postgres is the source of truth for historical and transactional data. Redis acts as a low-latency read layer and coordination space (locks, idempotency guards). Every Redis value must be reproducible from Postgres or upstream APIs.

---

## 2. Postgres Schema

### 2.1 Entity Overview

| Category | Table | Purpose | Primary Key | Relationship Notes |
|----------|-------|---------|-------------|---------------------|
| Reference | `models` | Catalog of arena participants. | `id` | Other tables carry `model_id` columns pointing at this logical key (checked in application). |
| Reference | `symbols` | Tradeable instruments (e.g., `BTC`). | `symbol` | `symbol` columns elsewhere mirror this value set; integrity validated in ingest. |
| Fact | `price_ticks` | Append-only price feed. | `id` (bigserial) | Rows include a `symbol` that must exist in `symbols`; no FK enforced. Indexed `(symbol, ts_ms DESC)`. |
| Fact | `price_latest` | Upserted latest price per symbol. | `symbol` | Maintained by ingest upsert, mirrors `symbols`. |
| Fact | `accounts` | Portfolio metadata per model. | `model_id` | Matches `models.id`; enforced by writers. |
| Fact | `account_equity_snapshots` | Time-series equity & PnL. | `id` (bigserial) | Contains `model_id`; consumer code uses it to join. |
| Fact | `positions` | Open positions. | `id` (text) | Stores `model_id` / `symbol` references without DB constraints. |
| Fact | `trades` | Closed trade executions. | `id` (text) | Same logical relationship as positions. |
| Aggregate | `model_analytics` | Denormalised analytics payloads. | `model_id` | One row per model; payload mirrors API schema. |
| Conversations | `conversations` | Conversation threads per model. | `id` (bigserial) | `model_id` column maps back to `models`. |
| Conversations | `conversation_messages` | Individual chat messages. | `id` (bigserial) | Stores `conversation_id`; writer ensures referential integrity. |
| Journaling | `decision_cycles` *(proposed)* | Persisted executor cycles. | `id` (bigserial) | Includes `model_id` for joinability; integrity handled by ingestion job. |

> **Note**: `decision_cycles` is not yet in the migrations; add it when journal persistence moves from filesystem to DB.

### 2.2 Detailed Schemas

#### `models`
```sql
id TEXT PRIMARY KEY,
display_name TEXT NOT NULL,
created_at TIMESTAMPTZ DEFAULT NOW()
```

#### `symbols`
```sql
symbol TEXT PRIMARY KEY,
quote_currency TEXT DEFAULT 'USD',
created_at TIMESTAMPTZ DEFAULT NOW()
```

#### `price_ticks`
```sql
id BIGSERIAL PRIMARY KEY,
symbol TEXT NOT NULL,
price DOUBLE PRECISION NOT NULL,
ts_ms BIGINT NOT NULL,
source TEXT DEFAULT 'hyperliquid',
ingested_at TIMESTAMPTZ DEFAULT NOW()
```
- Retention: keep 30 days in primary table; archive older rows to cold storage.
- Index: `CREATE INDEX idx_price_ticks_symbol_ts ON price_ticks(symbol, ts_ms DESC);`

#### `price_latest`
```sql
symbol TEXT PRIMARY KEY,
price DOUBLE PRECISION NOT NULL,
ts_ms BIGINT NOT NULL,
updated_at TIMESTAMPTZ DEFAULT NOW()
```
- Maintained by `INSERT ... ON CONFLICT (symbol) DO UPDATE`.

#### `accounts`
```sql
model_id TEXT PRIMARY KEY,
base_currency TEXT NOT NULL DEFAULT 'USD',
created_at TIMESTAMPTZ DEFAULT NOW(),
updated_at TIMESTAMPTZ DEFAULT NOW()
```

#### `account_equity_snapshots`
```sql
id BIGSERIAL PRIMARY KEY,
model_id TEXT NOT NULL,
ts_ms BIGINT NOT NULL,
equity_usd DOUBLE PRECISION NOT NULL,
realized_pnl DOUBLE PRECISION DEFAULT 0,
unrealized_pnl DOUBLE PRECISION DEFAULT 0,
run_id UUID DEFAULT gen_random_uuid(),
created_at TIMESTAMPTZ DEFAULT NOW()
```
- Index: `CREATE INDEX idx_equity_model_ts ON account_equity_snapshots(model_id, ts_ms DESC);`
- Optional uniqueness: `(model_id, ts_ms)` to prevent duplicates from replay.

#### `positions`
```sql
id TEXT PRIMARY KEY,
model_id TEXT NOT NULL,
symbol TEXT NOT NULL,
side TEXT NOT NULL CHECK (side IN ('long','short')),
entry_price DOUBLE PRECISION NOT NULL,
quantity DOUBLE PRECISION NOT NULL,
leverage DOUBLE PRECISION,
confidence DOUBLE PRECISION,
entry_ts_ms BIGINT NOT NULL,
commission DOUBLE PRECISION,
status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','closed')),
exit_ts_ms BIGINT,
exit_price DOUBLE PRECISION,
metadata JSONB DEFAULT '{}'::jsonb,
updated_at TIMESTAMPTZ DEFAULT NOW()
```
- Use partial index on `status='open'` to accelerate active position lookups.
- Mark-to-market values (current price, unrealized PnL, liquidation) should be computed during cache publish instead of persisted.

#### `trades`
```sql
id TEXT PRIMARY KEY,
model_id TEXT NOT NULL,
symbol TEXT NOT NULL,
side TEXT NOT NULL,
trade_type TEXT,
quantity DOUBLE PRECISION,
leverage DOUBLE PRECISION,
confidence DOUBLE PRECISION,
entry_price DOUBLE PRECISION,
entry_ts_ms BIGINT,
exit_price DOUBLE PRECISION,
exit_ts_ms BIGINT,
realized_gross_pnl DOUBLE PRECISION,
realized_net_pnl DOUBLE PRECISION,
total_commission_dollars DOUBLE PRECISION,
entry_oid BIGINT,
exit_oid BIGINT,
created_at TIMESTAMPTZ DEFAULT NOW()
```
- Index: `CREATE INDEX idx_trades_model_entry_ts ON trades(model_id, entry_ts_ms DESC);`
- For fast lookups by `exit_oid`, add `CREATE INDEX idx_trades_exit_oid ON trades(exit_oid);`

#### `model_analytics`
```sql
model_id TEXT PRIMARY KEY,
updated_at TIMESTAMPTZ NOT NULL,
payload JSONB NOT NULL,
server_time_ms BIGINT NOT NULL
```
- Optional `CHECK (jsonb_typeof(payload) = 'object')`.
- If Redis retains the full analytics payload, consider persisting only raw aggregates here and regenerating the final shape in cache jobs.

#### `conversations`
```sql
id BIGSERIAL PRIMARY KEY,
model_id TEXT NOT NULL,
created_at TIMESTAMPTZ DEFAULT NOW()
```

#### `conversation_messages`
```sql
id BIGSERIAL PRIMARY KEY,
conversation_id BIGINT NOT NULL,
role TEXT NOT NULL CHECK (role IN ('system','user','assistant')),
content TEXT NOT NULL,
ts_ms BIGINT,
metadata JSONB DEFAULT '{}'::jsonb
```
- Index: `CREATE INDEX idx_conv_msgs_conv_ts ON conversation_messages(conversation_id, ts_ms);`

#### `decision_cycles` *(proposed)*
```sql
id BIGSERIAL PRIMARY KEY,
model_id TEXT NOT NULL,
cycle_number INTEGER,
prompt_digest TEXT,
cot_trace TEXT,
decisions JSONB,
success BOOLEAN DEFAULT FALSE,
error_message TEXT,
executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```
- Index on `(model_id, executed_at DESC)` for audit queries. Rebuild account/position snapshots from canonical tables when auditing instead of persisting them here.

### 2.3 Materialized Views & Refresh

Existing views (see `migrations/002_refresh_helpers.sql`):

| View | Source | Purpose | Refresh Cadence |
|------|--------|---------|-----------------|
| `v_crypto_prices_latest` | `price_latest` | Aligns with `/api/crypto-prices`. | On every price upsert or via scheduled job (`refresh_views_nof0()`). |
| `v_leaderboard` | `account_equity_snapshots` + `models` | Supplies leaderboard snapshot. | After bulk analytics refresh (hourly). |
| `v_since_inception` | `account_equity_snapshots` | Feeds `/api/since-inception-values`. | Hourly or after nightly ingest. |

Add supporting indexes on view columns used in API filters (e.g., `model_id`).

### 2.4 Ingestion Pipelines

- **Prices**  
  1. Receive tick from exchange/ws feed.  
  2. `INSERT INTO price_ticks`.  
  3. `UPSERT price_latest`.  
  4. Publish to Redis keys (`nof0:price:latest:{symbol}`, `nof0:crypto_prices`).  
  5. Optionally `REFRESH MATERIALIZED VIEW CONCURRENTLY v_crypto_prices_latest`.

- **Trades**  
  1. Consume fill event.  
  2. Upsert `trades`.  
  3. Mark old positions `status='closed'`, open new ones if partial.  
  4. Update/insert `account_equity_snapshots` once positions reflect the latest state.  
  5. Recompute leaderboard metrics (prefer publishing to Redis; persist the aggregated result only when historical analytics must be retained).

- **Positions**  
  1. On position open/adjust close: upsert `positions`.  
  2. On full close: set `status='closed'`.  
  3. Compute mark-to-market metrics (current price, unrealized PnL, liquidation) in memory during cache publish rather than persisting them.  
  4. Push derived JSON to `nof0:positions:{model_id}` hash.

- **Analytics**  
  1. Batch job hydrates metrics from trades + positions.  
  2. Publish the computed payload to Redis (`nof0:analytics:{model_id}`) with a long TTL.  
  3. Optionally persist a trimmed JSON aggregate in `model_analytics` when historical snapshots are needed.

- **Conversations & Journals**  
  1. On new conversation message: insert into `conversation_messages`.  
  2. Update `nof0:conversations:{model_id}` list.  
  3. Decision loops write to `decision_cycles` (once table exists).

### 2.5 Maintenance & Retention

- **Vacuum / Analyze**: ensure autovacuum tuning for high-churn tables (`price_ticks`, `trades`).  
- **Partitioning (future)**: partition `price_ticks` and `account_equity_snapshots` by month for pruning.  
- **Archival**: move historical `decision_cycles` > 90 days to cold storage.  
- **Refresh Function**: use `SELECT refresh_views_nof0();` in cron or after ETL batches.  
- **Backup Strategy**: nightly logical dump + WAL archiving; ensure Redis caches can be rebuilt from Postgres if dropped.

---

## 3. Redis Keyspace Design

### 3.1 TTL Classes

| TTL Class (`etc/nof0.yaml`) | Duration | Example Use |
|-----------------------------|----------|-------------|
| `Short` | 10 seconds | Prices, open positions. |
| `Medium` | 60 seconds | Leaderboard slices, trades list. |
| `Long` | 300 seconds | Since-inception curves, analytics payloads. |

### 3.2 Key Patterns

| Key Pattern | Type | Value Schema | TTL | Source of Truth | Invalidation / Refresh |
|-------------|------|--------------|-----|-----------------|------------------------|
| `nof0:price:latest:{symbol}` | String JSON | `{"symbol":"BTC","price":111317.5,"timestamp":1761452335744}` | 10s | Postgres `price_latest` / live feed | Overwrite on every tick; falls back to DB when expired. |
| `nof0:crypto_prices` | String JSON | Map of symbol → `CryptoPrice`. | 10s | Aggregated from `price_latest`. | Rebuilt in same loop as per-symbol writes. |
| `nof0:positions:{model_id}` | Hash | Field = `symbol`, Value = position JSON (mirrors `internal/types.Position`). | 30s | Postgres `positions` (active rows). | Replace hash after sync; also cleared on position close. |
| `nof0:lock:positions:{model_id}` | String | `"1"` | 5s | Coordination only. | `SET NX PX 5000` before recompute. |
| `nof0:trades:recent:{model_id}` | List *(optional)* | JSON-encoded trade snapshots. | 60s | Postgres `trades` (latest N). | Maintain only if SQL query latency is insufficient; otherwise query DB directly. |
| `nof0:trades:stream` | Stream | Entry fields per trade (id, symbol, pnl). | 0 (no expiry) | Derived from ingest event. | Append-on-write for downstream consumers. |
| `nof0:ingest:trade:{trade_id}` | String | `"seen"` | 24h | Idempotency guard. | `SETNX` when processing trade; prevents double insert. |
| `nof0:leaderboard` | Sorted Set | Score = `return_pct` (or `equity`), member = `model_id`. | 60s | Derived from `v_leaderboard`. | Rewritten after leaderboard ETL. |
| `nof0:leaderboard:cache` | String JSON *(optional)* | Top-K leaderboard array. | 60s | Same as above. | Use only when API latency requires a pre-rendered payload. |
| `nof0:since_inception:{model_id}` | List | Ordered tuples `{timestamp,value}`. | 5m | `v_since_inception`. | Recomputed after snapshot refresh or nightly. |
| `nof0:analytics:{model_id}` | String JSON | `ModelAnalytics` payload. | 10m | `model_analytics.payload`. | Replace after analytics job. |
| `nof0:analytics:all` | String JSON *(optional)* | `AnalyticsResponse`. | 10m | Aggregated from per-model analytics. | Rebuild on demand when bulk response is needed. |
| `nof0:conversations:{model_id}` | List | Message JSON objects (role, content, ts). | 5m | `conversation_messages`. | Append on new message; trim to recency window. |
| `nof0:decision:last:{model_id}` | String JSON *(optional)* | Latest executor decision summary. | 60s | Runtime snapshots only; skip if `decision_cycles` persists audits. |

### 3.3 Patterns & Guardrails

- **Write-through**: whenever Postgres is updated, update Redis synchronously (best effort) to prevent stale reads.
- **Fail-safe TTLs**: short expirations ensure caches self-heal if invalidation fails.
- **Compression**: large payloads (`analytics`, `conversations`) can be stored using RESP3 `HELLO` compression if client supports it; otherwise rely on JSON.
- **Locking**: use the `lock` pattern with short expirations to avoid duplicate recomputations (e.g., positions, analytics).
- **Idempotency**: `nof0:ingest:trade:{trade_id}` prevents replaying the same trade ingestion event.

### 3.4 Cache Rebuild Procedures

1. **Cold Start**:  
   - Warm `price_latest` from Postgres then publish to per-symbol keys.  
   - Hydrate `leaderboard` & `analytics` via ETL job, then write caches.  
   - For conversations, replay latest N messages per model.

2. **Cache Drop**:  
   - Detect via Redis metrics; use background worker to repopulate keys by reading Postgres tables and materialized views.

3. **Config-driven TTLs**:  
   - Align all expirations with values defined in `etc/nof0.yaml` to maintain consistent cache invalidation strategy.

---

## 4. Consistency & Synchronisation

### 4.1 Event Ordering

| Event | Order of Operations |
|-------|---------------------|
| Trade fill | Update Postgres (`trades`, `positions`, `account_equity_snapshots`) → recompute derived aggregates → update Redis caches. |
| Position close | Mark position closed in Postgres → remove field from `nof0:positions:{model_id}` → push summary to `nof0:trades:recent:{model_id}`. |
| Analytics batch | Recompute stats → upsert `model_analytics` → refresh views → update Redis `analytics` keys. |
| Conversation message | Insert into `conversation_messages` → append to Redis list (trim). |

All writes should be wrapped in database transactions where multiple tables are updated (e.g., trade + equity snapshot) to ensure cache rebuilds receive consistent snapshots.

### 4.2 Replay Strategy

If ingest services crash mid-pipeline:
- Use PostgreSQL upserts with deterministic IDs (e.g., trade `id`, position `id`) so replays are safe.
- Redis idempotency keys (`nof0:ingest:trade:{trade_id}`) expire after 24h, giving enough time to prevent duplicates while allowing eventual reprocessing.

### 4.3 Observability

- **Metrics**: track write latency to Postgres, cache fill latency, Redis hit ratio, rate of `refresh_views_nof0()` runs.
- **Logs**: include model ID, symbol, and source event in ingestion logs for traceability.
- **Dashboards**: create Grafana panels for table growth (`price_ticks`, `trades`), cache key counts, and refresh cadence.

---

## 5. Migration Plan Summary

1. **Schema Deployment**: apply migrations 001 & 002; add new tables (`decision_cycles`, optional columns) via future migrations.
2. **Dual-write Phase**: loaders continue populating JSON while new ingestion services write to Postgres & Redis; API reads still use JSON to validate parity.
3. **Parity Validation**: compare API responses backed by caches vs. files using integration tests (`test/integration_test.go`).
4. **Cutover**: switch service context to Postgres/Redis repositories per endpoint, guarded by feature flags.
5. **Cleanup**: remove JSON loaders once confidence threshold met; ensure docs (`docs/engine.md`, this file) stay updated as schema evolves.

---

## 6. Reference Queries

```sql
-- Latest equity per model (leaderboard seed)
SELECT DISTINCT ON (model_id) model_id, ts_ms, equity_usd, realized_pnl, unrealized_pnl
FROM account_equity_snapshots
ORDER BY model_id, ts_ms DESC;

-- Active positions for a given model
SELECT *
FROM positions
WHERE model_id = $1 AND status = 'open'
ORDER BY entry_ts_ms DESC;

-- Trades over the last 24 hours
SELECT *
FROM trades
WHERE exit_ts_ms >= EXTRACT(EPOCH FROM (NOW() - INTERVAL '24 hours')) * 1000
ORDER BY exit_ts_ms DESC;

-- Refresh views after batch analytics
SELECT refresh_views_nof0();
```

---

This design should evolve with product requirements. When new features introduce additional data (e.g., risk audits, backtests), extend the tables and key patterns here and update the corresponding ingestion and cache flows.***
