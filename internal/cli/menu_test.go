package cli

import (
	"context"
	"os"
	"strings"
	"testing"
)

// runSession executes the menu with scripted input and returns its output.
func runSession(t *testing.T, db string, receipts string, input string) (string, int) {
	t.Helper()
	var stdout, stderr strings.Builder
	code := Run(context.Background(), nil, strings.NewReader(input), &stdout, &stderr, db, receipts)
	if stderr.String() != "" {
		t.Logf("stderr: %s", stderr.String())
	}
	return stdout.String(), code
}

// TestMenuQuitImmediately verifies instant exit from role selection.
func TestMenuQuitImmediately(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	out, code := runSession(t, db, receipts, "0\n")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "Select role") {
		t.Fatalf("output missing roles:\n%s", out)
	}
}

// TestMenuEOF verifies clean exit on end of input.
func TestMenuEOF(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	if _, code := runSession(t, db, receipts, ""); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
}

// TestMenuCashierSale verifies a full guided sale end to end.
func TestMenuCashierSale(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	ctx := context.Background()
	var stdout, stderr strings.Builder
	if code := Run(ctx, []string{"roast", "--bean", "GB-GAYO-WASHED", "--green", "2kg", "--roasted", "1700g"}, strings.NewReader(""), &stdout, &stderr, db, receipts); code != 0 {
		t.Fatalf("roast: %d\n%s", code, stderr.String())
	}
	input := "1\n1\n3\n2\n0\nn\n60000\n\nn\n0\n"
	out, code := runSession(t, db, receipts, input)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	for _, want := range []string{"Catalogue:", "Total: Rp50.000", "Change", "Rp10.000", "Thank you!"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	entries, err := os.ReadDir(receipts)
	if err != nil || len(entries) != 1 {
		t.Fatalf("receipt files = %v, %v; want 1", entries, err)
	}
}

// TestMenuInvalidInputs verifies reprompts without crashing.
func TestMenuInvalidInputs(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	ctx := context.Background()
	var stdout, stderr strings.Builder
	if code := Run(ctx, []string{"roast", "--bean", "GB-GAYO-WASHED", "--green", "2kg", "--roasted", "1700g"}, strings.NewReader(""), &stdout, &stderr, db, receipts); code != 0 {
		t.Fatalf("roast: %d\n%s", code, stderr.String())
	}
	input := "9\n1\n1\n99\n3\nx\n2\n0\nn\nabc\n60000\nqris\nn\n0\n"
	out, code := runSession(t, db, receipts, input)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "Unknown choice") {
		t.Fatalf("output missing reprompt:\n%s", out)
	}
	if !strings.Contains(out, "Total: Rp50.000") {
		t.Fatalf("sale did not complete:\n%s", out)
	}
}

// TestMenuRoaster verifies guided batch recording and role switching.
func TestMenuRoaster(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	input := "2\n1\n1\n2kg\n1700g\n\n2\n9\n0\n"
	out, code := runSession(t, db, receipts, input)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	for _, want := range []string{"Green beans:", "shrinkage", "GB-GAYO-WASHED"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

// TestMenuOwner verifies stock and report screens.
func TestMenuOwner(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	out, code := runSession(t, db, receipts, "3\n3\n4\n0\n")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	for _, want := range []string{"GREEN BEANS", "BATCH", "TODAY"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

// TestMenuExplicitCommand verifies the menu subcommand matches bare entry.
func TestMenuExplicitCommand(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	var stdout, stderr strings.Builder
	code := Run(context.Background(), []string{"menu"}, strings.NewReader("0\n"), &stdout, &stderr, db, receipts)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Owner / Manager") {
		t.Fatalf("output missing roles:\n%s", stdout.String())
	}
}

// TestMenuB2BRepromptAndStockFail verifies empty names and failed checkouts.
func TestMenuB2BRepromptAndStockFail(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	ctx := context.Background()
	var stdout, stderr strings.Builder
	roast := []string{"roast", "--bean", "GB-GAYO-WASHED", "--green", "100g", "--roasted", "85g"}
	if code := Run(ctx, roast, strings.NewReader(""), &stdout, &stderr, db, receipts); code != 0 {
		t.Fatalf("roast: %d\n%s", code, stderr.String())
	}
	input := "1\n1\n3\n1\n0\ny\n\nCafe\n25000\n\ny\n2\n1\n0\nn\n85000\n\n0\n"
	out, code := runSession(t, db, receipts, input)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	for _, want := range []string{"is required", "insufficient"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	entries, err := os.ReadDir(receipts)
	if err != nil || len(entries) != 1 {
		t.Fatalf("receipt files = %v, %v; want 1", entries, err)
	}
}

// TestMenuMethodReprompt verifies unknown payment methods are retried.
func TestMenuMethodReprompt(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	ctx := context.Background()
	var stdout, stderr strings.Builder
	roast := []string{"roast", "--bean", "GB-GAYO-WASHED", "--green", "2kg", "--roasted", "1700g"}
	if code := Run(ctx, roast, strings.NewReader(""), &stdout, &stderr, db, receipts); code != 0 {
		t.Fatalf("roast: %d\n%s", code, stderr.String())
	}
	input := "1\n1\n1\n1\n0\nn\n300000\ngold\n\nn\n0\n"
	out, code := runSession(t, db, receipts, input)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "Unknown method") {
		t.Fatalf("output missing method reprompt:\n%s", out)
	}
}

// TestMenuMidFlowEOF verifies clean exit from deep inside flows.
func TestMenuMidFlowEOF(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	if _, code := runSession(t, db, receipts, "2\n1\n1\n"); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if _, code := runSession(t, db, receipts, "1\n"); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	out, code := runSession(t, db, receipts, "1\n1\n0\n")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "Cart is empty") {
		t.Fatalf("output missing empty cart:\n%s", out)
	}
}

// TestMenuLevelVariants verifies level numbers, names, and reprompts.
func TestMenuLevelVariants(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	for _, level := range []string{"1", "3", "light", "x\n1000\n850\n2"} {
		input := "2\n1\n1\n" + "1000\n850\n" + level + "\n9\n0\n"
		if level == "x\n1000\n850\n2" {
			input = "2\n1\n1\n" + level + "\n9\n0\n"
		}
		out, code := runSession(t, db, receipts, input)
		if code != 0 {
			t.Fatalf("exit = %d, want 0", code)
		}
		if !strings.Contains(out, "recorded BATCH-") {
			t.Fatalf("output missing batch:\n%s", out)
		}
	}
}

// TestMenuOwnerSale verifies owner sale and roast reuse.
func TestMenuOwnerSale(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	ctx := context.Background()
	var stdout, stderr strings.Builder
	if code := Run(ctx, []string{"roast", "--bean", "GB-GAYO-WASHED", "--green", "2kg", "--roasted", "1700g"}, strings.NewReader(""), &stdout, &stderr, db, receipts); code != 0 {
		t.Fatalf("roast: %d\n%s", code, stderr.String())
	}
	input := "3\n1\n3\n1\n0\nn\n25000\n\nn\n2\n1\n1000\n850\n2\n9\n0\n"
	out, code := runSession(t, db, receipts, input)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "Total: Rp25.000") || !strings.Contains(out, "recorded BATCH-") {
		t.Fatalf("owner flows incomplete:\n%s", out)
	}
}

// TestMenuCashierCatalog verifies catalogue display from cashier menu.
func TestMenuCashierCatalog(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	out, code := runSession(t, db, receipts, "1\n2\n0\n")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "Hot Latte") {
		t.Fatalf("output missing catalogue:\n%s", out)
	}
}

// TestMenuYesNoReprompt verifies invalid confirmations are retried.
func TestMenuYesNoReprompt(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	ctx := context.Background()
	var stdout, stderr strings.Builder
	if code := Run(ctx, []string{"roast", "--bean", "GB-GAYO-WASHED", "--green", "2kg", "--roasted", "1700g"}, strings.NewReader(""), &stdout, &stderr, db, receipts); code != 0 {
		t.Fatalf("roast: %d\n%s", code, stderr.String())
	}
	input := "1\n1\n3\n1\n0\nmaybe\nn\n25000\n\nn\n0\n"
	out, code := runSession(t, db, receipts, input)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "Answer y or N.") {
		t.Fatalf("output missing reprompt:\n%s", out)
	}
}

// TestMenuWeightReprompt verifies invalid weights are retried.
func TestMenuWeightReprompt(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	input := "2\n1\n1\nabc\n1000\n0\n850\n2\n9\n0\n"
	out, code := runSession(t, db, receipts, input)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "invalid weight") {
		t.Fatalf("output missing weight error:\n%s", out)
	}
}

// TestStockLowMarker verifies the LOW status branch.
func TestStockLowMarker(t *testing.T) {
	t.Parallel()
	db, receipts := testEnv(t)
	ctx := context.Background()
	var stdout, stderr strings.Builder
	args := []string{"roast", "--bean", "GB-GAYO-WASHED", "--green", "400g", "--roasted", "340g"}
	if code := Run(ctx, args, strings.NewReader(""), &stdout, &stderr, db, receipts); code != 0 {
		t.Fatalf("roast: %d\n%s", code, stderr.String())
	}
	stdout.Reset()
	if code := Run(ctx, []string{"stock"}, strings.NewReader(""), &stdout, &stderr, db, receipts); code != 0 {
		t.Fatalf("stock: %d\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "LOW") {
		t.Fatalf("output missing LOW marker:\n%s", stdout.String())
	}
}
