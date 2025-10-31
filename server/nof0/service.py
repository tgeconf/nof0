"""
Runtime dependencies for request handlers.

The service context mirrors the Go implementation by wiring together the
configuration, data loader, and optional database access layer.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Optional

from psycopg_pool import ConnectionPool

from .config import Config
from .data_loader import DataLoader


@dataclass(slots=True)
class ServiceContext:
    config: Config
    data_loader: DataLoader
    db_pool: Optional[ConnectionPool] = None

    @classmethod
    def from_config(cls, config: Config) -> "ServiceContext":
        data_loader = DataLoader(config.data_dir)
        pool: Optional[ConnectionPool] = None

        if config.postgres.dsn:
            pool = ConnectionPool(
                conninfo=config.postgres.dsn,
                min_size=0,
                max_size=max(config.postgres.max_open, 1),
                timeout=5,
            )

        return cls(config=config, data_loader=data_loader, db_pool=pool)

    def close(self) -> None:
        if self.db_pool:
            self.db_pool.close()
