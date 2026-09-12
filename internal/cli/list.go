package cli

import (
	"context"
	"fmt"
	"text/tabwriter"

	"1stcrack/internal/repository"
)

// runBeans lists green beans with raw stock.
func (c *commander) runBeans(args []string) error {
	fs := c.newFlagSet("beans", "  1stcrack beans\n")
	if err := fs.Parse(args); err != nil {
		return err
	}
	beans, err := repository.NewBeanRepository(c.db).ListGreenBeans(context.Background())
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(c.stdout, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tORIGIN\tPROCESS\tSTOCK\tCOST/KG")
	for _, bean := range beans {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", bean.ID, bean.Name, bean.Origin, bean.Process, bean.StockMg.String(), bean.CostPerKg.String())
	}
	return w.Flush()
}

// runProducts lists sellable products with prices.
func (c *commander) runProducts(args []string) error {
	fs := c.newFlagSet("products", "  1stcrack products\n")
	if err := fs.Parse(args); err != nil {
		return err
	}
	products, err := repository.NewProductRepository(c.db).ListActiveProducts(context.Background())
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(c.stdout, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tCATEGORY\tPRICE")
	for _, product := range products {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", product.ID, product.Name, product.Category, product.Price.String())
	}
	return w.Flush()
}
