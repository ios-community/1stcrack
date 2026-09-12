# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-09-10

### Added
- **Core Domain & Engine**
  - Implemented fixed-point integer types `WeightMg` (milligrams) and `MoneyIDR` (Rupiah) with `math.Round` conversions to eliminate floating-point precision error.
  - Mathematical shrinkage loss percentage calculation via `CalcShrinkage()`.
  - Core domain entities: `GreenBean`, `RoastBatch`, `Product`, `Recipe`, `Order`, `OrderItem`, and `BatchDeduction`.
- **Roast Production Module**
  - Automatic batch identifier generation (`BATCH-YYYYMMDD-XXX`) based on daily production counters.
  - Atomic raw green stock deduction and roast batch creation.
- **Sales & Point of Sale (POS)**
  - Bill-of-Materials (BOM) recipe expansion linking drink orders to roasted bean weight consumption.
  - Retail (B2C) and wholesale (B2B with mandatory customer cafe name) sales support.
  - Cash, QRIS, and Bank Transfer payment validation and change calculation.
  - Fixed-width 32-column thermal receipt formatting with automatic `.txt` file export to `receipts/`.
- **Strict FIFO Inventory Engine**
  - Deterministic oldest-first roast batch consumption (`ORDER BY roasted_at ASC, id ASC`).
  - Automatic multi-batch splitting across consecutive batches.
  - 100% atomic transaction rollback on stock shortfalls with zero balance modification.
  - Inventory threshold monitoring (`OK`, `LOW`, `CRIT` status markers) with configurable thresholds.
- **Shift & Daily Reporting**
  - Daily sales summaries aggregating total revenue, transaction counts, and payment method breakdowns.
  - Detailed coffee consumption audits tracking exact milligram deductions per roast batch.
- **CLI & Interactive Role Menu**
  - Stateless subcommands: `beans`, `products`, `roast`, `sell`, `stock`, `report`, `menu`, `help`.
  - Global flag overrides for custom database paths (`--db`) and receipt export directories (`--receipts-dir`).
  - Interactive role-guided terminal menu with dedicated workflows for Cashiers, Head Roasters, and Managers.
  - Dependency-injected I/O streams (`io.Reader` and `io.Writer`) for automated terminal testing.
- **Embedded Database & Storage**
  - Pure-Go SQLite storage engine (`modernc.org/sqlite`, zero CGO requirement).
  - Write-Ahead Logging (WAL) configuration with 5,000ms busy timeout and single-connection pooling.
  - Automatic embedded schema migrations using Go `//go:embed`.
- **Quality & Verification Suite**
  - Comprehensive unit, integration, and fault-injection test suite achieving 88.1% overall workspace coverage (100% domain coverage).
  - Go Documentation Standard enforcement across all exported and unexported declarations.
