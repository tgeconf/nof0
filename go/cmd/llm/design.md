**Objective**
- Run the trading manager end-to-end from `cmd/llm`, wiring exchange, market, LLM, and manager configurations while constraining trading to BTC/ETH and a fixed 100 USD bankroll.

**Configuration Loading**
- Use flags to override defaults; resolve paths to `etc/exchange.yaml`, `etc/market.yaml`, `etc/llm.yaml`, and `etc/manager.yaml`.
- Call `confkit.LoadDotenvOnce()` so `${VAR}` placeholders in YAML expand from environment variables (e.g., exchange private key, LLM API key).
- Exchange: `pkg/exchange.LoadConfig` validates providers and expands env vars, then `BuildProviders` instantiates registered provider types (`_ "pkg/exchange/hyperliquid"`).
- Market: `pkg/market.LoadConfig` mirrors exchange loading; `BuildProviders` uses `_ "pkg/market/exchanges/hyperliquid"` for registration.
- LLM: `pkg/llm.LoadConfig` applies env overrides and sanity checks; `llm.NewClient` creates an OpenAI-compatible client with optional retries.
- Manager: `pkg/manager.LoadConfig` parses trader/risk settings, validates both manager and executor prompt templates, and now recognizes per-trader LLM model aliases (matching `etc/llm.yaml`).

**BTC/ETH Restriction**
- Parse `--symbols` (default `BTC,ETH`) into an uppercase set; fail fast if the set is empty.
- Wrap each market provider with `filteredMarket`, a light decorator exposing:
  - `Snapshot`: delegates only when the symbol is allowed.
  - `ListAssets`: filters the underlying asset list to the allowed set.
- Register the wrapped providers in the map passed to `manager.NewManager`.
- Optionally extend the wrapper later to surface friendly errors or telemetry when blocked symbols are requested.

**Capital & Risk Overrides**
- Adapt the loaded manager config before constructing the manager:
  - Force `Manager.TotalEquityUSD = 100` and `ReserveEquityPct = 0`.
  - Split `AllocationPct` evenly across all traders; last trader absorbs rounding.
  - Clamp `RiskParams.MaxPositionSizeUSD` to per-trader equity (100 / trader count).
  - Reduce `RiskParams.MaxPositions` and `ExecGuards.CandidateLimit` to `len(allowedSymbols)` (minimum of 1).
  - Re-run `cfg.Validate()` to ensure derived values remain consistent.
- These overrides keep existing YAML intact while enforcing runtime limits suitable for testing.

**Manager Wiring**
- Build an executor factory with `manager.NewBasicExecutorFactory(llmClient)`; it adapts each `TraderConfig` into an `executor.Config`.
- Instantiate the manager: `manager.NewManager(managerCfg, executorFactory, exchangeProviders, filteredMarkets)`.
- For each trader:
  - Call `mgr.RegisterTrader(traderCfg)`; this attaches exchange/market providers, creates an executor (honoring `executor_prompt_template` and `model`), and auto-starts when `AutoStart` is true.
  - Log the registration for observability.
- Consider adding telemetry hooks later (prompt digest, allocation summary).

**Runtime Loop**
- Create a cancellable context and signal handler (`SIGINT`, `SIGTERM`); on signal, cancel the context and invoke `mgr.Stop()` to close the internal stop channel.
- Call `mgr.RunTradingLoop(ctx)`; the loop:
  - Polls active traders every second.
  - Builds an executor context with account, positions, and candidate data (now BTC/ETH only).
  - Renders prompts via `executor.PromptRenderer`, calls the LLM client, validates decisions, then executes via exchange provider.
  - Journals cycles when configured.
- Exit cleanly when the context is cancelled; log completion.

**Operational Notes**
- Provide required secrets through environment variables (.env or shell export) before running the binary.
- Use Hyperliquid testnet credentials for safe testing; set `--market-config` / `--exchange-config` to alternative YAML files if needed.
- Future extensions:
  - Add `--dry-run` flag to bypass order placement.
  - Support dynamic per-trader symbol lists.
  - Emit metrics (e.g., via Prometheus) for decision cadence and execution outcomes.
