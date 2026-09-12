# Technical Design Document (TDD)

**System:** 1stcrack Core Engine & CLI

**Programming Language:** Go (Golang) 1.27.1+

**CLI Architecture:** Subcommands via stdlib `flag` + `text/tabwriter`, with no TUI dependencies. Each invocation opens the DB, runs one operation through the service layer, prints the result, then exits (exit code 0 on success / 1 on failure).

**Database:** Embedded SQLite via `modernc.org/sqlite` (Pure Go / CGO-Free)

**Migration Scheme:** Embedded SQL Files (`//go:embed`)

---

# 1. Folder & Package Structure (*Clean Layered Architecture*)

```
1stcrack/
├── cmd/
│   └── 1stcrack/
│       └── main.go               # CLI subcommand dispatch (exit code)
├── internal/
│   ├── cli/
│   │   ├── cli.go                # Run(), global flags --db/--receipts-dir, dispatch
│   │   ├── weight.go             # ParseWeight (mg/g/kg) & ParseMoney
│   │   ├── list.go               # beans & products subcommands
│   │   ├── roast.go              # roast subcommand
│   │   ├── sell.go               # sell subcommand + receipt export
│   │   ├── stock_report.go       # stock & report subcommands
│   │   ├── menu.go               # Role menu + prompter (stdin/stdout injection)
│   │   ├── menu_cashier.go       # Guided sale flow
│   │   ├── menu_roaster.go       # Guided batch recording flow
│   │   └── menu_owner.go         # Owner superset menu
├── internal/
│   ├── database/
│   │   ├── db.go                 # SQLite connection & WAL mode config
│   │   ├── migrator.go           # Automatic migration runner
│   │   └── migrations/           # SQL migration files
│   │       ├── 000001_init_schema.up.sql
│   │       └── 000002_seed_data.up.sql
│   ├── domain/
│   │   ├── errors.go             # Domain sentinel errors
│   │   ├── types.go              # Value objects (WeightMg, MoneyIDR)
│   │   ├── bean.go               # GreenBean & RoastBatch entities
│   │   ├── product.go            # Product & Recipe (BOM) entities
│   │   └── order.go              # Order & OrderItem entities
│   ├── repository/
│   │   ├── sqlite_bean.go        # GreenBean & RoastBatch CRUD (FIFO query)
│   │   ├── sqlite_order.go       # Atomic Checkout & Stock transactions
│   │   └── sqlite_product.go     # Product catalogue & recipe queries
│   ├── service/
│   │   ├── roasting_service.go   # Shrinkage calculation & batch execution
│   │   ├── order_service.go      # Cart validation, totals, & FIFO deduction
│   │   ├── inventory_service.go  # Stock reports & alerts
│   │   ├── receipt.go            # GenerateReceipt (stdout + file, single source)
│   │   └── report.go             # Reporter & SummariseDay (daily aggregation)
├── docs/
│   ├── PRD.md
│   └── TDD.md
├── LICENSE                       # MIT on behalf of Dzulkifli Anwar
├── go.mod
└── go.sum
```

---

## 2. Technical Design Decisions (*Technical Trade-Offs*)

## A. Unit Representation & Precision

To prevent *floating-point rounding errors* (e.g. `18.5g` becoming `18.499999999g`), all database and business logic calculations use the **`int64`** type:

- **Coffee Weight**
Stored in **Milligrams (`mg`)**.
    - $1 \text{ gram} = 1.000 \text{ mg}$
    - $18,5 \text{ gram} = 18.500 \text{ mg}$
    - $1 \text{ kg} = 1.000.000 \text{ mg}$
- **Currency**
Stored in **Rupiah (`int64`)**.

```go
// internal/domain/types.go
package domain

type WeightMg int64
type MoneyIDR int64

func GramsToMg(g float64) WeightMg {
    return WeightMg(g * 1000.0)
}

func (w WeightMg) ToGrams() float64 {
    return float64(w) / 1000.0
}
```

## B. FIFO Stock Deduction Algorithm

When an order requires `required_mg` of roasted beans:

1. Open a database transaction: `tx, err := db.BeginTx(ctx, nil)`.
2. Query active batches with remaining stock $> 0$, oldest first:
`SELECT id, remaining_mg FROM roast_batches WHERE remaining_mg > 0 AND green_bean_id = ? ORDER BY roasted_at ASC, id ASC;`
(Note: no `FOR UPDATE` — SQLite does not support it; atomicity is guaranteed by the transaction + `MaxOpenConns(1)`.)
3. Iterate (*loop*) over stock deduction:
    - If `batch.remaining_mg >= needed`: deduct from that batch, `needed = 0`, *break*.
    - If `batch.remaining_mg < needed`: empty the batch (`remaining_mg = 0`), subtract `needed -= batch.remaining_mg`, continue to the next batch.
4. Record the deduction history in the `batch_deductions` table for auditing.
5. If all batches are insufficient, `tx.Rollback()` and return `domain.ErrInsufficientStock`.
6. On success, `tx.Commit()`.

## C. CLI Architecture

One binary, many subcommands. There is no state between invocations, so entire bug classes around focus, modals, and key routing cannot occur:

```go
// internal/cli/cli.go
func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, dbPath string, receiptsDir string) int {
    // 1. Scan global flags --db / --receipts-dir
    // 2. database.Open + RunMigrations
    // 3. Dispatch menu | beans | products | roast | sell | stock | report | help
    // 4. Return 0 on success, 1 on failure (message to stderr)
    return 0
}
```

Usage examples:

```
1stcrack                                  # guided menu (roles → flows → exit)
1stcrack menu                             # same as above, explicit
1stcrack roast --bean GB-GAYO-WASHED --green 2kg --roasted 1700g --level Medium
1stcrack sell --item P-LATTE-HOT:2 --paid 60000
1stcrack sell --item P-BEANS-1KG:1 --paid 300000 --b2b --customer "Cafe X"
1stcrack stock [--threshold 500g]
1stcrack report
```

The guided menu (`menu.go`, `menu_cashier.go`, `menu_roaster.go`, `menu_owner.go`)
uses stdlib `bufio` on top of the same services and subcommands (the sale flow
invokes `runSell` with assembled arguments), so behaviour is identical to the
command line. The constructor depends only on `io.Reader`/`io.Writer`
so sessions can be tested with scripted stdin. Running with no arguments opens the menu;
`help` still prints help text.

## D. Database Schema & Embedded Migration

Uses the built-in Go `//go:embed` feature to bundle SQL migration files directly into the binary with no external files.

```go
// internal/database/migrator.go
package database

import (
    "database/sql"
    "embed"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func RunMigrations(db *sql.DB) error {
    // 1. Create the schema_migrations table if missing
    // 2. Read SQL files from migrationFS in order
    // 3. Execute DDL scripts inside transactions
    return nil
}
```

---

# 3. Database Schema (SQLite DDL)

```sql
-- internal/database/migrations/000001_init_schema.up.sql

CREATE TABLE IF NOT EXISTS green_beans (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    origin TEXT NOT NULL,
    process TEXT NOT NULL,               -- Washed, Natural, Honey, etc.
    stock_mg INTEGER NOT NULL DEFAULT 0,  -- Raw stock in milligrams
    cost_per_kg INTEGER NOT NULL,        -- Purchase price in Rupiah
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS roast_batches (
    id TEXT PRIMARY KEY,                 -- BATCH-YYYYMMDD-001
    green_bean_id TEXT NOT NULL,
    green_weight_mg INTEGER NOT NULL,    -- Raw input weight
    roasted_weight_mg INTEGER NOT NULL,  -- Roasted output weight
    remaining_mg INTEGER NOT NULL,       -- Current batch stock remainder
    shrinkage_pct REAL NOT NULL,         -- Weight loss ((G - R) / G) * 100
    roast_level TEXT NOT NULL,           -- Light, Medium, Dark
    roasted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (green_bean_id) REFERENCES green_beans(id)
);

CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,              -- DRINK, BEAN_RETAIL, BEAN_WHOLESALE
    price INTEGER NOT NULL,              -- Selling price in Rupiah
    is_active INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS product_recipes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id TEXT NOT NULL,
    green_bean_id TEXT,                  -- Specific required beans
    required_roasted_mg INTEGER NOT NULL,-- Consumed roasted grams
    FOREIGN KEY (product_id) REFERENCES products(id),
    FOREIGN KEY (green_bean_id) REFERENCES green_beans(id)
);

CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,                 -- ORD-YYYYMMDD-XXXX
    order_type TEXT NOT NULL,            -- B2C, B2B
    customer_name TEXT,
    total_amount INTEGER NOT NULL,
    paid_amount INTEGER NOT NULL,
    payment_method TEXT NOT NULL,        -- CASH, QRIS, TRANSFER
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

---

### 4. Domain Error Definitions & Service Contracts

```go
// internal/domain/errors.go
package domain

import "errors"

var (
    ErrInsufficientStock = errors.New("insufficient coffee stock for this order")
    ErrInvalidRoastWeight = errors.New("roasted weight must not exceed green weight")
    ErrProductNotFound   = errors.New("product not found")
    ErrEmptyCart         = errors.New("shopping cart is empty")
    ErrNegativePayment   = errors.New("paid amount is less than the order total")
)
```

```go
// internal/service/order_service.go
package service

import (
    "context"
    "1stcrack/internal/domain"
)

type CartItem struct {
    ProductID string
    Quantity  int
}

type CheckoutRequest struct {
    OrderType     string // B2C / B2B
    CustomerName  string
    Items         []CartItem
    PaidAmount    domain.MoneyIDR
    PaymentMethod string
}

type OrderService interface {
    CalculateTotal(ctx context.Context, items []CartItem) (domain.MoneyIDR, error)
    ProcessCheckout(ctx context.Context, req CheckoutRequest) (*domain.Order, error)
}
```

---

### 5. Testing Strategy (*Testing Strategy*)

1. **Unit Testing (`_test.go`)**
    - Mathematical `ShrinkagePercentage` calculation tests: verify the weight loss formula across varied inputs.
    - `GramsToMg` and `MgToGrams` conversion tests to guarantee zero lost precision.
    - Pure in-memory FIFO depletion algorithm tests on *memory structs*.
2. **Integration Testing:**
    - Temporary databases (temp files) + full migrations to test `ProcessCheckout` database transaction integrity and each CLI subcommand end-to-end.
    - Simulating mid-execution *insufficient stock* to prove *rollback* runs 100% without changing batch balances.
    - Simulating a closed database to prove I/O failures are reported cleanly via stderr + exit code 1.
