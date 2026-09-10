package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DefaultPath is the default SQLite file path for production use.
//
// It is intentionally relative so the database lives beside the binary
// during local standalone operation.
const DefaultPath = "1stcrack.db"

// Open opens an embedded SQLite database at the given path.
//
// It registers the pure-Go SQLite driver, restricts the pool to a single
// connection to avoid database-is-locked errors, and enables Write-Ahead
// Logging, foreign key enforcement, and a five second busy timeout for
// crash-safe standalone operation.
//
// The path parameter is the SQLite file path and must not be empty.
//
// It returns the configured connection pool.
//
// It returns an error if the path is empty or if any pragma cannot be applied.
func Open(path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("database path must not be empty")
	}
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA busy_timeout=5000;",
		"PRAGMA synchronous=NORMAL;",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply pragma %q: %w", p, err)
		}
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}
	return db, nil
}
