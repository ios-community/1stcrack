package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"1stcrack/internal/domain"
)

// TestCalculateTotalEdges verifies inactive, missing, and quantity guards.
func TestCalculateTotalEdges(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	orders := NewOrderService(db)
	if _, err := orders.CalculateTotal(ctx, []CartItem{{ProductID: "P-MISSING", Quantity: 1}}); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
	if _, err := orders.CalculateTotal(ctx, []CartItem{{ProductID: "P-LATTE-HOT", Quantity: 0}}); err == nil {
		t.Fatal("expected error for zero quantity")
	}
	if _, err := db.Exec(`UPDATE products SET is_active = 0 WHERE id = 'P-LATTE-HOT';`); err != nil {
		t.Fatalf("deactivate product: %v", err)
	}
	if _, err := orders.CalculateTotal(ctx, []CartItem{{ProductID: "P-LATTE-HOT", Quantity: 1}}); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound for inactive, got %v", err)
	}
}

// TestProcessCheckoutInvalid verifies order type and payment guards.
func TestProcessCheckoutInvalid(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	orders := NewOrderService(db)
	base := CheckoutRequest{Items: []CartItem{{ProductID: "P-LATTE-HOT", Quantity: 1}}, PaidAmount: 25000, PaymentMethod: domain.PaymentCash}
	badType := base
	badType.OrderType = "B2X"
	if _, err := orders.ProcessCheckout(ctx, badType); err == nil {
		t.Fatal("expected error for invalid order type")
	}
	badPay := base
	badPay.PaymentMethod = "GOLD"
	if _, err := orders.ProcessCheckout(ctx, badPay); err == nil {
		t.Fatal("expected error for invalid payment method")
	}
	missing := base
	missing.Items = []CartItem{{ProductID: "P-MISSING", Quantity: 1}}
	if _, err := orders.ProcessCheckout(ctx, missing); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

// TestRoastGuards verifies bean identifier and existence guards.
func TestRoastGuards(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	roasts := NewRoastingService(db)
	input := RoastInput{GreenWeightMg: 1000, RoastedWeightMg: 850, RoastLevel: domain.RoastMedium}
	if _, err := roasts.ExecuteBatch(ctx, input); err == nil {
		t.Fatal("expected error for empty bean id")
	}
	input.GreenBeanID = "GB-MISSING"
	if _, err := roasts.ExecuteBatch(ctx, input); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

// TestGenericRecipeSkipsDeduction verifies recipes without bean lineage.
func TestGenericRecipeSkipsDeduction(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO products (id, name, category, price, is_active) VALUES ('P-CUP', 'Paper Cup', 'DRINK', 2000, 1);`); err != nil {
		t.Fatalf("insert product: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO product_recipes (product_id, green_bean_id, required_roasted_mg) VALUES ('P-CUP', NULL, 0);`); err != nil {
		t.Fatalf("insert generic recipe: %v", err)
	}
	orders := NewOrderService(db)
	order, err := orders.ProcessCheckout(ctx, CheckoutRequest{Items: []CartItem{{ProductID: "P-CUP", Quantity: 1}}, PaidAmount: 2000, PaymentMethod: domain.PaymentQRIS})
	if err != nil {
		t.Fatalf("ProcessCheckout: %v", err)
	}
	if order.TotalAmount != 2000 {
		t.Fatalf("total = %d, want 2000", order.TotalAmount)
	}
}

// TestBatchAlerts verifies per-batch alert listing.
func TestBatchAlerts(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	roasts := NewRoastingService(db)
	roastForTest(t, roasts, "GB-GAYO-WASHED", 1000000, 850000)
	inventory := NewInventoryService(db)
	alerts, err := inventory.ListRoastBatchAlerts(ctx, "GB-GAYO-WASHED", 10000000)
	if err != nil || len(alerts) != 1 || alerts[0].Level != AlertCritical {
		t.Fatalf("alerts = %+v, %v", alerts, err)
	}
	empty, err := inventory.ListRoastBatchAlerts(ctx, "GB-LINTONG-NATURAL", 1000)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty alerts = %+v, %v", empty, err)
	}
}

// TestReceiptEdges verifies truncation, zero time, and unknown payment rendering.
func TestReceiptEdges(t *testing.T) {
	t.Parallel()
	order := &domain.Order{ID: "ORD-EDGE-1", OrderType: domain.OrderTypeB2B, CustomerName: "Kafe", TotalAmount: 2000000, PaidAmount: 2000000, PaymentMethod: "GOLD"}
	long := strings.Repeat("A", 60)
	out := GenerateReceipt(order, []ReceiptLine{{Name: long, Quantity: 1, Subtotal: 2000000}}, 0, "")
	for _, line := range strings.Split(out, "\n") {
		if len(line) > ReceiptWidth {
			t.Fatalf("line exceeds width %d: %q", ReceiptWidth, line)
		}
	}
	if !strings.Contains(out, "GOLD") || !strings.Contains(out, "-") {
		t.Fatalf("receipt missing fallback rendering:\n%s", out)
	}
	centered := GenerateReceipt(order, nil, 0, "")
	if !strings.Contains(centered, "Terima kasih!") {
		t.Fatalf("receipt missing footer:\n%s", centered)
	}
}
