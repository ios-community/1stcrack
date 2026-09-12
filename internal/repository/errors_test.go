package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"1stcrack/internal/database"
	"1stcrack/internal/domain"
)

// orderTestTime returns a fixed reference time for order listing tests.
func orderTestTime() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

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

// TestClosedDBBeanErrors verifies bean queries fail cleanly on closed storage.
func TestClosedDBBeanErrors(t *testing.T) {
	t.Parallel()
	db := closedTestDB(t)
	ctx := context.Background()
	beans := NewBeanRepository(db)
	if _, err := beans.GetGreenBean(ctx, "GB-GAYO-WASHED"); err == nil {
		t.Fatal("expected error for closed database")
	}
	if _, err := beans.GetRoastBatch(ctx, "B-1"); err == nil {
		t.Fatal("expected error for closed database")
	}
	if _, err := beans.ListActiveBatches(ctx, "GB-GAYO-WASHED"); err == nil {
		t.Fatal("expected error for closed database")
	}
	if _, err := beans.ListGreenBeans(ctx); err == nil {
		t.Fatal("expected error for closed database")
	}
	if _, err := beans.CountBatchesForDay(ctx, "BATCH-"); err == nil {
		t.Fatal("expected error for closed database")
	}
	if err := beans.UpdateGreenStock(ctx, "GB-GAYO-WASHED", 1); err == nil {
		t.Fatal("expected error for closed database")
	}
	if err := beans.CreateGreenBean(ctx, &domain.GreenBean{ID: "X"}); err == nil {
		t.Fatal("expected error for closed database")
	}
	if err := beans.CreateRoastBatch(ctx, &domain.RoastBatch{ID: "X"}); err == nil {
		t.Fatal("expected error for closed database")
	}
	if err := beans.DeductGreenAndCreateBatch(ctx, &domain.RoastBatch{ID: "X", GreenBeanID: "Y", GreenWeightMg: 1}); err == nil {
		t.Fatal("expected error for closed database")
	}
}

// TestClosedDBProductOrderErrors verifies catalogue and order failures on
// closed storage.
func TestClosedDBProductOrderErrors(t *testing.T) {
	t.Parallel()
	db := closedTestDB(t)
	ctx := context.Background()
	products := NewProductRepository(db)
	if _, err := products.ListActiveProducts(ctx); err == nil {
		t.Fatal("expected error for closed database")
	}
	if _, err := products.GetProduct(ctx, "P-1"); err == nil {
		t.Fatal("expected error for closed database")
	}
	if _, err := products.GetRecipes(ctx, "P-1"); err == nil {
		t.Fatal("expected error for closed database")
	}
	orders := NewOrderRepository(db)
	if _, err := orders.GetOrder(ctx, "ORD-1"); err == nil {
		t.Fatal("expected error for closed database")
	}
	if _, err := orders.ListOrdersSince(ctx, orderTestTime()); err == nil {
		t.Fatal("expected error for closed database")
	}
	if _, err := orders.ListDeductionsSince(ctx, orderTestTime()); err == nil {
		t.Fatal("expected error for closed database")
	}
	if _, err := orders.CountOrdersForDay(ctx, "ORD-"); err == nil {
		t.Fatal("expected error for closed database")
	}
	order := &domain.Order{ID: "ORD-X", OrderType: domain.OrderTypeB2C, TotalAmount: 1, PaidAmount: 1, PaymentMethod: domain.PaymentCash}
	items := []domain.OrderItem{{ProductID: "P-1", Quantity: 1, Subtotal: 1}}
	if err := orders.CreateOrder(ctx, order, items, map[string]domain.WeightMg{}); err == nil {
		t.Fatal("expected error for closed database")
	}
}
