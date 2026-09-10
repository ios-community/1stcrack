// Package database provides embedded SQLite initialisation and schema migration.
//
// It configures Write-Ahead Logging mode and applies embedded SQL migration
// files in lexicographic order using Go embed directives.
package database
