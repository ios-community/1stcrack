package cli

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"1stcrack/internal/domain"
	"1stcrack/internal/repository"
	"1stcrack/internal/service"
)

// runStock prints FIFO roast batches with alert markers.
func (c *commander) runStock(ctx context.Context, args []string) error {
	fs := c.newFlagSet("stock", `  1stcrack stock [--threshold 500g]

  The threshold accepts mg, g (default), and kg suffixes.
`)
	thresholdText := fs.String("threshold", "500g", "low-stock threshold weight")
	if err := fs.Parse(args); err != nil {
		return err
	}
	threshold, err := ParseWeight(*thresholdText)
	if err != nil {
		return err
	}
	beans := repository.NewBeanRepository(c.db)
	list, err := beans.ListGreenBeans(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	w := tabwriter.NewWriter(c.stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "BATCH\tBEAN\tLEFT\tAGE\tSTATUS")
	for _, bean := range list {
		batches, err := beans.ListActiveBatches(ctx, bean.ID)
		if err != nil {
			return err
		}
		for _, batch := range batches {
			age := 0
			if !batch.RoastedAt.IsZero() && now.After(batch.RoastedAt) {
				age = int(now.Sub(batch.RoastedAt).Hours() / 24)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%dd\t%s\n", batch.ID, bean.Name, batch.RemainingMg.String(), age, stockMarker(service.AlertLevelFor(batch.RemainingMg, threshold)))
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Fprintln(c.stdout, "\nGREEN BEANS")
	w = tabwriter.NewWriter(c.stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTOCK\tSTATUS")
	for _, bean := range list {
		fmt.Fprintf(w, "%s\t%s\t%s\n", bean.ID, bean.StockMg.String(), stockMarker(service.AlertLevelFor(bean.StockMg, threshold)))
	}
	return w.Flush()
}

// stockMarker converts an alert state into a compact marker.
func stockMarker(level service.AlertLevel) string {
	switch level {
	case service.AlertCritical:
		return "CRIT"
	case service.AlertLow:
		return "LOW"
	default:
		return "OK"
	}
}

// runReport prints today's revenue and consumption summary.
func (c *commander) runReport(ctx context.Context, args []string) error {
	fs := c.newFlagSet("report", "  1stcrack report\n")
	if err := fs.Parse(args); err != nil {
		return err
	}
	summary, err := service.NewReporter(c.db).SummariseDay(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.stdout, "TODAY  revenue %s  tx %d\n", summary.Revenue.String(), summary.Count)
	for _, method := range []string{domain.PaymentCash, domain.PaymentQRIS, domain.PaymentTransfer} {
		if amount, ok := summary.ByMethod[method]; ok {
			fmt.Fprintf(c.stdout, "  %s: %s\n", method, amount.String())
		}
	}
	fmt.Fprintf(c.stdout, "Coffee used: %s\n", summary.ConsumedMg.String())
	for _, batch := range summary.PerBatch {
		fmt.Fprintf(c.stdout, "  %s: %s\n", batch.BatchID, batch.ConsumedMg.String())
	}
	return nil
}
