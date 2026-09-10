package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"1stcrack/internal/domain"
	"1stcrack/internal/repository"
	"1stcrack/internal/service"
)

// itemFlags collects repeated --item PRODUCT:QTY flags into cart lines.
type itemFlags []service.CartItem

// String returns a human-readable summary of collected items.
func (f *itemFlags) String() string {
	parts := make([]string, 0, len(*f))
	for _, item := range *f {
		parts = append(parts, item.ProductID+":"+strconv.Itoa(item.Quantity))
	}
	return strings.Join(parts, ", ")
}

// Set parses one PRODUCT:QTY value and appends it to the collection.
func (f *itemFlags) Set(s string) error {
	id, qtyText, ok := strings.Cut(s, ":")
	if !ok || id == "" {
		return fmt.Errorf("invalid --item %q: want PRODUCT:QTY like P-LATTE-HOT:2", s)
	}
	qty, err := strconv.Atoi(qtyText)
	if err != nil || qty <= 0 {
		return fmt.Errorf("invalid --item %q: quantity must be a positive integer", s)
	}
	*f = append(*f, service.CartItem{ProductID: id, Quantity: qty})
	return nil
}

// runSell checks out a sale, prints the receipt, and exports the text file.
func (c *commander) runSell(ctx context.Context, args []string) error {
	fs := c.newFlagSet("sell", `  1stcrack sell --item P-LATTE-HOT:2 [--item P-BEANS-250:1] --paid 150000 [--b2b --customer "Kafe X"] [--method CASH]

  Payment methods: CASH, QRIS, TRANSFER.
`)
	var items itemFlags
	fs.Var(&items, "item", "product and quantity as ID:QTY (repeatable, required)")
	paidText := fs.String("paid", "", "amount tendered in Rupiah (required)")
	wholesale := fs.Bool("b2b", false, "wholesale sale requiring --customer")
	customer := fs.String("customer", "", "buyer name for B2B sales")
	method := fs.String("method", domain.PaymentCash, "payment method")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(items) == 0 {
		return fmt.Errorf("missing --item: add at least one PRODUCT:QTY")
	}
	paid, err := ParseMoney(*paidText)
	if err != nil {
		return err
	}
	orderType := domain.OrderTypeB2C
	if *wholesale {
		orderType = domain.OrderTypeB2B
	}
	products := repository.NewProductRepository(c.db)
	lines := make([]service.ReceiptLine, 0, len(items))
	for _, item := range items {
		product, err := products.GetProduct(ctx, item.ProductID)
		if err != nil {
			return err
		}
		lines = append(lines, service.ReceiptLine{Name: product.Name, Quantity: item.Quantity, Subtotal: product.Price * domain.MoneyIDR(item.Quantity)})
	}
	order, err := service.NewOrderService(c.db).ProcessCheckout(ctx, service.CheckoutRequest{
		OrderType:     orderType,
		CustomerName:  *customer,
		Items:         []service.CartItem(items),
		PaidAmount:    paid,
		PaymentMethod: *method,
	})
	if err != nil {
		return err
	}
	path := filepath.Join(c.receiptsDir, order.ID+".txt")
	receipt := service.GenerateReceipt(order, lines, paid-order.TotalAmount, path)
	if err := os.MkdirAll(c.receiptsDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(receipt), 0o644); err != nil {
		return err
	}
	fmt.Fprint(c.stdout, receipt)
	return nil
}
