# NOF0 Storage Redesign Implementation Plan

This document translates `docs/data-store.md` into an actionable implementation plan so that the trading platform can replace the JSON-backed `internal/model` usage with Postgres tables and Redis caches. The plan follows go-zero best practices, keeps table responsibilities isolated in dedicated repositories, and consolidates Redis key management.

## Objectives
- Ship production-ready Postgres persistence and Redis caching that match the design contract in `docs/data-store.md`.
- Isolate each table’s behaviour inside its own repository (`internal/repo/*_repo.go`) while reusing the generated go-zero models for CRUD operations.
- Centralise Redis key construction and TTL definitions to avoid key sprawl and align with `etc/nof0.yaml`.
- Provide a safe migration path that lets us dual write, validate parity, and then swap the application read paths from `internal/model` to the new repos.

## Current Assessment
- `internal/model` already contains go-zero generated models for the target tables but they are not orchestrated through a repository layer; no code currently uses them in business logic.
- `internal/svc/servicecontext.go` initialises a `sqlx.SqlConn`, optional `sqlc.CachedConn`, and a shared `cache.Cache`, which matches the integration test pattern in `internal/repo/storage_integration_test.go`. We should reuse these handles when wiring new repos.
- Redis keys are currently undocumented in code; `docs/data-store.md` lists a wide surface area (`nof0:*`). We need a key registry to ensure all code paths use the same names and TTLs.
- There is no migration or parity harness today; switching read paths abruptly would be risky without dual writes and comparison tooling.

## Postgres Implementation Plan
- **Migrations**: Confirm migrations `001_domain.sql` and `002_refresh_helpers.sql` cover all base tables. Create additive migrations for the remaining tables described in `docs/data-store.md` (`decision_cycles`, `trader_state`, etc.) and ensure indices / unique keys match the document. Use `goctl model pg` (or manual SQL) to regenerate models if schema changes.
- **Connection Pooling**: Keep using `sqlx.NewSqlConn("pgx", dsn)` with pool tuning via `applyPostgresPool`, as already implemented in `servicecontext.go`.
- **Transactions**: Provide helpers in the repo layer to run `sqlx.SqlConn.TransactCtx` so multi-table updates (e.g. `trades`, `positions`, `account_equity_snapshots`) stay atomic, aligning with Section 4 of the design doc.
- **Validation & Constraints**: Where the design describes relationships (e.g. `positions.model_id` referencing `models.id`), enforce them in application code first. If future migrations add FKs, update the generated model definitions.

## Redis Implementation Plan
- **Client Reuse**: Keep using go-zero’s `cache.Cache` from `servicecontext.go` and the cache options referenced in `internal/repo/storage_integration_test.go`. Provide typed wrappers that accept `context.Context` and propagate deadlines.
- **Key Registry**: Introduce `internal/repo/cache/keys.go` exporting functions such as `TraderPositionsKey(modelID string)` and `IngestTradeKey(tradeID string)`. The registry should:
  - Normalise prefixes (e.g. `nof0:positions:{model}`).
  - Document TTL classes (`Short`, `Medium`, `Long`) and provide helper constants pulled from config.
  - Support namespacing by provider or trader as described in Section 3 of `docs/data-store.md`.
- **Serialisation**: Standardise on JSON for hashes/lists unless RESP3 compression is available. Provide helper methods to marshal/unmarshal values to reduce duplication.
- **Fail-Safe TTLs**: Apply TTLs from config; for derived caches use `SetWithExpireCtx`. Persisted snapshots (e.g. idempotency keys) should honour the 24h guidance from the design doc.

## Repository Layer Structure
- Create a dedicated repo struct per table or cohesive aggregate located in `internal/repo`. Suggested files:
  - `models_repo.go`, `symbols_repo.go`, `accounts_repo.go`, `equity_repo.go` (wraps `account_equity_snapshots`), `positions_repo.go`, `trades_repo.go`, `price_repo.go`, `analytics_repo.go`, `conversations_repo.go`, `decision_cycles_repo.go`, `trader_state_repo.go`.
  - Each repo exposes an interface for CRUD operations, list helpers, and business-specific queries (e.g. `ListOpenByModel`, `InsertWithEquitySnapshot`).
  - Internally each repo embeds the generated model (e.g. `model.PositionsModel`) and the Redis cache handle when caches are involved.
  - For composite operations (trade ingestion updating multiple tables + caches), expose methods on an orchestrator repo (e.g. `TradingRepo`) that coordinates underlying repos inside a transaction.
- Repositories should accept the shared `sqlx.SqlConn` or `sqlc.CachedConn` plus the key registry and config-driven TTLs, staying idiomatic to go-zero layering.
- Where read-heavy queries exist (`price_latest`, `leaderboard`), leverage `sqlc.CachedConn` to combine DB queries with Redis caching through go-zero’s cached model pattern.

## Service Wiring
- Extend `svc.ServiceContext` to construct and expose the new repos after initialising the DB/Redis clients. Keep existing model fields (they remain useful for low-level access) but prefer injecting repositories into logic layers.
- Update constructors in `internal/logic/*` to depend on repository interfaces. This gradually decouples business logic from the raw models and enables easier testing.
- Ensure configuration values (`TTL`, Redis nodes, DSNs) still flow from `config.Config`. The integration test in `internal/repo/storage_integration_test.go` remains valid and can be expanded to assert repo health.

## Migration Strategy
1. **Dual Write Phase**: For each domain (prices, trades, positions, conversations), add repository usage alongside the existing JSON loaders so new data lands in Postgres/Redis while legacy paths continue to function.
2. **Parity Validation**: Build comparison jobs/tests that read from both sources and assert equivalence. Example: compare `positions` table results against JSON-backed responses for each trader.
3. **Read Switch**: Introduce feature flags or config toggles that let handlers prefer the repository outputs. Roll out per feature (prices → trades → analytics → conversations).
4. **Cleanup**: Once parity holds, remove JSON loaders and prune unused structs in `internal/data`.

## Testing & Observability
- **Unit Tests**: Add repo-level tests using a real Postgres/Redis instance (the integration test harness already demonstrates connectivity). Use small schema fixtures to verify transactional behaviour and cache writes.
- **Integration Tests**: Extend `internal/repo/storage_integration_test.go` with table/caching smoke tests (insert + fetch + cache read).
- **Metrics**: Instrument repo methods with timing/latency metrics, and extend existing Grafana dashboards as suggested in Section 4.3 of `docs/data-store.md`.
- **Logging**: Standardise structured logging across repos (model ID, symbol, provider) to match the observability guidelines in the design doc.

## Decision on Existing Code
- Do **not** delete the generated `internal/model` files; they already follow go-zero best practices for table access. Instead, wrap them inside the new repository structs and progressively replace direct usage of JSON loaders. Removing them would force us to re-run goctl generation without adding value.
- Retire legacy JSON readers only after parity is proven. Incremental migration provides faster feedback and lower risk than a full rewrite.

## Implementation Checklist
1. Confirm schema state and generate any missing migrations outlined in `docs/data-store.md`.
2. Build the Redis key registry and helper utilities.
3. Implement repository structs per table, including transactional helpers.
4. Wire repositories through `svc.ServiceContext` and update logic layers to consume them.
5. Introduce dual-write + parity validation tooling.
6. Flip feature flags to read from repositories, monitor, then remove legacy paths.

This plan keeps the codebase aligned with the storage design, leverages existing go-zero tooling, and provides a clear migration path without discarding useful generated code.
