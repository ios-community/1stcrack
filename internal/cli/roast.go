package cli

import (
	"context"
	"fmt"

	"1stcrack/internal/service"
)

// runRoast records a roast batch from flag input.
func (c *commander) runRoast(ctx context.Context, args []string) error {
	fs := c.newFlagSet("roast", `  1stcrack roast --bean ID --green 1kg --roasted 850g [--level Medium]

  Weights accept mg, g (default), and kg suffixes.
`)
	beanID := fs.String("bean", "", "green bean id (required)")
	green := fs.String("green", "", "raw input weight, e.g. 1kg (required)")
	roasted := fs.String("roasted", "", "roasted output weight, e.g. 850g (required)")
	level := fs.String("level", "Medium", "roast level: Light, Medium, or Dark")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return c.executeRoast(ctx, *beanID, *green, *roasted, *level)
}

// executeRoast validates weights and records the batch through the service
// layer.
func (c *commander) executeRoast(ctx context.Context, beanID string, green string, roasted string, level string) error {
	if beanID == "" {
		return fmt.Errorf("missing --bean: see `1stcrack products` for ids")
	}
	greenMg, err := ParseWeight(green)
	if err != nil {
		return err
	}
	roastedMg, err := ParseWeight(roasted)
	if err != nil {
		return err
	}
	batch, err := service.NewRoastingService(c.db).ExecuteBatch(ctx, service.RoastInput{
		GreenBeanID:     beanID,
		GreenWeightMg:   greenMg,
		RoastedWeightMg: roastedMg,
		RoastLevel:      level,
	})
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(c.stdout, "recorded %s — %.1f%% shrinkage, %s remaining\n", batch.ID, batch.ShrinkagePct, batch.RemainingMg.String())
	return nil
}
