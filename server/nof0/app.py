"""
FastAPI application factory mirroring the Go server bootstrap.
"""

from __future__ import annotations

import logging
from contextlib import asynccontextmanager
from pathlib import Path
from typing import AsyncIterator, Optional

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from .api.routes import router
from .config import Config, load_config
from .service import ServiceContext

logger = logging.getLogger(__name__)


def create_app(config: Optional[Config] = None, *, config_path: Optional[str | Path] = None) -> FastAPI:
    cfg = config or load_config(config_path)

    @asynccontextmanager
    async def lifespan(app: FastAPI) -> AsyncIterator[None]:
        service_context = ServiceContext.from_config(cfg)
        app.state.service_context = service_context
        logger.info("Starting server at %s:%d", cfg.host, cfg.port)
        try:
            yield
        finally:
            service_context.close()
            logger.info("Server shutdown complete")

    app = FastAPI(
        title="NOF0 API",
        version="1.0.0",
        description="Python implementation of the NOF0 backend.",
        lifespan=lifespan,
    )
    _configure_cors(app, cfg)
    app.include_router(router)
    return app


def _configure_cors(app: FastAPI, config: Config) -> None:
    cors = config.cors
    app.add_middleware(
        CORSMiddleware,
        allow_origins=cors.allow_origins,
        allow_methods=cors.allow_methods,
        allow_headers=cors.allow_headers,
        expose_headers=cors.expose_headers,
        allow_credentials=cors.allow_credentials,
        max_age=cors.max_age,
    )
