package percent

import (
	"errors"
	"math"
	"testing"
)

// near reports whether got is within a small relative tolerance of want,
// absorbing ordinary floating-point noise.
func near(got, want float64) bool {
	return math.Abs(got-want) <= 1e-9*max(1, math.Abs(want))
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		p, n float64
		want float64
	}{
		{"increase", 2, 100, 102},
		{"decrease", -2, 100, 98},
		{"zero percent", 0, 123.45, 123.45},
		{"fractional", 8.1, 5, 5.405},
		{"negative base", 10, -50, -55},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Add(tt.p, tt.n); !near(got, tt.want) {
				t.Errorf("Add(%v, %v) = %v, want %v", tt.p, tt.n, got, tt.want)
			}
		})
	}
}

func TestChange(t *testing.T) {
	tests := []struct {
		name     string
		from, to float64
		want     float64
	}{
		{"increase", 20, 20.8, 4},
		{"decrease", 100, 80, -20},
		{"no change", 50, 50, 0},
		{"negative from", -100, -110, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Change(tt.from, tt.to)
			if err != nil {
				t.Fatalf("Change(%v, %v) returned error: %v", tt.from, tt.to, err)
			}
			if !near(got, tt.want) {
				t.Errorf("Change(%v, %v) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestChangeFromZero(t *testing.T) {
	if _, err := Change(0, 5); !errors.Is(err, ErrZeroReference) {
		t.Errorf("Change(0, 5) error = %v, want ErrZeroReference", err)
	}
}

func TestShare(t *testing.T) {
	tests := []struct {
		name        string
		part, whole float64
		want        float64
	}{
		{"small share", 10, 200, 5},
		{"whole share", 200, 200, 100},
		{"oversized share", 300, 200, 150},
		{"zero part", 0, 200, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Share(tt.part, tt.whole)
			if err != nil {
				t.Fatalf("Share(%v, %v) returned error: %v", tt.part, tt.whole, err)
			}
			if !near(got, tt.want) {
				t.Errorf("Share(%v, %v) = %v, want %v", tt.part, tt.whole, got, tt.want)
			}
		})
	}
}

func TestShareOfZero(t *testing.T) {
	if _, err := Share(5, 0); !errors.Is(err, ErrZeroReference) {
		t.Errorf("Share(5, 0) error = %v, want ErrZeroReference", err)
	}
}

func TestCompound(t *testing.T) {
	tests := []struct {
		name    string
		p, n    float64
		periods int
		want    float64
	}{
		{"five periods", 7, 1000, 5, 1402.5517307},
		{"single period equals Add", 7, 1000, 1, 1070},
		{"zero periods", 7, 1000, 0, 1000},
		{"negative rate", -50, 1000, 2, 250},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Compound(tt.p, tt.n, tt.periods); !near(got, tt.want) {
				t.Errorf("Compound(%v, %v, %v) = %v, want %v", tt.p, tt.n, tt.periods, got, tt.want)
			}
		})
	}
}
