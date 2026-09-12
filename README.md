# 1stcrack

[![Go Version](https://img.shields.io/badge/Go-1.27.1+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Coverage](https://img.shields.io/badge/Coverage-88.1%25-brightgreen.svg)](TESTING.md)

1stcrack is a standalone, offline-first operations engine and CLI built specifically for independent micro-roasteries and specialty coffee shops. It manages raw green bean procurement, batch roast production with automated shrinkage calculation, dual-channel retail (B2C) and wholesale (B2B) sales, strict First-In First-Out (FIFO) stock depletion, and end-of-day auditing.

Built with a clean layered architecture in pure Go, 1stcrack compiles into a single portable binary with zero external runtime C libraries and no active network requirements.

---

## Key Engineering Highlights

- **Zero-Floating-Point Precision** \
All inventory weights and monetary amounts use `int64` primitives (`WeightMg` in milligrams, `MoneyIDR` in Indonesian Rupiah) with strict boundary rounding via `math.Round`. This prevents cumulative fractional drift across thousands of transactions.
- **Deterministic FIFO Stock Depletion** \
Sales automatically consume roasted coffee from the oldest available batches (`ORDER BY roasted_at ASC, id ASC`). If an order exceeds a single batch balance, it splits across successive batches. If total stock is insufficient, the entire transaction rolls back cleanly with zero balance mutation.
- **Crash-Resilient Embedded SQLite** \
Operates using Write-Ahead Logging (`PRAGMA journal_mode=WAL`), a five-second busy timeout, and a single-connection pool (`MaxOpenConns(1)`) to guarantee sequential ACID guarantees without `database is locked` panics.
- **Embedded Schema Migrations** \
Database tables and seed catalogs are compiled directly into the binary using Go `//go:embed`, ensuring zero-configuration startup on fresh machines.
- **Decoupled CLI & Guided Role Menu** \
Functions both as a stateless UNIX CLI tool (ideal for scripting and hotkeys) and an interactive, role-guided terminal menu with dependency-injected I/O streams (`io.Reader` and `io.Writer`) for automated testing.

---

## System Architecture

```text
+-----------------------------------------------------------------------+
|                          1stcrack CLI / Terminal                      |
|        (Stateless Subcommands: roast, sell, stock, report, beans)     |
|             (Interactive Guided Menu: Cashier, Roaster, Owner)        |
+-----------------------------------+-----------------------------------+
                                    |
                                    v
+-----------------------------------------------------------------------+
|                           Service Layer                               |
|   - RoastingService: validation, shrinkage math, batch ID generation  |
|   - OrderService: cart calculation, BOM recipe expansion, checkout    |
|   - InventoryService: threshold monitoring (OK / LOW / CRIT)          |
|   - Reporter: daily revenue, payment methods, batch consumption       |
|   - ReceiptGenerator: 32-column thermal layout and text file exporter |
+-----------------------------------+-----------------------------------+
                                    |
                                    v
+-----------------------------------------------------------------------+
|                         Repository Layer                              |
|   - BeanRepository: green bean inventory, roast batch registration    |
|   - ProductRepository: active catalog retrieval, bill-of-materials    |
|   - OrderRepository: atomic multi-table checkout & FIFO deduction     |
+-----------------------------------+-----------------------------------+
                                    |
                                    v
+-----------------------------------------------------------------------+
|                    Database & Storage (Pure Go SQLite)                |
|   - WAL Mode, foreign key constraints, embedded SQL migrations        |
|   - Tables: green_beans, roast_batches, products, product_recipes,    |
|             orders, order_items, batch_deductions                     |
+-----------------------------------------------------------------------+
```

---

## Quick Start

### Prerequisites

- Go 1.27.1 or newer.
- No C compiler (CGO) or external database server required.

### Installation & Compilation

Clone the repository and build the binary:

```bash
git clone https://github.com/ios-community/1stcrack.git
cd 1stcrack
go build -o 1stcrack ./cmd/1stcrack
```

Verify the build:

```bash
./1stcrack --version
./1stcrack help
```

On initial execution, `1stcrack.db` is created automatically in the working directory and migrated to the latest schema version.

---

## End-to-End Walkthrough

### 1. Inspect Initial Catalog
```bash
./1stcrack products
```
```text
ID            NAME              CATEGORY        PRICE
P-BEANS-1KG   Gayo Washed 1kg   BEAN_WHOLESALE  Rp300.000
P-BEANS-250   Gayo Washed 250g  BEAN_RETAIL     Rp85.000
P-LATTE-HOT   Hot Latte         DRINK           Rp25.000
```

### 2. Record Roast Production
Record a roast batch using 2kg of green beans producing 1.7kg of roasted output:
```bash
./1stcrack roast --bean GB-GAYO-WASHED --green 2kg --roasted 1700g --level Medium
```
```text
recorded BATCH-20260910-001: 15.0% shrinkage, 1700g remaining
```

### 3. Process Retail Sale (B2C)
Sell two Hot Lattes (each consumes 18g of roasted beans via recipe BOM):
```bash
./1stcrack sell --item P-LATTE-HOT:2 --paid 60000 --method CASH
```
```text
   1STCRACK - MICRO ROASTERY    
--------------------------------
ORD-20260910-0001            B2C
10-09-2026 13:34            Cash
--------------------------------
Hot Latte x2            Rp50.000
--------------------------------
Total                   Rp50.000
Cash                    Rp60.000
Change                  Rp10.000
--------------------------------
           Thank you!           
File: receipts/ORD-20260910-0001.txt
```

### 4. Process Wholesale Sale (B2B)
Sell one 1kg retail pack to an external partner cafe:
```bash
./1stcrack sell --item P-BEANS-1KG:1 --paid 300000 --b2b --customer "Cafe Merdeka" --method TRANSFER
```

### 5. Check Inventory & FIFO Status
Inspect remaining roast batches and green bean balances against threshold limits:
```bash
./1stcrack stock --threshold 500g
```
```text
BATCH               BEAN         LEFT    AGE  STATUS
BATCH-20260910-001  Gayo Washed  664g    0d   OK

GREEN BEANS
ID                  STOCK        STATUS
GB-GAYO-WASHED      3000g        OK
GB-LINTONG-NATURAL  3000g        OK
```

### 6. Generate Daily Audit Report
```bash
./1stcrack report
```
```text
TODAY  revenue Rp350.000  tx 2
  CASH: Rp50.000
  TRANSFER: Rp300.000
Coffee used: 1036g
  BATCH-20260910-001: 1036g
```

---

## Interactive Role-Guided Menu

Running `1stcrack` without subcommands opens the role-based guided menu:

```bash
./1stcrack
```

```text
=== 1stcrack ===

Select role:
  1. Cashier / Barista
  2. Head Roaster
  3. Owner / Manager
  0. Exit
Select [0-3]: 
```

- **Cashier / Barista** \
Guided shopping cart selection, automatic price summation, payment entry, change calculation, and receipt generation.
- **Head Roaster** \
Green bean selection, weight validation, shrinkage calculation, and batch logging.
- **Owner / Manager** \
Full operational access including stock status, low-stock alerts, and daily revenue recap.

---

## Command Reference

| Command | Arguments / Flags | Description |
|---|---|---|
| `beans` | None | List all registered green beans and current raw stock. |
| `products` | None | List active sellable catalog products with prices. |
| `roast` | `--bean ID --green W --roasted W [--level L]` | Record a roast batch and calculate shrinkage percentage. |
| `sell` | `--item ID:QTY... --paid N [--b2b --customer S] [--method M]` | Process a sale, deduct FIFO stock, and export receipt text file. |
| `stock` | `[--threshold W]` | Show all active roast batches and raw stock with alert status (`OK`/`LOW`/`CRIT`). |
| `report` | None | Print today's sales summary, payment breakdown, and batch consumption. |
| `menu` | None | Open the interactive role-guided menu. |
| `help` | None | Print usage instructions and available subcommands. |

### Global Flags
Global flags must be placed before the subcommand:
- `--db PATH`: Custom SQLite database path (default: `1stcrack.db`).
- `--receipts-dir DIR`: Custom receipts directory (default: `receipts`).

---

## Testing & Verification

The project follows a test-driven approach with high unit and integration coverage across all packages:

```bash
# Run unit and integration tests
go test ./... -v -count=1

# Run tests with race detection and coverage summary
go test ./... -race -covermode=atomic -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### Coverage Breakdown

| Package | Component | Test Coverage |
|---|---|---|
| `internal/domain` | Core entities, value objects, mathematical precision | **100.0%** |
| `internal/service` | Business logic, BOM expansion, reports, receipts | **93.1%** |
| `internal/repository` | Atomic FIFO transactions, SQLite access | **90.3%** |
| `internal/cli` | Subcommand parsing, flags, interactive menu flows | **86.5%** |
| `internal/database` | Connection setup, WAL configuration, migrations | **79.8%** |
| **Total Workspace** | **Complete Engine & CLI Suite** | **88.1%** |

See [TESTING.md](TESTING.md) for full testing protocols, fault injection simulations, and benchmark instructions.

---

## Documentation Index

- [ARCHITECTURE.md](ARCHITECTURE.md): Detailed architectural design, isolation models, and FIFO algorithms.
- [TESTING.md](TESTING.md): Testing strategy, coverage gates, and fault-injection scenarios.
- [CHANGELOG.md](CHANGELOG.md): Historical releases and versioning records.
- [CONTRIBUTING.md](CONTRIBUTING.md): Contribution guidelines and code quality standards.
- [SECURITY.md](SECURITY.md): Security model and private vulnerability reporting process.
- [docs/PRD.md](docs/PRD.md): Product Requirements Document.
- [docs/TDD.md](docs/TDD.md): Technical Design Document.

---

## License

This project is open-source software licensed under the [MIT License](LICENSE).
