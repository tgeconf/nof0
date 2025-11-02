# Manager/Executor/Hyperliquid — Decision-to-Execution Closed Loop (Design)

This document consolidates the current assessment and a practical plan to harden the decision pipeline across Manager, Executor, and the Hyperliquid integration. It focuses on closing the loop from decision generation → risk controls → order execution → logging and performance feedback, while avoiding any references to external competitors.

## Objectives

- Establish a robust, auditable, and configurable decision pipeline.
- Enforce exchange-aligned constraints (precision, step size, leverage limits, margin usage) before orders hit the book.
- Reduce rejection and drift by aligning prompts, validators, and executors to the same rule set.
- Provide decision-cycle logging and performance feedback that drive adaptive gating (e.g., Sharpe-based pause/slowdown).

## High-Level Gaps Observed

- Scheduling/context: Manager requests decisions with a minimal context; account, positions, candidate set, market snapshots (incl. OI/funding), and recent performance are not consistently provided to the Executor.
- Risk closure: Validator covers RR/min-confidence/leverage caps/position count, but lacks margin-usage cap, position value bands (BTC/ETH vs altcoins), liquidity threshold, and symbol cooldown. Manager does not enforce a second line of defense before placing orders.
- Execution quality: Orders are submitted as IOC “quasi-market” without precision/step-size clamping, idempotency, error-classification, or immediate RO SL/TP orchestration after fills.
- Observability: There is no per-cycle decision record (prompt, CoT, structured decisions, snapshots, actions), no basic statistics, and no performance loop-back to the Executor.

## Improvement Plan (Phased)

- P0 (close the loop; reduce rejections and drift)
  - Manager builds a complete Executor context per cycle (account, positions, candidates, market snapshots including OI/funding/trend deltas, asset meta, recent performance).
  - Executor returns multiple decisions; Validator enforces extended hard rules (liquidity, margin-usage, value bands, asset-level max leverage, cooldown). Manager sorts actions (close → open) and caps new opens per cycle.
  - Hyperliquid execution: price/size clamped to tick/step; idempotent `cloid`; open → set RO stop-loss/take-profit; error classification + minimal retries.
  - Decision-cycle JSON logs with prompt digest/CoT/decisions/snapshots/action results; basic analytics feed performance back to Executor.
- P1 (stability and UX)
  - Performance-gated prompting and throttling (Sharpe thresholds) and decision parsing robustness (fallback JSON repair + gentle single retry).
  - Execution orchestration refinements (partial fills, protective limits, simple scaling in/out).
- P2 (streaming and replay)
  - WebSocket shadow state (accounts, orders, fills, positions); end-to-end metrics; offline replay harness to validate precision/step/risk rejections and latency.

## Configuration Schema (Proposed)

These live at the trader level (manager config) and may also be mirrored into executor overrides when needed.

- `max_new_positions_per_cycle` (int, default 1)
- `cooldown_after_close` (duration, default 15m)
- `liquidity_threshold_usd` (float, default 15000000)
- `max_margin_usage_pct` (float, default 90)
- `btceth_position_value_min_equity_multiple` (float, default 5)
- `btceth_position_value_max_equity_multiple` (float, default 10)
- `alt_position_value_min_equity_multiple` (float, default 0.8)
- `alt_position_value_max_equity_multiple` (float, default 1.5)
- `sharpe_pause_threshold` (float, default -0.5)
- `pause_duration_on_breach` (duration, default 18m)

Example snippet:

```yaml
traders:
  - id: t01
    name: Spotter
    decision_interval: 3m
    exchange_provider: hyperliquid_testnet
    market_provider: hyperliquid
    allocation_pct: 100
    risk_params:
      major_coin_leverage: 20
      altcoin_leverage: 10
      min_confidence: 70
      min_risk_reward_ratio: 3.0
      max_positions: 3
      max_position_size_usd: 1500
    exec_guards:
      max_new_positions_per_cycle: 1
      cooldown_after_close: 15m
      liquidity_threshold_usd: 15000000
      max_margin_usage_pct: 90
      btceth_position_value_min_equity_multiple: 5
      btceth_position_value_max_equity_multiple: 10
      alt_position_value_min_equity_multiple: 0.8
      alt_position_value_max_equity_multiple: 1.5
      sharpe_pause_threshold: -0.5
      pause_duration_on_breach: 18m
```

## Executor Context Extensions

Extend `executorpkg.Context` with the following (populated by Manager each cycle):

- `AssetMeta map[string]AssetMeta`
  - `AssetMeta { MaxLeverage float64, PriceTick float64, SizeStep float64, SzDecimals int }`
- `RecentlyClosed map[string]time.Time`
- `MaxMarginUsagePct float64`
- `LiquidityThresholdUSD float64`
- `MaxNewPositionsPerCycle int`

Purpose:
- Align decision validation with exchange constraints (max leverage, precision/step).
- Enforce liquidity and margin budget gates in both Validator and execution.
- Support symbol cooldown and per-cycle budgeting.

## Decision Output (Array Contract)

Fields (kept compatible with the current single-decision structure):
- `symbol` string (required)
- `action` one of {`open_long`, `open_short`, `close_long`, `close_short`, `hold`, `wait`}
- `leverage` int (required for opens)
- `position_size_usd` float (required for opens)
- `entry_price` float (optional; fallback to snapshot)
- `stop_loss`, `take_profit` float (required for opens)
- `confidence` int [0..100]
- `risk_usd` float (optional)
- `reasoning` string (optional)
- `invalidation_condition` string (optional)

Parsing strategy:
- Prefer structured output (schema/JSON). If it fails: tolerant extraction (repair unicode quotes and locate the first complete JSON array). If still invalid, return an error while preserving CoT.

## Validator Extensions (Hard Constraints)

Apply in both Executor validator and Manager pre-execution checks:

- Price relationship & RR, min confidence, leverage caps (existing).
- Asset-level max leverage: clamp/deny using `AssetMeta.MaxLeverage`.
- Position value bands: BTC/ETH vs altcoins using equity multiples (min/max).
- Margin-usage cap: `(used_margin + new_margin)/equity ≤ max_margin_usage_pct`.
- Liquidity threshold for new opens: `open_interest × price ≥ liquidity_threshold_usd`.
- Cooldown: disallow new opens for `symbol` until `now - RecentlyClosed[symbol] ≥ cooldown_after_close`.
- No hedging/pyramiding: prohibit new opens on symbols with existing positions; closes always allowed.

## Manager Cycle (Proposed Flow)

Per active trader at each tick:

1) Fetch account and positions from the exchange provider.
2) Prewarm/refresh asset directory and build `AssetMeta` (tick/step, max leverage, decimals).
3) Build candidate set and batch-fetch market snapshots (price, EMA/RSI/MACD, OI, funding, 1h/4h changes).
4) Compute or load recent performance; call `Executor.UpdatePerformance(view)`.
5) Build `executorpkg.Context` with the above and call `GetFullDecision`.
6) Sort decisions (close first, then opens), cap new opens per cycle.
7) Execute actions via the orchestrator (see below), logging every step and error class.
8) Record decision timestamp, sync positions, write the decision-cycle log, and update performance metrics.

## Execution Orchestrator (Hyperliquid)

- Precision & step-size: resolve and clamp price/size using asset meta; introduce `FormatPrice(symbol, price)` and reuse existing `FormatSize`.
- Idempotency: generate a stable `cloid = hash(traderID, symbol, side, time_bucket, size_bucket)` and pass with orders; safe replays on temporary failures.
- Open workflow:
  - (Optional) `UpdateLeverage` after checking asset-level caps.
  - Place IOC or protective-limit order with `cloid`.
  - Immediately configure reduce-only trigger orders for SL/TP on the filled quantity.
- Close workflow:
  - Cancel symbol’s resting orders, then close using a small protective slippage limit to avoid rejections.
- Error classification:
  - `Temporary` (network/timeout/5xx) → backoff + jitter (e.g., 200ms base ×2 up to 3 attempts) with idempotency.
  - `RateLimited` → wait `Retry-After` or fixed backoff.
  - `InvalidParam` (precision/step/format) → fail fast; log details.
  - `RiskRejected` (margin/leverage/guard) → fail fast.
  - `Insufficient` (balance/margin) → fail with suggestion to size down.
  - `ExchangeDown` → fail the cycle.
  - `Unknown` → treat once as temporary then classify as unknown.

## Prompting & Performance Gating

- Inject resource budget in the prompt (remaining slots until `max_positions`, `max_new_positions_per_cycle`).
- If recent `Sharpe < sharpe_pause_threshold`, add strict language to pause/slow down and raise `min_confidence`.
- Maintain “close then open” bias in the prompt to minimize overlap and margin spikes.

## Decision-Cycle Logging & Analytics

Introduce a lightweight audit package (or manager-owned module) to write per-cycle JSON records:

- `timestamp`, `trader_id`, `cycle`
- `input_prompt_digest` (or full prompt when enabled), `cot_trace`, `decisions` (array JSON)
- `account_snapshot`, `positions_snapshot`, `candidates`
- `market_snap_digest` (selected fields to keep payload small)
- `actions[]` with `{symbol, action, qty, price, order_id, cloid, result, error_class, error_detail}`
- `success`, `error_message`

Analytics:
- `AnalyzePerformance(lastN)` → win-rate, PnL, Sharpe, trade frequency, rejection rate.
- Feed `VirtualTrader.Performance`, then `Executor.UpdatePerformance` at the next cycle.

## Testing & Acceptance (P0)

- Unit tests
  - Validator: cooldown/liquidity/value-band/margin/asset-cap boundaries.
  - Precision & step-size formatting: zero rejections from tick/step notches.
  - Idempotent replays: same intent yields at most one live order.
- Integration tests
  - Recorded Info/Exchange responses with injected faults to exercise retries and error classes.
  - Decision-cycle logs and analytics produce stable Sharpe and correct pause/slowdown triggers.
- KPIs
  - Rejection rate (precision/params/risk) drops materially.
  - New-open frequency per hour under configured ceilings.
  - Decision-to-order P95 latency observable and stable.

## Rollout & Safety

- Feature flags: multi-decision, SL/TP orchestration, idempotent retries, gating thresholds each toggleable.
- Shadow logging and gating (advise-only) before enforcing hard rejections.
- Rollback: disable features to fall back to single-decision + basic IOC execution.

---

This plan keeps the current architecture intact while tightening the interfaces between Manager, Executor, and the Hyperliquid provider. By sharing the same constraints across prompt, validator, and execution, and by adding idempotent, precise order placement with full-cycle logging, we materially reduce risk and improve traceability without sacrificing iteration speed.
