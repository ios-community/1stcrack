package service

import (
	"fmt"
	"strings"

	"1stcrack/internal/domain"
)

// ReceiptWidth is the fixed receipt line width in characters.
//
// The value targets 32-column thermal printers and plain text files.
const ReceiptWidth = 32

// ReceiptLine represents a printed product line on a receipt.
type ReceiptLine struct {
	// Name is the product display name.
	Name string
	// Quantity is the number of units sold.
	Quantity int
	// Subtotal is the line total in Rupiah.
	Subtotal domain.MoneyIDR
}

// GenerateReceipt renders a fixed-width receipt for display and file export.
//
// The order parameter supplies identifiers, totals, and payment details. The
// lines parameter supplies one entry per product. The change parameter is the
// overpayment amount. When filePath is non-empty a File reference line is
// appended so the terminal preview and the exported text stay identical.
//
// It returns the receipt as a single string with newline separators.
func GenerateReceipt(order *domain.Order, lines []ReceiptLine, change domain.MoneyIDR, filePath string) string {
	var b strings.Builder
	b.WriteString(receiptCenter("1STCRACK - MICRO ROASTERY") + "\n")
	b.WriteString(strings.Repeat("-", ReceiptWidth) + "\n")
	b.WriteString(receiptRow(order.ID, order.OrderType) + "\n")
	b.WriteString(receiptRow(receiptDate(order), paymentLabel(order.PaymentMethod)) + "\n")
	b.WriteString(strings.Repeat("-", ReceiptWidth) + "\n")
	for _, line := range lines {
		b.WriteString(receiptRow(fmt.Sprintf("%s x%d", line.Name, line.Quantity), line.Subtotal.String()) + "\n")
	}
	b.WriteString(strings.Repeat("-", ReceiptWidth) + "\n")
	b.WriteString(receiptRow("Total", order.TotalAmount.String()) + "\n")
	b.WriteString(receiptRow(paymentLabel(order.PaymentMethod), order.PaidAmount.String()) + "\n")
	b.WriteString(receiptRow("Change", change.String()) + "\n")
	b.WriteString(strings.Repeat("-", ReceiptWidth) + "\n")
	b.WriteString(receiptCenter("Thank you!") + "\n")
	if filePath != "" {
		b.WriteString("File: " + filePath + "\n")
	}
	return b.String()
}

// receiptRow aligns a label and value within the fixed receipt width.
func receiptRow(left string, right string) string {
	space := ReceiptWidth - len(left) - len(right)
	if space < 1 {
		keep := ReceiptWidth - len(right) - 1
		keep = max(keep, 0)
		left = left[:keep]
		space = 1
	}
	return left + strings.Repeat(" ", space) + right
}

// receiptCenter centres text within the fixed receipt width.
func receiptCenter(text string) string {
	if len(text) >= ReceiptWidth {
		return text[:ReceiptWidth]
	}
	pad := (ReceiptWidth - len(text)) / 2
	return strings.Repeat(" ", pad) + text
}

// receiptDate formats the order timestamp for printing.
func receiptDate(order *domain.Order) string {
	if order.CreatedAt.IsZero() {
		return "-"
	}
	return order.CreatedAt.Format("02-01-2006 15:04")
}

// paymentLabel converts a payment code into a printed label.
func paymentLabel(method string) string {
	switch method {
	case domain.PaymentCash:
		return "Cash"
	case domain.PaymentQRIS:
		return "QRIS"
	case domain.PaymentTransfer:
		return "Transfer"
	default:
		return method
	}
}
