# 1stcrack

Micro-roastery operations in one offline binary: record roast batches with automatic shrinkage, sell with atomic FIFO stock deduction, monitor low stock, and print daily reports with text receipts.

Status: MVP v1.0.0, CLI-only.

## Fitur

- Roasting: catat batch sangrai + kalkulasi shrinkage otomatis + ID `BATCH-YYYYMMDD-XXX`.
- Penjualan B2C/B2B: expand resep (BOM) otomatis, validasi bayar + kembalian, struk 32 kolom ke stdout dan file `receipts/`.
- Stok FIFO ketat: potong dari batch tertua, split otomatis, rollback 100% saat kurang.
- Alert stok `OK/LOW/CRIT` dengan threshold yang bisa diubah (`--threshold`, default 500g).
- Laporan harian: omzet, jumlah transaksi, per metode bayar, konsumsi per batch.

## Syarat

- Go 1.27.1+.
- Satu-satunya dependensi runtime adalah SQLite pure-Go (`modernc.org/sqlite`, tanpa CGO). Tanpa internet saat operasi.

## Build & Install

```bash
go build -o 1stcrack ./cmd/1stcrack
./1stcrack --version
./1stcrack help
```

Database (`1stcrack.db`) dan migrasi dibuat otomatis di folder kerja saat perintah pertama dijalankan.

## Panduan Cepat

```bash
./1stcrack products
./1stcrack roast --bean GB-GAYO-WASHED --green 2kg --roasted 1700g --level Medium
# recorded BATCH-20260910-001 — 15.0% shrinkage, 1700 g remaining

./1stcrack sell --item P-LATTE-HOT:2 --paid 60000
# struk tercetak: Total Rp50.000, Kembali Rp10.000 + file receipts/ORD-*.txt

./1stcrack sell --item P-BEANS-1KG:1 --paid 300000 --b2b --customer "Kafe X"
./1stcrack stock
./1stcrack report
```

Bobot memahami sufiks `mg`, `g` (default), `kg`: `850`, `850g`, `2.5kg`.

## Referensi Perintah

| Perintah | Fungsi |
|---|---|
| `beans` | Daftar green beans + stok mentah |
| `products` | Daftar produk jual + harga |
| `roast --bean ID --green W --roasted W [--level]` | Catat batch sangrai |
| `sell --item ID:QTY... --paid N [--b2b --customer N] [--method]` | Checkout + struk |
| `stock [--threshold W]` | Batch FIFO + umur + status |
| `report` | Rekap harian |
| `help` | Bantuan |

Global flags (sebelum subcommand): `--db PATH` (default `1stcrack.db`), `--receipts-dir DIR` (default `receipts`). Exit code 0 sukses, 1 gagal (pesan ke stderr).

## Struktur Project

```
cmd/1stcrack/main.go      # dispatch subcommand
internal/cli/             # subcommand (flag stdlib, tanpa state)
internal/service/         # aturan bisnis + receipt + report
internal/repository/      # SQLite + transaksi FIFO atomik
internal/domain/          # entity + WeightMg/MoneyIDR + sentinel errors
internal/database/        # open WAL + migrasi go:embed
docs/                     # PRD.md, TDD.md
```

Alur berlapis: `cli → service → repository → database`, `domain` tanpa dependensi. Semua presisi memakai `int64` (miligram & Rupiah) agar bebas floating-point error.

## Testing

```bash
go vet ./...
go test ./... -count=1
go test ./... -covermode=atomic -coverprofile=coverage.out -count=1
go tool cover -func=coverage.out | sort -k3 -n
```

Coverage total 87.7% (domain 100%). Termasuk uji rollback FIFO, closed-DB fault, dan benchmark (keypress+render ~34µs, open+migrasi ~6.7ms).

## Dokumen Terkait

- `docs/PRD.md` — kebutuhan produk.
- `docs/TDD.md` — desain teknis.
- `PLAN.md` — rencana eksekusi + demo E2E.
- Standar dokumentasi kode: Go Documentation Standard (setiap deklarasi publik/privat berdokumen).

## Lisensi

MIT atas nama Dzulkifli Anwar — lihat `LICENSE`.
