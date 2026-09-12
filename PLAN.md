# PLAN — 1stcrack Core Engine & CLI (MVP v1.0.0)

**Status:** Done \
**Date:** 2026-09-10 \
**References:**
- `docs/PRD.md` v1.0.0
- `docs/TDD.md`
- Stack: Go 1.27.1+, stdlib `flag` + `text/tabwriter` (no TUI dependencies), SQLite via `modernc.org/sqlite` (Pure Go / CGO-Free), migrations via `//go:embed`

**Locked decisions:**
1. Receipt = **print to stdout + auto-export `.txt`** (single source `GenerateReceipt()`, 32 columns wide).
2. FIFO = transaction + `ORDER BY roasted_at ASC, id ASC`, **without `FOR UPDATE`** (SQLite does not support it).
3. Units: weight as `int64 mg`, money as `int64 IDR`, conversion with `math.Round`.
4. Coverage target near 100% for testable logic, exclusions documented in Plan 07.
5. Documentation standard: Go Documentation Standard — every public and private declaration documented in English, summary starts with the symbol name.

---

## 0. Summary + Execution Order

```
01 Setup → 02 Domain → 03 Database → 04 Repository → 05 Service → 06 CLI → 07 Testing & NFR → 08 Guided Menu
```

| Plan | Depends on | Main output |
|------|-----------|-------------|
| 01 Setup | - | `go.mod`, folder structure, `cmd/1stcrack/main.go`, `receipts/`, `.golangci.yml` |
| 02 Domain | 01 | `internal/domain/*` pure, no DB |
| 03 Database | 01,02 | `internal/database/db.go`, `migrator.go`, `migrations/000001_*.sql`, `000002_*.sql` |
| 04 Repository | 02,03 | `internal/repository/sqlite_*.go` + atomic FIFO |
| 05 Service | 02,04 | `internal/service/*_service.go`, `receipt.go`, `report.go` |
| 06 CLI | 05 | `internal/cli/*` + `main.go` dispatch |
| 07 Testing & NFR | all | `*_test.go`, benchmarks, NFR checklist |
| 08 Guided Menu | 06,07 | `internal/cli/menu*.go` + FR-MENU-01, no arguments = menu |

Execution rule: do not skip. 04 must be green on integration before 05.

---

## Plan 01 — Project Setup

**Goal:** The repo builds with `go build ./...` from scratch.

**File scope:**
- `go.mod`, `go.sum`
- `cmd/1stcrack/main.go`
- Structure `internal/cli|database|domain|repository|service`
- `receipts/.gitkeep`, `.gitignore`, `.golangci.yml`

**Steps:**
1. `go mod init 1stcrack` (go 1.27.1+).
2. `go get modernc.org/sqlite@latest` (the only runtime dependency).
3. Create folders per TDD §1.
4. `main.go`: dispatch `cli.Run()` with the process exit code.
5. `.gitignore`: `*.db*`, `/receipts/*.txt`, `coverage.out`.

**Acceptance:**
- `go mod tidy && go build ./...` succeed.
- The binary needs no external SQL files (see Plan 03).

---

## Plan 02 — Domain Layer

**Goal:** Value objects + entities + sentinel errors, 100% unit-testable, with no DB/CLI imports.

**File scope:**
- `internal/domain/types.go`
- `internal/domain/errors.go`
- `internal/domain/bean.go`
- `internal/domain/product.go`
- `internal/domain/order.go`

**Steps:**
1. `types.go`:
    - `type WeightMg int64`, `type MoneyIDR int64`
    - `GramsToMg(g float64) WeightMg` using `math.Round(g*1000)`
    - `(w WeightMg) ToGrams() float64`, `(m MoneyIDR) String() string` with `Rp` dot separators.
2. `errors.go`: `ErrInsufficientStock`, `ErrInvalidRoastWeight`, `ErrProductNotFound`, `ErrEmptyCart`, `ErrNegativePayment`.
3. `bean.go`: `GreenBean{ID, Name, Origin, Process, StockMg, CostPerKg}`, `RoastBatch{ID, GreenBeanID, GreenWeightMg, RoastedWeightMg, RemainingMg, ShrinkagePct, RoastLevel, RoastedAt}`, `CalcShrinkage(green, roasted WeightMg) (float64, error)`.
4. `product.go`: `Product{ID, Name, Category DRINK/BEAN_RETAIL/BEAN_WHOLESALE, Price, IsActive}`, `Recipe{ProductID, GreenBeanID, RequiredRoastedMg}`.
5. `order.go`: `Order{ID ORD-YYYYMMDD-XXXX, OrderType B2C/B2B, CustomerName, Total, Paid, Method CASH/QRIS/TRANSFER}`, `OrderItem`, `BatchDeduction`.

**Acceptance:**
- `GramsToMg(18.5)=18500`, round-trip loses no precision.
- `CalcShrinkage(1000,850)=15%`, `roasted>green` → `ErrInvalidRoastWeight`, `green=0` → error.

---

## Plan 03 — Database + Embedded Migrations

**Goal:** SQLite WAL + automatic embedded migrations.

**File scope:**
- `internal/database/db.go`
- `internal/database/migrator.go`
- `internal/database/migrations/000001_init_schema.up.sql`
- `internal/database/migrations/000002_seed_data.up.sql`

**Steps:**
1. `db.go`: `sql.Open("sqlite", "file:1stcrack.db")`, set `PRAGMA journal_mode=WAL`, `foreign_keys=ON`, `busy_timeout=5000`, `MaxOpenConns=1` (avoids SQLite locks).
2. `migrator.go`:
    - `//go:embed migrations/*.sql`
    - Table `schema_migrations(version TEXT PRIMARY KEY)`.
    - Read the FS in lexicographic order, each file in its own transaction, record the version, idempotent.
3. `000001_init_schema.up.sql`: 7 tables per TDD §3 (`green_beans`, `roast_batches`, `products`, `product_recipes`, `orders`, `order_items`, `batch_deductions`). Weights as `INTEGER mg`, money as `INTEGER`.
4. `000002_seed_data.up.sql`: seed 2 beans, 3 products, 3 recipes.

**Acceptance:**
- Fresh migration run twice with no error.
- `PRAGMA journal_mode` = `wal`.
- The `go build` binary still migrates without an external SQL folder.

---

## Plan 04 — Repository + Atomic FIFO

**Goal:** DB access + atomic FIFO deduction, 100% rollback on short stock.

**File scope:**
- `internal/repository/sqlite_bean.go`
- `internal/repository/sqlite_product.go`
- `internal/repository/sqlite_order.go`

**Steps:**
1. `sqlite_bean.go`: `CreateGreenBean`, `GetGreenBean`, `UpdateGreenStock` (anti-negative guard), `CreateRoastBatch`, `DeductGreenAndCreateBatch` (atomic), `ListActiveBatches(beanID) ORDER BY roasted_at ASC, id ASC` (`remaining_mg>0`), `ListGreenBeans`, `CountBatchesForDay`.
2. `sqlite_product.go`: `ListActiveProducts`, `GetProduct`, `GetRecipes`.
3. `sqlite_order.go` — the core:
    - `BeginTx`, insert `orders` + `order_items`.
    - Deterministic FIFO loop (sorted beans): deduct `remaining_mg`, insert `batch_deductions`.
    - On shortfall → `tx.Rollback()` + `ErrInsufficientStock`, else `tx.Commit()`.
    - **No `FOR UPDATE`.**
4. Every query takes a `context.Context`.

**Acceptance:**
- Exact single-batch, multi-batch split, drain `remaining=0`.
- Insufficient stock → rollback, `remaining_mg` unchanged.
- `batch_deductions` total = total required.

**FR mapping:** FR-INV-01, FR-INV-02.

---

## Plan 05 — Service Layer

**Goal:** Roast / Order / Inventory / Report business rules + receipt generator.

**File scope:**
- `internal/service/roasting_service.go`
- `internal/service/order_service.go`
- `internal/service/inventory_service.go`
- `internal/service/receipt.go`
- `internal/service/report.go`
- DTOs: `CartItem{ProductID, Quantity}`, `CheckoutRequest{OrderType, CustomerName, Items, PaidAmount, PaymentMethod}`

**Steps:**
1. `roasting_service.go` (FR-ROAST-01..04):
    - Validate `roasted<=green`, `green>0`, valid level, bean exists.
    - `shrinkage = (G-R)/G*100`.
    - Generate `BATCH-YYYYMMDD-XXX` via per-day count (inject `now func() time.Time` for testability).
    - Atomic: decrement `green_beans.stock_mg` + insert `roast_batches`.
2. `order_service.go` (FR-POS-01..03):
    - `CalculateTotal(items)`: check `EmptyCart`, `ProductNotFound`, `qty<=0`, only `is_active=1`.
    - `ProcessCheckout(req)`: check `paid>=total` else `ErrNegativePayment`, B2B requires `customer_name`, expand BOM `required_mg*qty` per `green_bean_id` (generic recipes without beans are skipped), delegate to FIFO repo.
3. `inventory_service.go` (FR-INV-03):
    - `AlertLevelFor(remaining, threshold)` pure: `ok/low/critical`.
    - `ListGreenBeanAlerts`, `ListRoastBatchAlerts`.
4. `receipt.go` (FR-POS-04):
    - `GenerateReceipt(order, lines, change, filePath) string`, 32 chars wide + closing blank line. Pure, no I/O.
5. `report.go` (FR-REP-01..02):
    - `Reporter.SummariseDay` + `SummariseDay(ctx, orders, now)`: revenue, count, per-method, total + per-batch consumption since local midnight.

**Acceptance:**
- Invalid roast → `ErrInvalidRoastWeight`, raw stock decremented exactly.
- Short payment → `ErrNegativePayment`, stock untouched.
- B2B without a name → validation error.

---

## Plan 06 — CLI

**Goal:** Operational subcommands on top of the service layer, stdlib only.

**File scope:**
- `internal/cli/doc.go`
- `internal/cli/cli.go` (`Run`, global flags, dispatch, exit code)
- `internal/cli/weight.go` (`ParseWeight` mg/g/kg, `ParseMoney`)
- `internal/cli/list.go` (`beans`, `products`)
- `internal/cli/roast.go`
- `internal/cli/sell.go` (repeatable `--item`, receipt to stdout + file)
- `internal/cli/stock_report.go`

**Steps:**
1. `Run(ctx, args, stdin, stdout, stderr, dbPath, receiptsDir) int`: scan `--db`/`--receipts-dir` → open + migrate → dispatch → 0 on success / 1 on failure (message to stderr). No arguments opens the guided menu, exit 0.
2. `weight.go`: `ParseWeight("2kg")=2000000`, bare number means grams, rejects non-positive input.
3. `roast --bean ID --green WEIGHT --roasted WEIGHT [--level Medium]`: print batch ID + shrinkage.
4. `sell --item ID:QTY... --paid AMOUNT [--b2b --customer N] [--method CASH]`: build receipt lines from the catalogue → `ProcessCheckout` → write file + print receipt.
5. `stock [--threshold 500g]`: FIFO table (ID/bean/remainder/age/OK-LOW-CRIT) + green bean table via `tabwriter`.
6. `report`: print the `SummariseDay` aggregate.
7. `main.go`: `--version` → print version, else `os.Exit(cli.Run(...))` with `os.Stdin`.

**Acceptance:**
- Demo flow `roast → sell → stock → report` green end-to-end (see §8).
- Bad flags / short stock / short payment → stderr + exit 1, DB unchanged.

**FR mapping:** FR-ROAST-01..04, FR-POS-01..04, FR-INV-01..03, FR-REP-01..02.

---

## Plan 07 — Near-100% Testing + NFR

**Goal:** Public + private APIs covered except the impossible; measured NFRs.

**Principles:**
- Same-package tests (`package service`) for private functions.
- Table-test pure logic (`AlertLevelFor`, `GenerateReceipt`, `ParseWeight`).
- Temporary DB helper per test (`t.TempDir()` + `RunMigrations`).
- Closed-DB tests for I/O error branches.

**Coverage results:** total 88.1% — domain 100%, service 93.1%, repository 90.3%, cli 86.5%, database 79.8%.

**Exclusions (documented, not counted):**
- `cmd/1stcrack/main.go` (thin dispatch).
- `sql.Open`/pragma/ping failures + mid-migration failure (needs fault injection).
- Cable-pulling `power loss` — replaced by WAL + rollback tests.
- `chmod 000`/full-disk receipt — one negative path only.

**Commands:**
```bash
go test ./... -covermode=atomic -coverprofile=coverage.out -count=1
go tool cover -func=coverage.out | sort -k3 -n
```

---

## 8. End-to-End Demo (verified)

```bash
go build -o 1stcrack ./cmd/1stcrack
./1stcrack products
./1stcrack roast --bean GB-GAYO-WASHED --green 2kg --roasted 1700g --level Medium
./1stcrack sell --item P-LATTE-HOT:2 --paid 60000
./1stcrack sell --item P-BEANS-1KG:1 --paid 300000 --b2b --customer "Cafe X"
./1stcrack stock
./1stcrack report
ls receipts/
```

Guided menu session (`1stcrack` with no arguments): banner → select role
1 Cashier (guided sale through receipt) → 9 switch role → 2 Roaster
(record batch) → 9 → 3 Owner (stock + report) → 0 exit, exit 0.

---

## 9. NFR Checklist (measured)

- [x] NFR-PERF: one command takes ~µs–ms, startup <50ms.
- [x] NFR-RELIAB: atomic transactions, WAL, 100% tested rollback.
- [x] NFR-USAB: `1stcrack <command> [flags]`, `--help` per command, `mg/g/kg` weights, 80-column tables; no arguments opens the role-guided menu.
- [x] NFR-RES: ~10MB binary, no persistent process.
- [x] Offline-first, no internet, plain-text output.

---

## 10. Risks

1. SQLite busy lock → mitigate with `MaxOpenConns=1` + `busy_timeout`.
2. Float rounding → mitigate with `int64 mg` + `math.Round`.
3. Same-day ID collision → mitigate with count+tx + unique constraint.
4. Mistyped flags → mitigate with per-command `--help` + stderr error + exit 1.

---

## 11. Licence

MIT on behalf of Dzulkifli Anwar — see `LICENSE`.
