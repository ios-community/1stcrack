# Product Requirement Document (PRD)

**Product Name:** 1stcrack

**Version:** 1.0.0 (MVP)

**Application Type:** Command Line Interface (CLI)

**Status:** Approved for Implementation

---

# 1. Executive Summary & Background

1stcrack is an integrated operations system built specifically for independent *micro-roasteries* and *specialty coffee shops*. These businesses have unique operations: they buy raw materials (*green beans*), roast them (*roasting*) with weight loss (*shrinkage*), then sell the result through two channels:

1. **B2C (Retail Cafe)**
Per-cup drinks (consuming roasted bean grams per *espresso shot*) and retail packs (250g/500g).
2. **B2B (Wholesale)**
Kilo packs of roasted beans (1kg–10kg) for partner cafes.

1stcrack exists to replace manual records and spreadsheets, which are prone to stock drift, cannot track roast batch *freshness* age, and make real production costing difficult.

---

# 2. User Personas

| Persona | Role | Key Need |
| --- | --- | --- |
| **Budi (Cashier / Barista)** | B2C & B2B sales operations | Fast order entry via terminal commands, instant catalogue listing, printed receipt notes, no need to think about raw stock levels. |
| **Rian (Head Roaster)** | Production & batch management | Record incoming *green beans*, automatic roast shrinkage (*weight loss*) percentage, and label resulting roast batches. |
| **Siti (Owner / Manager)** | Business oversight & audit | See thinning coffee stock (*low-stock alert*), daily revenue reports, and per-batch coffee consumption audits. |

---

# 3. Functional Requirements

## Module 1: Roast Production Management (*Roasting Management*)

- **FR-ROAST-01 (Green Bean Recording)**
Users can register new *green bean* stock (Origin, Variety, Process, Purchase Date, Purchase Price per Kg, Incoming Weight).
- **FR-ROAST-02 (Roast Batch Execution)**
Users can create a new roast batch by selecting a *green bean* and entering the green weight (*green weight*) and roasted weight (*roasted weight*).
- **FR-ROAST-03 (Automatic Shrinkage Calculation)**
The system must automatically calculate the weight loss percentage with the formula:
    
    $$
    \text{Shrinkage (\%)} = \frac{\text{Green Weight} - \text{Roasted Weight}}{\text{Green Weight}} \times 100\%
    $$
    
- **FR-ROAST-04 (Batch Numbering)**
The system automatically creates a unique Batch ID (format: `BATCH-YYYYMMDD-XXX`) and records the roast level (*Light / Medium / Dark*).

## Module 2: Cashier & Sales (*Point of Sale*)

- **FR-POS-01 (Catalogue & Cart)**
Cashiers view the catalogue via `1stcrack products`, then sell via `1stcrack sell --item ID:QTY` (the `--item` flag is repeatable for multiple products).
- **FR-POS-02 (Automatic Bill of Materials / Recipes)**
Every drink sale automatically deducts the linked raw materials (Example: 1 Cup of *Hot Latte* deducts 18.0 grams of *Roasted Beans* and 1 unit of *Paper Cup*).
- **FR-POS-03 (B2B Wholesale Support)**
Cashiers can select B2B mode for kilo-pack coffee sales at wholesale prices with the buyer cafe name as input.
- **FR-POS-04 (Payment & Receipt Export)**
The system validates the payment amount, calculates change, and automatically exports the transaction receipt in neat `.txt` text format to the `/receipts` folder.

## Module 3: Inventory Management & FIFO Rules (*Strict FIFO Stock*)

- **FR-INV-01 (Oldest Batch Tracking - FIFO)**
On every sale (both B2C and B2B), the system **must deduct stock from the oldest available roast batch** (*First-In, First-Out*).
- **FR-INV-02 (Automatic Batch Splitting)**
When the oldest batch cannot cover one order, the system automatically empties that batch and takes the remainder from the next oldest batch.
- **FR-INV-03 (Low Stock Warning)**
The system shows an `OK/LOW/CRIT` status marker on the `1stcrack stock` command when coffee or *green bean* stock falls below the minimum (*threshold*, default 500g, adjustable via `--threshold`).

## Module 4: Shift Reports & Summaries (*Reporting*)

- **FR-REP-01 (Daily Sales Recap)**
Shows total revenue (Rupiah), transaction count, and payment methods (Cash/QRIS/Transfer).
- **FR-REP-02 (Coffee Consumption Audit)**
Shows total coffee grams consumed today with a breakdown of which batches were consumed.

## Module 5: Guided Interactive Menu (*Guided Menu*)

- **FR-MENU-01 (Role Menu & Guided Flows)**
Running `1stcrack` with no arguments (or `1stcrack menu`) opens an interactive menu: select a role (Cashier/Barista, Head Roaster, Owner/Manager — no password for MVP), then the role menu guides input step by step (sale, batch recording, stock, reports). Invalid input never stops the program; it reprompts with an error message. Choice `0` in any menu and end of input (Ctrl+D) close the program cleanly with exit code 0.

---

# 4. Non-Functional Requirements (*Non-Functional Requirements*)

- **NFR-PERF (Performance)**
Single command execution < 30ms excluding SQLite I/O; application ready for use (*startup time*, DB open + migration) < 50ms.
- **NFR-RELIAB (Reliability & Anti-Crash)**
All stock deduction transactions are atomic. Forced process termination or *power loss* must not corrupt the database file (*zero corruption*, WAL mode).
- **NFR-USAB (CLI Usability)**
Invocation pattern `1stcrack [--db PATH] <command> [flags]`, every command supports `--help`, weight input understands `mg/g/kg` suffixes, plain-text table output readable in an 80-column terminal. Running with no arguments opens the role-based guided menu. No mouse, no focus, no function keys.
- **NFR-RES (Resource Efficiency)**
RAM consumption does not exceed 25 MB under full operation.

---

# 5. System Constraints (*Constraints*)

- The application operates locally (*standalone offline-first*) with no internet dependency.
- Output is plain text (aligned tables) with no dependency on terminal colours or function keys.
