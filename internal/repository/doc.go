// Package repository provides SQLite persistence for domain entities.
//
// It implements atomic transactions and First-In First-Out stock deduction
// across roast batches. Callers must supply a context for cancellation.
package repository
