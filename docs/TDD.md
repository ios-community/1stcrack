# Technical Design Document (TDD)

**Sistem:** 1stcrack Core Engine & CLI

**Bahasa Pemrograman:** Go (Golang) 1.27.1+

**Arsitektur CLI:** Subcommand via stdlib `flag` + `text/tabwriter`, tanpa dependensi TUI. Tiap invocasi membuka DB, menjalankan satu operasi via service layer, mencetak hasil, lalu keluar (exit code 0 sukses / 1 gagal).

**Database:** Embedded SQLite via `modernc.org/sqlite` (Pure Go / CGO-Free)

**Skema Migrasi:** Embedded SQL Files (`//go:embed`)

---

# 1. Struktur Folder & Paket (*Clean Layered Architecture*)

```
1stcrack/
├── cmd/
│   └── 1stcrack/
│       └── main.go               # Dispatch subcommand CLI (exit code)
├── internal/
│   ├── cli/
│   │   ├── cli.go                # Run(), global flags --db/--receipts-dir, dispatch
│   │   ├── weight.go             # ParseWeight (mg/g/kg) & ParseMoney
│   │   ├── list.go               # Subcommand beans & products
│   │   ├── roast.go              # Subcommand roast
│   │   ├── sell.go               # Subcommand sell + ekspor struk
│   │   └── stock_report.go       # Subcommand stock & report
├── internal/
│   ├── database/
│   │   ├── db.go                 # SQLite connection & WAL mode config
│   │   ├── migrator.go           # Runner migrasi otomatis
│   │   └── migrations/           # File SQL migrasi
│   │       ├── 000001_init_schema.up.sql
│   │       └── 000002_seed_data.up.sql
│   ├── domain/
│   │   ├── errors.go             # Sentinel domain errors
│   │   ├── types.go              # Value objects (WeightMg, MoneyIDR)
│   │   ├── bean.go               # Entitas GreenBean & RoastBatch
│   │   ├── product.go            # Entitas Product & Recipe (BOM)
│   │   └── order.go              # Entitas Order & OrderItem
│   ├── repository/
│   │   ├── sqlite_bean.go        # CRUD GreenBean & RoastBatch (FIFO query)
│   │   ├── sqlite_order.go       # Transaksi atomik Checkout & Stok
│   │   └── sqlite_product.go     # Query katalog produk & resep
│   ├── service/
│   │   ├── roasting_service.go   # Hitung shrinkage & eksekusi batch
│   │   ├── order_service.go      # Validasi keranjang, kalkulasi, & FIFO deduction
│   │   ├── inventory_service.go  # Laporan stok & alert
│   │   ├── receipt.go            # GenerateReceipt (stdout + file, single source)
│   │   └── report.go             # Reporter & SummariseDay (agregasi harian)
├── docs/
│   ├── PRD.md
│   └── TDD.md
├── LICENSE                       # MIT atas nama Dzulkifli Anwar
├── go.mod
└── go.sum
```

---

## 2. Keputusan Desain Teknis (*Technical Trade-Offs*)

## A. Representasi Satuan & Presisi

Untuk mencegah *floating-point rounding error* (misal: `18.5g` menjadi `18.499999999g`), seluruh perhitungan di database dan logika bisnis menggunakan tipe **`int64`**:

- **Bobot Kopi**
Disimpan dalam satuan **Miligram (`mg`)**.
    - $1 \text{ gram} = 1.000 \text{ mg}$
    - $18,5 \text{ gram} = 18.500 \text{ mg}$
    - $1 \text{ kg} = 1.000.000 \text{ mg}$
- **Mata Uang**
Disimpan dalam satuan **Rupiah (`int64`)**.

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

## B. Algoritma Pemotongan Stok FIFO

Saat pesanan memerlukan sejumlah `required_mg` biji sangrai:

1. Buka transaksi database: `tx, err := db.BeginTx(ctx, nil)`.
2. Query batch aktif dengan sisa stok $> 0$, diurutkan dari tanggal tertua:
`SELECT id, remaining_mg FROM roast_batches WHERE remaining_mg > 0 AND green_bean_id = ? ORDER BY roasted_at ASC, id ASC;`
(Catatan: tanpa `FOR UPDATE` — SQLite tidak mendukungnya; atomicity dijamin transaksi + `MaxOpenConns(1)`.)
3. Lakukan iterasi (*looping*) pemotongan stok:
    - Jika `batch.remaining_mg >= needed`: Potong batch tersebut, `needed = 0`, *break*.
    - Jika `batch.remaining_mg < needed`: Kosongkan batch (`remaining_mg = 0`), kurangi `needed -= batch.remaining_mg`, lanjut ke batch berikutnya.
4. Catat riwayat pemotongan ke tabel `batch_deductions` untuk audit.
5. Jika seluruh batch tidak mencukupi, `tx.Rollback()` dan kembalikan `domain.ErrInsufficientStock`.
6. Jika sukses, `tx.Commit()`.

## C. Arsitektur CLI

Satu binary, banyak subcommand. Tidak ada state antar invocasi sehingga seluruh kelas bug fokus/modal/routing tombol tidak mungkin terjadi:

```go
// internal/cli/cli.go
func Run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer, dbPath string, receiptsDir string) int {
    // 1. Pindai global flags --db / --receipts-dir
    // 2. database.Open + RunMigrations
    // 3. Dispatch beans | products | roast | sell | stock | report | help
    // 4. Kembalikan 0 sukses, 1 gagal (pesan ke stderr)
    return 0
}
```

Contoh pemakaian:

```
1stcrack roast --bean GB-GAYO-WASHED --green 2kg --roasted 1700g --level Medium
1stcrack sell --item P-LATTE-HOT:2 --paid 60000
1stcrack sell --item P-BEANS-1KG:1 --paid 300000 --b2b --customer "Kafe X"
1stcrack stock [--threshold 500g]
1stcrack report
```

## D. Skema Database & Embedded Migration

Menggunakan fitur bawaan Go `//go:embed` untuk membungkus file migrasi SQL langsung ke dalam binary tanpa file eksternal.

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
    // 1. Buat tabel schema_migrations jika belum ada
    // 2. Baca file SQL dari migrationFS secara urut
    // 3. Eksekusi script DDL dalam transaksi
    return nil
}
```

---

# 3. Skema Basis Data (DDL SQLite)

```sql
-- internal/database/migrations/000001_init_schema.up.sql

CREATE TABLE IF NOT EXISTS green_beans (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    origin TEXT NOT NULL,
    process TEXT NOT NULL,               -- Washed, Natural, Honey, etc.
    stock_mg INTEGER NOT NULL DEFAULT 0,  -- Stok mentah dalam miligram
    cost_per_kg INTEGER NOT NULL,        -- Harga beli dalam Rupiah
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS roast_batches (
    id TEXT PRIMARY KEY,                 -- BATCH-YYYYMMDD-001
    green_bean_id TEXT NOT NULL,
    green_weight_mg INTEGER NOT NULL,    -- Berat mentah masuk
    roasted_weight_mg INTEGER NOT NULL,  -- Berat matang keluar
    remaining_mg INTEGER NOT NULL,       -- Sisa stok batch saat ini
    shrinkage_pct REAL NOT NULL,         -- Persentase susut ((G - R) / G) * 100
    roast_level TEXT NOT NULL,           -- Light, Medium, Dark
    roasted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (green_bean_id) REFERENCES green_beans(id)
);

CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,              -- DRINK, BEAN_RETAIL, BEAN_WHOLESALE
    price INTEGER NOT NULL,              -- Harga jual dalam Rupiah
    is_active INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS product_recipes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id TEXT NOT NULL,
    green_bean_id TEXT,                  -- Biji spesifik yang dibutuhkan
    required_roasted_mg INTEGER NOT NULL,-- Gramasi kopi matang yang dikonsumsi
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

### 4. Definisi Domain Error & Kontrak Service

```go
// internal/domain/errors.go
package domain

import "errors"

var (
    ErrInsufficientStock = errors.New("stok kopi tidak mencukupi untuk pesanan ini")
    ErrInvalidRoastWeight = errors.New("berat matang tidak boleh lebih besar dari berat mentah")
    ErrProductNotFound   = errors.New("produk tidak ditemukan")
    ErrEmptyCart         = errors.New("keranjang belanja masih kosong")
    ErrNegativePayment   = errors.New("nominal pembayaran kurang dari total belanja")
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

### 5. Strategi Pengujian (*Testing Strategy*)

1. **Unit Testing (`_test.go`)**
    - Pengujian kalkulasi matematis `ShrinkagePercentage`: Verifikasi rumus susut bobot dengan variasi input.
    - Pengujian konversi `GramsToMg` dan `MgToGrams` untuk memastikan nol presisi yang hilang.
    - Pengujian algoritma deplesi FIFO secara murni di level *memory struct*.
2. **Integration Testing:**
    - Menggunakan database sementara (temp file) + migrasi penuh untuk menguji integritas transaksi database `ProcessCheckout` dan tiap subcommand CLI end-to-end.
    - Mensimulasikan kondisi *insufficient stock* di tengah eksekusi untuk membuktikan *rollback* berjalan 100% tanpa mengubah saldo batch.
    - Mensimulasikan database tertutup untuk membuktikan kegagalan I/O dilaporkan bersih via stderr + exit code 1.