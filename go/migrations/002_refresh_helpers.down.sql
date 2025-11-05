-- Rollback helper materialized views and refresh routine

DROP FUNCTION IF EXISTS refresh_views_nof0();
DROP MATERIALIZED VIEW IF EXISTS v_since_inception;
DROP MATERIALIZED VIEW IF EXISTS v_leaderboard;
DROP MATERIALIZED VIEW IF EXISTS v_crypto_prices_latest;
