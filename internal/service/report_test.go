package service

import (
	"context"
	"testing"

	"1stcrack/internal/domain"
)

// TestSummariseDayService verifies daily aggregation in the service layer.
func TestSummariseDayService(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	roasts := NewRoastingService(db)
	roastForTest(t, roasts, "GB-GAYO-WASHED", 1000000, 850000)
	orders := NewOrderService(db)
	if _, err := orders.ProcessCheckout(ctx, CheckoutRequest{Items: []CartItem{{ProductID: "P-LATTE-HOT", Quantity: 1}}, PaidAmount: 25000, PaymentMethod: domain.PaymentCash}); err != nil {
		t.Fatalf("ProcessCheckout: %v", err)
	}
	summary, err := NewReporter(db).SummariseDay(ctx)
	if err != nil {
		t.Fatalf("SummariseDay: %v", err)
	}
	if summary.Revenue != 25000 || summary.Count != 1 || summary.ConsumedMg != 18000 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.ByMethod[domain.PaymentCash] != 25000 || len(summary.PerBatch) != 1 {
		t.Fatalf("unexpected breakdown: %+v", summary)
	}
}
