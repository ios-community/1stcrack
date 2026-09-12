package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"1stcrack/internal/domain"
	"1stcrack/internal/repository"
	"1stcrack/internal/service"
)

// menuCashier runs the cashier role menu.
//
// It reports true when the program must quit and false to return to roles.
func (c *commander) menuCashier(ctx context.Context, p *prompter) bool {
	for {
		_, _ = fmt.Fprintln(c.stdout, "\n--- Cashier / Barista ---")
		_, _ = fmt.Fprintln(c.stdout, "  1. Sell")
		_, _ = fmt.Fprintln(c.stdout, "  2. View catalogue")
		_, _ = fmt.Fprintln(c.stdout, "  9. Switch role")
		_, _ = fmt.Fprintln(c.stdout, "  0. Exit")
		sel, ok := p.askNumber("Select [1-2, 9, 0]: ", 0, 1, 2, 9)
		if !ok {
			return true
		}
		switch sel {
		case 1:
			c.flowSell(ctx, p)
		case 2:
			if err := c.runProducts(nil); err != nil {
				_, _ = fmt.Fprintf(c.stdout, "error: %v\n", err)
			}
		case 9:
			return false
		default:
			return true
		}
	}
}

// flowSell guides one sale from catalogue to printed receipt.
func (c *commander) flowSell(ctx context.Context, p *prompter) {
	products, err := repository.NewProductRepository(c.db).ListActiveProducts(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(c.stdout, "error: %v\n", err)
		return
	}
	_, _ = fmt.Fprintln(c.stdout, "\nCatalogue:")
	for i, product := range products {
		_, _ = fmt.Fprintf(c.stdout, "  %d. %s — %s\n", i+1, product.Name, product.Price.String())
	}
	var items []service.CartItem
	allowed := make([]int, 0, len(products)+1)
	for i := range products {
		allowed = append(allowed, i+1)
	}
	allowed = append(allowed, 0)
	for {
		sel, ok := p.askNumber("Select product number (0 = done): ", allowed...)
		if !ok {
			return
		}
		if sel == 0 {
			break
		}
		qty, ok := c.flowQuantity(p, products[sel-1])
		if !ok {
			return
		}
		items = append(items, service.CartItem{ProductID: products[sel-1].ID, Quantity: qty})
	}
	if len(items) == 0 {
		_, _ = fmt.Fprintln(c.stdout, "Cart is empty.")
		return
	}
	total, err := service.NewOrderService(c.db).CalculateTotal(ctx, items)
	if err != nil {
		_, _ = fmt.Fprintf(c.stdout, "error: %v\n", err)
		return
	}
	_, _ = fmt.Fprintf(c.stdout, "Total: %s\n", total.String())
	wholesale, ok := p.askYesNo("B2B (wholesale)? [y/N]: ")
	if !ok {
		return
	}
	customer := ""
	if wholesale {
		for {
			name, ok := p.line("Cafe name: ")
			if !ok {
				return
			}
			if name != "" {
				customer = name
				break
			}
			_, _ = fmt.Fprintln(c.stdout, "Cafe name is required for B2B.")
		}
	}
	var paid domain.MoneyIDR
	for {
		text, ok := p.line("Pay: ")
		if !ok {
			return
		}
		paid, err = ParseMoney(text)
		if err != nil {
			_, _ = fmt.Fprintf(c.stdout, "error: %v\n", err)
			continue
		}
		if paid < total {
			_, _ = fmt.Fprintf(c.stdout, "error: %v\n", domain.ErrNegativePayment)
			continue
		}
		break
	}
	method := c.flowMethod(p)
	if method == "" {
		return
	}
	sellArgs := []string{"--paid", strconv.FormatInt(int64(paid), 10), "--method", method}
	if wholesale {
		sellArgs = append(sellArgs, "--b2b", "--customer", customer)
	}
	for _, item := range items {
		sellArgs = append(sellArgs, "--item", item.ProductID+":"+strconv.Itoa(item.Quantity))
	}
	if err := c.runSell(ctx, sellArgs); err != nil {
		_, _ = fmt.Fprintf(c.stdout, "error: %v\n", err)
		return
	}
	again, ok := p.askYesNo("Sell again? [y/N]: ")
	if !ok {
		return
	}
	if again {
		c.flowSell(ctx, p)
	}
}

// flowQuantity reads a positive quantity for the chosen product.
func (c *commander) flowQuantity(p *prompter, product domain.Product) (int, bool) {
	for {
		text, ok := p.line("Quantity: ")
		if !ok {
			return 0, false
		}
		qty, err := strconv.Atoi(text)
		if err != nil || qty <= 0 {
			_, _ = fmt.Fprintln(p.out, "Quantity must be a positive integer.")
			continue
		}
		_, _ = fmt.Fprintf(c.stdout, "+ %s x%d = %s\n", product.Name, qty, (product.Price * domain.MoneyIDR(qty)).String())
		return qty, true
	}
}

// flowMethod reads the payment method defaulting to cash.
func (c *commander) flowMethod(p *prompter) string {
	for {
		text, ok := p.line("Method [CASH/QRIS/TRANSFER, default CASH]: ")
		if !ok {
			return ""
		}
		method := "CASH"
		if text != "" {
			method = strings.ToUpper(text)
		}
		switch method {
		case domain.PaymentCash, domain.PaymentQRIS, domain.PaymentTransfer:
			return method
		default:
			_, _ = fmt.Fprintln(p.out, "Unknown method: CASH, QRIS, or TRANSFER.")
		}
	}
}
