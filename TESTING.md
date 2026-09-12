# Testing & Validation Guide

This document explains the testing strategy, test execution commands, coverage requirements, and fault-injection scenarios implemented in **1stcrack**.

---

## 1. Testing Philosophy

The test suite ensures operational reliability in production environments where financial and stock correctness is critical:

- **Hermetic & Isolated**: Every integration test provisions an ephemeral database in an isolated directory (`t.TempDir()`), executes all migrations, runs operations, and tears down cleanly.
- **Deterministic**: Tests do not rely on external networks, wall-clock timing races, or global mutable state.
- **Fail-Safe Verification**: Comprehensive testing of failure paths, including stock shortfalls, negative payments, missing entities, and closed storage handles.

---

## 2. Test Execution Sequence

Run the complete validation sequence:

```bash
# 1. Format and static analysis checks
go vet ./...

# 2. Execute all unit and integration tests
go test ./... -v -count=1

# 3. Execute tests with race detection and coverage instrumentation
go test ./... -race -covermode=atomic -coverprofile=coverage.out

# 4. Inspect per-function coverage summary
go tool cover -func=coverage.out | sort -k3 -n
```

---

## 3. Specialized Test Scenarios

### A. Atomic FIFO Depletion & Rollback Verification
Location: `internal/repository/repository_test.go` (`TestFIFOExactAndSplit`, `TestFIFOInsufficientRollback`)

- **Multi-Batch Splitting**: Verifies that an order requiring more coffee than the oldest batch drains the oldest batch completely and takes the remainder from the subsequent batch in chronological order.
- **Transactional Rollback on Shortfall**: Tests that requesting stock exceeding the sum of all available batches aborts the transaction. Verifies that no batch balances are altered and no order records are written to disk.

### B. Fault Injection: Closed-Database Recovery
Location: `internal/database/database_test.go`, `internal/repository/errors_test.go`, `internal/service/errors_test.go`

- Verifies that I/O interruptions, closed storage handles, or corrupted files gracefully return explicit errors instead of panicking or leaving dangling memory structures.

### C. Scripted Interactive Menu Testing
Location: `internal/cli/menu_test.go`

- Tests the interactive guided menu using scripted standard input streams (`strings.NewReader`) passed into custom `prompter` instances.
- Exercises user flows for Cashier sales, Roaster batch recording, role transitions, invalid menu selections, and graceful EOF (`Ctrl+D`) shutdowns.

---

## 4. Coverage Target & Verification Gates

The CI pipeline enforces strict test coverage thresholds:

| Package | Lines Covered | Goal | Status |
|---|---|---|---|
| `internal/domain` | 100% | 100% | Met |
| `internal/service` | 93.1% | $\ge 90\%$ | Met |
| `internal/repository` | 90.3% | $\ge 90\%$ | Met |
| `internal/cli` | 86.5% | $\ge 80\%$ | Met |
| `internal/database` | 79.8% | $\ge 75\%$ | Met |
| **Total Workspace** | **88.1%** | **$\ge 85\%$** | **PASSED** |

### Excluded Paths
The following low-level conditions cannot be reliably triggered in software tests without OS-level kernel interception:
- `cmd/1stcrack/main.go`: Standard OS exit dispatch.
- Low-level SQLite driver CGO-free panic handlers.
- Physical disk write exhaustion during receipt file export.

---

## 5. Benchmarking

Measure critical database initialization and receipt generation latency:

```bash
go test ./internal/database -bench=BenchmarkOpenMigrate -benchmem
go test ./internal/service -bench=BenchmarkGenerateReceipt -benchmem
```

Target latency benchmarks:
- Database Open & Migration: $< 5.0\text{ ms}$
- Receipt Text Generation: $< 1.0\ \mu\text{s}$
