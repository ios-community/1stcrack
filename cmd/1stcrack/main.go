// Package main provides the 1stcrack command-line entry point.
//
// It dispatches subcommands for roastery operations. Database
// initialisation and migrations run inside the cli package per invocation.
package main

import (
	"context"
	"fmt"
	"os"

	"1stcrack/internal/cli"
	"1stcrack/internal/database"
)

// Version is the current application version.
//
// It is intentionally static for the MVP and is displayed by --version.
const Version = "1.0.0"

// receiptsDir is the export directory for receipt text files.
const receiptsDir = "receipts"

// main dispatches command-line arguments and exits with the command status.
func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("1stcrack", Version)
		return
	}
	os.Exit(cli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, database.DefaultPath, receiptsDir))
}
