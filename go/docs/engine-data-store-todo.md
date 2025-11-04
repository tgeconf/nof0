# Engine Data Store – Execution TODO

## P0 – unblock persistence plumbing (Week 0)

1. **Inject persistence and cache dependencies into `pkg/manager`.**
   - Extend `Manager` struct to accept the required `internal/model` repositories (positions, trades, account_equity_snapshots, decision_cycles, model_analytics, trader_state) plus a Redis/cache client interface.
   - Update constructors (`NewManager`, wiring in `internal/svc`) so every `VirtualTrader` can access these collaborators without global state.
2. **Define a dedicated persistence interface/service.**
   - Wrap common DB + Redis operations (e.g., `SavePosition`, `ClosePosition`, `SaveSnapshot`, `PublishTradeEvent`) behind a small interface so manager code stays testable.
   - Decide where transactions live (service vs. call site) and document retry semantics.

## P1 – core trading loop persistence (Week 1–2)

3. **Augment `ExecuteDecision` (open path) with Postgres + Redis writes.**
   - After a successful exchange submission, insert/ upsert into `positions`, include fill metadata, and update the `nof0:positions:{model_id}` hash.
   - Ensure cache-aside ordering (DB first, cache second) and guard against duplicate opens via idempotency keys.
4. **Handle closes with full lifecycle bookkeeping.**
   - On `close_long/close_short`, update `positions` status, insert `trades`, compute realized PnL/fees, and trim Redis caches (`HDEL`, `LPUSH` recent trades, `XADD` stream).
   - Wrap DB work in a transaction to keep position+trade consistent.
5. **Persist decision-cycle/journal data.**
   - Extend `writeJournalRecord` (or a new hook) to insert into `decision_cycles` and refresh `nof0:decision:last:{model_id}` while still writing the JSON journal file.
6. **Implement `SyncTraderPositions` writes.**
   - Batch insert `account_equity_snapshots`, update `model_analytics`, and refresh analytics/leaderboard caches with throttling (e.g., min 5 min between snapshots per trader).

## P2 – market data + consistency (Week 2–3)

7. **Design market-data ingestion path.**
   - Decide whether persistence happens inside `pkg/market` providers or via a separate ingestion worker.
   - Implement writes for `market_assets`, `market_asset_ctx`, `price_latest`, `price_ticks`, and matching Redis keys with proper rate limiting.
8. **Bootstrap cache warm-up + consistency jobs.**
   - On startup, hydrate Redis positions/trades/analytics from Postgres.
   - Add periodic consistency checks (compare DB vs cache vs exchange) and document remediation steps.

## P3 – resilience & observability (Week 3+)

9. **Error handling & retry strategy.**
   - Define which DB/cache failures block execution vs. get queued for async retry, and add structured logging/alerting for each failure class.
10. **Performance & monitoring additions.**
    - Introduce batching (COPY/`INSERT ... VALUES` for ticks, Redis pipelining) once correctness is proven.
    - Emit metrics: `db_writes_total`, `persistence_latency_seconds`, cache hit ratios, inconsistency counters.

> Tracking convention: mark each item as `[ ]` / `[x]` once implemented in code and keep links to the relevant PRs for auditability.
