package domain

import "time"

// Roast level identifiers supported by the roastery.
const (
	// RoastLight represents a light roast profile.
	RoastLight = "Light"
	// RoastMedium represents a medium roast profile.
	RoastMedium = "Medium"
	// RoastDark represents a dark roast profile.
	RoastDark = "Dark"
)

// GreenBean represents a raw coffee stock record.
//
// It stores origin details and the remaining raw weight. Monetary values use
// [MoneyIDR] per kilogram.
type GreenBean struct {
	// ID is the unique identifier of the green bean.
	ID string
	// Name is the commercial name of the green bean.
	Name string
	// Origin is the growing region of the green bean.
	Origin string
	// Process is the post-harvest process, for example Washed or Natural.
	Process string
	// StockMg is the remaining raw stock in milligrams.
	StockMg WeightMg
	// CostPerKg is the purchase price per kilogram in Rupiah.
	CostPerKg MoneyIDR
	// CreatedAt is the registration timestamp.
	CreatedAt time.Time
}

// RoastBatch represents a single roasting production batch.
//
// It records input and output weights, the remaining roasted stock, and the
// calculated shrinkage percentage. Batches are consumed in First-In
// First-Out order by RoastedAt.
type RoastBatch struct {
	// ID is the unique batch identifier in BATCH-YYYYMMDD-XXX format.
	ID string
	// GreenBeanID references the source [GreenBean].
	GreenBeanID string
	// GreenWeightMg is the raw input weight in milligrams.
	GreenWeightMg WeightMg
	// RoastedWeightMg is the roasted output weight in milligrams.
	RoastedWeightMg WeightMg
	// RemainingMg is the remaining roasted stock in milligrams.
	RemainingMg WeightMg
	// ShrinkagePct is the weight loss percentage at roast time.
	ShrinkagePct float64
	// RoastLevel is one of Light, Medium, or Dark.
	RoastLevel string
	// RoastedAt is the production timestamp used for FIFO ordering.
	RoastedAt time.Time
}

// CalcShrinkage calculates the roast weight loss percentage.
//
// The green parameter is the raw input weight and the roasted parameter is
// the roasted output weight, both in milligrams.
//
// It returns the shrinkage percentage computed as (green-roasted)/green*100.
//
// It returns [ErrInvalidRoastWeight] if green is not positive or if roasted
// exceeds green.
func CalcShrinkage(green WeightMg, roasted WeightMg) (float64, error) {
	if green <= 0 {
		return 0, ErrInvalidRoastWeight
	}
	if roasted < 0 || roasted > green {
		return 0, ErrInvalidRoastWeight
	}
	return float64(green-roasted) / float64(green) * 100.0, nil
}
