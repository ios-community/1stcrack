package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"1stcrack/internal/database"
	"1stcrack/internal/domain"
)

// closedTestDB returns a migrated database that is already closed.
func closedTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "closed.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}
	_ = db.Close()
	return db
}

// TestClosedDBServiceErrors verifies service failures on closed storage.
func TestClosedDBServiceErrors(t *testing.T) {
	t.Parallel()
	db := closedTestDB(t)
	ctx := context.Background()
	roasts := NewRoastingService(db)
	if _, err := roasts.ExecuteBatch(ctx, RoastInput{GreenBeanID: "GB-GAYO-WASHED", GreenWeightMg: 1000, RoastedWeightMg: 850, RoastLevel: domain.RoastMedium}); err == nil {
		t.Fatal("expected error for closed database")
	}
	orders := NewOrderService(db)
	if _, err := orders.CalculateTotal(ctx, []CartItem{{ProductID: "P-LATTE-HOT", Quantity: 1}}); err == nil {
		t.Fatal("expected error for closed database")
	}
	if _, err := orders.ProcessCheckout(ctx, CheckoutRequest{Items: []CartItem{{ProductID: "P-LATTE-HOT", Quantity: 1}}, PaidAmount: 25000, PaymentMethod: domain.PaymentCash}); err == nil {
		t.Fatal("expected error for closed database")
	}
	inventory := NewInventoryService(db)
	if _, err := inventory.ListGreenBeanAlerts(ctx, 1000); err == nil {
		t.Fatal("expected error for closed database")
	}
	if _, err := inventory.ListRoastBatchAlerts(ctx, "GB-GAYO-WASHED", 1000); err == nil {
		t.Fatal("expected error for closed database")
	}
}
