**Scope & Intent**
- Focus on `internal/config` and packages under `pkg/` (LLM, Executor, Manager, Market, Exchange, Prompt).
- Produce a practical verification plan for early-stage risks: design inconsistency, cannot run, dirty code, and missing implementations.

**Current Modules Snapshot**
- `internal/config`:
  - Loads root config (`etc/nof0.yaml`) and hydrates sub-sections for LLM/Executor/Manager/Exchange/Market.
  - Validates TTLs, required paths, and delegates to each sub-config validator.
- `pkg/llm`:
  - `Config` with `api_key`, `base_url`, `default_model`, retries/timeouts; client with structured output helpers and retry.
- `pkg/executor`:
  - Executes decisions produced by manager via LLM prompts; validates init and inputs. Parsing has TODOs.
- `pkg/manager`:
  - `Config` validates risk params, storage backend/path, monitoring; runtime `Manager` enforces invariants and executes decisions.
- `pkg/market`:
  - Provider registry + `Config` for market data; Hyperliquid provider at `pkg/market/exchanges/hyperliquid` registered via `init()`.
- `pkg/exchange`:
  - Separate “trading/execution” abstraction + `Config`; Hyperliquid implementation in `pkg/exchange/hyperliquid`.
- `pkg/prompt`:
  - Prompt templates and digest utilities.

Note: There are two Hyperliquid stacks (market vs exchange). Keep the split: market = data, exchange = trading. Verify no cross-leak.

**Verification Strategy**
- Unit Tests
  - Table-driven tests for config parsing, env expansion, duration parsing, defaulting and error messages.
  - Pure functions: risk checks, size/price validation, provider directory parsing, LLM request shaping.
  - Fuzz tests for YAML/JSON parsing and text parsers in executor (`Fuzz*`).
- Integration Tests
  - Boot API with sample `etc/*.yaml`, hit routes; verify JSON schemas and golden samples.
  - Use `go-vcr` for external HTTP stability (market provider).
  - 现状：已新增 go-vcr 录制测试（默认回放，缺 cassette 时需 `RECORD_CASSETTES=1` 录制）。
- Simulation (Paper Trading)
  - Add `exchange.Provider` sim backend; feed historical klines to manager/executor end-to-end and validate PnL metrics。
  - 现状：
    - 已提供最小可用 `pkg/exchange/sim`，支持建仓/平仓/杠杆设置、账户状态查询（简化版）。
    - 新增 `pkg/backtest`：
      - Engine（Feeder+Strategy+Exchange）串联模拟盘；支持 `InitialEquity`、`FeeBps`、`SlippageBps`，输出 `EquityCurve`、`Realized/Unreal/Total PnL`、`WinRate`、`MaxDDPct`、`Sharpe`、`Details`（逐笔明细：step/side/price/qty/fee/realized/position）；可选 `OutputPath` 写 JSON 报告。
      - `PriceFeeder`（静态序列）与 `ThresholdStrategy`（阈值触发买卖）作为最小闭环；新增 `CSVKlineFeeder` 支持基于 CSV 的 kline 回放（列：`ts,close`）。
    - 后续：对接更完整的回放器与撮合（滑点/费用/时延/部分成交），并导出交易明细与指标报表。
- Manual Checks
  - Runbook: env, build, start, probe endpoints, common failure injection (timeouts, bad keys).
- AI Review
  - Automated static analysis + LLM triage per module with focus on resource lifecycles, timeouts, error surfaces, and testability.

**Prioritized TODO (Config & pkg)**
- Config
  - Add golden YAML tests.
  - Validate file existence paths.
  - Strengthen TTL bounds.
  - Improve error messages consistency.
  - Document required env vars.
- LLM
  - Add retry/backoff tests.
  - Validate structured output mapping.
  - Timeouts and cancellation tests.
- Executor
  - Implement robust parser (CoT/JSON repair).
  - Add decision validation tests.
  - Enforce `max_concurrent_decisions`.
  - Add prompt-building golden tests.
- Manager
  - Validate risk params edge cases.
  - Test decision lifecycle & error paths.
  - Storage backend interface tests.
- Market
  - Duration/env expansion tests.
  - Symbol directory caching tests.
  - `Snapshot` aggregation correctness tests.
- Exchange
  - Signer init and signing tests.
  - Order validation (price/size) edge cases.
  - Error mapping from HTTP/API.

**Expected Errors & Handling**
- Config Load/Validate
  - Missing `DataPath` or invalid TTLs → Fail fast with actionable message; suggest sample `etc/nof0.yaml`.
  - Bad duration strings (`timeout`, `http_timeout`) → Return exact field name and offending value.
  - Unknown provider IDs (manager references) → List known IDs from `etc/exchange.yaml` / `etc/market.yaml`.
  - Missing env vars (`ZENMUX_API_KEY`, `HYPERLIQUID_PRIVATE_KEY`) → Clear error + pointer to setup.
- LLM Client
  - Empty messages / unsupported streaming → Return clear `llm:`-prefixed errors; recommend fallback to non-stream.
  - Timeout / 5xx → Retries with backoff; surface final error with attempt count and last status.
- Executor
  - Not initialized / nil inputs → Guard clauses with `executor:` prefix; include which field is nil.
  - Parse failure from model output → Return parse error with snippet and advice (enable JSON repair / stricter schema).
- Manager
  - Risk violations (negative size, missing symbol) → Block and emit structured error; no side effect.
  - Missing `executorFactory`/LLM → Startup failure with configuration guidance.
- Market Provider
  - Symbol not found → Map to `ErrSymbolNotFound`; suggest checking case and universe.
  - Directory refresh failure → Retry with jitter; cache last good set; degrade gracefully.
- Exchange Provider
  - Signer not initialized / empty message → Return precise precondition error; never attempt network.
  - Invalid order fields (price/size) → Validation error before sending; include limits.

**Design Inconsistencies To Watch**
- Hyperliquid appears in both `pkg/market/exchanges/hyperliquid` and `pkg/exchange/hyperliquid`.
  - Action: Ensure responsibilities don’t overlap; market only for data, exchange only for trading.
- Config coupling
  - `internal/config` hydrates modules; confirm file paths in `etc/nof0.yaml` match actual sub-configs.
- Error string style
  - Normalize prefixes: `module: message` for grep-ability and API surfacing.

**Dirty Code & Missing Implementations**
- Parser TODO in `pkg/executor/parser.go`.
- Placeholder metrics like `RecentTradesRate` in `pkg/manager/trader.go`.
- Averaging TODO in market data aggregation for Hyperliquid o/i.
- Action: Track these in unit tests as `t.Skip` with issue refs until implemented.

**Runbook: Reproduce & Verify**
- Env
  - Export `ZENMUX_API_KEY`, `HYPERLIQUID_PRIVATE_KEY`.
- Build & Unit
  - `python scripts/run_tests.py unit` (race, coverage, benchmarks summary).
- Integration
  - `python scripts/run_tests.py integration` (build, boot, probe endpoints).
- Manual
  - `go build -o nof0-api ./nof0.go && ./nof0-api -f etc/nof0.yaml`
  - Probe: `/api/crypto-prices`, `/api/leaderboard`, `/api/trades`.

**Acceptance Gates (CI)**
- `go test ./... -race` must pass.
- Coverage for `internal/config` and `pkg/*` ≥ 70% initially; raise later.
- `golangci-lint`, `staticcheck`, `govulncheck`, `gosec` clean or known exceptions.
- Fuzz critical parsers daily (1m) and turn crashes into regression tests.

**Next Steps (Suggested Order)**
- Add golden tests for `internal/config` and `pkg/market` durations/env expansion.
- Add executor decision validation tests; guard rails for nil/empty.
- Introduce `go-vcr` for market provider and record first cassette.
- Scaffold `sim` exchange provider for paper trading. [已完成]
- Add `pkg/backtest` end-to-end backtest harness with price feeder + simple strategy. [已完成]
- Normalize error prefixes and messages across modules.
