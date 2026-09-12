package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// prompter reads interactive input lines for the guided menu.
type prompter struct {
	// User input source.
	in *bufio.Reader
	// Prompt and result output.
	out io.Writer
}

// newPrompter creates a prompter over the given streams.
func newPrompter(stdin io.Reader, stdout io.Writer) *prompter {
	return &prompter{in: bufio.NewReader(stdin), out: stdout}
}

// line prints a prompt and reads one trimmed line.
//
// It reports false on end of input so callers exit cleanly.
func (p *prompter) line(prompt string) (string, bool) {
	_, _ = fmt.Fprint(p.out, prompt)
	text, err := p.in.ReadString('\n')
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(text), true
}

// askNumber prompts until input is one of the allowed integers.
//
// It reports false on end of input so callers exit cleanly.
func (p *prompter) askNumber(prompt string, allowed ...int) (int, bool) {
	for {
		text, ok := p.line(prompt)
		if !ok {
			return 0, false
		}
		v, err := strconv.Atoi(text)
		if err == nil {
			for _, a := range allowed {
				if v == a {
					return v, true
				}
			}
		}
		_, _ = fmt.Fprintln(p.out, "Unknown choice, try again.")
	}
}

// askYesNo asks a yes/no question defaulting to no.
//
// It reports false on end of input so callers exit cleanly.
func (p *prompter) askYesNo(prompt string) (bool, bool) {
	for {
		text, ok := p.line(prompt)
		if !ok {
			return false, false
		}
		switch strings.ToLower(text) {
		case "y", "yes":
			return true, true
		case "", "n", "no":
			return false, true
		default:
			_, _ = fmt.Fprintln(p.out, "Answer y or N.")
		}
	}
}

// runMenu opens the guided role menu and returns the exit code.
//
// Role 1 serves cashiers, role 2 serves roasters, and role 3 serves owners
// with every screen. Option 0 quits from any menu. End of input quits
// cleanly with success.
func (c *commander) runMenu(ctx context.Context) int {
	p := newPrompter(c.stdin, c.stdout)
	_, _ = fmt.Fprintln(c.stdout, "=== 1stcrack ===")
	for {
		_, _ = fmt.Fprintln(c.stdout, "\nSelect role:")
		_, _ = fmt.Fprintln(c.stdout, "  1. Cashier / Barista")
		_, _ = fmt.Fprintln(c.stdout, "  2. Head Roaster")
		_, _ = fmt.Fprintln(c.stdout, "  3. Owner / Manager")
		_, _ = fmt.Fprintln(c.stdout, "  0. Exit")
		sel, ok := p.askNumber("Select [0-3]: ", 0, 1, 2, 3)
		if !ok {
			return 0
		}
		var quit bool
		switch sel {
		case 1:
			quit = c.menuCashier(ctx, p)
		case 2:
			quit = c.menuRoaster(ctx, p)
		case 3:
			quit = c.menuOwner(ctx, p)
		default:
			return 0
		}
		if quit {
			return 0
		}
	}
}
