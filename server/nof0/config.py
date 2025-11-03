"""
Configuration handling for the Python NOF0 backend.

The shape of the configuration mirrors the Go implementation so that the
same YAML files can be reused. Only the fields that are consumed by the
Python code are materialised here; unrecognised fields are preserved and
ignored to maintain forward compatibility.
"""

from __future__ import annotations

from dataclasses import dataclass, field
import os
from pathlib import Path
from typing import Any, Dict, Iterable, Mapping, Optional

import yaml


@dataclass(slots=True)
class PostgresConfig:
    dsn: str = ""
    max_open: int = 10
    max_idle: int = 5

    @classmethod
    def from_mapping(cls, data: Mapping[str, Any] | None) -> "PostgresConfig":
        if not data:
            return cls()
        return cls(
            dsn=str(data.get("DSN", data.get("dsn", ""))),
            max_open=int(data.get("MaxOpen", data.get("max_open", 10))),
            max_idle=int(data.get("MaxIdle", data.get("max_idle", 5))),
        )


@dataclass(slots=True)
class RedisConfig:
    host: str = ""
    port: int = 6379
    type: str = "node"
    password: str = ""
    tls: bool = False

    @classmethod
    def from_mapping(cls, data: Mapping[str, Any] | None) -> "RedisConfig":
        if not data:
            return cls()
        return cls(
            host=str(data.get("Host", data.get("host", ""))),
            port=int(data.get("Port", data.get("port", 6379))),
            type=str(data.get("Type", data.get("type", "node"))),
            password=str(data.get("Pass", data.get("password", ""))),
            tls=bool(data.get("Tls", data.get("tls", False))),
        )


@dataclass(slots=True)
class CacheTTL:
    short: int = 10
    medium: int = 60
    long: int = 300

    @classmethod
    def from_mapping(cls, data: Mapping[str, Any] | None) -> "CacheTTL":
        if not data:
            return cls()
        return cls(
            short=int(data.get("Short", data.get("short", 10))),
            medium=int(data.get("Medium", data.get("medium", 60))),
            long=int(data.get("Long", data.get("long", 300))),
        )


@dataclass(slots=True)
class CorsConfig:
    allow_origins: list[str] = field(default_factory=lambda: ["*"])
    allow_methods: list[str] = field(default_factory=lambda: ["GET", "POST", "PUT", "DELETE", "OPTIONS"])
    allow_headers: list[str] = field(default_factory=lambda: ["Content-Type", "Authorization"])
    expose_headers: list[str] = field(default_factory=lambda: ["Content-Length"])
    allow_credentials: bool = False
    max_age: int = 3600

    @classmethod
    def from_mapping(cls, data: Mapping[str, Any] | None) -> "CorsConfig":
        if not data:
            return cls()

        def _list(value: Any, default: Iterable[str]) -> list[str]:
            if value is None:
                return list(default)
            if isinstance(value, str):
                return [value]
            return list(value)

        return cls(
            allow_origins=_list(data.get("AllowOrigins") or data.get("allow_origins"), ["*"]),
            allow_methods=_list(data.get("AllowMethods") or data.get("allow_methods"), ["GET", "POST", "PUT", "DELETE", "OPTIONS"]),
            allow_headers=_list(data.get("AllowHeaders") or data.get("allow_headers"), ["Content-Type", "Authorization"]),
            expose_headers=_list(data.get("ExposeHeaders") or data.get("expose_headers"), ["Content-Length"]),
            allow_credentials=bool(data.get("AllowCredentials", data.get("allow_credentials", False))),
            max_age=int(data.get("MaxAge", data.get("max_age", 3600))),
        )


@dataclass(slots=True)
class Config:
    name: str = "nof0"
    host: str = "0.0.0.0"
    port: int = 8888
    data_path: Path = Path("../mcp/data")
    postgres: PostgresConfig = field(default_factory=PostgresConfig)
    redis: RedisConfig = field(default_factory=RedisConfig)
    ttl: CacheTTL = field(default_factory=CacheTTL)
    cors: CorsConfig = field(default_factory=CorsConfig)
    extra: Dict[str, Any] = field(default_factory=dict)

    @classmethod
    def from_mapping(cls, data: Mapping[str, Any]) -> "Config":
        postgres = PostgresConfig.from_mapping(data.get("Postgres"))
        redis = RedisConfig.from_mapping(data.get("Redis"))
        ttl = CacheTTL.from_mapping(data.get("TTL"))
        cors = CorsConfig.from_mapping(data.get("Cors"))
        extra = {
            key: value
            for key, value in data.items()
            if key not in {"Name", "Host", "Port", "DataPath", "Postgres", "Redis", "TTL", "Cors"}
        }
        return cls(
            name=str(data.get("Name", data.get("name", "nof0"))),
            host=str(data.get("Host", data.get("host", "0.0.0.0"))),
            port=int(data.get("Port", data.get("port", 8888))),
            data_path=Path(data.get("DataPath", data.get("data_path", "../mcp/data"))),
            postgres=postgres,
            redis=redis,
            ttl=ttl,
            cors=cors,
            extra=extra,
        )

    @property
    def data_dir(self) -> Path:
        return self.data_path


DEFAULT_CONFIG_PATH = Path(__file__).resolve().parent.parent / "etc" / "nof0.yaml"


def load_config(path: Optional[str | Path] = None) -> Config:
    """Load configuration from a YAML file, defaulting to the bundled config."""
    path_to_use: Path
    if path:
        path_to_use = Path(path).expanduser().resolve()
    else:
        env_path = _env_config_path()
        path_to_use = Path(env_path) if env_path else DEFAULT_CONFIG_PATH
        path_to_use = path_to_use.expanduser().resolve()

    try:
        with path_to_use.open("r", encoding="utf-8") as handle:
            data = yaml.safe_load(handle) or {}
    except FileNotFoundError as exc:
        raise FileNotFoundError(f"Configuration file not found at {path_to_use}") from exc
    if not isinstance(data, Mapping):
        raise ValueError(f"Configuration at {path_to_use} is not a mapping")
    return Config.from_mapping(dict(data))


def _env_config_path() -> Optional[str]:
    for key in ("NOF0_CONFIG", "NOF0_CONFIG_PATH"):
        value = os.environ.get(key)
        if value:
            return value
    return None
