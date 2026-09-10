package domain

import "testing"

// TestWeightDisplay verifies gram conversions and weight formatting.
func TestWeightDisplay(t *testing.T) {
	t.Parallel()
	w := WeightMg(18500)
	if w.ToGrams() != 18.5 {
		t.Fatalf("ToGrams = %v, want 18.5", w.ToGrams())
	}
	if w.String() != "18.5 g" {
		t.Fatalf("String = %q, want 18.5 g", w.String())
	}
}

// TestMoneyDisplay verifies Rupiah grouping and negative handling.
func TestMoneyDisplay(t *testing.T) {
	t.Parallel()
	cases := []struct {
		amount MoneyIDR
		want   string
	}{
		{0, "Rp0"},
		{500, "Rp500"},
		{135000, "Rp135.000"},
		{1500000, "Rp1.500.000"},
		{-5000, "-Rp5.000"},
	}
	for _, c := range cases {
		if got := c.amount.String(); got != c.want {
			t.Fatalf("MoneyIDR(%d).String() = %q, want %q", int64(c.amount), got, c.want)
		}
	}
}

// TestCalcShrinkageEdges verifies boundary and invalid inputs.
func TestCalcShrinkageEdges(t *testing.T) {
	t.Parallel()
	if _, err := CalcShrinkage(0, 0); err != ErrInvalidRoastWeight {
		t.Fatalf("green=0: got %v, want ErrInvalidRoastWeight", err)
	}
	if _, err := CalcShrinkage(1000, -10); err != ErrInvalidRoastWeight {
		t.Fatalf("negative roasted: got %v, want ErrInvalidRoastWeight", err)
	}
	got, err := CalcShrinkage(1000, 0)
	if err != nil || got != 100.0 {
		t.Fatalf("total loss = %v, %v; want 100", got, err)
	}
	got, err = CalcShrinkage(1000, 1000)
	if err != nil || got != 0.0 {
		t.Fatalf("no loss = %v, %v; want 0", got, err)
	}
}
