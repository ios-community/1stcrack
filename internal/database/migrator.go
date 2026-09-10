package database

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

// migrationFS holds embedded SQL migration files.
//
// Files are applied in lexicographic order exactly once, tracked by the
// schema_migrations table.
//
//go:embed migrations/*.sql
var migrationFS embed.FS

// RunMigrations applies pending embedded SQL migrations in order.
//
// It creates the schema_migrations ledger when missing, skips already applied
// versions, and executes each pending file inside its own transaction. A
// failed file aborts the run without recording its version.
//
// It returns an error if the ledger cannot be prepared or if any migration
// fails.
func RunMigrations(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database connection must not be nil")
	}
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	files, err := listMigrationFiles()
	if err != nil {
		return err
	}
	applied, err := appliedMigrations(db)
	if err != nil {
		return err
	}
	for _, name := range files {
		version := migrationVersion(name)
		if applied[version] {
			continue
		}
		if err := executeMigration(db, name, version); err != nil {
			return err
		}
	}
	return nil
}

// ensureMigrationsTable creates the schema version ledger when missing.
func ensureMigrationsTable(db *sql.DB) error {
	const q = `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(q); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}
	return nil
}

// listMigrationFiles returns embedded migration filenames in sorted order.
func listMigrationFiles() ([]string, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		files = append(files, "migrations/"+e.Name())
	}
	sort.Strings(files)
	return files, nil
}

// appliedMigrations returns the set of already recorded migration versions.
func appliedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(`SELECT version FROM schema_migrations;`)
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	out := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan migration version: %w", err)
		}
		out[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration versions: %w", err)
	}
	return out, nil
}

// executeMigration executes a single migration file inside a transaction.
func executeMigration(db *sql.DB, name string, version string) error {
	content, err := migrationFS.ReadFile(name)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	for _, stmt := range splitStatements(string(content)) {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("execute migration %s: %w", name, err)
		}
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations(version) VALUES (?);`, version); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	committed = true
	return nil
}

// migrationVersion derives the ledger version from a migration filename.
func migrationVersion(name string) string {
	base := name
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	return strings.TrimSuffix(base, ".sql")
}

// splitStatements splits a SQL script into executable statements.
func splitStatements(script string) []string {
	parts := strings.Split(script, ";")
	var out []string
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		out = append(out, s+";")
	}
	return out
}
