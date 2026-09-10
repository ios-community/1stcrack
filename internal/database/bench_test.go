package database

import (
	"path/filepath"
	"strconv"
	"testing"
)

// BenchmarkOpenMigrate measures embedded database startup latency.
func BenchmarkOpenMigrate(b *testing.B) {
	dir := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db, err := Open(filepath.Join(dir, "bench"+strconv.Itoa(i)+".db"))
		if err != nil {
			b.Fatalf("Open returned error: %v", err)
		}
		if err := RunMigrations(db); err != nil {
			b.Fatalf("RunMigrations returned error: %v", err)
		}
		_ = db.Close()
	}
}
