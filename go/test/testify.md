# Testify Assert Migration TODO List

## Overview
Migrate all unit tests in the `go/pkg` directory from traditional Go testing assertions to testify/assert methods.

## Migration Principles

### Common Migration Patterns

| Original Code | Testify Code |
|---------------|--------------|
| `if err != nil { t.Fatalf("...: %v", err) }` | `assert.NoError(t, err, "...")` |
| `if err == nil { t.Fatal("expected error") }` | `assert.Error(t, err, "expected error")` |
| `if got != want { t.Fatalf("got %v, want %v", got, want) }` | `assert.Equal(t, want, got, "...")` |
| `if got == unexpected { t.Fatalf("unexpected: %v", got) }` | `assert.NotEqual(t, unexpected, got, "...")` |
| `if result == nil { t.Fatal("expected non-nil") }` | `assert.NotNil(t, result, "...")` |
| `if result != nil { t.Fatal("expected nil") }` | `assert.Nil(t, result, "...")` |
| `if !condition { t.Fatal("condition failed") }` | `assert.True(t, condition, "...")` |
| `if condition { t.Fatal("condition should be false") }` | `assert.False(t, condition, "...")` |
| `if len(slice) != n { t.Fatalf("len=%d", len(slice)) }` | `assert.Len(t, slice, n, "...")` |
| `if !strings.Contains(s, substr) { ... }` | `assert.Contains(t, s, substr, "...")` |

### Migration Steps
1. Add import at the top of test files: `"github.com/stretchr/testify/assert"`
2. Replace `t.Fatalf`/`t.Fatal` with corresponding `assert.*` methods
3. Run tests to ensure functionality remains unchanged: `go test ./go/pkg/...`
4. Use meaningful commit messages when committing changes

---

## Migration Task List

### Phase 1: Core Packages

#### 1. executor module
- [ ] `go/pkg/executor/executor_test.go`
  - Migrate `NewExecutor` error checks
  - Migrate `GetFullDecision` result validation
  - Migrate field value comparisons (Action, Symbol, Confidence, etc.)

- [ ] `go/pkg/executor/executor_timeout_test.go`
  - Migrate timeout-related assertions

- [ ] `go/pkg/executor/validator_test.go`
  - Migrate validation logic assertions

- [ ] `go/pkg/executor/config_test.go`
  - Migrate config loading error handling
  - Migrate config field validation

- [ ] `go/pkg/executor/prompt_test.go`
  - Migrate prompt-related assertions

#### 2. manager module
- [ ] `go/pkg/manager/config_test.go`
  - Migrate `LoadConfig` error checks
  - Migrate config field validation (RebalanceInterval, DecisionInterval, etc.)
  - Migrate string trim validation
  - Migrate allocation validation errors
  - Migrate missing prompt/market_provider error checks

- [ ] `go/pkg/manager/prompt_renderer_test.go`
  - Migrate prompt rendering assertions

#### 3. backtest module
- [ ] `go/pkg/backtest/backtest_test.go`
  - Migrate `GetAssetIndex` error checks
  - Migrate `Run` error checks
  - Migrate numerical comparisons (Steps, OrdersSent, etc.)
  - Migrate slice length checks (EquityCurve)
  - Migrate numerical range checks (MaxDDPct, Sharpe)
  - Migrate NaN checks

- [ ] `go/pkg/backtest/feeder_kline_test.go`
  - Migrate kline feeder related assertions

### Phase 2: Exchange & Market

#### 4. exchange module
- [ ] `go/pkg/exchange/config_test.go`
  - Migrate config loading and validation

- [ ] `go/pkg/exchange/hyperliquid/client_test.go`
  - Migrate Hyperliquid client tests
  - Migrate API response validation

- [ ] `go/pkg/exchange/sim/provider_test.go`
  - Migrate simulated exchange tests
  - Migrate order execution validation

#### 5. market module
- [ ] `go/pkg/market/config_test.go`
  - Migrate market config tests

- [ ] `go/pkg/market/config_env_test.go`
  - Migrate environment variable config tests

- [ ] `go/pkg/market/exchanges/hyperliquid/client_test.go`
  - Migrate Hyperliquid market client tests

- [ ] `go/pkg/market/exchanges/hyperliquid/client_recorded_test.go`
  - Migrate recorded test assertions

- [ ] `go/pkg/market/indicators/indicators_test.go`
  - Migrate indicator calculation tests
  - Migrate numerical precision comparisons (may need assert.InDelta)

### Phase 3: Utilities

#### 6. llm module
- [ ] `go/pkg/llm/client_test.go`
  - Migrate LLM client tests
  - Migrate structured response validation

#### 7. prompt module
- [ ] `go/pkg/prompt/template_test.go`
  - Migrate template parsing tests
  - Migrate template rendering validation

---

## Validation Checklist

After completing all migrations, perform the following validation:

- [ ] Run all tests: `go test ./go/pkg/... -v`
- [ ] Check test coverage: `go test ./go/pkg/... -cover`
- [ ] Ensure no remaining `t.Fatalf`/`t.Fatal` (except necessary setup errors)
- [ ] Verify all test files import the `assert` package
- [ ] Code review: ensure assertion messages are clear and meaningful
- [ ] Update CI/CD config (if needed)

---

## Important Notes

1. **Maintain test semantics**: Ensure test intent and behavior remain consistent during migration
2. **Meaningful assertion messages**: Use clear error messages for easier debugging
3. **Special case handling**:
   - Use `assert.InDelta` or `assert.InEpsilon` for floating-point comparisons
   - Complex object comparisons may need `assert.EqualValues`
   - Use `assert.*` for assertions that should continue execution, use `require.*` for immediate termination
4. **Setup errors**: Test setup phase errors can still use `t.Fatalf`, or switch to `require.*`
5. **Batch migration**: Recommended to migrate by module rather than all at once

---

## Progress Tracking

- **Total**: 19 test files
- **Completed**: 0
- **In Progress**: 0
- **Not Started**: 19

### By Module
- executor: 0/5
- manager: 0/2
- backtest: 0/2
- exchange: 0/3
- market: 0/5
- llm: 0/1
- prompt: 0/1

---

## References

- [testify/assert documentation](https://pkg.go.dev/github.com/stretchr/testify/assert)
- [testify/require documentation](https://pkg.go.dev/github.com/stretchr/testify/require)
- [testify GitHub](https://github.com/stretchr/testify)

---

*Last updated: 2025-11-01*
