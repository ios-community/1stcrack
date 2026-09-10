package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"1stcrack/internal/database"
	"1stcrack/internal/domain"
)

// openTestDB creates a migrated temporary database for service tests.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "svc.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}
	return db
}

// roastForTest records a roast batch or fails the test.
func roastForTest(t *testing.T, svc *RoastingService, beanID string, green domain.WeightMg, roasted domain.WeightMg) {
	t.Helper()
	_, err := svc.ExecuteBatch(context.Background(), RoastInput{GreenBeanID: beanID, GreenWeightMg: green, RoastedWeightMg: roasted, RoastLevel: domain.RoastMedium})
	if err != nil {
		t.Fatalf("ExecuteBatch: %v", err)
	}
}

// TestRoastSuccess verifies shrinkage and green stock deduction.
func TestRoastSuccess(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := NewRoastingService(db)
	batch, err := svc.ExecuteBatch(context.Background(), RoastInput{GreenBeanID: "GB-GAYO-WASHED", GreenWeightMg: 100000, RoastedWeightMg: 85000, RoastLevel: domain.RoastMedium})
	if err != nil {
		t.Fatalf("ExecuteBatch: %v", err)
	}
	if batch.ShrinkagePct != 15.0 || batch.RemainingMg != 85000 {
		t.Fatalf("unexpected batch: %+v", batch)
	}
	if !strings.HasPrefix(batch.ID, "BATCH-") {
		t.Fatalf("batch id = %q, want BATCH- prefix", batch.ID)
	}
}

// TestRoastInvalid verifies weight and level validation.
func TestRoastInvalid(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	svc := NewRoastingService(db)
	if _, err := svc.ExecuteBatch(context.Background(), RoastInput{GreenBeanID: "GB-GAYO-WASHED", GreenWeightMg: 1000, RoastedWeightMg: 1200, RoastLevel: domain.RoastMedium}); !errors.Is(err, domain.ErrInvalidRoastWeight) {
		t.Fatalf("expected ErrInvalidRoastWeight, got %v", err)
	}
	if _, err := svc.ExecuteBatch(context.Background(), RoastInput{GreenBeanID: "GB-GAYO-WASHED", GreenWeightMg: 1000, RoastedWeightMg: 900, RoastLevel: "Ultra"}); err == nil {
		t.Fatal("expected error for invalid roast level")
	}
	if _, err := svc.ExecuteBatch(context.Background(), RoastInput{GreenBeanID: "GB-GAYO-WASHED", GreenWeightMg: 999999999, RoastedWeightMg: 999999000, RoastLevel: domain.RoastMedium}); !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
}

// TestOrderCheckoutSuccess verifies totals and FIFO consumption end to end.
func TestOrderCheckoutSuccess(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	roasts := NewRoastingService(db)
	orders := NewOrderService(db)
	roastForTest(t, roasts, "GB-GAYO-WASHED", 1000000, 850000)
	total, err := orders.CalculateTotal(context.Background(), []CartItem{{ProductID: "P-LATTE-HOT", Quantity: 2}})
	if err != nil {
		t.Fatalf("CalculateTotal: %v", err)
	}
	if total != 50000 {
		t.Fatalf("total = %d, want 50000", total)
	}
	order, err := orders.ProcessCheckout(context.Background(), CheckoutRequest{OrderType: domain.OrderTypeB2C, Items: []CartItem{{ProductID: "P-LATTE-HOT", Quantity: 2}}, PaidAmount: 60000, PaymentMethod: domain.PaymentCash})
	if err != nil {
		t.Fatalf("ProcessCheckout: %v", err)
	}
	if order.TotalAmount != 50000 || order.PaidAmount-order.TotalAmount != 10000 {
		t.Fatalf("unexpected order totals: %+v", order)
	}
}

// TestOrderValidation verifies payment, B2B, and empty cart guards.
func TestOrderValidation(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	orders := NewOrderService(db)
	if _, err := orders.ProcessCheckout(context.Background(), CheckoutRequest{Items: []CartItem{{ProductID: "P-LATTE-HOT", Quantity: 1}}, PaidAmount: 1, PaymentMethod: domain.PaymentCash}); !errors.Is(err, domain.ErrNegativePayment) {
		t.Fatalf("expected ErrNegativePayment, got %v", err)
	}
	if _, err := orders.ProcessCheckout(context.Background(), CheckoutRequest{OrderType: domain.OrderTypeB2B, Items: []CartItem{{ProductID: "P-LATTE-HOT", Quantity: 1}}, PaidAmount: 25000, PaymentMethod: domain.PaymentCash}); err == nil {
		t.Fatal("expected error for B2B without customer name")
	}
	if _, err := orders.ProcessCheckout(context.Background(), CheckoutRequest{Items: nil, PaidAmount: 0, PaymentMethod: domain.PaymentCash}); !errors.Is(err, domain.ErrEmptyCart) {
		t.Fatalf("expected ErrEmptyCart, got %v", err)
	}
}

// TestAlertLevelFor verifies threshold boundaries.
func TestAlertLevelFor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		remaining domain.WeightMg
		threshold domain.WeightMg
		want      AlertLevel
	}{
		{100, 0, AlertOK},
		{0, 100, AlertCritical},
		{10, 100, AlertCritical},
		{60, 100, AlertLow},
		{200, 100, AlertOK},
	}
	for _, c := range cases {
		if got := AlertLevelFor(c.remaining, c.threshold); got != c.want {
			t.Fatalf("AlertLevelFor(%d,%d) = %s, want %s", c.remaining, c.threshold, got, c.want)
		}
	}
}

// TestInventoryAlerts verifies per-bean alert listing.
func TestInventoryAlerts(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	inventory := NewInventoryService(db)
	alerts, err := inventory.ListGreenBeanAlerts(context.Background(), 10000000)
	if err != nil {
		t.Fatalf("ListGreenBeanAlerts: %v", err)
	}
	if len(alerts) != 2 || alerts[0].Level != AlertLow {
		t.Fatalf("unexpected alerts: %+v", alerts)
	}
}

// TestGenerateReceipt verifies preview and file content markers.
func TestGenerateReceipt(t *testing.T) {
	t.Parallel()
	order := &domain.Order{ID: "ORD-20260214-0001", OrderType: domain.OrderTypeB2C, TotalAmount: 135000, PaidAmount: 150000, PaymentMethod: domain.PaymentCash}
	lines := []ReceiptLine{{Name: "Hot Latte", Quantity: 2, Subtotal: 50000}, {Name: "Beans 250g", Quantity: 1, Subtotal: 85000}}
	out := GenerateReceipt(order, lines, 15000, "receipts/ORD-20260214-0001.txt")
	for _, want := range []string{"ORD-20260214-0001", "Rp135.000", "Rp15.000", "Kembali", "receipts/"} {
		if !strings.Contains(out, want) {
			t.Fatalf("receipt missing %q:\n%s", want, out)
		}
	}
}

// ExampleGenerateReceipt demonstrates receipt rendering for preview and export.
func ExampleGenerateReceipt() {
	order := &domain.Order{ID: "ORD-1", OrderType: domain.OrderTypeB2C, TotalAmount: 25000, PaidAmount: 25000, PaymentMethod: domain.PaymentCash}
	out := GenerateReceipt(order, []ReceiptLine{{Name: "Hot Latte", Quantity: 1, Subtotal: 25000}}, 0, "")
	fmt.Println(len(out) > 0)
	// Output: true
}
