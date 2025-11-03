# NOF0 Storage Redesign TODO

## 1. Database Schema & goctl Models
- [x] Audit migrations `001_domain.sql` and `002_refresh_helpers.sql`; confirm they match the target schemas in `docs/data-store.md`.
- [x] Author additive migrations for missing entities (e.g. `decision_cycles`, `trader_state`, leaderboard materialisations) with all indexes/constraints described in the redesign.
- [x] Establish a migration walkthrough (local + CI) to validate the new schema against Postgres 14+.
- [x] Use `goctl model pg` (with cache flags where appropriate) after every schema change to regenerate `internal/model` packages for each table; ensure generated code compiles and is committed.
- [x] Document the goctl invocation (DSN env vars, output directories, cache options) in `docs/data-store.md` or a README snippet so future schema updates stay reproducible.

## 2. Redis Key Registry & Cache Strategy
- [x] Create `internal/repo/cache/keys.go` (or similar) defining strongly typed key builders and TTL classes aligned with `etc/nof0.yaml`.
- [x] Enumerate every Redis structure the redesign expects (`nof0:positions`, idempotency keys, analytics snapshots) and codify them with helpers plus comments on eviction policy.
- [ ] Replace ad-hoc cache touches in new code with the registry helpers to guarantee consistent prefixes and expirations.

## 3. Repository Layer Implementation
- [ ] Scaffold per-table repositories in `internal/repo` (models, symbols, price_ticks, price_latest, accounts, account_equity_snapshots, positions, trades, model_analytics, conversations, conversation_messages, decision_cycles, trader_state, leaderboard views).
- [ ] Embed the regenerated goctl models inside each repository, wiring in the shared `sqlx.SqlConn` / `sqlc.CachedConn` and cache handle from `ServiceContext`.
- [ ] Provide transactional helpers for multi-table write paths (e.g. trade ingestion updating positions + equity + caches).
- [ ] Expose repository interfaces that capture read/write operations needed by logic so we can inject mocks during tests.

## 4. Service Wiring & Configuration
- [ ] Extend `svc.ServiceContext` to construct the new repositories and key registry after DB/cache clients are initialised; keep `DataLoader` available temporarily for parity checks.
- [ ] Update logic constructors to accept repository interfaces (either directly or via a new `svc.Repos` struct) without breaking existing handlers.
- [ ] Ensure config-driven TTLs, DSNs, and feature flags flow through `config.Config` and are documented for operations.

## 5. Logic Conversion & Data Assembly
Create a dedicated assembly layer (e.g. `internal/assembler`) responsible for mapping repository outputs to the API DTOs in `internal/types/types.go`. For each endpoint:

### Account Totals & Positions (`types.AccountTotalsResponse`, `types.PositionsResponse`)
- [ ] Fetch latest account equity snapshots plus open positions per model; combine into `types.AccountTotal` objects, populating the nested `Positions map[string]types.Position`.
- [ ] Preserve JSON-era defaults (`LastHourlyMarkerRead`, empty maps) and inject `ServerTime = time.Now().UnixMilli()`.
- [ ] Provide helper to reuse the same assembled data for both `/accountTotals` and `/positions` logic paths to avoid divergence.

### Trades (`types.TradesResponse`)
- [ ] Read the most recent trades per model (respecting pagination/ordering rules) from Postgres; fall back to Redis cache if the redesign dictates.
- [ ] Map numeric/timestamp fields to the struct (including `EntryHumanTime`, `ExitHumanTime`, `EntryLiquidation`, etc.) and ensure transformation handles nullables gracefully.
- [ ] Inject `ServerTime` and maintain feature flag for dual-read during rollout.

### Crypto Prices (`types.CryptoPricesResponse`)
- [ ] Retrieve price ticks or cached latest prices for active symbols; normalize into the `map[string]types.CryptoPrice` expected by the API.
- [ ] Confirm timestamp units (ms) align with existing JSON payloads before switching handlers.

### Since Inception (`types.SinceInceptionResponse`)
- [ ] Pull inception metrics from the redesigned table/view; convert to `types.SinceInceptionValue` and set `ServerTime`.
- [ ] Backfill or calculate any derived markers (`NumInvocations`, `InceptionDate`) if the table stores different representations.

### Leaderboard (`types.LeaderboardResponse`)
- [ ] Source leaderboard rows from the planned Postgres view or cache; translate numeric ranks into `types.LeaderboardEntry`.
- [ ] Verify field names (`win_dollars`, `lose_dollars`, etc.) match API casing to avoid regressions.

### Analytics & Model Analytics (`types.AnalyticsResponse`, `types.ModelAnalyticsResponse`)
- [ ] Load analytics payloads from their storage table (JSONB or decomposed columns); unmarshal or assemble into `types.ModelAnalytics` and `types.BreakdownTable`.
- [ ] Ensure per-model endpoint reuses shared assembly logic and respects cache invalidation strategy.

### Conversations (`types.ConversationsResponse`)
- [ ] Query conversations and messages tables in chronological order, mapping to `types.Conversation` and `types.ConversationMessage`.
- [ ] Handle optional timestamps (JSON `interface{}` fields) and confirm text encodings remain UTF-8 when read from Postgres.

## 6. Dual Write, Feature Flags & Parity Validation
- [ ] Implement dual-write paths so incoming data populates both JSON legacy storage and new repositories until parity is proven.
- [ ] Build parity checks comparing repository assemblies to existing `DataLoader` outputs for each endpoint; surface diffs via CLI or metrics.
- [ ] Add feature flags/config toggles to switch individual endpoints from DataLoader to repo-backed implementations gradually.

## 7. Testing & Tooling
- [ ] Expand `internal/repo/storage_integration_test.go` to cover new repositories, transactional flows, and Redis interactions.
- [ ] Add unit tests for the assembler layer verifying deterministic mapping from repo entities to `types.*` structs.
- [ ] Provide seed fixtures or factories for Postgres/Redis to enable reproducible local testing.
- [ ] Wire CI to run migrations, repository tests, and assembler tests.

## 8. Observability & Operations
- [ ] Instrument repository and assembler methods with structured logs and latency metrics as outlined in the redesign.
- [ ] Update dashboards/alerts to track Postgres/Redis health, cache hit ratios, and parity job success.
- [ ] Document operational playbooks for backfilling tables and cache warming.

## 9. Cleanup & Decommissioning
- [ ] Once parity is validated, flip feature flags to make repository outputs authoritative for each endpoint.
- [ ] Remove JSON files, `internal/data/loader.go`, and related configuration once the new path is stable.
- [ ] Archive or migrate any scripts relying on the JSON dataset.
