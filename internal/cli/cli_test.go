package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testEnv creates an isolated database and receipts directory for CLI tests.
func testEnv(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "cli.db"), filepath.Join(dir, "receipts")
}

// TestParseWeight verifies unit suffix handling.
func TestParseWeight(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  int64
	}{
		{"850", 850000},
		{"850g", 850000},
		{"2.5kg", 2500000},
		{"1KG", 1000000},
		{"500mg", 500},
		{" 100 g ", 100000},
	}
	for _, c := range cases {
		got, err := ParseWeight(c.input)
		if err != nil || int64(got) != c.want {
			t.Fatalf("ParseWeight(%q) = %d, %v; want %d", c.input, got, err, c.want)
		}
	}
	for _, bad := range []string{"", "abc", "-5g", "0", "10lb"} {
		if _, err := ParseWeight(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

// TestParseMoney verifies digit extraction.
func TestParseMoney(t *testing.T) {
	t.Parallel()
	got, err := ParseMoney("50.000")
	if err != nil || got != 50000 {
		t.Fatalf("ParseMoney = %d, %v; want 50000", got, err)
	}
	if _, err := ParseMoney(""); err == nil {
		t.Fatal("expected error for empty input")
	}
}

// TestHelpAndUnknown verifies help output, menu entry, and exit codes.
func TestHelpAndUnknown(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	var stdout, stderr strings.Builder
	if code := Run(context.Background(), []string{"help"}, strings.NewReader(""), &stdout, &stderr, db, receipts); code != 0 {
		t.Fatalf("help exit = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "roast") {
		t.Fatalf("help missing commands:\n%s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(context.Background(), nil, strings.NewReader(""), &stdout, &stderr, db, receipts); code != 0 {
		t.Fatalf("menu EOF exit = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Select role") {
		t.Fatalf("menu missing role selection:\n%s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(context.Background(), []string{"nope"}, strings.NewReader(""), &stdout, &stderr, db, receipts); code == 0 {
		t.Fatal("expected non-zero exit for unknown command")
	}
}

// TestRoastSellStockReport verifies the full daily flow end to end.
func TestRoastSellStockReport(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	ctx := context.Background()
	var stdout, stderr strings.Builder
	run := func(args ...string) int {
		stdout.Reset()
		stderr.Reset()
		return Run(ctx, args, strings.NewReader(""), &stdout, &stderr, db, receipts)
	}
	if code := run("beans"); code != 0 {
		t.Fatalf("beans: %d\n%s", code, stderr.String())
	}
	if code := run("products"); code != 0 {
		t.Fatalf("products: %d\n%s", code, stderr.String())
	}
	if code := run("roast", "--bean", "GB-GAYO-WASHED", "--green", "2kg", "--roasted", "1700g"); code != 0 {
		t.Fatalf("roast: %d\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "15.0%") {
		t.Fatalf("roast missing shrinkage:\n%s", stdout.String())
	}
	if code := run("roast", "--bean", "GB-GAYO-WASHED", "--green", "1kg", "--roasted", "2kg"); code == 0 {
		t.Fatal("expected failure for roasted exceeding green")
	}
	if code := run("sell", "--item", "P-LATTE-HOT:2", "--paid", "60000"); code != 0 {
		t.Fatalf("sell: %d\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Change") {
		t.Fatalf("receipt missing change:\n%s", stdout.String())
	}
	entries, err := os.ReadDir(receipts)
	if err != nil || len(entries) != 1 {
		t.Fatalf("receipt files = %v, %v; want 1", entries, err)
	}
	if code := run("sell", "--item", "P-LATTE-HOT:1", "--paid", "1"); code == 0 {
		t.Fatal("expected failure for short payment")
	}
	if code := run("sell", "--item", "P-BEANS-1KG:99", "--paid", "999999999"); code == 0 {
		t.Fatal("expected failure for insufficient stock")
	}
	if code := run("stock"); code != 0 {
		t.Fatalf("stock: %d\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "OK") {
		t.Fatalf("stock missing markers:\n%s", stdout.String())
	}
	if code := run("report"); code != 0 {
		t.Fatalf("report: %d\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Rp50.000") {
		t.Fatalf("report missing revenue:\n%s", stdout.String())
	}
}

// TestSellB2B verifies wholesale checkout with customer name.
func TestSellB2B(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	ctx := context.Background()
	var stdout, stderr strings.Builder
	if code := Run(ctx, []string{"roast", "--bean", "GB-GAYO-WASHED", "--green", "2kg", "--roasted", "1700g"}, strings.NewReader(""), &stdout, &stderr, db, receipts); code != 0 {
		t.Fatalf("roast: %d\n%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	args := []string{"sell", "--item", "P-BEANS-1KG:1", "--paid", "300000", "--b2b", "--customer", "Cafe X"}
	if code := Run(ctx, args, strings.NewReader(""), &stdout, &stderr, db, receipts); code != 0 {
		t.Fatalf("sell b2b: %d\n%s", code, stderr.String())
	}
}

// TestSellItemValidation verifies item flag parsing errors.
func TestSellItemValidation(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	ctx := context.Background()
	var stdout, stderr strings.Builder
	for _, args := range [][]string{
		{"sell", "--paid", "1000"},
		{"sell", "--item", "NOQTY", "--paid", "1000"},
		{"sell", "--item", "P-LATTE-HOT:0", "--paid", "1000"},
	} {
		stdout.Reset()
		stderr.Reset()
		if code := Run(ctx, args, strings.NewReader(""), &stdout, &stderr, db, receipts); code == 0 {
			t.Fatalf("expected failure for %v", args)
		}
	}
}

// TestItemFlagsString verifies the repeatable flag summary.
func TestItemFlagsString(t *testing.T) {
	t.Parallel()
	var flags itemFlags
	if err := flags.Set("P-LATTE-HOT:2"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if got := flags.String(); got != "P-LATTE-HOT:2" {
		t.Fatalf("String = %q, want P-LATTE-HOT:2", got)
	}
}

// TestRoastMissingBean verifies the required bean flag guard.
func TestRoastMissingBean(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	var stdout, stderr strings.Builder
	args := []string{"roast", "--green", "1kg", "--roasted", "850g"}
	if code := Run(context.Background(), args, strings.NewReader(""), &stdout, &stderr, db, receipts); code == 0 {
		t.Fatal("expected failure for missing --bean")
	}
}

// TestBadDatabasePath verifies clean failure on unusable storage.
func TestBadDatabasePath(t *testing.T) {
	t.Parallel()
	var stdout, stderr strings.Builder
	args := []string{"--db", "/nonexistent-dir-1stcrack/x.db", "beans"}
	if code := Run(context.Background(), args, strings.NewReader(""), &stdout, &stderr, "/nonexistent-dir-1stcrack/x.db", t.TempDir()); code == 0 {
		t.Fatal("expected failure for bad database path")
	}
}

// ExampleParseWeight demonstrates weight flag parsing.
func ExampleParseWeight() {
	w, _ := ParseWeight("850g")
	fmt.Println(int64(w))
	// Output: 850000
}
