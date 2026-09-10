package cli

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"1stcrack/internal/domain"
)

// ParseWeight converts a human weight into milligrams.
//
// Accepted forms are case-insensitive with optional spaces: 850, 850g,
// 2.5kg, and 500mg. A bare number means grams. Supported units are mg, g,
// and kg.
//
// It returns the equivalent weight as [domain.WeightMg].
//
// It returns an error for unknown units and non-positive amounts.
func ParseWeight(input string) (domain.WeightMg, error) {
	s := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(input), " ", ""))
	if s == "" {
		return 0, fmt.Errorf("invalid weight %q: empty", input)
	}
	multiplier := 1000.0
	number := s
	switch {
	case strings.HasSuffix(s, "kg"):
		multiplier = 1000000.0
		number = strings.TrimSuffix(s, "kg")
	case strings.HasSuffix(s, "mg"):
		multiplier = 1.0
		number = strings.TrimSuffix(s, "mg")
	case strings.HasSuffix(s, "g"):
		multiplier = 1000.0
		number = strings.TrimSuffix(s, "g")
	}
	v, err := strconv.ParseFloat(number, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 0, fmt.Errorf("invalid weight %q: want a positive amount like 850g", input)
	}
	return domain.WeightMg(math.Round(v * multiplier)), nil
}

// ParseMoney converts a typed Rupiah amount into money.
//
// Thousand separators such as dots and spaces are ignored, so 50000 and
// 50.000 both mean fifty thousand Rupiah.
//
// It returns an error for empty or non-numeric input.
func ParseMoney(input string) (domain.MoneyIDR, error) {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, input)
	if digits == "" {
		return 0, fmt.Errorf("invalid amount %q: want digits like 50000", input)
	}
	v, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q", input)
	}
	return domain.MoneyIDR(v), nil
}
