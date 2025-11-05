-- NOF0 baseline schema.
-- Creates core Postgres tables needed to migrate off JSON loaders.

CREATE TABLE IF NOT EXISTS models (
    id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    description TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS symbols (
    symbol TEXT PRIMARY KEY,
    base_asset TEXT,
    quote_asset TEXT,
    base_precision INT,
    quote_precision INT,
    tick_size DOUBLE PRECISION,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS price_ticks (
    id BIGSERIAL PRIMARY KEY,
    provider TEXT NOT NULL,
    symbol TEXT NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    ts_ms BIGINT NOT NULL,
    volume DOUBLE PRECISION,
    raw JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_price_ticks_symbol_ts_desc
    ON price_ticks(symbol, ts_ms DESC);

CREATE INDEX IF NOT EXISTS idx_price_ticks_provider_symbol_ts_desc
    ON price_ticks(provider, symbol, ts_ms DESC);

CREATE TABLE IF NOT EXISTS price_latest (
    id BIGSERIAL PRIMARY KEY,
    provider TEXT NOT NULL,
    symbol TEXT NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    ts_ms BIGINT NOT NULL,
    raw JSONB,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, symbol)
);

CREATE INDEX IF NOT EXISTS idx_price_latest_symbol
    ON price_latest(symbol);

CREATE TABLE IF NOT EXISTS accounts (
    model_id TEXT PRIMARY KEY,
    exchange_provider TEXT NOT NULL,
    account_tag TEXT,
    margin_mode TEXT,
    base_currency TEXT,
    leverage_mode TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS account_equity_snapshots (
    id BIGSERIAL PRIMARY KEY,
    model_id TEXT NOT NULL,
    ts_ms BIGINT NOT NULL,
    dollar_equity DOUBLE PRECISION NOT NULL,
    realized_pnl DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_unrealized_pnl DOUBLE PRECISION NOT NULL DEFAULT 0,
    cum_pnl_pct DOUBLE PRECISION,
    sharpe_ratio DOUBLE PRECISION,
    since_inception_hourly_marker INT,
    since_inception_minute_marker INT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (model_id, ts_ms)
);

CREATE INDEX IF NOT EXISTS idx_equity_snapshots_model_ts_desc
    ON account_equity_snapshots(model_id, ts_ms DESC);

CREATE TABLE IF NOT EXISTS positions (
    id TEXT PRIMARY KEY,
    model_id TEXT NOT NULL,
    exchange_provider TEXT NOT NULL,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL DEFAULT 'long' CHECK (side IN ('long', 'short', 'flat')),
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    entry_oid BIGINT,
    risk_usd DOUBLE PRECISION,
    confidence DOUBLE PRECISION,
    index_col JSONB,
    exit_plan JSONB,
    entry_time_ms BIGINT NOT NULL,
    entry_price DOUBLE PRECISION NOT NULL,
    tp_oid BIGINT,
    margin DOUBLE PRECISION,
    wait_for_fill BOOLEAN NOT NULL DEFAULT FALSE,
    sl_oid BIGINT,
    current_price DOUBLE PRECISION,
    closed_pnl DOUBLE PRECISION,
    liquidation_price DOUBLE PRECISION,
    commission DOUBLE PRECISION,
    leverage DOUBLE PRECISION,
    slippage DOUBLE PRECISION,
    quantity DOUBLE PRECISION NOT NULL,
    unrealized_pnl DOUBLE PRECISION,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_positions_model
    ON positions(model_id);

CREATE INDEX IF NOT EXISTS idx_positions_symbol
    ON positions(symbol);

CREATE INDEX IF NOT EXISTS idx_positions_model_exchange_open
    ON positions(model_id, exchange_provider)
    WHERE status = 'open';

CREATE TABLE IF NOT EXISTS trades (
    id TEXT PRIMARY KEY,
    model_id TEXT NOT NULL,
    exchange_provider TEXT NOT NULL,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    trade_type TEXT,
    trade_id TEXT,
    quantity DOUBLE PRECISION,
    leverage DOUBLE PRECISION,
    confidence DOUBLE PRECISION,
    entry_price DOUBLE PRECISION,
    entry_ts_ms BIGINT NOT NULL,
    entry_human_time TEXT,
    entry_sz DOUBLE PRECISION,
    entry_tid BIGINT,
    entry_oid BIGINT,
    entry_crossed BOOLEAN NOT NULL DEFAULT FALSE,
    entry_liquidation JSONB,
    entry_commission_dollars DOUBLE PRECISION,
    entry_closed_pnl DOUBLE PRECISION,
    exit_price DOUBLE PRECISION,
    exit_ts_ms BIGINT,
    exit_human_time TEXT,
    exit_sz DOUBLE PRECISION,
    exit_tid BIGINT,
    exit_oid BIGINT,
    exit_crossed BOOLEAN,
    exit_liquidation JSONB,
    exit_commission_dollars DOUBLE PRECISION,
    exit_closed_pnl DOUBLE PRECISION,
    exit_plan JSONB,
    realized_gross_pnl DOUBLE PRECISION,
    realized_net_pnl DOUBLE PRECISION,
    total_commission_dollars DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trades_model_entry_ts_desc
    ON trades(model_id, entry_ts_ms DESC);

CREATE INDEX IF NOT EXISTS idx_trades_exit_oid
    ON trades(exit_oid);

CREATE TABLE IF NOT EXISTS model_analytics (
    model_id TEXT PRIMARY KEY,
    payload JSONB NOT NULL,
    server_time_ms BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS conversations (
    id BIGSERIAL PRIMARY KEY,
    model_id TEXT NOT NULL,
    topic TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conversations_model
    ON conversations(model_id);

CREATE TABLE IF NOT EXISTS conversation_messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant')),
    content TEXT NOT NULL,
    ts_ms BIGINT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conversation_messages_conv_ts
    ON conversation_messages(conversation_id, ts_ms);

CREATE TABLE IF NOT EXISTS decision_cycles (
    id BIGSERIAL PRIMARY KEY,
    model_id TEXT NOT NULL,
    cycle_number INT,
    prompt_digest TEXT,
    cot_trace TEXT,
    decisions JSONB,
    success BOOLEAN NOT NULL DEFAULT FALSE,
    error_message TEXT,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_decision_cycles_model_executed_at_desc
    ON decision_cycles(model_id, executed_at DESC);

CREATE TABLE IF NOT EXISTS market_assets (
    id BIGSERIAL PRIMARY KEY,
    provider TEXT NOT NULL,
    symbol TEXT NOT NULL,
    name TEXT,
    sz_decimals INT,
    max_leverage DOUBLE PRECISION,
    only_isolated BOOLEAN,
    margin_table_id INT,
    is_delisted BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, symbol)
);

CREATE TABLE IF NOT EXISTS market_asset_ctx (
    id BIGSERIAL PRIMARY KEY,
    provider TEXT NOT NULL,
    symbol TEXT NOT NULL,
    funding DOUBLE PRECISION,
    open_interest DOUBLE PRECISION,
    oracle_px DOUBLE PRECISION,
    mark_px DOUBLE PRECISION,
    mid_px DOUBLE PRECISION,
    impact_pxs JSONB,
    prev_day_px DOUBLE PRECISION,
    day_ntl_vlm DOUBLE PRECISION,
    day_base_vlm DOUBLE PRECISION,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, symbol)
);

CREATE TABLE IF NOT EXISTS trader_state (
    trader_id TEXT PRIMARY KEY,
    exchange_provider TEXT NOT NULL,
    market_provider TEXT NOT NULL,
    allocation_pct DOUBLE PRECISION,
    cooldown JSONB NOT NULL DEFAULT '{}'::jsonb,
    risk_guards JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_decision_at TIMESTAMPTZ,
    pause_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
