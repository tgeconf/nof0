# NOF0 API Type Reference

This document translates the Go structs defined in `internal/types/types.go` into product-facing documentation. It explains what each field represents, whether the value is persisted as *primary data* (fetched from Postgres or Redis), or whether it is a *derived metric* computed from other fields. The data architecture that backs these payloads is outlined in `docs/data-architecture.md`.

## Legend
- **Primary (DB)** – persisted facts sourced from Postgres tables (e.g. `positions`, `trades`, `account_equity_snapshots`) and optionally cached in Redis without further transformation.
- **Primary (Redis)** – JSON blobs hydrated in Redis (e.g. `nof0:positions:{model_id}`) that mirror the Postgres source of truth.
- **Derived** – metrics computed from primary data. Unless stated otherwise, derived numbers are materialized by nightly/hourly analytics jobs and cached in Redis so responses stay read-only.

---

## Portfolio Snapshot Types

### `Position`
Open position snapshot keyed by `symbol` inside `AccountTotal.Positions`.

| Field | Go Type | Description | Classification | Storage or Formula |
|-------|---------|-------------|----------------|--------------------|
| `entry_oid` | `int64` | Exchange order identifier used to open the position. | Primary (DB) | Stored on ingest; persisted in `positions.id`/`entry_oid` and cached under `nof0:positions:{model_id}`. |
| `risk_usd` | `float64` | Budgeted capital-at-risk for the position. | Derived | Calculated by the risk engine from position size and stop-loss distance: `abs(stop_loss - entry_price) * abs(quantity)` rounded to model-specific risk bands. |
| `confidence` | `float64` | Model confidence score (0–1). | Primary (DB) | Ingested with the position; stored in `positions.confidence`. |
| `index_col` | `interface{}` | UI helper for tabular rendering (currently unused / null). | Derived | Set by presentation layer when needed; defaults to `null`. |
| `exit_plan` | `interface{}` | JSON document describing TP/SL levels and invalidation rules. | Primary (Redis) | Stored as JSON blob alongside the position payload in Redis. |
| `entry_time` | `float64` | Unix timestamp (seconds with decimals) when position was opened. | Primary (DB) | `positions.entry_ts_ms / 1000`. |
| `symbol` | `string` | Instrument ticker (e.g. `BTC`). | Primary (DB) | `positions.symbol`. |
| `entry_price` | `float64` | Executed entry price. | Primary (DB) | `positions.entry_price`. |
| `tp_oid` | `int64` | Take-profit order id, `-1` if absent. | Primary (DB) | Stored with position/order metadata. |
| `margin` | `float64` | Margin allocated to hold the position. | Derived | Computed from notional and leverage: `abs(quantity) * entry_price / leverage`, adjusted for exchange haircuts; stored with the cached payload. |
| `wait_for_fill` | `bool` | Indicates pending entry fill. | Primary (Redis) | Set while the position is staged; cleared once filled. |
| `sl_oid` | `int64` | Stop-loss order id, `-1` if absent. | Primary (DB) | Stored with order metadata. |
| `oid` | `int64` | Umbrella order id tying the position together. | Primary (DB) | Mirrors `entry_oid` for active positions. |
| `current_price` | `float64` | Latest mark price. | Primary (Redis) | Pulled from `nof0:price:latest:{symbol}` and attached at read time. |
| `closed_pnl` | `float64` | Realized PnL already locked in for this ticket (e.g. partial closes). | Derived | Aggregated from trade fills associated with `entry_oid`. |
| `liquidation_price` | `float64` | Exchange-calculated liquidation threshold. | Derived | Calculated from margin, maintenance margin rate, and leverage using venue formula. |
| `commission` | `float64` | Cumulative trading fees paid to open/adjust the position. | Primary (DB) | `positions.commission` with incremental updates from fill ingestion. |
| `leverage` | `float64` | Effective leverage used at entry. | Primary (DB) | `positions.leverage`. |
| `slippage` | `float64` | Slippage applied relative to quoted price. | Derived | Difference between intended quote and executed entry; stored if non-zero. |
| `quantity` | `float64` | Signed position size (negative for shorts). | Primary (DB) | `positions.quantity`. |
| `unrealized_pnl` | `float64` | Mark-to-market PnL. | Derived | `(current_price - entry_price) * quantity - accrued_fees_adjustment`. Rounded to cents. |

### `AccountTotal`
Aggregated account snapshot for a model at a point in time.

| Field | Go Type | Description | Classification | Storage or Formula |
|-------|---------|-------------|----------------|--------------------|
| `id` | `string` | Synthetic identifier `<model_id>_<sequence>` for ordering snapshots. | Derived | Generated when the snapshot is materialized. |
| `model_id` | `string` | Model / account identifier. | Primary (DB) | `account_equity_snapshots.model_id`. |
| `timestamp` | `float64` | Snapshot capture time (seconds). | Primary (DB) | `account_equity_snapshots.ts_ms / 1000`. |
| `dollar_equity` | `float64` | Total account equity in USD. | Primary (DB) | `account_equity_snapshots.equity_usd`. |
| `realized_pnl` | `float64` | Cumulative realized PnL. | Primary (DB) | `account_equity_snapshots.realized_pnl`. |
| `total_unrealized_pnl` | `float64` | Sum of open-position unrealized PnL. | Derived | Calculated as `Σ position.unrealized_pnl` and persisted with the snapshot. |
| `cum_pnl_pct` | `float64` | Cumulative return since \$10,000 seed capital. | Derived | `((dollar_equity - 10_000) / 10_000) * 100`, rounded to 2 decimals. |
| `sharpe_ratio` | `float64` | Rolling Sharpe ratio of hourly equity returns. | Derived | Computed over trailing 90-day equity series: `(mean(return) / std(return)) * sqrt(24)`; stored with the snapshot. |
| `since_inception_hourly_marker` | `int` | Cursor used for incremental hourly syncs. | Derived | `floor((timestamp - inception_ts)/3600)` cached in Redis to support `lastHourlyMarker`. |
| `since_inception_minute_marker` | `int` | Cursor for minute-level syncs. | Derived | `floor((timestamp - inception_ts)/60)`. |
| `positions` | `map[string]Position` | Open positions keyed by symbol. | Primary (Redis) | Hydrated from `nof0:positions:{model_id}` at snapshot time. |

### `AccountTotalsRequest`

| Field | Go Type | Description | Classification | Notes |
|-------|---------|-------------|----------------|-------|
| `lastHourlyMarker` | `int` | Optional cursor to request deltas since a given hourly bucket. | Derived | Client-provided filter; compared against `AccountTotal.since_inception_hourly_marker`. |

### `AccountTotalsResponse`

| Field | Go Type | Description | Classification | Storage or Formula |
|-------|---------|-------------|----------------|--------------------|
| `accountTotals` | `[]AccountTotal` | Ordered snapshots returned to the caller. | Primary (Redis) | Read from cached materialization keyed by `nof0:account_totals`. |
| `lastHourlyMarkerRead` | `int` | Highest hourly marker included in the response. | Derived | `max(AccountTotal.since_inception_hourly_marker)` of payload. |
| `serverTime` | `int64` | Millisecond timestamp when the response is served. | Derived | `time.Now().UnixMilli()` from API process. |

### `AccountValue`
One point on the “since inception” equity curve.

| Field | Go Type | Description | Classification | Storage |
|-------|---------|-------------|----------------|---------|
| `timestamp` | `int64` | Millisecond timestamp for the data point. | Primary (DB) | `v_since_inception.timestamp`. |
| `value` | `float64` | Account value at the timestamp. | Primary (DB) | `v_since_inception.value` (a view over `account_equity_snapshots`). |

---

## Market Data Types

### `CryptoPrice`

| Field | Go Type | Description | Classification | Storage |
|-------|---------|-------------|----------------|---------|
| `symbol` | `string` | Asset ticker (BTC, ETH, …). | Primary (DB) | Upserted in `price_latest.symbol`. |
| `price` | `float64` | Latest trade/mark price. | Primary (DB) | `price_latest.price`; mirrored in Redis `nof0:price:latest:{symbol}`. |
| `timestamp` | `int64` | Millisecond timestamp of the quote. | Primary (DB) | `price_latest.ts_ms`. |

### `CryptoPricesResponse`

| Field | Go Type | Description | Classification | Storage or Formula |
|-------|---------|-------------|----------------|--------------------|
| `prices` | `map[string]CryptoPrice` | Map of symbols to latest quotes. | Primary (Redis) | `nof0:crypto_prices` cache snapshot (refreshed every ~10s). |
| `serverTime` | `int64` | Time of response generation (ms). | Derived | `time.Now().UnixMilli()`. |

---

## Leaderboard Types

### `LeaderboardEntry`

| Field | Go Type | Description | Classification | Storage or Formula |
|-------|---------|-------------|----------------|--------------------|
| `id` | `string` | Model identifier (slug). | Primary (DB) | `models.id`. |
| `num_trades` | `int` | Total completed trades. | Derived | Count of `trades` rows for the model. |
| `sharpe` | `float64` | Annualized Sharpe ratio. | Derived | Computed from hourly equity returns; cached in Redis leaderboard payload. |
| `win_dollars` | `float64` | Cumulative gross dollars from winning trades. | Derived | `Σ realized_gross_pnl` over winning trades. |
| `num_losses` | `int` | Losing trade count. | Derived | Count of trades where `realized_net_pnl < 0`. |
| `lose_dollars` | `float64` | Absolute gross dollars lost. | Derived | `Σ |realized_gross_pnl|` over losers (stored as positive magnitude). |
| `return_pct` | `float64` | Total return since inception. | Derived | `(latest_equity - 10_000) / 10_000 * 100`. |
| `equity` | `float64` | Latest account equity. | Primary (DB) | Latest row from `account_equity_snapshots`. |
| `num_wins` | `int` | Winning trade count. | Derived | Count of trades where `realized_net_pnl > 0`. |

### `LeaderboardResponse`

| Field | Go Type | Description | Classification | Storage |
|-------|---------|-------------|----------------|---------|
| `leaderboard` | `[]LeaderboardEntry` | Sorted leaderboard snapshot. | Primary (Redis) | Backed by sorted-set cache `nof0:leaderboard:cache`, refreshed from materialized view `v_leaderboard`. |

---

## Analytics Types

### `BreakdownTable`
Container reused across analytics sections. Every field is **Derived** and produced by the analytics ETL from trades, signals, and invocation telemetry. Values are materialized into `model_analytics.payload` (JSONB) and cached at `nof0:analytics:{model_id}`.

#### Fee & PnL Metrics

| Field | Description | Calculation |
|-------|-------------|-------------|
| `std_net_pnl` | Standard deviation of net PnL per trade. | `stdev(realized_net_pnl)` across trade set. |
| `total_fees_paid` | Aggregate taker fees. | `Σ (entry_commission_dollars + exit_commission_dollars + adjustments)`. |
| `overall_pnl_without_fees` | Gross PnL sum. | `Σ realized_gross_pnl`. |
| `total_fees_as_pct_of_pnl` | Fee drag ratio. | `total_fees_paid / max(overall_pnl_without_fees, ε) * 100`. |
| `overall_pnl_with_fees` | Net PnL after fees. | `overall_pnl_without_fees - total_fees_paid`. |
| `avg_taker_fee` | Mean fee per trade. | `total_fees_paid / trade_count`. |
| `std_gross_pnl` | Std dev of gross PnL per trade. | `stdev(realized_gross_pnl)`. |
| `avg_net_pnl` | Mean net PnL per trade. | `overall_pnl_with_fees / trade_count`. |
| `biggest_net_loss` | Worst single-trade net PnL. | `min(realized_net_pnl)`. |
| `biggest_net_gain` | Best single-trade net PnL. | `max(realized_net_pnl)`. |
| `avg_gross_pnl` | Mean gross PnL per trade. | `overall_pnl_without_fees / trade_count`. |
| `std_taker_fee` | Std dev of total fees per trade. | `stdev(total_fee_per_trade)`. |

#### Winners vs. Losers Metrics

| Field | Description |
|-------|-------------|
| `std_losers_notional` | Std dev of notional size among losing trades. |
| `std_winners_notional` | Std dev of notional size among winners. |
| `avg_winners_net_pnl` | Mean net PnL for winning trades. |
| `win_rate` | `num_wins / trade_count * 100`. |
| `std_losers_net_pnl` | Std dev of net PnL across losers. |
| `avg_losers_net_pnl` | Mean net PnL across losers (negative). |
| `std_losers_holding_period` | Std dev of holding period (minutes) for losers. |
| `avg_losers_notional` | Mean notional value of losing trades. |
| `avg_losers_holding_period` | Mean minutes in position for losers. |
| `avg_winners_holding_period` | Mean minutes in position for winners. |
| `std_winners_net_pnl` | Std dev of winners’ net PnL. |
| `avg_winners_notional` | Mean notional value of winners. |
| `std_winners_holding_period` | Std dev of winners’ holding period (minutes). |

#### Long vs. Short Metrics

| Field | Description |
|-------|-------------|
| `std_longs_holding_period` | Std dev of holding time for longs. |
| `std_longs_notional` | Std dev of notional exposure for longs. |
| `std_shorts_notional` | Std dev of notional exposure for shorts. |
| `num_long_trades` | Count of long trades. |
| `avg_longs_notional` | Mean notional size of longs. |
| `avg_shorts_holding_period` | Mean minutes held for shorts. |
| `avg_shorts_net_pnl` | Mean net PnL for shorts. |
| `avg_longs_net_pnl` | Mean net PnL for longs. |
| `std_shorts_holding_period` | Std dev of short holding periods. |
| `num_short_trades` | Count of short trades. |
| `long_short_trades_ratio` | `num_long_trades / max(num_short_trades, 1)`. |
| `std_shorts_net_pnl` | Std dev of shorts’ net PnL. |
| `avg_longs_holding_period` | Mean holding period for longs. |
| `std_longs_net_pnl` | Std dev of longs’ net PnL. |
| `avg_shorts_notional` | Mean notional size of shorts. |

#### Signals Metrics

| Field | Description |
|-------|-------------|
| `num_short_signals` | Short entry signal count. |
| `avg_confidence_close` | Mean model confidence for close signals. |
| `avg_leverage_long` | Mean leverage at long entries. |
| `std_leverage` | Std dev of leverage across all signals. |
| `pct_mins_flat_combined` | Percent of monitored minutes with no open position. |
| `num_close_signals` | Close signal count. |
| `mins_long_combined` | Aggregate minutes spent long. |
| `std_confidence` | Std dev of confidence across signals. |
| `long_signal_pct` | `num_long_signals / total_signals * 100`. |
| `mins_short_combined` | Aggregate minutes spent short. |
| `num_hold_signals` | Hold signal count. |
| `std_confidence_short` | Std dev of confidence for short signals. |
| `avg_leverage` | Overall mean leverage. |
| `median_leverage` | Median leverage across signals. |
| `hold_signal_pct` | `num_hold_signals / total_signals * 100`. |
| `close_signal_pct` | `num_close_signals / total_signals * 100`. |
| `avg_confidence_long` | Mean confidence for long signals. |
| `avg_confidence` | Mean confidence across all signals. |
| `median_confidence` | Median signal confidence. |
| `total_signals` | Total signals observed. |
| `short_signal_pct` | `num_short_signals / total_signals * 100`. |
| `std_leverage_long` | Std dev of leverage for longs. |
| `num_long_signals` | Long signal count. |
| `long_short_ratio` | `num_long_signals / max(num_short_signals, 1)`. |
| `pct_mins_short_combined` | Percent of time spent short. |
| `std_confidence_hold` | Std dev of hold-signal confidence. |
| `mins_flat_combined` | Aggregate minutes flat. |
| `std_leverage_short` | Std dev of leverage for shorts. |
| `std_confidence_long` | Std dev of long-signal confidence. |
| `avg_confidence_hold` | Mean confidence on hold signals. |
| `avg_confidence_short` | Mean confidence for short signals. |
| `std_confidence_close` | Std dev of close-signal confidence. |
| `avg_leverage_short` | Mean leverage for shorts. |
| `pct_mins_long_combined` | Percent of time spent long. |

#### Invocation Cadence Metrics

| Field | Description |
|-------|-------------|
| `max_invocation_break_mins` | Longest gap between model invocations. |
| `std_invocation_break_mins` | Std dev of invocation gaps. |
| `num_invocations` | Invocation count within window. |
| `min_invocation_break_mins` | Smallest gap between invocations. |
| `avg_invocation_break_mins` | Mean gap (minutes). |

#### Overall Trade Overview Metrics

| Field | Description |
|-------|-------------|
| `avg_convo_leverage` | Mean leverage stated in conversation prompts. |
| `avg_holding_period_mins` | Mean holding period across all trades. |
| `avg_take_profit_distance_pct` | Mean TP distance as % of entry. |
| `median_holding_period_mins` | Median holding period. |
| `std_size_of_trade_notional` | Std dev of trade notional size. |
| `avg_size_of_trade_notional` | Mean notional size. |
| `total_trades` | Count of trades analysed. |
| `median_size_of_trade_notional` | Median trade notional. |
| `avg_size_of_trade_portfolio_pct` | Mean trade size as % of equity. |
| `std_holding_period_mins` | Std dev of holding period. |
| `avg_stop_loss_distance_pct` | Mean stop-loss distance as % of entry. |
| `std_size_of_trade_portfolio_pct` | Std dev of trade size % of equity. |
| `median_convo_leverage` | Median leverage referenced in prompts. |

### `ModelAnalytics`

| Field | Go Type | Description | Classification | Storage |
|-------|---------|-------------|----------------|---------|
| `id` | `string` | Unique analytics document id. | Primary (DB) | `model_analytics.model_id`. |
| `model_id` | `string` | Model identifier. | Primary (DB) | Same as `id`. |
| `updated_at` | `float64` | Last computation timestamp (seconds). | Primary (DB) | Derived from `model_analytics.updated_at`. |
| `fee_pnl_moves_breakdown_table` | `BreakdownTable` | Fee & PnL metrics. | Derived | Precomputed analytics payload. |
| `winners_losers_breakdown_table` | `BreakdownTable` | Winner/loser stats. | Derived | Precomputed. |
| `signals_breakdown_table` | `BreakdownTable` | Signal cadence metrics. | Derived | Precomputed. |
| `last_trade_exit_time` | `float64` | Timestamp of latest closed trade. | Derived | `max(trades.exit_ts_ms) / 1000`. |
| `invocation_breakdown_table` | `BreakdownTable` | Invocation gaps. | Derived | Precomputed. |
| `longs_shorts_breakdown_table` | `BreakdownTable` | Long/short stats. | Derived | Precomputed. |
| `last_convo_doc_id` | `string` | Conversation document identifier. | Primary (DB) | Stored alongside conversation metadata in analytics payload. |
| `last_convo_timestamp` | `float64` | Timestamp of latest coaching conversation. | Derived | `max(conversation_messages.ts_ms)/1000` included in payload. |
| `overall_trades_overview_table` | `BreakdownTable` | Portfolio-level view. | Derived | Precomputed. |
| `last_trade_doc_id` | `string` | Identifier of latest trade document. | Primary (DB) | Provided by ingest process. |

### `AnalyticsResponse`

| Field | Go Type | Description | Classification | Storage |
|-------|---------|-------------|----------------|---------|
| `analytics` | `[]ModelAnalytics` | Analytics for all models. | Primary (Redis) | Cached bundle from `nof0:analytics:all`. |
| `serverTime` | `int64` | Response timestamp (ms). | Derived | Generated at request time. |

### `ModelAnalyticsResponse`

| Field | Go Type | Description | Classification | Storage |
|-------|---------|-------------|----------------|---------|
| `analytics` | `ModelAnalytics` | Analytics for a single model. | Primary (Redis) | `nof0:analytics:{model_id}`. |
| `serverTime` | `int64` | Response timestamp. | Derived | Generated at request time. |

---

## Since Inception Value Types

### `SinceInceptionValue`

| Field | Go Type | Description | Classification | Storage |
|-------|---------|-------------|----------------|---------|
| `id` | `string` | Snapshot document id. | Derived | Generated when exporting data. |
| `nav_since_inception` | `float64` | Cumulative NAV (USD). | Derived | Rolling equity normalized to initial capital: `equity / 10_000`. |
| `inception_date` | `float64` | Timestamp (seconds) of competition start for the model. | Primary (DB) | Stored with model metadata. |
| `num_invocations` | `int` | Total strategy invocations observed. | Derived | Count of entries in invocation logs. |
| `model_id` | `string` | Model identifier. | Primary (DB) | `account_equity_snapshots.model_id`. |

### `SinceInceptionResponse`

| Field | Go Type | Description | Classification | Storage |
|-------|---------|-------------|----------------|---------|
| `sinceInceptionValues` | `[]SinceInceptionValue` | Down-sampled equity curves per model. | Primary (Redis) | Cached list keyed by `nof0:since_inception:{model_id}`. |
| `serverTime` | `int64` | Response timestamp (ms). | Derived | Generated at request time. |

---

## Trade History Types

### `Trade`

| Field | Go Type | Description | Classification | Storage / Formula |
|-------|---------|-------------|----------------|-------------------|
| `id` | `string` | Trade document id `<model_uuid>_<uuid>`. | Primary (DB) | `trades.id`. |
| `model_id` | `string` | Owning model. | Primary (DB) | `trades.model_id`. |
| `symbol` | `string` | Instrument ticker. | Primary (DB) | `trades.symbol`. |
| `side` | `string` | Direction (`long`/`short`). | Primary (DB) | `trades.side`. |
| `trade_type` | `string` | Strategy-defined trade classification. | Primary (DB) | `trades.trade_type`. |
| `trade_id` | `string` | Exchange/strategy composite id. | Primary (DB) | Stored for traceability. |
| `quantity` | `float64` | Executed size (signed). | Primary (DB) | `trades.quantity`. |
| `leverage` | `float64` | Leverage applied. | Primary (DB) | `trades.leverage`. |
| `confidence` | `float64` | Confidence score at execution. | Primary (DB) | `trades.confidence`. |
| `entry_price` | `float64` | Entry fill price. | Primary (DB) | `trades.entry_price`. |
| `entry_time` | `float64` | Entry timestamp (seconds). | Primary (DB) | `trades.entry_ts_ms / 1000`. |
| `entry_human_time` | `string` | Human-readable entry timestamp. | Derived | `time.UnixMilli(entry_ts_ms).Format("2006-01-02 15:04:05.000000")`. |
| `entry_sz` | `float64` | Notional size signed at entry. | Derived | `quantity` adjusted for contract multiplier (alias to `quantity` for spot pairs). |
| `entry_tid` | `int64` | Exchange trade identifier for entry. | Primary (DB) | Stored with execution audit. |
| `entry_oid` | `int64` | Entry order id. | Primary (DB) | `trades.entry_oid`. |
| `entry_crossed` | `bool` | Whether entry crossed the book (market order). | Derived | Inferred from execution flags; persisted in payload. |
| `entry_liquidation` | `interface{}` | Liquidation info at entry (usually `null`). | Primary (DB) | Reserved for venue data. |
| `entry_commission_dollars` | `float64` | Fees incurred on entry. | Primary (DB) | Stored alongside fills. |
| `entry_closed_pnl` | `float64` | Partial PnL locked after entry (normally negative fees). | Derived | `-entry_commission_dollars` plus any rebates. |
| `exit_price` | `float64` | Exit fill price. | Primary (DB) | `trades.exit_price`. |
| `exit_time` | `float64` | Exit timestamp (seconds). | Primary (DB) | `trades.exit_ts_ms / 1000`. |
| `exit_human_time` | `string` | Human-readable exit time. | Derived | Formatted from `exit_time`. |
| `exit_sz` | `float64` | Size closed. | Primary (DB) | For full closes equals `abs(quantity)`; supports partials. |
| `exit_tid` | `int64` | Exit trade identifier. | Primary (DB) | Stored with execution audit. |
| `exit_oid` | `int64` | Exit order id. | Primary (DB) | `trades.exit_oid`. |
| `exit_crossed` | `bool` | Whether exit crossed the book. | Derived | Inferred from execution flags. |
| `exit_liquidation` | `interface{}` | Liquidation metadata at exit. | Primary (DB) | Typically `null`. |
| `exit_commission_dollars` | `float64` | Fees incurred on exit. | Primary (DB) | Stored with fills. |
| `exit_closed_pnl` | `float64` | PnL component recognized at exit before netting with entry fees. | Derived | `realized_gross_pnl - exit_commission_dollars`. |
| `exit_plan` | `interface{}` | Exit rationale/plan snapshot. | Primary (DB) | Captured from model plan JSON. |
| `realized_gross_pnl` | `float64` | Gross PnL from entry vs exit. | Derived | `(exit_price - entry_price) * quantity` (sign-aware). |
| `realized_net_pnl` | `float64` | Net PnL after fees. | Derived | `realized_gross_pnl - total_commission_dollars`. |
| `total_commission_dollars` | `float64` | Total fees for the trade. | Derived | `entry_commission_dollars + exit_commission_dollars + adjustments`. |

### `TradesResponse`

| Field | Go Type | Description | Classification | Storage |
|-------|---------|-------------|----------------|---------|
| `trades` | `[]Trade` | Chronological list of trades. | Primary (Redis) | Cached slice under `nof0:trades:recent:{model_id}` and `nof0:trades:stream`. |
| `serverTime` | `int64` | Response timestamp (ms). | Derived | Generated at request time. |

---

## Position Collection Types

### `PositionsRequest`

| Field | Go Type | Description | Classification | Notes |
|-------|---------|-------------|----------------|-------|
| `limit` | `int` | Cap on number of models returned (default 1000). | Derived | Parsed from query string; used to bound Redis fetch. |

### `PositionsByModel`

| Field | Go Type | Description | Classification | Storage |
|-------|---------|-------------|----------------|---------|
| `model_id` | `string` | Portfolio owner. | Primary (DB) | `models.id`. |
| `positions` | `map[string]Position` | Open positions keyed by symbol. | Primary (Redis) | `nof0:positions:{model_id}`. |

### `PositionsResponse`

| Field | Go Type | Description | Classification | Storage |
|-------|---------|-------------|----------------|---------|
| `accountTotals` | `[]PositionsByModel` | Open positions grouped by model. | Primary (Redis) | Cache hydration from per-model keys during request. |
| `serverTime` | `int64` | Response timestamp (ms). | Derived | Generated at request time. |

---

## Conversation Types

### `ConversationMessage`

| Field | Go Type | Description | Classification | Storage |
|-------|---------|-------------|----------------|---------|
| `role` | `string` | Speaker (`system`, `user`, `assistant`). | Primary (DB) | `conversation_messages.role`. |
| `content` | `string` | Message body. | Primary (DB) | `conversation_messages.content`. |
| `timestamp` | `interface{}` | Optional timestamp (ms). | Primary (DB) | `conversation_messages.ts_ms`; omitted when unavailable. |

### `Conversation`

| Field | Go Type | Description | Classification | Storage |
|-------|---------|-------------|----------------|---------|
| `model_id` | `string` | Owning model. | Primary (DB) | `conversations.model_id`. |
| `messages` | `[]ConversationMessage` | Ordered chat transcript. | Primary (DB) | Joined via `conversation_messages`. |

### `ConversationsResponse`

| Field | Go Type | Description | Classification | Storage |
|-------|---------|-------------|----------------|---------|
| `conversations` | `[]Conversation` | Recent conversations per model. | Primary (Redis) | Cached under `nof0:conversations:{model_id}` with fallback to DB. |
| `serverTime` | `int64` | Response timestamp (ms). | Derived | Generated at request time. |

---

## Response Timestamp Convention

For every `*Response` struct that exposes `serverTime`, the API populates the field with `time.Now().UnixMilli()` immediately before serializing the payload. This guarantees clients can measure staleness relative to local clocks even when the underlying data originates from materialized views or Redis caches updated on a schedule.
