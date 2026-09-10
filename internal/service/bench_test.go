package service

import (
	"testing"

	"1stcrack/internal/domain"
)

// BenchmarkGenerateReceipt measures receipt rendering latency.
func BenchmarkGenerateReceipt(b *testing.B) {
	order := &domain.Order{ID: "ORD-BENCH-1", OrderType: domain.OrderTypeB2C, TotalAmount: 135000, PaidAmount: 150000, PaymentMethod: domain.PaymentCash}
	lines := []ReceiptLine{{Name: "Hot Latte", Quantity: 2, Subtotal: 50000}, {Name: "Beans 250g", Quantity: 1, Subtotal: 85000}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateReceipt(order, lines, 15000, "receipts/ORD-BENCH-1.txt")
	}
}
