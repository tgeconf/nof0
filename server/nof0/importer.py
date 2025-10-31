"""
Data importer equivalent to the Go `cmd/importer` utility.
"""

from __future__ import annotations

import argparse
import json
import logging
from collections.abc import Iterable
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, Optional, Set

import psycopg

from .data_loader import DataLoader

logger = logging.getLogger(__name__)


@dataclass(slots=True)
class ImportOptions:
    dsn: str
    data_path: Path
    truncate: bool = False


def parse_args(argv: Optional[Iterable[str]] = None) -> ImportOptions:
    parser = argparse.ArgumentParser(description="Import JSON data into Postgres for NOF0.")
    parser.add_argument(
        "--dsn",
        default="postgres://nof0:nof0@localhost:5432/nof0?sslmode=disable",
        help="Postgres connection string.",
    )
    parser.add_argument(
        "--data",
        default=Path(__file__).resolve().parents[2] / "mcp" / "data",
        type=Path,
        help="Path to the MCP data directory.",
    )
    parser.add_argument(
        "--truncate",
        action="store_true",
        help="Truncate destination tables before import.",
    )
    args = parser.parse_args(argv)
    return ImportOptions(dsn=args.dsn, data_path=args.data.resolve(), truncate=args.truncate)


def run_import(options: ImportOptions) -> None:
    logging.basicConfig(level=logging.INFO, format="%(levelname)s %(name)s %(message)s")
    logger.info("connecting to %s", options.dsn)

    loader = DataLoader(options.data_path)
    model_ids: Set[str] = set()
    symbols: Set[str] = set()

    with psycopg.connect(options.dsn) as conn:
        conn.autocommit = True

        if options.truncate:
            _truncate_tables(conn)

        # Crypto prices
        try:
            prices_payload = loader.load_crypto_prices()
            prices = prices_payload.get("prices", {})
            for symbol, payload in prices.items():
                symbols.add(symbol)
                upsert_symbol(conn, symbol)
                upsert_price_latest(conn, symbol, float(payload.get("price", 0.0)), to_ms(payload.get("timestamp")))
            logger.info("imported crypto prices: %d symbols", len(prices))
        except FileNotFoundError:
            logger.warning("skip crypto prices: file missing")

        # Since inception dataset currently skipped, but log for parity
        try:
            loader.load_since_inception()
            logger.info("skip since-inception: source contains summary only")
        except FileNotFoundError:
            logger.warning("skip since-inception: file missing")

        # Trades
        try:
            trades_payload = loader.load_trades()
            for trade in trades_payload.get("trades", []):
                model_id = str(trade.get("model_id", "")).strip()
                if model_id:
                    model_ids.add(model_id)
                    upsert_model(conn, model_id, model_id)
                symbol = str(trade.get("symbol", "")).strip()
                if symbol:
                    symbols.add(symbol)
                    upsert_symbol(conn, symbol)
                entry_ms = to_ms_float(trade.get("entry_time"))
                exit_ms = to_ms_float(trade.get("exit_time"))
                insert_trade(conn, trade, entry_ms, exit_ms)
            logger.info("imported trades: %d", len(trades_payload.get("trades", [])))
        except FileNotFoundError:
            logger.warning("skip trades: file missing")

        # Positions
        try:
            positions_payload = loader.load_positions()
            for model_data in positions_payload.get("accountTotals", []):
                model_id = str(model_data.get("model_id", "")).strip()
                if model_id:
                    model_ids.add(model_id)
                    upsert_model(conn, model_id, model_id)
                positions = model_data.get("positions", {}) or {}
                for symbol, pos in positions.items():
                    symbols.add(symbol)
                    upsert_symbol(conn, symbol)
                    entry_ms = to_ms_float(pos.get("entry_time"))
                    insert_position_open(conn, model_id, symbol, pos, entry_ms)
            logger.info("imported positions: %d models", len(positions_payload.get("accountTotals", [])))
        except FileNotFoundError:
            logger.warning("skip positions: file missing")

        # Analytics (list and raw payload)
        try:
            analytics_payload = loader.load_analytics()
            analytics_list = analytics_payload.get("analytics", [])
            for item in analytics_list:
                model_id = str(item.get("model_id", "")).strip()
                if model_id:
                    model_ids.add(model_id)
                    upsert_model(conn, model_id, model_id)

            raw_data = (options.data_path / "analytics.json").read_text(encoding="utf-8")
            raw_obj = json.loads(raw_data)
            for blob in raw_obj.get("analytics", []):
                model_id = ""
                if isinstance(blob, dict):
                    model_id = str(blob.get("model_id", "")).strip()
                else:
                    try:
                        probe = json.loads(blob)
                        model_id = str(probe.get("model_id", "")).strip()
                        blob = probe
                    except Exception:  # pragma: no cover - defensive
                        model_id = ""
                if model_id:
                    upsert_model_analytics(conn, model_id, blob)
            logger.info("imported analytics payloads: %d", len(analytics_list))
        except FileNotFoundError:
            logger.warning("skip analytics: file missing")

        # Conversations
        try:
            convo_payload = loader.load_conversations()
            for convo in convo_payload.get("conversations", []):
                model_id = str(convo.get("model_id", "")).strip()
                if model_id:
                    model_ids.add(model_id)
                    upsert_model(conn, model_id, model_id)
                conversation_id = insert_conversation(conn, model_id)
                for message in convo.get("messages", []):
                    ts = to_ms(message.get("timestamp"))
                    insert_conversation_message(
                        conn,
                        conversation_id,
                        message.get("role") or "assistant",
                        message.get("content", ""),
                        ts,
                    )
            logger.info("imported conversations: %d", len(convo_payload.get("conversations", [])))
        except FileNotFoundError:
            logger.warning("skip conversations: file missing")

    logger.info(
        "models upserted: %d, symbols upserted: %d",
        len(model_ids),
        len(symbols),
    )
    logger.info("done.")


def _truncate_tables(conn: psycopg.Connection) -> None:
    sql = (
        "TRUNCATE TABLE conversation_messages, conversations, model_analytics, trades, "
        "positions, account_equity_snapshots, accounts, price_ticks, price_latest, symbols, models "
        "RESTART IDENTITY CASCADE"
    )
    conn.execute(sql)
    logger.info("truncated target tables")


def upsert_model(conn: psycopg.Connection, model_id: str, display_name: str) -> None:
    sql = (
        "INSERT INTO models(id, display_name) VALUES (%s,%s) "
        "ON CONFLICT (id) DO UPDATE SET display_name=EXCLUDED.display_name"
    )
    conn.execute(sql, (model_id.strip(), display_name))


def upsert_symbol(conn: psycopg.Connection, symbol: str) -> None:
    sql = "INSERT INTO symbols(symbol) VALUES (%s) ON CONFLICT (symbol) DO NOTHING"
    conn.execute(sql, (symbol.strip(),))


def upsert_price_latest(conn: psycopg.Connection, symbol: str, price: float, timestamp_ms: int) -> None:
    sql = (
        "INSERT INTO price_latest(symbol, price, ts_ms) VALUES (%s,%s,%s) "
        "ON CONFLICT (symbol) DO UPDATE SET price=EXCLUDED.price, ts_ms=EXCLUDED.ts_ms"
    )
    conn.execute(sql, (symbol, price, timestamp_ms))


def insert_trade(conn: psycopg.Connection, trade: Dict[str, Any], entry_ms: int, exit_ms: int) -> None:
    sql = (
        "INSERT INTO trades("
        "id, model_id, symbol, side, trade_type, quantity, leverage, confidence, "
        "entry_price, entry_ts_ms, exit_price, exit_ts_ms, realized_gross_pnl, "
        "realized_net_pnl, total_commission_dollars"
        ") VALUES (%(id)s,%(model_id)s,%(symbol)s,%(side)s,%(trade_type)s,%(quantity)s,"
        "%(leverage)s,%(confidence)s,%(entry_price)s,%(entry_ts_ms)s,%(exit_price)s,"
        "%(exit_ts_ms)s,%(realized_gross_pnl)s,%(realized_net_pnl)s,%(total_commission_dollars)s) "
        "ON CONFLICT (id) DO NOTHING"
    )
    params = {
        "id": trade.get("id"),
        "model_id": trade.get("model_id"),
        "symbol": trade.get("symbol"),
        "side": trade.get("side"),
        "trade_type": null_if_empty(trade.get("trade_type")),
        "quantity": null_float(trade.get("quantity")),
        "leverage": null_float(trade.get("leverage")),
        "confidence": null_float(trade.get("confidence")),
        "entry_price": trade.get("entry_price"),
        "entry_ts_ms": entry_ms,
        "exit_price": trade.get("exit_price"),
        "exit_ts_ms": exit_ms,
        "realized_gross_pnl": trade.get("realized_gross_pnl"),
        "realized_net_pnl": trade.get("realized_net_pnl"),
        "total_commission_dollars": trade.get("total_commission_dollars"),
    }
    conn.execute(sql, params)


def insert_position_open(conn: psycopg.Connection, model_id: str, symbol: str, pos: Dict[str, Any], entry_ms: int) -> None:
    sql = (
        "INSERT INTO positions("
        "id, model_id, symbol, side, entry_price, quantity, leverage, confidence, entry_ts_ms, status"
        ") VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,'open') "
        "ON CONFLICT (id) DO NOTHING"
    )
    identifier = f"{model_id}:{symbol}:{entry_ms}"
    side = "long"
    conn.execute(
        sql,
        (
            identifier,
            model_id,
            symbol,
            side,
            pos.get("entry_price"),
            pos.get("quantity"),
            null_float(pos.get("leverage")),
            null_float(pos.get("confidence")),
            entry_ms,
        ),
    )


def upsert_model_analytics(conn: psycopg.Connection, model_id: str, payload: Any) -> None:
    if not isinstance(payload, str):
        payload = json.dumps(payload)
    sql = (
        "INSERT INTO model_analytics(model_id, payload) VALUES (%s,%s) "
        "ON CONFLICT (model_id) DO UPDATE SET payload=EXCLUDED.payload, updated_at=now()"
    )
    conn.execute(sql, (model_id, payload))


def insert_conversation(conn: psycopg.Connection, model_id: str) -> int:
    sql = "INSERT INTO conversations(model_id) VALUES (%s) RETURNING id"
    with conn.cursor() as cur:
        cur.execute(sql, (model_id,))
        identifier = cur.fetchone()[0]
    return int(identifier)


def insert_conversation_message(
    conn: psycopg.Connection,
    conversation_id: int,
    role: str,
    content: str,
    timestamp_ms: int,
) -> None:
    sql = (
        "INSERT INTO conversation_messages(conversation_id, role, content, ts_ms) "
        "VALUES (%s,%s,%s,%s)"
    )
    conn.execute(sql, (conversation_id, role or "assistant", content, timestamp_ms))


def null_if_empty(value: Any) -> Any:
    if isinstance(value, str) and not value.strip():
        return None
    return value


def null_float(value: Any) -> Optional[float]:
    try:
        number = float(value)
    except (TypeError, ValueError):
        return None
    return None if number == 0 else number


def to_ms(value: Any) -> int:
    if value is None:
        return 0
    if isinstance(value, (int, float)):
        if float(value) < 1e12:
            return int(float(value) * 1000)
        return int(value)
    if isinstance(value, str):
        value = value.strip()
        if not value:
            return 0
        try:
            parsed = float(value)
            if parsed < 1e12:
                return int(parsed * 1000)
            return int(parsed)
        except ValueError:
            try:
                from datetime import datetime

                dt = datetime.fromisoformat(value.replace("Z", "+00:00"))
                return int(dt.timestamp() * 1000)
            except ValueError:
                return 0
    if isinstance(value, dict) and "$numberInt" in value:
        return int(value["$numberInt"])
    return 0


def to_ms_float(value: Any) -> int:
    if value is None:
        return 0
    try:
        number = float(value)
    except (TypeError, ValueError):
        return to_ms(value)
    if number < 1e12:
        return int(number * 1000)
    return int(number)


def main(argv: Optional[Iterable[str]] = None) -> None:
    options = parse_args(argv)
    run_import(options)


if __name__ == "__main__":
    main()
