package cli

import (
	"context"
	"fmt"
)

// menuOwner runs the owner role menu with every screen.
//
// It reports true when the program must quit and false to return to roles.
func (c *commander) menuOwner(ctx context.Context, p *prompter) bool {
	for {
		_, _ = fmt.Fprintln(c.stdout, "\n--- Owner / Manager ---")
		_, _ = fmt.Fprintln(c.stdout, "  1. Sell")
		_, _ = fmt.Fprintln(c.stdout, "  2. Roasting")
		_, _ = fmt.Fprintln(c.stdout, "  3. Stock")
		_, _ = fmt.Fprintln(c.stdout, "  4. Daily report")
		_, _ = fmt.Fprintln(c.stdout, "  9. Switch role")
		_, _ = fmt.Fprintln(c.stdout, "  0. Exit")
		sel, ok := p.askNumber("Select [1-4, 9, 0]: ", 0, 1, 2, 3, 4, 9)
		if !ok {
			return true
		}
		switch sel {
		case 1:
			c.flowSell(ctx, p)
		case 2:
			c.flowRoast(ctx, p)
		case 3:
			if err := c.runStock(ctx, nil); err != nil {
				_, _ = fmt.Fprintf(c.stdout, "error: %v\n", err)
			}
		case 4:
			if err := c.runReport(ctx, nil); err != nil {
				_, _ = fmt.Fprintf(c.stdout, "error: %v\n", err)
			}
		case 9:
			return false
		default:
			return true
		}
	}
}
