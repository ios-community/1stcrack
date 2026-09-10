# Product Requirement Document (PRD)

**Nama Produk:** 1stcrack

**Versi:** 1.0.0 (MVP)

**Tipe Aplikasi:** Command Line Interface (CLI)

**Status:** Disetujui untuk Implementasi

---

# 1. Ringkasan Eksekutif & Latar Belakang

1stcrack adalah sistem operasional terintegrasi yang dirancang khusus untuk *micro-roastery* dan *specialty coffee shop* independen. Bisnis ini memiliki keunikan operasional: mereka membeli bahan mentah (*green beans*), menyangrainya (*roasting*) yang mengalami penyusutan bobot (*shrinkage*), lalu menjual hasilnya melalui dua jalur:

1. **B2C (Retail Kafe)**
Minuman per cangkir (mengonsumsi gramasi biji kopi matang per *shot espresso*) dan kemasan retail (250g/500g).
2. **B2B (Grosir)**
Penjualan biji kopi sangrai kiloan (1kg–10kg) ke kafe rekanan.

1stcrack hadir untuk menggantikan pencatatan manual/spreadsheet yang rawan selisih stok, tidak mampu melacak usia *freshness* batch sangrai, dan sulit menghitung biaya produksi riil.

---

# 2. Profil Pengguna (*User Personas*)

| Persona | Peran | Kebutuhan Utama |
| --- | --- | --- |
| **Budi (Kasir / Barista)** | Operasional Penjualan B2C & B2B | Input pesanan cepat via perintah terminal, daftar katalog instan, cetak struk nota, tidak perlu pusing memikirkan sisa stok mentah. |
| **Rian (Head Roaster)** | Produksi & Manajemen Batch | Mencatat *green beans* masuk, menghitung otomatis persentase susut sangrai (*weight loss*), dan memberi label batch hasil sangrai. |
| **Siti (Owner / Manajer)** | Pengawasan & Audit Bisnis | Melihat stok biji kopi yang menipis (*low-stock alert*), laporan omzet harian, dan audit konsumsi kopi per batch. |

---

# 3. Kebutuhan Fungsional (*Functional Requirements*)

## Modul 1: Manajemen Produksi Sangrai (*Roasting Management*)

- **FR-ROAST-01 (Pencatatan Green Beans)**
Pengguna dapat mendaftarkan stok *green beans* baru (Origin, Varietas, Proses, Tanggal Beli, Harga Beli per Kg, Bobot Masuk).
- **FR-ROAST-02 (Eksekusi Batch Sangrai)**
Pengguna dapat membuat batch sangrai baru dengan memilih *green bean*, memasukkan bobot mentah (*green weight*), dan bobot matang (*roasted weight*).
- **FR-ROAST-03 (Kalkulasi Shrinkage Otomatis)**
Sistem wajib menghitung persentase kehilangan bobot (*weight loss*) secara otomatis dengan formula:
    
    $$
    \text{Shrinkage (\%)} = \frac{\text{Berat Mentah} - \text{Berat Matang}}{\text{Berat Mentah}} \times 100\%
    $$
    
- **FR-ROAST-04 (Penomoran Batch)**
Sistem otomatis membuat ID Batch unik (format: `BATCH-YYYYMMDD-XXX`) dan mencatat level sangrai (*Light / Medium / Dark*).

## Modul 2: Kasir & Penjualan (*Point of Sale*)

- **FR-POS-01 (Katalog & Keranjang)**
Kasir melihat katalog via `1stcrack products`, lalu menjual via `1stcrack sell --item ID:QTY` (flag `--item` dapat diulang untuk banyak produk).
- **FR-POS-02 (Bill of Materials / Resep Otomatis)**
Setiap penjualan menu minuman otomatis memicu pengurangan stok bahan baku terhubung (Contoh: 1 Cup *Hot Latte* memotong 18.0 gram *Roasted Beans* dan 1 unit *Paper Cup*).
- **FR-POS-03 (Dukungan B2B Wholesale)**
Kasir dapat memilih mode B2B untuk penjualan biji kopi kiloan dengan harga grosir dan input nama kafe pembeli.
- **FR-POS-04 (Pembayaran & Ekspor Struk)**
Sistem memvalidasi nominal pembayaran, menghitung kembalian, dan secara otomatis mengekspor struk transaksi dalam format teks `.txt` rapi ke folder `/receipts`.

## Modul 3: Manajemen Inventori & Aturan FIFO (*Strict FIFO Stock*)

- **FR-INV-01 (Pelacakan Batch Tertua - FIFO)**
Saat terjadi penjualan (baik B2C maupun B2B), sistem **wajib memotong stok dari batch sangrai tertua** yang masih tersedia (*First-In, First-Out*).
- **FR-INV-02 (Split Batch Otomatis)**
Jika sisa kopi pada batch tertua tidak mencukupi satu pesanan, sistem secara otomatis menghabiskan batch tersebut dan mengambil sisanya dari batch tertua berikutnya.
- **FR-INV-03 (Peringatan Stok Rendah)**
Sistem menampilkan penanda status `OK/LOW/CRIT` pada perintah `1stcrack stock` jika stok biji kopi atau *green beans* berada di bawah batas minimum (*threshold*, default 500g, dapat diubah via `--threshold`).

## Modul 4: Laporan & Ringkasan Shift (*Reporting*)

- **FR-REP-01 (Rekap Penjualan Harian)**
Menampilkan total pendapatan (Rupiah), jumlah transaksi, dan metode pembayaran (Tunai/QRIS/Transfer).
- **FR-REP-02 (Audit Konsumsi Kopi)**
Menampilkan total gramasi kopi yang terpakai hari ini beserta perincian batch mana saja yang terkonsumsi.

---

# 4. Kebutuhan Non-Fungsional (*Non-Functional Requirements*)

- **NFR-PERF (Performa)**
Eksekusi satu perintah < 30ms di luar I/O SQLite; aplikasi siap digunakan (*startup time*, open + migrasi DB) < 50ms.
- **NFR-RELIAB (Keandalan & Anti-Crash)**
Seluruh transaksi pemotongan stok bersifat atomik. Penghentian paksa proses atau *power loss* tidak boleh merusak file database (*zero corruption*, mode WAL).
- **NFR-USAB (Kemudahan Penggunaan CLI)**
Pola pemanggilan `1stcrack [--db PATH] <command> [flags]`, setiap perintah mendukung `--help`, input bobot memahami sufiks `mg/g/kg`, output tabel teks biasa yang terbaca di terminal selebar 80 karakter. Tanpa mouse, tanpa fokus, tanpa tombol fungsi.
- **NFR-RES (Efisiensi Resource)**
Konsumsi RAM tidak melebihi 25 MB dalam kondisi operasional penuh.

---

# 5. Batasan Sistem (*Constraints*)

- Aplikasi beroperasi secara lokal (*standalone offline-first*) tanpa ketergantungan koneksi internet.
- Output berupa teks biasa (tabel sejajar) tanpa ketergantungan warna terminal atau tombol fungsi.