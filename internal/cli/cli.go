package cli

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"strings"

	"1stcrack/internal/database"
)

// generalUsage describes top-level invocation.
const generalUsage = `1stcrack — micro-roastery operations.

Usage:
  1stcrack [--db PATH] [--receipts-dir DIR] <command> [flags]
  1stcrack                                  Open the guided menu

Commands:
  menu        Open the guided menu (roles, sale, roast, stock, report)
  beans       List green beans
  products    List sellable products
  roast       Record a roast batch
  sell        Checkout a sale and print the receipt
  stock       Show FIFO roast batches with alerts
  report      Show today's revenue and consumption
  help        Show this help

Global flags must precede the command.
`

// commander carries shared invocation dependencies for subcommands.
type commander struct {
	// Shared SQLite connection pool.
	db *sql.DB
	// Interactive menu input source.
	stdin io.Reader
	// Normal output destination.
	stdout io.Writer
	// Error and flag usage output destination.
	stderr io.Writer
	// Receipt text file export directory.
	receiptsDir string
}

// Run executes one CLI invocation and returns the process exit code.
//
// Global flags --db and --receipts-dir must precede the command and override
// the given defaults. With no command it opens the guided menu reading from
// stdin. It returns 0 on success and 1 on any failure.
func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, dbPath string, receiptsDir string) int {
	rest := args
	for len(rest) > 0 {
		head := rest[0]
		switch {
		case head == "--db" && len(rest) > 1:
			dbPath = rest[1]
			rest = rest[2:]
		case strings.HasPrefix(head, "--db="):
			dbPath = strings.TrimPrefix(head, "--db=")
			rest = rest[1:]
		case head == "--receipts-dir" && len(rest) > 1:
			receiptsDir = rest[1]
			rest = rest[2:]
		case strings.HasPrefix(head, "--receipts-dir="):
			receiptsDir = strings.TrimPrefix(head, "--receipts-dir=")
			rest = rest[1:]
		default:
			goto dispatch
		}
	}
dispatch:
	if len(rest) == 0 {
		return openAndRun(stdin, dbPath, receiptsDir, stdout, stderr, func(cmd *commander) int {
			return cmd.runMenu(ctx)
		})
	}
	if rest[0] == "help" || rest[0] == "--help" || rest[0] == "-h" {
		_, _ = fmt.Fprint(stdout, generalUsage)
		return 0
	}
	return openAndRun(stdin, dbPath, receiptsDir, stdout, stderr, func(cmd *commander) int {
		var runErr error
		switch rest[0] {
		case "menu":
			return cmd.runMenu(ctx)
		case "beans":
			runErr = cmd.runBeans(rest[1:])
		case "products":
			runErr = cmd.runProducts(rest[1:])
		case "roast":
			runErr = cmd.runRoast(ctx, rest[1:])
		case "sell":
			runErr = cmd.runSell(ctx, rest[1:])
		case "stock":
			runErr = cmd.runStock(ctx, rest[1:])
		case "report":
			runErr = cmd.runReport(ctx, rest[1:])
		default:
			_, _ = fmt.Fprintf(stderr, "unknown command %q\n\n%s", rest[0], generalUsage)
			return 1
		}
		if runErr != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", runErr)
			return 1
		}
		return 0
	})
}

// openAndRun opens the database, migrates it, and runs the given action.
func openAndRun(stdin io.Reader, dbPath string, receiptsDir string, stdout io.Writer, stderr io.Writer, action func(cmd *commander) int) int {
	db, err := database.Open(dbPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "open database: %v\n", err)
		return 1
	}
	defer func() {
		_ = db.Close()
	}()
	if err := database.RunMigrations(db); err != nil {
		_, _ = fmt.Fprintf(stderr, "run migrations: %v\n", err)
		return 1
	}
	return action(&commander{db: db, stdin: stdin, stdout: stdout, stderr: stderr, receiptsDir: receiptsDir})
}

// newFlagSet creates a subcommand flag set writing usage to stderr.
func (c *commander) newFlagSet(name string, usage string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	fs.Usage = func() {
		_, _ = fmt.Fprintf(c.stderr, "Usage: 1stcrack %s\n\n%s", name, usage)
	}
	return fs
}
