# NOF0 Backend – Python Port

This directory contains the canonical Python implementation of the NOF0 backend.  It continues to mirror the legacy Go service so that clients see identical REST endpoints and JSON payloads.

## Highlights
- **FastAPI** re-implements the `/api/*` routes that were previously served via go-zero.
- **Configuration** is shared through `etc/nof0.yaml`, mirroring the Go struct layout so existing config files keep working.
- **Data loader** reads the MCP JSON files just like the Go `DataLoader`.
- **Importer utility** (`nof0/importer.py`) handles Postgres ingestion from the MCP JSON fixtures.

## Getting Started

```bash
cd server
python -m venv .venv
source .venv/bin/activate
pip install -e .
python -m nof0.importer --help
python main.py --reload
```

The server listens on the host/port defined by the YAML config (`etc/nof0.yaml` by default).  All routes are served under the `/api` prefix:

- `GET /api/crypto-prices`
- `GET /api/leaderboard`
- `GET /api/trades`
- `GET /api/account-totals`
- `GET /api/analytics`
- `GET /api/analytics/{modelId}`
- `GET /api/positions`
- `GET /api/conversations`
- `GET /api/since-inception-values`

## Testing

Install the optional development dependencies and run `pytest` inside `server/` once you add tests.  The Go project’s data fixtures remain valid, so the same responses should be observable.
