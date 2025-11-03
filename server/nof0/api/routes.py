"""
HTTP route definitions for the NOF0 API.
"""

from __future__ import annotations

import logging
from typing import Callable, Dict, Optional

from fastapi import APIRouter, Depends, HTTPException, Path, Query, Request

from ..service import ServiceContext

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api")


def get_service_ctx(request: Request) -> ServiceContext:
    ctx: ServiceContext | None = getattr(request.app.state, "service_context", None)
    if ctx is None:
        raise RuntimeError("Service context is not initialised")
    return ctx


def _call_loader(loader: Callable[[], Dict], error_message: str) -> Dict:
    try:
        return loader()
    except FileNotFoundError as exc:
        logger.exception("%s (missing file)", error_message)
        raise HTTPException(status_code=500, detail=error_message) from exc
    except Exception as exc:  # pragma: no cover - defensive
        logger.exception("%s", error_message)
        raise HTTPException(status_code=500, detail=error_message) from exc


@router.get("/account-totals")
def account_totals(
    _: Optional[int] = Query(None, alias="lastHourlyMarker"),
    svc: ServiceContext = Depends(get_service_ctx),
) -> Dict:
    return _call_loader(svc.data_loader.load_account_totals, "failed to load account totals")


@router.get("/analytics")
def analytics(svc: ServiceContext = Depends(get_service_ctx)) -> Dict:
    return _call_loader(svc.data_loader.load_analytics, "failed to load analytics overview")


@router.get("/analytics/{model_id}")
def model_analytics(
    model_id: str = Path(..., description="Model identifier"),
    svc: ServiceContext = Depends(get_service_ctx),
) -> Dict:
    return _call_loader(lambda: svc.data_loader.load_model_analytics(model_id), "failed to load model analytics")


@router.get("/crypto-prices")
def crypto_prices(svc: ServiceContext = Depends(get_service_ctx)) -> Dict:
    return _call_loader(svc.data_loader.load_crypto_prices, "failed to load crypto prices")


@router.get("/leaderboard")
def leaderboard(svc: ServiceContext = Depends(get_service_ctx)) -> Dict:
    return _call_loader(svc.data_loader.load_leaderboard, "failed to load leaderboard")


@router.get("/since-inception-values")
def since_inception(svc: ServiceContext = Depends(get_service_ctx)) -> Dict:
    return _call_loader(svc.data_loader.load_since_inception, "failed to load since-inception values")


@router.get("/trades")
def trades(svc: ServiceContext = Depends(get_service_ctx)) -> Dict:
    return _call_loader(svc.data_loader.load_trades, "failed to load trades")


@router.get("/positions")
def positions(svc: ServiceContext = Depends(get_service_ctx)) -> Dict:
    return _call_loader(svc.data_loader.load_positions, "failed to load positions")


@router.get("/conversations")
def conversations(svc: ServiceContext = Depends(get_service_ctx)) -> Dict:
    return _call_loader(svc.data_loader.load_conversations, "failed to load conversations")
