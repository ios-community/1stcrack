package domain

import (
	"math"
	"strconv"
	"strings"
)

// WeightMg represents a coffee weight in milligrams.
//
// It uses int64 to avoid floating-point rounding errors in stock calculations.
// One gram equals 1000 milligrams.
type WeightMg int64

// MoneyIDR represents a monetary amount in Indonesian Rupiah.
//
// It uses int64 to avoid floating-point rounding errors in payment
// calculations.
type MoneyIDR int64

// GramsToMg converts a gram amount to milligrams.
//
// The g parameter represents the weight in grams and may contain fractions.
// Values are rounded to the nearest milligram to preserve precision.
//
// It returns the equivalent weight as [WeightMg].
func GramsToMg(g float64) WeightMg {
	return WeightMg(math.Round(g * 1000.0))
}

// ToGrams converts a milligram amount to grams.
//
// It returns the equivalent weight in grams as a float64 for display purposes.
// Internal calculations should remain in [WeightMg].
func (w WeightMg) ToGrams() float64 {
	return float64(w) / 1000.0
}

// String returns a human-readable gram representation.
//
// It returns a decimal string suffixed with g, for example 18.5g.
func (w WeightMg) String() string {
	return strconv.FormatFloat(w.ToGrams(), 'f', -1, 64) + "g"
}

// String returns a human-readable Rupiah representation.
//
// It returns amounts grouped with dot separators and prefixed with Rp,
// for example Rp135.000. Negative amounts retain a leading minus sign.
func (m MoneyIDR) String() string {
	v := int64(m)
	if v < 0 {
		return "-" + MoneyIDR(-v).String()
	}
	s := strconv.FormatInt(v, 10)
	if len(s) <= 3 {
		return "Rp" + s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
		b.WriteByte('.')
	}
	for i := rem; i < len(s); i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < len(s) {
			b.WriteByte('.')
		}
	}
	return "Rp" + b.String()
}
