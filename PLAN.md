# PLAN — 1stcrack Core Engine & CLI (MVP v1.0.0)

**Status:** Selesai
**Tanggal:** 2026-09-10
**Referensi:**
- `docs/PRD.md` v1.0.0
- `docs/TDD.md`
- Stack: Go 1.27.1+, stdlib `flag` + `text/tabwriter` (tanpa dep TUI), SQLite via `modernc.org/sqlite` (Pure Go / CGO-Free), migrasi `//go:embed`

**Keputusan terkunci:**
1. Receipt = **cetak stdout + auto-export `.txt`** (single source `GenerateReceipt()`, lebar 32 kolom).
2. FIFO = transaksi + `ORDER BY roasted_at ASC, id ASC`, **tanpa `FOR UPDATE`** (SQLite tidak mendukungnya).
3. Satuan: bobot `int64 mg`, uang `int64 IDR`, konversi pakai `math.Round`.
4. Coverage target mendekati 100% untuk logic testable, exclusion didokumentasikan di Plan 07.
5. Standar dokumentasi: Go Documentation Standard — setiap deklarasi publik maupun privat berdokumen Inggris, summary mulai nama simbol.

---

## 0. Ringkasan + Urutan Eksekusi

```
01 Setup → 02 Domain → 03 Database → 04 Repository → 05 Service → 06 CLI → 07 Testing & NFR
```

| Plan | Ketergantungan | Output utama |
|------|----------------|--------------|
| 01 Setup | - | `go.mod`, struktur folder, `cmd/1stcrack/main.go`, `receipts/`, `.golangci.yml` |
| 02 Domain | 01 | `internal/domain/*` murni tanpa DB |
| 03 Database | 01,02 | `internal/database/db.go`, `migrator.go`, `migrations/000001_*.sql`, `000002_*.sql` |
| 04 Repository | 02,03 | `internal/repository/sqlite_*.go` + FIFO atomik |
| 05 Service | 02,04 | `internal/service/*_service.go`, `receipt.go`, `report.go` |
| 06 CLI | 05 | `internal/cli/*` + dispatch `main.go` |
| 07 Testing & NFR | semua | `*_test.go`, benchmark, checklist NFR |

Aturan eksekusi: jangan loncat. 04 harus hijau integrasi sebelum 05.

---

## Plan 01 — Setup Proyek

**Tujuan:** Repo bisa `go build ./...` dari nol.

**Scope file:**
- `go.mod`, `go.sum`
- `cmd/1stcrack/main.go`
- Struktur `internal/cli|database|domain|repository|service`
- `receipts/.gitkeep`, `.gitignore`, `.golangci.yml`

**Tahap:**
1. `go mod init 1stcrack` (go 1.27.1+).
2. `go get modernc.org/sqlite@latest` (satu-satunya dependensi runtime).
3. Buat folder sesuai TDD §1.
4. `main.go`: dispatch `cli.Run()` dengan exit code proses.
5. `.gitignore`: `*.db*`, `/receipts/*.txt`, `coverage.out`.

**Acceptance:**
- `go mod tidy && go build ./...` sukses.
- Binary tidak butuh file SQL eksternal (cek di Plan 03).

---

## Plan 02 — Domain Layer

**Tujuan:** Value object + entity + sentinel error, 100% unit-testable, tanpa import DB/CLI.

**Scope file:**
- `internal/domain/types.go`
- `internal/domain/errors.go`
- `internal/domain/bean.go`
- `internal/domain/product.go`
- `internal/domain/order.go`

**Tahap:**
1. `types.go`:
    - `type WeightMg int64`, `type MoneyIDR int64`
    - `GramsToMg(g float64) WeightMg` pakai `math.Round(g*1000)`
    - `(w WeightMg) ToGrams() float64`, `(m MoneyIDR) String() string` format `Rp` pemisah titik.
2. `errors.go`: `ErrInsufficientStock`, `ErrInvalidRoastWeight`, `ErrProductNotFound`, `ErrEmptyCart`, `ErrNegativePayment`.
3. `bean.go`: `GreenBean{ID, Name, Origin, Process, StockMg, CostPerKg}`, `RoastBatch{ID, GreenBeanID, GreenWeightMg, RoastedWeightMg, RemainingMg, ShrinkagePct, RoastLevel, RoastedAt}`, `CalcShrinkage(green, roasted WeightMg) (float64, error)`.
4. `product.go`: `Product{ID, Name, Category DRINK/BEAN_RETAIL/BEAN_WHOLESALE, Price, IsActive}`, `Recipe{ProductID, GreenBeanID, RequiredRoastedMg}`.
5. `order.go`: `Order{ID ORD-YYYYMMDD-XXXX, OrderType B2C/B2B, CustomerName, Total, Paid, Method CASH/QRIS/TRANSFER}`, `OrderItem`, `BatchDeduction`.

**Acceptance:**
- `GramsToMg(18.5)=18500`, round-trip tidak hilang presisi.
- `CalcShrinkage(1000,850)=15%`, `roasted>green` → `ErrInvalidRoastWeight`, `green=0` → error.

---

## Plan 03 — Database + Migrasi Embed

**Tujuan:** SQLite WAL + migrasi otomatis embedded.

**Scope file:**
- `internal/database/db.go`
- `internal/database/migrator.go`
- `internal/database/migrations/000001_init_schema.up.sql`
- `internal/database/migrations/000002_seed_data.up.sql`

**Tahap:**
1. `db.go`: `sql.Open("sqlite", "file:1stcrack.db")`, set `PRAGMA journal_mode=WAL`, `foreign_keys=ON`, `busy_timeout=5000`, `MaxOpenConns=1` (hindari lock SQLite).
2. `migrator.go`:
    - `//go:embed migrations/*.sql`
    - Tabel `schema_migrations(version TEXT PRIMARY KEY)`.
    - Baca FS urut leksikografis, tiap file dalam transaksi sendiri, catat versi, idempotent.
3. `000001_init_schema.up.sql`: 7 tabel sesuai TDD §3 (`green_beans`, `roast_batches`, `products`, `product_recipes`, `orders`, `order_items`, `batch_deductions`). Bobot `INTEGER mg`, uang `INTEGER`.
4. `000002_seed_data.up.sql`: seed 2 beans, 3 products, 3 recipes.

**Acceptance:**
- Fresh run migrasi 2x tanpa error.
- `PRAGMA journal_mode` = `wal`.
- Binary hasil `go build` tetap migrasi tanpa folder SQL eksternal.

---

## Plan 04 — Repository + FIFO Atomik

**Tujuan:** Akses DB + pemotongan FIFO atomik, rollback 100% saat stok kurang.

**Scope file:**
- `internal/repository/sqlite_bean.go`
- `internal/repository/sqlite_product.go`
- `internal/repository/sqlite_order.go`

**Tahap:**
1. `sqlite_bean.go`: `CreateGreenBean`, `GetGreenBean`, `UpdateGreenStock` (guard anti-negatif), `CreateRoastBatch`, `DeductGreenAndCreateBatch` (atomik), `ListActiveBatches(beanID) ORDER BY roasted_at ASC, id ASC` (`remaining_mg>0`), `ListGreenBeans`, `CountBatchesForDay`.
2. `sqlite_product.go`: `ListActiveProducts`, `GetProduct`, `GetRecipes`.
3. `sqlite_order.go` — inti:
    - `BeginTx`, insert `orders` + `order_items`.
    - Loop FIFO deterministik (bean terurut): potong `remaining_mg`, insert `batch_deductions`.
    - Jika kurang → `tx.Rollback()` + `ErrInsufficientStock`, else `tx.Commit()`.
    - **Tidak pakai `FOR UPDATE`.**
4. Semua query pakai `context.Context`.

**Acceptance:**
- Single-batch exact, split multi-batch, drain `remaining=0`.
- Insufficient → rollback, `remaining_mg` tidak berubah.
- `batch_deductions` total = total required.

**Mapping FR:** FR-INV-01, FR-INV-02.

---

## Plan 05 — Service Layer

**Tujuan:** Aturan bisnis Roast / Order / Inventory / Report + receipt generator.

**Scope file:**
- `internal/service/roasting_service.go`
- `internal/service/order_service.go`
- `internal/service/inventory_service.go`
- `internal/service/receipt.go`
- `internal/service/report.go`
- DTO: `CartItem{ProductID, Quantity}`, `CheckoutRequest{OrderType, CustomerName, Items, PaidAmount, PaymentMethod}`

**Tahap:**
1. `roasting_service.go` (FR-ROAST-01..04):
    - Validasi `roasted<=green`, `green>0`, level valid, bean ada.
    - `shrinkage = (G-R)/G*100`.
    - Generate `BATCH-YYYYMMDD-XXX` via count per hari (inject `now func() time.Time` agar testable).
    - Atomik: kurangi `green_beans.stock_mg` + insert `roast_batches`.
2. `order_service.go` (FR-POS-01..03):
    - `CalculateTotal(items)`: cek `EmptyCart`, `ProductNotFound`, `qty<=0`, hanya `is_active=1`.
    - `ProcessCheckout(req)`: cek `paid>=total` else `ErrNegativePayment`, B2B wajib `customer_name`, expand BOM `required_mg*qty` per `green_bean_id` (resep generik tanpa bean dilewati), delegasi ke repo FIFO.
3. `inventory_service.go` (FR-INV-03):
    - `AlertLevelFor(remaining, threshold)` pure: `ok/low/critical`.
    - `ListGreenBeanAlerts`, `ListRoastBatchAlerts`.
4. `receipt.go` (FR-POS-04):
    - `GenerateReceipt(order, lines, change, filePath) string` lebar 32 char + baris kosong penutup. Pure, tanpa I/O.
5. `report.go` (FR-REP-01..02):
    - `Reporter.SummariseDay` + `SummariseDay(ctx, orders, now)`: revenue, count, per-method, konsumsi total + per batch sejak tengah malam lokal.

**Acceptance:**
- Roast invalid → `ErrInvalidRoastWeight`, stok mentah berkurang pas.
- Checkout kurang bayar → `ErrNegativePayment`, stok tak berubah.
- B2B tanpa nama → error validasi.

---

## Plan 06 — CLI

**Tujuan:** Subcommand operasional di atas service layer, stdlib only.

**Scope file:**
- `internal/cli/doc.go`
- `internal/cli/cli.go` (`Run`, global flags, dispatch, exit code)
- `internal/cli/weight.go` (`ParseWeight` mg/g/kg, `ParseMoney`)
- `internal/cli/list.go` (`beans`, `products`)
- `internal/cli/roast.go`
- `internal/cli/sell.go` (`--item` repeatable, struk stdout + file)
- `internal/cli/stock_report.go`

**Tahap:**
1. `Run(ctx, args, stdout, stderr, dbPath, receiptsDir) int`: pindai `--db`/`--receipts-dir` → open + migrasi → dispatch → 0 sukses / 1 gagal (pesan ke stderr). Tanpa argumen cetak help, exit 0.
2. `weight.go`: `ParseWeight("2kg")=2000000`, bare number = gram, tolak non-positif.
3. `roast --bean ID --green WEIGHT --roasted WEIGHT [--level Medium]`: cetak ID batch + shrinkage.
4. `sell --item ID:QTY... --paid AMOUNT [--b2b --customer N] [--method CASH]`: bangun receipt lines dari katalog → `ProcessCheckout` → tulis file + cetak struk.
5. `stock [--threshold 500g]`: tabel FIFO (ID/bean/sisa/umur/OK-LOW-CRIT) + tabel green beans via `tabwriter`.
6. `report`: agregasi `SummariseDay` tercetak.
7. `main.go`: `--version` → cetak versi, else `os.Exit(cli.Run(...))`.

**Acceptance:**
- Alur demo `roast → sell → stock → report` hijau end-to-end (lihat §8).
- Flag salah / stok kurang / bayar kurang → stderr + exit 1, DB tak berubah.

**Mapping FR:** FR-ROAST-01..04, FR-POS-01..04, FR-INV-01..03, FR-REP-01..02.

---

## Plan 07 — Testing Mendekati 100% + NFR

**Tujuan:** Public + private API tercover, kecuali mustahil; NFR terukur.

**Prinsip:**
- Test package sama (`package service`) untuk fungsi private.
- Pure logic (`AlertLevelFor`, `GenerateReceipt`, `ParseWeight`) diuji tabel.
- Helper DB sementara per test (`t.TempDir()` + `RunMigrations`).
- Closed-DB test untuk cabang error I/O.

**Hasil coverage (2026-09-10):** total 87.7% — domain 100%, service 93.1%, repository 90.3%, cli 83.3%, database 79.8%.

**Exclusion (didokumentasikan, tidak dihitung):**
- `cmd/1stcrack/main.go` (dispatch tipis).
- `sql.Open`/pragma/ping failures + migrasi gagal di tengah (butuh fault injection).
- `power loss` cabut kabel — diganti uji WAL + rollback.
- `chmod 000`/disk-full receipt — 1 negative path saja.

**Command:**
```bash
go test ./... -covermode=atomic -coverprofile=coverage.out -count=1
go tool cover -func=coverage.out | sort -k3 -n
```

---

## 8. Demo End-to-End (terverifikasi)

```bash
go build -o 1stcrack ./cmd/1stcrack
./1stcrack products
./1stcrack roast --bean GB-GAYO-WASHED --green 2kg --roasted 1700g --level Medium
./1stcrack sell --item P-LATTE-HOT:2 --paid 60000
./1stcrack sell --item P-BEANS-1KG:1 --paid 300000 --b2b --customer "Kafe X"
./1stcrack stock
./1stcrack report
ls receipts/
```

---

## 9. NFR Checklist (terukur)

- [x] NFR-PERF: satu perintah ~µs–ms (keypress+render 34µs, open+migrasi 6.7ms), startup <50ms.
- [x] NFR-RELIAB: transaksi atomik, WAL, rollback 100% teruji.
- [x] NFR-USAB: `1stcrack <command> [flags]`, tiap perintah `--help`, bobot `mg/g/kg`, tabel 80 kolom.
- [x] NFR-RES: binary ~10MB, tanpa proses persisten.
- [x] Offline-first, tanpa internet, output teks biasa.

---

## 10. Risiko

1. SQLite lock busy → mitigasi `MaxOpenConns=1` + `busy_timeout`.
2. Float rounding → mitigasi `int64 mg` + `math.Round`.
3. ID collision per hari → mitigasi count+tx + unique constraint.
4. Flag salah ketik → mitigasi `--help` per perintah + pesan error ke stderr + exit 1.

---

## 11. Lisensi

MIT atas nama Dzulkifli Anwar — lihat `LICENSE`.
