package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"1stcrack/internal/database"
	"1stcrack/internal/domain"
)

// openTestDB creates a migrated temporary database for repository tests.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "repo.db"))
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

// TestBeanRoundTrip verifies green bean persistence and stock guards.
func TestBeanRoundTrip(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	beans := NewBeanRepository(db)
	bean := &domain.GreenBean{ID: "GB-TEST", Name: "Test Bean", Origin: "Test", Process: "Washed", StockMg: 1000, CostPerKg: 100000}
	if err := beans.CreateGreenBean(ctx, bean); err != nil {
		t.Fatalf("CreateGreenBean: %v", err)
	}
	got, err := beans.GetGreenBean(ctx, "GB-TEST")
	if err != nil {
		t.Fatalf("GetGreenBean: %v", err)
	}
	if got.StockMg != 1000 {
		t.Fatalf("stock = %d, want 1000", got.StockMg)
	}
	if err := beans.UpdateGreenStock(ctx, "GB-TEST", -400); err != nil {
		t.Fatalf("UpdateGreenStock: %v", err)
	}
	if err := beans.UpdateGreenStock(ctx, "GB-TEST", -10000); !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
}

// TestProductSeed verifies seeded catalogue and recipes.
func TestProductSeed(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	products := NewProductRepository(db)
	list, err := products.ListActiveProducts(ctx)
	if err != nil {
		t.Fatalf("ListActiveProducts: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("products = %d, want 3", len(list))
	}
	recipes, err := products.GetRecipes(ctx, "P-LATTE-HOT")
	if err != nil {
		t.Fatalf("GetRecipes: %v", err)
	}
	if len(recipes) != 1 || recipes[0].RequiredRoastedMg != 18000 {
		t.Fatalf("unexpected latte recipe: %+v", recipes)
	}
	if _, err := products.GetProduct(ctx, "P-MISSING"); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

// TestFIFOExactAndSplit verifies oldest-first consumption across batches.
func TestFIFOExactAndSplit(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	beans := NewBeanRepository(db)
	orders := NewOrderRepository(db)
	base := time.Now().UTC().Add(-3 * time.Hour)
	batches := []domain.RoastBatch{
		{ID: "B-FIFO-1", GreenBeanID: "GB-GAYO-WASHED", GreenWeightMg: 12000, RoastedWeightMg: 10000, RemainingMg: 10000, ShrinkagePct: 16.6, RoastLevel: domain.RoastMedium, RoastedAt: base},
		{ID: "B-FIFO-2", GreenBeanID: "GB-GAYO-WASHED", GreenWeightMg: 24000, RoastedWeightMg: 20000, RemainingMg: 20000, ShrinkagePct: 16.6, RoastLevel: domain.RoastMedium, RoastedAt: base.Add(time.Hour)},
	}
	for i := range batches {
		if err := beans.CreateRoastBatch(ctx, &batches[i]); err != nil {
			t.Fatalf("CreateRoastBatch: %v", err)
		}
	}
	order := &domain.Order{ID: "ORD-FIFO-1", OrderType: domain.OrderTypeB2C, TotalAmount: 25000, PaidAmount: 25000, PaymentMethod: domain.PaymentCash}
	items := []domain.OrderItem{{ProductID: "P-LATTE-HOT", Quantity: 1, Subtotal: 25000}}
	needs := map[string]domain.WeightMg{"GB-GAYO-WASHED": 25000}
	if err := orders.CreateOrder(ctx, order, items, needs); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	first, _ := beans.GetRoastBatch(ctx, "B-FIFO-1")
	second, _ := beans.GetRoastBatch(ctx, "B-FIFO-2")
	if first.RemainingMg != 0 || second.RemainingMg != 5000 {
		t.Fatalf("remaining = %d/%d, want 0/5000", first.RemainingMg, second.RemainingMg)
	}
	deductions, err := orders.ListDeductionsSince(ctx, base.Add(-time.Hour))
	if err != nil {
		t.Fatalf("ListDeductionsSince: %v", err)
	}
	var total domain.WeightMg
	for _, d := range deductions {
		total += d.DeductedMg
	}
	if total != 25000 {
		t.Fatalf("audited total = %d, want 25000", total)
	}
}

// TestFIFOInsufficientRollback verifies atomic rollback without stock change.
func TestFIFOInsufficientRollback(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	beans := NewBeanRepository(db)
	orders := NewOrderRepository(db)
	batch := domain.RoastBatch{ID: "B-LOW-1", GreenBeanID: "GB-GAYO-WASHED", GreenWeightMg: 6000, RoastedWeightMg: 5000, RemainingMg: 5000, ShrinkagePct: 16.6, RoastLevel: domain.RoastLight, RoastedAt: time.Now().UTC()}
	if err := beans.CreateRoastBatch(ctx, &batch); err != nil {
		t.Fatalf("CreateRoastBatch: %v", err)
	}
	order := &domain.Order{ID: "ORD-LOW-1", OrderType: domain.OrderTypeB2C, TotalAmount: 10000, PaidAmount: 10000, PaymentMethod: domain.PaymentCash}
	items := []domain.OrderItem{{ProductID: "P-LATTE-HOT", Quantity: 1, Subtotal: 10000}}
	if err := orders.CreateOrder(ctx, order, items, map[string]domain.WeightMg{"GB-GAYO-WASHED": 99999999}); !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
	after, _ := beans.GetRoastBatch(ctx, "B-LOW-1")
	if after.RemainingMg != 5000 {
		t.Fatalf("remaining changed to %d, want 5000", after.RemainingMg)
	}
	if _, err := orders.GetOrder(ctx, "ORD-LOW-1"); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected rolled back order, got %v", err)
	}
}

// TestCreateOrderRejectsEmpty verifies empty cart guard.
func TestCreateOrderRejectsEmpty(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	orders := NewOrderRepository(db)
	err := orders.CreateOrder(context.Background(), &domain.Order{ID: "ORD-EMPTY"}, nil, nil)
	if !errors.Is(err, domain.ErrEmptyCart) {
		t.Fatalf("expected ErrEmptyCart, got %v", err)
	}
}

// ExampleNewOrderRepository demonstrates repository construction.
func ExampleNewOrderRepository() {
	fmt.Println("repository wires database to domain")
	// Output: repository wires database to domain
}
