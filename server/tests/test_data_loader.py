from pathlib import Path

from nof0.data_loader import DataLoader


def _data_dir() -> Path:
    return Path(__file__).resolve().parents[2] / "mcp" / "data"


def test_account_totals_has_server_time() -> None:
    loader = DataLoader(_data_dir())
    payload = loader.load_account_totals()
    assert "accountTotals" in payload
    assert isinstance(payload["serverTime"], int)
    assert payload["serverTime"] > 0


def test_model_analytics_existing_model_returns_payload() -> None:
    loader = DataLoader(_data_dir())
    payload = loader.load_model_analytics("gpt-5")
    assert payload["analytics"]["model_id"] == "gpt-5"


def test_model_analytics_missing_model_returns_placeholder() -> None:
    loader = DataLoader(_data_dir())
    payload = loader.load_model_analytics("non-existent-model")
    assert payload["analytics"]["model_id"] == "non-existent-model"
