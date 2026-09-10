package domain

import (
	"fmt"
	"testing"
)

// TestGramsToMg verifies gram to milligram conversion precision.
func TestGramsToMg(t *testing.T) {
	t.Parallel()
	if got := GramsToMg(18.5); got != 18500 {
		t.Fatalf("GramsToMg(18.5) = %d, want 18500", got)
	}
}

// TestCalcShrinkage verifies the weight loss formula and validation.
func TestCalcShrinkage(t *testing.T) {
	t.Parallel()
	got, err := CalcShrinkage(1000, 850)
	if err != nil {
		t.Fatalf("CalcShrinkage returned error: %v", err)
	}
	if got != 15.0 {
		t.Fatalf("CalcShrinkage(1000,850) = %v, want 15", got)
	}
	if _, err := CalcShrinkage(1000, 1200); err != ErrInvalidRoastWeight {
		t.Fatalf("expected ErrInvalidRoastWeight, got %v", err)
	}
}

// ExampleGramsToMg demonstrates gram conversion for an espresso dose.
func ExampleGramsToMg() {
	fmt.Println(int64(GramsToMg(18.5)))
	// Output: 18500
}

// ExampleCalcShrinkage demonstrates shrinkage calculation for a roast batch.
func ExampleCalcShrinkage() {
	pct, _ := CalcShrinkage(1000, 850)
	fmt.Println(pct == 15.0)
	// Output: true
}
