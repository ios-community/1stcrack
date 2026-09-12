# Architecture & Technical Design Specification

This document details the architectural decisions, module boundaries, data integrity rules, and database concurrency models implemented in **1stcrack**.

---

## 1. Architectural Style: Clean Layered Architecture

1stcrack strictly separates business rules from database access and user interfaces. Dependencies flow inward toward domain models:

```text
[ CLI & Menu Layer ] (internal/cli)
         |
         v
[ Service Layer ] (internal/service)
         |
         v
[ Repository Layer ] (internal/repository)
         |
         v
[ Database Layer ] (internal/database) <---+
         |                                 |
         +---------------------------------+
         |
         v
[ Domain Layer ] (internal/domain: Entities, Value Objects, Errors)
```

### Module Responsibilities

1. **`internal/domain`**: Pure business models with zero third-party dependencies.
   - Types: `WeightMg` (int64), `MoneyIDR` (int64).
   - Entities: `GreenBean`, `RoastBatch`, `Product`, `Recipe`, `Order`, `OrderItem`, `BatchDeduction`.
   - Sentinel Errors: `ErrInsufficientStock`, `ErrInvalidRoastWeight`, `ErrProductNotFound`, `ErrEmptyCart`, `ErrNegativePayment`.
2. **`internal/database`**: Embedded SQLite lifecycle and schema migration management via `//go:embed`.
3. **`internal/repository`**: Data persistence and atomic transactional queries across database tables.
4. **`internal/service`**: Use-case orchestration, bill-of-materials expansion, shrinkage calculations, receipt formatting, and aggregate reporting.
5. **`internal/cli`**: Stateless command-line flag handling and interactive terminal session prompting.

---

## 2. Technical Invariants & Critical Decisions

### A. Zero-Float Integer Arithmetic
Floating-point numbers in standard computing introduce precision truncation (such as `18.5 - 18.0 = 0.5000000000000004`). In a roastery managing inventory and financial audits, fractional drift causes stock discrepancies.

- **Weight**: Stored as **milligrams (`int64`)**.
  - 1 gram = 1,000 mg
  - 18.5 grams = 18,500 mg
  - 1 kilogram = 1,000,000 mg
- **Currency**: Stored as **Indonesian Rupiah (`int64`)**.
- **Conversion Boundary**: Conversions from human float inputs occur strictly at input boundaries using `math.Round`:
  ```go
  func GramsToMg(g float64) WeightMg {
      return WeightMg(math.Round(g * 1000.0))
  }
  ```

### B. Strict FIFO Stock Depletion Algorithm
To preserve specialty coffee freshness, orders must deplete roasted coffee from the oldest available batches. Because SQLite does not support `SELECT ... FOR UPDATE`, concurrency safety and atomicity are achieved via database transactions and single-connection serialization:

1. **Transaction Initialization**: Begin a database transaction (`db.BeginTx(ctx, nil)`).
2. **Batch Query**: Query active roast batches with remaining stock greater than zero, ordered deterministically:
   ```sql
   SELECT id, remaining_mg 
   FROM roast_batches 
   WHERE green_bean_id = ? AND remaining_mg > 0 
   ORDER BY roasted_at ASC, id ASC;
   ```
3. **Iterative Depletion Loop**:
   - If `batch.remaining_mg >= remaining_needed`: Deduct the required weight from this batch, insert a `batch_deductions` audit record, and break the loop.
   - If `batch.remaining_mg < remaining_needed`: Deduct all available stock (`remaining_mg = 0`), insert a partial `batch_deductions` record, subtract `batch.remaining_mg` from `remaining_needed`, and proceed to the next batch.
4. **Shortfall Check**: If all active batches are exhausted and `remaining_needed > 0`, the transaction executes `tx.Rollback()` and returns `domain.ErrInsufficientStock`.
5. **Commit**: On full fulfillment, `tx.Commit()` applies all mutations atomically.

### C. Database Concurrency & Crash Resilience
SQLite in a multi-invocation CLI environment requires specific pragma tuning to prevent locking collisions and disk corruption:

- **`PRAGMA journal_mode=WAL;`**: Enables Write-Ahead Logging for concurrent readers and atomic write commits.
- **`PRAGMA foreign_keys=ON;`**: Enforces referential integrity at the database engine level.
- **`PRAGMA busy_timeout=5000;`**: Sets a 5,000 ms busy handler to wait for transient locks rather than failing immediately.
- **`PRAGMA synchronous=NORMAL;`**: Balances durability with disk write performance in WAL mode.
- **`db.SetMaxOpenConns(1)`**: Restricts the connection pool to a single connection per CLI process, eliminating intra-process locking deadlocks.

### D. Single-Source Receipt Rendering
Receipt formatting is defined in `internal/service/receipt.go` via `GenerateReceipt()`. It formats a 32-column fixed-width text layout suitable for standard 58mm thermal POS printers. The terminal output and the exported `.txt` file in `receipts/` use the exact same pure generator function to guarantee consistency.

---

## 3. Database Schema Specification

```sql
CREATE TABLE IF NOT EXISTS green_beans (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    origin TEXT NOT NULL,
    process TEXT NOT NULL,
    stock_mg INTEGER NOT NULL DEFAULT 0,
    cost_per_kg INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS roast_batches (
    id TEXT PRIMARY KEY,
    green_bean_id TEXT NOT NULL,
    green_weight_mg INTEGER NOT NULL,
    roasted_weight_mg INTEGER NOT NULL,
    remaining_mg INTEGER NOT NULL,
    shrinkage_pct REAL NOT NULL,
    roast_level TEXT NOT NULL,
    roasted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (green_bean_id) REFERENCES green_beans(id)
);

CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price INTEGER NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS product_recipes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id TEXT NOT NULL,
    green_bean_id TEXT,
    required_roasted_mg INTEGER NOT NULL,
    FOREIGN KEY (product_id) REFERENCES products(id),
    FOREIGN KEY (green_bean_id) REFERENCES green_beans(id)
);

CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    order_type TEXT NOT NULL,
    customer_name TEXT,
    total_amount INTEGER NOT NULL,
    paid_amount INTEGER NOT NULL,
    payment_method TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS order_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL,
    product_id TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    subtotal INTEGER NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
);

CREATE TABLE IF NOT EXISTS batch_deductions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL,
    roast_batch_id TEXT NOT NULL,
    deducted_mg INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (roast_batch_id) REFERENCES roast_batches(id)
);
```
