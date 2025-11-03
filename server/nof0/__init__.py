"""
Python implementation of the NOF0 backend services.

The package mirrors the structure of the original Go backend to ensure
feature parity and predictable behavior for existing clients.
"""

from .app import create_app

__all__ = ["create_app"]
