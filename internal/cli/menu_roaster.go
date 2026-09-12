package cli

import (
	"context"
	"fmt"
	"strings"

	"1stcrack/internal/domain"
	"1stcrack/internal/repository"
)

// roastLevels lists selectable roast profiles in menu order.
var roastLevels = []string{domain.RoastLight, domain.RoastMedium, domain.RoastDark}

// menuRoaster runs the head roaster role menu.
//
// It reports true when the program must quit and false to return to roles.
func (c *commander) menuRoaster(ctx context.Context, p *prompter) bool {
	for {
		_, _ = fmt.Fprintln(c.stdout, "\n--- Head Roaster ---")
		_, _ = fmt.Fprintln(c.stdout, "  1. Record roast batch")
		_, _ = fmt.Fprintln(c.stdout, "  2. View green beans")
		_, _ = fmt.Fprintln(c.stdout, "  9. Switch role")
		_, _ = fmt.Fprintln(c.stdout, "  0. Exit")
		sel, ok := p.askNumber("Select [1-2, 9, 0]: ", 0, 1, 2, 9)
		if !ok {
			return true
		}
		switch sel {
		case 1:
			c.flowRoast(ctx, p)
		case 2:
			if err := c.runBeans(nil); err != nil {
				_, _ = fmt.Fprintf(c.stdout, "error: %v\n", err)
			}
		case 9:
			return false
		default:
			return true
		}
	}
}

// flowRoast guides one roast batch recording with live validation.
func (c *commander) flowRoast(ctx context.Context, p *prompter) {
	beans, err := repository.NewBeanRepository(c.db).ListGreenBeans(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(c.stdout, "error: %v\n", err)
		return
	}
	_, _ = fmt.Fprintln(c.stdout, "\nGreen beans:")
	for i, bean := range beans {
		_, _ = fmt.Fprintf(c.stdout, "  %d. %s (%s) stock %s\n", i+1, bean.Name, bean.Origin, bean.StockMg.String())
	}
	allowed := make([]int, 0, len(beans))
	for i := range beans {
		allowed = append(allowed, i+1)
	}
	sel, ok := p.askNumber("Select bean [number]: ", allowed...)
	if !ok {
		return
	}
	bean := beans[sel-1]
	green, ok := c.flowWeight(p, "Green weight [e.g. 2kg]: ")
	if !ok {
		return
	}
	roasted, ok := c.flowWeight(p, "Roasted weight [e.g. 1700g]: ")
	if !ok {
		return
	}
	level, ok := c.flowLevel(p)
	if !ok {
		return
	}
	if err := c.runRoast(ctx, []string{"--bean", bean.ID, "--green", green, "--roasted", roasted, "--level", level}); err != nil {
		_, _ = fmt.Fprintf(c.stdout, "error: %v\n", err)
		return
	}
}

// flowWeight reads a positive weight keeping the raw text for the subcommand.
func (c *commander) flowWeight(p *prompter, prompt string) (string, bool) {
	for {
		text, ok := p.line(prompt)
		if !ok {
			return "", false
		}
		if _, err := ParseWeight(text); err != nil {
			_, _ = fmt.Fprintf(c.stdout, "error: %v\n", err)
			continue
		}
		return text, true
	}
}

// flowLevel reads the roast level defaulting to medium.
func (c *commander) flowLevel(p *prompter) (string, bool) {
	for {
		text, ok := p.line("Level [1 Light, 2 Medium, 3 Dark, default 2]: ")
		if !ok {
			return "", false
		}
		if text == "" || text == "2" || strings.EqualFold(text, roastLevels[1]) {
			return roastLevels[1], true
		}
		if text == "1" || strings.EqualFold(text, roastLevels[0]) {
			return roastLevels[0], true
		}
		if text == "3" || strings.EqualFold(text, roastLevels[2]) {
			return roastLevels[2], true
		}
		_, _ = fmt.Fprintln(p.out, "Unknown level: 1, 2, 3, or the name.")
	}
}
