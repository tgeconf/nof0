# Configuration & Assembly Guidelines (go-zero service, pkg modules, cmd tools)

This document defines a three-layer approach to configuration and dependency assembly that works for:
- go-zero HTTP services inside `internal/*`
- reusable modules in `pkg/*`
- standalone executables under `cmd/*`

It balances module autonomy, application composition, and testability.

## Goals
- Modules load and validate their own config independently (unit tests and CLI tools don’t depend on the API service).
- Applications (API service or CLI) compose multiple module configs, construct runtime dependencies, and perform cross-module validation.
- `internal/config` remains minimal: top-level service config structure + simple validation only.

## Architecture Overview
- Module layer (pkg/*): self-contained config and validation; zero dependency on `internal/*`.
- Assembly layer (svc/cmd): wires modules together, builds providers/clients, applies environment-specific defaults, validates cross-references.
- Shared utilities (pkg/confkit): small helpers for consistent file resolution and config loading across all binaries.

```
+----------------------+        +-------------------------+        +-----------------------+
|      pkg/<module>    |  --->  |   internal/svc (API)    |  <-->  |       cmd/<tool>      |
|  - Config, Load, Val |        |  - Compose & build deps |        |  - Compose/build deps |
+----------------------+        +-------------------------+        +-----------------------+
                ^                            ^                               ^
                |                            |                               |
                +----------------------------+-------------------------------+
                                   pkg/confkit
```

## Do / Don’t
- Do: keep `internal/config` limited to top-level service fields and simple validation.
- Do: have each `pkg/*` module expose `type Config`, `LoadConfig(path)`, and `Validate()`.
- Do: construct providers/clients and enforce cross-module rules in `internal/svc` (or in `cmd/*` boot code).
- Don’t: make `internal/config` import or hold pointers to `pkg/*` module Config types.
- Don’t: let `pkg/*` import anything from `internal/*`.

## Module Pattern (pkg/*)
Each reusable module should provide:

- `type Config` containing only the module’s own concerns.
- `func LoadConfig(path string) (*Config, error)` that parses the file and performs field-level validation. No cross-module composition here.
- `func (c *Config) Validate() error` for direct programmatic validation.
- Optional environment defaults via one of:
  - `func ApplyEnvDefaults(c *Config, env string)`; or
  - `func LoadConfigWith(path string, opts ...Option) (*Config, error)` with `WithEnv(env)`.

Unit tests and CLI tools can use these directly without pulling in API service code.

## Application Assembly (API service)
In a go-zero HTTP service, do composition and injection in `internal/svc`:

- Read the main service config (from `internal/config`). It may include fields like `Env`, `DataPath`, and per-module section paths (e.g., `LLM.File`).
- For each section, use the module’s `LoadConfig` to obtain `*Config` values. Use `pkg/confkit.ResolvePath` to resolve relative paths against the main config directory.
- Build runtime dependencies (providers/clients/renderers) from these configs.
- Validate cross-module relationships (e.g., ID mappings between manager traders and exchange/market providers). Fail fast on invalid references.
- Apply environment-specific defaults (e.g., cheaper models for `test`) either through module helpers or via `svc` if truly application-specific.

Example (illustrative):

```go
// in internal/svc/servicecontext.go
base := confkit.BaseDir(mainConfigPath)
llmCfg, _ := llm.LoadConfig(confkit.ResolvePath(base, c.LLM.File))
exCfg, _ := exchange.LoadConfig(confkit.ResolvePath(base, c.Exchange.File))
mkCfg, _ := market.LoadConfig(confkit.ResolvePath(base, c.Market.File))

providers, _ := exCfg.BuildProviders()
marketProviders, _ := mkCfg.BuildProviders()
// wire manager trader -> providers with strict validation
```

## Application Assembly (cmd/* tools)
For single-module tools:
- Parse `--config` (or env var), call the module’s `LoadConfig`, run.

For multi-module tools:
- Option A: accept a main config with per-module section paths, reuse the same assembly routine as the API service.
- Option B: accept multiple `--<module>-config` flags and compose them in a small `boot` package local to the command.

Prefer extracting common assembly code into a tiny `cmd/<tool>/boot` package when it’s command-specific and not shared with the API.

## Shared Utilities (pkg/confkit)
Provide a lightweight helper package used by modules, API, and CLI alike:

- `ResolvePath(base, file string) string`: env expansion + relative-to-main-config resolution.
- `BaseDir(mainPath string) string`: get directory of the main config file.
- `LoadFile[T any](path string, useEnv bool) (*T, error)`: unify `conf.Load` behavior.
- Optional generic section helper:

```go
type Section[T any] struct {
    File  string `json:",optional"`
    Value *T     `json:"-"`
}
func (s *Section[T]) Hydrate(base string, loader func(string) (*T, error)) error {
    if s.File == "" { return nil }
    p := ResolvePath(base, s.File)
    v, err := loader(p)
    if err != nil { return err }
    s.File, s.Value = p, v
    return nil
}
```

These utilities keep path resolution and env overlay consistent across all binaries and tests.

## Internal Config (service)
Keep it minimal. Example shape:

```go
type Config struct {
    rest.RestConf
    Env      string          `json:",default=test"`
    DataPath string          `json:",default=../../mcp/data"`
    Postgres PostgresConf    `json:",optional"`
    Redis    redis.RedisConf `json:",optional"`
    TTL      CacheTTL        `json:",optional"`

    LLM      struct{ File string }      `json:",optional"`
    Executor struct{ File string }      `json:",optional"`
    Manager  struct{ File string }      `json:",optional"`
    Exchange struct{ File string }      `json:",optional"`
    Market   struct{ File string }      `json:",optional"`
}
```

No pointers to module configs here; hydration belongs to assembly.

## Migration Plan (minimal disruption)
1. Trim `internal/config`: remove `*<module>.Config` pointers and the `hydrate/load*` functions; keep simple field validation.
2. Add `pkg/confkit` implementing `ResolvePath`, `BaseDir`, and (optionally) generic `Section[T]`.
3. Move all module-config loading and provider construction into `internal/svc.NewServiceContext`.
4. If modules need env defaults, add `ApplyEnvDefaults` or loader options within each module.
5. Update tests to use module-level `LoadConfig` (and `confkit` helpers for path resolution when needed).
6. For cmd tools, add a small `boot` package per tool or reuse the service assembly pattern as appropriate.

## Compatibility with Tests
- pkg unit tests call `pkg/<module>.LoadConfig("testdata/<module>.yaml")` directly.
- If relative paths are involved, use `confkit.ResolvePath` or set the working directory accordingly.
- Avoid requiring `internal/config` or `internal/svc` in module tests.

## Exceptions
- If a “composed configuration” is unique to a single application and not reusable, keep its hydration in that application’s private `boot` package rather than in `internal/config` or `pkg/*`.

## Quick Checklist
- internal/config: defines structure only; no cross-module imports or hydration.
- internal/svc: builds dependencies, applies env defaults, validates references.
- pkg/*: owns its config types, loader, and validation; no dependency on internal/*.
- cmd/*: composes configs as needed, reusing module loaders and `confkit`.
- confkit: single place for path/env handling and optional generic section hydration.

## Rationale
This separation reduces coupling, improves testability, and allows pkg modules to be reused in unit tests and CLI tools without dragging in the go-zero API’s internals. The API service and cmd tools remain free to compose modules differently while sharing a consistent foundation for configuration parsing and path handling.

