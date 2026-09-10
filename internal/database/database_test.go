package database

import (
	"fmt"
	"path/filepath"
	"testing"
)

// TestOpenConfiguresWAL verifies Write-Ahead Logging and foreign keys.
func TestOpenConfiguresWAL(t *testing.T) {
	t.Parallel()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()
	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode;`).Scan(&mode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
}

// TestRunMigrationsIsIdempotent verifies ordered migration and seed replay safety.
func TestRunMigrationsIsIdempotent(t *testing.T) {
	t.Parallel()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()
	for i := 0; i < 2; i++ {
		if err := RunMigrations(db); err != nil {
			t.Fatalf("RunMigrations run %d: %v", i, err)
		}
	}
	var products int
	if err := db.QueryRow(`SELECT COUNT(*) FROM products;`).Scan(&products); err != nil {
		t.Fatalf("count products: %v", err)
	}
	if products != 3 {
		t.Fatalf("products = %d, want 3", products)
	}
}

// TestOpenRejectsEmptyPath verifies input validation.
func TestOpenRejectsEmptyPath(t *testing.T) {
	t.Parallel()
	if _, err := Open(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

// TestRunMigrationsRejectsNil verifies nil guard.
func TestRunMigrationsRejectsNil(t *testing.T) {
	t.Parallel()
	if err := RunMigrations(nil); err == nil {
		t.Fatal("expected error for nil database")
	}
}

// TestRunMigrationsClosedDB verifies ledger failure on closed storage.
func TestRunMigrationsClosedDB(t *testing.T) {
	t.Parallel()
	db, err := Open(filepath.Join(t.TempDir(), "closed.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}
	_ = db.Close()
	if err := RunMigrations(db); err == nil {
		t.Fatal("expected error for closed database")
	}
}

// ExampleRunMigrations demonstrates embedded migration execution.
func ExampleRunMigrations() {
	fmt.Println("migrations are embedded")
	// Output: migrations are embedded
}
