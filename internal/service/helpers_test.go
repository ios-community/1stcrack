package service

import (
	"strings"
	"testing"
	"time"

	"1stcrack/internal/domain"
)

// TestReceiptHelpers verifies centering, dates, and payment labels directly.
func TestReceiptHelpers(t *testing.T) {
	t.Parallel()
	if got := receiptCenter(strings.Repeat("X", 40)); len(got) != ReceiptWidth {
		t.Fatalf("centered len = %d, want %d", len(got), ReceiptWidth)
	}
	order := &domain.Order{CreatedAt: time.Date(2026, 2, 14, 10, 2, 0, 0, time.UTC)}
	if got := receiptDate(order); got != "14-02-2026 10:02" {
		t.Fatalf("date = %q, want 14-02-2026 10:02", got)
	}
	if paymentLabel(domain.PaymentQRIS) != "QRIS" || paymentLabel(domain.PaymentTransfer) != "Transfer" {
		t.Fatal("unexpected payment labels")
	}
}
