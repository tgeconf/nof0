-- Helper materialized views and refresh routine for NOF0 storage redesign.

BEGIN;

CREATE MATERIALIZED VIEW IF NOT EXISTS v_crypto_prices_latest AS
SELECT
    pl.provider,
    pl.symbol,
    pl.price,
    pl.ts_ms AS timestamp_ms,
    pl.updated_at
FROM price_latest pl;

CREATE UNIQUE INDEX IF NOT EXISTS idx_v_crypto_prices_latest_symbol
    ON v_crypto_prices_latest(provider, symbol);

DROP MATERIALIZED VIEW IF EXISTS v_leaderboard;

CREATE MATERIALIZED VIEW v_leaderboard AS
WITH latest_snapshot AS (
    SELECT DISTINCT ON (aes.model_id)
        aes.model_id,
        aes.ts_ms,
        aes.dollar_equity,
        aes.realized_pnl,
        aes.total_unrealized_pnl,
        aes.cum_pnl_pct,
        aes.sharpe_ratio,
        aes.since_inception_hourly_marker,
        aes.since_inception_minute_marker
    FROM account_equity_snapshots aes
    ORDER BY aes.model_id, aes.ts_ms DESC
),
trade_stats AS (
    SELECT
        t.model_id,
        COUNT(*) AS num_trades,
        COUNT(*) FILTER (WHERE t.realized_net_pnl > 0) AS num_wins,
        COUNT(*) FILTER (WHERE t.realized_net_pnl <= 0) AS num_losses,
        COALESCE(SUM(GREATEST(t.realized_net_pnl, 0)), 0) AS win_dollars,
        COALESCE(ABS(SUM(LEAST(t.realized_net_pnl, 0))), 0) AS lose_dollars
    FROM trades t
    GROUP BY t.model_id
)
SELECT
    ls.model_id,
    COALESCE(m.display_name, ls.model_id) AS display_name,
    ls.dollar_equity AS equity,
    COALESCE(ts.num_trades, 0) AS num_trades,
    COALESCE(ts.num_wins, 0) AS num_wins,
    COALESCE(ts.num_losses, 0) AS num_losses,
    COALESCE(ts.win_dollars, 0) AS win_dollars,
    COALESCE(ts.lose_dollars, 0) AS lose_dollars,
    COALESCE(ls.cum_pnl_pct, 0) AS return_pct,
    COALESCE(ls.sharpe_ratio, 0) AS sharpe
FROM latest_snapshot ls
LEFT JOIN trade_stats ts ON ts.model_id = ls.model_id
LEFT JOIN models m ON m.id = ls.model_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_v_leaderboard_model
    ON v_leaderboard(model_id);

DROP MATERIALIZED VIEW IF EXISTS v_since_inception;

CREATE MATERIALIZED VIEW v_since_inception AS
WITH initial_equity AS (
    SELECT DISTINCT ON (aes.model_id)
        aes.model_id,
        aes.ts_ms AS inception_ts_ms,
        aes.dollar_equity AS initial_equity
    FROM account_equity_snapshots aes
    ORDER BY aes.model_id, aes.ts_ms ASC
),
invocations AS (
    SELECT
        dc.model_id,
        COUNT(*) AS num_invocations
    FROM decision_cycles dc
    GROUP BY dc.model_id
)
SELECT
    aes.model_id,
    aes.ts_ms AS timestamp_ms,
    CASE
        WHEN ie.initial_equity IS NULL OR ie.initial_equity = 0
            THEN NULL
        ELSE aes.dollar_equity / ie.initial_equity
    END AS nav_since_inception,
    ie.inception_ts_ms AS inception_ts_ms,
    COALESCE(inv.num_invocations, 0) AS num_invocations
FROM account_equity_snapshots aes
JOIN initial_equity ie ON ie.model_id = aes.model_id
LEFT JOIN invocations inv ON inv.model_id = aes.model_id;

CREATE INDEX IF NOT EXISTS idx_v_since_inception_model_ts
    ON v_since_inception(model_id, timestamp_ms DESC);

CREATE OR REPLACE FUNCTION refresh_views_nof0()
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
    REFRESH MATERIALIZED VIEW v_crypto_prices_latest;
    REFRESH MATERIALIZED VIEW v_leaderboard;
    REFRESH MATERIALIZED VIEW v_since_inception;
END;
$$;

COMMIT;
