"""
Data loading utilities mirroring the Go implementation.

All methods return plain dictionaries so that the JSON payloads stay
byte-for-byte compatible with the existing API contract.
"""

from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path
from time import time
from typing import Any, Dict, Optional


def _current_millis() -> int:
    return int(time() * 1000)


@dataclass(slots=True)
class DataLoader:
    data_path: Path

    def __post_init__(self) -> None:
        if not isinstance(self.data_path, Path):
            self.data_path = Path(self.data_path)

    def load_crypto_prices(self) -> Dict[str, Any]:
        return self._load_json("crypto-prices.json")

    def load_account_totals(self) -> Dict[str, Any]:
        payload = self._load_json("account-totals.json")
        payload["serverTime"] = _current_millis()
        return payload

    def load_trades(self) -> Dict[str, Any]:
        data = self._load_json("trades.json")
        trades = data.get("trades", [])
        return {"trades": trades, "serverTime": _current_millis()}

    def load_since_inception(self) -> Dict[str, Any]:
        payload = self._load_json("since-inception-values.json")
        payload["serverTime"] = _current_millis()
        return payload

    def load_leaderboard(self) -> Dict[str, Any]:
        return self._load_json("leaderboard.json")

    def load_analytics(self) -> Dict[str, Any]:
        payload = self._load_json("analytics.json")
        payload["serverTime"] = _current_millis()
        return payload

    def load_model_analytics(self, model_id: str) -> Dict[str, Any]:
        filename = f"analytics-{model_id}.json"
        payload: Optional[Dict[str, Any]] = None
        try:
            data = self._load_json(filename)
            analytics = data.get("analytics")
            if analytics is None:
                analytics = data
            if isinstance(analytics, list):
                analytics = analytics[0] if analytics else {}
            payload = {"analytics": analytics}
        except FileNotFoundError:
            payload = None

        if payload is None:
            analytics_payload = self.load_analytics()
            analytics_list = analytics_payload.get("analytics", [])
            for item in analytics_list:
                if isinstance(item, dict) and item.get("model_id") == model_id:
                    payload = {"analytics": item}
                    break

        if payload is None:
            payload = {"analytics": {"model_id": model_id}}

        payload["serverTime"] = _current_millis()
        return payload

    def load_positions(self) -> Dict[str, Any]:
        data = self._load_json("positions.json")
        account_totals = data.get("accountTotals", [])
        return {"accountTotals": account_totals, "serverTime": _current_millis()}

    def load_conversations(self) -> Dict[str, Any]:
        data = self._load_json("conversations.json")
        conversations = data.get("conversations", [])
        return {"conversations": conversations, "serverTime": _current_millis()}

    def _load_json(self, filename: str) -> Dict[str, Any]:
        file_path = self.data_path / filename
        with file_path.open("r", encoding="utf-8") as handle:
            return json.load(handle)
