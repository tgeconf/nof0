-- Rollback baseline schema

DROP TABLE IF EXISTS trader_state;
DROP TABLE IF EXISTS market_asset_ctx;
DROP TABLE IF EXISTS market_assets;
DROP TABLE IF EXISTS decision_cycles;
DROP TABLE IF EXISTS conversation_messages;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS model_analytics;
DROP TABLE IF EXISTS trades;
DROP TABLE IF EXISTS positions;
DROP TABLE IF EXISTS account_equity_snapshots;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS price_latest;
DROP TABLE IF EXISTS price_ticks;
DROP TABLE IF EXISTS symbols;
DROP TABLE IF EXISTS models;
