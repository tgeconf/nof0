"""
Command-line entrypoint mirroring the original Go bootstrap.
"""

from __future__ import annotations

import argparse
import logging
from pathlib import Path

import uvicorn

from nof0.app import create_app
from nof0.config import load_config


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run the NOF0 API server (Python port).")
    parser.add_argument(
        "-f",
        "--config",
        default=Path(__file__).resolve().parent / "etc" / "nof0.yaml",
        help="Path to configuration file.",
    )
    parser.add_argument(
        "--reload",
        action="store_true",
        help="Enable auto-reload (development only).",
    )
    return parser.parse_args()


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(levelname)s %(name)s %(message)s")

    args = parse_args()
    config = load_config(args.config)
    app = create_app(config)

    uvicorn.run(
        app,
        host=config.host,
        port=config.port,
        reload=args.reload,
        log_level="info",
    )


if __name__ == "__main__":
    main()
