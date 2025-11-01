# Decision Journal (pkg/journal)

Lightweight utility to persist per‑cycle decision artifacts for analysis and auditing.
It is deliberately minimal: a `Writer` that saves a `CycleRecord` as a JSON file
in a target directory.

## When to use
- You want a durable, human‑readable record of each decision cycle (inputs, actions, outcomes).
- You plan to compute simple analytics (win rate, rejection rate) or feed performance
  snapshots back into prompts.

If the journal is only used by the Manager, consider relocating this package to
`internal/journal` later. Keep the public API small so the storage backend can
change (local files → object storage or DB) without touching callers.

## Concepts
- `CycleRecord`: A compact JSON structure capturing:
  - `timestamp`, `trader_id`, `cycle_number`
  - `prompt_digest` (SHA‑256 of the prompt text; avoids storing the full prompt)
  - `cot_trace` (optional), `decisions_json` (raw model output)
  - `account_snapshot`, `positions_snapshot`, `candidates`
  - `market_snap_digest` (selected fields like price, 1h/4h change, OI, funding)
  - `actions[]`: `{symbol, action, qty, price, order_id?, cloid?, result, error?}`
  - `success`, `error_message`
- `Writer`: Creates timestamped files named like
  `cycle_YYYYMMDD_HHMMSS_00001.json` under the configured directory.

## Quick start
```go
w := journal.NewWriter("journal/trader_t01")
rec := &journal.CycleRecord{
    TraderID: "trader_t01",
    DecisionsJSON: "[]",
    Success: true,
}
path, err := w.WriteCycle(rec)
if err != nil { /* handle */ }
fmt.Println("wrote", path)
```

## Notes
- The package does not enforce retention or rotation. Callers should clean up
  old files as needed.
- The record is intentionally compact. If you need full prompts or full market
  snapshots, add fields conservatively to avoid bloating artifacts.
- For reproducibility, prefer storing digests/IDs in the record and keep raw
  large blobs out of the hot path.

## Roadmap (optional)
- Pluggable backends (local FS / S3 / DB) behind a small interface.
- Derived analytics helpers (Sharpe, frequency, rejection rate) over recent N cycles.
