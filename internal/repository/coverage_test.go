package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"1stcrack/internal/domain"
)

// TestListActiveBatchesOrder verifies FIFO ordering and drained exclusion.
func TestListActiveBatchesOrder(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	beans := NewBeanRepository(db)
	base := time.Now().UTC().Add(-3 * time.Hour)
	for _, b := range []domain.RoastBatch{
		{ID: "B-ORD-2", GreenBeanID: "GB-GAYO-WASHED", GreenWeightMg: 12000, RoastedWeightMg: 10000, RemainingMg: 4000, ShrinkagePct: 16.6, RoastLevel: domain.RoastMedium, RoastedAt: base.Add(2 * time.Hour)},
		{ID: "B-ORD-1", GreenBeanID: "GB-GAYO-WASHED", GreenWeightMg: 12000, RoastedWeightMg: 10000, RemainingMg: 9000, ShrinkagePct: 16.6, RoastLevel: domain.RoastMedium, RoastedAt: base},
		{ID: "B-ORD-0", GreenBeanID: "GB-GAYO-WASHED", GreenWeightMg: 12000, RoastedWeightMg: 10000, RemainingMg: 0, ShrinkagePct: 16.6, RoastLevel: domain.RoastMedium, RoastedAt: base.Add(-time.Hour)},
	} {
		if err := beans.CreateRoastBatch(ctx, &b); err != nil {
			t.Fatalf("CreateRoastBatch: %v", err)
		}
	}
	list, err := beans.ListActiveBatches(ctx, "GB-GAYO-WASHED")
	if err != nil {
		t.Fatalf("ListActiveBatches: %v", err)
	}
	if len(list) != 2 || list[0].ID != "B-ORD-1" || list[1].ID != "B-ORD-2" {
		t.Fatalf("unexpected FIFO order: %+v", list)
	}
}

// TestBeanListAndCounts verifies catalogue listing and day counters.
func TestBeanListAndCounts(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	beans := NewBeanRepository(db)
	orders := NewOrderRepository(db)
	list, err := beans.ListGreenBeans(ctx)
	if err != nil || len(list) != 2 {
		t.Fatalf("ListGreenBeans = %d, %v; want 2", len(list), err)
	}
	if n, err := beans.CountBatchesForDay(ctx, "BATCH-20990101-"); err != nil || n != 0 {
		t.Fatalf("CountBatchesForDay = %d, %v; want 0", n, err)
	}
	if n, err := orders.CountOrdersForDay(ctx, "ORD-20990101-"); err != nil || n != 0 {
		t.Fatalf("CountOrdersForDay = %d, %v; want 0", n, err)
	}
	order := &domain.Order{ID: "ORD-CNT-1", OrderType: domain.OrderTypeB2C, TotalAmount: 1000, PaidAmount: 1000, PaymentMethod: domain.PaymentCash}
	items := []domain.OrderItem{{ProductID: "P-LATTE-HOT", Quantity: 1, Subtotal: 1000}}
	if err := orders.CreateOrder(ctx, order, items, map[string]domain.WeightMg{}); err != nil {
		t.Fatalf("CreateOrder without needs: %v", err)
	}
	got, err := orders.GetOrder(ctx, "ORD-CNT-1")
	if err != nil || got.TotalAmount != 1000 {
		t.Fatalf("GetOrder = %+v, %v", got, err)
	}
	since := time.Now().UTC().Add(-time.Hour)
	found, err := orders.ListOrdersSince(ctx, since)
	if err != nil || len(found) != 1 {
		t.Fatalf("ListOrdersSince = %d, %v; want 1", len(found), err)
	}
	deductions, err := orders.ListDeductionsSince(ctx, since)
	if err != nil || len(deductions) != 0 {
		t.Fatalf("ListDeductionsSince = %d, %v; want 0", len(deductions), err)
	}
}

// TestBeanGuards verifies nil and missing-entity error paths.
func TestBeanGuards(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	if err := NewBeanRepository(nil).CreateGreenBean(ctx, &domain.GreenBean{}); err == nil {
		t.Fatal("expected error for nil database")
	}
	beans := NewBeanRepository(db)
	if err := beans.CreateGreenBean(ctx, nil); err == nil {
		t.Fatal("expected error for nil bean")
	}
	if err := beans.CreateRoastBatch(ctx, nil); err == nil {
		t.Fatal("expected error for nil batch")
	}
	if _, err := beans.GetGreenBean(ctx, "GB-MISSING"); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
	if _, err := beans.GetRoastBatch(ctx, "B-MISSING"); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
	if err := beans.UpdateGreenStock(ctx, "GB-MISSING", 100); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

// TestDeductBatchAtomically verifies direct atomic roast recording.
func TestDeductBatchAtomically(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	beans := NewBeanRepository(db)
	batch := &domain.RoastBatch{ID: "B-ATOM-1", GreenBeanID: "GB-GAYO-WASHED", GreenWeightMg: 100000, RoastedWeightMg: 85000, RemainingMg: 85000, ShrinkagePct: 15.0, RoastLevel: domain.RoastMedium}
	if err := beans.DeductGreenAndCreateBatch(ctx, batch); err != nil {
		t.Fatalf("DeductGreenAndCreateBatch: %v", err)
	}
	bean, _ := beans.GetGreenBean(ctx, "GB-GAYO-WASHED")
	if bean.StockMg != 4900000 {
		t.Fatalf("stock = %d, want 4900000", bean.StockMg)
	}
	huge := &domain.RoastBatch{ID: "B-ATOM-2", GreenBeanID: "GB-GAYO-WASHED", GreenWeightMg: 999999999, RoastedWeightMg: 1, RemainingMg: 1, ShrinkagePct: 99.9, RoastLevel: domain.RoastDark}
	if err := beans.DeductGreenAndCreateBatch(ctx, huge); !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
	if err := beans.DeductGreenAndCreateBatch(ctx, nil); err == nil {
		t.Fatal("expected error for nil batch")
	}
}

// TestOrderGuards verifies nil-database and nil-order error paths.
func TestOrderGuards(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	if err := NewOrderRepository(nil).CreateOrder(ctx, &domain.Order{}, []domain.OrderItem{{}}, nil); err == nil {
		t.Fatal("expected error for nil database")
	}
	db := openTestDB(t)
	if err := NewOrderRepository(db).CreateOrder(ctx, nil, []domain.OrderItem{{}}, nil); err == nil {
		t.Fatal("expected error for nil order")
	}
}
