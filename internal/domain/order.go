package domain

import "time"

// Order type identifiers.
const (
	// OrderTypeB2C represents a retail cafe sale.
	OrderTypeB2C = "B2C"
	// OrderTypeB2B represents a wholesale sale to a partner cafe.
	OrderTypeB2B = "B2B"
)

// Payment method identifiers.
const (
	// PaymentCash represents a cash payment.
	PaymentCash = "CASH"
	// PaymentQRIS represents a QRIS payment.
	PaymentQRIS = "QRIS"
	// PaymentTransfer represents a bank transfer payment.
	PaymentTransfer = "TRANSFER"
)

// Order represents a completed checkout transaction.
//
// Totals and payments use [MoneyIDR]. An order aggregates [OrderItem] lines
// and [BatchDeduction] audit lines.
type Order struct {
	// ID is the unique order identifier in ORD-YYYYMMDD-XXXX format.
	ID string
	// OrderType is B2C for retail or B2B for wholesale.
	OrderType string
	// CustomerName is the buyer name, required for B2B wholesale.
	CustomerName string
	// TotalAmount is the order total in Rupiah.
	TotalAmount MoneyIDR
	// PaidAmount is the amount tendered in Rupiah.
	PaidAmount MoneyIDR
	// PaymentMethod is CASH, QRIS, or TRANSFER.
	PaymentMethod string
	// CreatedAt is the checkout timestamp.
	CreatedAt time.Time
}

// OrderItem represents a single product line within an order.
type OrderItem struct {
	// ID is the surrogate identifier of the order line.
	ID int64
	// OrderID references the owning [Order].
	OrderID string
	// ProductID references the sold [Product].
	ProductID string
	// Quantity is the number of units sold and must be positive.
	Quantity int
	// Subtotal is the line total in Rupiah.
	Subtotal MoneyIDR
}

// BatchDeduction represents an audit record of FIFO stock consumption.
//
// It records how many milligrams were taken from a specific roast batch for
// an order, enabling freshness and cost audits.
type BatchDeduction struct {
	// ID is the surrogate identifier of the deduction record.
	ID int64
	// OrderID references the consuming [Order].
	OrderID string
	// RoastBatchID references the consumed [RoastBatch].
	RoastBatchID string
	// DeductedMg is the consumed weight in milligrams.
	DeductedMg WeightMg
}
